package generator

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"slices"
	"time"

	"github.com/grafana/tempo/modules/generator/processor/hostinfo"
	"github.com/grafana/tempo/modules/generator/processor/localblocks"
	"github.com/grafana/tempo/modules/generator/processor/servicegraphs"
	"github.com/grafana/tempo/modules/generator/processor/spanmetrics"
	"github.com/grafana/tempo/modules/generator/registry"
	"github.com/grafana/tempo/modules/generator/storage"
	"github.com/grafana/tempo/pkg/ingest"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/wal"
	"go.uber.org/multierr"
)

const (
	// generatorRingKey is the default key under which we store the metric-generator's ring in the KVStore.
	generatorRingKey = "metrics-generator"

	// ringNameForServer is the name of the ring used by the metrics-generator server.
	ringNameForServer = "metrics-generator"

	ConsumerGroup = "metrics-generator"

	// codecPushBytes refers to the codec used for decoding tempopb.PushBytesRequest
	codecPushBytes = "push-bytes"
	// codecOTLP refers to the codec used for decoding ptrace.Traces
	codecOTLP = "otlp"
)

var validCodecs = []string{codecPushBytes, codecOTLP}

// Config for a generator.
type Config struct {
	Ring           RingConfig      `yaml:"ring"`
	Processor      ProcessorConfig `yaml:"processor"`
	Registry       registry.Config `yaml:"registry"`
	Storage        storage.Config  `yaml:"storage"`
	TracesWAL      wal.Config      `yaml:"traces_storage"`
	TracesQueryWAL wal.Config      `yaml:"traces_query_storage"`
	// MetricsIngestionSlack is the max amount of time passed since a span's end time
	// for the span to be considered in metrics generation
	MetricsIngestionSlack time.Duration `yaml:"metrics_ingestion_time_range_slack"`
	QueryTimeout          time.Duration `yaml:"query_timeout"`
	OverrideRingKey       string        `yaml:"override_ring_key"`

	// Codec controls which decoder to use for data consumed from Kafka.
	Codec string `yaml:"codec"`
	// DisableLocalBlocks controls whether the local blocks processor should be run.
	// When this flag is enabled, the processor is never instantiated.
	DisableLocalBlocks bool `yaml:"disable_local_blocks"`
	// DisableGRPC controls whether to run a gRPC server with the metrics generator endpoints.
	DisableGRPC bool `yaml:"disable_grpc"`

	// This config is dynamically injected because defined outside the generator config.
	Ingest            ingest.Config `yaml:"-"`
	IngestConcurrency uint          `yaml:"ingest_concurrency"`
	InstanceID        string        `yaml:"instance_id" doc:"default=<hostname>" category:"advanced"`
}

// RegisterFlagsAndApplyDefaults registers the flags.
func (cfg *Config) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	cfg.Ring.RegisterFlagsAndApplyDefaults(prefix, f)
	cfg.Processor.RegisterFlagsAndApplyDefaults(prefix, f)
	cfg.Registry.RegisterFlagsAndApplyDefaults(prefix, f)
	cfg.Storage.RegisterFlagsAndApplyDefaults(prefix, f)
	cfg.TracesWAL.RegisterFlags(f)
	cfg.TracesWAL.Version = encoding.DefaultEncoding().Version()
	cfg.TracesQueryWAL.RegisterFlags(f)
	cfg.TracesQueryWAL.Version = encoding.DefaultEncoding().Version()
	cfg.Ingest.RegisterFlagsAndApplyDefaults(prefix, f)
	cfg.IngestConcurrency = 16

	// setting default for max span age before discarding to 30s
	cfg.MetricsIngestionSlack = 30 * time.Second
	cfg.QueryTimeout = 30 * time.Second
	cfg.OverrideRingKey = generatorRingKey
	cfg.Codec = codecPushBytes

	hostname, err := os.Hostname()
	if err != nil {
		fmt.Printf("failed to get hostname: %v", err)
		os.Exit(1)
	}
	f.StringVar(&cfg.InstanceID, prefix+".instance-id", hostname, "Instance id.")
}

func (cfg *Config) Validate() error {
	if err := cfg.Ingest.Validate(); err != nil {
		return err
	}

	if cfg.IngestConcurrency == 0 {
		return errors.New("ingest concurrency must be greater than zero")
	}

	if err := cfg.Processor.Validate(); err != nil {
		return err
	}

	// Only validate if being used
	if cfg.TracesWAL.Filepath != "" {
		if err := cfg.TracesWAL.Validate(); err != nil {
			return err
		}
	}
	if cfg.TracesQueryWAL.Filepath != "" {
		if err := cfg.TracesQueryWAL.Validate(); err != nil {
			return err
		}
	}

	if !slices.Contains(validCodecs, cfg.Codec) {
		return fmt.Errorf("invalid codec: %s, valid choices are %s", cfg.Codec, validCodecs)
	}

	return nil
}

type ProcessorConfig struct {
	ServiceGraphs servicegraphs.Config `yaml:"service_graphs"`
	SpanMetrics   spanmetrics.Config   `yaml:"span_metrics"`
	LocalBlocks   localblocks.Config   `yaml:"local_blocks"`
	HostInfo      hostinfo.Config      `yaml:"host_info"`
}

func (cfg *ProcessorConfig) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	cfg.ServiceGraphs.RegisterFlagsAndApplyDefaults(prefix, f)
	cfg.SpanMetrics.RegisterFlagsAndApplyDefaults(prefix, f)
	cfg.LocalBlocks.RegisterFlagsAndApplyDefaults(prefix, f)
	cfg.HostInfo.RegisterFlagsAndApplyDefaults(prefix, f)
}

func (cfg *ProcessorConfig) Validate() error {
	var errs []error
	if err := cfg.LocalBlocks.Validate(); err != nil {
		errs = append(errs, err)
	}
	if err := cfg.HostInfo.Validate(); err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return multierr.Combine(errs...)
	}
	return nil
}

// copyWithOverrides creates a copy of the config using values set in the overrides.
func (cfg *ProcessorConfig) copyWithOverrides(o metricsGeneratorOverrides, userID string) (ProcessorConfig, error) {
	copyCfg := *cfg

	if buckets := o.MetricsGeneratorProcessorServiceGraphsHistogramBuckets(userID); buckets != nil {
		copyCfg.ServiceGraphs.HistogramBuckets = buckets
	}
	if dimensions := o.MetricsGeneratorProcessorServiceGraphsDimensions(userID); dimensions != nil {
		copyCfg.ServiceGraphs.Dimensions = dimensions
	}
	if peerAttrs := o.MetricsGeneratorProcessorServiceGraphsPeerAttributes(userID); peerAttrs != nil {
		copyCfg.ServiceGraphs.PeerAttributes = peerAttrs
	}
	if buckets := o.MetricsGeneratorProcessorSpanMetricsHistogramBuckets(userID); buckets != nil {
		copyCfg.SpanMetrics.HistogramBuckets = buckets
	}
	if dimensions := o.MetricsGeneratorProcessorSpanMetricsDimensions(userID); dimensions != nil {
		copyCfg.SpanMetrics.Dimensions = dimensions
	}
	if dimensions := o.MetricsGeneratorProcessorSpanMetricsIntrinsicDimensions(userID); dimensions != nil {
		err := copyCfg.SpanMetrics.IntrinsicDimensions.ApplyFromMap(dimensions)
		if err != nil {
			return ProcessorConfig{}, fmt.Errorf("fail to apply overrides: %w", err)
		}
	}
	if filterPolicies := o.MetricsGeneratorProcessorSpanMetricsFilterPolicies(userID); filterPolicies != nil {
		copyCfg.SpanMetrics.FilterPolicies = filterPolicies
	}

	if max := o.MetricsGeneratorProcessorLocalBlocksMaxLiveTraces(userID); max > 0 {
		copyCfg.LocalBlocks.MaxLiveTraces = max
	}

	if max := o.MetricsGeneratorProcessorLocalBlocksMaxBlockDuration(userID); max > 0 {
		copyCfg.LocalBlocks.MaxBlockDuration = max
	}

	if max := o.MetricsGeneratorProcessorLocalBlocksMaxBlockBytes(userID); max > 0 {
		copyCfg.LocalBlocks.MaxBlockBytes = max
	}

	if period := o.MetricsGeneratorProcessorLocalBlocksFlushCheckPeriod(userID); period > 0 {
		copyCfg.LocalBlocks.FlushCheckPeriod = period
	}

	if period := o.MetricsGeneratorProcessorLocalBlocksTraceIdlePeriod(userID); period > 0 {
		copyCfg.LocalBlocks.TraceIdlePeriod = period
	}

	if timeout := o.MetricsGeneratorProcessorLocalBlocksCompleteBlockTimeout(userID); timeout > 0 {
		copyCfg.LocalBlocks.CompleteBlockTimeout = timeout
	}

	if histograms := o.MetricsGeneratorGenerateNativeHistograms(userID); histograms != "" {
		copyCfg.ServiceGraphs.HistogramOverride = registry.HistogramModeToValue[string(histograms)]
		copyCfg.SpanMetrics.HistogramOverride = registry.HistogramModeToValue[string(histograms)]
	}

	copyCfg.SpanMetrics.DimensionMappings = o.MetricsGeneratorProcessorSpanMetricsDimensionMappings(userID)

	if enableTargetInfo, ok := o.MetricsGeneratorProcessorSpanMetricsEnableTargetInfo(userID); ok {
		copyCfg.SpanMetrics.EnableTargetInfo = enableTargetInfo
	}

	copyCfg.SpanMetrics.TargetInfoExcludedDimensions = o.MetricsGeneratorProcessorSpanMetricsTargetInfoExcludedDimensions(userID)

	if enableClientServerPrefix := o.MetricsGeneratorProcessorServiceGraphsEnableClientServerPrefix(userID); enableClientServerPrefix {
		copyCfg.ServiceGraphs.EnableClientServerPrefix = enableClientServerPrefix
	}

	if enableMessagingSystemLatencyHistogram, ok := o.MetricsGeneratorProcessorServiceGraphsEnableMessagingSystemLatencyHistogram(userID); ok {
		copyCfg.ServiceGraphs.EnableMessagingSystemLatencyHistogram = enableMessagingSystemLatencyHistogram
	}

	if enableVirtualNodeLabel, ok := o.MetricsGeneratorProcessorServiceGraphsEnableVirtualNodeLabel(userID); ok {
		copyCfg.ServiceGraphs.EnableVirtualNodeLabel = enableVirtualNodeLabel
	}

	if hostIdentifiers := o.MetricsGeneratorProcessorHostInfoHostIdentifiers(userID); hostIdentifiers != nil {
		copyCfg.HostInfo.HostIdentifiers = o.MetricsGeneratorProcessorHostInfoHostIdentifiers(userID)
	}

	if hostInfoMetricName := o.MetricsGeneratorProcessorHostInfoMetricName(userID); hostInfoMetricName != "" {
		copyCfg.HostInfo.MetricName = o.MetricsGeneratorProcessorHostInfoMetricName(userID)
	}

	copySubprocessors := make(map[spanmetrics.Subprocessor]bool)
	for sp, enabled := range cfg.SpanMetrics.Subprocessors {
		copySubprocessors[sp] = enabled
	}
	copyCfg.SpanMetrics.Subprocessors = copySubprocessors

	return copyCfg, nil
}
