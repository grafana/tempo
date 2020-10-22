package distributor

import (
	"flag"
	"time"

	cortex_distributor "github.com/cortexproject/cortex/pkg/distributor"
	"github.com/cortexproject/cortex/pkg/ring"
	ring_client "github.com/cortexproject/cortex/pkg/ring/client"
	"github.com/cortexproject/cortex/pkg/util/flagext"
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
	//  otel collector: https://github.com/open-telemetry/opentelemetry-collector/tree/master/receiver
	Receivers       map[string]interface{} `yaml:"receivers"`
	OverrideRingKey string                 `yaml:"override_ring_key"`

	// For testing.
	factory func(addr string) (ring_client.PoolClient, error) `yaml:"-"`
}

// RegisterFlagsAndApplyDefaults registers flags and applies defaults
func (cfg *Config) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	flagext.DefaultValues(&cfg.DistributorRing)
	cfg.DistributorRing.KVStore.Store = "memberlist"
	cfg.DistributorRing.HeartbeatTimeout = 5 * time.Minute

	cfg.OverrideRingKey = ring.DistributorRingKey
}
