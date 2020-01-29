package querier

import (
	"flag"
	"time"
)

// Config for a querier.
type Config struct {
	QueryTimeout    time.Duration `yaml:"query_timeout"`
	ExtraQueryDelay time.Duration `yaml:"extra_query_delay,omitempty"`
}

// RegisterFlags register flags.
func (cfg *Config) RegisterFlags(f *flag.FlagSet) {
	f.DurationVar(&cfg.QueryTimeout, "querier.query_timeout", 1*time.Minute, "Timeout when querying backends (ingesters or storage) during the execution of a query request")
	f.DurationVar(&cfg.ExtraQueryDelay, "distributor.extra-query-delay", 0, "Time to wait before sending more than the minimum successful query requests.")
}
