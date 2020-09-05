package distributor

import (
	"flag"

	cortex_distributor "github.com/cortexproject/cortex/pkg/distributor"
	ring_client "github.com/cortexproject/cortex/pkg/ring/client"
	"github.com/cortexproject/cortex/pkg/util/flagext"
	"go.opentelemetry.io/collector/receiver/jaegerreceiver"
	"go.opentelemetry.io/collector/receiver/otlpreceiver"
)

var defaultReceivers map[string]interface{}

// Config for a Distributor.
type Config struct {
	// Distributors ring
	DistributorRing cortex_distributor.RingConfig `yaml:"ring,omitempty"`

	// receivers map for shim.
	//  This receivers node is equivalent in format to the receiver node in the
	//  otel collector: https://github.com/open-telemetry/opentelemetry-collector/tree/master/receiver
	Receivers map[string]interface{} `yaml:"receivers"`

	// For testing.
	factory func(addr string) (ring_client.PoolClient, error) `yaml:"-"`
}

// RegisterFlagsAndApplyDefaults registers flags and applies defaults
func (cfg *Config) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	flagext.DefaultValues(&cfg.DistributorRing)

	defaultReceivers = map[string]interface{}{
		"jaeger": jaegerreceiver.NewFactory().CreateDefaultConfig(),
		"otlp":   otlpreceiver.NewFactory().CreateDefaultConfig(),
	}
}
