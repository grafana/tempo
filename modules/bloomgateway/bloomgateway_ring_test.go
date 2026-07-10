package bloomgateway

import (
	"context"
	"flag"
	"fmt"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/grafana/dskit/kv"
	"github.com/grafana/dskit/kv/consul"
	"github.com/grafana/dskit/ring"
	"github.com/grafana/dskit/services"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	tempo_ring "github.com/grafana/tempo/pkg/ring"
)

// Test sizing throughout this file uses a small d (4-6) per the task's own
// instruction: NewDirectory-scale allocations belong to later WPs, and
// 2^25 directories do not belong in unit tests. Ring/token sizing (512
// tokens/instance) is dskit's own fixed cost and independent of d.

const (
	testHeartbeatPeriod   = 20 * time.Millisecond
	testHeartbeatTimeout  = 200 * time.Millisecond
	testWaitActiveTimeout = 5 * time.Second
)

// newTestRingConfig returns a tempo_ring.Config wired to a shared in-memory
// KV client, following modules/distributor/distributor_ring_mixed_test.go's
// pattern (cfg.KVStore.Mock = store) rather than kv.Config{Store:
// "inmemory"}: the latter resolves to a single PROCESS-WIDE singleton (see
// vendor/.../dskit/kv/client.go's inmemoryStoreInit sync.Once), which would
// leak ring state between test functions in this package unless every test
// used a distinct ring key. A directly-constructed consul in-memory client
// is isolated per test by construction instead.
func newTestRingConfig(t *testing.T, store kv.Client) tempo_ring.Config {
	t.Helper()
	var cfg tempo_ring.Config
	// tempo_ring.Config uses this repo's own RegisterFlagsAndApplyDefaults
	// convention (prefix + FlagSet), not dskit's plain Registerer
	// interface, so flagext.DefaultValues (which only recognizes
	// RegisterFlags(*flag.FlagSet)) can't initialize it — call it
	// directly instead, matching modules/livestore's own test setup.
	cfg.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})
	cfg.KVStore.Mock = store
	cfg.HeartbeatPeriod = testHeartbeatPeriod
	cfg.HeartbeatTimeout = testHeartbeatTimeout
	cfg.InstanceAddr = "127.0.0.1"
	cfg.InstancePort = 0
	cfg.ListenPort = 0
	return cfg
}

// newTestRingManagers is this repo's first multi-lifecycler test harness
// (ring-lifecycler / testing-conventions reports: no existing test spins up
// several ring.BasicLifecyclers against one shared KV store and asserts
// real token ownership; distributor_ring_mixed_test.go comes closest but
// mixes legacy/basic lifecyclers 1:1:1, not N instances of the identical
// shape this test needs). Reusable by later WPs (WP18/WP20 per the plan).
//
// Returns the started managers, already registered to stop via t.Cleanup;
// instance IDs are "bloom-gateway-0".."bloom-gateway-<n-1>" (the "name-N"
// shape SpreadMinimizingTokenGenerator requires).
func newTestRingManagers(t *testing.T, n, numTokens int) []*RingManager {
	t.Helper()

	logger := log.NewNopLogger()
	store, closer := consul.NewInMemoryClient(ring.GetCodec(), logger, nil)

	cfg := newTestRingConfig(t, store)

	rms := make([]*RingManager, n)
	for i := 0; i < n; i++ {
		rm, err := NewRingManager(cfg, fmt.Sprintf("bloom-gateway-%d", i), "", numTokens, logger, prometheus.NewRegistry())
		require.NoError(t, err)
		rms[i] = rm
	}

	ctx := context.Background()
	for _, rm := range rms {
		for _, svc := range rm.Services() {
			require.NoError(t, services.StartAndAwaitRunning(ctx, svc))
		}
	}

	stop := func() {
		for _, rm := range rms {
			for _, svc := range rm.Services() {
				_ = services.StopAndAwaitTerminated(context.Background(), svc)
			}
		}
		_ = closer.Close()
	}
	t.Cleanup(stop)

	// WaitInstanceState on each rm's own ID only confirms that instance
	// sees ITSELF as ACTIVE in its own *ring.Ring view — each instance's
	// read Ring polls/watches the shared KV independently, so it does not
	// guarantee this instance has also observed its siblings' (equally
	// racy) registrations yet. Wait for every ring to agree on the full,
	// stable membership instead (repo convention: require.Eventually for
	// async convergence, not a single-shot per-instance check).
	activeOp := ring.NewOp([]ring.InstanceState{ring.ACTIVE}, nil)
	require.Eventually(t, func() bool {
		for _, rm := range rms {
			rs, err := rm.Ring.GetAllHealthy(activeOp)
			if err != nil || len(rs.Instances) != n {
				return false
			}
		}
		return true
	}, testWaitActiveTimeout, 10*time.Millisecond, "every instance's ring view should converge on all %d instances being ACTIVE", n)

	return rms
}

func TestNewRingManager_AllInstancesReachActive(t *testing.T) {
	rms := newTestRingManagers(t, 4, 16)

	for _, rm := range rms {
		state, err := rm.Ring.GetInstanceState(rm.Lifecycler.GetInstanceID())
		require.NoError(t, err)
		assert.Equal(t, ring.ACTIVE, state)
	}
}

func TestNewRingManager_TokenCountMatchesRequested(t *testing.T) {
	for _, numTokens := range []int{1, 16, 128, MaxNumTokens} {
		t.Run(fmt.Sprintf("numTokens=%d", numTokens), func(t *testing.T) {
			rms := newTestRingManagers(t, 1, numTokens)
			assert.Len(t, rms[0].Lifecycler.GetTokens(), numTokens)
		})
	}
}

func TestNewRingManager_FailsFast(t *testing.T) {
	logger := log.NewNopLogger()
	store, closer := consul.NewInMemoryClient(ring.GetCodec(), logger, nil)
	t.Cleanup(func() { _ = closer.Close() })

	t.Run("zero heartbeat timeout", func(t *testing.T) {
		cfg := newTestRingConfig(t, store)
		cfg.HeartbeatTimeout = 0
		_, err := NewRingManager(cfg, "bloom-gateway-0", "", 16, logger, prometheus.NewRegistry())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "heartbeat timeout")
	})

	t.Run("num tokens exceeds dskit's hard cap", func(t *testing.T) {
		cfg := newTestRingConfig(t, store)
		_, err := NewRingManager(cfg, "bloom-gateway-0", "", MaxNumTokens+1, logger, prometheus.NewRegistry())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "num tokens")
	})

	t.Run("num tokens zero", func(t *testing.T) {
		cfg := newTestRingConfig(t, store)
		_, err := NewRingManager(cfg, "bloom-gateway-0", "", 0, logger, prometheus.NewRegistry())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "num tokens")
	})

	t.Run("instance ID does not match name-N", func(t *testing.T) {
		cfg := newTestRingConfig(t, store)
		_, err := NewRingManager(cfg, "not-a-statefulset-name", "", 16, logger, prometheus.NewRegistry())
		require.Error(t, err, "a bare hostname without a numeric ordinal suffix must fail construction, not some later unrelated-looking error")
		assert.Contains(t, err.Error(), "name-N")
	})
}

// TestNewRingManager_TokensAreDeterministic pins down the property that
// makes "replacement reuses the same ordinal, no reconstruction cost
// beyond what scale-out costs" (§ Availability model, § Replacement) true:
// two INDEPENDENTLY constructed RingManagers (separate in-memory KV
// stores — i.e. separate "cells", not a shared one) for the same
// (instance ID, zone) must register byte-for-byte identical tokens.
func TestNewRingManager_TokensAreDeterministic(t *testing.T) {
	logger := log.NewNopLogger()

	buildOne := func(t *testing.T) ring.Tokens {
		t.Helper()
		store, closer := consul.NewInMemoryClient(ring.GetCodec(), logger, nil)
		t.Cleanup(func() { _ = closer.Close() })

		cfg := newTestRingConfig(t, store)
		rm, err := NewRingManager(cfg, "bloom-gateway-3", "", 32, logger, prometheus.NewRegistry())
		require.NoError(t, err)

		ctx := context.Background()
		for _, svc := range rm.Services() {
			require.NoError(t, services.StartAndAwaitRunning(ctx, svc))
		}
		t.Cleanup(func() {
			for _, svc := range rm.Services() {
				_ = services.StopAndAwaitTerminated(context.Background(), svc)
			}
		})
		waitCtx, cancel := context.WithTimeout(ctx, testWaitActiveTimeout)
		defer cancel()
		require.NoError(t, ring.WaitInstanceState(waitCtx, rm.Ring, rm.Lifecycler.GetInstanceID(), ring.ACTIVE))

		return rm.Lifecycler.GetTokens()
	}

	first := buildOne(t)
	second := buildOne(t)

	require.NotEmpty(t, first)
	assert.Equal(t, first, second)
}

// TestOwnedLeafRanges_FullCoveragePartition is the "full-coverage
// OwnedLeafRanges partition check at small d" the plan calls for: at a
// real (if small) ring built from newTestRingManagers, every leaf index in
// [0, 2^d) must be owned by exactly one of the N instances, with no gaps
// and no double-ownership.
func TestOwnedLeafRanges_FullCoveragePartition(t *testing.T) {
	const (
		d = 5 // 32 leaves — small per the task's own instruction
		n = 3
	)
	rms := newTestRingManagers(t, n, 16)

	rs, err := rms[0].Ring.GetAllHealthy(ring.NewOp([]ring.InstanceState{ring.ACTIVE}, nil))
	require.NoError(t, err)
	require.Len(t, rs.Instances, n)

	owner := make([]string, uint32(1)<<d) // leaf idx -> owning instance ID, "" = unowned
	for i := 0; i < n; i++ {
		instanceID := fmt.Sprintf("bloom-gateway-%d", i)
		ranges := OwnedLeafRanges(rs.Instances, instanceID, d)
		for _, r := range ranges {
			for leaf := r.Start; leaf < r.End; leaf++ {
				require.Emptyf(t, owner[leaf], "leaf %d double-owned by %s and %s", leaf, owner[leaf], instanceID)
				owner[leaf] = instanceID
			}
		}
	}

	for leaf, id := range owner {
		assert.NotEmptyf(t, id, "leaf %d has no owner", leaf)
	}
}

// TestOwnedLeafRanges_PureTokenLeafPositionMapping is the pure, no-KV unit
// test the implementation plan calls for in place of DESIGN.md's own open
// item #2 ("validate the 512-token spread against leaf-position math in a
// dev cell"): instantiate SpreadMinimizingTokenGenerator directly for
// ordinals 0..N-1 (no ring, no lifecycler, no KV at all) and feed the
// resulting tokens straight into OwnedLeafRanges.
func TestOwnedLeafRanges_PureTokenLeafPositionMapping(t *testing.T) {
	const (
		d = 6 // 64 leaves
		n = 8 // DESIGN.md's reference instance count (§ Sizing)
	)

	instances := make([]ring.InstanceDesc, n)
	for i := 0; i < n; i++ {
		gen := ring.NewSpreadMinimizingTokenGeneratorForInstanceAndZoneID("bloom-gateway-", i, 0, false)
		instances[i] = ring.InstanceDesc{
			Id:     fmt.Sprintf("bloom-gateway-%d", i),
			Tokens: gen.GenerateTokens(MaxNumTokens, nil),
		}
	}

	owner := make([]string, uint32(1)<<d)
	for _, inst := range instances {
		for _, r := range OwnedLeafRanges(instances, inst.Id, d) {
			for leaf := r.Start; leaf < r.End; leaf++ {
				require.Emptyf(t, owner[leaf], "leaf %d double-owned", leaf)
				owner[leaf] = inst.Id
			}
		}
	}
	for leaf, id := range owner {
		assert.NotEmptyf(t, id, "leaf %d has no owner", leaf)
	}
}

func TestOwnedLeafRanges_Unit(t *testing.T) {
	t.Run("no instances owns nothing", func(t *testing.T) {
		assert.Nil(t, OwnedLeafRanges(nil, "bloom-gateway-0", 4))
	})

	t.Run("instance with no tokens owns nothing", func(t *testing.T) {
		instances := []ring.InstanceDesc{{Id: "bloom-gateway-0", Tokens: nil}}
		assert.Nil(t, OwnedLeafRanges(instances, "bloom-gateway-0", 4))
	})

	t.Run("sole instance owns the entire ring", func(t *testing.T) {
		const d = 4
		instances := []ring.InstanceDesc{{Id: "solo", Tokens: []uint32{0, 1 << 30, 1 << 31}}}
		got := OwnedLeafRanges(instances, "solo", d)
		require.Len(t, got, 1)
		assert.Equal(t, LeafRange{Start: 0, End: uint32(1) << d}, got[0])
	})

	t.Run("querying an unknown instance owns nothing", func(t *testing.T) {
		instances := []ring.InstanceDesc{{Id: "solo", Tokens: []uint32{0}}}
		assert.Nil(t, OwnedLeafRanges(instances, "someone-else", 4))
	})

	t.Run("exact-boundary token: position equal to a token belongs to the NEXT token's owner", func(t *testing.T) {
		// d=1: two leaves, positions 0 and 1<<31. Token exactly at 1<<31
		// owned by "b"; per searchToken's rule (owner of a position is
		// the smallest STRICTLY GREATER token, wrapping), position 1<<31
		// itself belongs to whichever token is > 1<<31, i.e. wraps to the
		// smallest token "a" owns at 0. So leaf 1 (position 1<<31) must
		// be owned by "a", not "b".
		const d = 1
		instances := []ring.InstanceDesc{
			{Id: "a", Tokens: []uint32{0}},
			{Id: "b", Tokens: []uint32{1 << 31}},
		}
		aRanges := OwnedLeafRanges(instances, "a", d)
		bRanges := OwnedLeafRanges(instances, "b", d)

		require.Len(t, aRanges, 1)
		require.Len(t, bRanges, 1)
		assert.Equal(t, LeafRange{Start: 1, End: 2}, aRanges[0], "leaf 1 (position 2^31, exactly b's token) wraps to a's ownership")
		assert.Equal(t, LeafRange{Start: 0, End: 1}, bRanges[0])
	})

	t.Run("coalesces adjacent ranges from multiple tokens of the same instance", func(t *testing.T) {
		const d = 4 // 16 leaves, step = 2^28
		step := uint32(1) << (32 - d)
		instances := []ring.InstanceDesc{
			// "solo" owns tokens at leaf boundaries 0, 4, 8 (leaves
			// [0,4), [4,8), [8,16) after wraparound) — contiguous, must
			// coalesce into exactly one range covering the whole ring.
			{Id: "solo", Tokens: []uint32{0, 4 * step, 8 * step}},
		}
		got := OwnedLeafRanges(instances, "solo", d)
		require.Len(t, got, 1)
		assert.Equal(t, LeafRange{Start: 0, End: 16}, got[0])
	})

	t.Run("duplicate ownership under ring disagreement does not double count within one instance's own ranges", func(t *testing.T) {
		// Two DIFFERENT instance IDs, deliberately given overlapping-
		// looking (but not literally equal) tokens is not representable
		// here (tokens key ownership uniquely) — instead this checks that
		// asking for ranges twice for the same instance is stable/pure.
		const d = 4
		instances := []ring.InstanceDesc{
			{Id: "a", Tokens: []uint32{0, 1 << 30}},
			{Id: "b", Tokens: []uint32{1 << 31}},
		}
		first := OwnedLeafRanges(instances, "a", d)
		second := OwnedLeafRanges(instances, "a", d)
		assert.Equal(t, first, second)
	})
}

func TestLeafRingToken(t *testing.T) {
	tests := []struct {
		name    string
		leafIdx uint32
		d       uint8
		want    uint32
	}{
		{"leaf 0 always maps to position 0", 0, 25, 0},
		{"d=1 leaf 1 maps to the ring midpoint", 1, 1, 1 << 31},
		{"d=4 leaf 8 maps to the ring midpoint", 8, 4, 1 << 31},
		{"d=32 is a 1:1 mapping", 12345, 32, 12345},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, LeafRingToken(tt.leafIdx, tt.d))
		})
	}
}
