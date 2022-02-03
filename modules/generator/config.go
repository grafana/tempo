package generator

import (
	"flag"
	"time"

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

	// CollectionInterval controls how often to collect and remote write metrics.
	// Defaults to 15s.
	CollectionInterval time.Duration `yaml:"collection_interval"`

	// ExternalLabels are added to any time-series exported by this instance.
	ExternalLabels map[string]string `yaml:"external_labels,omitempty"`

	// Add a label `tempo_instance_id` to every metric. This is necessary when running multiple
	// instances of the metrics-generator as each instance will push the same time series.
	AddInstanceIDLabel bool `yaml:"add_instance_id_label"`

	RemoteWrite RemoteWriteConfig `yaml:"remote_write,omitempty"`
}

// RegisterFlagsAndApplyDefaults registers the flags.
func (cfg *Config) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	cfg.Ring.RegisterFlagsAndApplyDefaults(prefix, f)
	cfg.Processor.RegisterFlagsAndApplyDefaults(prefix, f)

	cfg.CollectionInterval = 15 * time.Second
	cfg.AddInstanceIDLabel = true

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
	// Enable remote0write requests. If disabled all generated metrics will be discarded.
	Enabled bool `yaml:"enabled"`

	Client config.RemoteWriteConfig `yaml:"client"`
}

func (cfg *RemoteWriteConfig) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
}
