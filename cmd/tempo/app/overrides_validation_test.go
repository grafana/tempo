package app

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/grafana/tempo/modules/distributor"
	"github.com/grafana/tempo/modules/distributor/forwarder"
	"github.com/grafana/tempo/modules/generator"
	"github.com/grafana/tempo/modules/overrides/userconfigurable/client"
	filterconfig "github.com/grafana/tempo/pkg/spanfilter/config"
)

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
				MetricsGenerator: &client.LimitsMetricsGenerator{
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
				MetricsGenerator: &client.LimitsMetricsGenerator{Processor: &client.LimitsMetricsGeneratorProcessor{SpanMetrics: &client.LimitsMetricsGeneratorProcessorSpanMetrics{FilterPolicies: &[]filterconfig.FilterPolicy{{
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
				MetricsGenerator: &client.LimitsMetricsGenerator{Processor: &client.LimitsMetricsGeneratorProcessor{SpanMetrics: &client.LimitsMetricsGeneratorProcessorSpanMetrics{FilterPolicies: &[]filterconfig.FilterPolicy{
					{
						Include: &filterconfig.PolicyMatch{
							MatchType: "invalid",
						},
					},
				}}}},
			},
			expErr: "invalid include policy: invalid match type: invalid",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			validator := NewOverridesValidator(&tc.cfg)

			err := validator.Validate(&tc.limits)
			if tc.expErr != "" {
				assert.EqualError(t, err, tc.expErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
