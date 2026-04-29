package distributor

import (
	"flag"
	"time"

	"github.com/grafana/dskit/flagext"
	"github.com/grafana/tempo/pkg/ingest"

	"github.com/grafana/tempo/modules/distributor/forwarder"
	"github.com/grafana/tempo/modules/distributor/usage"
	"github.com/grafana/tempo/pkg/util"
)

var defaultReceivers = map[string]interface{}{
	"jaeger": map[string]interface{}{
		"protocols": map[string]interface{}{
			"grpc":        nil,
			"thrift_http": nil,
		},
	},
	"otlp": map[string]interface{}{
		"protocols": map[string]interface{}{
			"grpc": nil,
		},
	},
}

// Config for a Distributor.
type Config struct {
	// Distributors ring
	DistributorRing RingConfig `yaml:"ring,omitempty"`
	// receivers map for shim.
	//  This receivers node is equivalent in format to the receiver node in the
	//  otel collector: https://github.com/open-telemetry/opentelemetry-collector/tree/main/receiver
	Receivers           map[string]interface{}    `yaml:"receivers"`
	OverrideRingKey     string                    `yaml:"override_ring_key"`
	LogReceivedSpans    LogSpansConfig            `yaml:"log_received_spans,omitempty"`
	LogDiscardedSpans   LogSpansConfig            `yaml:"log_discarded_spans,omitempty"`
	MetricReceivedSpans MetricReceivedSpansConfig `yaml:"metric_received_spans,omitempty"`
	Forwarders          forwarder.ConfigList      `yaml:"forwarders"`
	Usage               usage.Config              `yaml:"usage,omitempty"`

	KafkaConfig ingest.KafkaConfig `yaml:"kafka_config"`

	// Internal routing toggle set by app wiring (not user-configurable).
	PushSpansToKafka bool `yaml:"-"`

	// configures the distributor to indicate to the client that it should retry resource exhausted errors after the
	// provided duration
	RetryAfterOnResourceExhausted time.Duration `yaml:"retry_after_on_resource_exhausted"`

	// TracePushMiddlewares are hooks called when a trace push request is received.
	// Middleware errors are logged but don't fail the push (fail open behavior).
	TracePushMiddlewares []TracePushMiddleware `yaml:"-"`

	MaxAttributeBytes int `yaml:"max_attribute_bytes"`

	// ArtificialDelay is an optional duration to introduce a delay for artificial processing in the distributor.
	ArtificialDelay time.Duration `yaml:"artificial_delay,omitempty"`
}

type LogSpansConfig struct {
	Enabled              bool `yaml:"enabled"`
	IncludeAllAttributes bool `yaml:"include_all_attributes"`
	FilterByStatusError  bool `yaml:"filter_by_status_error"`
}

type MetricReceivedSpansConfig struct {
	Enabled  bool `yaml:"enabled"`
	RootOnly bool `yaml:"root_only"`
}

// RegisterFlagsAndApplyDefaults registers flags and applies defaults
func (cfg *Config) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	flagext.DefaultValues(&cfg.DistributorRing)
	cfg.DistributorRing.KVStore.Store = "memberlist"
	cfg.DistributorRing.HeartbeatTimeout = 5 * time.Minute

	// enable RetryInfo hints by default with 5s RetryAfter
	cfg.RetryAfterOnResourceExhausted = 5 * time.Second
	cfg.OverrideRingKey = distributorRingKey

	cfg.MaxAttributeBytes = 2048 // 2KB

	f.BoolVar(&cfg.LogReceivedSpans.Enabled, util.PrefixConfig(prefix, "log-received-spans.enabled"), false, "Enable to log every received span to help debug ingestion or calculate span error distributions using the logs.")
	f.BoolVar(&cfg.LogReceivedSpans.IncludeAllAttributes, util.PrefixConfig(prefix, "log-received-spans.include-attributes"), false, "Enable to include span attributes in the logs.")
	f.BoolVar(&cfg.LogReceivedSpans.FilterByStatusError, util.PrefixConfig(prefix, "log-received-spans.filter-by-status-error"), false, "Enable to filter out spans without status error.")

	f.BoolVar(&cfg.LogDiscardedSpans.Enabled, util.PrefixConfig(prefix, "log-discarded-spans.enabled"), false, "Enable to log every discarded span to help debug ingestion or calculate span error distributions using the logs.")
	f.BoolVar(&cfg.LogDiscardedSpans.IncludeAllAttributes, util.PrefixConfig(prefix, "log-discarded-spans.include-attributes"), false, "Enable to include span attributes in the logs.")
	f.BoolVar(&cfg.LogDiscardedSpans.FilterByStatusError, util.PrefixConfig(prefix, "log-discarded-spans.filter-by-status-error"), false, "Enable to filter out spans without status error.")

	cfg.Usage.RegisterFlagsAndApplyDefaults(prefix, f)
}

func (cfg *Config) Validate() error {
	if cfg.PushSpansToKafka {
		if err := cfg.KafkaConfig.Validate(); err != nil {
			return err
		}
	}

	return nil
}
