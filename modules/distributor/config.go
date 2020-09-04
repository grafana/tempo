package distributor

import (
	"flag"

	cortex_distributor "github.com/cortexproject/cortex/pkg/distributor"
	ring_client "github.com/cortexproject/cortex/pkg/ring/client"
	"github.com/cortexproject/cortex/pkg/util/flagext"
	"go.opentelemetry.io/collector/receiver/jaegerreceiver"
	"go.opentelemetry.io/collector/receiver/otlpreceiver"
)

// Config for a Distributor.
type Config struct {
	// Distributors ring
	DistributorRing cortex_distributor.RingConfig `yaml:"ring,omitempty"`

	// receivers map for shim.
	//  This receivers node is equivalent in format to the receiver node in the
	//  otel collector: https://github.com/open-telemetry/opentelemetry-collector/tree/master/receiver
	Receivers        map[string]interface{} `yaml:"receivers"`
	DefaultReceivers map[string]interface{} `yaml:"-"`

	// For testing.
	factory func(addr string) (ring_client.PoolClient, error) `yaml:"-"`
}

const (
	defaultGRPCBindEndpoint = "0.0.0.0:14250"
)

// RegisterFlagsAndApplyDefaults registers flags and applies defaults
func (cfg *Config) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	// create a default config that listens for otlp and jaeger
	cfg.DefaultReceivers = map[string]interface{}{
		"jaeger": jaegerreceiver.NewFactory().CreateDefaultConfig(),
		"otlp":   otlpreceiver.NewFactory().CreateDefaultConfig(),
	}

	flagext.DefaultValues(&cfg.DistributorRing)
}
