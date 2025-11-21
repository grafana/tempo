package app

import (
	"fmt"
	"testing"
	"time"

	"github.com/grafana/dskit/ring"
	"github.com/grafana/tempo/modules/distributor"
	"github.com/grafana/tempo/modules/distributor/forwarder"
	"github.com/grafana/tempo/modules/generator/processor"
	"github.com/grafana/tempo/modules/generator/validation"
	"github.com/grafana/tempo/modules/ingester"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/modules/overrides/userconfigurable/client"
	filterconfig "github.com/grafana/tempo/pkg/spanfilter/config"
	"github.com/stretchr/testify/assert"
)

func strPtr(s string) *string {
	return &s
}

func float64Ptr(f float64) *float64 {
	return &f
}

func boolMapPtr(m map[string]bool) *map[string]bool {
	return &m
}

func Test_runtimeOverridesValidator(t *testing.T) {
	testCases := []struct {
		name      string
		cfg       Config
		overrides overrides.Overrides
		expErr    string
	}{
		{
			name: "ingestion.tenant_shard_size smaller than RF",
			cfg: Config{
				Ingester: ingester.Config{
					LifecyclerConfig: ring.LifecyclerConfig{
						RingConfig: ring.Config{
							ReplicationFactor: 3,
						},
					},
				},
			},
			overrides: overrides.Overrides{Ingestion: overrides.IngestionOverrides{TenantShardSize: 2}},
			expErr:    "ingester.tenant.shard_size is lower than replication factor (2 < 3)",
		},
		{
			name: "ingestion.tenant_shard_size equal to RF",
			cfg: Config{
				Ingester: ingester.Config{
					LifecyclerConfig: ring.LifecyclerConfig{
						RingConfig: ring.Config{
							ReplicationFactor: 3,
						},
					},
				},
			},
			overrides: overrides.Overrides{Ingestion: overrides.IngestionOverrides{TenantShardSize: 3}},
		},
		{
			name: "metrics_generator.generate_native_histograms invalid",
			cfg:  Config{},
			overrides: overrides.Overrides{MetricsGenerator: overrides.MetricsGeneratorOverrides{
				GenerateNativeHistograms: "invalid",
			}},
			expErr: "metrics_generator.generate_native_histograms \"invalid\" is not a valid value, valid values: classic, native, both",
		},
		{
			name: "metrics_generator.generate_native_histograms classic",
			cfg:  Config{},
			overrides: overrides.Overrides{MetricsGenerator: overrides.MetricsGeneratorOverrides{
				GenerateNativeHistograms: "classic",
			}},
		},
		{
			name: "metrics_generator.generate_native_histograms native",
			cfg:  Config{},
			overrides: overrides.Overrides{MetricsGenerator: overrides.MetricsGeneratorOverrides{
				GenerateNativeHistograms: "native",
			}},
		},
		{
			name: "metrics_generator.generate_native_histograms both",
			cfg:  Config{},
			overrides: overrides.Overrides{MetricsGenerator: overrides.MetricsGeneratorOverrides{
				GenerateNativeHistograms: "both",
			}},
		},
		{
			name: "service graphs histogram buckets must be increasing",
			cfg:  Config{},
			overrides: overrides.Overrides{
				MetricsGenerator: overrides.MetricsGeneratorOverrides{
					Processor: overrides.ProcessorOverrides{
						ServiceGraphs: overrides.ServiceGraphsOverrides{
							HistogramBuckets: []float64{2, 1},
						},
					},
				},
			},
			expErr: "metrics_generator.processor.service_graphs.histogram_buckets must be strictly increasing: bucket[1]=1 is <= bucket[0]=2",
		},
		{
			name: "span metrics histogram buckets must be increasing",
			cfg:  Config{},
			overrides: overrides.Overrides{
				MetricsGenerator: overrides.MetricsGeneratorOverrides{
					Processor: overrides.ProcessorOverrides{
						SpanMetrics: overrides.SpanMetricsOverrides{
							HistogramBuckets: []float64{0.5, 0.5},
						},
					},
				},
			},
			expErr: "metrics_generator.processor.span_metrics.histogram_buckets must be strictly increasing: bucket[1]=0.5 is <= bucket[0]=0.5",
		},
		{
			name: "service graphs histogram buckets valid",
			cfg:  Config{},
			overrides: overrides.Overrides{
				MetricsGenerator: overrides.MetricsGeneratorOverrides{
					Processor: overrides.ProcessorOverrides{
						ServiceGraphs: overrides.ServiceGraphsOverrides{
							HistogramBuckets: []float64{0.01, 0.1, 1},
						},
					},
				},
			},
		},
		{
			name: "span metrics histogram buckets valid",
			cfg:  Config{},
			overrides: overrides.Overrides{
				MetricsGenerator: overrides.MetricsGeneratorOverrides{
					Processor: overrides.ProcessorOverrides{
						SpanMetrics: overrides.SpanMetricsOverrides{
							HistogramBuckets: []float64{0.005, 0.01, 0.02},
						},
					},
				},
			},
		},
		{
			name: "native histogram bucket factor invalid",
			cfg:  Config{},
			overrides: overrides.Overrides{
				MetricsGenerator: overrides.MetricsGeneratorOverrides{
					NativeHistogramBucketFactor: 1,
				},
			},
			expErr: "metrics_generator.native_histogram_bucket_factor must be greater than 1",
		},
		{
			name: "native histogram bucket factor valid",
			cfg:  Config{},
			overrides: overrides.Overrides{
				MetricsGenerator: overrides.MetricsGeneratorOverrides{
					NativeHistogramBucketFactor: 1.5,
				},
			},
		},
		{
			name: "valid cost attribution dimensions",
			cfg:  Config{},
			overrides: overrides.Overrides{
				CostAttribution: overrides.CostAttributionOverrides{
					Dimensions: map[string]string{"span.name": "op_name"},
				},
			},
		},
		{
			name: "invalid cost attribution dimensions",
			cfg:  Config{},
			overrides: overrides.Overrides{
				CostAttribution: overrides.CostAttributionOverrides{
					Dimensions: map[string]string{"span.name": "__name__"},
				},
			},
			expErr: "cost_attribution.dimensions config has invalid label name: '__name__'",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			validator := newRuntimeConfigValidator(&tc.cfg)

			err := validator.Validate(&tc.overrides)
			if tc.expErr != "" {
				assert.EqualError(t, err, tc.expErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_overridesValidator(t *testing.T) {
	testCases := []struct {
		name   string
		cfg    Config
		limits client.Limits
		expErr string
	}{
		{
			name: "forwarders - valid",
			cfg: Config{
				Distributor: distributor.Config{
					Forwarders: []forwarder.Config{
						{Name: "forwarder-1"},
						{Name: "forwarder-2"},
						{Name: "forwarder-3"},
					},
				},
			},
			limits: client.Limits{
				Forwarders: &[]string{"forwarder-1", "forwarder-3"},
			},
		},
		{
			name: "forwarders - invalid",
			cfg: Config{
				Distributor: distributor.Config{
					Forwarders: []forwarder.Config{
						{Name: "forwarder-1"},
						{Name: "forwarder-2"},
					},
				},
			},
			limits: client.Limits{
				Forwarders: &[]string{"forwarder-1", "some-forwarder"},
			},
			expErr: "forwarder \"some-forwarder\" is not a known forwarder, contact your system administrator",
		},
		{
			name: "metrics_generator.processor",
			cfg:  Config{},
			limits: client.Limits{
				MetricsGenerator: client.LimitsMetricsGenerator{
					Processors: map[string]struct{}{
						"service-graphs": {},
						"span-span":      {},
					},
				},
			},
			expErr: fmt.Sprintf("metrics_generator.processor \"span-span\" is not a known processor, valid values: %v", validation.SupportedProcessors),
		},
		{
			name: "filter policies",
			cfg:  Config{},
			limits: client.Limits{
				Forwarders: &[]string{},
				MetricsGenerator: client.LimitsMetricsGenerator{Processor: client.LimitsMetricsGeneratorProcessor{SpanMetrics: client.LimitsMetricsGeneratorProcessorSpanMetrics{FilterPolicies: &[]filterconfig.FilterPolicy{{
					Include: &filterconfig.PolicyMatch{
						MatchType: filterconfig.Strict,
						Attributes: []filterconfig.MatchPolicyAttribute{
							{
								Key:   "span.kind",
								Value: "SPAN_KIND_SERVER",
							},
						},
					},
				}}}}},
			},
		},
		{
			name: "filter policies - invalid",
			cfg:  Config{},
			limits: client.Limits{
				Forwarders: &[]string{},
				MetricsGenerator: client.LimitsMetricsGenerator{Processor: client.LimitsMetricsGeneratorProcessor{SpanMetrics: client.LimitsMetricsGeneratorProcessorSpanMetrics{FilterPolicies: &[]filterconfig.FilterPolicy{
					{
						Include: &filterconfig.PolicyMatch{
							MatchType: "invalid",
						},
					},
				}}}},
			},
			expErr: "invalid include policy: invalid match type: invalid",
		},
		{
			name: "metrics_generator.collection_interval valid",
			cfg:  Config{},
			limits: client.Limits{
				MetricsGenerator: client.LimitsMetricsGenerator{
					CollectionInterval: &client.Duration{Duration: 60 * time.Second},
				},
			},
		},
		{
			name: "metrics_generator.collection_interval minimum",
			cfg:  Config{},
			limits: client.Limits{
				MetricsGenerator: client.LimitsMetricsGenerator{
					CollectionInterval: &client.Duration{Duration: 1 * time.Second},
				},
			},
			expErr: "metrics_generator.collection_interval \"1s\" is outside acceptable range of 15s to 5m",
		},
		{
			name: "metrics_generator.collection_interval maximum",
			cfg:  Config{},
			limits: client.Limits{
				MetricsGenerator: client.LimitsMetricsGenerator{
					CollectionInterval: &client.Duration{Duration: 10 * time.Minute},
				},
			},
			expErr: "metrics_generator.collection_interval \"10m0s\" is outside acceptable range of 15s to 5m",
		},
		{
			name: "metrics_generator.trace_id_label_name empty is allowed",
			cfg:  Config{},
			limits: client.Limits{
				MetricsGenerator: client.LimitsMetricsGenerator{
					TraceIDLabelName: strPtr(""),
				},
			},
		},
		{
			name: "metrics_generator.trace_id_label_name invalid",
			cfg:  Config{},
			limits: client.Limits{
				MetricsGenerator: client.LimitsMetricsGenerator{
					TraceIDLabelName: strPtr("trace-id"),
				},
			},
			expErr: "trace_id_label_name \"trace-id\" is not a valid Prometheus label name",
		},
		{
			name: "metrics_generator.ingestion_time_range_slack valid",
			cfg:  Config{},
			limits: client.Limits{
				MetricsGenerator: client.LimitsMetricsGenerator{
					IngestionSlack: &client.Duration{Duration: 2 * time.Minute},
				},
			},
		},
		{
			name: "metrics_generator.ingestion_time_range_slack invalid",
			cfg:  Config{},
			limits: client.Limits{
				MetricsGenerator: client.LimitsMetricsGenerator{
					IngestionSlack: &client.Duration{Duration: 15 * time.Hour},
				},
			},
			expErr: "metrics_generator.ingestion_time_range_slack \"15h0m0s\" is outside acceptable range of 0s to 12h",
		},
		{
			name: "service graphs histogram buckets must be increasing",
			cfg:  Config{},
			limits: client.Limits{
				MetricsGenerator: client.LimitsMetricsGenerator{
					Processor: client.LimitsMetricsGeneratorProcessor{
						ServiceGraphs: client.LimitsMetricsGeneratorProcessorServiceGraphs{
							HistogramBuckets: &[]float64{5, 4},
						},
					},
				},
			},
			expErr: "metrics_generator.processor.service_graphs.histogram_buckets must be strictly increasing: bucket[1]=4 is <= bucket[0]=5",
		},
		{
			name: "span metrics histogram buckets must be increasing",
			cfg:  Config{},
			limits: client.Limits{
				MetricsGenerator: client.LimitsMetricsGenerator{
					Processor: client.LimitsMetricsGeneratorProcessor{
						SpanMetrics: client.LimitsMetricsGeneratorProcessorSpanMetrics{
							HistogramBuckets: &[]float64{1, 1},
						},
					},
				},
			},
			expErr: "metrics_generator.processor.span_metrics.histogram_buckets must be strictly increasing: bucket[1]=1 is <= bucket[0]=1",
		},
		{
			name: "service graphs histogram buckets valid",
			cfg:  Config{},
			limits: client.Limits{
				MetricsGenerator: client.LimitsMetricsGenerator{
					Processor: client.LimitsMetricsGeneratorProcessor{
						ServiceGraphs: client.LimitsMetricsGeneratorProcessorServiceGraphs{
							HistogramBuckets: &[]float64{0.01, 0.1, 1},
						},
					},
				},
			},
		},
		{
			name: "span metrics histogram buckets valid",
			cfg:  Config{},
			limits: client.Limits{
				MetricsGenerator: client.LimitsMetricsGenerator{
					Processor: client.LimitsMetricsGeneratorProcessor{
						SpanMetrics: client.LimitsMetricsGeneratorProcessorSpanMetrics{
							HistogramBuckets: &[]float64{0.005, 0.01, 0.02},
						},
					},
				},
			},
		},
		{
			name: "native histogram bucket factor invalid",
			cfg:  Config{},
			limits: client.Limits{
				MetricsGenerator: client.LimitsMetricsGenerator{
					NativeHistogramBucketFactor: float64Ptr(1),
				},
			},
			expErr: "metrics_generator.native_histogram_bucket_factor must be greater than 1",
		},
		{
			name: "native histogram bucket factor valid",
			cfg:  Config{},
			limits: client.Limits{
				MetricsGenerator: client.LimitsMetricsGenerator{
					NativeHistogramBucketFactor: float64Ptr(1.5),
				},
			},
		},
		{
			name: "valid cost attribution dimensions",
			cfg:  Config{},
			limits: client.Limits{
				CostAttribution: client.CostAttribution{
					Dimensions: &map[string]string{"span.name": "op_name"},
				},
			},
		},
		{
			name: "invalid cost attribution dimensions",
			cfg:  Config{},
			limits: client.Limits{
				CostAttribution: client.CostAttribution{
					Dimensions: &map[string]string{"span.name": "__name__"},
				},
			},
			expErr: "cost_attribution.dimensions config has invalid label name: '__name__'",
		},
		{
			name: "metrics_generator.span_metrics.intrinsic_dimensions valid",
			cfg:  Config{},
			limits: client.Limits{
				MetricsGenerator: client.LimitsMetricsGenerator{
					Processor: client.LimitsMetricsGeneratorProcessor{
						SpanMetrics: client.LimitsMetricsGeneratorProcessorSpanMetrics{
							IntrinsicDimensions: boolMapPtr(map[string]bool{
								processor.DimService:  true,
								processor.DimSpanKind: false,
							}),
						},
					},
				},
			},
		},
		{
			name: "metrics_generator.span_metrics.intrinsic_dimensions invalid",
			cfg:  Config{},
			limits: client.Limits{
				MetricsGenerator: client.LimitsMetricsGenerator{
					Processor: client.LimitsMetricsGeneratorProcessor{
						SpanMetrics: client.LimitsMetricsGeneratorProcessorSpanMetrics{
							IntrinsicDimensions: boolMapPtr(map[string]bool{
								"not_supported": true,
							}),
						},
					},
				},
			},
			expErr: fmt.Sprintf("intrinsic dimension \"%s\" is not supported, valid values: %v", "not_supported", validation.SupportedIntrinsicDimensions),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			validator := newOverridesValidator(&tc.cfg)

			err := validator.Validate(&tc.limits)
			if tc.expErr != "" {
				assert.EqualError(t, err, tc.expErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
