package generator

import (
	"flag"
	"time"

	"github.com/pkg/errors"

	"github.com/grafana/tempo/modules/generator/processor/servicegraphs"
	"github.com/grafana/tempo/modules/generator/processor/spanmetrics"
	"github.com/grafana/tempo/modules/generator/registry"
	"github.com/grafana/tempo/modules/generator/storage"
)

const (
	// RingKey is the key under which we store the metric-generator's ring in the KVStore.
	RingKey = "metrics-generator"

	// ringNameForServer is the name of the ring used by the metrics-generator server.
	ringNameForServer = "metrics-generator"
)

// Config for a generator.
type Config struct {
	Ring      RingConfig      `yaml:"ring"`
	Processor ProcessorConfig `yaml:"processor"`
	Registry  registry.Config `yaml:"registry"`
	Storage   storage.Config  `yaml:"storage"`
	// MetricsIngestionSlack is the max amount of time passed since a span's start time
	// for the span to be considered in metrics generation
	MetricsIngestionSlack time.Duration `yaml:"metrics_ingestion_time_range_slack"`
}

// RegisterFlagsAndApplyDefaults registers the flags.
func (cfg *Config) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	cfg.Ring.RegisterFlagsAndApplyDefaults(prefix, f)
	cfg.Processor.RegisterFlagsAndApplyDefaults(prefix, f)
	cfg.Registry.RegisterFlagsAndApplyDefaults(prefix, f)
	cfg.Storage.RegisterFlagsAndApplyDefaults(prefix, f)
	// setting default for max span age before discarding to 30s
	cfg.MetricsIngestionSlack = 30 * time.Second
}

type ProcessorConfig struct {
	ServiceGraphs servicegraphs.Config `yaml:"service_graphs"`
	SpanMetrics   spanmetrics.Config   `yaml:"span_metrics"`
}

func (cfg *ProcessorConfig) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	cfg.ServiceGraphs.RegisterFlagsAndApplyDefaults(prefix, f)
	cfg.SpanMetrics.RegisterFlagsAndApplyDefaults(prefix, f)
}

// copyWithOverrides creates a copy of the config using values set in the overrides.
func (cfg *ProcessorConfig) copyWithOverrides(o metricsGeneratorOverrides, userID string) (ProcessorConfig, error) {
	copyCfg := *cfg

	if buckets := o.MetricsGeneratorProcessorServiceGraphsHistogramBuckets(userID); buckets != nil {
		copyCfg.ServiceGraphs.HistogramBuckets = buckets
	}
	if dimensions := o.MetricsGeneratorProcessorServiceGraphsDimensions(userID); dimensions != nil {
		copyCfg.ServiceGraphs.Dimensions = dimensions
	}
	if buckets := o.MetricsGeneratorProcessorSpanMetricsHistogramBuckets(userID); buckets != nil {
		copyCfg.SpanMetrics.HistogramBuckets = buckets
	}
	if dimensions := o.MetricsGeneratorProcessorSpanMetricsDimensions(userID); dimensions != nil {
		copyCfg.SpanMetrics.Dimensions = dimensions
	}
	if dimensions := o.MetricsGeneratorProcessorSpanMetricsIntrinsicDimensions(userID); dimensions != nil {
		err := copyCfg.SpanMetrics.IntrinsicDimensions.ApplyFromMap(dimensions)
		if err != nil {
			return ProcessorConfig{}, errors.Wrap(err, "fail to apply overrides")
		}
	}

	return copyCfg, nil
}
