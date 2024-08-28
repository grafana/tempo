package app

import (
	"fmt"
	"testing"
	"time"

	"github.com/grafana/dskit/ring"
	"github.com/stretchr/testify/assert"

	"github.com/grafana/tempo/modules/distributor"
	"github.com/grafana/tempo/modules/distributor/forwarder"
	"github.com/grafana/tempo/modules/generator"
	"github.com/grafana/tempo/modules/ingester"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/modules/overrides/userconfigurable/client"
	filterconfig "github.com/grafana/tempo/pkg/spanfilter/config"
)

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
			expErr: fmt.Sprintf("metrics_generator.processor \"span-span\" is not a known processor, valid values: %v", generator.SupportedProcessors),
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
