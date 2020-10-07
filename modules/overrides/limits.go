package overrides

import (
	"flag"
	"time"
)

const (
	// Local ingestion rate strategy
	LocalIngestionRateStrategy = "local"

	// Global ingestion rate strategy
	GlobalIngestionRateStrategy = "global"
)

// Limits describe all the limits for users; can be used to describe global default
// limits via flags, or per-user limits via yaml config.
type Limits struct {
	// Distributor enforced limits.
	IngestionRateStrategy string `yaml:"ingestion_rate_strategy"`
	IngestionRateSpans    int    `yaml:"ingestion_rate_limit"`
	IngestionMaxBatchSize int    `yaml:"ingestion_max_batch_size"`

	// Ingester enforced limits.
	MaxLocalTracesPerUser  int `yaml:"max_traces_per_user"`
	MaxGlobalTracesPerUser int `yaml:"max_global_traces_per_user"`
	MaxSpansPerTrace       int `yaml:"max_spans_per_trace"`

	// Config for overrides, convenient if it goes here.
	PerTenantOverrideConfig string        `yaml:"per_tenant_override_config"`
	PerTenantOverridePeriod time.Duration `yaml:"per_tenant_override_period"`
}

// RegisterFlags adds the flags required to config this to the given FlagSet
func (l *Limits) RegisterFlags(f *flag.FlagSet) {
	// Distributor Limits
	f.StringVar(&l.IngestionRateStrategy, "distributor.rate-limit-strategy", "local", "Whether the various ingestion rate limits should be applied individually to each distributor instance (local), or evenly shared across the cluster (global).")
	f.IntVar(&l.IngestionRateSpans, "distributor.ingestion-rate-limit", 100000, "Per-user ingestion rate limit in spans per second.")
	f.IntVar(&l.IngestionMaxBatchSize, "distributor.ingestion-max-batch-size", 1000, "Per-user allowed ingestion max batch size (in number of spans).")

	// Ingester limits
	f.IntVar(&l.MaxLocalTracesPerUser, "ingester.max-traces-per-user", 10e3, "Maximum number of active traces per user, per ingester. 0 to disable.")
	f.IntVar(&l.MaxGlobalTracesPerUser, "ingester.max-global-traces-per-user", 0, "Maximum number of active traces per user, across the cluster. 0 to disable.")
	f.IntVar(&l.MaxSpansPerTrace, "ingester.max-spans-per-trace", 50e3, "Maximum number of spans per trace.  0 to disable.")

	f.StringVar(&l.PerTenantOverrideConfig, "limits.per-user-override-config", "", "File name of per-user overrides.")
	f.DurationVar(&l.PerTenantOverridePeriod, "limits.per-user-override-period", 10*time.Second, "Period with this to reload the overrides.")
}
