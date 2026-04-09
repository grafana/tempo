package generator

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"slices"
	"time"

	"github.com/grafana/tempo/modules/generator/processor/hostinfo"
	"github.com/grafana/tempo/modules/generator/processor/servicegraphs"
	"github.com/grafana/tempo/modules/generator/processor/spanmetrics"
	"github.com/grafana/tempo/modules/generator/registry"
	"github.com/grafana/tempo/modules/generator/storage"
	"github.com/grafana/tempo/modules/generator/validation"
	"github.com/grafana/tempo/pkg/ingest"
	"github.com/grafana/tempo/pkg/ring"
	"go.uber.org/multierr"
)

const (
	// generatorRingKey is the default key under which we store the metric-generator's ring in the KVStore.
	generatorRingKey = "metrics-generator"

	ConsumerGroup = "metrics-generator"

	// codecPushBytes refers to the codec used for decoding tempopb.PushBytesRequest
	codecPushBytes = "push-bytes"
	// codecOTLP refers to the codec used for decoding ptrace.Traces
	codecOTLP = "otlp"
)

type LimiterType string

type RingMode string

const (
	LimiterTypeSeries LimiterType = "series"
	LimiterTypeEntity LimiterType = "entity"

	RingModePartition RingMode = "partition"
	RingModeGenerator RingMode = "generator"
)

var validCodecs = []string{codecPushBytes, codecOTLP}

// Config for a generator.
type Config struct {
	Ring      ring.Config     `yaml:"ring"`
	Processor ProcessorConfig `yaml:"processor"`
	Registry  registry.Config `yaml:"registry"`
	Storage   storage.Config  `yaml:"storage"`
	// MetricsIngestionSlack is the max amount of time passed since a span's end time
	// for the span to be considered in metrics generation
	MetricsIngestionSlack time.Duration `yaml:"metrics_ingestion_time_range_slack"`
	OverrideRingKey       string        `yaml:"override_ring_key"`
	RingMode              RingMode      `yaml:"ring_mode"`

	// Codec controls which decoder to use for data consumed from Kafka.
	Codec string `yaml:"codec"`

	// ConsumeFromKafka controls whether the generator should consume spans from Kafka.
	// This is wired by deployment model in app init and not user configurable.
	ConsumeFromKafka bool `yaml:"-"`

	// LimiterType configures the type of limiter to use.
	// Defaults to "series". Available options are "series" and "entity".
	LimiterType LimiterType `yaml:"limiter_type"`

	// This config is dynamically injected because defined outside the generator config.
	Ingest            ingest.Config `yaml:"-"`
	IngestConcurrency uint          `yaml:"ingest_concurrency"`
	InstanceID        string        `yaml:"instance_id" doc:"default=<hostname>" category:"advanced"`

	// LeaveConsumerGroupOnShutdown, when true, sends LeaveGroup to the Kafka coordinator
	// on shutdown so partitions are reassigned immediately rather than waiting for session
	// timeout (~3 min). Defaults to false so operators can control rollout behaviour
	// explicitly; set to true when deploying as a Deployment (random pod names) to avoid
	// the session-timeout delay on partition reassignment. Requires Kafka ingest enabled.
	LeaveConsumerGroupOnShutdown bool `yaml:"leave_consumer_group_on_shutdown" category:"advanced"`
}

// RegisterFlagsAndApplyDefaults registers the flags.
func (cfg *Config) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	cfg.Ring.RegisterFlagsAndApplyDefaults(prefix, f)
	cfg.Processor.RegisterFlagsAndApplyDefaults(prefix, f)
	cfg.Registry.RegisterFlagsAndApplyDefaults(prefix, f)
	cfg.Storage.RegisterFlagsAndApplyDefaults(prefix, f)
	cfg.Ingest.RegisterFlagsAndApplyDefaults(prefix, f)
	cfg.IngestConcurrency = 16

	// setting default for max span age before discarding to 30s
	cfg.MetricsIngestionSlack = 30 * time.Second
	cfg.OverrideRingKey = generatorRingKey
	cfg.RingMode = RingModePartition
	cfg.Codec = codecPushBytes
	cfg.LimiterType = LimiterTypeSeries

	hostname, err := os.Hostname()
	if err != nil {
		fmt.Printf("failed to get hostname: %v", err)
		os.Exit(1)
	}
	f.StringVar(&cfg.InstanceID, prefix+".instance-id", hostname, "Instance id.")
	f.BoolVar(&cfg.LeaveConsumerGroupOnShutdown, prefix+".leave-consumer-group-on-shutdown", false, "If true, send LeaveGroup to Kafka on shutdown for immediate partition reassignment. Default false; set to true for Deployment rollouts where pod names change on each restart.")
}

func (cfg *Config) Validate() error {
	if cfg.ConsumeFromKafka {
		if err := cfg.Ingest.Validate(); err != nil {
			return err
		}
	}

	if cfg.IngestConcurrency == 0 {
		return errors.New("ingest concurrency must be greater than zero")
	}

	if err := cfg.Processor.Validate(); err != nil {
		return err
	}

	if err := cfg.Storage.Validate(); err != nil {
		return err
	}

	if !slices.Contains(validCodecs, cfg.Codec) {
		return fmt.Errorf("invalid codec: %s, valid choices are %s", cfg.Codec, validCodecs)
	}

	switch cfg.RingMode {
	case RingModePartition, RingModeGenerator:
	default:
		return fmt.Errorf("invalid ring mode: %s, valid values are %s and %s", cfg.RingMode, RingModePartition, RingModeGenerator)
	}

	if cfg.ConsumeFromKafka && cfg.RingMode == RingModeGenerator && cfg.Ingest.Kafka.ConsumerGroup == "" {
		return errors.New("ingest.kafka.consumer_group must be configured when metrics-generator ring mode is generator")
	}

	switch cfg.LimiterType {
	case LimiterTypeSeries, LimiterTypeEntity:
	default:
		return fmt.Errorf("invalid limiter type: %s, valid values are %s and %s", cfg.LimiterType, LimiterTypeSeries, LimiterTypeEntity)
	}

	return nil
}

type ProcessorConfig struct {
	ServiceGraphs servicegraphs.Config `yaml:"service_graphs"`
	SpanMetrics   spanmetrics.Config   `yaml:"span_metrics"`
	HostInfo      hostinfo.Config      `yaml:"host_info"`
}

func (cfg *ProcessorConfig) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	cfg.ServiceGraphs.RegisterFlagsAndApplyDefaults(prefix, f)
	cfg.SpanMetrics.RegisterFlagsAndApplyDefaults(prefix, f)
	cfg.HostInfo.RegisterFlagsAndApplyDefaults(prefix, f)
}

func (cfg *ProcessorConfig) Validate() error {
	var errs []error
	if err := validation.ValidateHostInfoHostIdentifiers(cfg.HostInfo.HostIdentifiers); err != nil {
		errs = append(errs, err)
	}
	if err := validation.ValidateHostInfoMetricName(cfg.HostInfo.MetricName); err != nil {
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
	if filterPolicies := o.MetricsGeneratorProcessorServiceGraphsFilterPolicies(userID); filterPolicies != nil {
		copyCfg.ServiceGraphs.FilterPolicies = filterPolicies
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

	if histograms := o.MetricsGeneratorGenerateNativeHistograms(userID); histograms != "" {
		copyCfg.ServiceGraphs.HistogramOverride = registry.HistogramModeToValue[string(histograms)]
		copyCfg.SpanMetrics.HistogramOverride = registry.HistogramModeToValue[string(histograms)]
	}

	if dimensionMappings := o.MetricsGeneratorProcessorSpanMetricsDimensionMappings(userID); dimensionMappings != nil {
		copyCfg.SpanMetrics.DimensionMappings = dimensionMappings
	}

	if enableTargetInfo, ok := o.MetricsGeneratorProcessorSpanMetricsEnableTargetInfo(userID); ok {
		copyCfg.SpanMetrics.EnableTargetInfo = enableTargetInfo
	}

	if targetInfoExcludedDimensions := o.MetricsGeneratorProcessorSpanMetricsTargetInfoExcludedDimensions(userID); targetInfoExcludedDimensions != nil {
		copyCfg.SpanMetrics.TargetInfoExcludedDimensions = targetInfoExcludedDimensions
	}

	if EnableInstanceLabel, ok := o.MetricsGeneratorProcessorSpanMetricsEnableInstanceLabel(userID); ok {
		copyCfg.SpanMetrics.EnableInstanceLabel = EnableInstanceLabel
	}

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

	if spanMultiplierKey := o.MetricsGeneratorProcessorServiceGraphsSpanMultiplierKey(userID); spanMultiplierKey != "" {
		copyCfg.ServiceGraphs.SpanMultiplierKey = spanMultiplierKey
	}

	if enableTraceStateSpanMultiplier, ok := o.MetricsGeneratorProcessorServiceGraphsEnableTraceStateSpanMultiplier(userID); ok {
		copyCfg.ServiceGraphs.EnableTraceStateSpanMultiplier = enableTraceStateSpanMultiplier
	}

	if spanMultiplierKey := o.MetricsGeneratorProcessorSpanMetricsSpanMultiplierKey(userID); spanMultiplierKey != "" {
		copyCfg.SpanMetrics.SpanMultiplierKey = spanMultiplierKey
	}

	if enableTraceStateSpanMultiplier, ok := o.MetricsGeneratorProcessorSpanMetricsEnableTraceStateSpanMultiplier(userID); ok {
		copyCfg.SpanMetrics.EnableTraceStateSpanMultiplier = enableTraceStateSpanMultiplier
	}

	copySubprocessors := make(map[spanmetrics.Subprocessor]bool)
	for sp, enabled := range cfg.SpanMetrics.Subprocessors {
		copySubprocessors[sp] = enabled
	}
	copyCfg.SpanMetrics.Subprocessors = copySubprocessors

	return copyCfg, nil
}
