package spanmetrics

import (
	"flag"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	Name = "span-metrics"
)

type Config struct {
	// Buckets for latency histogram in seconds.
	HistogramBuckets []float64 `yaml:"histogram_buckets"`
	// Additional dimensions (labels) to be added to the metric,
	// along with the default ones (service, span_name, span_kind and span_status).
	Dimensions []string `yaml:"dimensions"`
}

func (cfg *Config) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	// TODO: Revisit this default value.
	cfg.HistogramBuckets = prometheus.ExponentialBuckets(0.002, 2, 12)
}
