package hostinfo

import (
	"errors"
	"flag"

	"github.com/prometheus/common/model"
)

const (
	defaultHostInfoMetric = "traces_host_info"
)

type Config struct {
	// HostIdentifiers defines the list of resource attributes used to derive
	// a unique `grafana.host.id` value. In most cases, this should be [ "host.id" ]
	HostIdentifiers []string `yaml:"host_identifiers"`
	// MetricName defines the name of the metric that will be generated
	MetricName string `yaml:"metric_name"`
	// Help defines the help text for the metric that is sent as metadata.
	Help string `yaml:"help"`
	// Unit defines the unit text for the metric that is sent as metadata.
	Unit string `yaml:"unit"`
}

func (cfg *Config) RegisterFlagsAndApplyDefaults(string, *flag.FlagSet) {
	cfg.HostIdentifiers = []string{"k8s.node.name", "host.id"}
	cfg.MetricName = defaultHostInfoMetric
}

func (cfg *Config) Validate() error {
	if len(cfg.HostIdentifiers) == 0 {
		return errors.New("at least one value must be provided in host_identifiers")
	}

	if !model.IsValidMetricName(model.LabelValue(cfg.MetricName)) {
		return errors.New("metric_name is invalid")
	}

	return nil
}
