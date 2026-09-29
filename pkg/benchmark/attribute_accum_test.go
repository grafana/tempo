package benchmark

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/v3/pkg/traceql"
)

func TestAccumulatorStringValues(t *testing.T) {
	a := newAttrAccumulator(attrKey{AttributeScopeSpan, "http.method", AttributeTypeString})
	get, post := traceql.NewStaticString("GET"), traceql.NewStaticString("POST")
	for _, v := range []traceql.Static{get, get, post} {
		a.observe([]traceql.Static{v}, 1)
	}

	require.Equal(t, AttributeProfile{
		Scope: AttributeScopeSpan, Name: "http.method", Type: AttributeTypeString,
		TotalBytes: 10, Cardinality: 2, Density: 0.75,
		Values: []AttributeValue{{Value: "GET", Selectivity: 0.5}, {Value: "POST", Selectivity: 0.25}},
	}, a.finish(4))
}

// A resource attribute is stored once but matches every span beneath it, so
// its bytes count once and its values count for each of those spans.
func TestAccumulatorWeightsValuesBySpans(t *testing.T) {
	a := newAttrAccumulator(attrKey{AttributeScopeResource, "service.name", AttributeTypeString})
	a.observe([]traceql.Static{traceql.NewStaticString("svc-a")}, 3)
	a.observe([]traceql.Static{traceql.NewStaticString("svc-b")}, 1)

	require.Equal(t, AttributeProfile{
		Scope: AttributeScopeResource, Name: "service.name", Type: AttributeTypeString,
		TotalBytes: 10, Cardinality: 2, Density: 0.5,
		Values: []AttributeValue{{Value: "svc-a", Selectivity: 0.375}, {Value: "svc-b", Selectivity: 0.125}},
	}, a.finish(8))
}

// An array matches `= v` once per span however often it repeats v, but every
// element counts towards its bytes.
func TestAccumulatorArrayCountsSpanOncePerValue(t *testing.T) {
	a := newAttrAccumulator(attrKey{AttributeScopeSpan, "tags", AttributeTypeString})
	a.observe([]traceql.Static{traceql.NewStaticString("a"), traceql.NewStaticString("b"), traceql.NewStaticString("a")}, 1)

	p := a.finish(2)
	require.Equal(t, uint64(3), p.TotalBytes)
	require.Equal(t, uint64(2), p.Cardinality)
	require.Equal(t, 0.5, p.Density)
	require.Equal(t, []AttributeValue{{Value: "a", Selectivity: 0.5}, {Value: "b", Selectivity: 0.5}}, p.Values)
}

func TestAccumulatorNumericSwitchesToQuantiles(t *testing.T) {
	profile := func(distinct int) AttributeProfile {
		a := newAttrAccumulator(attrKey{AttributeScopeSpan, "n", AttributeTypeInt})
		for i := range distinct {
			a.observe([]traceql.Static{traceql.NewStaticInt(i)}, 1)
		}
		return a.finish(uint64(distinct))
	}

	atLimit := profile(topValues)
	require.Len(t, atLimit.Values, topValues)
	require.Empty(t, atLimit.Quantiles)
	require.Equal(t, uint64(topValues), atLimit.Cardinality)
	require.NoError(t, atLimit.validate())

	past := profile(topValues + 1)
	require.Empty(t, past.Values)
	require.Len(t, past.Quantiles, len(profileQuantiles))
	require.Greater(t, past.Cardinality, uint64(topValues))
	require.NoError(t, past.validate())
}

// An integer quantile is rendered as the floor of its bucket bound, which
// splits the integers exactly where the bound does.
func TestAccumulatorIntQuantileSelectivityIsExact(t *testing.T) {
	const n = 1000
	a := newAttrAccumulator(attrKey{AttributeScopeSpan, "n", AttributeTypeInt})
	for i := range n {
		a.observe([]traceql.Static{traceql.NewStaticInt(i - n/2)}, 1)
	}

	p := a.finish(n)
	require.Len(t, p.Quantiles, len(profileQuantiles))
	for _, q := range p.Quantiles {
		v, err := strconv.Atoi(q.Value)
		require.NoError(t, err)

		above := 0
		for i := range n {
			if i-n/2 > v {
				above++
			}
		}
		require.InDelta(t, float64(above)/n, q.Selectivity, 1e-12, "q=%v value=%v", q.Q, q.Value)
	}
}

// `> x` matches a span if any element of its array exceeds x, which is when
// its largest element does.
func TestAccumulatorNumericArrayQuantilesUseLargestElement(t *testing.T) {
	const n = 200
	a := newAttrAccumulator(attrKey{AttributeScopeSpan, "n", AttributeTypeInt})
	for i := range n {
		a.observe([]traceql.Static{traceql.NewStaticInt(-i), traceql.NewStaticInt(i)}, 1)
	}

	p := a.finish(n)
	for _, q := range p.Quantiles {
		v, err := strconv.Atoi(q.Value)
		require.NoError(t, err)

		above := 0
		for i := range n {
			if i > v {
				above++
			}
		}
		require.InDelta(t, float64(above)/n, q.Selectivity, 1e-12, "q=%v value=%v", q.Q, q.Value)
	}
}

func TestAccumulatorDurationQuantilesRenderAsDurations(t *testing.T) {
	a := newAttrAccumulator(attrKey{AttributeScopeIntrinsic, "duration", AttributeTypeDuration})
	for i := range 500 {
		a.observe([]traceql.Static{traceql.NewStaticDuration(time.Duration(i+1) * time.Millisecond)}, 1)
	}

	p := a.finish(500)
	require.Len(t, p.Quantiles, len(profileQuantiles))
	for _, q := range p.Quantiles {
		_, err := time.ParseDuration(q.Value)
		require.NoError(t, err, q.Value)
	}
}

// Past valueCounters distinct values the counts are no longer exact, so the
// cardinality comes from the sketch.
func TestAccumulatorHighCardinalityString(t *testing.T) {
	const n = 3 * valueCounters
	a := newAttrAccumulator(attrKey{AttributeScopeSpan, "url", AttributeTypeString})
	for i := range n {
		a.observe([]traceql.Static{traceql.NewStaticString(fmt.Sprintf("/u/%d", i))}, 1)
	}

	p := a.finish(n)
	require.Len(t, p.Values, topValues)
	require.InEpsilon(t, float64(n), float64(p.Cardinality), 0.05)
	require.NoError(t, p.validate())
}

func TestAccumulatorBytesByType(t *testing.T) {
	for _, tc := range []struct {
		typ  string
		v    traceql.Static
		want uint64
	}{
		{AttributeTypeString, traceql.NewStaticString("abcd"), 4},
		{AttributeTypeInt, traceql.NewStaticInt(7), 8},
		{AttributeTypeFloat, traceql.NewStaticFloat(1.5), 8},
		{AttributeTypeDuration, traceql.NewStaticDuration(time.Second), 8},
		{AttributeTypeBool, traceql.NewStaticBool(true), 1},
		{AttributeTypeStatus, traceql.NewStaticStatus(traceql.StatusError), 1},
		{AttributeTypeKind, traceql.NewStaticKind(traceql.KindServer), 1},
	} {
		a := newAttrAccumulator(attrKey{AttributeScopeSpan, "x", tc.typ})
		a.observe([]traceql.Static{tc.v}, 1)
		require.Equal(t, tc.want, a.finish(1).TotalBytes, tc.typ)
	}
}
