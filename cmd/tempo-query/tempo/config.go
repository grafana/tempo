package tempo

import (
	"github.com/grafana/dskit/crypto/tls"
	"github.com/jaegertracing/jaeger/plugin/storage/grpc/shared"
	"github.com/spf13/viper"
)

// Config holds the configuration for redbull.
type Config struct {
	Backend               string           `yaml:"backend"`
	TLSEnabled            bool             `yaml:"tls_enabled" category:"advanced"`
	TLS                   tls.ClientConfig `yaml:",inline"`
	TenantHeaderKey       string           `yaml:"tenant_header_key"`
	QueryServicesDuration string           `yaml:"services_query_duration"`
}

// InitFromViper initializes the options struct with values from Viper
func (c *Config) InitFromViper(v *viper.Viper) {
	c.Backend = v.GetString("backend")
	c.TLSEnabled = v.GetBool("tls_enabled")
	c.TLS.CertPath = v.GetString("tls_cert_path")
	c.TLS.KeyPath = v.GetString("tls_key_path")
	c.TLS.CAPath = v.GetString("tls_ca_path")
	c.TLS.ServerName = v.GetString("tls_server_name")
	c.TLS.InsecureSkipVerify = v.GetBool("tls_insecure_skip_verify")
	c.TLS.CipherSuites = v.GetString("tls_cipher_suites")
	c.TLS.MinVersion = v.GetString("tls_min_version")
	c.QueryServicesDuration = v.GetString("services_query_duration")

	tenantHeader := v.GetString("tenant_header_key")
	if tenantHeader == "" {
		tenantHeader = shared.BearerTokenKey
	}
	c.TenantHeaderKey = tenantHeader
}
