package distributor

import (
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/grafana/dskit/flagext"
	"github.com/stretchr/testify/require"
)

// TestToBasicLifecyclerConfig_RejectsNonPositiveHeartbeat ensures the
// BasicLifecycler path preserves the legacy ring.Lifecycler fail-fast on a
// zero (or negative) heartbeat period/timeout. A zero period disables
// heartbeating and a zero timeout makes the auto-forget period zero, both of
// which leave the distributor ring in a broken state.
func TestToBasicLifecyclerConfig_RejectsNonPositiveHeartbeat(t *testing.T) {
	logger := log.NewNopLogger()

	newCfg := func() RingConfig {
		cfg := RingConfig{}
		flagext.DefaultValues(&cfg)
		cfg.InstanceID = "d1"
		cfg.InstanceAddr = "127.0.0.1" // avoid network interface lookup
		return cfg
	}

	t.Run("defaults are valid", func(t *testing.T) {
		_, err := toBasicLifecyclerConfig(newCfg(), logger)
		require.NoError(t, err)
	})

	for _, tc := range []struct {
		name   string
		mutate func(*RingConfig)
	}{
		{"zero heartbeat period", func(c *RingConfig) { c.HeartbeatPeriod = 0 }},
		{"negative heartbeat period", func(c *RingConfig) { c.HeartbeatPeriod = -time.Second }},
		{"zero heartbeat timeout", func(c *RingConfig) { c.HeartbeatTimeout = 0 }},
		{"negative heartbeat timeout", func(c *RingConfig) { c.HeartbeatTimeout = -time.Second }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := newCfg()
			tc.mutate(&cfg)
			_, err := toBasicLifecyclerConfig(cfg, logger)
			require.Error(t, err)
		})
	}
}
