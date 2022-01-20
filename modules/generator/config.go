package generator

import (
	"flag"

	"github.com/prometheus/prometheus/config"

	"github.com/grafana/tempo/modules/generator/processor/servicegraphs"
	"github.com/grafana/tempo/modules/generator/processor/spanmetrics"
)

const (
	// RingKey is the key under which we store the metric-generator's ring in the KVStore.
	RingKey = "metrics-generator"

	// ringNameForServer is the name of the ring used by the metrics-generator server.
	ringNameForServer = "metrics-generator"
)

// Config for a generator.
type Config struct {
	Ring RingConfig `yaml:"ring"`

	Processor ProcessorConfig `yaml:"processor"`

	RemoteWrite RemoteWriteConfig `yaml:"remote_write,omitempty"`
}

// RegisterFlagsAndApplyDefaults registers the flags.
func (cfg *Config) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	cfg.Ring.RegisterFlagsAndApplyDefaults(prefix, f)
	cfg.Processor.RegisterFlagsAndApplyDefaults(prefix, f)
	cfg.RemoteWrite.RegisterFlagsAndApplyDefaults(prefix, f)
}

type ProcessorConfig struct {
	ServiceGraphs servicegraphs.Config `yaml:"service_graphs"`
	SpanMetrics   spanmetrics.Config   `yaml:"span_metrics"`
}

func (cfg *ProcessorConfig) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	cfg.ServiceGraphs.RegisterFlagsAndApplyDefaults(prefix, f)
	cfg.SpanMetrics.RegisterFlagsAndApplyDefaults(prefix, f)
}

type RemoteWriteConfig struct {
	Client  config.RemoteWriteConfig `yaml:"client"`
	Enabled bool                     `yaml:"enabled"`
}

func (c *RemoteWriteConfig) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
}
