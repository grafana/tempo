package api

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"

	"github.com/grafana/tempo/modules/overrides"
	filterconfig "github.com/grafana/tempo/pkg/spanfilter/config"
)

func Test_limitsFromOverrides(t *testing.T) {
	userID := "foo"

	cfg := overrides.Config{
		Defaults: overrides.Overrides{
			Forwarders: []string{"my-forwarder"},
			MetricsGenerator: overrides.MetricsGeneratorOverrides{
				Processors:         map[string]struct{}{"service-graphs": {}},
				CollectionInterval: 15 * time.Second,
				DisableCollection:  true,
				Processor: overrides.ProcessorOverrides{
					ServiceGraphs: overrides.ServiceGraphsOverrides{
						HistogramBuckets:         []float64{0.1, 0.2, 0.5},
						Dimensions:               []string{"my-dim1", "my-dim2"},
						PeerAttributes:           []string{"db.name"},
						EnableClientServerPrefix: boolPtr(true),
					},
					SpanMetrics: overrides.SpanMetricsOverrides{
						Dimensions:       []string{"your-dim1", "your-dim2"},
						EnableTargetInfo: boolPtr(true),
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
        ]
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
