package secretdetection

import (
	"flag"

	"github.com/grafana/tempo/pkg/sharedconfig"
)

type Config struct {
	// SpanMetricsInfo holds the span-metrics configuration needed to determine
	// whether a detected secret will land in generated metrics (Mimir).
	// Populated at construction time from the tenant's span-metrics config.
	SpanMetricsInfo SpanMetricsInfo `yaml:"-"`
}

// SpanMetricsInfo contains the subset of span-metrics config needed to
// determine if a secret-bearing attribute will appear as a metric label value.
type SpanMetricsInfo struct {
	// Dimensions is the set of span attribute keys configured as span-metrics dimensions.
	Dimensions map[string]struct{}
	// DimensionMappingSourceLabels is the set of attribute keys used as source labels
	// in dimension mappings. A secret in any of these ends up in metrics.
	DimensionMappingSourceLabels map[string]struct{}
	// EnableTargetInfo indicates whether target_info is enabled, which dumps all
	// resource attributes as metric label values.
	EnableTargetInfo bool
	// TargetInfoExcludedDimensions is the set of resource attribute keys excluded from target_info.
	TargetInfoExcludedDimensions map[string]struct{}
}

// NewSpanMetricsInfo builds a SpanMetricsInfo from the raw span-metrics config fields.
func NewSpanMetricsInfo(dimensions []string, dimensionMappings []sharedconfig.DimensionMappings, enableTargetInfo bool, targetInfoExcludedDimensions []string) SpanMetricsInfo {
	dimSet := make(map[string]struct{}, len(dimensions))
	for _, d := range dimensions {
		dimSet[d] = struct{}{}
	}

	srcSet := make(map[string]struct{})
	for _, m := range dimensionMappings {
		for _, src := range m.SourceLabel {
			srcSet[src] = struct{}{}
		}
	}

	exclSet := make(map[string]struct{}, len(targetInfoExcludedDimensions))
	for _, d := range targetInfoExcludedDimensions {
		exclSet[d] = struct{}{}
	}

	return SpanMetricsInfo{
		Dimensions:                   dimSet,
		DimensionMappingSourceLabels: srcSet,
		EnableTargetInfo:             enableTargetInfo,
		TargetInfoExcludedDimensions: exclSet,
	}
}

func (cfg *Config) RegisterFlagsAndApplyDefaults(string, *flag.FlagSet) {}
