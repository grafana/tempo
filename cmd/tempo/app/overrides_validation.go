package app

import (
	"fmt"

	"github.com/grafana/tempo/modules/generator/validation"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/modules/overrides/userconfigurable/api"
	"github.com/grafana/tempo/modules/overrides/userconfigurable/client"
	filterconfig "github.com/grafana/tempo/pkg/spanfilter/config"
)

type runtimeConfigValidator struct {
	cfg *Config
}

var _ overrides.Validator = (*runtimeConfigValidator)(nil)

// newRuntimeConfigValidator validates runtime overrides
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

	if config.MetricsGenerator.GenerateNativeHistograms != "" {
		if err := validation.ValidateHistogramMode(string(config.MetricsGenerator.GenerateNativeHistograms)); err != nil {
			return err
		}
	}

	if config.MetricsGenerator.NativeHistogramBucketFactor != 0 {
		if err := validation.ValidateNativeHistogramBucketFactor(config.MetricsGenerator.NativeHistogramBucketFactor); err != nil {
			return err
		}
	}

	if config.CostAttribution.Dimensions != nil {
		if err := validation.ValidateCostAttributionDimensions(config.CostAttribution.Dimensions); err != nil {
			return err
		}
	}

	serviceBuckets := config.MetricsGenerator.Processor.ServiceGraphs.HistogramBuckets
	if err := validation.ValidateHistogramBuckets(serviceBuckets, "metrics_generator.processor.service_graphs.histogram_buckets"); err != nil {
		return err
	}

	spanBuckets := config.MetricsGenerator.Processor.SpanMetrics.HistogramBuckets
	if err := validation.ValidateHistogramBuckets(spanBuckets, "metrics_generator.processor.span_metrics.histogram_buckets"); err != nil {
		return err
	}

	return nil
}

type overridesValidator struct {
	cfg *Config

	validForwarders map[string]struct{}
}

var _ api.Validator = (*overridesValidator)(nil)

// newOverridesValidator validates user-configurable overrides
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
			if err := validation.ValidateProcessor(p); err != nil {
				return err
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
		if err := validation.ValidateCollectionInterval(collectionInterval); err != nil {
			return err
		}
	}

	if traceIDLabelName, ok := limits.GetMetricsGenerator().GetTraceIDLabelName(); ok {
		if err := validation.ValidateTraceIDLabelName(traceIDLabelName); err != nil {
			return err
		}
	}

	if ingestionSlack, ok := limits.GetMetricsGenerator().GetIngestionSlack(); ok {
		if err := validation.ValidateIngestionTimeRangeSlack(ingestionSlack); err != nil {
			return err
		}
	}

	if factor, ok := limits.GetMetricsGenerator().GetNativeHistogramBucketFactor(); ok {
		if err := validation.ValidateNativeHistogramBucketFactor(factor); err != nil {
			return err
		}
	}

	if buckets, ok := limits.GetMetricsGenerator().GetProcessor().GetServiceGraphs().GetHistogramBuckets(); ok {
		if err := validation.ValidateHistogramBuckets(buckets, "metrics_generator.processor.service_graphs.histogram_buckets"); err != nil {
			return err
		}
	}

	if buckets, ok := limits.GetMetricsGenerator().GetProcessor().GetSpanMetrics().GetHistogramBuckets(); ok {
		if err := validation.ValidateHistogramBuckets(buckets, "metrics_generator.processor.span_metrics.histogram_buckets"); err != nil {
			return err
		}
	}

	if dims, ok := limits.GetCostAttribution().GetDimensions(); ok {
		if err := validation.ValidateCostAttributionDimensions(dims); err != nil {
			return err
		}
	}

	if intrinsicDimensions, ok := limits.GetMetricsGenerator().GetProcessor().GetSpanMetrics().GetIntrinsicDimensions(); ok {
		if err := validation.ValidateIntrinsicDimensions(intrinsicDimensions); err != nil {
			return err
		}
	}

	return nil
}
