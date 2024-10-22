package usage

import (
	"flag"
	"time"
)

const (
	defaultMaxCardinality = uint64(10000)
	defaultStaleDuration  = 15 * time.Minute
	defaultPurgePeriod    = time.Minute
)

type PerTrackerConfig struct {
	Enabled        bool          `yaml:"enabled,omitempty" json:"enabled,omitempty"`
	MaxCardinality uint64        `yaml:"max_cardinality,omitempty" json:"max_cardinality,omitempty"`
	StaleDuration  time.Duration `yaml:"stale_duration,omitempty" json:"stale_duration,omitempty"`
}

type Config struct {
	CostAttribution PerTrackerConfig `yaml:"cost_attribution,omitempty" json:"cost_attribution,omitempty"`
}

func (c *Config) RegisterFlagsAndApplyDefaults(_ string, _ *flag.FlagSet) {
	c.CostAttribution = PerTrackerConfig{
		Enabled:        false,
		MaxCardinality: defaultMaxCardinality,
		StaleDuration:  defaultStaleDuration,
	}
}
