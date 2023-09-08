package app

import (
	"fmt"

	"golang.org/x/exp/slices"

	"github.com/grafana/tempo/modules/generator"
	"github.com/grafana/tempo/modules/overrides/userconfigurable/api"
	"github.com/grafana/tempo/modules/overrides/userconfigurable/client"
	filterconfig "github.com/grafana/tempo/pkg/spanfilter/config"
)

type overridesValidator struct {
	cfg *Config

	validForwarders map[string]struct{}
}

var _ api.Validator = (*overridesValidator)(nil)

func NewOverridesValidator(cfg *Config) api.Validator {
	validForwarders := map[string]struct{}{}
	for _, f := range cfg.Distributor.Forwarders {
		validForwarders[f.Name] = struct{}{}
	}

	return &overridesValidator{
		cfg: cfg,

		validForwarders: validForwarders,
	}
}

func (v *overridesValidator) Validate(limits *client.Limits) error {
	if forwarders, ok := limits.GetForwarders(); ok {
		for _, f := range forwarders {
			if _, ok := v.validForwarders[f]; !ok {
				return fmt.Errorf("forwarder \"%s\" is not a known forwarder, contact your system administrator", f)
			}
		}
	}

	if processors, ok := limits.GetMetricsGenerator().GetProcessors(); ok {
		for p := range processors.GetMap() {
			if !slices.Contains(generator.SupportedProcessors, p) {
				return fmt.Errorf("metrics_generator.processor \"%s\" is not a known processor, valid values: %v", p, generator.SupportedProcessors)
			}
		}
	}

	if filterPolicies, ok := limits.GetMetricsGenerator().GetProcessor().GetSpanMetrics().GetFilterPolicies(); ok {
		for _, fp := range filterPolicies {
			if err := filterconfig.ValidateFilterPolicy(fp); err != nil {
				return err
			}
		}
	}

	return nil
}
