package overrides

import (
	"encoding/json"
	"fmt"
	"maps"
	"sync"
	"time"

	"github.com/grafana/tempo/modules/overrides/histograms"
	"github.com/grafana/tempo/pkg/util/listtomap"
	"github.com/grafana/tempo/tempodb/backend"

	"github.com/prometheus/common/model"

	"github.com/grafana/tempo/pkg/sharedconfig"
	filterconfig "github.com/grafana/tempo/pkg/spanfilter/config"
)

func (c *Overrides) toLegacy() LegacyOverrides {
	return LegacyOverrides{
		IngestionRateStrategy:      c.Ingestion.RateStrategy,
		IngestionRateLimitBytes:    c.Ingestion.RateLimitBytes,
		IngestionBurstSizeBytes:    c.Ingestion.BurstSizeBytes,
		IngestionTenantShardSize:   c.Ingestion.TenantShardSize,
		MaxLocalTracesPerUser:      c.Ingestion.MaxLocalTracesPerUser,
		MaxGlobalTracesPerUser:     c.Ingestion.MaxGlobalTracesPerUser,
		IngestionMaxAttributeBytes: c.Ingestion.MaxAttributeBytes,
		IngestionArtificialDelay:   c.Ingestion.ArtificialDelay,
		IngestionRetryInfoEnabled:  c.Ingestion.RetryInfoEnabled,

		Forwarders: c.Forwarders,

		MetricsGeneratorRingSize:                                                    c.MetricsGenerator.RingSize,
		MetricsGeneratorProcessors:                                                  c.MetricsGenerator.Processors,
		MetricsGeneratorMaxActiveSeries:                                             c.MetricsGenerator.MaxActiveSeries,
		MetricsGeneratorMaxActiveEntities:                                           c.MetricsGenerator.MaxActiveEntities,
		MetricsGeneratorCollectionInterval:                                          c.MetricsGenerator.CollectionInterval,
		MetricsGeneratorDisableCollection:                                           c.MetricsGenerator.DisableCollection,
		MetricsGeneratorGenerateNativeHistograms:                                    c.MetricsGenerator.GenerateNativeHistograms,
		MetricsGeneratorTraceIDLabelName:                                            c.MetricsGenerator.TraceIDLabelName,
		MetricsGeneratorRemoteWriteHeaders:                                          c.MetricsGenerator.RemoteWriteHeaders,
		MetricsGeneratorForwarderQueueSize:                                          c.MetricsGenerator.Forwarder.QueueSize,
		MetricsGeneratorForwarderWorkers:                                            c.MetricsGenerator.Forwarder.Workers,
		MetricsGeneratorProcessorServiceGraphsHistogramBuckets:                      c.MetricsGenerator.Processor.ServiceGraphs.HistogramBuckets,
		MetricsGeneratorProcessorServiceGraphsDimensions:                            c.MetricsGenerator.Processor.ServiceGraphs.Dimensions,
		MetricsGeneratorProcessorServiceGraphsPeerAttributes:                        c.MetricsGenerator.Processor.ServiceGraphs.PeerAttributes,
		MetricsGeneratorProcessorServiceGraphsFilterPolicies:                        c.MetricsGenerator.Processor.ServiceGraphs.FilterPolicies,
		MetricsGeneratorProcessorServiceGraphsEnableClientServerPrefix:              c.MetricsGenerator.Processor.ServiceGraphs.EnableClientServerPrefix,
		MetricsGeneratorProcessorServiceGraphsEnableMessagingSystemLatencyHistogram: c.MetricsGenerator.Processor.ServiceGraphs.EnableMessagingSystemLatencyHistogram,
		MetricsGeneratorProcessorServiceGraphsEnableVirtualNodeLabel:                c.MetricsGenerator.Processor.ServiceGraphs.EnableVirtualNodeLabel,
		MetricsGeneratorProcessorServiceGraphsSpanMultiplierKey:                     c.MetricsGenerator.Processor.ServiceGraphs.SpanMultiplierKey,
		MetricsGeneratorProcessorServiceGraphsEnableTraceStateSpanMultiplier:        c.MetricsGenerator.Processor.ServiceGraphs.EnableTraceStateSpanMultiplier,
		MetricsGeneratorProcessorSpanMetricsHistogramBuckets:                        c.MetricsGenerator.Processor.SpanMetrics.HistogramBuckets,
		MetricsGeneratorProcessorSpanMetricsDimensions:                              c.MetricsGenerator.Processor.SpanMetrics.Dimensions,
		MetricsGeneratorProcessorSpanMetricsIntrinsicDimensions:                     c.MetricsGenerator.Processor.SpanMetrics.IntrinsicDimensions,
		MetricsGeneratorProcessorSpanMetricsFilterPolicies:                          c.MetricsGenerator.Processor.SpanMetrics.FilterPolicies,
		MetricsGeneratorProcessorSpanMetricsDimensionMappings:                       c.MetricsGenerator.Processor.SpanMetrics.DimensionMappings,
		MetricsGeneratorProcessorSpanMetricsEnableTargetInfo:                        c.MetricsGenerator.Processor.SpanMetrics.EnableTargetInfo,
		MetricsGeneratorProcessorSpanMetricsTargetInfoExcludedDimensions:            c.MetricsGenerator.Processor.SpanMetrics.TargetInfoExcludedDimensions,
		MetricsGeneratorProcessorSpanMetricsEnableInstanceLabel:                     c.MetricsGenerator.Processor.SpanMetrics.EnableInstanceLabel,
		MetricsGeneratorProcessorSpanMetricsSpanMultiplierKey:                       c.MetricsGenerator.Processor.SpanMetrics.SpanMultiplierKey,
		MetricsGeneratorProcessorSpanMetricsEnableTraceStateSpanMultiplier:          c.MetricsGenerator.Processor.SpanMetrics.EnableTraceStateSpanMultiplier,
		MetricsGeneratorProcessorHostInfoHostIdentifiers:                            c.MetricsGenerator.Processor.HostInfo.HostIdentifiers,
		MetricsGeneratorProcessorHostInfoMetricName:                                 c.MetricsGenerator.Processor.HostInfo.MetricName,
		MetricsGeneratorIngestionSlack:                                              c.MetricsGenerator.IngestionSlack,
		MetricsGeneratorNativeHistogramBucketFactor:                                 c.MetricsGenerator.NativeHistogramBucketFactor,
		MetricsGeneratorNativeHistogramMaxBucketNumber:                              c.MetricsGenerator.NativeHistogramMaxBucketNumber,
		MetricsGeneratorNativeHistogramMinResetDuration:                             c.MetricsGenerator.NativeHistogramMinResetDuration,
		MetricsGeneratorSpanNameSanitization:                                        c.MetricsGenerator.SpanNameSanitization,
		MetricsGeneratorMaxCardinalityPerLabel:                                      c.MetricsGenerator.MaxCardinalityPerLabel,

		BlockRetention:     c.Compaction.BlockRetention,
		CompactionWindow:   c.Compaction.CompactionWindow,
		CompactionDisabled: c.Compaction.CompactionDisabled,

		MaxBytesPerTagValuesQuery:     c.Read.MaxBytesPerTagValuesQuery,
		MaxBlocksPerTagValuesQuery:    c.Read.MaxBlocksPerTagValuesQuery,
		MaxConditionGroupsPerTagQuery: c.Read.MaxConditionGroupsPerTagQuery,
		MaxSearchDuration:             c.Read.MaxSearchDuration,
		MaxMetricsDuration:            c.Read.MaxMetricsDuration,
		UnsafeQueryHints:              c.Read.UnsafeQueryHints,
		LeftPadTraceIDs:               c.Read.LeftPadTraceIDs,
		MetricsSpanOnlyFetch:          c.Read.MetricsSpanOnlyFetch,

		MaxBytesPerTrace: c.Global.MaxBytesPerTrace,

		DedicatedColumns: c.Storage.DedicatedColumns,
		CostAttribution: CostAttributionOverrides{
			Dimensions:     c.CostAttribution.Dimensions,
			MaxCardinality: c.CostAttribution.MaxCardinality,
		},
		Extensions: maps.Clone(c.Extensions), // Copy extensions to avoid modifying the original
	}
}

// LegacyOverrides describe all the limits for users; can be used to describe global default
// limits via flags, or per-user limits via yaml config.
type LegacyOverrides struct {
	// Distributor enforced limits.
	IngestionRateStrategy      string         `yaml:"ingestion_rate_strategy" json:"ingestion_rate_strategy"`
	IngestionRateLimitBytes    int            `yaml:"ingestion_rate_limit_bytes" json:"ingestion_rate_limit_bytes"`
	IngestionBurstSizeBytes    int            `yaml:"ingestion_burst_size_bytes" json:"ingestion_burst_size_bytes"`
	IngestionTenantShardSize   int            `yaml:"ingestion_tenant_shard_size" json:"ingestion_tenant_shard_size"`
	IngestionMaxAttributeBytes int            `yaml:"ingestion_max_attribute_bytes" json:"ingestion_max_attribute_bytes"`
	IngestionArtificialDelay   *time.Duration `yaml:"ingestion_artificial_delay" json:"ingestion_artificial_delay"`
	IngestionRetryInfoEnabled  bool           `yaml:"ingestion_retry_info_enabled" json:"ingestion_retry_info_enabled"`

	// Ingester enforced limits.
	MaxLocalTracesPerUser  int `yaml:"max_traces_per_user" json:"max_traces_per_user"`
	MaxGlobalTracesPerUser int `yaml:"max_global_traces_per_user" json:"max_global_traces_per_user"`

	// Forwarders
	Forwarders []string `yaml:"forwarders" json:"forwarders"`

	// Metrics-generator config
	MetricsGeneratorRingSize                                                    int                              `yaml:"metrics_generator_ring_size" json:"metrics_generator_ring_size"`
	MetricsGeneratorProcessors                                                  listtomap.ListToMap              `yaml:"metrics_generator_processors" json:"metrics_generator_processors"`
	MetricsGeneratorMaxActiveSeries                                             uint32                           `yaml:"metrics_generator_max_active_series" json:"metrics_generator_max_active_series"`
	MetricsGeneratorMaxActiveEntities                                           uint32                           `yaml:"metrics_generator_max_active_entities" json:"metrics_generator_max_active_entities"`
	MetricsGeneratorCollectionInterval                                          time.Duration                    `yaml:"metrics_generator_collection_interval" json:"metrics_generator_collection_interval"`
	MetricsGeneratorDisableCollection                                           bool                             `yaml:"metrics_generator_disable_collection" json:"metrics_generator_disable_collection"`
	MetricsGeneratorGenerateNativeHistograms                                    histograms.HistogramMethod       `yaml:"metrics_generator_generate_native_histograms" json:"metrics_generator_generate_native_histograms"`
	MetricsGeneratorNativeHistogramBucketFactor                                 float64                          `yaml:"metrics_generator_native_histogram_bucket_factor,omitempty" json:"metrics_generator_native_histogram_bucket_factor,omitempty"`
	MetricsGeneratorNativeHistogramMaxBucketNumber                              uint32                           `yaml:"metrics_generator_native_histogram_max_bucket_number,omitempty" json:"metrics_generator_native_histogram_max_bucket_number,omitempty"`
	MetricsGeneratorNativeHistogramMinResetDuration                             time.Duration                    `yaml:"metrics_generator_native_histogram_min_reset_duration,omitempty" json:"native_histogram_min_reset_duration,omitempty"`
	MetricsGeneratorSpanNameSanitization                                        string                           `yaml:"metrics_generator_span_name_sanitization" json:"metrics_generator_span_name_sanitization"`
	MetricsGeneratorMaxCardinalityPerLabel                                      uint64                           `yaml:"metrics_generator_max_cardinality_per_label,omitempty" json:"metrics_generator_max_cardinality_per_label,omitempty"`
	MetricsGeneratorTraceIDLabelName                                            string                           `yaml:"metrics_generator_trace_id_label_name" json:"metrics_generator_trace_id_label_name"`
	MetricsGeneratorForwarderQueueSize                                          int                              `yaml:"metrics_generator_forwarder_queue_size" json:"metrics_generator_forwarder_queue_size"`
	MetricsGeneratorForwarderWorkers                                            int                              `yaml:"metrics_generator_forwarder_workers" json:"metrics_generator_forwarder_workers"`
	MetricsGeneratorRemoteWriteHeaders                                          RemoteWriteHeaders               `yaml:"metrics_generator_remote_write_headers,omitempty" json:"metrics_generator_remote_write_headers,omitempty"`
	MetricsGeneratorProcessorServiceGraphsHistogramBuckets                      []float64                        `yaml:"metrics_generator_processor_service_graphs_histogram_buckets" json:"metrics_generator_processor_service_graphs_histogram_buckets"`
	MetricsGeneratorProcessorServiceGraphsDimensions                            []string                         `yaml:"metrics_generator_processor_service_graphs_dimensions" json:"metrics_generator_processor_service_graphs_dimensions"`
	MetricsGeneratorProcessorServiceGraphsPeerAttributes                        []string                         `yaml:"metrics_generator_processor_service_graphs_peer_attributes" json:"metrics_generator_processor_service_graphs_peer_attributes"`
	MetricsGeneratorProcessorServiceGraphsFilterPolicies                        []filterconfig.FilterPolicy      `yaml:"metrics_generator_processor_service_graphs_filter_policies" json:"metrics_generator_processor_service_graphs_filter_policies"`
	MetricsGeneratorProcessorServiceGraphsEnableClientServerPrefix              *bool                            `yaml:"metrics_generator_processor_service_graphs_enable_client_server_prefix" json:"metrics_generator_processor_service_graphs_enable_client_server_prefix"`
	MetricsGeneratorProcessorServiceGraphsEnableMessagingSystemLatencyHistogram *bool                            `yaml:"metrics_generator_processor_service_graphs_enable_messaging_system_latency_histogram" json:"metrics_generator_processor_service_graphs_enable_messaging_system_latency_histogram"`
	MetricsGeneratorProcessorServiceGraphsEnableVirtualNodeLabel                *bool                            `yaml:"metrics_generator_processor_service_graphs_enable_virtual_node_label" json:"metrics_generator_processor_service_graphs_enable_virtual_node_label"`
	MetricsGeneratorProcessorServiceGraphsSpanMultiplierKey                     string                           `yaml:"metrics_generator_processor_service_graphs_span_multiplier_key" json:"metrics_generator_processor_service_graphs_span_multiplier_key"`
	MetricsGeneratorProcessorServiceGraphsEnableTraceStateSpanMultiplier        *bool                            `yaml:"metrics_generator_processor_service_graphs_enable_tracestate_span_multiplier" json:"metrics_generator_processor_service_graphs_enable_tracestate_span_multiplier"`
	MetricsGeneratorProcessorSpanMetricsHistogramBuckets                        []float64                        `yaml:"metrics_generator_processor_span_metrics_histogram_buckets" json:"metrics_generator_processor_span_metrics_histogram_buckets"`
	MetricsGeneratorProcessorSpanMetricsDimensions                              []string                         `yaml:"metrics_generator_processor_span_metrics_dimensions" json:"metrics_generator_processor_span_metrics_dimensions"`
	MetricsGeneratorProcessorSpanMetricsIntrinsicDimensions                     map[string]bool                  `yaml:"metrics_generator_processor_span_metrics_intrinsic_dimensions" json:"metrics_generator_processor_span_metrics_intrinsic_dimensions"`
	MetricsGeneratorProcessorSpanMetricsFilterPolicies                          []filterconfig.FilterPolicy      `yaml:"metrics_generator_processor_span_metrics_filter_policies" json:"metrics_generator_processor_span_metrics_filter_policies"`
	MetricsGeneratorProcessorSpanMetricsDimensionMappings                       []sharedconfig.DimensionMappings `yaml:"metrics_generator_processor_span_metrics_dimension_mappings" json:"metrics_generator_processor_span_metrics_dimension_mapings"`
	MetricsGeneratorProcessorSpanMetricsEnableTargetInfo                        *bool                            `yaml:"metrics_generator_processor_span_metrics_enable_target_info" json:"metrics_generator_processor_span_metrics_enable_target_info"`
	MetricsGeneratorProcessorSpanMetricsTargetInfoExcludedDimensions            []string                         `yaml:"metrics_generator_processor_span_metrics_target_info_excluded_dimensions" json:"metrics_generator_processor_span_metrics_target_info_excluded_dimensions"`
	MetricsGeneratorProcessorSpanMetricsEnableInstanceLabel                     *bool                            `yaml:"metrics_generator_processor_span_metrics_enable_instance_label" json:"metrics_generator_processor_span_metrics_enable_instance_label"`
	MetricsGeneratorProcessorSpanMetricsSpanMultiplierKey                       string                           `yaml:"metrics_generator_processor_span_metrics_span_multiplier_key" json:"metrics_generator_processor_span_metrics_span_multiplier_key"`
	MetricsGeneratorProcessorSpanMetricsEnableTraceStateSpanMultiplier          *bool                            `yaml:"metrics_generator_processor_span_metrics_enable_tracestate_span_multiplier" json:"metrics_generator_processor_span_metrics_enable_tracestate_span_multiplier"`
	MetricsGeneratorProcessorHostInfoHostIdentifiers                            []string                         `yaml:"metrics_generator_processor_host_info_host_identifiers" json:"metrics_generator_processor_host_info_host_identifiers"`
	MetricsGeneratorProcessorHostInfoMetricName                                 string                           `yaml:"metrics_generator_processor_host_info_metric_name" json:"metrics_generator_processor_host_info_metric_name"`
	MetricsGeneratorIngestionSlack                                              time.Duration                    `yaml:"metrics_generator_ingestion_time_range_slack" json:"metrics_generator_ingestion_time_range_slack,omitempty"`

	// Backend-worker/scheduler enforced limits.
	BlockRetention     model.Duration `yaml:"block_retention" json:"block_retention"`
	CompactionDisabled bool           `yaml:"compaction_disabled" json:"compaction_disabled"`
	CompactionWindow   model.Duration `yaml:"compaction_window" json:"compaction_window"`

	// Querier and Ingester enforced limits.
	MaxBytesPerTagValuesQuery     int `yaml:"max_bytes_per_tag_values_query" json:"max_bytes_per_tag_values_query"`
	MaxBlocksPerTagValuesQuery    int `yaml:"max_blocks_per_tag_values_query" json:"max_blocks_per_tag_values_query"`
	MaxConditionGroupsPerTagQuery int `yaml:"max_condition_groups_per_tag_query" json:"max_condition_groups_per_tag_query"`

	// QueryFrontend enforced limits
	MaxSearchDuration    model.Duration `yaml:"max_search_duration" json:"max_search_duration"`
	MaxMetricsDuration   model.Duration `yaml:"max_metrics_duration" json:"max_metrics_duration"`
	UnsafeQueryHints     bool           `yaml:"unsafe_query_hints" json:"unsafe_query_hints"`
	LeftPadTraceIDs      bool           `yaml:"left_pad_trace_ids" json:"left_pad_trace_ids"`
	MetricsSpanOnlyFetch *bool          `yaml:"metrics_spanonly_fetch,omitempty" json:"metrics_spanonly_fetch,omitempty"`

	// MaxBytesPerTrace is enforced in the Ingester, Compactor, Querier (Search). It
	//  is not used when doing a trace by id lookup.
	MaxBytesPerTrace int `yaml:"max_bytes_per_trace" json:"max_bytes_per_trace"`

	CostAttribution CostAttributionOverrides `yaml:"cost_attribution" json:"cost_attribution"`

	// tempodb limits
	DedicatedColumns backend.DedicatedColumns `yaml:"parquet_dedicated_columns" json:"parquet_dedicated_columns"`

	// Extensions mirrors Overrides.Extensions: typed instances keyed by nested Key() after unmarshal.
	Extensions map[string]any `yaml:",inline" json:"-"`
}

// knownLegacyOverridesJSONFields returns the JSON key names declared on LegacyOverrides
var knownLegacyOverridesJSONFields = sync.OnceValue(func() map[string]struct{} {
	return fieldNamesFor(LegacyOverrides{}, "json")
})

// knownLegacyOverridesYAMLFields returns the YAML key names declared on LegacyOverrides.
var knownLegacyOverridesYAMLFields = sync.OnceValue(func() map[string]struct{} {
	return fieldNamesFor(LegacyOverrides{}, "yaml")
})

// isKnownLegacyOverridesField reports whether key matches any YAML or JSON field
func isKnownLegacyOverridesField(key string) bool {
	_, inJSON := knownLegacyOverridesJSONFields()[key]
	_, inYAML := knownLegacyOverridesYAMLFields()[key]
	return inJSON || inYAML
}

func (l *LegacyOverrides) UnmarshalJSON(data []byte) error {
	type plain LegacyOverrides
	if err := json.Unmarshal(data, (*plain)(l)); err != nil {
		return err
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	for key := range knownLegacyOverridesJSONFields() {
		delete(raw, key)
	}
	if len(raw) == 0 {
		// No extension keys in this payload; clear any stale Extensions from a prior decode.
		l.Extensions = nil
		return nil
	}

	l.Extensions = make(map[string]any, len(raw))
	for k, v := range raw {
		var val any
		if err := json.Unmarshal(v, &val); err != nil {
			return err
		}
		l.Extensions[k] = val
	}

	// Convert flat legacy keys to typed instances; processExtensions must not be called here
	// as it expects nested keys (e.g. "my_ext"), not flat legacy keys (e.g. "my_ext_field").
	return processLegacyExtensions(l)
}

func (l LegacyOverrides) MarshalJSON() ([]byte, error) {
	type plain LegacyOverrides
	data, err := json.Marshal(plain(l))
	if err != nil {
		return nil, err
	}
	if len(l.Extensions) == 0 {
		return data, nil
	}

	var m map[string]json.RawMessage
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	// Flatten typed extensions to legacy flat keys; copy other entries as-is.
	for k, v := range l.Extensions {
		ext, ok := v.(Extension)
		if !ok {
			return nil, fmt.Errorf("extension %q is not an Extension", k)
		}

		for fk, fv := range ext.ToLegacy() {
			if _, exists := m[fk]; exists {
				continue // known fields take precedence
			}
			b, err := json.Marshal(fv)
			if err != nil {
				return nil, err
			}
			m[fk] = b
		}
	}
	return json.Marshal(m)
}

func (l *LegacyOverrides) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type plain LegacyOverrides
	if err := unmarshal((*plain)(l)); err != nil {
		return err
	}
	// Convert registered extension flat keys to typed instances, matching Overrides.Extensions.
	return processLegacyExtensions(l)
}

func (l LegacyOverrides) MarshalYAML() (interface{}, error) {
	type plain LegacyOverrides
	if len(l.Extensions) == 0 {
		return plain(l), nil
	}
	// Flatten typed extensions to legacy flat keys so the YAML wire format matches the legacy shape.
	knownLegacy := knownLegacyOverridesYAMLFields()
	flat, err := flattenExtensionEntries(l.Extensions)
	if err != nil {
		return nil, err
	}

	filtered := make(map[string]any, len(flat))
	for k, v := range flat {
		if _, known := knownLegacy[k]; known {
			continue
		}
		filtered[k] = v
	}
	cp := l
	cp.Extensions = filtered
	return plain(cp), nil
}

func (l *LegacyOverrides) toNewLimits() *Overrides {
	return &Overrides{
		Ingestion: IngestionOverrides{
			RateStrategy:           l.IngestionRateStrategy,
			RateLimitBytes:         l.IngestionRateLimitBytes,
			BurstSizeBytes:         l.IngestionBurstSizeBytes,
			MaxLocalTracesPerUser:  l.MaxLocalTracesPerUser,
			MaxGlobalTracesPerUser: l.MaxGlobalTracesPerUser,
			TenantShardSize:        l.IngestionTenantShardSize,
			MaxAttributeBytes:      l.IngestionMaxAttributeBytes,
			ArtificialDelay:        l.IngestionArtificialDelay,
			RetryInfoEnabled:       l.IngestionRetryInfoEnabled,
		},
		Read: ReadOverrides{
			MaxBytesPerTagValuesQuery:     l.MaxBytesPerTagValuesQuery,
			MaxBlocksPerTagValuesQuery:    l.MaxBlocksPerTagValuesQuery,
			MaxConditionGroupsPerTagQuery: l.MaxConditionGroupsPerTagQuery,
			MaxSearchDuration:             l.MaxSearchDuration,
			MaxMetricsDuration:            l.MaxMetricsDuration,
			UnsafeQueryHints:              l.UnsafeQueryHints,
			LeftPadTraceIDs:               l.LeftPadTraceIDs,
			MetricsSpanOnlyFetch:          l.MetricsSpanOnlyFetch,
		},
		Compaction: CompactionOverrides{
			BlockRetention:     l.BlockRetention,
			CompactionDisabled: l.CompactionDisabled,
			CompactionWindow:   l.CompactionWindow,
		},
		MetricsGenerator: MetricsGeneratorOverrides{
			RingSize:                 l.MetricsGeneratorRingSize,
			Processors:               l.MetricsGeneratorProcessors,
			MaxActiveSeries:          l.MetricsGeneratorMaxActiveSeries,
			MaxActiveEntities:        l.MetricsGeneratorMaxActiveEntities,
			CollectionInterval:       l.MetricsGeneratorCollectionInterval,
			DisableCollection:        l.MetricsGeneratorDisableCollection,
			TraceIDLabelName:         l.MetricsGeneratorTraceIDLabelName,
			IngestionSlack:           l.MetricsGeneratorIngestionSlack,
			RemoteWriteHeaders:       l.MetricsGeneratorRemoteWriteHeaders,
			GenerateNativeHistograms: l.MetricsGeneratorGenerateNativeHistograms,
			Forwarder: ForwarderOverrides{
				QueueSize: l.MetricsGeneratorForwarderQueueSize,
				Workers:   l.MetricsGeneratorForwarderWorkers,
			},
			Processor: ProcessorOverrides{
				ServiceGraphs: ServiceGraphsOverrides{
					HistogramBuckets:                      l.MetricsGeneratorProcessorServiceGraphsHistogramBuckets,
					Dimensions:                            l.MetricsGeneratorProcessorServiceGraphsDimensions,
					PeerAttributes:                        l.MetricsGeneratorProcessorServiceGraphsPeerAttributes,
					FilterPolicies:                        l.MetricsGeneratorProcessorServiceGraphsFilterPolicies,
					EnableClientServerPrefix:              l.MetricsGeneratorProcessorServiceGraphsEnableClientServerPrefix,
					EnableMessagingSystemLatencyHistogram: l.MetricsGeneratorProcessorServiceGraphsEnableMessagingSystemLatencyHistogram,
					EnableVirtualNodeLabel:                l.MetricsGeneratorProcessorServiceGraphsEnableVirtualNodeLabel,
					SpanMultiplierKey:                     l.MetricsGeneratorProcessorServiceGraphsSpanMultiplierKey,
					EnableTraceStateSpanMultiplier:        l.MetricsGeneratorProcessorServiceGraphsEnableTraceStateSpanMultiplier,
				},
				SpanMetrics: SpanMetricsOverrides{
					HistogramBuckets:               l.MetricsGeneratorProcessorSpanMetricsHistogramBuckets,
					Dimensions:                     l.MetricsGeneratorProcessorSpanMetricsDimensions,
					IntrinsicDimensions:            l.MetricsGeneratorProcessorSpanMetricsIntrinsicDimensions,
					FilterPolicies:                 l.MetricsGeneratorProcessorSpanMetricsFilterPolicies,
					DimensionMappings:              l.MetricsGeneratorProcessorSpanMetricsDimensionMappings,
					EnableTargetInfo:               l.MetricsGeneratorProcessorSpanMetricsEnableTargetInfo,
					TargetInfoExcludedDimensions:   l.MetricsGeneratorProcessorSpanMetricsTargetInfoExcludedDimensions,
					EnableInstanceLabel:            l.MetricsGeneratorProcessorSpanMetricsEnableInstanceLabel,
					SpanMultiplierKey:              l.MetricsGeneratorProcessorSpanMetricsSpanMultiplierKey,
					EnableTraceStateSpanMultiplier: l.MetricsGeneratorProcessorSpanMetricsEnableTraceStateSpanMultiplier,
				},
				HostInfo: HostInfoOverrides{
					HostIdentifiers: l.MetricsGeneratorProcessorHostInfoHostIdentifiers,
					MetricName:      l.MetricsGeneratorProcessorHostInfoMetricName,
				},
			},
			NativeHistogramBucketFactor:     l.MetricsGeneratorNativeHistogramBucketFactor,
			NativeHistogramMaxBucketNumber:  l.MetricsGeneratorNativeHistogramMaxBucketNumber,
			NativeHistogramMinResetDuration: l.MetricsGeneratorNativeHistogramMinResetDuration,
			SpanNameSanitization:            l.MetricsGeneratorSpanNameSanitization,
			MaxCardinalityPerLabel:          l.MetricsGeneratorMaxCardinalityPerLabel,
		},
		Forwarders: l.Forwarders,
		Global: GlobalOverrides{
			MaxBytesPerTrace: l.MaxBytesPerTrace,
		},
		Storage: StorageOverrides{
			DedicatedColumns: l.DedicatedColumns,
		},
		CostAttribution: CostAttributionOverrides{
			Dimensions:     l.CostAttribution.Dimensions,
			MaxCardinality: l.CostAttribution.MaxCardinality,
		},
		Extensions: maps.Clone(l.Extensions), // copy extensions to avoid modifying the original
	}
}

// perTenantLegacyOverrides represents the Overrides config file with the legacy representation
type perTenantLegacyOverrides struct {
	TenantLimits map[string]*LegacyOverrides `yaml:"overrides"`
}

// Convert to new format
func (l *perTenantLegacyOverrides) toNewOverrides() perTenantOverrides {
	overrides := perTenantOverrides{
		TenantLimits: make(map[string]*Overrides, len(l.TenantLimits)),
	}

	for tenantID, legacyLimits := range l.TenantLimits {
		overrides.TenantLimits[tenantID] = legacyLimits.toNewLimits()
	}

	return overrides
}
