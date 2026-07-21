package bloomgateway

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/pkg/deltauuid"
	"github.com/grafana/tempo/pkg/tempopb"
)

// testQuerySeed is this file's fixed raw seed -- every test derives
// expectations (e.g. SeedFingerprint) from it directly rather than
// hardcoding a captured constant, so the tests stay correct even if
// hash.go's derivation scheme is later revisited.
const testQuerySeed = "bloom-gateway-query-test-seed"

// newTestServer builds a Server over fresh Wave-1 structures at the
// package's shared small testD/testF sizing (events_test.go), returning
// the structures alongside so tests can populate them directly.
func newTestServer(t *testing.T) (*Server, *Directory, *Registry, *TenantSet) {
	t.Helper()
	dir := NewDirectory(testD)
	reg := NewRegistry()
	tenants := NewTenantSet()
	m := newMetrics(prometheus.NewRegistry())
	srv := NewServer(dir, reg, tenants, []byte(testQuerySeed), testD, testF, m)
	return srv, dir, reg, tenants
}

// completeLeaf drives idx's leaf nil -> constructing -> complete with no
// entries, so the caller can InsertLive into it afterward and Query will
// actually serve it.
func completeLeaf(t *testing.T, dir *Directory, idx uint32) {
	t.Helper()
	leaf, started := dir.BeginConstructing(idx)
	require.True(t, started)
	require.NoError(t, dir.Complete(idx, leaf))
}

// liveBlock creates a Live block for tenant "tenant-a" spanning [start, end),
// adds it to A_T, and returns its handle -- the common "make a real,
// rejectable block" fixture step almost every test below needs. Every call
// site in this package uses the same single-tenant fixture (its own local
// `const tenantID = "tenant-a"`, kept where those other usages need it), so
// this hardcodes it rather than carrying a parameter that never varies.
func liveBlock(t *testing.T, reg *Registry, tenants *TenantSet, n int, start, end time.Time) Handle {
	t.Helper()
	const tenantID = "tenant-a"
	uuid := testUUID(t, n)
	b, _ := reg.GetOrCreate(uuid, tenantID, start, end)
	require.NoError(t, reg.CommitLive(uuid, false))
	tenants.AddBlock(tenantID, b.Handle, start, end)
	return b.Handle
}

// resolveAndSort resolves handles to UUIDs and sorts them exactly the way
// Server.encodeRejected does, so a test can build its "want" value with
// the same real registry the server itself will resolve against, rather
// than hand-computing UUID bytes.
func resolveAndSort(t *testing.T, reg *Registry, handles ...Handle) [][16]byte {
	t.Helper()
	uuids := reg.ResolveHandles(handles)
	out := make([][16]byte, len(uuids))
	for i, u := range uuids {
		out[i] = [16]byte(u)
	}
	sort.Slice(out, func(i, j int) bool { return uuidLess(out[i], out[j]) })
	return out
}

// decodeRejected is the test-side inverse of Server.encodeRejected.
func decodeRejected(t *testing.T, wire []byte) [][16]byte {
	t.Helper()
	got, err := deltauuid.DecodeSortedDeltas(wire)
	require.NoError(t, err)
	return got
}

// TestQuery_NilLeafEmptyResponse_VsEmptyCompleteLeafRejectsAll is the named
// test for invariant #7 (§7, wire-level half of directory_test.go's
// TestDirectory_EmptyCompleteLeafDistinctFromNil): a nil (unowned) leaf and
// an owned-but-empty-complete leaf must produce OBSERVABLY DIFFERENT wire
// responses, even though both compute to "nothing to reject" in some
// sense -- FlagLeafUnavailable + empty for the former (an abstention);  no
// flag + rejected == the entire tenant window for the latter (a genuine,
// computed reject-all answer, DESIGN.md § Query path: "No match ->
// rejection set = A_X_window").
func TestQuery_NilLeafEmptyResponse_VsEmptyCompleteLeafRejectsAll(t *testing.T) {
	const tenantID = "tenant-a"
	tr := testTimeRange()

	t.Run("nil leaf: FlagLeafUnavailable and empty rejected, regardless of A_T contents", func(t *testing.T) {
		srv, dir, reg, tenants := newTestServer(t)
		id := traceID(1)
		leafIdx, _ := Address(id, srv.hashSeed, testD, testF)
		require.Equal(t, LeafNil, dir.State(leafIdx), "test setup: this leaf must start out unowned")

		// A block that WOULD be rejected if the leaf were served -- this
		// is what proves the empty response below is a genuine
		// abstention, not merely an empty A_T window.
		liveBlock(t, reg, tenants, 1, tr.start, tr.end)

		resp, err := srv.Query(context.Background(), &tempopb.BloomGatewayQueryRequest{TenantId: tenantID, TraceId: id})
		require.NoError(t, err)
		assert.Equal(t, wireVersion, resp.Version)
		assert.Equal(t, FlagLeafUnavailable, resp.Flags)
		assert.Empty(t, resp.Rejected, "an unavailable leaf must never carry a rejection set")
		assert.Equal(t, SeedFingerprint([]byte(testQuerySeed)), resp.SeedFingerprint)
	})

	t.Run("empty complete leaf: no flag, rejected equals the entire tenant window", func(t *testing.T) {
		srv, dir, reg, tenants := newTestServer(t)
		id := traceID(2)
		leafIdx, _ := Address(id, srv.hashSeed, testD, testF)
		completeLeaf(t, dir, leafIdx) // complete, but zero entries: no match is possible

		h1 := liveBlock(t, reg, tenants, 1, tr.start, tr.end)
		h2 := liveBlock(t, reg, tenants, 2, tr.start, tr.end)

		resp, err := srv.Query(context.Background(), &tempopb.BloomGatewayQueryRequest{TenantId: tenantID, TraceId: id})
		require.NoError(t, err)
		assert.Equal(t, wireVersion, resp.Version)
		assert.Zero(t, resp.Flags, "a genuine computed answer must never carry FlagLeafUnavailable")
		assert.Equal(t, resolveAndSort(t, reg, h1, h2), decodeRejected(t, resp.Rejected))
	})
}

// TestQuery_RejectionRequiresLiveAndAT is the named test for invariant #2
// (§7, read-side confirmation of events_test.go's
// TestApply_ATMembershipImpliesLive): a block absent from the tenant
// window -- whether unknown to the gateway entirely or merely Pending
// (chunks still arriving, entries already in the leaf but not yet
// committed to A_T) -- must never appear in rejected, even when its handle
// is what the leaf lookup actually matched.
func TestQuery_RejectionRequiresLiveAndAT(t *testing.T) {
	const tenantID = "tenant-a"
	tr := testTimeRange()

	srv, dir, reg, tenants := newTestServer(t)
	id := traceID(3)
	leafIdx, fp16 := Address(id, srv.hashSeed, testD, testF)
	fp := uint16(fp16)
	completeLeaf(t, dir, leafIdx)

	// A Pending block: its handle is already in the leaf (§ Event
	// processing step 2 always runs before step 3's commit), but it must
	// never be rejectable -- it isn't Live, and critically it isn't in
	// A_T yet either.
	pendingUUID := testUUID(t, 10)
	pendingBlock, _ := reg.GetOrCreate(pendingUUID, tenantID, tr.start, tr.end)
	require.Equal(t, BlockPending, pendingBlock.State)
	require.True(t, dir.InsertLive(leafIdx, fp, pendingBlock.Handle))

	// A genuinely Live, in-A_T block sharing the SAME fingerprint -- the
	// real candidate this query should surface.
	liveHandle := liveBlock(t, reg, tenants, 11, tr.start, tr.end)
	require.True(t, dir.InsertLive(leafIdx, fp, liveHandle))

	resp, err := srv.Query(context.Background(), &tempopb.BloomGatewayQueryRequest{TenantId: tenantID, TraceId: id})
	require.NoError(t, err)

	got := decodeRejected(t, resp.Rejected)
	assert.NotContains(t, got, [16]byte(pendingUUID), "a Pending block must never appear in rejected, even though its handle matched the leaf lookup")

	// window = {liveHandle} (the only block ever committed to A_T);
	// candidates = window ∩ matched = {liveHandle} (the pending block's
	// handle matched the leaf too, but isn't in window so can't become a
	// candidate); rejected = window - candidates = {} exactly.
	assert.Empty(t, got)
}

// TestQuery_Table is WP15's own test-plan table: {no leaf, leaf/no match,
// leaf/1 match, leaf/multiple collisions, unscoped window, scoped window
// partially overlapping buckets}.
func TestQuery_Table(t *testing.T) {
	const tenantID = "tenant-a"
	tr := testTimeRange()

	tests := []struct {
		name string
		// setup builds one scenario's Server + request, and returns the
		// expected flags and rejected-UUID set (nil meaning "empty").
		setup func(t *testing.T) (srv *Server, req *tempopb.BloomGatewayQueryRequest, wantFlags uint32, wantRejected [][16]byte)
	}{
		{
			name: "no leaf: unavailable even though the tenant has a rejectable block",
			setup: func(t *testing.T) (*Server, *tempopb.BloomGatewayQueryRequest, uint32, [][16]byte) {
				srv, _, reg, tenants := newTestServer(t)
				id := traceID(20)
				liveBlock(t, reg, tenants, 20, tr.start, tr.end) // leaf for id is never owned below
				return srv, &tempopb.BloomGatewayQueryRequest{TenantId: tenantID, TraceId: id}, FlagLeafUnavailable, nil
			},
		},
		{
			name: "leaf owned, no match: reject-all",
			setup: func(t *testing.T) (*Server, *tempopb.BloomGatewayQueryRequest, uint32, [][16]byte) {
				srv, dir, reg, tenants := newTestServer(t)
				id := traceID(21)
				leafIdx, _ := Address(id, srv.hashSeed, testD, testF)
				completeLeaf(t, dir, leafIdx)
				h := liveBlock(t, reg, tenants, 21, tr.start, tr.end)
				return srv, &tempopb.BloomGatewayQueryRequest{TenantId: tenantID, TraceId: id}, uint32(0), resolveAndSort(t, reg, h)
			},
		},
		{
			name: "leaf owned, one match: the matched block is a candidate, excluded from rejected",
			setup: func(t *testing.T) (*Server, *tempopb.BloomGatewayQueryRequest, uint32, [][16]byte) {
				srv, dir, reg, tenants := newTestServer(t)
				id := traceID(22)
				leafIdx, fp := Address(id, srv.hashSeed, testD, testF)
				completeLeaf(t, dir, leafIdx)
				matchHandle := liveBlock(t, reg, tenants, 22, tr.start, tr.end)
				require.True(t, dir.InsertLive(leafIdx, uint16(fp), matchHandle))
				otherHandle := liveBlock(t, reg, tenants, 23, tr.start, tr.end)
				return srv, &tempopb.BloomGatewayQueryRequest{TenantId: tenantID, TraceId: id}, uint32(0), resolveAndSort(t, reg, otherHandle)
			},
		},
		{
			name: "leaf owned, multiple collisions on the same fingerprint: all excluded from rejected",
			setup: func(t *testing.T) (*Server, *tempopb.BloomGatewayQueryRequest, uint32, [][16]byte) {
				srv, dir, reg, tenants := newTestServer(t)
				id := traceID(24)
				leafIdx, fp := Address(id, srv.hashSeed, testD, testF)
				completeLeaf(t, dir, leafIdx)
				matchA := liveBlock(t, reg, tenants, 24, tr.start, tr.end)
				matchB := liveBlock(t, reg, tenants, 25, tr.start, tr.end)
				require.True(t, dir.InsertLive(leafIdx, uint16(fp), matchA))
				require.True(t, dir.InsertLive(leafIdx, uint16(fp), matchB))
				otherHandle := liveBlock(t, reg, tenants, 26, tr.start, tr.end)
				return srv, &tempopb.BloomGatewayQueryRequest{TenantId: tenantID, TraceId: id}, uint32(0), resolveAndSort(t, reg, otherHandle)
			},
		},
		{
			name: "unscoped window rejects every A_T block regardless of time",
			setup: func(t *testing.T) (*Server, *tempopb.BloomGatewayQueryRequest, uint32, [][16]byte) {
				srv, dir, reg, tenants := newTestServer(t)
				id := traceID(27)
				leafIdx, _ := Address(id, srv.hashSeed, testD, testF)
				completeLeaf(t, dir, leafIdx)
				h1 := liveBlock(t, reg, tenants, 27, hourMark(0), hourMark(0).Add(time.Minute))
				h2 := liveBlock(t, reg, tenants, 28, hourMark(100), hourMark(100).Add(time.Minute))
				return srv, &tempopb.BloomGatewayQueryRequest{TenantId: tenantID, TraceId: id}, uint32(0), resolveAndSort(t, reg, h1, h2)
			},
		},
		{
			name: "scoped window only rejects blocks in overlapping buckets",
			setup: func(t *testing.T) (*Server, *tempopb.BloomGatewayQueryRequest, uint32, [][16]byte) {
				srv, dir, reg, tenants := newTestServer(t)
				id := traceID(29)
				leafIdx, _ := Address(id, srv.hashSeed, testD, testF)
				completeLeaf(t, dir, leafIdx)
				hIn := liveBlock(t, reg, tenants, 29, hourMark(5).Add(10*time.Minute), hourMark(5).Add(20*time.Minute))
				liveBlock(t, reg, tenants, 30, hourMark(50), hourMark(50).Add(time.Minute)) // out of the requested range
				req := &tempopb.BloomGatewayQueryRequest{
					TenantId:          tenantID,
					TraceId:           id,
					StartTimeUnixNano: hourMark(5).Add(10 * time.Minute).UnixNano(),
					EndTimeUnixNano:   hourMark(5).Add(20 * time.Minute).UnixNano(),
				}
				return srv, req, uint32(0), resolveAndSort(t, reg, hIn)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv, req, wantFlags, wantRejected := tt.setup(t)

			resp, err := srv.Query(context.Background(), req)
			require.NoError(t, err)
			assert.Equal(t, wireVersion, resp.Version)
			assert.Equal(t, wantFlags, resp.Flags)

			got := decodeRejected(t, resp.Rejected)
			if len(wantRejected) == 0 {
				assert.Empty(t, got)
			} else {
				assert.Equal(t, wantRejected, got)
			}
		})
	}
}

// TestQuery_SeedFingerprintMatchesHashSeedFingerprint locks in that every
// response -- available or unavailable alike -- carries
// hash.SeedFingerprint(seed) exactly (DESIGN.md § Protocol: lets the QF
// detect seed drift actively).
func TestQuery_SeedFingerprintMatchesHashSeedFingerprint(t *testing.T) {
	srv, _, _, _ := newTestServer(t)

	// Deliberately a nil-leaf (unavailable) response: the fingerprint must
	// still be populated even on the abstention path.
	resp, err := srv.Query(context.Background(), &tempopb.BloomGatewayQueryRequest{TenantId: "tenant-a", TraceId: traceID(99)})
	require.NoError(t, err)
	assert.Equal(t, FlagLeafUnavailable, resp.Flags)
	assert.Equal(t, SeedFingerprint([]byte(testQuerySeed)), resp.SeedFingerprint)
}
