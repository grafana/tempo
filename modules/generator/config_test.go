package generator

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/grafana/tempo/modules/generator/processor/servicegraphs"
	"github.com/grafana/tempo/modules/generator/processor/spanmetrics"
)

func TestProcessorConfig_copyWithOverrides(t *testing.T) {
	original := &ProcessorConfig{
		ServiceGraphs: servicegraphs.Config{
			HistogramBuckets: []float64{1},
			Dimensions:       []string{},
		},
		SpanMetrics: spanmetrics.Config{
			HistogramBuckets: []float64{1, 2},
			Dimensions:       []string{"namespace"},
		},
	}

	t.Run("overrides buckets and dimension", func(t *testing.T) {
		o := &mockOverrides{
			serviceGraphsHistogramBuckets: []float64{1, 2},
			serviceGraphsDimensions:       []string{"namespace"},
			spanMetricsHistogramBuckets:   []float64{1, 2, 3},
			spanMetricsDimensions:         []string{"cluster", "namespace"},
		}

		copied := original.copyWithOverrides(o, "tenant")

		assert.NotEqual(t, *original, copied)

		// assert nothing changed
		assert.Equal(t, []float64{1}, original.ServiceGraphs.HistogramBuckets)
		assert.Equal(t, []string{}, original.ServiceGraphs.Dimensions)
		assert.Equal(t, []float64{1, 2}, original.SpanMetrics.HistogramBuckets)
		assert.Equal(t, []string{"namespace"}, original.SpanMetrics.Dimensions)

		// assert overrides were applied
		assert.Equal(t, []float64{1, 2}, copied.ServiceGraphs.HistogramBuckets)
		assert.Equal(t, []string{"namespace"}, copied.ServiceGraphs.Dimensions)
		assert.Equal(t, []float64{1, 2, 3}, copied.SpanMetrics.HistogramBuckets)
		assert.Equal(t, []string{"cluster", "namespace"}, copied.SpanMetrics.Dimensions)
	})

	t.Run("empty overrides", func(t *testing.T) {
		o := &mockOverrides{}

		copied := original.copyWithOverrides(o, "tenant")

		assert.Equal(t, *original, copied)
	})
}
