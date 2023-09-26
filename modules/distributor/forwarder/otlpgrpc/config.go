package otlpgrpc

import (
	"errors"

	"github.com/grafana/dskit/flagext"
)

type Config struct {
	Endpoints flagext.StringSlice `yaml:"endpoints"`
	TLS       TLSConfig           `yaml:"tls"`
}

func (cfg *Config) Validate() error {
	// TODO: Validate if endpoints are in form host:port?
	return cfg.TLS.Validate()
}

type TLSConfig struct {
	Insecure bool   `yaml:"insecure"`
	CertFile string `yaml:"cert_file"`
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
