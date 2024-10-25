package distributor

import (
	"flag"
	"time"

	"github.com/grafana/dskit/flagext"
	ring_client "github.com/grafana/dskit/ring/client"

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

	// disables write extension with inactive ingesters. Use this along with ingester.lifecycler.unregister_on_shutdown = true
	//  note that setting these two config values reduces tolerance to failures on rollout b/c there is always one guaranteed to be failing replica
	ExtendWrites bool `yaml:"extend_writes"`

	// configures the distributor to indicate to the client that it should retry resource exhausted errors after the
	// provided duration
	RetryAfterOnResourceExhausted time.Duration `yaml:"retry_after_on_resource_exhausted"`

	// For testing.
	factory ring_client.PoolAddrFunc `yaml:"-"`
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

	cfg.RetryAfterOnResourceExhausted = 0
	cfg.OverrideRingKey = distributorRingKey
	cfg.ExtendWrites = true

	f.BoolVar(&cfg.LogReceivedSpans.Enabled, util.PrefixConfig(prefix, "log-received-spans.enabled"), false, "Enable to log every received span to help debug ingestion or calculate span error distributions using the logs.")
	f.BoolVar(&cfg.LogReceivedSpans.IncludeAllAttributes, util.PrefixConfig(prefix, "log-received-spans.include-attributes"), false, "Enable to include span attributes in the logs.")
	f.BoolVar(&cfg.LogReceivedSpans.FilterByStatusError, util.PrefixConfig(prefix, "log-received-spans.filter-by-status-error"), false, "Enable to filter out spans without status error.")

	f.BoolVar(&cfg.LogDiscardedSpans.Enabled, util.PrefixConfig(prefix, "log-discarded-spans.enabled"), false, "Enable to log every discarded span to help debug ingestion or calculate span error distributions using the logs.")
	f.BoolVar(&cfg.LogDiscardedSpans.IncludeAllAttributes, util.PrefixConfig(prefix, "log-discarded-spans.include-attributes"), false, "Enable to include span attributes in the logs.")
	f.BoolVar(&cfg.LogDiscardedSpans.FilterByStatusError, util.PrefixConfig(prefix, "log-discarded-spans.filter-by-status-error"), false, "Enable to filter out spans without status error.")

	cfg.Usage.RegisterFlagsAndApplyDefaults(prefix, f)
}
