package distributor

import (
	"flag"

	cortex_distributor "github.com/cortexproject/cortex/pkg/distributor"
	"google.golang.org/grpc/health/grpc_health_v1"
)

// Config for a Distributor.
type Config struct {
	// Distributors ring
	DistributorRing cortex_distributor.RingConfig `yaml:"ring,omitempty"`

	// receivers map for shim.
	Receivers map[string]interface{} `yaml:"receivers"`

	// For testing.
	factory func(addr string) (grpc_health_v1.HealthClient, error) `yaml:"-"`
}

// RegisterFlags registers the flags.
func (cfg *Config) RegisterFlags(f *flag.FlagSet) {
	cfg.DistributorRing.RegisterFlags(f)
}
