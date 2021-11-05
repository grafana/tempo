package distributor

import (
	"flag"
	"time"

	cortex_distributor "github.com/cortexproject/cortex/pkg/distributor"
	"github.com/grafana/dskit/flagext"
	"github.com/grafana/dskit/ring"
	ring_client "github.com/grafana/dskit/ring/client"
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
	DistributorRing cortex_distributor.RingConfig `yaml:"ring,omitempty"`
	// receivers map for shim.
	//  This receivers node is equivalent in format to the receiver node in the
	//  otel collector: https://github.com/open-telemetry/opentelemetry-collector/tree/main/receiver
	Receivers         map[string]interface{} `yaml:"receivers"`
	OverrideRingKey   string                 `yaml:"override_ring_key"`
	LogReceivedTraces bool                   `yaml:"log_received_traces"`

	// disables write extension with inactive ingesters. Use this along with ingester.lifecycler.unregister_on_shutdown = true
	//  note that setting these two config values reduces tolerance to failures on rollout b/c there is always one guaranteed to be failing replica
	ExtendWrites bool `yaml:"extend_writes"`

	SearchTagsDenyList []string `yaml:"search_tags_deny_list"`

	// For testing.
	factory func(addr string) (ring_client.PoolClient, error) `yaml:"-"`
}

// RegisterFlagsAndApplyDefaults registers flags and applies defaults
func (cfg *Config) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	flagext.DefaultValues(&cfg.DistributorRing)
	cfg.DistributorRing.KVStore.Store = "memberlist"
	cfg.DistributorRing.HeartbeatTimeout = 5 * time.Minute

	cfg.OverrideRingKey = ring.DistributorRingKey
	cfg.ExtendWrites = true

	f.BoolVar(&cfg.LogReceivedTraces, prefix+".log-received-traces", false, "Enable to log every received trace id to help debug ingestion.")
}
