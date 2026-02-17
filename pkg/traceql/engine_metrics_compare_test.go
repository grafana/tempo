package traceql

import (
	"math"
	"testing"
	"time"

	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompare(t *testing.T) {
	// Test that the compare function correctly multiplies results based on sampling multiplier
	// The multiplication happens in result() method: s.Values[i] *= multiplier

	// Instant query
	req := &tempopb.QueryRangeRequest{
		Start: 1,
		End:   uint64(3 * time.Second),
		Step:  uint64(2 * time.Second),
		Query: `{ } | compare({ .service="selected" })`,
	}

	// Test data with clear baseline vs selection distinction
	spans := []Span{
		// Baseline spans (service != "selected")
		newMockSpan(nil).WithStartTime(uint64(1*time.Second)).WithSpanString("service", "baseline1").WithSpanString("environment", "prod"),
		newMockSpan(nil).WithStartTime(uint64(1*time.Second)).WithSpanString("service", "baseline2").WithSpanString("environment", "dev"),
		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("service", "baseline1").WithSpanString("environment", "prod"),

		// Selection spans (service == "selected")
		newMockSpan(nil).WithStartTime(uint64(1*time.Second)).WithSpanString("service", "selected").WithSpanString("environment", "prod"),
		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("service", "selected").WithSpanString("environment", "dev"),
		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("service", "selected").WithSpanString("environment", "dev"),
		newMockSpan(nil).WithStartTime(uint64(3*time.Second)).WithSpanString("service", "selected").WithSpanString("environment", "prod"),
	}

	ss, _, err := runTraceQLMetric(req, spans)
	require.NoError(t, err)

	expected := []TimeSeries{
		// Baseline values
		{
			Labels: Labels{
				internalLabelTypeBaseline,
				Label{Name: "span.service", Value: NewStaticString("baseline1")},
			},
			Values: []float64{2, 0},
		},
		{
			Labels: Labels{
				internalLabelTypeBaseline,
				Label{Name: "span.service", Value: NewStaticString("baseline2")},
			},
			Values: []float64{1, 0},
		},
		{
			Labels: Labels{
				internalLabelTypeBaseline,
				Label{Name: "span.environment", Value: NewStaticString("prod")},
			},
			Values: []float64{2, 0},
		},
		{
			Labels: Labels{
				internalLabelTypeBaseline,
				Label{Name: "span.environment", Value: NewStaticString("dev")},
			},
			Values: []float64{1, 0},
		},
		{
			Labels: Labels{
				internalLabelTypeBaselineTotal,
				Label{Name: "span.service", Value: NewStaticString("nil")},
			},
			Values: []float64{3, 0},
		},
		{
			Labels: Labels{
				internalLabelTypeBaselineTotal,
				Label{Name: "span.environment", Value: NewStaticString("nil")},
			},
			Values: []float64{3, 0},
		},

		// Selection values
		{
			Labels: Labels{
				internalLabelTypeSelection,
				Label{Name: "span.service", Value: NewStaticString("selected")},
			},
			Values: []float64{3, 1},
		},
		{
			Labels: Labels{
				internalLabelTypeSelection,
				Label{Name: "span.environment", Value: NewStaticString("prod")},
			},
			Values: []float64{1, 1},
		},
		{
			Labels: Labels{
				internalLabelTypeSelection,
				Label{Name: "span.environment", Value: NewStaticString("dev")},
			},
			Values: []float64{2, 0},
		},
		{
			Labels: Labels{
				internalLabelTypeSelectionTotal,
				Label{Name: "span.service", Value: NewStaticString("nil")},
			},
			Values: []float64{3, 1},
		},
		{
			Labels: Labels{
				internalLabelTypeSelectionTotal,
				Label{Name: "span.environment", Value: NewStaticString("nil")},
			},
			Values: []float64{3, 1},
		},
	}

	requireEqualSeriesSets(t, expected, ss)
}

func TestCompareScalesResults(t *testing.T) {
	// Test that the compare function correctly multiplies results based on sampling multiplier
	// The multiplication happens in result() method: s.Values[i] *= multiplier

	// Instant query
	req := &tempopb.QueryRangeRequest{
		Start: 1,
		End:   uint64(3 * time.Second),
		Step:  uint64(2 * time.Second),
		Query: `{ } | compare({ .service="selected" })`,
	}

	a := newMetricsCompare(newSpansetFilter(
		newBinaryOperation(OpEqual,
			NewAttribute("service"),
			NewStaticString("selected"),
		)), 10, 0, 0)
	a.init(req, AggregateModeRaw)

	// Test data with clear baseline vs selection distinction
	spans := []Span{
		// Baseline spans (service != "selected")
		newMockSpan(nil).WithStartTime(uint64(1*time.Second)).WithSpanString("service", "baseline1").WithSpanString("environment", "prod"),
		newMockSpan(nil).WithStartTime(uint64(1*time.Second)).WithSpanString("service", "baseline2").WithSpanString("environment", "dev"),
		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("service", "baseline1").WithSpanString("environment", "prod"),

		// Selection spans (service == "selected")
		newMockSpan(nil).WithStartTime(uint64(1*time.Second)).WithSpanString("service", "selected").WithSpanString("environment", "prod"),
		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("service", "selected").WithSpanString("environment", "dev"),
		newMockSpan(nil).WithStartTime(uint64(2*time.Second)).WithSpanString("service", "selected").WithSpanString("environment", "dev"),
		newMockSpan(nil).WithStartTime(uint64(3*time.Second)).WithSpanString("service", "selected").WithSpanString("environment", "prod"),
	}

	for _, span := range spans {
		a.observe(span)
	}

	// Double all counts
	ss := a.result(2.0)

	expected := []TimeSeries{
		// Baseline values
		{
			Labels: Labels{
				internalLabelTypeBaseline,
				Label{Name: "span.service", Value: NewStaticString("baseline1")},
			},
			Values: []float64{4, 0},
		},
		{
			Labels: Labels{
				internalLabelTypeBaseline,
				Label{Name: "span.service", Value: NewStaticString("baseline2")},
			},
			Values: []float64{2, 0},
		},
		{
			Labels: Labels{
				internalLabelTypeBaseline,
				Label{Name: "span.environment", Value: NewStaticString("prod")},
			},
			Values: []float64{4, 0},
		},
		{
			Labels: Labels{
				internalLabelTypeBaseline,
				Label{Name: "span.environment", Value: NewStaticString("dev")},
			},
			Values: []float64{2, 0},
		},
		{
			Labels: Labels{
				internalLabelTypeBaselineTotal,
				Label{Name: "span.service"},
			},
			Values: []float64{6, 0},
		},
		{
			Labels: Labels{
				internalLabelTypeBaselineTotal,
				Label{Name: "span.environment"},
			},
			Values: []float64{6, 0},
		},

		// Selection values
		{
			Labels: Labels{
				internalLabelTypeSelection,
				Label{Name: "span.service", Value: NewStaticString("selected")},
			},
			Values: []float64{6, 2},
		},
		{
			Labels: Labels{
				internalLabelTypeSelection,
				Label{Name: "span.environment", Value: NewStaticString("prod")},
			},
			Values: []float64{2, 2},
		},
		{
			Labels: Labels{
				internalLabelTypeSelection,
				Label{Name: "span.environment", Value: NewStaticString("dev")},
			},
			Values: []float64{4, 0},
		},
		{
			Labels: Labels{
				internalLabelTypeSelectionTotal,
				Label{Name: "span.service"},
			},
			Values: []float64{6, 2},
		},
		{
			Labels: Labels{
				internalLabelTypeSelectionTotal,
				Label{Name: "span.environment"},
			},
			Values: []float64{6, 2},
		},
	}

	requireEqualSeriesSets(t, expected, ss)
}

func TestTopNNaNHandling(t *testing.T) {
	top := topN[string]{}

	// Add entries with some NaN values in the slices
	top.add("series1", []float64{10.0, 20.0, math.NaN(), 30.0})
	top.add("series2", []float64{math.NaN(), math.NaN(), math.NaN()}) // All NaN = sum 0
	top.add("series3", []float64{5.0, 5.0, 5.0})                      // sum = 15

	// Get top 2 - should be series1 (sum=60) and series3 (sum=15)
	var results []string
	top.get(2, func(key string) {
		results = append(results, key)
	})

	require.Len(t, results, 2)
	// series1 has highest sum (60), series3 has second (15), series2 has 0
	assert.Contains(t, results, "series1")
	assert.Contains(t, results, "series3")
}
