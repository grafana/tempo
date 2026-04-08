package traceql

import (
	"fmt"
	"math"

	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/prometheus/common/model"
)

var noLabelsSeriesMapKey = SeriesMapKey{}

// mathExpression implements secondStageElement and evaluates binary arithmetic
// operations (+, -, *, /) between metrics sub-query results. The tree structure
// lives here — RootExpr stays flat.
//
// Leaf nodes (op == OpNone) extract their fragment from the input SeriesSet by
// matching __query_fragment labels, then optionally apply a per-leaf filter
// (e.g. topk inside parentheses).
//
// Binary nodes recursively process children and combine with applyBinaryOp.
type mathExpression struct {
	op     Operator
	key    string
	lhs    *mathExpression
	rhs    *mathExpression
	filter secondStageElement
}

var _ secondStageElement = (*mathExpression)(nil)

func (m *mathExpression) rewriteKeys(keyMap map[string]string) *mathExpression {
	if m == nil {
		return nil
	}
	cp := *m
	if m.op == OpNone {
		if newKey, ok := keyMap[m.key]; ok {
			cp.key = newKey
		}
	} else {
		cp.lhs = m.lhs.rewriteKeys(keyMap)
		cp.rhs = m.rhs.rewriteKeys(keyMap)
	}
	return &cp
}

func newFlatExpression(key string, filter secondStageElement) *mathExpression {
	return &mathExpression{
		op:     OpNone,
		key:    key,
		filter: filter,
	}
}

func (m *mathExpression) String() string {
	var s string
	if m.op == OpNone {
		s = m.key
	} else {
		s = "(" + m.lhs.String() + ") " + m.op.String() + " (" + m.rhs.String() + ")"
	}
	if m.filter != nil {
		s += m.filter.String()
	}
	return s
}

func (m *mathExpression) validate() error {
	if m.op != OpNone {
		if !m.op.isArithmetic() {
			return fmt.Errorf("unsupported math operation between queries: %s", m.op)
		}
		if err := m.lhs.validate(); err != nil {
			return err
		}
		if err := m.rhs.validate(); err != nil {
			return err
		}
	}
	if m.filter != nil {
		return m.filter.validate()
	}
	return nil
}

func (m *mathExpression) init(req *tempopb.QueryRangeRequest) {
	if m.op != OpNone {
		m.lhs.init(req)
		m.rhs.init(req)
	}
	if m.filter != nil {
		m.filter.init(req)
	}
}

func (m *mathExpression) separator() string {
	return ""
}

// process evaluates the math expression tree against a combined SeriesSet that
// contains series from all sub-queries, tagged with __query_fragment labels.
func (m *mathExpression) process(input SeriesSet) SeriesSet {
	if m.op == OpNone {
		return m.processLeaf(input)
	}

	lhs := m.lhs.process(input)
	rhs := m.rhs.process(input)

	out := applyBinaryOp(m.op, lhs, rhs)

	if m.filter != nil {
		out = m.filter.process(out)
	}
	return out
}

// processLeaf extracts series matching this leaf's __query_fragment key,
// strips internal labels, and optionally applies a per-leaf filter.
func (m *mathExpression) processLeaf(input SeriesSet) SeriesSet {
	var result SeriesSet
	result = make(SeriesSet, len(input))
	for smk, v := range input {
		fv := v.Labels.GetValue(internalLabelQueryFragment)
		if fv.Type != TypeNil && // pass through for single sub-query
			// filter by query fragment
			(fv.Type != TypeString || fv.EncodeToString(false) != m.key) {
			continue
		}

		// Build new key without __query_fragment and __name__
		key := SeriesMapKey{}
		j := 0
		for i := range smk {
			if smk[i].Name == internalLabelQueryFragment {
				continue
			}
			key[j] = smk[i]
			j++
		}

		// Copy the series — must not mutate input since other leaves
		// may read the same series from the shared input.
		stripped := make(Labels, 0, len(v.Labels))
		for _, l := range v.Labels {
			if l.Name != internalLabelQueryFragment {
				stripped = append(stripped, l)
			}
		}
		values := make([]float64, len(v.Values))
		copy(values, v.Values)
		result[key] = TimeSeries{
			Labels:    stripped,
			Values:    values,
			Exemplars: v.Exemplars,
		}
	}

	if m.filter != nil {
		result = m.filter.process(result)
	}
	return result
}

func applyBinaryOp(op Operator, lhs, rhs SeriesSet) SeriesSet {
	target := lhs
	if _, ok := rhs[noLabelsSeriesMapKey]; !ok {
		target = rhs
	}

	result := make(SeriesSet, len(target))

	// pre-allocate array once to avoid multiple smaller allocations
	var valuesLen int
	for _, t := range target {
		valuesLen += len(t.Values)
	}
	buf := make([]float64, valuesLen)
	var offset int

	for k := range target {
		l, lFanOut, lOk := getTSMatch(lhs, k)
		r, rFanOut, rOk := getTSMatch(rhs, k)
		if !lOk || !rOk {
			continue
		}

		var exemplars []Exemplar
		switch {
		case lFanOut == rFanOut: // either both fan-out or both not
			exemplars = mergeExemplars(l.Exemplars, r.Exemplars)
		case lFanOut:
			exemplars = r.Exemplars
		default:
			exemplars = l.Exemplars
		}

		n := min(len(r.Values), len(l.Values))
		values := buf[offset : offset+n]
		for j := 0; j < n; j++ {
			values[j] = applyArithmeticOp(op, l.Values[j], r.Values[j])
		}

		result[k] = TimeSeries{
			Labels:    mergeLabels(op, l.Labels, r.Labels),
			Values:    values,
			Exemplars: exemplars,
		}
		offset += n
	}
	return result
}

func mergeLabels(op Operator, l, r Labels) Labels {
	out := make(Labels, len(l), len(l)+len(r))
	copy(out, l)
	for _, label := range r {
		if label.Name == model.MetricNameLabel {
			continue
		}
		if l.Has(label.Name) { // dedup
			continue
		}
		out = append(out, label)
	}

	lNameValue := l.GetValue(model.MetricNameLabel)
	rNameValue := r.GetValue(model.MetricNameLabel)

	if lNameValue.Type != TypeString || rNameValue.Type != TypeString {
		for i := 0; i < len(out); i++ {
			if out[i].Name == model.MetricNameLabel {
				out[i] = out[len(out)-1]
				out = out[:len(out)-1]
				break
			}
		}
		return out
	}

	lName := lNameValue.EncodeToString(false)
	rName := rNameValue.EncodeToString(false)

	combinedName := fmt.Sprintf("(%s %s %s)", lName, op.String(), rName)

	for i := range out {
		if out[i].Name == model.MetricNameLabel {
			out[i].Value = NewStaticString(combinedName)
			break
		}
	}
	return out
}

// getTSMatch looks for a time series in the set matching the given key, or if not found,
// it looks for a series with no labels (fan-out case) and returns it if found. Returns false if neither is found.
// TODO: we might need to support prometheus-like on() and group_left()
func getTSMatch(set SeriesSet, key SeriesMapKey) (TimeSeries, bool, bool) {
	if s, ok := set[key]; ok {
		return s, false, true
	}
	if s, ok := set[noLabelsSeriesMapKey]; ok {
		return s, true, true
	}
	return TimeSeries{}, false, false
}

func mergeExemplars(a, b []Exemplar) []Exemplar {
	if len(a) == 0 {
		return b
	}
	if len(b) == 0 {
		return a
	}
	result := make([]Exemplar, 0, len(a)+len(b))
	result = append(result, a...)
	result = append(result, b...)
	return result
}

// MetricsScalarOp implements second stage scalar arithmetic on metrics results.
// It applies an arithmetic operation between each sample value and a scalar constant.
// Example: 100 * ({} | rate()) or ({} | rate()) / 1000
type MetricsScalarOp struct {
	op           Operator // OpAdd, OpSub, OpMult, OpDiv
	value        float64
	scalarOnLeft bool // true: value OP series, false: series OP value
}

func newMetricsScalarOp(op Operator, value float64, scalarOnLeft bool) *MetricsScalarOp {
	return &MetricsScalarOp{op: op, value: value, scalarOnLeft: scalarOnLeft}
}

func (m *MetricsScalarOp) String() string {
	v := formatFloat(m.value)
	if m.scalarOnLeft {
		return v + " " + m.op.String() + " "
	}
	return " " + m.op.String() + " " + v
}

func (m *MetricsScalarOp) validate() error {
	if !m.op.isArithmetic() {
		return fmt.Errorf("unsupported scalar operation: %s", m.op.String())
	}
	return nil
}

func (m *MetricsScalarOp) init(_ *tempopb.QueryRangeRequest) {}

func (m *MetricsScalarOp) process(input SeriesSet) SeriesSet {
	result := make(SeriesSet, len(input))
	for key, series := range input {
		values := make([]float64, len(series.Values))
		if m.scalarOnLeft {
			for i, v := range series.Values {
				values[i] = applyArithmeticOp(m.op, m.value, v)
			}
		} else {
			for i, v := range series.Values {
				values[i] = applyArithmeticOp(m.op, v, m.value)
			}
		}

		result[key] = TimeSeries{
			Labels:    series.Labels,
			Values:    values,
			Exemplars: series.Exemplars,
		}
	}
	return result
}

func (m *MetricsScalarOp) separator() string {
	return ""
}

var _ secondStageElement = (*MetricsScalarOp)(nil)

func applyArithmeticOp(op Operator, lhs, rhs float64) float64 {
	switch op {
	case OpAdd:
		return lhs + rhs
	case OpSub:
		return lhs - rhs
	case OpMult:
		return lhs * rhs
	case OpDiv:
		if rhs == 0 {
			return math.NaN()
		}
		return lhs / rhs
	default:
		return math.NaN()
	}
}
