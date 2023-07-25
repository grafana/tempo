package overrides

import (
	"time"

	"github.com/grafana/tempo/pkg/sharedconfig"
	filterconfig "github.com/grafana/tempo/pkg/spanfilter/config"
	"github.com/prometheus/common/model"
)

func (l *Limits) toLegacy() LegacyLimits {
	return LegacyLimits{
		IngestionRateStrategy:   l.Ingestion.RateStrategy,
		IngestionRateLimitBytes: l.Ingestion.RateLimitBytes,
		IngestionBurstSizeBytes: l.Ingestion.BurstSizeBytes,
		MaxLocalTracesPerUser:   l.Ingestion.MaxLocalTracesPerUser,
		MaxGlobalTracesPerUser:  l.Ingestion.MaxGlobalTracesPerUser,

		Forwarders: l.Forwarders,

		MetricsGeneratorRingSize:                                       l.MetricsGenerator.RingSize,
		MetricsGeneratorProcessors:                                     l.MetricsGenerator.Processors,
		MetricsGeneratorMaxActiveSeries:                                l.MetricsGenerator.MaxActiveSeries,
		MetricsGeneratorCollectionInterval:                             l.MetricsGenerator.CollectionInterval,
		MetricsGeneratorDisableCollection:                              l.MetricsGenerator.DisableCollection,
		MetricsGeneratorForwarderQueueSize:                             l.MetricsGenerator.Forwarder.QueueSize,
		MetricsGeneratorForwarderWorkers:                               l.MetricsGenerator.Forwarder.Workers,
		MetricsGeneratorProcessorServiceGraphsHistogramBuckets:         l.MetricsGenerator.Processor.ServiceGraphs.HistogramBuckets,
		MetricsGeneratorProcessorServiceGraphsDimensions:               l.MetricsGenerator.Processor.ServiceGraphs.Dimensions,
		MetricsGeneratorProcessorServiceGraphsPeerAttributes:           l.MetricsGenerator.Processor.ServiceGraphs.PeerAttributes,
		MetricsGeneratorProcessorServiceGraphsEnableClientServerPrefix: l.MetricsGenerator.Processor.ServiceGraphs.EnableClientServerPrefix,
		MetricsGeneratorProcessorSpanMetricsHistogramBuckets:           l.MetricsGenerator.Processor.SpanMetrics.HistogramBuckets,
		MetricsGeneratorProcessorSpanMetricsDimensions:                 l.MetricsGenerator.Processor.SpanMetrics.Dimensions,
		MetricsGeneratorProcessorSpanMetricsIntrinsicDimensions:        l.MetricsGenerator.Processor.SpanMetrics.IntrinsicDimensions,
		MetricsGeneratorProcessorSpanMetricsFilterPolicies:             l.MetricsGenerator.Processor.SpanMetrics.FilterPolicies,
		MetricsGeneratorProcessorSpanMetricsDimensionMappings:          l.MetricsGenerator.Processor.SpanMetrics.DimensionMappings,
		MetricsGeneratorProcessorSpanMetricsEnableTargetInfo:           l.MetricsGenerator.Processor.SpanMetrics.EnableTargetInfo,
		MetricsGeneratorProcessorLocalBlocksMaxLiveTraces:              l.MetricsGenerator.Processor.LocalBlocks.MaxLiveTraces,
		MetricsGeneratorProcessorLocalBlocksMaxBlockDuration:           l.MetricsGenerator.Processor.LocalBlocks.MaxBlockDuration,
		MetricsGeneratorProcessorLocalBlocksMaxBlockBytes:              l.MetricsGenerator.Processor.LocalBlocks.MaxBlockBytes,
		MetricsGeneratorProcessorLocalBlocksFlushCheckPeriod:           l.MetricsGenerator.Processor.LocalBlocks.FlushCheckPeriod,
		MetricsGeneratorProcessorLocalBlocksTraceIdlePeriod:            l.MetricsGenerator.Processor.LocalBlocks.TraceIdlePeriod,
		MetricsGeneratorProcessorLocalBlocksCompleteBlockTimeout:       l.MetricsGenerator.Processor.LocalBlocks.CompleteBlockTimeout,

		BlockRetention: l.Compaction.BlockRetention,

		MaxBytesPerTagValuesQuery:  l.Read.MaxBytesPerTagValuesQuery,
		MaxBlocksPerTagValuesQuery: l.Read.MaxBlocksPerTagValuesQuery,
		MaxSearchDuration:          l.Read.MaxSearchDuration,

		MaxBytesPerTrace: l.Global.MaxBytesPerTrace,
	}
}

// LegacyLimits describe all the limits for users; can be used to describe global default
// limits via flags, or per-user limits via yaml config.
type LegacyLimits struct {
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
	MetricsGeneratorRingSize                                       int                              `yaml:"metrics_generator_ring_size" json:"metrics_generator_ring_size"`
	MetricsGeneratorProcessors                                     ListToMap                        `yaml:"metrics_generator_processors" json:"metrics_generator_processors"`
	MetricsGeneratorMaxActiveSeries                                uint32                           `yaml:"metrics_generator_max_active_series" json:"metrics_generator_max_active_series"`
	MetricsGeneratorCollectionInterval                             time.Duration                    `yaml:"metrics_generator_collection_interval" json:"metrics_generator_collection_interval"`
	MetricsGeneratorDisableCollection                              bool                             `yaml:"metrics_generator_disable_collection" json:"metrics_generator_disable_collection"`
	MetricsGeneratorForwarderQueueSize                             int                              `yaml:"metrics_generator_forwarder_queue_size" json:"metrics_generator_forwarder_queue_size"`
	MetricsGeneratorForwarderWorkers                               int                              `yaml:"metrics_generator_forwarder_workers" json:"metrics_generator_forwarder_workers"`
	MetricsGeneratorProcessorServiceGraphsHistogramBuckets         []float64                        `yaml:"metrics_generator_processor_service_graphs_histogram_buckets" json:"metrics_generator_processor_service_graphs_histogram_buckets"`
	MetricsGeneratorProcessorServiceGraphsDimensions               []string                         `yaml:"metrics_generator_processor_service_graphs_dimensions" json:"metrics_generator_processor_service_graphs_dimensions"`
	MetricsGeneratorProcessorServiceGraphsPeerAttributes           []string                         `yaml:"metrics_generator_processor_service_graphs_peer_attributes" json:"metrics_generator_processor_service_graphs_peer_attributes"`
	MetricsGeneratorProcessorServiceGraphsEnableClientServerPrefix bool                             `yaml:"metrics_generator_processor_service_graphs_enable_client_server_prefix" json:"metrics_generator_processor_service_graphs_enable_client_server_prefix"`
	MetricsGeneratorProcessorSpanMetricsHistogramBuckets           []float64                        `yaml:"metrics_generator_processor_span_metrics_histogram_buckets" json:"metrics_generator_processor_span_metrics_histogram_buckets"`
	MetricsGeneratorProcessorSpanMetricsDimensions                 []string                         `yaml:"metrics_generator_processor_span_metrics_dimensions" json:"metrics_generator_processor_span_metrics_dimensions"`
	MetricsGeneratorProcessorSpanMetricsIntrinsicDimensions        map[string]bool                  `yaml:"metrics_generator_processor_span_metrics_intrinsic_dimensions" json:"metrics_generator_processor_span_metrics_intrinsic_dimensions"`
	MetricsGeneratorProcessorSpanMetricsFilterPolicies             []filterconfig.FilterPolicy      `yaml:"metrics_generator_processor_span_metrics_filter_policies" json:"metrics_generator_processor_span_metrics_filter_policies"`
	MetricsGeneratorProcessorSpanMetricsDimensionMappings          []sharedconfig.DimensionMappings `yaml:"metrics_generator_processor_span_metrics_dimension_mappings" json:"metrics_generator_processor_span_metrics_dimension_mapings"`
	MetricsGeneratorProcessorSpanMetricsEnableTargetInfo           bool                             `yaml:"metrics_generator_processor_span_metrics_enable_target_info" json:"metrics_generator_processor_span_metrics_enable_target_info"`
	MetricsGeneratorProcessorLocalBlocksMaxLiveTraces              uint64                           `yaml:"metrics_generator_processor_local_blocks_max_live_traces" json:"metrics_generator_processor_local_blocks_max_live_traces"`
	MetricsGeneratorProcessorLocalBlocksMaxBlockDuration           time.Duration                    `yaml:"metrics_generator_processor_local_blocks_max_block_duration" json:"metrics_generator_processor_local_blocks_max_block_duration"`
	MetricsGeneratorProcessorLocalBlocksMaxBlockBytes              uint64                           `yaml:"metrics_generator_processor_local_blocks_max_block_bytes" json:"metrics_generator_processor_local_blocks_max_block_bytes"`
	MetricsGeneratorProcessorLocalBlocksFlushCheckPeriod           time.Duration                    `yaml:"metrics_generator_processor_local_blocks_flush_check_period" json:"metrics_generator_processor_local_blocks_flush_check_period"`
	MetricsGeneratorProcessorLocalBlocksTraceIdlePeriod            time.Duration                    `yaml:"metrics_generator_processor_local_blocks_trace_idle_period" json:"metrics_generator_processor_local_blocks_trace_idle_period"`
	MetricsGeneratorProcessorLocalBlocksCompleteBlockTimeout       time.Duration                    `yaml:"metrics_generator_processor_local_blocks_complete_block_timeout" json:"metrics_generator_processor_local_blocks_complete_block_timeout"`

	// Compactor enforced limits.
	BlockRetention model.Duration `yaml:"block_retention" json:"block_retention"`

	// Querier and Ingester enforced limits.
	MaxBytesPerTagValuesQuery  int `yaml:"max_bytes_per_tag_values_query" json:"max_bytes_per_tag_values_query"`
	MaxBlocksPerTagValuesQuery int `yaml:"max_blocks_per_tag_values_query" json:"max_blocks_per_tag_values_query"`

	// QueryFrontend enforced limits
	MaxSearchDuration model.Duration `yaml:"max_search_duration" json:"max_search_duration"`

	// MaxBytesPerTrace is enforced in the Ingester, Compactor, Querier (Search) and Serverless (Search). It
	//  is not used when doing a trace by id lookup.
	MaxBytesPerTrace int `yaml:"max_bytes_per_trace" json:"max_bytes_per_trace"`
}

func (l *LegacyLimits) toNewLimits() Limits {
	return Limits{
		Ingestion: IngestionConfig{
			RateStrategy:           l.IngestionRateStrategy,
			RateLimitBytes:         l.IngestionRateLimitBytes,
			BurstSizeBytes:         l.IngestionBurstSizeBytes,
			MaxLocalTracesPerUser:  l.MaxLocalTracesPerUser,
			MaxGlobalTracesPerUser: l.MaxGlobalTracesPerUser,
		},
		Read: ReadConfig{
			MaxBytesPerTagValuesQuery:  l.MaxBytesPerTagValuesQuery,
			MaxBlocksPerTagValuesQuery: l.MaxBlocksPerTagValuesQuery,
			MaxSearchDuration:          l.MaxSearchDuration,
		},
		Compaction: CompactionConfig{
			BlockRetention: l.BlockRetention,
		},
		MetricsGenerator: MetricsGeneratorConfig{
			RingSize:           l.MetricsGeneratorRingSize,
			Processors:         l.MetricsGeneratorProcessors,
			MaxActiveSeries:    l.MetricsGeneratorMaxActiveSeries,
			CollectionInterval: l.MetricsGeneratorCollectionInterval,
			DisableCollection:  l.MetricsGeneratorDisableCollection,
			Forwarder: ForwarderConfig{
				QueueSize: l.MetricsGeneratorForwarderQueueSize,
				Workers:   l.MetricsGeneratorForwarderWorkers,
			},
			Processor: ProcessorConfig{
				ServiceGraphs: ServiceGraphsConfig{
					HistogramBuckets:         l.MetricsGeneratorProcessorServiceGraphsHistogramBuckets,
					Dimensions:               l.MetricsGeneratorProcessorServiceGraphsDimensions,
					PeerAttributes:           l.MetricsGeneratorProcessorServiceGraphsPeerAttributes,
					EnableClientServerPrefix: l.MetricsGeneratorProcessorServiceGraphsEnableClientServerPrefix,
				},
				SpanMetrics: SpanMetricsConfig{
					HistogramBuckets:    l.MetricsGeneratorProcessorSpanMetricsHistogramBuckets,
					Dimensions:          l.MetricsGeneratorProcessorSpanMetricsDimensions,
					IntrinsicDimensions: l.MetricsGeneratorProcessorSpanMetricsIntrinsicDimensions,
					FilterPolicies:      l.MetricsGeneratorProcessorSpanMetricsFilterPolicies,
					DimensionMappings:   l.MetricsGeneratorProcessorSpanMetricsDimensionMappings,
					EnableTargetInfo:    l.MetricsGeneratorProcessorSpanMetricsEnableTargetInfo,
				},
				LocalBlocks: LocalBlocksConfig{
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
		Global: GlobalLimitsConfig{
			MaxBytesPerTrace: l.MaxBytesPerTrace,
		},
	}
}

// perTenantLegacyOverrides represents the Overrides config file with the legacy representation
type perTenantLegacyOverrides struct {
	TenantLimits map[string]*LegacyLimits `yaml:"overrides"`
}

// Convert to new format
func (l *perTenantLegacyOverrides) toNewOverrides() perTenantOverrides {
	overrides := perTenantOverrides{
		TenantLimits: make(map[string]*Limits, len(l.TenantLimits)),
	}

	for tenantID, legacyLimits := range l.TenantLimits {
		limits := legacyLimits.toNewLimits()
		overrides.TenantLimits[tenantID] = &limits
	}

	return overrides
}
