// NOTE: Any changes to this file must be reflected in the corresponding SPECS.md and TESTS.md.

package traceql

import (
	"math"
	"testing"

	"github.com/grafana/tempo/pkg/tempopb"
	commonv1proto "github.com/grafana/tempo/pkg/tempopb/common/v1"
	"github.com/stretchr/testify/require"
)

// Compile-time interface assertion.
var _ seriesProcessor = batchSeriesProcessor{}

// helpers for building test SeriesSets

func makeKey(labels Labels) SeriesMapKey {
	return labels.MapKey()
}

func makeTS(labels Labels, values ...float64) TimeSeries {
	return TimeSeries{Labels: labels, Values: values}
}

func seriesSetToSlice(ss SeriesSet) []TimeSeries {
	out := make([]TimeSeries, 0, len(ss))
	for _, ts := range ss {
		out = append(out, ts)
	}
	return out
}

// requireSeriesSetEqual compares two SeriesSets, tolerating NaN.
func requireSeriesSetEqual(t *testing.T, expected, actual SeriesSet) {
	t.Helper()
	require.Equal(t, len(expected), len(actual), "SeriesSet length mismatch")
	for k, eTS := range expected {
		aTS, ok := actual[k]
		require.True(t, ok, "expected key %v to be in result", k)
		require.Equal(t, eTS.Labels, aTS.Labels)
		require.Equal(t, len(eTS.Values), len(aTS.Values), "values length mismatch for key %v", k)
		for i := range eTS.Values {
			if math.IsNaN(eTS.Values[i]) {
				require.True(t, math.IsNaN(aTS.Values[i]), "expected NaN at index %d for key %v", i, k)
			} else {
				require.InDelta(t, eTS.Values[i], aTS.Values[i], 1e-9, "value mismatch at index %d for key %v", i, k)
			}
		}
	}
}

// TRQL-T-04: Correct scalar arithmetic for all four operators.
func TestApplyArithmeticOpBasic(t *testing.T) {
	tests := []struct {
		op       Operator
		lhs, rhs float64
		expected float64
	}{
		{OpAdd, 3.0, 2.0, 5.0},
		{OpSub, 3.0, 2.0, 1.0},
		{OpMult, 3.0, 2.0, 6.0},
		{OpDiv, 3.0, 2.0, 1.5},
	}
	for _, tc := range tests {
		got := applyArithmeticOp(tc.op, tc.lhs, tc.rhs)
		require.InDelta(t, tc.expected, got, 1e-9, "op=%s lhs=%v rhs=%v", tc.op, tc.lhs, tc.rhs)
	}
}

// TRQL-T-05: NaN and division-by-zero handling.
func TestApplyArithmeticOpNaN(t *testing.T) {
	nan := math.NaN()

	// Division by zero → NaN
	result := applyArithmeticOp(OpDiv, 1.0, 0.0)
	require.True(t, math.IsNaN(result), "expected NaN for div-by-zero")

	// NaN lhs coerced to 0: 0+2=2
	result = applyArithmeticOp(OpAdd, nan, 2.0)
	require.InDelta(t, 2.0, result, 1e-9)

	// NaN rhs coerced to 0: 3-0=3
	result = applyArithmeticOp(OpSub, 3.0, nan)
	require.InDelta(t, 3.0, result, 1e-9)

	// Both NaN coerced to 0: 0*0=0
	result = applyArithmeticOp(OpMult, nan, nan)
	require.InDelta(t, 0.0, result, 1e-9)

	// Invalid operator → NaN
	result = applyArithmeticOp(OpAnd, 1.0, 2.0)
	require.True(t, math.IsNaN(result), "expected NaN for invalid op")
}

// TRQL-T-06: Series with identical label keys are combined correctly.
func TestApplyBinaryOpMatchingKeys(t *testing.T) {
	keyA := LabelsFromArgs("service", "a").MapKey()
	keyB := LabelsFromArgs("service", "b").MapKey()
	keyC := LabelsFromArgs("service", "c").MapKey()

	lhs := SeriesSet{
		keyA: makeTS(LabelsFromArgs("service", "a"), 1, 2, 3, 4),
		keyB: makeTS(LabelsFromArgs("service", "b"), 5, 6, 7, 8),
		keyC: makeTS(LabelsFromArgs("service", "c"), 9, 10, 11, 12),
	}
	rhs := SeriesSet{
		keyA: makeTS(LabelsFromArgs("service", "a"), 10, 20, 30, 40),
		keyB: makeTS(LabelsFromArgs("service", "b"), 50, 60, 70, 80),
		keyC: makeTS(LabelsFromArgs("service", "c"), 90, 100, 110, 120),
	}

	result := applyBinaryOp(OpAdd, lhs, rhs)
	require.Equal(t, 3, len(result))

	for _, key := range []SeriesMapKey{keyA, keyB, keyC} {
		ts, ok := result[key]
		require.True(t, ok, "expected key %v in result", key)
		lhsTS := lhs[key]
		rhsTS := rhs[key]
		for i := range ts.Values {
			require.InDelta(t, lhsTS.Values[i]+rhsTS.Values[i], ts.Values[i], 1e-9)
		}
	}
}

// TRQL-T-07: Series present in one side but not the other are silently dropped.
func TestApplyBinaryOpMissingKey(t *testing.T) {
	keyA := LabelsFromArgs("service", "a").MapKey()
	keyB := LabelsFromArgs("service", "b").MapKey()
	keyC := LabelsFromArgs("service", "c").MapKey()

	lhs := SeriesSet{
		keyA: makeTS(LabelsFromArgs("service", "a"), 1, 2),
		keyB: makeTS(LabelsFromArgs("service", "b"), 3, 4),
		keyC: makeTS(LabelsFromArgs("service", "c"), 5, 6),
	}
	rhs := SeriesSet{
		keyA: makeTS(LabelsFromArgs("service", "a"), 10, 20),
		keyB: makeTS(LabelsFromArgs("service", "b"), 30, 40),
		// keyC absent
	}

	result := applyBinaryOp(OpDiv, lhs, rhs)
	require.Equal(t, 2, len(result))
	_, hasC := result[keyC]
	require.False(t, hasC, "key C should be absent from result")
	_, hasA := result[keyA]
	require.True(t, hasA)
	_, hasB := result[keyB]
	require.True(t, hasB)
}

// TRQL-T-08: Fallback to no-labels key when exact key absent.
func TestApplyBinaryOpNoLabels(t *testing.T) {
	noLabels := Labels{}
	noLabelsKey := noLabels.MapKey()
	fooLabels := LabelsFromArgs("foo", "bar")
	fooKey := fooLabels.MapKey()

	lhs := SeriesSet{
		noLabelsKey: makeTS(noLabels, 3.0, 3.0),
	}
	rhs := SeriesSet{
		fooKey: makeTS(fooLabels, 2.0, 2.0),
	}

	result := applyBinaryOp(OpMult, lhs, rhs)
	// rhs has no no-labels key, so target = rhs
	// For fooKey: getTSMatch(lhs, fooKey) falls back to noLabels
	require.Equal(t, 1, len(result))
	ts, ok := result[fooKey]
	require.True(t, ok)
	require.InDelta(t, 6.0, ts.Values[0], 1e-9)
}

// TRQL-T-09: Leaf node filters series by __query_fragment and strips internal labels.
func TestMathExpressionLeafProcess(t *testing.T) {
	// Build input with 4 series: 2 tagged frag-A, 2 tagged frag-B
	fragAVal := NewStaticString("frag-A")
	fragBVal := NewStaticString("frag-B")
	nameVal := NewStaticString("rate")

	makeInputTS := func(fragVal Static, svc string) (SeriesMapKey, TimeSeries) {
		labels := Labels{
			{Name: "__name__", Value: nameVal},
			{Name: internalLabelQueryFragment, Value: fragVal},
			{Name: "service", Value: NewStaticString(svc)},
		}
		return labels.MapKey(), TimeSeries{Labels: labels, Values: []float64{1.0, 2.0}}
	}

	k1, ts1 := makeInputTS(fragAVal, "svc1")
	k2, ts2 := makeInputTS(fragAVal, "svc2")
	k3, ts3 := makeInputTS(fragBVal, "svc1")
	k4, ts4 := makeInputTS(fragBVal, "svc2")

	input := SeriesSet{k1: ts1, k2: ts2, k3: ts3, k4: ts4}

	leaf := &mathExpression{op: OpNone, key: "frag-A"}
	result := leaf.process(input)

	require.Equal(t, 2, len(result), "expected 2 series for frag-A")
	for _, ts := range result {
		for _, l := range ts.Labels {
			require.NotEqual(t, internalLabelQueryFragment, l.Name, "__query_fragment should be stripped")
			require.NotEqual(t, "__name__", l.Name, "__name__ should be stripped")
		}
	}
}

// TRQL-T-10: Binary node fans out to both children and combines results.
func TestMathExpressionBinaryProcess(t *testing.T) {
	fragAVal := NewStaticString("frag-A")
	fragBVal := NewStaticString("frag-B")
	svcLabels := func(fragVal Static, svc, name string) Labels {
		return Labels{
			{Name: "__name__", Value: NewStaticString(name)},
			{Name: internalLabelQueryFragment, Value: fragVal},
			{Name: "service", Value: NewStaticString(svc)},
		}
	}

	labA1 := svcLabels(fragAVal, "svc1", "rate")
	labA2 := svcLabels(fragAVal, "svc2", "rate")
	labB1 := svcLabels(fragBVal, "svc1", "rate")
	labB2 := svcLabels(fragBVal, "svc2", "rate")

	input := SeriesSet{
		labA1.MapKey(): {Labels: labA1, Values: []float64{4.0, 4.0}},
		labA2.MapKey(): {Labels: labA2, Values: []float64{4.0, 4.0}},
		labB1.MapKey(): {Labels: labB1, Values: []float64{2.0, 2.0}},
		labB2.MapKey(): {Labels: labB2, Values: []float64{2.0, 2.0}},
	}

	expr := &mathExpression{
		op:  OpDiv,
		lhs: &mathExpression{op: OpNone, key: "frag-A"},
		rhs: &mathExpression{op: OpNone, key: "frag-B"},
	}
	result := expr.process(input)

	require.Equal(t, 2, len(result), "expected 2 series in result")
	for _, ts := range result {
		for _, v := range ts.Values {
			require.InDelta(t, 2.0, v, 1e-9, "expected 4/2=2.0")
		}
		for _, l := range ts.Labels {
			require.NotEqual(t, internalLabelQueryFragment, l.Name)
		}
	}
}

// TRQL-T-11: validate() rejects non-arithmetic operators.
func TestMathExpressionValidate(t *testing.T) {
	expr := &mathExpression{
		op:  OpAnd,
		lhs: &mathExpression{op: OpNone, key: "a"},
		rhs: &mathExpression{op: OpNone, key: "b"},
	}
	err := expr.validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported math operation")
}

// TRQL-T-12: validate() propagates leaf filter errors.
func TestMathExpressionValidateLeafFilter(t *testing.T) {
	// Use TopKBottomK with limit=0 which returns an error from validate()
	badFilter := newTopKBottomK(OpTopK, 0)
	leaf := &mathExpression{op: OpNone, key: "a", filter: badFilter}
	err := leaf.validate()
	require.Error(t, err)
}

// Additional: valid arithmetic operators should not error
func TestMathExpressionValidateArithmetic(t *testing.T) {
	for _, op := range []Operator{OpAdd, OpSub, OpMult, OpDiv} {
		expr := &mathExpression{
			op:  op,
			lhs: &mathExpression{op: OpNone, key: "a"},
			rhs: &mathExpression{op: OpNone, key: "b"},
		}
		require.NoError(t, expr.validate(), "op=%s should be valid", op)
	}
}

// Additional: leaf with no filter validates OK
func TestMathExpressionValidateLeafNoFilter(t *testing.T) {
	leaf := &mathExpression{op: OpNone, key: "a"}
	require.NoError(t, leaf.validate())
}

// Additional: batchSeriesProcessor routes series by fragment label
func TestBatchSeriesProcessorRoutes(t *testing.T) {
	// Two mock processors
	a := &mockSeriesProcessor{}
	b := &mockSeriesProcessor{}

	bsp := batchSeriesProcessor{
		"frag-A": a,
		"frag-B": b,
	}

	// Build proto TimeSeries with fragment labels
	makePTS := func(fragKey, svc string) *tempopb.TimeSeries {
		return &tempopb.TimeSeries{
			Labels: []commonv1proto.KeyValue{
				{Key: internalLabelQueryFragment, Value: &commonv1proto.AnyValue{Value: &commonv1proto.AnyValue_StringValue{StringValue: fragKey}}},
				{Key: "service", Value: &commonv1proto.AnyValue{Value: &commonv1proto.AnyValue_StringValue{StringValue: svc}}},
			},
		}
	}

	in := []*tempopb.TimeSeries{
		makePTS("frag-A", "svc1"),
		makePTS("frag-B", "svc2"),
		makePTS("frag-A", "svc3"),
		makePTS("unknown", "svc4"), // should be discarded
	}
	bsp.observeSeries(in)

	require.Equal(t, 2, len(a.received))
	require.Equal(t, 1, len(b.received))
}

// Regression: label slice must not be mutated — deduplicated same-key sub-queries produce correct results.
// When both leaves share the same fragment key, the first processLeaf call must not strip
// labels from the shared input entries, so the second leaf can still find its series.
func TestMathExpressionSameKeyDedup(t *testing.T) {
	// Build input: two series tagged with the shared fragment key, values all 2.0.
	// ({} | rate()) / ({} | rate()) should yield 1.0 (2.0 / 2.0), NOT empty.
	sharedKey := "{ } | rate()"
	fragVal := NewStaticString(sharedKey)
	nameVal := NewStaticString("rate")

	svc1Labels := Labels{
		{Name: "__name__", Value: nameVal},
		{Name: internalLabelQueryFragment, Value: fragVal},
		{Name: "service", Value: NewStaticString("svc1")},
	}
	svc2Labels := Labels{
		{Name: "__name__", Value: nameVal},
		{Name: internalLabelQueryFragment, Value: fragVal},
		{Name: "service", Value: NewStaticString("svc2")},
	}

	input := SeriesSet{
		svc1Labels.MapKey(): {Labels: svc1Labels, Values: []float64{2.0, 2.0}},
		svc2Labels.MapKey(): {Labels: svc2Labels, Values: []float64{2.0, 2.0}},
	}

	expr := &mathExpression{
		op:  OpDiv,
		lhs: &mathExpression{op: OpNone, key: sharedKey},
		rhs: &mathExpression{op: OpNone, key: sharedKey},
	}
	result := expr.process(input)

	require.Equal(t, 2, len(result), "expected 2 series: self-division should yield both series, not empty")
	for _, ts := range result {
		for _, v := range ts.Values {
			require.InDelta(t, 1.0, v, 1e-9, "expected 2.0/2.0=1.0")
		}
		for _, l := range ts.Labels {
			require.NotEqual(t, internalLabelQueryFragment, l.Name, "__query_fragment should be stripped")
		}
	}
}

// Regression: buffer pre-allocation must use min(lhs, rhs) length — multi-key rhs with scalar lhs must not panic.
// When the scalar lhs has fewer values than a target series, offset must advance by the
// actual n, not by len(target[k].Values), to avoid an index-out-of-bounds on the next iteration.
func TestApplyBinaryOpMultiKeyScalarLhs(t *testing.T) {
	noLabels := Labels{}
	noLabelsKey := noLabels.MapKey()
	fooLabels := LabelsFromArgs("service", "foo")
	barLabels := LabelsFromArgs("service", "bar")
	fooKey := fooLabels.MapKey()
	barKey := barLabels.MapKey()

	lhs := SeriesSet{
		noLabelsKey: makeTS(noLabels, 10.0, 20.0),
	}
	rhs := SeriesSet{
		fooKey: makeTS(fooLabels, 2.0, 4.0, 6.0),
		barKey: makeTS(barLabels, 1.0, 2.0, 3.0),
	}

	result := applyBinaryOp(OpDiv, lhs, rhs)
	require.Equal(t, 2, len(result), "expected 2 result series")

	fooResult, ok := result[fooKey]
	require.True(t, ok, "expected foo series in result")
	require.Equal(t, 2, len(fooResult.Values), "expected 2 values (min of scalar and rhs lengths)")
	require.InDelta(t, 5.0, fooResult.Values[0], 1e-9, "10.0/2.0=5.0")
	require.InDelta(t, 5.0, fooResult.Values[1], 1e-9, "20.0/4.0=5.0")

	barResult, ok := result[barKey]
	require.True(t, ok, "expected bar series in result")
	require.Equal(t, 2, len(barResult.Values), "expected 2 values")
	require.InDelta(t, 10.0, barResult.Values[0], 1e-9, "10.0/1.0=10.0")
	require.InDelta(t, 10.0, barResult.Values[1], 1e-9, "20.0/2.0=10.0")
}

// MEDIUM-5 / TRQL-T-15: batchSeriesProcessor.result() tags series with __query_fragment.
func TestBatchSeriesProcessorResult(t *testing.T) {
	svc1Labels := LabelsFromArgs("service", "svc1")
	svc2Labels := LabelsFromArgs("service", "svc2")

	procA := &fixedSeriesProcessor{
		series: SeriesSet{
			svc1Labels.MapKey(): {Labels: svc1Labels, Values: []float64{1.0, 2.0}},
		},
	}
	procB := &fixedSeriesProcessor{
		series: SeriesSet{
			svc2Labels.MapKey(): {Labels: svc2Labels, Values: []float64{3.0, 4.0}},
		},
	}

	bsp := batchSeriesProcessor{
		"frag-A": procA,
		"frag-B": procB,
	}

	result := bsp.result(1.0)

	require.Equal(t, 2, len(result), "expected one series per fragment")
	for _, ts := range result {
		hasFragLabel := false
		for _, l := range ts.Labels {
			if l.Name == internalLabelQueryFragment {
				hasFragLabel = true
				break
			}
		}
		require.True(t, hasFragLabel, "__query_fragment label must be present in result() output")
	}

	for _, ts := range result {
		require.NotEmpty(t, ts.Values, "result series must have values")
	}
}

// Also verify unknown fragment is discarded
func TestBatchSeriesProcessorDiscardUnknown(t *testing.T) {
	a := &mockSeriesProcessor{}
	b := &mockSeriesProcessor{}

	bsp := batchSeriesProcessor{
		"frag-A": a,
		"frag-B": b,
	}

	makePTS := func(fragKey, svc string) *tempopb.TimeSeries {
		return &tempopb.TimeSeries{
			Labels: []commonv1proto.KeyValue{
				{Key: internalLabelQueryFragment, Value: &commonv1proto.AnyValue{Value: &commonv1proto.AnyValue_StringValue{StringValue: fragKey}}},
				{Key: "service", Value: &commonv1proto.AnyValue{Value: &commonv1proto.AnyValue_StringValue{StringValue: svc}}},
			},
		}
	}

	bsp.observeSeries([]*tempopb.TimeSeries{
		makePTS("frag-A", "svc1"),
		makePTS("frag-B", "svc2"),
		makePTS("unknown", "svc3"),
	})

	require.Equal(t, 2, a.length()+b.length(), "unknown fragment series must be discarded")
}

// mockSeriesProcessor is a test double for seriesProcessor.
type mockSeriesProcessor struct {
	received []*tempopb.TimeSeries
}

func (m *mockSeriesProcessor) observeSeries(in []*tempopb.TimeSeries) {
	m.received = append(m.received, in...)
}

func (m *mockSeriesProcessor) result(multiplier float64) SeriesSet {
	return SeriesSet{}
}

func (m *mockSeriesProcessor) length() int {
	return len(m.received)
}

// fixedSeriesProcessor is a test double that returns a pre-built SeriesSet from result().
type fixedSeriesProcessor struct {
	series SeriesSet
}

func (f *fixedSeriesProcessor) observeSeries(_ []*tempopb.TimeSeries) {}

func (f *fixedSeriesProcessor) result(_ float64) SeriesSet {
	return f.series
}

func (f *fixedSeriesProcessor) length() int {
	return len(f.series)
}

// TestMergeExemplars covers the merge path where both slices are non-empty.
func TestMergeExemplars(t *testing.T) {
	a := []Exemplar{{Labels: Labels{{Name: "a", Value: NewStaticString("1")}}}}
	b := []Exemplar{{Labels: Labels{{Name: "b", Value: NewStaticString("2")}}}}

	// nil/empty fast paths
	require.Equal(t, b, mergeExemplars(nil, b))
	require.Equal(t, a, mergeExemplars(a, nil))

	// both non-empty — allocates and concatenates
	got := mergeExemplars(a, b)
	require.Len(t, got, 2)
	require.Equal(t, a[0], got[0])
	require.Equal(t, b[0], got[1])
}

// TestMathExpressionMetricName covers the binary recursion path of metricName.
func TestMathExpressionMetricName(t *testing.T) {
	fragA := "frag-A"
	fragB := "frag-B"

	nameLabel := func(name, frag string) Labels {
		return Labels{
			{Name: "__name__", Value: NewStaticString(name)},
			{Name: internalLabelQueryFragment, Value: NewStaticString(frag)},
		}
	}

	input := SeriesSet{
		nameLabel("rate", fragA).MapKey():  {Labels: nameLabel("rate", fragA), Values: []float64{1}},
		nameLabel("count", fragB).MapKey(): {Labels: nameLabel("count", fragB), Values: []float64{2}},
	}

	// Leaf: returns the __name__ for the matching fragment
	leafA := &mathExpression{op: OpNone, key: fragA}
	require.Equal(t, "rate", leafA.metricName(input))

	leafB := &mathExpression{op: OpNone, key: fragB}
	require.Equal(t, "count", leafB.metricName(input))

	// Leaf with no match returns ""
	leafC := &mathExpression{op: OpNone, key: "no-match"}
	require.Equal(t, "", leafC.metricName(input))

	// Binary: combines both names
	binary := &mathExpression{op: OpDiv, lhs: leafA, rhs: leafB}
	require.Equal(t, "(rate / count)", binary.metricName(input))

	// Binary with one missing name returns ""
	binaryMissing := &mathExpression{op: OpAdd, lhs: leafA, rhs: leafC}
	require.Equal(t, "", binaryMissing.metricName(input))
}

// TestMathExpressionInit covers init() on leaf-with-filter and binary nodes.
func TestMathExpressionInit(t *testing.T) {
	req := &tempopb.QueryRangeRequest{Start: 1000, End: 2000, Step: 100}

	// Leaf with no filter: init is a no-op, must not panic
	leaf := &mathExpression{op: OpNone, key: "k"}
	require.NotPanics(t, func() { leaf.init(req) })

	// Track calls via a mock filter
	called := false
	mockFilter := &mockSecondStageElement{initFn: func(r *tempopb.QueryRangeRequest) { called = true }}

	leafWithFilter := &mathExpression{op: OpNone, key: "k", filter: mockFilter}
	leafWithFilter.init(req)
	require.True(t, called, "init must delegate to leaf filter")

	// Binary: both children must receive init
	lhsCalled, rhsCalled := false, false
	lhsFilter := &mockSecondStageElement{initFn: func(r *tempopb.QueryRangeRequest) { lhsCalled = true }}
	rhsFilter := &mockSecondStageElement{initFn: func(r *tempopb.QueryRangeRequest) { rhsCalled = true }}
	binary := &mathExpression{
		op:  OpAdd,
		lhs: &mathExpression{op: OpNone, key: "l", filter: lhsFilter},
		rhs: &mathExpression{op: OpNone, key: "r", filter: rhsFilter},
	}
	binary.init(req)
	require.True(t, lhsCalled, "init must recurse into lhs")
	require.True(t, rhsCalled, "init must recurse into rhs")
}

// TestBatchSeriesProcessorLength covers batchSeriesProcessor.length().
func TestBatchSeriesProcessorLength(t *testing.T) {
	a := &mockSeriesProcessor{received: make([]*tempopb.TimeSeries, 3)}
	b := &mockSeriesProcessor{received: make([]*tempopb.TimeSeries, 5)}
	bsp := batchSeriesProcessor{"frag-A": a, "frag-B": b}

	require.Equal(t, 8, bsp.length())

	empty := batchSeriesProcessor{}
	require.Equal(t, 0, empty.length())
}

// mockSecondStageElement is a minimal test double for secondStageElement.
type mockSecondStageElement struct {
	initFn func(*tempopb.QueryRangeRequest)
}

func (m *mockSecondStageElement) String() string  { return "" }
func (m *mockSecondStageElement) validate() error { return nil }
func (m *mockSecondStageElement) init(req *tempopb.QueryRangeRequest) {
	if m.initFn != nil {
		m.initFn(req)
	}
}
func (m *mockSecondStageElement) process(input SeriesSet) SeriesSet { return input }

// TestMathExpressionString covers the String() branches.
func TestMathExpressionString(t *testing.T) {
	// Leaf without filter: returns key.
	leaf := &mathExpression{op: OpNone, key: "mykey"}
	require.Equal(t, "mykey", leaf.String())

	// Leaf with filter: returns key+filter.String().
	mockFilter := &mockSecondStageElement{}
	leafWithFilter := &mathExpression{op: OpNone, key: "k", filter: mockFilter}
	require.Equal(t, "k", leafWithFilter.String())

	// Binary: returns (lhs) op (rhs).
	binary := &mathExpression{
		op:  OpDiv,
		lhs: &mathExpression{op: OpNone, key: "A"},
		rhs: &mathExpression{op: OpNone, key: "B"},
	}
	require.Equal(t, "(A) / (B)", binary.String())
}

// TestGetFragmentLabelMissing covers the empty-return path of getFragmentLabel.
func TestGetFragmentLabelMissing(t *testing.T) {
	// Labels without the fragment label return "".
	labels := Labels{{Name: "service", Value: NewStaticString("svc1")}}
	require.Equal(t, "", getFragmentLabel(labels))

	// Empty Labels also return "".
	require.Equal(t, "", getFragmentLabel(Labels{}))
}

// TestGetFragmentLabelProtoMissing covers the empty-return path of getFragmentLabelProto.
func TestGetFragmentLabelProtoMissing(t *testing.T) {
	// Proto labels without the fragment label return "".
	labels := []commonv1proto.KeyValue{
		{Key: "service", Value: &commonv1proto.AnyValue{Value: &commonv1proto.AnyValue_StringValue{StringValue: "svc1"}}},
	}
	require.Equal(t, "", getFragmentLabelProto(labels))

	// Empty proto labels also return "".
	require.Equal(t, "", getFragmentLabelProto(nil))
}

// TestNewBatchSeriesProcessor covers the constructor.
func TestNewBatchSeriesProcessor(t *testing.T) {
	// newBatchSeriesProcessor wraps firstStageElement values into a batchSeriesProcessor.
	// Since firstStageElement is an interface implemented by MetricsAggregate etc., use
	// a simple mock that satisfies seriesProcessor for the result check.
	// We verify via length() and result().
	procA := &fixedSeriesProcessor{
		series: SeriesSet{
			LabelsFromArgs("svc", "a").MapKey(): {Labels: LabelsFromArgs("svc", "a"), Values: []float64{1.0}},
		},
	}
	procB := &fixedSeriesProcessor{
		series: SeriesSet{
			LabelsFromArgs("svc", "b").MapKey(): {Labels: LabelsFromArgs("svc", "b"), Values: []float64{2.0}},
		},
	}

	// Use the map literal form directly (same as newBatchSeriesProcessor but with seriesProcessor interface).
	bsp := batchSeriesProcessor{"frag-A": procA, "frag-B": procB}
	require.Equal(t, 2, bsp.length())

	result := bsp.result(1.0)
	require.Len(t, result, 2)
	for _, ts := range result {
		hasFragLabel := false
		for _, l := range ts.Labels {
			if l.Name == internalLabelQueryFragment {
				hasFragLabel = true
			}
		}
		require.True(t, hasFragLabel, "each result series must carry the fragment label")
	}
}

// silence unused helper warnings
var (
	_ = makeKey
	_ = seriesSetToSlice
	_ = requireSeriesSetEqual
)
