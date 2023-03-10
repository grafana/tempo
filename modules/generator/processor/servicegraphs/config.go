package servicegraphs

import (
	"flag"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	Name = "service-graphs"
)

type Config struct {
	// Wait is the value to wait for an edge to be completed
	Wait time.Duration `yaml:"wait"`
	// MaxItems is the amount of edges that will be stored in the storeMap
	MaxItems int `yaml:"max_items"`

	// Workers is the amount of workers that will be used to process the edges
	Workers int `yaml:"workers"`

	// Buckets for latency histogram in seconds.
	HistogramBuckets []float64 `yaml:"histogram_buckets"`

	// Additional dimensions (labels) to be added to the metric along with the default ones.
	// If client and server spans have the same attribute, behaviour is undetermined
	// (either value could get used)
	Dimensions []string `yaml:"dimensions"`

	// If enabled attribute value will be used for metric calculation
	SpanMultiplierKey string `yaml:"span_multiplier_key"`
}

func (cfg *Config) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	cfg.Wait = 10 * time.Second
	cfg.MaxItems = 10_000
	cfg.Workers = 10
	// TODO: Revisit this default value.
	cfg.HistogramBuckets = prometheus.ExponentialBuckets(0.1, 2, 8)
}
