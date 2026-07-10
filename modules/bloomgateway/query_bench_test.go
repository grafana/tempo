package bloomgateway

import (
	"context"
	"encoding/binary"
	"math/rand"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/tempodb/backend"
)

// benchQueryD/F are the reference DESIGN.md sizing's fingerprint width
// (F=16, § Sizing) with a small directory (D much smaller than the
// reference D=25): only ONE leaf slot is ever populated or touched by this
// benchmark, so a huge directory would only inflate setup memory (the
// testing-conventions report's own CI-budget gotcha) for zero benchmark
// value -- D just needs to be large enough that LeafRingToken-style
// addressing math stays realistic, which any modest value satisfies.
const (
	benchQueryD uint8 = 16
	benchQueryF uint8 = 16
)

// setupQueryBenchmark builds a Server with exactly one populated leaf at
// DESIGN.md's reference ~596-entries/leaf sizing (§ Sizing table;
// leaf_bench_test.go's own referenceEntriesPerLeaf constant), plus
// numBlocksInWindow Live blocks in one tenant's A_T window -- reproducing
// the two axes DESIGN.md's § Cost claims separately: leaf lookup cost
// (~10 probes over ~3.6 KiB) and, at large numBlocksInWindow, response
// serialization cost (§ Cost: "Sub-ms except for large rejection sets,
// where serialization dominates").
func setupQueryBenchmark(b *testing.B, numBlocksInWindow int) (*Server, *tempopb.BloomGatewayQueryRequest) {
	b.Helper()

	dir := NewDirectory(benchQueryD)
	reg := NewRegistry()
	tenants := NewTenantSet()
	m := newMetrics(prometheus.NewRegistry())

	seed := []byte("bloom-gateway-query-benchmark-seed")
	hashSeed := HashSeed(seed)

	const tenantID = "bench-tenant"
	start := time.Unix(0, 0).UTC()
	end := start.Add(10 * time.Minute) // single bucket; window size is driven by block COUNT, not bucket count

	// A query trace ID that (almost certainly) does not collide with any
	// of the reference-sized filler entries below -- the dominant,
	// ~99%-of-queries "no match" path (DESIGN.md § Cost end-to-end
	// figures), which is exactly the path whose cost this benchmark
	// measures.
	queryTraceID := traceID(0)
	leafIdx, _ := Address(queryTraceID, hashSeed, benchQueryD, benchQueryF)

	leaf, started := dir.BeginConstructing(leafIdx)
	require.True(b, started)

	rng := rand.New(rand.NewSource(1))
	for i := 0; i < referenceEntriesPerLeaf; i++ {
		var buf [2]byte
		_, _ = rng.Read(buf[:])
		dir.InsertLive(leafIdx, binary.BigEndian.Uint16(buf[:]), Handle(i+1))
	}
	require.NoError(b, dir.Complete(leafIdx, leaf))

	// Random (not sequential) UUIDs, matching DESIGN.md's own "Block UUIDs
	// ... are uniform in 128-bit space" (§ Protocol) -- sequential test
	// UUIDs (testUUID) would make the delta encoding artificially compact
	// and understate the real rejected_bytes this benchmark reports.
	for i := 0; i < numBlocksInWindow; i++ {
		uuid := backend.NewUUID()
		blk, _ := reg.GetOrCreate(uuid, tenantID, start, end)
		require.NoError(b, reg.CommitLive(uuid, false))
		tenants.AddBlock(tenantID, blk.Handle, start, end)
	}

	srv := NewServer(dir, reg, tenants, seed, benchQueryD, benchQueryF, m)
	req := &tempopb.BloomGatewayQueryRequest{TenantId: tenantID, TraceId: queryTraceID}
	return srv, req
}

// BenchmarkQuery is one of the implementation plan's three named hot-path
// benchmarks. Sub-benchmarks isolate DESIGN.md's two cost regimes
// separately (§ Cost): a modest window, where the leaf's binary search
// dominates, and the 100k-block unscoped window from § Response size's own
// worked example, where serialization dominates.
func BenchmarkQuery(b *testing.B) {
	sizes := []struct {
		name   string
		blocks int
	}{
		{"reference_leaf_small_window_1k_blocks", 1_000},
		{"100k_block_unscoped", 100_000},
	}

	for _, sz := range sizes {
		b.Run(sz.name, func(b *testing.B) {
			srv, req := setupQueryBenchmark(b, sz.blocks)
			ctx := context.Background()

			b.ReportAllocs()
			b.ResetTimer()
			var lastRejectedBytes int
			for i := 0; i < b.N; i++ {
				resp, err := srv.Query(ctx, req)
				if err != nil {
					b.Fatal(err)
				}
				lastRejectedBytes = len(resp.Rejected)
			}
			b.StopTimer()
			b.ReportMetric(float64(lastRejectedBytes), "rejected_bytes")
		})
	}
}
