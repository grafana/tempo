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
	Wait time.Duration `mapstructure:"wait"`
	// MaxItems is the amount of edges that will be stored in the storeMap
	MaxItems int `mapstructure:"max_items"`

	// Workers is the amount of workers that will be used to process the edges
	Workers int `mapstructure:"workers"`

	// Buckets for latency histogram in seconds.
	HistogramBuckets []float64 `yaml:"histogram_buckets"`

	// SuccessCodes *successCodes `mapstructure:"success_codes"`
}

func (cfg *Config) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	cfg.Wait = 10 * time.Second
	cfg.MaxItems = 10_000
	cfg.Workers = 10
	// TODO: Revisit this default value.
	cfg.HistogramBuckets = prometheus.ExponentialBuckets(0.1, 2, 8)
}
