package overrides

import (
	"time"

	"github.com/grafana/tempo/pkg/util/listtomap"

	"github.com/grafana/tempo/tempodb/backend"

	"github.com/grafana/tempo/pkg/sharedconfig"
	filterconfig "github.com/grafana/tempo/pkg/spanfilter/config"
	"github.com/prometheus/common/model"
)

func (c *Overrides) toLegacy() LegacyOverrides {
	return LegacyOverrides{
		IngestionRateStrategy:   c.Ingestion.RateStrategy,
		IngestionRateLimitBytes: c.Ingestion.RateLimitBytes,
		IngestionBurstSizeBytes: c.Ingestion.BurstSizeBytes,
		MaxLocalTracesPerUser:   c.Ingestion.MaxLocalTracesPerUser,
		MaxGlobalTracesPerUser:  c.Ingestion.MaxGlobalTracesPerUser,

		Forwarders: c.Forwarders,

		MetricsGeneratorRingSize:                                         c.MetricsGenerator.RingSize,
		MetricsGeneratorProcessors:                                       c.MetricsGenerator.Processors,
		MetricsGeneratorMaxActiveSeries:                                  c.MetricsGenerator.MaxActiveSeries,
		MetricsGeneratorCollectionInterval:                               c.MetricsGenerator.CollectionInterval,
		MetricsGeneratorDisableCollection:                                c.MetricsGenerator.DisableCollection,
		MetricsGeneratorTraceIDLabelName:                                 c.MetricsGenerator.TraceIDLabelName,
		MetricsGeneratorRemoteWriteHeaders:                               c.MetricsGenerator.RemoteWriteHeaders,
		MetricsGeneratorForwarderQueueSize:                               c.MetricsGenerator.Forwarder.QueueSize,
		MetricsGeneratorForwarderWorkers:                                 c.MetricsGenerator.Forwarder.Workers,
		MetricsGeneratorProcessorServiceGraphsHistogramBuckets:           c.MetricsGenerator.Processor.ServiceGraphs.HistogramBuckets,
		MetricsGeneratorProcessorServiceGraphsDimensions:                 c.MetricsGenerator.Processor.ServiceGraphs.Dimensions,
		MetricsGeneratorProcessorServiceGraphsPeerAttributes:             c.MetricsGenerator.Processor.ServiceGraphs.PeerAttributes,
		MetricsGeneratorProcessorServiceGraphsEnableClientServerPrefix:   c.MetricsGenerator.Processor.ServiceGraphs.EnableClientServerPrefix,
		MetricsGeneratorProcessorSpanMetricsHistogramBuckets:             c.MetricsGenerator.Processor.SpanMetrics.HistogramBuckets,
		MetricsGeneratorProcessorSpanMetricsDimensions:                   c.MetricsGenerator.Processor.SpanMetrics.Dimensions,
		MetricsGeneratorProcessorSpanMetricsIntrinsicDimensions:          c.MetricsGenerator.Processor.SpanMetrics.IntrinsicDimensions,
		MetricsGeneratorProcessorSpanMetricsFilterPolicies:               c.MetricsGenerator.Processor.SpanMetrics.FilterPolicies,
		MetricsGeneratorProcessorSpanMetricsDimensionMappings:            c.MetricsGenerator.Processor.SpanMetrics.DimensionMappings,
		MetricsGeneratorProcessorSpanMetricsEnableTargetInfo:             c.MetricsGenerator.Processor.SpanMetrics.EnableTargetInfo,
		MetricsGeneratorProcessorSpanMetricsTargetInfoExcludedDimensions: c.MetricsGenerator.Processor.SpanMetrics.TargetInfoExcludedDimensions,
		MetricsGeneratorProcessorLocalBlocksMaxLiveTraces:                c.MetricsGenerator.Processor.LocalBlocks.MaxLiveTraces,
		MetricsGeneratorProcessorLocalBlocksMaxBlockDuration:             c.MetricsGenerator.Processor.LocalBlocks.MaxBlockDuration,
		MetricsGeneratorProcessorLocalBlocksMaxBlockBytes:                c.MetricsGenerator.Processor.LocalBlocks.MaxBlockBytes,
		MetricsGeneratorProcessorLocalBlocksFlushCheckPeriod:             c.MetricsGenerator.Processor.LocalBlocks.FlushCheckPeriod,
		MetricsGeneratorProcessorLocalBlocksTraceIdlePeriod:              c.MetricsGenerator.Processor.LocalBlocks.TraceIdlePeriod,
		MetricsGeneratorProcessorLocalBlocksCompleteBlockTimeout:         c.MetricsGenerator.Processor.LocalBlocks.CompleteBlockTimeout,
		MetricsGeneratorIngestionSlack:                                   c.MetricsGenerator.IngestionSlack,

		BlockRetention:   c.Compaction.BlockRetention,
		CompactionWindow: c.Compaction.CompactionWindow,

		MaxBytesPerTagValuesQuery:  c.Read.MaxBytesPerTagValuesQuery,
		MaxBlocksPerTagValuesQuery: c.Read.MaxBlocksPerTagValuesQuery,
		MaxSearchDuration:          c.Read.MaxSearchDuration,

		MaxBytesPerTrace: c.Global.MaxBytesPerTrace,

		DedicatedColumns: c.Storage.DedicatedColumns,
	}
}

// LegacyOverrides describe all the limits for users; can be used to describe global default
// limits via flags, or per-user limits via yaml config.
type LegacyOverrides struct {
	// Distributor enforced limits.
	IngestionRateStrategy   string `yaml:"ingestion_rate_strategy" json:"ingestion_rate_strategy"`
	IngestionRateLimitBytes int    `yaml:"ingestion_rate_limit_bytes" json:"ingestion_rate_limit_bytes"`
	IngestionBurstSizeBytes int    `yaml:"ingestion_burst_size_bytes" json:"ingestion_burst_size_bytes"`

	// Ingester enforced limits.
	MaxLocalTracesPerUser  int `yaml:"max_traces_per_user" json:"max_traces_per_user"`
	MaxGlobalTracesPerUser int `yaml:"max_global_traces_per_user" json:"max_global_traces_per_user"`

	// Forwarders
	Forwarders []string `yaml:"forwarders" json:"forwarders"`

	// Metrics-generator config
	MetricsGeneratorRingSize                                         int                              `yaml:"metrics_generator_ring_size" json:"metrics_generator_ring_size"`
	MetricsGeneratorProcessors                                       listtomap.ListToMap              `yaml:"metrics_generator_processors" json:"metrics_generator_processors"`
	MetricsGeneratorMaxActiveSeries                                  uint32                           `yaml:"metrics_generator_max_active_series" json:"metrics_generator_max_active_series"`
	MetricsGeneratorCollectionInterval                               time.Duration                    `yaml:"metrics_generator_collection_interval" json:"metrics_generator_collection_interval"`
	MetricsGeneratorDisableCollection                                bool                             `yaml:"metrics_generator_disable_collection" json:"metrics_generator_disable_collection"`
	MetricsGeneratorTraceIDLabelName                                 string                           `yaml:"metrics_generator_trace_id_label_name" json:"metrics_generator_trace_id_label_name"`
	MetricsGeneratorForwarderQueueSize                               int                              `yaml:"metrics_generator_forwarder_queue_size" json:"metrics_generator_forwarder_queue_size"`
	MetricsGeneratorForwarderWorkers                                 int                              `yaml:"metrics_generator_forwarder_workers" json:"metrics_generator_forwarder_workers"`
	MetricsGeneratorRemoteWriteHeaders                               RemoteWriteHeaders               `yaml:"metrics_generator_remote_write_headers,omitempty" json:"metrics_generator_remote_write_headers,omitempty"`
	MetricsGeneratorProcessorServiceGraphsHistogramBuckets           []float64                        `yaml:"metrics_generator_processor_service_graphs_histogram_buckets" json:"metrics_generator_processor_service_graphs_histogram_buckets"`
	MetricsGeneratorProcessorServiceGraphsDimensions                 []string                         `yaml:"metrics_generator_processor_service_graphs_dimensions" json:"metrics_generator_processor_service_graphs_dimensions"`
	MetricsGeneratorProcessorServiceGraphsPeerAttributes             []string                         `yaml:"metrics_generator_processor_service_graphs_peer_attributes" json:"metrics_generator_processor_service_graphs_peer_attributes"`
	MetricsGeneratorProcessorServiceGraphsEnableClientServerPrefix   bool                             `yaml:"metrics_generator_processor_service_graphs_enable_client_server_prefix" json:"metrics_generator_processor_service_graphs_enable_client_server_prefix"`
	MetricsGeneratorProcessorSpanMetricsHistogramBuckets             []float64                        `yaml:"metrics_generator_processor_span_metrics_histogram_buckets" json:"metrics_generator_processor_span_metrics_histogram_buckets"`
	MetricsGeneratorProcessorSpanMetricsDimensions                   []string                         `yaml:"metrics_generator_processor_span_metrics_dimensions" json:"metrics_generator_processor_span_metrics_dimensions"`
	MetricsGeneratorProcessorSpanMetricsIntrinsicDimensions          map[string]bool                  `yaml:"metrics_generator_processor_span_metrics_intrinsic_dimensions" json:"metrics_generator_processor_span_metrics_intrinsic_dimensions"`
	MetricsGeneratorProcessorSpanMetricsFilterPolicies               []filterconfig.FilterPolicy      `yaml:"metrics_generator_processor_span_metrics_filter_policies" json:"metrics_generator_processor_span_metrics_filter_policies"`
	MetricsGeneratorProcessorSpanMetricsDimensionMappings            []sharedconfig.DimensionMappings `yaml:"metrics_generator_processor_span_metrics_dimension_mappings" json:"metrics_generator_processor_span_metrics_dimension_mapings"`
	MetricsGeneratorProcessorSpanMetricsEnableTargetInfo             bool                             `yaml:"metrics_generator_processor_span_metrics_enable_target_info" json:"metrics_generator_processor_span_metrics_enable_target_info"`
	MetricsGeneratorProcessorSpanMetricsTargetInfoExcludedDimensions []string                         `yaml:"metrics_generator_processor_span_metrics_target_info_excluded_dimensions" json:"metrics_generator_processor_span_metrics_target_info_excluded_dimensions"`
	MetricsGeneratorProcessorLocalBlocksMaxLiveTraces                uint64                           `yaml:"metrics_generator_processor_local_blocks_max_live_traces" json:"metrics_generator_processor_local_blocks_max_live_traces"`
	MetricsGeneratorProcessorLocalBlocksMaxBlockDuration             time.Duration                    `yaml:"metrics_generator_processor_local_blocks_max_block_duration" json:"metrics_generator_processor_local_blocks_max_block_duration"`
	MetricsGeneratorProcessorLocalBlocksMaxBlockBytes                uint64                           `yaml:"metrics_generator_processor_local_blocks_max_block_bytes" json:"metrics_generator_processor_local_blocks_max_block_bytes"`
	MetricsGeneratorProcessorLocalBlocksFlushCheckPeriod             time.Duration                    `yaml:"metrics_generator_processor_local_blocks_flush_check_period" json:"metrics_generator_processor_local_blocks_flush_check_period"`
	MetricsGeneratorProcessorLocalBlocksTraceIdlePeriod              time.Duration                    `yaml:"metrics_generator_processor_local_blocks_trace_idle_period" json:"metrics_generator_processor_local_blocks_trace_idle_period"`
	MetricsGeneratorProcessorLocalBlocksCompleteBlockTimeout         time.Duration                    `yaml:"metrics_generator_processor_local_blocks_complete_block_timeout" json:"metrics_generator_processor_local_blocks_complete_block_timeout"`
	MetricsGeneratorIngestionSlack                                   time.Duration                    `yaml:"metrics_generator_ingestion_time_range_slack" json:"metrics_generator_ingestion_time_range_slack"`

	// Compactor enforced limits.
	BlockRetention   model.Duration `yaml:"block_retention" json:"block_retention"`
	CompactionWindow model.Duration `yaml:"compaction_window" json:"compaction_window"`

	// Querier and Ingester enforced limits.
	MaxBytesPerTagValuesQuery  int `yaml:"max_bytes_per_tag_values_query" json:"max_bytes_per_tag_values_query"`
	MaxBlocksPerTagValuesQuery int `yaml:"max_blocks_per_tag_values_query" json:"max_blocks_per_tag_values_query"`

	// QueryFrontend enforced limits
	MaxSearchDuration model.Duration `yaml:"max_search_duration" json:"max_search_duration"`

	// MaxBytesPerTrace is enforced in the Ingester, Compactor, Querier (Search) and Serverless (Search). It
	//  is not used when doing a trace by id lookup.
	MaxBytesPerTrace int `yaml:"max_bytes_per_trace" json:"max_bytes_per_trace"`

	// tempodb limits
	DedicatedColumns backend.DedicatedColumns `yaml:"parquet_dedicated_columns" json:"parquet_dedicated_columns"`
}

func (l *LegacyOverrides) toNewLimits() Overrides {
	return Overrides{
		Ingestion: IngestionOverrides{
			RateStrategy:           l.IngestionRateStrategy,
			RateLimitBytes:         l.IngestionRateLimitBytes,
			BurstSizeBytes:         l.IngestionBurstSizeBytes,
			MaxLocalTracesPerUser:  l.MaxLocalTracesPerUser,
			MaxGlobalTracesPerUser: l.MaxGlobalTracesPerUser,
		},
		Read: ReadOverrides{
			MaxBytesPerTagValuesQuery:  l.MaxBytesPerTagValuesQuery,
			MaxBlocksPerTagValuesQuery: l.MaxBlocksPerTagValuesQuery,
			MaxSearchDuration:          l.MaxSearchDuration,
		},
		Compaction: CompactionOverrides{
			BlockRetention:   l.BlockRetention,
			CompactionWindow: l.CompactionWindow,
		},
		MetricsGenerator: MetricsGeneratorOverrides{
			RingSize:           l.MetricsGeneratorRingSize,
			Processors:         l.MetricsGeneratorProcessors,
			MaxActiveSeries:    l.MetricsGeneratorMaxActiveSeries,
			CollectionInterval: l.MetricsGeneratorCollectionInterval,
			DisableCollection:  l.MetricsGeneratorDisableCollection,
			TraceIDLabelName:   l.MetricsGeneratorTraceIDLabelName,
			IngestionSlack:     l.MetricsGeneratorIngestionSlack,
			RemoteWriteHeaders: l.MetricsGeneratorRemoteWriteHeaders,
			Forwarder: ForwarderOverrides{
				QueueSize: l.MetricsGeneratorForwarderQueueSize,
				Workers:   l.MetricsGeneratorForwarderWorkers,
			},
			Processor: ProcessorOverrides{
				ServiceGraphs: ServiceGraphsOverrides{
					HistogramBuckets:         l.MetricsGeneratorProcessorServiceGraphsHistogramBuckets,
					Dimensions:               l.MetricsGeneratorProcessorServiceGraphsDimensions,
					PeerAttributes:           l.MetricsGeneratorProcessorServiceGraphsPeerAttributes,
					EnableClientServerPrefix: l.MetricsGeneratorProcessorServiceGraphsEnableClientServerPrefix,
				},
				SpanMetrics: SpanMetricsOverrides{
					HistogramBuckets:             l.MetricsGeneratorProcessorSpanMetricsHistogramBuckets,
					Dimensions:                   l.MetricsGeneratorProcessorSpanMetricsDimensions,
					IntrinsicDimensions:          l.MetricsGeneratorProcessorSpanMetricsIntrinsicDimensions,
					FilterPolicies:               l.MetricsGeneratorProcessorSpanMetricsFilterPolicies,
					DimensionMappings:            l.MetricsGeneratorProcessorSpanMetricsDimensionMappings,
					EnableTargetInfo:             l.MetricsGeneratorProcessorSpanMetricsEnableTargetInfo,
					TargetInfoExcludedDimensions: l.MetricsGeneratorProcessorSpanMetricsTargetInfoExcludedDimensions,
				},
				LocalBlocks: LocalBlocksOverrides{
					MaxLiveTraces:        l.MetricsGeneratorProcessorLocalBlocksMaxLiveTraces,
					MaxBlockDuration:     l.MetricsGeneratorProcessorLocalBlocksMaxBlockDuration,
					MaxBlockBytes:        l.MetricsGeneratorProcessorLocalBlocksMaxBlockBytes,
					FlushCheckPeriod:     l.MetricsGeneratorProcessorLocalBlocksFlushCheckPeriod,
					TraceIdlePeriod:      l.MetricsGeneratorProcessorLocalBlocksTraceIdlePeriod,
					CompleteBlockTimeout: l.MetricsGeneratorProcessorLocalBlocksCompleteBlockTimeout,
				},
			},
		},
		Forwarders: l.Forwarders,
		Global: GlobalOverrides{
			MaxBytesPerTrace: l.MaxBytesPerTrace,
		},
		Storage: StorageOverrides{
			DedicatedColumns: l.DedicatedColumns,
		},
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
		limits := legacyLimits.toNewLimits()
		overrides.TenantLimits[tenantID] = &limits
	}

	return overrides
}
