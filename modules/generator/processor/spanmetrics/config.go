package spanmetrics

import (
	"flag"
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
	cfg.HistogramBuckets = []float64{.002, .004, .006, .01, .05, .1, .2, .4, .8, 1., 1.4, 2., 5., 10., 15.}
}
