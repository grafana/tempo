package traceql

// ast_metrics_math_test.go contains tests for confirmed bugs in the math
// expression evaluation pipeline.  Each test asserts CORRECT (desired)
// behaviour so that it FAILS against the current code and PASSES once the
// corresponding bug is fixed.

import (
	"math"
	"testing"

	"github.com/grafana/tempo/pkg/tempopb"
	commonv1proto "github.com/grafana/tempo/pkg/tempopb/common/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFrontendMathEvaluatorSubtractionReturnsCorrectValues exercises the full
// frontend path for a math query compiled in AggregateModeFinal.
//
// Bug: batchSeriesProcessor.result() merges all sub-query results WITHOUT
// adding __query_fragment labels back to the keys.  When
// mathExpression.processLeaf reads the merged SeriesSet it checks
//
//	fv.Type != TypeNil
//
// to decide whether a series belongs to a specific leaf.  Because the
// fragment label was never stored on the map key, GetValue always returns
// TypeNil, so BOTH leaves receive ALL series.  Both LHS and RHS therefore
// get the same data and, for subtraction, produce 0 instead of the expected
// non-zero difference.
//
// Expected (correct) behaviour: LHS values [10, 20, 30] minus RHS values
// [1, 2, 3] should yield [9, 18, 27].  At minimum, not all results should
// be zero.
func TestFrontendMathEvaluatorSubtractionReturnsCorrectValues(t *testing.T) {
	const query = `({} | rate()) - ({} | count_over_time())`

	req := &tempopb.QueryRangeRequest{
		Query: query,
		Start: 1000,
		End:   4000,
		Step:  1000,
	}

	eval, err := NewEngine().CompileMetricsQueryRangeNonRaw(req, AggregateModeFinal)
	require.NoError(t, err)
	require.NotNil(t, eval)

	// Discover the two sub-query fragment keys from the parsed expression.
	expr, err := Parse(query)
	require.NoError(t, err)
	require.Len(t, expr.SeriesProcessor, 2, "expected exactly two sub-query processors for a binary math expression")

	keys := make([]string, 0, 2)
	for k := range expr.SeriesProcessor {
		keys = append(keys, k)
	}

	// Assign deterministic roles: lhsKey is keys[0], rhsKey is keys[1].
	// The exact strings match the canonical form produced by the parser
	// (e.g. "{ } | rate()" and "{ } | count_over_time()").
	lhsKey := keys[0]
	rhsKey := keys[1]

	lhsValues := []tempopb.Sample{
		{TimestampMs: 1000, Value: 10.0},
		{TimestampMs: 2000, Value: 20.0},
		{TimestampMs: 3000, Value: 30.0},
	}
	rhsValues := []tempopb.Sample{
		{TimestampMs: 1000, Value: 1.0},
		{TimestampMs: 2000, Value: 2.0},
		{TimestampMs: 3000, Value: 3.0},
	}

	makeTaggedSeries := func(fragmentKey string, samples []tempopb.Sample) *tempopb.TimeSeries {
		return &tempopb.TimeSeries{
			Labels: []commonv1proto.KeyValue{
				{
					Key: internalLabelQueryFragment,
					Value: &commonv1proto.AnyValue{
						Value: &commonv1proto.AnyValue_StringValue{StringValue: fragmentKey},
					},
				},
			},
			Samples: samples,
		}
	}

	eval.ObserveSeries([]*tempopb.TimeSeries{
		makeTaggedSeries(lhsKey, lhsValues),
		makeTaggedSeries(rhsKey, rhsValues),
	})

	result := eval.Results()
	require.NotEmpty(t, result, "expected at least one series in result")

	// With the bug the subtraction of identical data produces all zeros.
	// After the fix the result for lhsKey-series minus rhsKey-series must
	// contain at least one non-zero value.
	allZero := true
	for _, ts := range result {
		for _, v := range ts.Values {
			if !math.IsNaN(v) && v != 0 {
				allZero = false
			}
		}
	}
	assert.False(t, allZero, "subtraction of distinct series must produce non-zero values; got all zeros (bug: both leaves receive the same data)")
}

// TestApplyBinaryOpLHSOnlySeriesPreserved verifies that applyBinaryOp
// retains series that are present ONLY in LHS.
//
// Bug: applyBinaryOp unconditionally switches the iteration target to rhs
// when rhs does not contain noLabelsSeriesMapKey (the scalar sentinel).
// Iterating only rhs means any key that exists solely in lhs is silently
// dropped rather than appearing in the result (with NaN for the missing rhs
// match).
//
// Expected (correct) behaviour: the result contains every key from lhs,
// with a NaN or zero value where no rhs counterpart exists.
func TestApplyBinaryOpLHSOnlySeriesPreserved(t *testing.T) {
	// Build two distinct SeriesMapKeys representing series "A" and "B".
	keyA := SeriesMapKey{}
	keyA[0] = SeriesMapLabel{Name: "span.service", Value: NewStaticString("alpha").MapKey()}

	keyB := SeriesMapKey{}
	keyB[0] = SeriesMapLabel{Name: "span.service", Value: NewStaticString("beta").MapKey()}

	lhs := SeriesSet{
		keyA: TimeSeries{Values: []float64{5.0}},
		keyB: TimeSeries{Values: []float64{10.0}},
	}
	// rhs only has keyB; keyA is absent.
	rhs := SeriesSet{
		keyB: TimeSeries{Values: []float64{3.0}},
	}

	result := applyBinaryOp(OpSub, lhs, rhs)

	// keyB should be present and equal to 10 - 3 = 7.
	require.Contains(t, result, keyB, "keyB must appear in result")
	assert.Equal(t, 7.0, result[keyB].Values[0], "keyB value should be lhs - rhs = 7")

	// keyA is LHS-only: it must appear in the result (the missing rhs
	// match is treated as NaN or 0).
	// Currently fails: applyBinaryOp iterates rhs, so keyA is silently dropped.
	assert.Contains(t, result, keyA, "LHS-only series (keyA) must appear in result; currently dropped because only rhs is iterated")
}

// TestProcessLeafExemplarsCopied verifies that processLeaf performs a deep
// copy of Exemplars, honouring the "must not mutate input" contract stated in
// the comment above processLeaf.
//
// Bug: processLeaf deep-copies Values but assigns Exemplars by sharing the
// slice directly:
//
//	Exemplars: v.Exemplars
//
// Mutating the result's Exemplars therefore corrupts the original input.
//
// Expected (correct) behaviour: mutating exemplar fields on the returned
// series must not change the input series.
func TestProcessLeafExemplarsCopied(t *testing.T) {
	inputExemplars := []Exemplar{
		{Value: 1.0, TimestampMs: 1000},
	}

	// Build an input SeriesSet that processLeaf will pass through.
	// Using an empty key string means the leaf matches all series that have no
	// __query_fragment label (TypeNil path in processLeaf).
	inputKey := SeriesMapKey{}
	input := SeriesSet{
		inputKey: TimeSeries{
			Labels:    Labels{},
			Values:    []float64{42.0},
			Exemplars: inputExemplars,
		},
	}

	expr := newFlatExpression("", nil)
	result := expr.process(input)

	require.Contains(t, result, inputKey, "processLeaf must keep the series in the result")

	// Mutate the exemplar in the result.
	ts := result[inputKey]
	require.NotEmpty(t, ts.Exemplars, "result must have exemplars")
	ts.Exemplars[0].Value = 999.0
	result[inputKey] = ts

	// The original input exemplar must be unaffected.
	// Currently fails: both slices share the same backing array.
	assert.Equal(t, 1.0, input[inputKey].Exemplars[0].Value,
		"input exemplars must not be modified by mutations to processLeaf result (shallow copy bug)")
}

// TestRootExprStringZeroValue verifies that RootExpr{}.String() does not
// panic.
//
// Bug: ast_stringer.go calls r.expression.String() with no nil guard.
// A zero-value RootExpr has expression == nil, causing a nil pointer
// dereference.
//
// Expected (correct) behaviour: String() on a zero-value RootExpr returns
// some non-panicking string (empty string or a safe placeholder).
func TestRootExprStringZeroValue(t *testing.T) {
	// Currently panics: nil pointer dereference on r.expression.String()
	assert.NotPanics(t, func() {
		_ = RootExpr{}.String()
	}, "RootExpr{}.String() must not panic on zero-value struct")
}

// TestExtractFetchRequestMathExpressionIncludesAllConditions verifies that
// ExtractFetchRequest returns conditions from ALL sub-queries in a math
// expression, not just the first one encountered during map iteration.
//
// Bug: lenient_extract.go iterates the map of sub-query requests and breaks
// on the very first entry:
//
//	for _, v := range requests {
//	    req = v
//	    break
//	}
//
// For a math expression like A + B, only one sub-query's conditions are
// returned; the other is silently discarded.
//
// Expected (correct) behaviour: the returned FetchSpansRequest must contain
// conditions from every sub-query.
func TestExtractFetchRequestMathExpressionIncludesAllConditions(t *testing.T) {
	query := `({span.http.url = "/api"} | rate()) + ({span.status_code = "200"} | rate())`

	req := ExtractFetchRequest(query)
	require.NotNil(t, req, "ExtractFetchRequest must return a non-nil request for a valid math query")

	conditionNames := make([]string, 0, len(req.Conditions))
	for _, c := range req.Conditions {
		conditionNames = append(conditionNames, c.Attribute.Name)
	}

	// Currently only one sub-query's conditions are present because the loop
	// breaks after the first map entry.
	assert.Contains(t, conditionNames, "http.url",
		"first sub-query condition (http.url) must be present in ExtractFetchRequest result")
	assert.Contains(t, conditionNames, "status_code",
		"second sub-query condition (status_code) must be present in ExtractFetchRequest result")
}

// TestApplyArithmeticOpNaNPropagatesLikePrometheus documents the expected
// IEEE-754 / Prometheus semantics for NaN inputs.
//
// Bug: applyArithmeticOp coerces NaN inputs to 0 before performing
// arithmetic.  This diverges from Prometheus, where any arithmetic involving
// a NaN (which represents "no data") propagates NaN through the result.
//
// Expected (correct) behaviour: NaN op value == NaN, matching Prometheus and
// IEEE 754.
func TestApplyArithmeticOpNaNPropagatesLikePrometheus(t *testing.T) {
	// Currently applyArithmeticOp converts NaN to 0, so NaN+5 returns 5,
	// NaN-5 returns -5, and NaN*5 returns 0 — all non-NaN, all wrong.
	assert.True(t, math.IsNaN(applyArithmeticOp(OpAdd, math.NaN(), 5.0)),
		"NaN + 5 should be NaN (missing data + value = missing); current code returns 5 due to NaN→0 coercion")
	assert.True(t, math.IsNaN(applyArithmeticOp(OpSub, math.NaN(), 5.0)),
		"NaN - 5 should be NaN; current code returns -5 due to NaN→0 coercion")
	assert.True(t, math.IsNaN(applyArithmeticOp(OpMult, math.NaN(), 5.0)),
		"NaN * 5 should be NaN; current code returns 0 due to NaN→0 coercion")
}
