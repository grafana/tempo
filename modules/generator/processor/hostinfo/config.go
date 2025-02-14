package hostinfo

import (
	"flag"
	"time"
)

type Config struct {
	// HostIdentifiers defines the list of resource attributes used to derive
	// a unique `grafana.host.id` value. In most cases, this should be [ "host.id" ]
	HostIdentifiers      []string      `yaml:"host_identifiers"`
	MetricsFlushInterval time.Duration `yaml:"metrics_flush_interval"`
}

func (cfg *Config) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	cfg.HostIdentifiers = []string{"host.id"}
	cfg.MetricsFlushInterval = 60 * time.Second
}
