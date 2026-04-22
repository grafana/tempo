package backendworker

import (
	"flag"
	"testing"

	"github.com/grafana/tempo/tempodb"
	"github.com/stretchr/testify/require"
)

func TestValidateConfig(t *testing.T) {
	validBase := func() Config {
		cfg := Config{}
		cfg.RegisterFlagsAndApplyDefaults("backend-worker", flag.NewFlagSet("test", flag.ContinueOnError))
		cfg.BackendSchedulerAddr = "localhost:1234"
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
			name:    "missing backend scheduler addr",
			modify:  func(c *Config) { c.BackendSchedulerAddr = "" },
			wantErr: "backend scheduler address is required",
		},
		{
			name:    "zero compaction window",
			modify:  func(c *Config) { c.Compactor = tempodb.CompactorConfig{} },
			wantErr: "compactor config: compaction window can't be 0",
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
