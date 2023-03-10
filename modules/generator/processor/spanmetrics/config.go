package spanmetrics

import (
	"flag"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	Name = "span-metrics"

	dimService       = "service"
	dimSpanName      = "span_name"
	dimSpanKind      = "span_kind"
	dimStatusCode    = "status_code"
	dimStatusMessage = "status_message"
)

type Config struct {
	// Buckets for latency histogram in seconds.
	HistogramBuckets []float64 `yaml:"histogram_buckets"`
	// Intrinsic dimensions (labels) added to the metric, that are generated from fixed span
	// data. The dimensions service, span_name, span_kind, and status_code are enabled by
	// default, whereas the dimension status_message must be enabled explicitly.
	IntrinsicDimensions IntrinsicDimensions `yaml:"intrinsic_dimensions"`
	// Additional dimensions (labels) to be added to the metric. The dimensions are generated
	// from span attributes and are created along with the intrinsic dimensions.
	Dimensions []string `yaml:"dimensions"`

	// If enabled attribute value will be used for metric calculation
	SpanMultiplierKey string `yaml:"span_multiplier_key"`
}

func (cfg *Config) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	cfg.HistogramBuckets = prometheus.ExponentialBuckets(0.002, 2, 14)
	cfg.IntrinsicDimensions.Service = true
	cfg.IntrinsicDimensions.SpanName = true
	cfg.IntrinsicDimensions.SpanKind = true
	cfg.IntrinsicDimensions.StatusCode = true
}

type IntrinsicDimensions struct {
	Service       bool `yaml:"service"`
	SpanName      bool `yaml:"span_name"`
	SpanKind      bool `yaml:"span_kind"`
	StatusCode    bool `yaml:"status_code"`
	StatusMessage bool `yaml:"status_message,omitempty"`
}

func (ic *IntrinsicDimensions) ApplyFromMap(dimensions map[string]bool) error {
	for label, active := range dimensions {
		switch label {
		case dimService:
			ic.Service = active
		case dimSpanName:
			ic.SpanName = active
		case dimSpanKind:
			ic.SpanKind = active
		case dimStatusCode:
			ic.StatusCode = active
		case dimStatusMessage:
			ic.StatusMessage = active
		default:
			return errors.Errorf("%s is not a valid intrinsic dimension", label)
		}
	}
	return nil
}
