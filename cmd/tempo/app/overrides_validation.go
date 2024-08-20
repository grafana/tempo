package app

import (
	"fmt"
	"time"

	"golang.org/x/exp/slices"

	"github.com/grafana/tempo/v2/modules/generator"
	"github.com/grafana/tempo/v2/modules/generator/registry"
	"github.com/grafana/tempo/v2/modules/overrides"
	"github.com/grafana/tempo/v2/modules/overrides/userconfigurable/api"
	"github.com/grafana/tempo/v2/modules/overrides/userconfigurable/client"
	filterconfig "github.com/grafana/tempo/v2/pkg/spanfilter/config"
)

type runtimeConfigValidator struct {
	cfg *Config
}

var _ overrides.Validator = (*runtimeConfigValidator)(nil)

func newRuntimeConfigValidator(cfg *Config) overrides.Validator {
	return &runtimeConfigValidator{
		cfg: cfg,
	}
}

func (r *runtimeConfigValidator) Validate(config *overrides.Overrides) error {
	if config.Ingestion.TenantShardSize != 0 {
		ingesterReplicationFactor := r.cfg.Ingester.LifecyclerConfig.RingConfig.ReplicationFactor
		if config.Ingestion.TenantShardSize < ingesterReplicationFactor {
			return fmt.Errorf("ingester.tenant.shard_size is lower than replication factor (%d < %d)", config.Ingestion.TenantShardSize, ingesterReplicationFactor)
		}
	}

	if _, ok := registry.HistogramModeToValue[string(config.MetricsGenerator.GenerateNativeHistograms)]; !ok {
		if config.MetricsGenerator.GenerateNativeHistograms != "" {
			return fmt.Errorf("metrics_generator.generate_native_histograms \"%s\" is not a valid value, valid values: classic, native, both", config.MetricsGenerator.GenerateNativeHistograms)
		}
	}

	return nil
}

type overridesValidator struct {
	cfg *Config

	validForwarders map[string]struct{}
}

var _ api.Validator = (*overridesValidator)(nil)

func newOverridesValidator(cfg *Config) api.Validator {
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
