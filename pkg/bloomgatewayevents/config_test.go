package bloomgatewayevents

import (
	"flag"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// validConfig returns a Config that RegisterFlagsAndApplyDefaults has
// populated, with Enabled forced on -- i.e. one that Validate() should
// accept. Test cases mutate a copy of this to isolate exactly one invalid
// field at a time, mirroring modules/bloomgateway/config_test.go's
// validConfig helper.
func validConfig(t *testing.T) Config {
	t.Helper()
	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("bloom-gateway-producer", flag.NewFlagSet("test", flag.ContinueOnError))
	cfg.Enabled = true
	require.NoError(t, cfg.Validate(), "defaults + enabled must be valid")
	return cfg
}

// TestConfig_DefaultsMatchGatewayConsumer pins the producer-side Kafka
// defaults to the exact literals the consumer uses
// (modules/bloomgateway/config.go's defaultKafkaTopic and
// defaultAutoCreateTopicDefaultPartitions, ~line 63 and ~line 69): the two
// sides must agree by construction, or producers and the gateway's consumer
// never meet on the same topic/partition count. Deliberately does not
// import modules/bloomgateway here (heavy transitive deps for a small
// config package) -- the literals themselves are the contract.
func TestConfig_DefaultsMatchGatewayConsumer(t *testing.T) {
	var cfg Config
	cfg.RegisterFlagsAndApplyDefaults("bloom-gateway-producer", flag.NewFlagSet("test", flag.ContinueOnError))

	assert.Equal(t, "tempo.bloom-gateway.events", cfg.Kafka.Topic)
	assert.Equal(t, 16, cfg.Kafka.AutoCreateTopicDefaultPartitions)
}

// TestConfig_DefaultsDisabled locks in that publishing is opt-in: an
// unconfigured Config must never start producing bloom-gateway events.
func TestConfig_DefaultsDisabled(t *testing.T) {
	var cfg Config
	cfg.RegisterFlagsAndApplyDefaults("bloom-gateway-producer", flag.NewFlagSet("test", flag.ContinueOnError))

	assert.False(t, cfg.Enabled)
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(cfg *Config)
		wantErr string
	}{
		{
			name: "disabled with an otherwise-invalid config is valid",
			mutate: func(cfg *Config) {
				cfg.Enabled = false
				cfg.ChunkSize = 0
				cfg.Kafka.Topic = ""
			},
			wantErr: "",
		},
		{
			name:    "enabled with defaults is valid",
			mutate:  func(_ *Config) {},
			wantErr: "",
		},
		{
			name: "enabled, chunk size zero",
			mutate: func(cfg *Config) {
				cfg.ChunkSize = 0
			},
			wantErr: "chunk_size must be > 0",
		},
		{
			name: "enabled, chunk size negative",
			mutate: func(cfg *Config) {
				cfg.ChunkSize = -1
			},
			wantErr: "chunk_size must be > 0",
		},
		{
			name: "enabled, invalid kafka sub-config surfaces through Validate",
			mutate: func(cfg *Config) {
				cfg.Kafka.Topic = ""
			},
			wantErr: "kafka:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig(t)
			tt.mutate(&cfg)

			err := cfg.Validate()
			if tt.wantErr == "" {
				assert.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

// TestConfig_RegisterFlagsAndApplyDefaults_Idempotent guards the module-
// wiring convention that RegisterFlagsAndApplyDefaults must be side-effect-
// free beyond mutating its own receiver: NewDefaultConfig()-style callers
// invoke it a second time, on a throwaway Config, purely to compute
// /status/config?mode=defaults|diff. Two independent zero-value Configs,
// each registered against its own fresh FlagSet, must end up identical.
func TestConfig_RegisterFlagsAndApplyDefaults_Idempotent(t *testing.T) {
	var cfg1, cfg2 Config
	cfg1.RegisterFlagsAndApplyDefaults("bloom-gateway-producer", flag.NewFlagSet("one", flag.ContinueOnError))
	cfg2.RegisterFlagsAndApplyDefaults("bloom-gateway-producer", flag.NewFlagSet("two", flag.ContinueOnError))

	assert.Equal(t, cfg1, cfg2)
}
