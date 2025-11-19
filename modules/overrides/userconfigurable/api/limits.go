package api

import (
	"time"

	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/modules/overrides/histograms"
	"github.com/grafana/tempo/modules/overrides/userconfigurable/client"
	"github.com/grafana/tempo/pkg/sharedconfig"
	"github.com/grafana/tempo/pkg/spanfilter/config"
)

// limitsFromOverrides will reconstruct a client.Limits from the overrides module
func limitsFromOverrides(overrides overrides.Interface, userID string) *client.Limits {
	return &client.Limits{
		Forwarders: strArrPtr(overrides.Forwarders(userID)),
		MetricsGenerator: client.LimitsMetricsGenerator{
			Processors:                      overrides.MetricsGeneratorProcessors(userID),
			DisableCollection:               boolPtr(overrides.MetricsGeneratorDisableCollection(userID)),
			CollectionInterval:              timePtr(overrides.MetricsGeneratorCollectionInterval(userID)),
			TraceIDLabelName:                strPtr(overrides.MetricsGeneratorTraceIDLabelName(userID)),
			IngestionSlack:                  timePtr(overrides.MetricsGeneratorIngestionSlack(userID)),
			GenerateNativeHistograms:        histogramModePtr(overrides.MetricsGeneratorGenerateNativeHistograms(userID)),
			NativeHistogramMaxBucketNumber:  uint32Ptr(overrides.MetricsGeneratorNativeHistogramMaxBucketNumber(userID)),
			NativeHistogramBucketFactor:     floatPtr(overrides.MetricsGeneratorNativeHistogramBucketFactor(userID)),
			NativeHistogramMinResetDuration: timePtr(overrides.MetricsGeneratorNativeHistogramMinResetDuration(userID)),
			Processor: client.LimitsMetricsGeneratorProcessor{
				ServiceGraphs: client.LimitsMetricsGeneratorProcessorServiceGraphs{
					Dimensions:               strArrPtr(overrides.MetricsGeneratorProcessorServiceGraphsDimensions(userID)),
					EnableClientServerPrefix: boolPtr(overrides.MetricsGeneratorProcessorServiceGraphsEnableClientServerPrefix(userID)),
					PeerAttributes:           strArrPtr(overrides.MetricsGeneratorProcessorServiceGraphsPeerAttributes(userID)),
					HistogramBuckets:         floatArrPtr(overrides.MetricsGeneratorProcessorServiceGraphsHistogramBuckets(userID)),
				},
				SpanMetrics: client.LimitsMetricsGeneratorProcessorSpanMetrics{
					Dimensions:          strArrPtr(overrides.MetricsGeneratorProcessorSpanMetricsDimensions(userID)),
					IntrinsicDimensions: mapBoolPtr(overrides.MetricsGeneratorProcessorSpanMetricsIntrinsicDimensions(userID)),
					DimensionMappings:   dimensionMappingsPtr(overrides.MetricsGeneratorProcessorSpanMetricsDimensionMappings(userID)),
					EnableTargetInfo: func() *bool {
						val, _ := overrides.MetricsGeneratorProcessorSpanMetricsEnableTargetInfo(userID)
						return boolPtr(val)
					}(),
					FilterPolicies:               filterPoliciesPtr(overrides.MetricsGeneratorProcessorSpanMetricsFilterPolicies(userID)),
					HistogramBuckets:             floatArrPtr(overrides.MetricsGeneratorProcessorSpanMetricsHistogramBuckets(userID)),
					TargetInfoExcludedDimensions: strArrPtr(overrides.MetricsGeneratorProcessorSpanMetricsTargetInfoExcludedDimensions(userID)),
					EnableInstanceLabel: func() *bool {
						val, _ := overrides.MetricsGeneratorProcessorSpanMetricsEnableInstanceLabel(userID)
						return boolPtr(val)
					}(),
				},
				HostInfo: client.LimitsMetricGeneratorProcessorHostInfo{
					HostIdentifiers: strArrPtr(overrides.MetricsGeneratorProcessorHostInfoHostIdentifiers(userID)),
					MetricName:      strPtr(overrides.MetricsGeneratorProcessorHostInfoMetricName(userID)),
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

func strPtr(s string) *string {
	return &s
}

func strArrPtr(s []string) *[]string {
	return &s
}

func floatArrPtr(f []float64) *[]float64 {
	return &f
}

func floatPtr(f float64) *float64 {
	return &f
}

func mapBoolPtr(m map[string]bool) *map[string]bool {
	return &m
}

func dimensionMappingsPtr(m []sharedconfig.DimensionMappings) *[]sharedconfig.DimensionMappings {
	if len(m) == 0 {
		return nil
	}
	return &m
}

func filterPoliciesPtr(p []config.FilterPolicy) *[]config.FilterPolicy {
	return &p
}

func histogramModePtr(h histograms.HistogramMethod) *histograms.HistogramMethod {
	return &h
}

func uint32Ptr(u uint32) *uint32 {
	return &u
}
