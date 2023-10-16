package client

import (
	"time"

	filterconfig "github.com/grafana/tempo/pkg/spanfilter/config"
	"github.com/grafana/tempo/pkg/util/listtomap"
)

type Limits struct {
	Forwarders *[]string `json:"forwarders,omitempty"`

	MetricsGenerator *LimitsMetricsGenerator `json:"metrics_generator,omitempty"`
}

func (l *Limits) GetForwarders() ([]string, bool) {
	if l != nil && l.Forwarders != nil {
		return *l.Forwarders, true
	}
	return nil, false
}

func (l *Limits) GetMetricsGenerator() *LimitsMetricsGenerator {
	if l != nil {
		return l.MetricsGenerator
	}
	return nil
}

type LimitsMetricsGenerator struct {
	Processors         listtomap.ListToMap `json:"processors,omitempty"`
	DisableCollection  *bool               `json:"disable_collection,omitempty"`
	CollectionInterval *Duration           `json:"collection_interval,omitempty"`

	Processor *LimitsMetricsGeneratorProcessor `json:"processor,omitempty"`
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

func (l *LimitsMetricsGenerator) GetProcessor() *LimitsMetricsGeneratorProcessor {
	if l != nil {
		return l.Processor
	}
	return nil
}

func (l *LimitsMetricsGenerator) GetCollectionInterval() (time.Duration, bool) {
	if l != nil && l.CollectionInterval != nil {
		return l.CollectionInterval.Duration, true
	}
	return 0, false
}

type LimitsMetricsGeneratorProcessor struct {
	ServiceGraphs *LimitsMetricsGeneratorProcessorServiceGraphs `json:"service_graphs,omitempty"`
	SpanMetrics   *LimitsMetricsGeneratorProcessorSpanMetrics   `json:"span_metrics,omitempty"`
}

func (l *LimitsMetricsGeneratorProcessor) GetServiceGraphs() *LimitsMetricsGeneratorProcessorServiceGraphs {
	if l != nil {
		return l.ServiceGraphs
	}
	return nil
}

func (l *LimitsMetricsGeneratorProcessor) GetSpanMetrics() *LimitsMetricsGeneratorProcessorSpanMetrics {
	if l != nil {
		return l.SpanMetrics
	}
	return nil
}

type LimitsMetricsGeneratorProcessorServiceGraphs struct {
	Dimensions               *[]string  `json:"dimensions,omitempty"`
	EnableClientServerPrefix *bool      `json:"enable_client_server_prefix,omitempty"`
	PeerAttributes           *[]string  `json:"peer_attributes,omitempty"`
	HistogramBuckets         *[]float64 `json:"histogram_buckets,omitempty"`
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
	Dimensions                   *[]string                    `json:"dimensions,omitempty"`
	EnableTargetInfo             *bool                        `json:"enable_target_info,omitempty"`
	FilterPolicies               *[]filterconfig.FilterPolicy `json:"filter_policies,omitempty"`
	HistogramBuckets             *[]float64                   `json:"histogram_buckets,omitempty"`
	TargetInfoExcludedDimensions *[]string                    `json:"target_info_excluded_dimensions,omitempty"`
}

func (l *LimitsMetricsGeneratorProcessorSpanMetrics) GetDimensions() ([]string, bool) {
	if l != nil && l.Dimensions != nil {
		return *l.Dimensions, true
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
