package userconfigurableapi

import (
	"github.com/grafana/tempo/pkg/util/listtomap"
)

type UserConfigurableLimits struct {
	Forwarders *[]string `json:"forwarders,omitempty"`

	MetricsGenerator *UserConfigurableOverridesMetricsGenerator `json:"metrics_generator,omitempty"`
}

func (l *UserConfigurableLimits) GetForwarders() ([]string, bool) {
	if l != nil && l.Forwarders != nil {
		return *l.Forwarders, true
	}
	return nil, false
}

func (l *UserConfigurableLimits) GetMetricsGenerator() *UserConfigurableOverridesMetricsGenerator {
	if l != nil {
		return l.MetricsGenerator
	}
	return nil
}

type UserConfigurableOverridesMetricsGenerator struct {
	Processors        listtomap.ListToMap `json:"processors,omitempty"`
	DisableCollection *bool               `json:"disable_collection,omitempty"`

	Processor *UserConfigurableOverridesMetricsGeneratorProcessor `json:"processor,omitempty"`
}

func (l *UserConfigurableOverridesMetricsGenerator) GetProcessors() (listtomap.ListToMap, bool) {
	if l != nil && l.Processors != nil {
		return l.Processors, true
	}
	return nil, false
}

func (l *UserConfigurableOverridesMetricsGenerator) GetDisableCollection() (bool, bool) {
	if l != nil && l.DisableCollection != nil {
		return *l.DisableCollection, true
	}
	return false, false
}

func (l *UserConfigurableOverridesMetricsGenerator) GetProcessor() *UserConfigurableOverridesMetricsGeneratorProcessor {
	if l != nil {
		return l.Processor
	}
	return nil
}

type UserConfigurableOverridesMetricsGeneratorProcessor struct {
	ServiceGraphs *UserConfigurableOverridesMetricsGeneratorProcessorServiceGraphs `json:"service_graphs,omitempty"`
	SpanMetrics   *UserConfigurableOverridesMetricsGeneratorProcessorSpanMetrics   `json:"span_metrics,omitempty"`
}

func (l *UserConfigurableOverridesMetricsGeneratorProcessor) GetServiceGraphs() *UserConfigurableOverridesMetricsGeneratorProcessorServiceGraphs {
	if l != nil {
		return l.ServiceGraphs
	}
	return nil
}

func (l *UserConfigurableOverridesMetricsGeneratorProcessor) GetSpanMetrics() *UserConfigurableOverridesMetricsGeneratorProcessorSpanMetrics {
	if l != nil {
		return l.SpanMetrics
	}
	return nil
}

type UserConfigurableOverridesMetricsGeneratorProcessorServiceGraphs struct {
	Dimensions               *[]string `json:"dimensions,omitempty"`
	EnableClientServerPrefix *bool     `json:"enable_client_server_prefix,omitempty"`
	PeerAttributes           *[]string `json:"peer_attributes,omitempty"`
}

func (l *UserConfigurableOverridesMetricsGeneratorProcessorServiceGraphs) GetDimensions() ([]string, bool) {
	if l != nil && l.Dimensions != nil {
		return *l.Dimensions, true
	}
	return nil, false
}

func (l *UserConfigurableOverridesMetricsGeneratorProcessorServiceGraphs) GetEnableClientServerPrefix() (bool, bool) {
	if l != nil && l.EnableClientServerPrefix != nil {
		return *l.EnableClientServerPrefix, true
	}
	return false, false
}

func (l *UserConfigurableOverridesMetricsGeneratorProcessorServiceGraphs) GetPeerAttributes() ([]string, bool) {
	if l != nil && l.PeerAttributes != nil {
		return *l.PeerAttributes, true
	}
	return nil, false
}

type UserConfigurableOverridesMetricsGeneratorProcessorSpanMetrics struct {
	Dimensions       *[]string `json:"dimensions,omitempty"`
	EnableTargetInfo *bool     `json:"enable_target_info,omitempty"`
}

func (l *UserConfigurableOverridesMetricsGeneratorProcessorSpanMetrics) GetDimensions() ([]string, bool) {
	if l != nil && l.Dimensions != nil {
		return *l.Dimensions, true
	}
	return nil, false
}

func (l *UserConfigurableOverridesMetricsGeneratorProcessorSpanMetrics) GetEnableTargetInfo() (bool, bool) {
	if l != nil && l.EnableTargetInfo != nil {
		return *l.EnableTargetInfo, true
	}
	return false, false
}
