package traceql

import (
	"testing"
	"time"

	"github.com/grafana/tempo/pkg/tempopb"
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

// TestMetricsCompareObserveExemplarLimit verifies that MetricsCompare.observeExemplar
// caps baseline and selection exemplars at m.maxExemplars (set from req.Exemplars).
func TestMetricsCompareObserveExemplarLimit(t *testing.T) {
	const limit = 5
	req := &tempopb.QueryRangeRequest{
		Start:     1,
		End:       uint64(100 * time.Second),
		Step:      uint64(10 * time.Second),
		Query:     `{ } | compare({ .service="selected" })`,
		Exemplars: limit,
	}

	a := newMetricsCompare(newSpansetFilter(
		newBinaryOperation(OpEqual,
			NewAttribute("service"),
			NewStaticString("selected"),
		)), 10, 0, 0)
	a.init(req, AggregateModeRaw)

	// Alternate baseline and selection spans so both slices fill up.
	for i := 0; i < limit*3; i++ {
		ts := uint64(i+1) * uint64(time.Second)
		a.observeExemplar(newMockSpan(nil).WithStartTime(ts).WithSpanString("service", "baseline"))
		a.observeExemplar(newMockSpan(nil).WithStartTime(ts).WithSpanString("service", "selected"))
	}

	require.LessOrEqual(t, len(a.baselineExemplars), limit, "baseline exemplars must not exceed req.Exemplars")
	require.LessOrEqual(t, len(a.selectionExemplars), limit, "selection exemplars must not exceed req.Exemplars")
	require.Greater(t, len(a.baselineExemplars)+len(a.selectionExemplars), 0, "some exemplars must be collected")
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
