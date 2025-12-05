package api

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"

	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/modules/overrides/histograms"
	"github.com/grafana/tempo/pkg/sharedconfig"
	filterconfig "github.com/grafana/tempo/pkg/spanfilter/config"
)

func Test_limitsFromOverrides(t *testing.T) {
	userID := "foo"

	cfg := overrides.Config{
		Defaults: overrides.Overrides{
			Forwarders: []string{"my-forwarder"},
			MetricsGenerator: overrides.MetricsGeneratorOverrides{
				Processors:                      map[string]struct{}{"service-graphs": {}},
				CollectionInterval:              15 * time.Second,
				IngestionSlack:                  time.Minute,
				DisableCollection:               true,
				TraceIDLabelName:                "trace_id",
				GenerateNativeHistograms:        histograms.HistogramMethodBoth,
				NativeHistogramMaxBucketNumber:  160,
				NativeHistogramBucketFactor:     1.2,
				NativeHistogramMinResetDuration: 10 * time.Minute,
				Processor: overrides.ProcessorOverrides{
					ServiceGraphs: overrides.ServiceGraphsOverrides{
						HistogramBuckets:         []float64{0.1, 0.2, 0.5},
						Dimensions:               []string{"my-dim1", "my-dim2"},
						PeerAttributes:           []string{"db.name"},
						EnableClientServerPrefix: boolPtr(true),
					},
					SpanMetrics: overrides.SpanMetricsOverrides{
						Dimensions:          []string{"your-dim1", "your-dim2"},
						IntrinsicDimensions: map[string]bool{"service": true, "span_name": false},
						DimensionMappings:   []sharedconfig.DimensionMappings{{Name: "env", SourceLabel: []string{"k8s.namespace", "foo"}, Join: "/"}},
						EnableTargetInfo:    boolPtr(true),
						EnableInstanceLabel: boolPtr(true),
						FilterPolicies: []filterconfig.FilterPolicy{
							{
								Exclude: &filterconfig.PolicyMatch{
									MatchType: filterconfig.Regex,
									Attributes: []filterconfig.MatchPolicyAttribute{
										{
											Key:   "resource.service.name",
											Value: "unknown_service:myservice",
										},
									},
								},
							},
						},
						HistogramBuckets:             []float64{1, 2, 5},
						TargetInfoExcludedDimensions: []string{"no"},
					},
					HostInfo: overrides.HostInfoOverrides{
						HostIdentifiers: []string{"k8s.node.name", "host.id"},
						MetricName:      "traces_host_info",
					},
				},
			},
		},
	}
	overridesInt, err := overrides.NewOverrides(cfg, nil, prometheus.DefaultRegisterer)
	assert.NoError(t, err)

	limits := limitsFromOverrides(overridesInt, userID)
	limitsJSON, err := json.MarshalIndent(limits, "", "  ")
	assert.NoError(t, err)

	expectedJSON := `{
  "forwarders": [
    "my-forwarder"
  ],
  "cost_attribution": {},
  "metrics_generator": {
    "processors": [
      "service-graphs"
    ],
    "disable_collection": true,
    "collection_interval": "15s",
    "trace_id_label_name": "trace_id",
    "ingestion_time_range_slack": "1m0s",
    "generate_native_histograms": "both",
    "native_histogram_max_bucket_number": 160,
    "native_histogram_bucket_factor": 1.2,
    "native_histogram_min_reset_duration": "10m0s",
    "processor": {
      "service_graphs": {
        "dimensions": [
          "my-dim1",
          "my-dim2"
        ],
        "enable_client_server_prefix": true,
        "peer_attributes": [
          "db.name"
        ],
        "histogram_buckets": [
          0.1,
          0.2,
          0.5
        ]
      },
      "span_metrics": {
        "dimensions": [
          "your-dim1",
          "your-dim2"
        ],
        "intrinsic_dimensions": {
          "service": true,
          "span_name": false
        },
        "dimension_mappings": [
          {
            "name": "env",
            "source_labels": [
              "k8s.namespace",
              "foo"
            ],
            "join": "/"
          }
        ],
        "enable_target_info": true,
        "filter_policies": [
          {
            "exclude": {
              "match_type": "regex",
              "attributes": [
                {
                  "key": "resource.service.name",
                  "value": "unknown_service:myservice"
                }
              ]
            }
          }
        ],
        "histogram_buckets": [
          1,
          2,
          5
        ],
        "target_info_excluded_dimensions": [
          "no"
        ],
        "enable_instance_label": true
      },
      "host_info": {
        "host_identifiers": [
          "k8s.node.name",
          "host.id"
        ],
        "metric_name": "traces_host_info"
      }
    }
  }
}`
	assert.Equal(t, expectedJSON, string(limitsJSON))
}
