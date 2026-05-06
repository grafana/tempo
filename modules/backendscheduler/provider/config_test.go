package provider

import (
	"flag"
	"testing"

	"github.com/grafana/tempo/tempodb"
	"github.com/stretchr/testify/require"
)

func TestValidateConfig(t *testing.T) {
	validBase := func() Config {
		cfg := Config{}
		cfg.RegisterFlagsAndApplyDefaults("provider", flag.NewFlagSet("test", flag.ContinueOnError))
		return cfg
	}

	tests := []struct {
		name    string
		modify  func(*Config)
		wantErr string
	}{
		{
			name:   "valid config",
			modify: func(_ *Config) {},
		},
		{
			name:    "zero compaction window",
			modify:  func(c *Config) { c.Compaction.Compactor = tempodb.CompactorConfig{} },
			wantErr: "compaction config: compaction window can't be 0",
		},
		{
			name:    "zero max_jobs_per_tenant",
			modify:  func(c *Config) { c.Compaction.MaxJobsPerTenant = 0 },
			wantErr: "max_jobs_per_tenant must be greater than 0",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := validBase()
			tc.modify(&cfg)
			err := ValidateConfig(&cfg)
			if tc.wantErr != "" {
				require.EqualError(t, err, tc.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
