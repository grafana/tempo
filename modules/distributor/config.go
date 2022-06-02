package distributor

import (
	"flag"
	"time"

	"github.com/grafana/dskit/flagext"
	ring_client "github.com/grafana/dskit/ring/client"
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
	Receivers         map[string]interface{}  `yaml:"receivers"`
	OverrideRingKey   string                  `yaml:"override_ring_key"`
	LogReceivedTraces LogReceivedTracesConfig `yaml:"log_received_traces,omitempty"`

	// disables write extension with inactive ingesters. Use this along with ingester.lifecycler.unregister_on_shutdown = true
	//  note that setting these two config values reduces tolerance to failures on rollout b/c there is always one guaranteed to be failing replica
	ExtendWrites bool `yaml:"extend_writes"`

	SearchTagsDenyList []string `yaml:"search_tags_deny_list"`

	// For testing.
	factory func(addr string) (ring_client.PoolClient, error) `yaml:"-"`
}

type LogReceivedTracesConfig struct {
	Enabled             bool `yaml:"enabled"`
	IncludeAttributes   bool `yaml:"include_attributes"`
	FilterByStatusError bool `yaml:"filter_by_status_error"`
}

// RegisterFlagsAndApplyDefaults registers flags and applies defaults
func (cfg *Config) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	flagext.DefaultValues(&cfg.DistributorRing)
	cfg.DistributorRing.KVStore.Store = "memberlist"
	cfg.DistributorRing.HeartbeatTimeout = 5 * time.Minute

	cfg.OverrideRingKey = distributorRingKey
	cfg.ExtendWrites = true

	f.BoolVar(&cfg.LogReceivedTraces.Enabled, util.PrefixConfig(prefix, "log-received-traces.enabled"), false, "Enable to log every received span to help debug ingestion or calculate span error distributions using the logs.")
	f.BoolVar(&cfg.LogReceivedTraces.IncludeAttributes, util.PrefixConfig(prefix, " log-received-traces.include-attributes"), false, "Enable to include span attributes in the logs.")
	f.BoolVar(&cfg.LogReceivedTraces.FilterByStatusError, util.PrefixConfig(prefix, "log-received-traces.filter-by-status-error"), false, "Enable to filter out spans without status error.")
}
