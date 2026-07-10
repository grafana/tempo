package bloomgateway

import (
	"context"
	"encoding/binary"
	"math/rand"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

// benchSweepD/Leaves/DeletedFraction size BenchmarkSweepPass at a
// CI-budget-friendly scale (testing-conventions report: this package runs
// in the "test-with-cover-others" CI bucket, with no extra GOMEMLIMIT, so
// DESIGN.md's reference ~2.5x10^9-entries/instance figure is never
// attempted here). referenceEntriesPerLeaf (leaf_bench_test.go) keeps each
// leaf at the same reference-sizing population BenchmarkQuery/
// BenchmarkLeafInsert already use.
const (
	benchSweepD               uint8   = 9 // 512 slots; only benchSweepLeaves of them are populated
	benchSweepLeaves          uint32  = 500
	benchSweepDeletedFraction float64 = 0.05 // 5% of blocks start Deleted: enough to exercise RemoveWhere/Reclaim without dominating the fixture
)

// BenchmarkSweepPass is one of the implementation plan's three named
// hot-path benchmarks (DESIGN.md § Garbage collection: "a full pass visits
// ~2.5x10^9 entries per instance -- seconds of one core"). It reports
// entries/sec (total EntriesVisited across every b.N call, divided by
// total elapsed wall time) so a reader can extrapolate that claim at any
// scale, rather than asserting the reference figure directly -- honest
// numbers over a hardcoded target, matching this plan's own convention
// elsewhere (§0 D13's benchmark-not-assert finding).
//
// Only the FIRST of the b.N calls finds the ~5% seeded garbage; every
// later call walks an already-compacted (very slightly smaller) directory
// -- exactly DESIGN.md's own steady state, where most passes find little
// to no NEW garbage since the previous pass already swept it. Dividing
// cumulative entries visited by cumulative elapsed time (rather than
// reporting a single iteration's ns/op) is what keeps entries/sec
// meaningful despite that shrinkage.
func BenchmarkSweepPass(b *testing.B) {
	dir := NewDirectory(benchSweepD)
	reg := NewRegistry()
	tenants := NewTenantSet()
	m := newMetrics(prometheus.NewRegistry())
	cfg := SweepConfig{FullPassPeriod: time.Hour, ReplayHorizon: time.Hour}
	sweeper := NewSweeper(dir, reg, tenants, cfg, m, log.NewNopLogger())

	const tenantID = "bench-tenant"
	start := time.Unix(0, 0).UTC()
	end := start.Add(time.Minute)

	rng := rand.New(rand.NewSource(1))
	handleN := 0
	for i := uint32(0); i < benchSweepLeaves; i++ {
		leaf, started := dir.BeginConstructing(i)
		require.True(b, started)

		for j := 0; j < referenceEntriesPerLeaf; j++ {
			handleN++
			uuid := testUUID(b, handleN)
			blk, _ := reg.GetOrCreate(uuid, tenantID, start, end)
			require.NoError(b, reg.CommitLive(uuid, false))

			if rng.Float64() < benchSweepDeletedFraction {
				require.NoError(b, reg.MarkDeleted(uuid)) // never added to A_T: models the end state directly
			} else {
				tenants.AddBlock(tenantID, blk.Handle, start, end)
			}

			var fpBuf [2]byte
			_, _ = rng.Read(fpBuf[:])
			leaf.InsertIfAbsent(binary.BigEndian.Uint16(fpBuf[:]), blk.Handle) // sequential setup: no concurrent access, so bypassing Directory's lock is safe
		}
		require.NoError(b, dir.Complete(i, leaf))
	}

	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()
	var totalVisited int
	for i := 0; i < b.N; i++ {
		stats := sweeper.Pass(ctx)
		totalVisited += stats.EntriesVisited
	}
	elapsed := b.Elapsed()
	b.StopTimer()

	if elapsed > 0 {
		b.ReportMetric(float64(totalVisited)/elapsed.Seconds(), "entries/sec")
	}
}
