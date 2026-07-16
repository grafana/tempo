package bloomgateway

import (
	"flag"
	"testing"
	"time"

	"github.com/grafana/dskit/flagext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// validConfig returns a Config that RegisterFlagsAndApplyDefaults has
// populated, plus the one field it deliberately leaves empty (Seed) filled
// in — i.e. a config that Validate() should accept. Test cases mutate a
// copy of this to isolate exactly one invalid field at a time.
func validConfig(t *testing.T) Config {
	t.Helper()
	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("bloom-gateway", flag.NewFlagSet("test", flag.ContinueOnError))
	require.NoError(t, cfg.Seed.Set("test-seed"))
	require.NoError(t, cfg.Validate(), "defaults + a seed must be valid")
	return cfg
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(cfg *Config)
		wantErr string
	}{
		{
			name: "missing seed",
			mutate: func(cfg *Config) {
				cfg.Seed = flagext.Secret{}
			},
			wantErr: "seed is required",
		},
		{
			name: "f exceeds v1 storage width",
			mutate: func(cfg *Config) {
				cfg.F = maxFingerprintBits + 1
			},
			wantErr: "f must be <=",
		},
		{
			name: "f at v1 storage width is valid",
			mutate: func(cfg *Config) {
				cfg.F = maxFingerprintBits
				cfg.D = maxLeafAddressBits // 32+16=48 <= 64, and d is at its own cap
			},
			wantErr: "",
		},
		{
			name: "d is zero",
			mutate: func(cfg *Config) {
				cfg.D = 0
			},
			wantErr: "d must be between 1 and",
		},
		{
			name: "d exceeds ring token space",
			mutate: func(cfg *Config) {
				cfg.D = maxLeafAddressBits + 1
				cfg.F = 0
			},
			wantErr: "d must be between 1 and",
		},
		{
			// d=32 is individually valid (at its own cap); f=33 exceeds
			// BOTH maxFingerprintBits and, combined with d, d+f<=64. This
			// case exists to pin down that Validate checks d+f before f's
			// own bound (see the ordering comment in Validate), which is
			// what makes the d+f check reachable/observable at all today
			// given d<=32 and f<=16 alone can never sum past 64.
			name: "d+f exceeds hash width",
			mutate: func(cfg *Config) {
				cfg.D = 32
				cfg.F = 33
			},
			wantErr: "d+f must be <=",
		},
		{
			name: "d+f exactly at hash width is valid",
			mutate: func(cfg *Config) {
				cfg.D = 32
				cfg.F = 16 // <= maxFingerprintBits, and 32+16=48 <= 64
			},
			wantErr: "",
		},
		{
			name: "num_tokens exceeds ring cap",
			mutate: func(cfg *Config) {
				cfg.NumTokens = maxRingTokens + 1
			},
			wantErr: "num_tokens must be between 1 and",
		},
		{
			name: "num_tokens zero",
			mutate: func(cfg *Config) {
				cfg.NumTokens = 0
			},
			wantErr: "num_tokens must be between 1 and",
		},
		{
			name: "num_tokens negative",
			mutate: func(cfg *Config) {
				cfg.NumTokens = -1
			},
			wantErr: "num_tokens must be between 1 and",
		},
		{
			name: "num_tokens at cap is valid",
			mutate: func(cfg *Config) {
				cfg.NumTokens = maxRingTokens
			},
			wantErr: "",
		},
		{
			name: "snapshot enabled without a path",
			mutate: func(cfg *Config) {
				cfg.Snapshot.Interval = 1
				cfg.Snapshot.Path = ""
			},
			wantErr: "snapshot.path is required",
		},
		{
			name: "snapshot disabled without a path is valid",
			mutate: func(cfg *Config) {
				cfg.Snapshot.Interval = 0
				cfg.Snapshot.Path = ""
			},
			wantErr: "",
		},
		{
			name: "invalid kafka sub-config surfaces through Validate",
			mutate: func(cfg *Config) {
				cfg.Kafka.Topic = ""
			},
			wantErr: "kafka:",
		},
		{
			name: "ring_stability_window zero",
			mutate: func(cfg *Config) {
				cfg.RingStabilityWindow = 0
			},
			wantErr: "ring_stability_window must be greater than 0",
		},
		{
			name: "ring_stability_window negative",
			mutate: func(cfg *Config) {
				cfg.RingStabilityWindow = -time.Second
			},
			wantErr: "ring_stability_window must be greater than 0",
		},
		{
			name: "ring_stability_timeout shorter than ring_stability_window",
			mutate: func(cfg *Config) {
				cfg.RingStabilityWindow = 30 * time.Second
				cfg.RingStabilityTimeout = 15 * time.Second
			},
			wantErr: "ring_stability_timeout",
		},
		{
			name: "ring_stability_timeout exactly equal to ring_stability_window is valid",
			mutate: func(cfg *Config) {
				cfg.RingStabilityWindow = 15 * time.Second
				cfg.RingStabilityTimeout = 15 * time.Second
			},
			wantErr: "",
		},
		{
			name: "ring_auto_forget_timeout less than ring heartbeat timeout",
			mutate: func(cfg *Config) {
				cfg.Ring.HeartbeatTimeout = time.Minute
				cfg.RingAutoForgetTimeout = 30 * time.Second
			},
			wantErr: "ring_auto_forget_timeout",
		},
		{
			// The boundary itself: == is rejected, not just <, since
			// forgetting no later than reassignment races it rather than
			// following it (defaultRingAutoForgetTimeout's own comment).
			name: "ring_auto_forget_timeout exactly equal to ring heartbeat timeout is rejected",
			mutate: func(cfg *Config) {
				cfg.Ring.HeartbeatTimeout = time.Minute
				cfg.RingAutoForgetTimeout = time.Minute
			},
			wantErr: "ring_auto_forget_timeout",
		},
		{
			name: "ring_auto_forget_timeout just above ring heartbeat timeout is valid",
			mutate: func(cfg *Config) {
				cfg.Ring.HeartbeatTimeout = time.Minute
				cfg.RingAutoForgetTimeout = time.Minute + time.Second
			},
			wantErr: "",
		},
		{
			name: "shutdown_marker_dir empty",
			mutate: func(cfg *Config) {
				cfg.ShutdownMarkerDir = ""
			},
			wantErr: "shutdown_marker_dir is required",
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

// TestConfig_RingStabilityDefaults pins down the bug report's own suggested
// starting point (window ~15s, timeout ~1m) so a future default change is
// deliberate, not accidental -- and that the defaults RegisterFlags
// AndApplyDefaults applies satisfy Validate on their own.
func TestConfig_RingStabilityDefaults(t *testing.T) {
	cfg := validConfig(t)
	assert.Equal(t, 15*time.Second, cfg.RingStabilityWindow)
	assert.Equal(t, time.Minute, cfg.RingStabilityTimeout)
	assert.NoError(t, cfg.Validate())
}

// TestConfig_ShutdownDefaults pins down the 2026-07-16 shutdown-semantics
// redesign's own defaults (DESIGN.md § Availability model amendment) so a
// future change to any of them is deliberate, not accidental: the ring
// heartbeat timeout is widened well past pkg/ring.Config's own shared
// 1-minute default (bloom-gateway-specific override, config.go), the
// auto-forget timeout is independently defaulted rather than derived from
// it, a graceful stop keeps the instance in the ring by default, and the
// shutdown marker has a real, non-empty default directory.
func TestConfig_ShutdownDefaults(t *testing.T) {
	cfg := validConfig(t)
	assert.Equal(t, 15*time.Minute, cfg.Ring.HeartbeatTimeout)
	assert.Equal(t, time.Hour, cfg.RingAutoForgetTimeout)
	assert.False(t, cfg.UnregisterOnShutdown)
	assert.Equal(t, defaultShutdownMarkerDir, cfg.ShutdownMarkerDir)
	assert.NoError(t, cfg.Validate())
}

// TestConfig_RegisterFlagsAndApplyDefaults_Idempotent guards the module-
// wiring convention that RegisterFlagsAndApplyDefaults must be side-effect-
// free beyond mutating its own receiver: NewDefaultConfig()-style callers
// invoke it a second time, on a throwaway Config, purely to compute
// /status/config?mode=defaults|diff. Two independent zero-value Configs,
// each registered against its own fresh FlagSet, must end up identical.
func TestConfig_RegisterFlagsAndApplyDefaults_Idempotent(t *testing.T) {
	var cfg1, cfg2 Config
	cfg1.RegisterFlagsAndApplyDefaults("bloom-gateway", flag.NewFlagSet("one", flag.ContinueOnError))
	cfg2.RegisterFlagsAndApplyDefaults("bloom-gateway", flag.NewFlagSet("two", flag.ContinueOnError))

	assert.Equal(t, cfg1, cfg2)
}

// TestConfig_RegisterFlagsAndApplyDefaults_DFlagRoundTrips exercises the
// flag.Func-based -<prefix>.d/-<prefix>.f registrations end to end (parse
// failure, valid value), since flag.Func has no built-in range/type
// checking the way flag.IntVar does.
func TestConfig_RegisterFlagsAndApplyDefaults_DFlagRoundTrips(t *testing.T) {
	var cfg Config
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	cfg.RegisterFlagsAndApplyDefaults("bloom-gateway", fs)

	require.NoError(t, fs.Parse([]string{"-bloom-gateway.d=20", "-bloom-gateway.f=12"}))
	assert.EqualValues(t, 20, cfg.D)
	assert.EqualValues(t, 12, cfg.F)

	fs2 := flag.NewFlagSet("test2", flag.ContinueOnError)
	var cfg2 Config
	cfg2.RegisterFlagsAndApplyDefaults("bloom-gateway", fs2)
	fs2.SetOutput(discardWriter{})
	assert.Error(t, fs2.Parse([]string{"-bloom-gateway.d=not-a-number"}))
}

// discardWriter silences flag.FlagSet's default os.Stderr usage-error
// output for the negative-parse test case above.
type discardWriter struct{}

func (discardWriter) Write(p []byte) (int, error) { return len(p), nil }
