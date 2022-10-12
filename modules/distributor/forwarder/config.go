package forwarder

import (
	"flag"
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/grafana/tempo/modules/distributor/forwarder/otlpgrpc"
	"github.com/grafana/tempo/pkg/util"
)

const (
	OTLPGRPCBackend = "otlpgrpc"
)

type Config struct {
	Name     string          `yaml:"name"`
	Backend  string          `yaml:"backend"`
	OTLPGRPC otlpgrpc.Config `yaml:"otlpgrpc"`
}

// RegisterFlagsAndApplyDefaults registers flags and applies defaults
func (cfg *Config) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	f.StringVar(&cfg.Name, util.PrefixConfig(prefix, "name"), "", "Reference name that can be used in the overrides.yaml config.")
	f.StringVar(&cfg.Backend, util.PrefixConfig(prefix, "backend"), "", fmt.Sprintf("Forwarding backend to use. Supported backends: %s", strings.Join([]string{OTLPGRPCBackend}, ",")))

	cfg.OTLPGRPC.RegisterFlagsAndApplyDefaults(util.PrefixConfig(prefix, "otlpgrpc"), f)
}

func (cfg *Config) Validate() error {
	if cfg.Name == "" {
		return errors.New("name is empty")
	}

	switch cfg.Backend {
	case OTLPGRPCBackend:
		return cfg.OTLPGRPC.Validate()
	default:
	}

	return fmt.Errorf("%s backend is not supported", cfg.Backend)
}

type ConfigList []Config

// RegisterFlagsAndApplyDefaults registers flags and applies defaults
func (cfgs ConfigList) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	for _, cfg := range cfgs {
		cfg.RegisterFlagsAndApplyDefaults(prefix, f)
	}
}

func (cfgs ConfigList) Validate() error {
	for i, cfg := range cfgs {
		if err := cfg.Validate(); err != nil {
			return fmt.Errorf("failed to validate config at index=%d: %w", i, err)
		}
	}

	return nil
}
