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
		Forwarders: new(overrides.Forwarders(userID)),
		MetricsGenerator: client.LimitsMetricsGenerator{
			Processors:                      overrides.MetricsGeneratorProcessors(userID),
			DisableCollection:               new(overrides.MetricsGeneratorDisableCollection(userID)),
			CollectionInterval:              timePtr(overrides.MetricsGeneratorCollectionInterval(userID)),
			TraceIDLabelName:                new(overrides.MetricsGeneratorTraceIDLabelName(userID)),
			IngestionSlack:                  timePtr(overrides.MetricsGeneratorIngestionSlack(userID)),
			GenerateNativeHistograms:        new(overrides.MetricsGeneratorGenerateNativeHistograms(userID)),
			NativeHistogramMaxBucketNumber:  new(overrides.MetricsGeneratorNativeHistogramMaxBucketNumber(userID)),
			NativeHistogramBucketFactor:     new(overrides.MetricsGeneratorNativeHistogramBucketFactor(userID)),
			NativeHistogramMinResetDuration: timePtr(overrides.MetricsGeneratorNativeHistogramMinResetDuration(userID)),
			SpanNameSanitization: func() *string {
				s := overrides.MetricsGeneratorSpanNameSanitization(userID)
				if s == "" {
					return nil
				}
				return &s
			}(),
			Processor: client.LimitsMetricsGeneratorProcessor{
				ServiceGraphs: client.LimitsMetricsGeneratorProcessorServiceGraphs{
					Dimensions:               new(overrides.MetricsGeneratorProcessorServiceGraphsDimensions(userID)),
					EnableClientServerPrefix: new(overrides.MetricsGeneratorProcessorServiceGraphsEnableClientServerPrefix(userID)),
					PeerAttributes:           new(overrides.MetricsGeneratorProcessorServiceGraphsPeerAttributes(userID)),
					FilterPolicies:           new(overrides.MetricsGeneratorProcessorServiceGraphsFilterPolicies(userID)),
					HistogramBuckets:         new(overrides.MetricsGeneratorProcessorServiceGraphsHistogramBuckets(userID)),
				},
				SpanMetrics: client.LimitsMetricsGeneratorProcessorSpanMetrics{
					Dimensions:          new(overrides.MetricsGeneratorProcessorSpanMetricsDimensions(userID)),
					IntrinsicDimensions: new(overrides.MetricsGeneratorProcessorSpanMetricsIntrinsicDimensions(userID)),
					DimensionMappings:   dimensionMappingsPtr(overrides.MetricsGeneratorProcessorSpanMetricsDimensionMappings(userID)),
					EnableTargetInfo: func() *bool {
						val, _ := overrides.MetricsGeneratorProcessorSpanMetricsEnableTargetInfo(userID)
						return new(val)
					}(),
					FilterPolicies:               new(overrides.MetricsGeneratorProcessorSpanMetricsFilterPolicies(userID)),
					HistogramBuckets:             new(overrides.MetricsGeneratorProcessorSpanMetricsHistogramBuckets(userID)),
					TargetInfoExcludedDimensions: new(overrides.MetricsGeneratorProcessorSpanMetricsTargetInfoExcludedDimensions(userID)),
					EnableInstanceLabel: func() *bool {
						val, _ := overrides.MetricsGeneratorProcessorSpanMetricsEnableInstanceLabel(userID)
						return new(val)
					}(),
				},
				HostInfo: client.LimitsMetricGeneratorProcessorHostInfo{
					HostIdentifiers: new(overrides.MetricsGeneratorProcessorHostInfoHostIdentifiers(userID)),
					MetricName:      new(overrides.MetricsGeneratorProcessorHostInfoMetricName(userID)),
				},
			},
		},
	}
}

//go:fix inline
func boolPtr(b bool) *bool {
	return new(b)
}

func timePtr(t time.Duration) *client.Duration {
	return &client.Duration{Duration: t}
}

//go:fix inline
func strPtr(s string) *string {
	return new(s)
}

//go:fix inline
func strArrPtr(s []string) *[]string {
	return new(s)
}

//go:fix inline
func floatArrPtr(f []float64) *[]float64 {
	return new(f)
}

//go:fix inline
func floatPtr(f float64) *float64 {
	return new(f)
}

//go:fix inline
func mapBoolPtr(m map[string]bool) *map[string]bool {
	return new(m)
}

func dimensionMappingsPtr(m []sharedconfig.DimensionMappings) *[]sharedconfig.DimensionMappings {
	if len(m) == 0 {
		return nil
	}
	return &m
}

//go:fix inline
func filterPoliciesPtr(p []config.FilterPolicy) *[]config.FilterPolicy {
	return new(p)
}

//go:fix inline
func histogramModePtr(h histograms.HistogramMethod) *histograms.HistogramMethod {
	return new(h)
}

//go:fix inline
func uint32Ptr(u uint32) *uint32 {
	return new(u)
}
