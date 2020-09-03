package validation

import (
	"flag"
	"io"
	"time"

	"gopkg.in/yaml.v2"
)

const (
	// Local ingestion rate strategy
	LocalIngestionRateStrategy = "local"

	// Global ingestion rate strategy
	GlobalIngestionRateStrategy = "global"

	bytesInMB = 1048576
)

// OverridesConfig represents the overrides config file
type OverridesConfig struct {
	TenantLimits map[string]*Limits `yaml:"overrides"`
}

// LoadOverridesConfig is of type runtimeconfig.Loader
func LoadOverridesConfig(r io.Reader) (interface{}, error) {
	var overrides = &OverridesConfig{}

	decoder := yaml.NewDecoder(r)
	decoder.SetStrict(true)
	if err := decoder.Decode(&overrides); err != nil {
		return nil, err
	}

	return overrides, nil
}

// Limits describe all the limits for users; can be used to describe global default
// limits via flags, or per-user limits via yaml config.
type Limits struct {
	// Distributor enforced limits.
	IngestionRateStrategy string `yaml:"ingestion_rate_strategy"`
	IngestionRate         int    `yaml:"ingestion_rate"`
	IngestionBurstSize    int    `yaml:"ingestion_burst_size"`

	// Ingester enforced limits.
	MaxLocalTracesPerUser  int `yaml:"max_traces_per_user"`
	MaxGlobalTracesPerUser int `yaml:"max_global_traces_per_user"`

	// Config for overrides, convenient if it goes here.
	PerTenantOverrideConfig string        `yaml:"per_tenant_override_config"`
	PerTenantOverridePeriod time.Duration `yaml:"per_tenant_override_period"`
}

// RegisterFlags adds the flags required to config this to the given FlagSet
func (l *Limits) RegisterFlags(f *flag.FlagSet) {
	// Distributor Limits
	f.StringVar(&l.IngestionRateStrategy, "distributor.rate-limit-strategy", "local", "Whether the various ingestion rate limits should be applied individually to each distributor instance (local), or evenly shared across the cluster (global).")
	f.IntVar(&l.IngestionRate, "distributor.ingestion-rate-limit", 100000, "Per-user ingestion rate limit in spans per second.")
	f.IntVar(&l.IngestionBurstSize, "distributor.ingestion-burst-size", 50000, "Per-user allowed ingestion burst size (in number of spans).") // jpe ??

	// Ingester limits
	f.IntVar(&l.MaxLocalTracesPerUser, "ingester.max-traces-per-user", 10e3, "Maximum number of active streams per user, per ingester. 0 to disable.")
	f.IntVar(&l.MaxGlobalTracesPerUser, "ingester.max-global-traces-per-user", 0, "Maximum number of active streams per user, across the cluster. 0 to disable.")

	f.StringVar(&l.PerTenantOverrideConfig, "limits.per-user-override-config", "", "File name of per-user overrides.")
	f.DurationVar(&l.PerTenantOverridePeriod, "limits.per-user-override-period", 10*time.Second, "Period with this to reload the overrides.")
}

// UnmarshalYAML implements the yaml.Unmarshaler interface.
func (l *Limits) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// We want to set c to the defaults and then overwrite it with the input.
	// To make unmarshal fill the plain data struct rather than calling UnmarshalYAML
	// again, we have to hide it using a type indirection.  See prometheus/config.

	// During startup we wont have a default value so we don't want to overwrite them
	if defaultLimits != nil {
		*l = *defaultLimits
	}
	type plain Limits
	return unmarshal((*plain)(l))
}

// When we load YAML from disk, we want the various per-customer limits
// to default to any values specified on the command line, not default
// command line values.  This global contains those values.  I (Tom) cannot
// find a nicer way I'm afraid.
var defaultLimits *Limits

// TenantLimits is a function that returns limits for given tenant, or
// nil, if there are no tenant-specific limits.
type TenantLimits func(userID string) *Limits

// Overrides periodically fetch a set of per-user overrides, and provides convenience
// functions for fetching the correct value.
type Overrides struct {
	defaultLimits *Limits
	tenantLimits  TenantLimits
}

// NewOverrides makes a new Overrides.
// We store the supplied limits in a global variable to ensure per-tenant limits
// are defaulted to those values.  As such, the last call to NewOverrides will
// become the new global defaults.
func NewOverrides(defaults Limits, tenantLimits TenantLimits) (*Overrides, error) {
	defaultLimits = &defaults
	return &Overrides{
		tenantLimits:  tenantLimits,
		defaultLimits: &defaults,
	}, nil
}

// IngestionRateStrategy returns whether the ingestion rate limit should be individually applied
// to each distributor instance (local) or evenly shared across the cluster (global).
func (o *Overrides) IngestionRateStrategy() string {
	// The ingestion rate strategy can't be overridden on a per-tenant basis,
	// so here we just pick the value for a not-existing user ID (empty string).
	return o.getOverridesForUser("").IngestionRateStrategy
}

// MaxLocalTracesPerUser returns the maximum number of streams a user is allowed to store
// in a single ingester.
func (o *Overrides) MaxLocalTracesPerUser(userID string) int {
	return o.getOverridesForUser(userID).MaxLocalTracesPerUser
}

// MaxGlobalTracesPerUser returns the maximum number of streams a user is allowed to store
// across the cluster.
func (o *Overrides) MaxGlobalTracesPerUser(userID string) int {
	return o.getOverridesForUser(userID).MaxGlobalTracesPerUser
}

// IngestionRateSpans is the number of spans per second allowed for this tenant
func (o *Overrides) IngestionRateSpans(userID string) float64 {
	return float64(o.getOverridesForUser(userID).IngestionRate)
}

// IngestionBurstSpans is the burst size in spans allowed for this tenant
func (o *Overrides) IngestionBurstSpans(userID string) int {
	return o.getOverridesForUser(userID).IngestionBurstSize
}

func (o *Overrides) getOverridesForUser(userID string) *Limits {
	if o.tenantLimits != nil {
		l := o.tenantLimits(userID)
		if l != nil {
			return l
		}
	}
	return o.defaultLimits
}
