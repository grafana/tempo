package hostinfo

import (
	"flag"
	"time"
)

type Config struct {
	// HostIdentifiers defines the list of resource attributes used to derive
	// a unique `grafana.host.id` value. In most cases, this should be [ "host.id" ]
	HostIdentifiers []string      `yaml:"host_identifiers"`
	StaleDuration   time.Duration `yaml:"stale_duration"`
	// MetricName defines the name of the metric that will be generated
	MetricName string `yaml:"metric_name"`
}

func (cfg *Config) RegisterFlagsAndApplyDefaults(string, *flag.FlagSet) {
	cfg.HostIdentifiers = []string{"k8s.node.name", "host.id"}
	cfg.StaleDuration = 2 * time.Minute
	cfg.MetricName = hostInfoMetric
}
