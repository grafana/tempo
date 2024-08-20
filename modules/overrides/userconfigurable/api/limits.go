package api

import (
	"time"

	"github.com/grafana/tempo/v2/modules/overrides"
	"github.com/grafana/tempo/v2/modules/overrides/userconfigurable/client"
	"github.com/grafana/tempo/v2/pkg/spanfilter/config"
)

// limitsFromOverrides will reconstruct a client.Limits from the overrides module
func limitsFromOverrides(overrides overrides.Interface, userID string) *client.Limits {
	return &client.Limits{
		Forwarders: strArrPtr(overrides.Forwarders(userID)),
		MetricsGenerator: client.LimitsMetricsGenerator{
			Processors:         overrides.MetricsGeneratorProcessors(userID),
			DisableCollection:  boolPtr(overrides.MetricsGeneratorDisableCollection(userID)),
			CollectionInterval: timePtr(overrides.MetricsGeneratorCollectionInterval(userID)),
			Processor: client.LimitsMetricsGeneratorProcessor{
				ServiceGraphs: client.LimitsMetricsGeneratorProcessorServiceGraphs{
					Dimensions:               strArrPtr(overrides.MetricsGeneratorProcessorServiceGraphsDimensions(userID)),
					EnableClientServerPrefix: boolPtr(overrides.MetricsGeneratorProcessorServiceGraphsEnableClientServerPrefix(userID)),
					PeerAttributes:           strArrPtr(overrides.MetricsGeneratorProcessorServiceGraphsPeerAttributes(userID)),
					HistogramBuckets:         floatArrPtr(overrides.MetricsGeneratorProcessorServiceGraphsHistogramBuckets(userID)),
				},
				SpanMetrics: client.LimitsMetricsGeneratorProcessorSpanMetrics{
					Dimensions:                   strArrPtr(overrides.MetricsGeneratorProcessorSpanMetricsDimensions(userID)),
					EnableTargetInfo:             boolPtr(overrides.MetricsGeneratorProcessorSpanMetricsEnableTargetInfo(userID)),
					FilterPolicies:               filterPoliciesPtr(overrides.MetricsGeneratorProcessorSpanMetricsFilterPolicies(userID)),
					HistogramBuckets:             floatArrPtr(overrides.MetricsGeneratorProcessorSpanMetricsHistogramBuckets(userID)),
					TargetInfoExcludedDimensions: strArrPtr(overrides.MetricsGeneratorProcessorSpanMetricsTargetInfoExcludedDimensions(userID)),
				},
			},
		},
	}
}

func boolPtr(b bool) *bool {
	return &b
}

func timePtr(t time.Duration) *client.Duration {
	return &client.Duration{Duration: t}
}

func strArrPtr(s []string) *[]string {
	return &s
}

func floatArrPtr(f []float64) *[]float64 {
	return &f
}

func filterPoliciesPtr(p []config.FilterPolicy) *[]config.FilterPolicy {
	return &p
}
