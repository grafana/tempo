package traceql

import (
	"math"
	"testing"
	"time"

	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/stretchr/testify/require"
)

func TestAvgOverTime(t *testing.T) {
	req := &tempopb.QueryRangeRequest{
		Start: 1,
		End:   uint64(3 * time.Second),
		Step:  uint64(1 * time.Second),
		Query: "{ } | avg_over_time(duration) by (span.service)",
	}

	// Test data designed to verify multiplication accuracy in weighted mean calculation
	// The addWeigthedMean function uses: meanDelta := ((mean - currentMean.mean) * weight) / sumWeights
	in := []Span{
		// Time interval 1: service=a has 2 spans with durations 100, 200 (mean=150, count=2)
		newMockSpan(nil).WithStartTime(uint64(1*time.Second)).WithSpanString("service", "a").WithDuration(uint64(100 * time.Second)),
		newMockSpan(nil).WithStartTime(uint64(1*time.Second)).WithSpanString("service", "a").WithDuration(uint64(200 * time.Second)),

		// Time interval 1: service=b has 3 spans with durations 300, 600, 900 (mean=600, count=3)
		newMockSpan(nil).WithStartTime(uint64(1*time.Second)).WithSpanString("service", "b").WithDuration(uint64(300 * time.Second)),
		newMockSpan(nil).WithStartTime(uint64(1*time.Second)).WithSpanString("service", "b").WithDuration(uint64(600 * time.Second)),
		newMockSpan(nil).WithStartTime(uint64(1*time.Second)).WithSpanString("service", "b").WithDuration(uint64(900 * time.Second)),

		// Time interval 2: service=a has 4 spans with durations 400, 800, 1200, 1600 (mean=1000, count=4)
		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("service", "a").WithDuration(uint64(400 * time.Second)),
		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("service", "a").WithDuration(uint64(800 * time.Second)),
		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("service", "a").WithDuration(uint64(1200 * time.Second)),
		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("service", "a").WithDuration(uint64(1600 * time.Second)),

		// Time interval 2: service=b has 1 span with duration 2000 (mean=2000, count=1)
		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("service", "b").WithDuration(uint64(2000 * time.Second)),

		// Time interval 3: service=a has varying precision values to test floating point accuracy
		newMockSpan(nil).WithStartTime(uint64(3*time.Second)).WithSpanString("service", "a").WithDuration(uint64(123 * time.Second)),
		newMockSpan(nil).WithStartTime(uint64(3*time.Second)).WithSpanString("service", "a").WithDuration(uint64(456 * time.Second)),
		newMockSpan(nil).WithStartTime(uint64(3*time.Second)).WithSpanString("service", "a").WithDuration(uint64(789 * time.Second)),
	}

	result, _, err := runTraceQLMetric(req, in)
	require.NoError(t, err)
	require.Equal(t, 2, len(result))

	actualServiceA := result[LabelsFromArgs("span.service", "a").MapKey()]
	actualServiceB := result[LabelsFromArgs("span.service", "b").MapKey()]

	expectedServiceA := []struct {
		interval int
		expected float64
	}{
		{0, 150.0},  // (100 + 200) / 2 = 150 s
		{1, 1000.0}, // (400 + 800 + 1200 + 1600) / 4 = 1000 s
		{2, 456.0},  // (123 + 456 + 789) / 3 = 456 s
	}
	expectedServiceB := []struct {
		interval int
		expected float64
		isNaN    bool
	}{
		{0, 600.0, false},  // (300 + 600 + 900) / 3 = 600 s
		{1, 2000.0, false}, // 2000 / 1 = 2000 s
		{2, 0.0, true},     // no spans, should be NaN
	}

	// Verify multiplication accuracy for service A
	for _, tc := range expectedServiceA {
		actual := actualServiceA.Values[tc.interval]
		require.InDelta(t, tc.expected, actual, 0.001,
			"Service A interval %d: expected %v, got %v",
			tc.interval, tc.expected, actual)
	}

	// Verify multiplication accuracy for service B
	for _, tc := range expectedServiceB {
		if tc.isNaN {
			require.True(t, math.IsNaN(actualServiceB.Values[tc.interval]),
				"Service B interval %d should be NaN", tc.interval)
		} else {
			actual := actualServiceB.Values[tc.interval]
			require.InDelta(t, tc.expected, actual, 0.001,
				"Service B interval %d: expected %v, got %v",
				tc.interval, tc.expected, actual)
		}
	}
}

func TestAvgOverTimeScalesResults(t *testing.T) {
	// Test data designed to verify multiplication accuracy in weighted mean calculation
	// The addWeigthedMean function uses: meanDelta := ((mean - currentMean.mean) * weight) / sumWeights
	in := []Span{
		// Time interval 1: service=a has 2 spans with durations 100, 200 (mean=150, count=2)
		newMockSpan(nil).WithStartTime(uint64(1*time.Second)).WithSpanString("service", "a").WithDuration(uint64(100 * time.Second)),
		newMockSpan(nil).WithStartTime(uint64(1*time.Second)).WithSpanString("service", "a").WithDuration(uint64(200 * time.Second)),

		// Time interval 1: service=b has 3 spans with durations 300, 600, 900 (mean=600, count=3)
		newMockSpan(nil).WithStartTime(uint64(1*time.Second)).WithSpanString("service", "b").WithDuration(uint64(300 * time.Second)),
		newMockSpan(nil).WithStartTime(uint64(1*time.Second)).WithSpanString("service", "b").WithDuration(uint64(600 * time.Second)),
		newMockSpan(nil).WithStartTime(uint64(1*time.Second)).WithSpanString("service", "b").WithDuration(uint64(900 * time.Second)),

		// Time interval 2: service=a has 4 spans with durations 400, 800, 1200, 1600 (mean=1000, count=4)
		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("service", "a").WithDuration(uint64(400 * time.Second)),
		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("service", "a").WithDuration(uint64(800 * time.Second)),
		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("service", "a").WithDuration(uint64(1200 * time.Second)),
		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("service", "a").WithDuration(uint64(1600 * time.Second)),

		// Time interval 2: service=b has 1 span with duration 2000 (mean=2000, count=1)
		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("service", "b").WithDuration(uint64(2000 * time.Second)),

		// Time interval 3: service=a has varying precision values to test floating point accuracy
		newMockSpan(nil).WithStartTime(uint64(3*time.Second)).WithSpanString("service", "a").WithDuration(uint64(123 * time.Second)),
		newMockSpan(nil).WithStartTime(uint64(3*time.Second)).WithSpanString("service", "a").WithDuration(uint64(456 * time.Second)),
		newMockSpan(nil).WithStartTime(uint64(3*time.Second)).WithSpanString("service", "a").WithDuration(uint64(789 * time.Second)),
	}

	a := newAverageOverTimeMetricsAggregator(IntrinsicDurationAttribute, []Attribute{NewAttribute("service")})
	a.init(&tempopb.QueryRangeRequest{
		Start: 1,
		End:   uint64(3 * time.Second),
		Step:  uint64(1 * time.Second),
	}, AggregateModeRaw)

	for _, span := range in {
		a.observe(span)
	}

	ss := a.result(1.0)
	expected := []TimeSeries{
		{
			Labels: Labels{
				Label{Name: ".service", Value: NewStaticString("a")},
			},
			Values:    []float64{150.0, 1000.0, 456.0},
			Exemplars: []Exemplar{},
		},
		{
			Labels: Labels{
				Label{Name: ".service", Value: NewStaticString("a")},
				Label{internalLabelMetaType, NewStaticString(internalMetaTypeCount)},
			},
			Values: []float64{2, 4, 3},
		},
		{
			Labels: Labels{
				Label{Name: ".service", Value: NewStaticString("b")},
			},
			Values:    []float64{600.0, 2000.0, math.NaN()},
			Exemplars: []Exemplar{},
		},
		{
			Labels: Labels{
				Label{Name: ".service", Value: NewStaticString("b")},
				Label{internalLabelMetaType, NewStaticString(internalMetaTypeCount)},
			},
			Values: []float64{3, 1, math.NaN()},
		},
	}
	requireEqualSeriesSets(t, expected, ss)

	// Now check that scaling only effects the counts.
	ss2 := a.result(2.0)
	expected2 := []TimeSeries{
		{
			Labels: Labels{
				Label{Name: ".service", Value: NewStaticString("a")},
			},
			Values:    []float64{150.0, 1000.0, 456.0},
			Exemplars: []Exemplar{},
		},
		{
			Labels: Labels{
				Label{Name: ".service", Value: NewStaticString("a")},
				Label{internalLabelMetaType, NewStaticString(internalMetaTypeCount)},
			},
			Values: []float64{4, 8, 6},
		},
		{
			Labels: Labels{
				Label{Name: ".service", Value: NewStaticString("b")},
			},
			Values:    []float64{600.0, 2000.0, math.NaN()},
			Exemplars: []Exemplar{},
		},
		{
			Labels: Labels{
				Label{Name: ".service", Value: NewStaticString("b")},
				Label{internalLabelMetaType, NewStaticString(internalMetaTypeCount)},
			},
			Values: []float64{6, 2, math.NaN()},
		},
	}
	requireEqualSeriesSets(t, expected2, ss2)
}
