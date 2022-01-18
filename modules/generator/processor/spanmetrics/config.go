package spanmetrics

import (
	"flag"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type Config struct {
	// Duration after which to delete an inactive metric. A metric series inactive if it hasn't been
	// updated anymore.
	// Default: 2m
	DeleteAfterLastUpdate time.Duration `yaml:"delete_after_last_update"`
	// Buckets for latency histogram in seconds.
	HistogramBuckets []float64 `yaml:"histogram_buckets"`
	// Additional dimensions (labels) to be added to the metric,
	// along with the default ones (service, span_name, span_kind and span_status).
	Dimensions []string `yaml:"dimensions"`
}

func (cfg *Config) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	cfg.DeleteAfterLastUpdate = 2 * time.Minute
	// TODO: Revisit this default value.
	cfg.HistogramBuckets = prometheus.ExponentialBuckets(0.1, 2, 8)
}
