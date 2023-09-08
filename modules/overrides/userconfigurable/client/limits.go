package client

import (
	"encoding/json"
	"fmt"
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

type Duration struct {
	time.Duration
}

// MarshalJSON implements json.Marshaler.
func (d *Duration) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%s"`, d.String())), nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (d *Duration) UnmarshalJSON(input []byte) error {
	var unmarshalledJson interface{}

	err := json.Unmarshal(input, &unmarshalledJson)
	if err != nil {
		return err
	}

	switch value := unmarshalledJson.(type) {
	case float64:
		d.Duration = time.Duration(value)
	case string:
		d.Duration, err = time.ParseDuration(value)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("invalid duration: %#v", unmarshalledJson)
	}

	return nil
}

var _ json.Marshaler = (*Duration)(nil)
var _ json.Unmarshaler = (*Duration)(nil)

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
	Dimensions               *[]string `json:"dimensions,omitempty"`
	EnableClientServerPrefix *bool     `json:"enable_client_server_prefix,omitempty"`
	PeerAttributes           *[]string `json:"peer_attributes,omitempty"`
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

type LimitsMetricsGeneratorProcessorSpanMetrics struct {
	Dimensions       *[]string                    `json:"dimensions,omitempty"`
	EnableTargetInfo *bool                        `json:"enable_target_info,omitempty"`
	FilterPolicies   *[]filterconfig.FilterPolicy `json:"filter_policies,omitempty"`
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
	return nil, true
}
