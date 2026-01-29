package spanmetrics

import (
	"flag"
	"fmt"

	"github.com/grafana/tempo/modules/generator/processor"
	"github.com/grafana/tempo/modules/generator/registry"
	"github.com/grafana/tempo/modules/generator/validation"
	"github.com/grafana/tempo/pkg/sharedconfig"
	filterconfig "github.com/grafana/tempo/pkg/spanfilter/config"
	"github.com/prometheus/client_golang/prometheus"
)

var targetInfoIntrinsicLabelsSet map[string]struct{}

func init() {
	targetInfoIntrinsicLabelsSet = make(map[string]struct{})
	for dim := range validation.SupportedIntrinsicDimensionsSet {
		targetInfoIntrinsicLabelsSet[dim] = struct{}{}
	}
	targetInfoIntrinsicLabelsSet[processor.DimJob] = struct{}{}
	targetInfoIntrinsicLabelsSet[processor.DimInstance] = struct{}{}
}

type Config struct {
	// Buckets for latency histogram in seconds.
	HistogramBuckets []float64 `yaml:"histogram_buckets"`

	// The histogram mode to select.
	HistogramOverride registry.HistogramMode `yaml:"-"`

	// Intrinsic dimensions (labels) added to the metric, that are generated from fixed span
	// data. The dimensions service, span_name, span_kind, status_code, job and instance are enabled by
	// default, whereas the dimension status_message must be enabled explicitly.
	IntrinsicDimensions IntrinsicDimensions `yaml:"intrinsic_dimensions"`

	// Additional dimensions (labels) to be added to the metric. The dimensions are generated
	// from span attributes and are created along with the intrinsic dimensions.
	Dimensions []string `yaml:"dimensions"`

	// Dimension label mapping to allow the user to rename attributes in their metrics
	DimensionMappings []sharedconfig.DimensionMappings `yaml:"dimension_mappings"`

	// Enable target_info as a metrics
	EnableTargetInfo bool `yaml:"enable_target_info"`

	// If enabled attribute value will be used for metric calculation
	SpanMultiplierKey string `yaml:"span_multiplier_key"`

	// Subprocessor options for this Processor include Latency, Count, Size
	// These are metrics categories that exist under the umbrella of Span Metrics
	Subprocessors map[Subprocessor]bool

	// FilterPolicies is a list of policies that will be applied to spans for inclusion or exlusion.
	FilterPolicies []filterconfig.FilterPolicy `yaml:"filter_policies"`

	// Allow user to specify labels they want to drop from target_info
	TargetInfoExcludedDimensions []string `yaml:"target_info_excluded_dimensions"`

	// Allow user to disable instance label from all span metrics series
	EnableInstanceLabel bool `yaml:"enable_instance_label"`
}

func (cfg *Config) RegisterFlagsAndApplyDefaults(string, *flag.FlagSet) {
	cfg.HistogramBuckets = prometheus.ExponentialBuckets(0.002, 2, 14)
	cfg.HistogramOverride = registry.HistogramModeClassic
	cfg.IntrinsicDimensions.Service = true
	cfg.IntrinsicDimensions.SpanName = true
	cfg.IntrinsicDimensions.SpanKind = true
	cfg.IntrinsicDimensions.StatusCode = true
	cfg.Subprocessors = make(map[Subprocessor]bool)
	cfg.Subprocessors[Latency] = true
	cfg.Subprocessors[Count] = true
	cfg.Subprocessors[Size] = true
	cfg.EnableInstanceLabel = true
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
		case processor.DimService:
			ic.Service = active
		case processor.DimSpanName:
			ic.SpanName = active
		case processor.DimSpanKind:
			ic.SpanKind = active
		case processor.DimStatusCode:
			ic.StatusCode = active
		case processor.DimStatusMessage:
			ic.StatusMessage = active
		default:
			return fmt.Errorf("%s is not a valid intrinsic dimension", label)
		}
	}
	return nil
}
