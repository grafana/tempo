package main

import (
	"flag"
	"time"
)

// TempoInstance represents a single Tempo instance configuration
type TempoInstance struct {
	// Name is a friendly name for this instance
	Name string `yaml:"name"`
	// Endpoint is the base URL for this Tempo instance (e.g., "http://tempo-1:3200")
	Endpoint string `yaml:"endpoint"`
	// OrgID is the tenant ID to use for this instance (optional)
	OrgID string `yaml:"org_id,omitempty"`
	// Timeout is the request timeout for this instance
	Timeout time.Duration `yaml:"timeout,omitempty"`
	// Headers are additional headers to send with requests
	Headers map[string]string `yaml:"headers,omitempty"`
}

// Config holds the configuration for the federated querier
type Config struct {
	// Server configuration
	HTTPListenAddress string `yaml:"http_listen_address"`
	HTTPListenPort    int    `yaml:"http_listen_port"`

	// Tempo instances to query
	Instances []TempoInstance `yaml:"instances"`

	// Query settings
	QueryTimeout          time.Duration `yaml:"query_timeout"`
	MaxConcurrentQueries  int           `yaml:"max_concurrent_queries"`
	MaxBytesPerTrace      int           `yaml:"max_bytes_per_trace"`
	AllowPartialResponses bool          `yaml:"allow_partial_responses"`

}

// RegisterFlagsAndApplyDefaults registers flags and sets default values
func (cfg *Config) RegisterFlagsAndApplyDefaults(f *flag.FlagSet) {
	f.StringVar(&cfg.HTTPListenAddress, "server.http-listen-address", "0.0.0.0", "HTTP server listen address")
	f.IntVar(&cfg.HTTPListenPort, "server.http-listen-port", 3200, "HTTP server listen port")
	f.DurationVar(&cfg.QueryTimeout, "query.timeout", 30*time.Second, "Timeout for trace by ID queries")
	f.IntVar(&cfg.MaxConcurrentQueries, "query.max-concurrent", 20, "Maximum concurrent queries per request")
	f.IntVar(&cfg.MaxBytesPerTrace, "query.max-bytes-per-trace", 50*1024*1024, "Maximum bytes per trace (50MB default)")
	f.BoolVar(&cfg.AllowPartialResponses, "query.allow-partial-responses", true, "Allow partial responses if some instances fail")
}

// Validate validates the configuration
func (cfg *Config) Validate() error {
	if len(cfg.Instances) == 0 {
		return errNoInstances
	}

	for i, inst := range cfg.Instances {
		if inst.Endpoint == "" {
			return errInstanceEndpointRequired(i)
		}
		if inst.Name == "" {
			cfg.Instances[i].Name = inst.Endpoint
		}
		if inst.Timeout == 0 {
			cfg.Instances[i].Timeout = cfg.QueryTimeout
		}
	}

	return nil
}

// ExampleConfig returns an example configuration YAML
func ExampleConfig() string {
	return `# Federated Tempo Querier Configuration
http_listen_address: "0.0.0.0"
http_listen_port: 3200

# Tempo instances to federate
instances:
  - name: "tempo-region-1"
    endpoint: "http://tempo-1.example.com:3200"
    org_id: "tenant-1"
    timeout: 30s
  - name: "tempo-region-2"
    endpoint: "http://tempo-2.example.com:3200"
    org_id: "tenant-1"
    timeout: 30s
  - name: "tempo-region-3"
    endpoint: "http://tempo-3.example.com:3200"
    org_id: "tenant-1"
    timeout: 30s
  - name: "tempo-region-4"
    endpoint: "http://tempo-4.example.com:3200"
    org_id: "tenant-1"
    timeout: 30s

# Query settings
query_timeout: 30s
max_concurrent_queries: 20
max_bytes_per_trace: 52428800  # 50MB
allow_partial_responses: true
`
}
