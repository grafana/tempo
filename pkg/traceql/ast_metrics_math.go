package traceql

import (
	"fmt"
	"math"

	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/prometheus/prometheus/model/labels"
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
		// Match by __query_fragment
		fv := v.Labels.GetValue(internalLabelQueryFragment)
		if fv.Type != TypeString || fv.EncodeToString(false) != m.key {
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
		l, lOk := getTSMatch(lhs, k)
		r, rOk := getTSMatch(rhs, k)
		if !lOk || !rOk {
			continue
		}

		n := min(len(r.Values), len(l.Values))
		values := buf[offset : offset+n]
		for j := 0; j < n; j++ {
			values[j] = applyArithmeticOp(op, l.Values[j], r.Values[j])
		}

		result[k] = TimeSeries{
			Labels:    mergeLabels(op, l.Labels, r.Labels),
			Values:    values,
			Exemplars: mergeExemplars(l.Exemplars, r.Exemplars),
		}
		offset += n
	}
	return result
}

func mergeLabels(op Operator, l, r Labels) Labels {
	out := make(Labels, len(l), len(l)+len(r))
	copy(out, l)
	for _, label := range r {
		if label.Name == labels.MetricName {
			continue
		}
		if l.Has(label.Name) { // dedup
			continue
		}
		out = append(out, label)
	}

	lNameValue := l.GetValue(labels.MetricName)
	rNameValue := r.GetValue(labels.MetricName)

	if lNameValue.Type != TypeString || rNameValue.Type != TypeString {
		for i := 0; i < len(out); i++ {
			if out[i].Name == labels.MetricName {
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
		if out[i].Name == labels.MetricName {
			out[i].Value = NewStaticString(combinedName)
			break
		}
	}
	return out
}

func getTSMatch(set SeriesSet, key SeriesMapKey) (TimeSeries, bool) {
	if s, ok := set[key]; ok {
		return s, true
	}
	if s, ok := set[noLabelsSeriesMapKey]; ok {
		return s, true
	}
	return TimeSeries{}, false
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

func applyArithmeticOp(op Operator, lhs, rhs float64) float64 {
	if math.IsNaN(lhs) {
		lhs = 0
	}
	if math.IsNaN(rhs) {
		rhs = 0
	}
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
