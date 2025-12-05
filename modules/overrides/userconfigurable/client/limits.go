package client

import (
	"time"

	"github.com/grafana/tempo/modules/overrides/histograms"
	"github.com/grafana/tempo/pkg/sharedconfig"
	filterconfig "github.com/grafana/tempo/pkg/spanfilter/config"
	"github.com/grafana/tempo/pkg/util/listtomap"
)

type Limits struct {
	Forwarders       *[]string              `yaml:"forwarders,omitempty" json:"forwarders,omitempty"`
	CostAttribution  CostAttribution        `yaml:"cost_attribution,omitempty" json:"cost_attribution,omitempty"`
	MetricsGenerator LimitsMetricsGenerator `yaml:"metrics_generator,omitempty" json:"metrics_generator,omitempty"`
}

func (l *Limits) GetForwarders() ([]string, bool) {
	if l != nil && l.Forwarders != nil {
		return *l.Forwarders, true
	}
	return nil, false
}

func (l *Limits) GetMetricsGenerator() *LimitsMetricsGenerator {
	if l != nil {
		return &l.MetricsGenerator
	}
	return nil
}

func (l *Limits) GetCostAttribution() *CostAttribution {
	if l != nil {
		return &l.CostAttribution
	}
	return nil
}

type LimitsMetricsGenerator struct {
	Processors                      listtomap.ListToMap         `yaml:"processors,omitempty" json:"processors,omitempty"`
	DisableCollection               *bool                       `yaml:"disable_collection,omitempty" json:"disable_collection,omitempty"`
	CollectionInterval              *Duration                   `yaml:"collection_interval,omitempty" json:"collection_interval,omitempty"`
	TraceIDLabelName                *string                     `yaml:"trace_id_label_name,omitempty" json:"trace_id_label_name,omitempty"`
	IngestionSlack                  *Duration                   `yaml:"ingestion_time_range_slack,omitempty" json:"ingestion_time_range_slack,omitempty"`
	GenerateNativeHistograms        *histograms.HistogramMethod `yaml:"generate_native_histograms" json:"generate_native_histograms,omitempty"`
	NativeHistogramMaxBucketNumber  *uint32                     `yaml:"native_histogram_max_bucket_number,omitempty" json:"native_histogram_max_bucket_number,omitempty"`
	NativeHistogramBucketFactor     *float64                    `yaml:"native_histogram_bucket_factor,omitempty" json:"native_histogram_bucket_factor,omitempty"`
	NativeHistogramMinResetDuration *Duration                   `yaml:"native_histogram_min_reset_duration,omitempty" json:"native_histogram_min_reset_duration,omitempty"`

	Processor LimitsMetricsGeneratorProcessor `yaml:"processor,omitempty" json:"processor,omitempty"`
}

func (l *LimitsMetricsGenerator) GetProcessors() (listtomap.ListToMap, bool) {
	if l != nil && l.Processors != nil {
		return l.Processors, true
	}
	return nil, false
}

func (l *LimitsMetricsGenerator) GetDisableCollection() (bool, bool) {
	if l != nil && l.DisableCollection != nil {
		return *l.DisableCollection, true
	}
	return false, false
}

func (l *LimitsMetricsGenerator) GetGenerateNativeHistograms() (histograms.HistogramMethod, bool) {
	if l != nil && l.GenerateNativeHistograms != nil {
		return *l.GenerateNativeHistograms, true
	}
	return histograms.HistogramMethodClassic, false
}

func (l *LimitsMetricsGenerator) GetNativeHistogramMaxBucketNumber() (uint32, bool) {
	if l != nil && l.NativeHistogramMaxBucketNumber != nil && *l.NativeHistogramMaxBucketNumber > 0 {
		return *l.NativeHistogramMaxBucketNumber, true
	}
	return 0, false
}

func (l *LimitsMetricsGenerator) GetNativeHistogramBucketFactor() (float64, bool) {
	if l != nil && l.NativeHistogramBucketFactor != nil {
		return *l.NativeHistogramBucketFactor, true
	}
	return 0, false
}

func (l *LimitsMetricsGenerator) GetNativeHistogramMinResetDuration() (time.Duration, bool) {
	if l != nil && l.NativeHistogramMinResetDuration != nil {
		return l.NativeHistogramMinResetDuration.Duration, true
	}
	return 0, false
}

func (l *LimitsMetricsGenerator) GetProcessor() *LimitsMetricsGeneratorProcessor {
	if l != nil {
		return &l.Processor
	}
	return nil
}

func (l *LimitsMetricsGenerator) GetCollectionInterval() (time.Duration, bool) {
	if l != nil && l.CollectionInterval != nil {
		return l.CollectionInterval.Duration, true
	}
	return 0, false
}

func (l *LimitsMetricsGenerator) GetTraceIDLabelName() (string, bool) {
	if l != nil && l.TraceIDLabelName != nil {
		return *l.TraceIDLabelName, true
	}
	return "", false
}

func (l *LimitsMetricsGenerator) GetIngestionSlack() (time.Duration, bool) {
	if l != nil && l.IngestionSlack != nil {
		return l.IngestionSlack.Duration, true
	}
	return 0, false
}

type LimitsMetricsGeneratorProcessor struct {
	ServiceGraphs LimitsMetricsGeneratorProcessorServiceGraphs `yaml:"service_graphs,omitempty" json:"service_graphs,omitempty"`
	SpanMetrics   LimitsMetricsGeneratorProcessorSpanMetrics   `yaml:"span_metrics,omitempty" json:"span_metrics,omitempty"`
	HostInfo      LimitsMetricGeneratorProcessorHostInfo       `yaml:"host_info,omitempty" json:"host_info,omitempty"`
}

func (l *LimitsMetricsGeneratorProcessor) GetServiceGraphs() *LimitsMetricsGeneratorProcessorServiceGraphs {
	if l != nil {
		return &l.ServiceGraphs
	}
	return nil
}

func (l *LimitsMetricsGeneratorProcessor) GetSpanMetrics() *LimitsMetricsGeneratorProcessorSpanMetrics {
	if l != nil {
		return &l.SpanMetrics
	}
	return nil
}

func (l *LimitsMetricsGeneratorProcessor) GetHostInfo() *LimitsMetricGeneratorProcessorHostInfo {
	if l != nil {
		return &l.HostInfo
	}
	return nil
}

type LimitsMetricsGeneratorProcessorServiceGraphs struct {
	Dimensions                            *[]string  `yaml:"dimensions,omitempty" json:"dimensions,omitempty"`
	EnableClientServerPrefix              *bool      `yaml:"enable_client_server_prefix,omitempty" json:"enable_client_server_prefix,omitempty"`
	EnableMessagingSystemLatencyHistogram *bool      `yaml:"enable_messaging_system_latency_histogram,omitempty" json:"enable_messaging_system_latency_histogram,omitempty"`
	EnableVirtualNodeLabel                *bool      `yaml:"enable_virtual_node_label,omitempty" json:"enable_virtual_node_label,omitempty"`
	PeerAttributes                        *[]string  `yaml:"peer_attributes,omitempty" json:"peer_attributes,omitempty"`
	HistogramBuckets                      *[]float64 `yaml:"histogram_buckets,omitempty" json:"histogram_buckets,omitempty"`
}

func (l *LimitsMetricsGeneratorProcessorServiceGraphs) GetDimensions() ([]string, bool) {
	if l != nil && l.Dimensions != nil {
		return *l.Dimensions, true
	}
	return nil, false
}

func (l *LimitsMetricsGeneratorProcessorServiceGraphs) GetEnableClientServerPrefix() (bool, bool) {
	if l != nil && l.EnableClientServerPrefix != nil {
		return *l.EnableClientServerPrefix, true
	}
	return false, false
}

func (l *LimitsMetricsGeneratorProcessorServiceGraphs) GetEnableVirtualNodeLabel() (bool, bool) {
	if l != nil && l.EnableVirtualNodeLabel != nil {
		return *l.EnableVirtualNodeLabel, true
	}
	return false, false
}

func (l *LimitsMetricsGeneratorProcessorServiceGraphs) GetPeerAttributes() ([]string, bool) {
	if l != nil && l.PeerAttributes != nil {
		return *l.PeerAttributes, true
	}
	return nil, false
}

func (l *LimitsMetricsGeneratorProcessorServiceGraphs) GetHistogramBuckets() ([]float64, bool) {
	if l != nil && l.HistogramBuckets != nil {
		return *l.HistogramBuckets, true
	}
	return nil, false
}

type LimitsMetricsGeneratorProcessorSpanMetrics struct {
	Dimensions                   *[]string                         `yaml:"dimensions,omitempty" json:"dimensions,omitempty"`
	IntrinsicDimensions          *map[string]bool                  `yaml:"intrinsic_dimensions,omitempty" json:"intrinsic_dimensions,omitempty"`
	DimensionMappings            *[]sharedconfig.DimensionMappings `yaml:"dimension_mappings,omitempty" json:"dimension_mappings,omitempty"`
	EnableTargetInfo             *bool                             `yaml:"enable_target_info,omitempty" json:"enable_target_info,omitempty"`
	FilterPolicies               *[]filterconfig.FilterPolicy      `yaml:"filter_policies,omitempty" json:"filter_policies,omitempty"`
	HistogramBuckets             *[]float64                        `yaml:"histogram_buckets,omitempty" json:"histogram_buckets,omitempty"`
	TargetInfoExcludedDimensions *[]string                         `yaml:"target_info_excluded_dimensions,omitempty" json:"target_info_excluded_dimensions,omitempty"`
	EnableInstanceLabel          *bool                             `yaml:"enable_instance_label,omitempty" json:"enable_instance_label,omitempty"`
}

func (l *LimitsMetricsGeneratorProcessorSpanMetrics) GetDimensions() ([]string, bool) {
	if l != nil && l.Dimensions != nil {
		return *l.Dimensions, true
	}
	return nil, false
}

func (l *LimitsMetricsGeneratorProcessorSpanMetrics) GetIntrinsicDimensions() (map[string]bool, bool) {
	if l != nil && l.IntrinsicDimensions != nil {
		return *l.IntrinsicDimensions, true
	}
	return nil, false
}

func (l *LimitsMetricsGeneratorProcessorSpanMetrics) GetDimensionMappings() ([]sharedconfig.DimensionMappings, bool) {
	if l != nil && l.DimensionMappings != nil {
		return *l.DimensionMappings, true
	}
	return nil, false
}

func (l *LimitsMetricsGeneratorProcessorSpanMetrics) GetEnableTargetInfo() (bool, bool) {
	if l != nil && l.EnableTargetInfo != nil {
		return *l.EnableTargetInfo, true
	}
	return false, false
}

func (l *LimitsMetricsGeneratorProcessorSpanMetrics) GetFilterPolicies() ([]filterconfig.FilterPolicy, bool) {
	if l != nil && l.FilterPolicies != nil {
		return *l.FilterPolicies, true
	}
	return nil, false
}

func (l *LimitsMetricsGeneratorProcessorSpanMetrics) GetHistogramBuckets() ([]float64, bool) {
	if l != nil && l.HistogramBuckets != nil {
		return *l.HistogramBuckets, true
	}
	return nil, false
}

func (l *LimitsMetricsGeneratorProcessorSpanMetrics) GetTargetInfoExcludedDimensions() ([]string, bool) {
	if l != nil && l.TargetInfoExcludedDimensions != nil {
		return *l.TargetInfoExcludedDimensions, true
	}
	return nil, false
}

func (l *LimitsMetricsGeneratorProcessorSpanMetrics) GetEnableInstanceLabel() (bool, bool) {
	if l != nil && l.EnableInstanceLabel != nil {
		return *l.EnableInstanceLabel, true
	}
	return true, false // default to true if not set
}

type LimitsMetricGeneratorProcessorHostInfo struct {
	HostIdentifiers *[]string `yaml:"host_identifiers,omitempty" json:"host_identifiers,omitempty"`
	MetricName      *string   `yaml:"metric_name,omitempty" json:"metric_name,omitempty"`
}

func (l *LimitsMetricGeneratorProcessorHostInfo) GetHostIdentifiers() ([]string, bool) {
	if l != nil && l.HostIdentifiers != nil {
		return *l.HostIdentifiers, true
	}
	return nil, false
}

func (l *LimitsMetricGeneratorProcessorHostInfo) GetMetricName() (string, bool) {
	if l != nil && l.MetricName != nil {
		return *l.MetricName, true
	}
	return "", false
}

type CostAttribution struct {
	Dimensions *map[string]string `yaml:"dimensions,omitempty" json:"dimensions,omitempty"`
}

func (l *CostAttribution) GetDimensions() (map[string]string, bool) {
	if l != nil && l.Dimensions != nil {
		return *l.Dimensions, true
	}
	return nil, false
}
