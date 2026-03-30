// NOTE: Any changes to this file must be reflected in the corresponding SPECS.md, TESTS.md, and BENCHMARKS.md.

package traceql

import (
	"fmt"
	"math"

	"github.com/grafana/tempo/pkg/tempopb"
	commonv1proto "github.com/grafana/tempo/pkg/tempopb/common/v1"
)

// seriesProcessor is the minimal interface for accumulating proto time-series
// and producing a final SeriesSet. It is satisfied by both firstStageElement
// (via adapter) and batchSeriesProcessor.
//
// Result must be called exactly once per query lifetime.
type seriesProcessor interface {
	observeSeries([]*tempopb.TimeSeries)
	result(multiplier float64) SeriesSet
	length() int
}

// mathExpression is a node in the metrics math expression tree.
//
// Leaf node (lhs == nil): key is the fragment key, filter is an optional
// per-leaf second-stage filter.
//
// Binary node (lhs != nil): op is an arithmetic operator, lhs and rhs are
// child expressions.
//
// mathExpression is stateless after init(). Concurrent calls to process() are
// safe after initialization completes.
type mathExpression struct {
	op     Operator
	key    string             // non-empty for leaf nodes only
	lhs    *mathExpression    // non-nil for binary nodes only
	rhs    *mathExpression    // non-nil for binary nodes only
	filter secondStageElement // optional per-leaf second-stage filter
}

// String returns the canonical string form.
// Leaf: "<key>" or "<key><filterString>".
// Binary: "(<left>) <op> (<right>)".
func (m *mathExpression) String() string {
	if m.lhs == nil {
		if m.filter != nil {
			return m.key + m.filter.String()
		}
		return m.key
	}
	return fmt.Sprintf("(%s) %s (%s)", m.lhs.String(), m.op.String(), m.rhs.String())
}

// validate returns an error if this node is structurally invalid.
// Binary: operator must be arithmetic. Leaf: delegates to filter.
func (m *mathExpression) validate() error {
	if m.lhs == nil {
		if m.filter != nil {
			return m.filter.validate()
		}
		return nil
	}
	if !m.op.isArithmetic() {
		return fmt.Errorf("unsupported math operation: %s", m.op.String())
	}
	if err := m.lhs.validate(); err != nil {
		return err
	}
	return m.rhs.validate()
}

// init initializes the node for the given query parameters.
// Must be called exactly once. Callers are responsible for single invocation.
func (m *mathExpression) init(req *tempopb.QueryRangeRequest) {
	if m.lhs == nil {
		if m.filter != nil {
			m.filter.init(req)
		}
		return
	}
	m.lhs.init(req)
	m.rhs.init(req)
}

// metricName returns the "__name__" label value for this node's output set.
// Leaf: scans input for a series matching this key and returns its __name__ label.
// Binary: returns "(leftName op rightName)" only when both are non-empty.
func (m *mathExpression) metricName(input SeriesSet) string {
	if m.lhs == nil {
		for _, ts := range input {
			if getFragmentLabel(ts.Labels) != m.key {
				continue
			}
			for _, label := range ts.Labels {
				if label.Name == "__name__" {
					return label.Value.EncodeToString(false)
				}
			}
			return ""
		}
		return ""
	}
	leftName := m.lhs.metricName(input)
	rightName := m.rhs.metricName(input)
	if leftName == "" || rightName == "" {
		return ""
	}
	return fmt.Sprintf("(%s %s %s)", leftName, m.op.String(), rightName)
}

// getFragmentLabel returns the internalLabelQueryFragment label value,
// or "" if not present.
func getFragmentLabel(ls Labels) string {
	for _, label := range ls {
		if label.Name == internalLabelQueryFragment {
			return label.Value.EncodeToString(false)
		}
	}
	return ""
}

// processLeaf selects series for this leaf's key, strips the fragment and __name__
// labels (allocating a new label slice — never mutating input), and applies the
// optional leaf filter.
//
// IMPORTANT: label slices are never mutated. When two leaves share a fragment
// key (deduplicated sub-queries), both receive the same underlying series.
// Mutating the first leaf's labels would corrupt the second leaf's lookup.
func (m *mathExpression) processLeaf(input SeriesSet) SeriesSet {
	out := make(SeriesSet)
	for _, ts := range input {
		if getFragmentLabel(ts.Labels) != m.key {
			continue
		}
		newLabels := make(Labels, 0, len(ts.Labels))
		for _, label := range ts.Labels {
			if label.Name == internalLabelQueryFragment || label.Name == "__name__" {
				continue
			}
			newLabels = append(newLabels, label)
		}
		newKey := newLabels.MapKey()
		out[newKey] = TimeSeries{
			Labels:    newLabels,
			Values:    ts.Values,
			Exemplars: ts.Exemplars,
		}
	}
	if m.filter != nil {
		return m.filter.process(out)
	}
	return out
}

// process evaluates this node. Implements secondStageElement.
//
// Binary: extract metric name, fan out to both children, combine element-wise,
// re-label with combined name.
// Leaf: delegates to processLeaf.
func (m *mathExpression) process(input SeriesSet) SeriesSet {
	if m.lhs == nil {
		return m.processLeaf(input)
	}
	combinedName := m.metricName(input)
	leftResult := m.lhs.process(input)
	rightResult := m.rhs.process(input)
	combined := applyBinaryOp(m.op, leftResult, rightResult)
	if combinedName == "" {
		return combined
	}
	nameLabel := Label{Name: "__name__", Value: NewStaticString(combinedName)}
	out := make(SeriesSet, len(combined))
	for _, ts := range combined {
		newLabels := ts.Labels.Add(nameLabel)
		newKey := newLabels.MapKey()
		out[newKey] = TimeSeries{
			Labels:    newLabels,
			Values:    ts.Values,
			Exemplars: ts.Exemplars,
		}
	}
	return out
}

var _ secondStageElement = (*mathExpression)(nil)

// applyBinaryOp combines two SeriesSets element-wise using op.
//
// Two-pass algorithm (O(1) allocations per call regardless of set size):
//
//	Pass 1: for each key in the target set, find its match in the other set.
//	        Record matched pairs and their combined value count.
//	        Sum counts → total buffer size.
//	Pass 2: allocate one contiguous float64 buffer. Slice into per-series
//	        segments. Compute each value via applyArithmeticOp.
//
// The set without a no-labels entry is chosen as the target to maximize
// exact key matches. Series with no match are silently dropped.
func applyBinaryOp(op Operator, left, right SeriesSet) SeriesSet {
	emptyKey := SeriesMapKey{}
	_, leftHasEmpty := left[emptyKey]
	_, rightHasEmpty := right[emptyKey]

	// Choose target as the side less likely to produce no-labels fallback hits.
	// Track swap so we can restore operand order when computing values.
	target, other := left, right
	swapped := false
	if leftHasEmpty && !rightHasEmpty {
		target, other = right, left
		swapped = true
	}

	type pair struct {
		targetTS TimeSeries
		otherTS  TimeSeries
		count    int
	}
	pairs := make([]pair, 0, len(target))
	totalValues := 0

	// Pass 1.
	for k, ts := range target {
		otherTS, ok := getTSMatch(k, other)
		if !ok {
			continue
		}
		count := len(ts.Values)
		if len(otherTS.Values) < count {
			count = len(otherTS.Values)
		}
		pairs = append(pairs, pair{ts, otherTS, count})
		totalValues += count
	}

	if len(pairs) == 0 {
		return SeriesSet{}
	}

	// Pass 2: one allocation for all value data.
	buf := make([]float64, totalValues)
	out := make(SeriesSet, len(pairs))
	offset := 0
	for _, p := range pairs {
		segment := buf[offset : offset+p.count]
		offset += p.count
		for i := 0; i < p.count; i++ {
			if swapped {
				// target came from rhs, other from lhs — restore original order.
				segment[i] = applyArithmeticOp(op, p.otherTS.Values[i], p.targetTS.Values[i])
			} else {
				segment[i] = applyArithmeticOp(op, p.targetTS.Values[i], p.otherTS.Values[i])
			}
		}
		newKey := p.targetTS.Labels.MapKey()
		out[newKey] = TimeSeries{
			Labels:    p.targetTS.Labels,
			Values:    segment,
			Exemplars: mergeExemplars(p.targetTS.Exemplars, p.otherTS.Exemplars),
		}
	}
	return out
}

// getTSMatch returns the series for key in ss, with no-labels fallback.
// Returns (series, true) if found; (zero, false) if not found.
func getTSMatch(key SeriesMapKey, ss SeriesSet) (TimeSeries, bool) {
	if ts, ok := ss[key]; ok {
		return ts, true
	}
	emptyKey := SeriesMapKey{}
	if ts, ok := ss[emptyKey]; ok {
		return ts, true
	}
	return TimeSeries{}, false
}

// applyArithmeticOp applies op to l and r with NaN semantics (SPECS.md §6.3):
//   - NaN inputs are coerced to 0 before operating.
//   - Division by zero returns NaN.
//   - Unknown operators return NaN.
//   - Never panics.
func applyArithmeticOp(op Operator, l, r float64) float64 {
	if math.IsNaN(l) {
		l = 0
	}
	if math.IsNaN(r) {
		r = 0
	}
	switch op {
	case OpAdd:
		return l + r
	case OpSub:
		return l - r
	case OpMult:
		return l * r
	case OpDiv:
		if r == 0 {
			return math.NaN()
		}
		return l / r
	default:
		return math.NaN()
	}
}

// mergeExemplars merges two exemplar slices with no deduplication.
// Returns one side unchanged if the other is empty.
func mergeExemplars(a, b []Exemplar) []Exemplar {
	if len(a) == 0 {
		return b
	}
	if len(b) == 0 {
		return a
	}
	merged := make([]Exemplar, 0, len(a)+len(b))
	merged = append(merged, a...)
	merged = append(merged, b...)
	return merged
}

// batchSeriesProcessor routes incoming proto time-series to per-fragment
// sub-processors keyed by internalLabelQueryFragment. Implements seriesProcessor.
//
// Used as the series processor in MetricsFrontendEvaluator for math queries
// (SPECS.md §7.2).
type batchSeriesProcessor map[string]seriesProcessor

func newBatchSeriesProcessor(procs map[string]firstStageElement) batchSeriesProcessor {
	bsp := make(batchSeriesProcessor, len(procs))
	for k, v := range procs {
		bsp[k] = v
	}
	return bsp
}

// observeSeries routes each series to its processor by fragment key.
// Series with unrecognized fragment keys are silently discarded.
func (b batchSeriesProcessor) observeSeries(in []*tempopb.TimeSeries) {
	byKey := make(map[string][]*tempopb.TimeSeries, len(b))
	for _, ts := range in {
		key := getFragmentLabelProto(ts.Labels)
		if _, ok := b[key]; ok {
			byKey[key] = append(byKey[key], ts)
		}
	}
	for key, batch := range byKey {
		b[key].observeSeries(batch)
	}
}

// result merges all sub-processor outputs, tagging each series with its
// fragment key label before merging.
func (b batchSeriesProcessor) result(multiplier float64) SeriesSet {
	out := make(SeriesSet)
	for key, proc := range b {
		subSet := proc.result(multiplier)
		fragLabel := Label{Name: internalLabelQueryFragment, Value: NewStaticString(key)}
		for _, ts := range subSet {
			// Prepend the fragment label so it is always within the maxGroupBys limit.
			tagged := append(Labels{fragLabel}, ts.Labels...)
			taggedKey := tagged.MapKey()
			out[taggedKey] = TimeSeries{
				Labels:    tagged,
				Values:    ts.Values,
				Exemplars: ts.Exemplars,
			}
		}
	}
	return out
}

// length returns the total series count across all sub-processors.
func (b batchSeriesProcessor) length() int {
	total := 0
	for _, proc := range b {
		total += proc.length()
	}
	return total
}

var _ seriesProcessor = batchSeriesProcessor{}

// getFragmentLabelProto extracts internalLabelQueryFragment from proto labels.
// Returns "" if not found.
func getFragmentLabelProto(labels []commonv1proto.KeyValue) string {
	for _, kv := range labels {
		if kv.Key == internalLabelQueryFragment {
			return StaticFromAnyValue(kv.Value).EncodeToString(false)
		}
	}
	return ""
}
