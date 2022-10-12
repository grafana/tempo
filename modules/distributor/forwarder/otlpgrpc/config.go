package otlpgrpc

import (
	"flag"

	"github.com/grafana/dskit/flagext"
	"github.com/pkg/errors"

	"github.com/grafana/tempo/pkg/util"
)

type Config struct {
	Endpoints flagext.StringSlice `yaml:"endpoints"`
	TLS       TLSConfig           `yaml:"tls"`
}

// RegisterFlagsAndApplyDefaults registers flags and applies defaults
func (cfg *Config) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	f.Var(&cfg.Endpoints, util.PrefixConfig(prefix, "endpoints"), "OTLP GRPC endpoints to which the tracing data will be sent.")

	cfg.TLS.RegisterFlagsAndApplyDefaults(util.PrefixConfig(prefix, "tls"), f)
}

func (cfg *Config) Validate() error {
	// TODO: Validate if endpoints are in form host:port?
	return cfg.TLS.Validate()
}

type TLSConfig struct {
	Insecure bool   `yaml:"insecure"`
	CertFile string `yaml:"cert_file"`
}

// RegisterFlagsAndApplyDefaults registers flags and applies defaults
func (cfg *TLSConfig) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	f.BoolVar(&cfg.Insecure, util.PrefixConfig(prefix, "insecure"), false, "")
	f.StringVar(&cfg.CertFile, util.PrefixConfig(prefix, "cert_file"), "", "Path to ")
}

func (cfg *TLSConfig) Validate() error {
	if cfg.Insecure {
		return nil
	}

	if cfg.CertFile == "" {
		return errors.New("cert_file is empty")
	}

	return nil
}
