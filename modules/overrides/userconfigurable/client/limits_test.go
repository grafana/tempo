package client

import (
	"reflect"
	"testing"
	"time"

	"github.com/grafana/tempo/modules/overrides/histograms"
	"github.com/grafana/tempo/pkg/sharedconfig"
	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"
)

func strPtr(s string) *string {
	return &s
}

func mapBoolPtr(m map[string]bool) *map[string]bool {
	return &m
}

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
    "trace_id_label_name": "my_trace_id",
    "ingestion_time_range_slack": "45s",
    "native_histogram_bucket_factor": 1.2,
    "native_histogram_min_reset_duration": "10m",
    "native_histogram_max_bucket_number": 101,
    "generate_native_histograms": "native",
    "processor": {
      "service_graphs": {
        "dimensions": ["cluster"]
      },
      "span_metrics": {
        "dimensions": ["cluster"],
        "intrinsic_dimensions": {
          "service": true
        },
        "dimension_mappings": [
          {
            "name": "foo",
            "source_labels": [
              "bar",
			  "bar2"
            ],
            "join": "/"
          },
		  {
            "name": "abc",
            "source_labels": [
              "def"
            ],
            "join": ""
          }
        ],
        "histogram_buckets": [0.1, 0.2, 0.5]
      },
      "host_info": {
        "host_identifiers": ["k8s.node.name", "host.id"],
        "metric_name": "traces_host_info"
      }
    }
  }
}`,
			Limits{
				Forwarders: &[]string{"dev"},
				MetricsGenerator: LimitsMetricsGenerator{
					Processors:                      map[string]struct{}{"service-graphs": {}},
					CollectionInterval:              &Duration{Duration: 30 * time.Second},
					TraceIDLabelName:                strPtr("my_trace_id"),
					IngestionSlack:                  &Duration{Duration: 45 * time.Second},
					NativeHistogramBucketFactor:     func(f float64) *float64 { return &f }(1.2),
					NativeHistogramMinResetDuration: &Duration{Duration: 10 * time.Minute},
					NativeHistogramMaxBucketNumber:  func(u uint32) *uint32 { return &u }(101),
					GenerateNativeHistograms:        (*histograms.HistogramMethod)(strPtr("native")),
					Processor: LimitsMetricsGeneratorProcessor{
						ServiceGraphs: LimitsMetricsGeneratorProcessorServiceGraphs{
							Dimensions: &[]string{"cluster"},
						},
						SpanMetrics: LimitsMetricsGeneratorProcessorSpanMetrics{
							Dimensions:          &[]string{"cluster"},
							HistogramBuckets:    &[]float64{0.1, 0.2, 0.5},
							IntrinsicDimensions: mapBoolPtr(map[string]bool{"service": true}),
							DimensionMappings: &[]sharedconfig.DimensionMappings{
								{
									Name:        "foo",
									SourceLabel: []string{"bar", "bar2"},
									Join:        "/",
								},
								{
									Name:        "abc",
									SourceLabel: []string{"def"},
									Join:        "",
								},
							},
						},
						HostInfo: LimitsMetricGeneratorProcessorHostInfo{
							HostIdentifiers: &[]string{"k8s.node.name", "host.id"},
							MetricName:      strPtr("traces_host_info"),
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
