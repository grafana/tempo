// NOTE: Any changes to this file must be reflected in the corresponding SPECS.md or NOTES.md.

package traceql

import (
	"fmt"
	"math"

	"github.com/grafana/tempo/pkg/tempopb"
)

// seriesProcessor is the subset of firstStageElement used by the math path.
// batchSeriesProcessor and individual firstStageElement instances both satisfy this.
type seriesProcessor interface {
	observeSeries([]*tempopb.TimeSeries)
	result(multiplier float64) SeriesSet
	length() int
}

// batchSeriesProcessor routes incoming TimeSeries batches to per-fragment sub-processors.
// It implements seriesProcessor and firstStageElement (for use as metricsPipeline field).
// Keyed by the fragment string (__query_fragment label value).
type batchSeriesProcessor map[string]seriesProcessor

// observeSeries routes each incoming series to the appropriate sub-processor by
// scanning for the __query_fragment label.
func (b batchSeriesProcessor) observeSeries(in []*tempopb.TimeSeries) {
	// Partition series by fragment key
	fragments := make(map[string][]*tempopb.TimeSeries)
	for _, ts := range in {
		var fragmentKey string
		for _, lv := range ts.Labels {
			if lv.Key == internalLabelQueryFragment {
				fragmentKey = lv.Value.GetStringValue()
				break
			}
		}
		if fragmentKey != "" {
			fragments[fragmentKey] = append(fragments[fragmentKey], ts)
		}
	}
	for key, batch := range fragments {
		if proc, ok := b[key]; ok {
			proc.observeSeries(batch)
		}
	}
}

// result merges per-fragment results into a single SeriesSet, tagging each series
// with the fragment key as __query_fragment label.
func (b batchSeriesProcessor) result(multiplier float64) SeriesSet {
	merged := make(SeriesSet)
	for q, proc := range b {
		for _, v := range proc.result(multiplier) {
			v.Labels = v.Labels.Add(Label{
				Name:  internalLabelQueryFragment,
				Value: NewStaticString(q),
			})
			merged[v.Labels.MapKey()] = v
		}
	}
	return merged
}

// length returns the total length across all sub-processors.
func (b batchSeriesProcessor) length() int {
	total := 0
	for _, proc := range b {
		total += proc.length()
	}
	return total
}

// mathExpression is an AST node for a binary arithmetic operation (+, -, *, /) on
// two SeriesSets, or a leaf node that filters by __query_fragment label.
//
// Leaf node:   op == OpNone, key set, lhs/rhs nil
// Binary node: op != OpNone, lhs/rhs set, key empty
type mathExpression struct {
	op     Operator           // OpAdd/OpSub/OpMult/OpDiv for binary; OpNone for leaf
	key    string             // leaf only: __query_fragment value to match
	lhs    *mathExpression    // binary only: left child
	rhs    *mathExpression    // binary only: right child
	filter secondStageElement // leaf only: optional per-leaf second stage
}

var _ secondStageElement = (*mathExpression)(nil)

// String returns the canonical string form of the math expression.
// Leaf:   key + filter.String() (filter omitted if nil)
// Binary: "(lhs) op (rhs)"
func (m *mathExpression) String() string {
	if m.op == OpNone {
		// leaf
		if m.filter != nil {
			return m.key + m.filter.String()
		}
		return m.key
	}
	return "(" + m.lhs.String() + ") " + m.op.String() + " (" + m.rhs.String() + ")"
}

// validate checks that the operator is a supported arithmetic operator.
func (m *mathExpression) validate() error {
	if m.op == OpNone {
		// leaf node
		if m.filter != nil {
			return m.filter.validate()
		}
		return nil
	}
	// binary node
	if !m.op.isArithmetic() {
		return fmt.Errorf("unsupported math operation between queries: %s", m.op)
	}
	if err := m.lhs.validate(); err != nil {
		return err
	}
	return m.rhs.validate()
}

// init initialises the expression tree before the first process call.
func (m *mathExpression) init(req *tempopb.QueryRangeRequest) {
	if m.op == OpNone {
		if m.filter != nil {
			m.filter.init(req)
		}
		return
	}
	m.lhs.init(req)
	m.rhs.init(req)
}

// process evaluates the expression tree against the combined input SeriesSet.
// For leaf nodes it filters and strips internal labels. For binary nodes it fans
// out to children and applies the arithmetic operation.
func (m *mathExpression) process(input SeriesSet) SeriesSet {
	if m.op == OpNone {
		return m.processLeaf(input)
	}
	// Binary path: extract metric names before children strip __name__
	lhsName := m.lhs.metricName(input)
	rhsName := m.rhs.metricName(input)

	lhs := m.lhs.process(input)
	rhs := m.rhs.process(input)

	result := applyBinaryOp(m.op, lhs, rhs)

	// Re-key result with combined __name__ label
	if lhsName != "" && rhsName != "" {
		combinedName := "(" + lhsName + " " + m.op.String() + " " + rhsName + ")"
		out := make(SeriesSet, len(result))
		for _, v := range result {
			// Rebuild label set: prepend __name__, keep existing labels (already stripped by processLeaf)
			newLabels := make(Labels, 0, len(v.Labels)+1)
			newLabels = append(newLabels, Label{Name: "__name__", Value: NewStaticString(combinedName)})
			for _, l := range v.Labels {
				if l.Name != "__name__" {
					newLabels = append(newLabels, l)
				}
			}
			v.Labels = newLabels
			out[v.Labels.MapKey()] = v
		}
		return out
	}
	return result
}

// processLeaf filters the input to series matching this leaf's fragment key and
// strips internal labels (__query_fragment, __name__).
func (m *mathExpression) processLeaf(input SeriesSet) SeriesSet {
	result := make(SeriesSet)
	for smk, v := range input {
		fragVal, ok := v.Labels.GetValue(internalLabelQueryFragment)
		if !ok {
			continue
		}
		if fragVal.Type != TypeString || fragVal.EncodeToString(false) != m.key {
			continue
		}
		// Build new SeriesMapKey excluding __name__ and __query_fragment
		newKey := SeriesMapKey{}
		idx := 0
		for _, entry := range smk {
			if entry.Name == "" {
				break
			}
			if entry.Name == "__name__" || entry.Name == internalLabelQueryFragment {
				continue
			}
			if idx < len(newKey) {
				newKey[idx] = entry
				idx++
			}
		}
		// Build a new Labels slice, filtering out __name__ and __query_fragment.
		// We must NOT modify the original backing array because deduplicated sub-queries
		// share the same series entries in input; mutating one leaf's copy would break
		// the second leaf's lookup (e.g. ({} | rate()) / ({} | rate())).
		newLabels := make(Labels, 0, len(v.Labels))
		for _, l := range v.Labels {
			if l.Name != "__name__" && l.Name != internalLabelQueryFragment {
				newLabels = append(newLabels, l)
			}
		}
		v.Labels = newLabels
		result[newKey] = v
	}
	if m.filter != nil {
		result = m.filter.process(result)
	}
	return result
}

// metricName extracts the __name__ label value for the series matching this leaf's
// fragment key. Returns "" if not found. For binary nodes, combines both sides.
func (m *mathExpression) metricName(input SeriesSet) string {
	if m.op == OpNone {
		for _, v := range input {
			fragVal, ok := v.Labels.GetValue(internalLabelQueryFragment)
			if !ok {
				continue
			}
			if fragVal.Type != TypeString || fragVal.EncodeToString(false) != m.key {
				continue
			}
			nameVal, ok := v.Labels.GetValue("__name__")
			if !ok {
				continue
			}
			return nameVal.EncodeToString(false)
		}
		return ""
	}
	lhsName := m.lhs.metricName(input)
	rhsName := m.rhs.metricName(input)
	if lhsName != "" && rhsName != "" {
		return "(" + lhsName + " " + m.op.String() + " " + rhsName + ")"
	}
	return ""
}

// keyAndN pairs a SeriesMapKey with the number of values that will be written for it.
type keyAndN struct {
	k SeriesMapKey
	n int
}

// applyBinaryOp combines two SeriesSets element-wise using the given operator.
// O(1) allocations: a single buffer is pre-allocated for all value slices.
// Two-pass approach: first determine the actual n per series, then allocate once.
func applyBinaryOp(op Operator, lhs, rhs SeriesSet) SeriesSet {
	// Select the target set (prefer the one without a no-labels key for exact matching)
	target := lhs
	noLabelsKey := SeriesMapKey{}
	if _, hasNoLabels := rhs[noLabelsKey]; !hasNoLabels {
		target = rhs
	}

	// Pass 1: collect matching pairs and compute actual n per key.
	// This is required because getTSMatch may fall back to the no-labels scalar,
	// which can have fewer values than target[k]; using target[k]'s length for
	// pre-allocation would over-count and advance offset past the buffer end.
	pairs := make([]keyAndN, 0, len(target))
	totalN := 0
	for k := range target {
		l, lOk := getTSMatch(lhs, k)
		r, rOk := getTSMatch(rhs, k)
		if !lOk || !rOk {
			continue
		}
		n := min(len(l.Values), len(r.Values))
		pairs = append(pairs, keyAndN{k, n})
		totalN += n
	}

	// Pass 2: allocate a single backing buffer and fill results.
	buf := make([]float64, totalN)
	offset := 0
	result := make(SeriesSet, len(pairs))
	for _, p := range pairs {
		l, _ := getTSMatch(lhs, p.k)
		r, _ := getTSMatch(rhs, p.k)
		values := buf[offset : offset+p.n]
		for j := 0; j < p.n; j++ {
			values[j] = applyArithmeticOp(op, l.Values[j], r.Values[j])
		}
		// Pick the label set from the side with labels
		ll := l.Labels
		if len(ll) == 0 {
			ll = r.Labels
		}
		result[p.k] = TimeSeries{
			Labels:    ll,
			Values:    values,
			Exemplars: mergeExemplars(l.Exemplars, r.Exemplars),
		}
		offset += p.n
	}
	return result
}

// getTSMatch looks up a series by exact key, falling back to the no-labels key
// (SeriesMapKey{}) for scalar+vector arithmetic.
func getTSMatch(set SeriesSet, key SeriesMapKey) (TimeSeries, bool) {
	if ts, ok := set[key]; ok {
		return ts, true
	}
	noLabels := SeriesMapKey{}
	if ts, ok := set[noLabels]; ok {
		return ts, true
	}
	return TimeSeries{}, false
}

// mergeExemplars returns b if a is empty, a if b is empty, otherwise appends both.
func mergeExemplars(a, b []Exemplar) []Exemplar {
	if len(a) == 0 {
		return b
	}
	if len(b) == 0 {
		return a
	}
	out := make([]Exemplar, 0, len(a)+len(b))
	out = append(out, a...)
	out = append(out, b...)
	return out
}

// applyArithmeticOp applies a single arithmetic operation to two float64 values.
// NaN inputs are normalised to 0 before applying the operation.
// Division by zero returns math.NaN(). Unknown operators return math.NaN().
// Never panics.
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
