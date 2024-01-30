package client

import (
	"reflect"
	"testing"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"
)

func TestLimits_parseJson(t *testing.T) {
	testCases := []struct {
		name     string
		json     string
		expected Limits
	}{
		{
			"empty json",
			`{}`,
			Limits{},
		},
		{
			"populated json",
			`{
  "forwarders": ["dev"],
  "metrics_generator": {
    "processors": ["service-graphs"],
    "collection_interval": "30s",
    "processor": {
      "service_graphs": {
        "dimensions": ["cluster"]
      },
      "span_metrics": {
        "dimensions": ["cluster"],
        "histogram_buckets": [0.1, 0.2, 0.5]
      }
    }
  }
}`,
			Limits{
				Forwarders: &[]string{"dev"},
				MetricsGenerator: LimitsMetricsGenerator{
					Processors:         map[string]struct{}{"service-graphs": {}},
					CollectionInterval: &Duration{Duration: 30 * time.Second},
					Processor: LimitsMetricsGeneratorProcessor{
						ServiceGraphs: LimitsMetricsGeneratorProcessorServiceGraphs{
							Dimensions: &[]string{"cluster"},
						},
						SpanMetrics: LimitsMetricsGeneratorProcessorSpanMetrics{
							Dimensions:       &[]string{"cluster"},
							HistogramBuckets: &[]float64{0.1, 0.2, 0.5},
						},
					},
				},
			},
		},
		{
			"empty struct field",
			`{"metrics_generator": {}}`,
			Limits{},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var limits Limits
			err := jsoniter.Unmarshal([]byte(tc.json), &limits)
			assert.NoError(t, err)

			assert.Equal(t, tc.expected, limits)
			assert.True(t, reflect.DeepEqual(tc.expected, limits))
		})
	}
}
