package generator

import (
	"flag"

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
}

// RegisterFlagsAndApplyDefaults registers the flags.
func (cfg *Config) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	cfg.Ring.RegisterFlagsAndApplyDefaults(prefix, f)
	cfg.Processor.RegisterFlagsAndApplyDefaults(prefix, f)
	cfg.Registry.RegisterFlagsAndApplyDefaults(prefix, f)
	cfg.Storage.RegisterFlagsAndApplyDefaults(prefix, f)
}

type ProcessorConfig struct {
	ServiceGraphs servicegraphs.Config `yaml:"service_graphs"`
	SpanMetrics   spanmetrics.Config   `yaml:"span_metrics"`
}

func (cfg *ProcessorConfig) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	cfg.ServiceGraphs.RegisterFlagsAndApplyDefaults(prefix, f)
	cfg.SpanMetrics.RegisterFlagsAndApplyDefaults(prefix, f)
}
