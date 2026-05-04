package distributor

import (
	"context"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/grafana/dskit/flagext"
	"github.com/grafana/dskit/kv"
	"github.com/grafana/dskit/kv/consul"
	"github.com/grafana/dskit/ring"
	"github.com/grafana/dskit/services"
	"github.com/stretchr/testify/require"
)

// TestDistributorRing_MixedLifecyclerRollout exercises the rollout path where
// some distributors run the legacy ring.Lifecycler and others run the new
// ring.BasicLifecycler with AutoForget+LeaveOnStopping delegates against a
// shared KV store. It pins down:
//
//   - Wire compatibility: a BasicLifecycler reads a Lifecycler's entry and
//     vice versa.
//   - Asymmetric rollout safety: AutoForget on a new-style peer evicts a
//     stale legacy-shaped entry within ~2*heartbeat_timeout.
//   - Graceful shutdown still removes the entry cleanly for both styles.
//   - The HealthyInstancesCount adapter (used by globalStrategy) reflects
//     reality across both versions.
func TestDistributorRing_MixedLifecyclerRollout(t *testing.T) {
	// Heartbeat timing scaled down so the test completes in seconds.
	const (
		heartbeatPeriod  = 50 * time.Millisecond
		heartbeatTimeout = 200 * time.Millisecond
		ringKey          = "distributor"
	)
	forgetPeriod := ringAutoForgetUnhealthyPeriods * heartbeatTimeout

	logger := log.NewNopLogger()
	ctx := context.Background()

	// Single in-memory KV shared between all lifecyclers — simulates a single
	// ring backend during a rolling deploy.
	mockStore, closer := consul.NewInMemoryClient(ring.GetCodec(), logger, nil)
	t.Cleanup(func() { _ = closer.Close() })

	// Reader used to observe ring state from the test's perspective.
	reader := newRingReader(t, mockStore, ringKey, heartbeatTimeout, logger)
	require.NoError(t, services.StartAndAwaitRunning(ctx, reader))
	t.Cleanup(func() { _ = services.StopAndAwaitTerminated(ctx, reader) })

	// "Old version" pod: legacy ring.Lifecycler, no delegate, no auto-forget.
	// Built via the same ToLifecyclerConfig() path used in production today.
	oldLC := newLegacyDistributorLifecycler(t, mockStore, ringKey, "old-pod", heartbeatPeriod, heartbeatTimeout, logger)

	// "New version" pods: BasicLifecycler with AutoForget+LeaveOnStopping.
	newLC1 := newBasicDistributorLifecycler(t, mockStore, ringKey, "new-pod-1", heartbeatPeriod, heartbeatTimeout, forgetPeriod, logger)
	newLC2 := newBasicDistributorLifecycler(t, mockStore, ringKey, "new-pod-2", heartbeatPeriod, heartbeatTimeout, forgetPeriod, logger)

	require.NoError(t, services.StartAndAwaitRunning(ctx, oldLC))
	require.NoError(t, services.StartAndAwaitRunning(ctx, newLC1))
	require.NoError(t, services.StartAndAwaitRunning(ctx, newLC2))

	// (1) Coexistence: all three IDs visible & ACTIVE. If the codec or
	// ring key ever drifts between Lifecycler and BasicLifecycler, this
	// fails.
	require.Eventually(t, func() bool {
		ids := healthyIDs(t, reader)
		return ids["old-pod"] && ids["new-pod-1"] && ids["new-pod-2"]
	}, 2*time.Second, 10*time.Millisecond, "all three lifecyclers should appear ACTIVE in the ring")

	// (4) HealthyInstancesCount adapter agrees with reality.
	counter := ringHealthyCounter{r: reader}
	require.Eventually(t, func() bool {
		return counter.HealthyInstancesCount() == 3
	}, time.Second, 10*time.Millisecond)

	// (2) Inject a stale entry directly via CAS — simulates a pod that
	// crashed without graceful unregister and left a stale record. The
	// codec is identical between Lifecycler and BasicLifecycler, so this
	// is exactly what either would have written.
	const ghostID = "ghost-pod"
	staleHeartbeat := time.Now().Add(-10 * forgetPeriod).Unix()
	require.NoError(t, mockStore.CAS(ctx, ringKey, func(in any) (out any, retry bool, err error) {
		desc := ring.GetOrCreateRingDesc(in)
		desc.Ingesters[ghostID] = ring.InstanceDesc{
			Id:        ghostID,
			Addr:      "127.0.0.1:0",
			Timestamp: staleHeartbeat,
			State:     ring.ACTIVE,
			Tokens:    []uint32{42},
		}
		return desc, true, nil
	}))

	// Verify the ghost is initially present in the raw KV.
	require.Eventually(t, func() bool {
		return allIDsFromKV(t, ctx, mockStore, ringKey)[ghostID]
	}, time.Second, 10*time.Millisecond)

	// (3) AutoForget on the new lifecyclers must evict the stale entry on
	// the next heartbeat past forgetPeriod. This is the load-bearing
	// assertion for the migration: it works against legacy-shaped entries.
	require.Eventually(t, func() bool {
		return !allIDsFromKV(t, ctx, mockStore, ringKey)[ghostID]
	}, 5*forgetPeriod+time.Second, 10*time.Millisecond, "auto-forget should evict the stale entry")

	// Survivors stay healthy. Eventually-wrap because the reader ring polls
	// KV on its own cadence and may not have observed the ghost removal yet.
	require.Eventually(t, func() bool {
		return counter.HealthyInstancesCount() == 3
	}, 2*time.Second, 10*time.Millisecond)

	// (5) Graceful shutdown of legacy lifecycler removes its entry.
	require.NoError(t, services.StopAndAwaitTerminated(ctx, oldLC))
	require.Eventually(t, func() bool {
		return !allIDsFromKV(t, ctx, mockStore, ringKey)["old-pod"]
	}, 2*time.Second, 10*time.Millisecond, "legacy lifecycler should unregister on graceful stop")

	// (6) Graceful shutdown of basic lifecycler also removes its entry —
	// verifies LeaveOnStoppingDelegate + KeepInstanceInTheRingOnShutdown=false
	// produce the same end state as the legacy UnregisterOnShutdown=true.
	require.NoError(t, services.StopAndAwaitTerminated(ctx, newLC1))
	require.Eventually(t, func() bool {
		return !allIDsFromKV(t, ctx, mockStore, ringKey)["new-pod-1"]
	}, 2*time.Second, 10*time.Millisecond, "basic lifecycler should unregister on graceful stop")

	// Last surviving instance count drops accordingly.
	require.Eventually(t, func() bool {
		return counter.HealthyInstancesCount() == 1
	}, time.Second, 10*time.Millisecond)

	require.NoError(t, services.StopAndAwaitTerminated(ctx, newLC2))
}

// newLegacyDistributorLifecycler builds a ring.Lifecycler exactly as today's
// production code does (RingConfig.ToLifecyclerConfig + ring.NewLifecycler with
// nil flush transferer). This is the call site being replaced; using it here
// guarantees the test would have caught the original bug as well.
func newLegacyDistributorLifecycler(t *testing.T, store kv.Client, ringKey, id string, heartbeatPeriod, heartbeatTimeout time.Duration, logger log.Logger) *ring.Lifecycler {
	t.Helper()

	cfg := RingConfig{}
	flagext.DefaultValues(&cfg)
	cfg.KVStore.Mock = store
	cfg.HeartbeatPeriod = heartbeatPeriod
	cfg.HeartbeatTimeout = heartbeatTimeout
	cfg.InstanceID = id
	cfg.InstanceAddr = "127.0.0.1"
	cfg.InstancePort = 0
	cfg.ListenPort = 0

	lc, err := ring.NewLifecycler(cfg.ToLifecyclerConfig(), nil, "distributor", ringKey, false, logger, nil)
	require.NoError(t, err)
	return lc
}

// newBasicDistributorLifecycler builds a BasicLifecycler with the production
// delegate chain (AutoForget -> LeaveOnStopping -> distributorLifecyclerDelegate).
func newBasicDistributorLifecycler(t *testing.T, store kv.Client, ringKey, id string, heartbeatPeriod, heartbeatTimeout, forgetPeriod time.Duration, logger log.Logger) *ring.BasicLifecycler {
	t.Helper()

	cfg := RingConfig{}
	flagext.DefaultValues(&cfg)
	cfg.KVStore.Mock = store
	cfg.HeartbeatPeriod = heartbeatPeriod
	cfg.HeartbeatTimeout = heartbeatTimeout
	cfg.InstanceID = id
	cfg.InstanceAddr = "127.0.0.1"
	cfg.InstancePort = 0
	cfg.ListenPort = 0

	basicCfg, err := toBasicLifecyclerConfig(cfg, logger)
	require.NoError(t, err)

	var delegate ring.BasicLifecyclerDelegate = &distributorLifecyclerDelegate{}
	delegate = ring.NewLeaveOnStoppingDelegate(delegate, logger)
	delegate = ring.NewAutoForgetDelegate(forgetPeriod, delegate, logger)

	lc, err := ring.NewBasicLifecycler(basicCfg, "distributor", ringKey, store, delegate, logger, nil)
	require.NoError(t, err)
	return lc
}

func newRingReader(t *testing.T, store kv.Client, ringKey string, heartbeatTimeout time.Duration, logger log.Logger) *ring.Ring {
	t.Helper()
	cfg := ring.Config{}
	flagext.DefaultValues(&cfg)
	cfg.HeartbeatTimeout = heartbeatTimeout
	cfg.ReplicationFactor = 1
	r, err := ring.NewWithStoreClientAndStrategy(cfg, "distributor", ringKey, store, ring.NewDefaultReplicationStrategy(), nil, logger)
	require.NoError(t, err)
	return r
}

func healthyIDs(t *testing.T, r *ring.Ring) map[string]bool {
	t.Helper()
	rs, err := r.GetAllHealthy(ringOp)
	require.NoError(t, err)
	out := map[string]bool{}
	for _, ing := range rs.Instances {
		out[ing.Id] = true
	}
	return out
}

// allIDsFromKV reads the ring descriptor straight from the KV store, which
// includes unhealthy/stale entries that GetAllHealthy filters out.
func allIDsFromKV(t *testing.T, ctx context.Context, store kv.Client, ringKey string) map[string]bool {
	t.Helper()
	val, err := store.Get(ctx, ringKey)
	require.NoError(t, err)
	out := map[string]bool{}
	if val == nil {
		return out
	}
	desc, ok := val.(*ring.Desc)
	require.True(t, ok, "expected *ring.Desc")
	for id := range desc.Ingesters {
		out[id] = true
	}
	return out
}
