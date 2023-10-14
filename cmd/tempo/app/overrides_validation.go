package app

import (
	"fmt"
	"slices"
	"time"

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

	if collectionInterval, ok := limits.GetMetricsGenerator().GetCollectionInterval(); ok {
		if collectionInterval < 15*time.Second || collectionInterval > 5*time.Minute {
			return fmt.Errorf("metrics_generator.collection_interval \"%s\" is outside acceptable range of 15s to 5m", collectionInterval.String())
		}
	}

	return nil
}
