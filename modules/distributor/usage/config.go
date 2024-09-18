package usage

import (
	"flag"
	"time"
)

const (
	defaultMaxCardinality = 1000
	defaultStaleDuration  = 15 * time.Minute
	defaultPurgePeriod    = time.Minute
)

type Config struct {
	Enabled        bool          `yaml:"enabled,omitempty" json:"enabled,omitempty"`
	MaxCardinality uint          `yaml:"max_cardinality,omitempty" json:"max_cardinality,omitempty"`
	StaleDuration  time.Duration `yaml:"stale_duration,omitempty" json:"stale_duration,omitempty"`
	PurgePeriod    time.Duration `yaml:"purge_period,omitempty" json:"purge_period,omitempty"`
}

func (c *Config) RegisterFlagsAndApplyDefaults(_ string, _ *flag.FlagSet) {
	c.Enabled = true
	c.MaxCardinality = defaultMaxCardinality
	c.StaleDuration = defaultStaleDuration
	c.PurgePeriod = defaultPurgePeriod
}
