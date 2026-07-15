package bloomgateway

import (
	"context"
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"

	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/vparquet3"
	"github.com/grafana/tempo/tempodb/encoding/vparquet4"
)

// fakeBackendReader is a hand-rolled backend.Reader fixture (repo
// convention: no mock generators), covering only the two methods
// ReconstructionQueue ever calls directly -- Tenants and TenantIndex.
// FetchAndApplyBlockColumn never reaches any OTHER Reader method for a
// block whose meta resolves to an encoding lacking encoding.
// TraceIDProjector -- every fixture block in this file deliberately uses a
// real, registered, non-vparquet5 VersionedEncoding (vparquet3/vparquet4)
// so the unsupported-encoding path is exercised without standing up a real
// on-disk vparquet5 block (that path is WP8's own test's job, tempodb/
// encoding/vparquet5/block_traceid_projection_test.go). Every other
// backend.Reader method panics if ever called, so a test that accidentally
// reaches one fails loudly instead of silently returning a wrong value.
type fakeBackendReader struct {
	mu sync.Mutex

	tenantIDs []string
	indexes   map[string]*backend.TenantIndex

	tenantsCalls     int
	tenantIndexCalls map[string]int

	// tenantsErr, if set, makes Tenants() fail — used to drive processBatch
	// into a hard error after step 1 (BeginConstructing) has already run.
	tenantsErr error

	// indexGate, if non-nil, is received from before every TenantIndex call
	// returns -- lets a test hold a batch "mid-enumeration" open long
	// enough to inject concurrent activity before letting it proceed
	// (TestReconstruction_LiveWritesAccumulateDuringConstructing).
	indexGate <-chan struct{}
}

func newFakeBackendReader() *fakeBackendReader {
	return &fakeBackendReader{
		indexes:          make(map[string]*backend.TenantIndex),
		tenantIndexCalls: make(map[string]int),
	}
}

// setTenantIndex registers tenantID's TenantIndex fixture, adding tenantID
// to the Tenants() list on first use.
func (f *fakeBackendReader) setTenantIndex(tenantID string, idx *backend.TenantIndex) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.indexes[tenantID]; !ok {
		f.tenantIDs = append(f.tenantIDs, tenantID)
	}
	f.indexes[tenantID] = idx
}

func (f *fakeBackendReader) blockTenantIndexOn(gate <-chan struct{}) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.indexGate = gate
}

func (f *fakeBackendReader) Tenants(context.Context) ([]string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.tenantsCalls++
	if f.tenantsErr != nil {
		return nil, f.tenantsErr
	}
	return append([]string(nil), f.tenantIDs...), nil
}

func (f *fakeBackendReader) TenantIndex(_ context.Context, tenantID string) (*backend.TenantIndex, error) {
	f.mu.Lock()
	gate := f.indexGate
	f.mu.Unlock()
	if gate != nil {
		<-gate
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	f.tenantIndexCalls[tenantID]++
	idx, ok := f.indexes[tenantID]
	if !ok {
		return nil, backend.ErrDoesNotExist
	}
	return idx, nil
}

func (f *fakeBackendReader) tenantsCallCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.tenantsCalls
}

func (f *fakeBackendReader) tenantIndexCallCount(tenantID string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.tenantIndexCalls[tenantID]
}

// Every method below is unused by ReconstructionQueue for a fixture block
// whose encoding lacks encoding.TraceIDProjector (every fixture in this
// file) -- panicking makes an accidental real call fail loudly rather than
// silently returning a zero value a test could misinterpret as success.
func (f *fakeBackendReader) Read(context.Context, string, uuid.UUID, string, *backend.CacheInfo) ([]byte, error) {
	panic("fakeBackendReader: Read not implemented")
}

func (f *fakeBackendReader) StreamReader(context.Context, string, uuid.UUID, string) (io.ReadCloser, int64, error) {
	panic("fakeBackendReader: StreamReader not implemented")
}

func (f *fakeBackendReader) ReadRange(context.Context, string, uuid.UUID, string, uint64, []byte, *backend.CacheInfo) error {
	panic("fakeBackendReader: ReadRange not implemented")
}

func (f *fakeBackendReader) Blocks(context.Context, string) ([]uuid.UUID, []uuid.UUID, error) {
	panic("fakeBackendReader: Blocks not implemented")
}

func (f *fakeBackendReader) BlockMeta(context.Context, uuid.UUID, string) (*backend.BlockMeta, error) {
	panic("fakeBackendReader: BlockMeta not implemented")
}

func (f *fakeBackendReader) Find(context.Context, backend.KeyPath, backend.FindFunc) error {
	panic("fakeBackendReader: Find not implemented")
}

func (f *fakeBackendReader) HasNoCompactFlag(context.Context, uuid.UUID, string) (bool, error) {
	panic("fakeBackendReader: HasNoCompactFlag not implemented")
}

func (f *fakeBackendReader) Shutdown() {}

var _ backend.Reader = (*fakeBackendReader)(nil)

// fakeRewinder is a hand-rolled PositionRewinder fixture giving tests
// precise, deterministic control over the rewind-replay-catchup sequence
// (AMENDMENT A6: inject a fake PositionRewinder for exactly this kind of
// ordering assertion, rather than contorting kfake's own timing).
//
// By default Rewind does NOT disturb current -- current already equals
// whatever processBatch captures as preRewind, so waitForCatchUp is
// trivially satisfied and a test that doesn't care about catch-up timing
// (e.g. TestReconstruction_CoalescesBatchIntoOneColumnPass) never blocks.
// A test that DOES care (TestReconstruction_LiveWritesAccumulateDuring
// Constructing) calls enableSimulatedLag, which makes Rewind actually move
// current back to atOrBefore's values -- exactly like a real rewind -- so
// the test can then drive the catch-up moment explicitly via advance.
type fakeRewinder struct {
	mu sync.Mutex

	current     map[int32]int64
	atOrBefore  map[int32]int64
	simulateLag bool
	rewindCalls int
	lastRewind  map[int32]int64
}

func newFakeRewinder(current map[int32]int64) *fakeRewinder {
	return &fakeRewinder{current: cloneOffsets(current), atOrBefore: map[int32]int64{}}
}

func (f *fakeRewinder) setAtOrBefore(offsets map[int32]int64) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.atOrBefore = cloneOffsets(offsets)
}

func (f *fakeRewinder) enableSimulatedLag() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.simulateLag = true
}

func (f *fakeRewinder) CurrentFetchOffsets() map[int32]int64 {
	f.mu.Lock()
	defer f.mu.Unlock()
	return cloneOffsets(f.current)
}

func (f *fakeRewinder) OffsetsAtOrBefore(context.Context, time.Time) (map[int32]int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return cloneOffsets(f.atOrBefore), nil
}

func (f *fakeRewinder) Rewind(offsets map[int32]int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.rewindCalls++
	f.lastRewind = cloneOffsets(offsets)
	if f.simulateLag {
		for p, at := range offsets {
			f.current[p] = at
		}
	}
	return nil
}

// advance simulates replay progress: current[partition] = offset, as if
// the consumer had now fetched up through offset-1. Only meaningful after
// enableSimulatedLag; otherwise current is never behind in the first
// place.
func (f *fakeRewinder) advance(partition int32, offset int64) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.current[partition] = offset
}

func (f *fakeRewinder) rewindCallCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.rewindCalls
}

func (f *fakeRewinder) lastRewindOffsets() map[int32]int64 {
	f.mu.Lock()
	defer f.mu.Unlock()
	return cloneOffsets(f.lastRewind)
}

func cloneOffsets(m map[int32]int64) map[int32]int64 {
	out := make(map[int32]int64, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

var _ PositionRewinder = (*fakeRewinder)(nil)

// ownsEverything returns an ownedRangesFn (ReconstructionQueue's own
// ownership-check dependency, bloomgateway.go's currentOwnedRanges in
// production) reporting the full [0, 2^d) range as owned, always -- the
// ownership fixture for tests exercising reconstruction mechanics unrelated
// to ownership races, so the claim-time/flip-time ownership scoping added
// for the ring-churn resurrection fix never trims what these tests enqueue.
func ownsEverything(d uint8) func() ([]LeafRange, error) {
	total := uint32(1) << d
	return func() ([]LeafRange, error) {
		return []LeafRange{{Start: 0, End: total}}, nil
	}
}

// dynamicOwnership is a test fixture giving direct, mutable, concurrency-safe
// control over what a ReconstructionQueue's ownedRangesFn reports -- used to
// simulate runOwnershipWatch's ring-driven ownership changing mid-batch (or
// between a failed batch and its retry) without standing up a real ring.
type dynamicOwnership struct {
	mu     sync.Mutex
	ranges []LeafRange
}

func newDynamicOwnership(initial []LeafRange) *dynamicOwnership {
	return &dynamicOwnership{ranges: append([]LeafRange(nil), initial...)}
}

func (d *dynamicOwnership) set(ranges []LeafRange) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.ranges = append([]LeafRange(nil), ranges...)
}

func (d *dynamicOwnership) fn() ([]LeafRange, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]LeafRange(nil), d.ranges...), nil
}

// blockMetaFixture builds a minimal *backend.BlockMeta covering exactly
// the fields FetchAndApplyBlockColumn/traceIDProjectorFor read: Version
// (the encoding-lookup key), BlockID/TenantID, and the block's time range.
func blockMetaFixture(id backend.UUID, tenantID string, tr timeRange, version string) *backend.BlockMeta {
	return &backend.BlockMeta{
		BlockID:   id,
		TenantID:  tenantID,
		Version:   version,
		StartTime: tr.start,
		EndTime:   tr.end,
	}
}

// findTraceIDForLeaf returns the first traceID(n) (events_test.go) that
// hashes to leafIdx under seed/d/f -- lets a test construct fixtures whose
// leaf routing is known in advance without duplicating hash.go's
// internals. Guaranteed to terminate quickly at this package's small
// testD (roughly 1-in-2^testD odds per candidate).
func findTraceIDForLeaf(t testing.TB, seed uint64, d, f uint8, leafIdx uint32) []byte {
	t.Helper()
	for n := 0; n < 1_000_000; n++ {
		id := traceID(n)
		if idx, _ := Address(id, seed, d, f); idx == leafIdx {
			return id
		}
	}
	t.Fatalf("no trace ID found hashing to leaf %d after 1,000,000 attempts", leafIdx)
	return nil
}

// promtestGaugeVec reads one label's current value out of a GaugeVec --
// the GaugeVec analogue of events_test.go's promtestCounter.
func promtestGaugeVec(t *testing.T, vec *prometheus.GaugeVec, label string) float64 {
	t.Helper()
	g, err := vec.GetMetricWithLabelValues(label)
	require.NoError(t, err)
	return testutil.ToFloat64(g)
}

// TestReconstruction_UnsupportedEncodingBlockNeverRejectable is the named
// deliverable for invariant #10 (§7, §0 D7): a block whose real backend
// meta resolves to an encoding lacking encoding.TraceIDProjector must be
// registered (so it is visible/observable) but never enter A_T -- so it
// can never be rejected, only ever "unknown, must be searched".
func TestReconstruction_UnsupportedEncodingBlockNeverRejectable(t *testing.T) {
	applier, _, reg, tenants := newTestApplier(t, true)

	tr := testTimeRange()
	id := testUUID(t, 1)
	meta := blockMetaFixture(id, "tenant-a", tr, vparquet3.VersionString)
	reader := newFakeBackendReader() // Tenants/TenantIndex unused: FetchAndApplyBlockColumn is called directly here, not through the queue

	applied, err := FetchAndApplyBlockColumn(context.Background(), reader, meta, applier, applier.metrics)
	require.NoError(t, err)
	assert.False(t, applied, "an unsupported-encoding block is never counted as applied")

	state, ok := reg.State(id)
	require.True(t, ok, "the block must still be registered -- observable, just never rejectable")
	assert.Equal(t, BlockLiveUnsupportedEncoding, state)

	block, ok := reg.LookupUUID(id)
	require.True(t, ok)
	assert.False(t, handleInWindow(tenants, "tenant-a", block.Handle), "an unsupported-encoding block must never enter A_T")
	assert.Equal(t, float64(1), promtestGaugeVec(t, applier.metrics.unsupportedEncodingBlocks, "tenant-a"))

	// A window query over the block's own time range must never reject it:
	// absent from A_T, it rides the "unknown to gateway, must search" path.
	window := tenants.Window("tenant-a", tr.start, tr.end)
	assert.False(t, window.Contains(uint32(block.Handle)))
}

// TestReconstruction_LiveBlockDemotedWhenUnsupported is AMENDMENT A1's named
// deliverable: a block made Live via a LIVE Kafka Add (encoding-agnostic,
// pre-enumerated trace IDs -- DESIGN.md §2) can reach BlockLive/A_T before
// this instance ever learns its real parquet encoding is unsupported. When
// reconstruction/reconciliation later encounters it via
// FetchAndApplyBlockColumn, it must be DEMOTED (never left rejectable, but
// also never simply rejected/dropped outright).
func TestReconstruction_LiveBlockDemotedWhenUnsupported(t *testing.T) {
	applier, _, reg, tenants := newTestApplier(t, true)

	tr := testTimeRange()
	id := testUUID(t, 2)

	require.NoError(t, applier.ApplyAddChunk(chunkFor(id, "tenant-a", tr, 0, 1, traceIDs(2))))
	state, ok := reg.State(id)
	require.True(t, ok)
	require.Equal(t, BlockLive, state, "precondition: the live Add must have made the block Live")
	block, ok := reg.LookupUUID(id)
	require.True(t, ok)
	require.True(t, handleInWindow(tenants, "tenant-a", block.Handle), "precondition: the live Add must have entered A_T")

	// Reconstruction later encounters the SAME block's real backend meta
	// and finds its encoding unsupported.
	meta := blockMetaFixture(id, "tenant-a", tr, vparquet4.VersionString)
	reader := newFakeBackendReader()

	applied, err := FetchAndApplyBlockColumn(context.Background(), reader, meta, applier, applier.metrics)
	require.NoError(t, err)
	assert.False(t, applied)

	state, ok = reg.State(id)
	require.True(t, ok)
	assert.Equal(t, BlockLiveUnsupportedEncoding, state, "a block already Live via a live Add must be DEMOTED, never left as-is nor rejected outright")
	assert.False(t, handleInWindow(tenants, "tenant-a", block.Handle), "demotion must remove the block from A_T")

	window := tenants.Window("tenant-a", tr.start, tr.end)
	assert.False(t, window.Contains(uint32(block.Handle)), "a demoted block must never be rejected")
}

// TestReconstruction_CoalescesBatchIntoOneColumnPass: however many separate
// ranges are enqueued (covering every leaf between them), RunBatch must
// claim and process them as exactly ONE batch -- one tenant enumeration,
// one TenantIndex read per tenant, one rewind -- never once per range
// (DESIGN.md § Reconstruction / § Scaling: "the queue coalesces all
// pending ranges... into a single column pass").
// TestReconstruction_BatchFailureRevertsConstructingLeaves is the regression
// guard for the Phase C finding that a batch failing after BeginConstructing
// stranded its leaves in LeafConstructing forever — which would permanently
// hold the readiness gate shut (it requires zero constructing leaves) and
// prevent any retry (BeginConstructing only fires from LeafNil). A hard error
// in the batch (here: Tenants() fails, right after step 1) must leave every
// leaf this batch started back in LeafNil.
func TestReconstruction_BatchFailureRevertsConstructingLeaves(t *testing.T) {
	applier, dir, _, _ := newTestApplier(t, false /* leaves start nil so BeginConstructing fires */)

	reader := newFakeBackendReader()
	reader.tenantsErr = errors.New("simulated object-store outage")

	rewinder := newFakeRewinder(map[int32]int64{0: 0})
	q := NewReconstructionQueue(dir, applier, rewinder, reader, ReconstructionConfig{Concurrency: 2}, ownsEverything(testD), rate.NewLimiter(rate.Inf, 0), applier.metrics, log.NewNopLogger())

	total := uint32(1) << testD
	q.Enqueue([]LeafRange{{Start: 0, End: total}})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err := q.RunBatch(ctx)
	require.Error(t, err, "batch must surface the Tenants() failure")

	for idx := uint32(0); idx < total; idx++ {
		require.Equalf(t, LeafNil, dir.State(idx), "leaf %d must be reverted to nil after a failed batch, not stranded constructing", idx)
	}
}

// TestReconstruction_ShedMidBatchNeverResurrectsOrCompletesShedLeaves is the
// bug report's own "Shed-mid-batch" test, for a live incident (tempo-dev-02,
// 2026-07-15): a reconstruction batch claimed at startup kept re-marking
// already-shed leaves constructing and inserting into them, because the
// batch trusted its own claimed `ranges` for the whole pass instead of
// re-checking ownership as it went. This drives one batch over range R,
// changes ownership mid-pass in the two distinct ways runOwnershipWatch
// (bloomgateway.go) can leave a leaf behind, and asserts neither sub-range
// is ever re-marked constructing or served as complete:
//
//   - shedDirectly: Directory.Shed is called AND the leaf drops out of
//     ownedRangesFn -- the ordinary path once runOwnershipWatch's own tick
//     has caught up to a ring change. Directory.InsertLive's existing
//     nil-check alone already stops accumulation the instant this fires;
//     this sub-range mainly confirms processBatch does not undo that.
//   - unownedNotYetShed: the leaf drops out of ownedRangesFn but Shed is
//     deliberately NOT called -- the narrow window between a ring change
//     and the next ownership tick actually observing it. The leaf is
//     still genuinely LeafConstructing here, so Directory.Complete's own
//     state check cannot refuse it; only this fix's flip-time ownership
//     re-check (processBatch step 6) can.
func TestReconstruction_ShedMidBatchNeverResurrectsOrCompletesShedLeaves(t *testing.T) {
	applier, dir, _, _ := newTestApplier(t, false /* leaves start nil */)

	tr := testTimeRange()
	total := uint32(1) << testD // 16 leaves

	keepOwned := LeafRange{Start: 12, End: 16}
	shedDirectly := LeafRange{Start: 0, End: 4}
	unownedNotYetShed := LeafRange{Start: 4, End: 12}

	reader := newFakeBackendReader()
	// One (unsupported-encoding) block, purely so there is a TenantIndex
	// call this test can gate to hold the batch open mid-pass -- its
	// content is irrelevant here, since the accumulation assertions below
	// drive Directory.InsertLive directly (TestReconstruction_
	// LiveWritesAccumulateDuringConstructing's own pattern), not block
	// application.
	reader.setTenantIndex("tenant-a", &backend.TenantIndex{
		CreatedAt: tr.start,
		Meta: []*backend.BlockMeta{
			blockMetaFixture(testUUID(t, 1), "tenant-a", tr, vparquet3.VersionString),
		},
	})
	gate := make(chan struct{})
	reader.blockTenantIndexOn(gate)

	rewinder := newFakeRewinder(map[int32]int64{0: 0})
	rewinder.setAtOrBefore(map[int32]int64{0: 0})

	own := newDynamicOwnership([]LeafRange{{Start: 0, End: total}}) // claim time: all of R owned
	q := NewReconstructionQueue(dir, applier, rewinder, reader, ReconstructionConfig{Concurrency: 2}, own.fn, rate.NewLimiter(rate.Inf, 0), applier.metrics, log.NewNopLogger())
	q.Enqueue([]LeafRange{{Start: 0, End: total}})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	done := make(chan struct{})
	var stats BatchStats
	var runErr error
	go func() {
		stats, runErr = q.RunBatch(ctx)
		close(done)
	}()

	require.Eventually(t, func() bool {
		for idx := range total {
			if dir.State(idx) != LeafConstructing {
				return false
			}
		}
		return true
	}, 2*time.Second, 5*time.Millisecond, "step 1 must claim the whole batch before step 2 blocks on the gate")

	// While still constructing, shedDirectly accepts live writes exactly
	// like any other constructing leaf (DESIGN.md § Leaf lifecycle).
	require.True(t, dir.InsertLive(shedDirectly.Start, 1, Handle(1)), "a constructing leaf must accumulate live writes")

	// Simulate ownership moving on mid-pass, exactly as runOwnershipWatch
	// would across two different points in its own tick cadence.
	for idx := shedDirectly.Start; idx < shedDirectly.End; idx++ {
		dir.Shed(idx)
	}
	own.set([]LeafRange{keepOwned}) // both shedDirectly and unownedNotYetShed are now un-owned per the ring

	// The instant it is Shed, the leaf must stop accumulating -- a write
	// that would have grown it is dropped, not merely ignored later.
	assert.False(t, dir.InsertLive(shedDirectly.Start, 2, Handle(2)), "a shed leaf must drop writes immediately, not just at flip time")

	close(gate) // let the column pass proceed

	<-done
	require.NoError(t, runErr)

	for idx := shedDirectly.Start; idx < shedDirectly.End; idx++ {
		assert.Equal(t, LeafNil, dir.State(idx), "leaf %d was Shed mid-batch; must never be re-marked constructing or completed", idx)
	}
	for idx := unownedNotYetShed.Start; idx < unownedNotYetShed.End; idx++ {
		assert.Equal(t, LeafNil, dir.State(idx), "leaf %d lost ring ownership mid-batch even though never directly Shed; the flip-time re-check must Abandon it, not complete it", idx)
	}
	for idx := keepOwned.Start; idx < keepOwned.End; idx++ {
		assert.Equal(t, LeafComplete, dir.State(idx), "leaf %d was never shed; must complete normally", idx)
	}
	assert.EqualValues(t, total, stats.LeavesStarted, "step 1 claimed the whole original batch before ownership changed mid-pass")
}

// TestReconstruction_RetryAfterOwnershipChangeDoesNotResurrectShedLeaves
// pins down the OTHER half of the same incident: a batch that fails and is
// retried (RunBatch's own re-enqueue of its unmodified input, see Run) must
// re-scope its retry to whatever is CURRENTLY owned, not blindly re-claim
// the stale superset it started with. Without processBatch's claim-time
// ownership filter, the retry's own step 1 would call BeginConstructing
// again on leaves that ownership had since moved away from -- silently
// resurrecting them into constructing, and (absent a second failure) all
// the way to complete, serving leaves this instance no longer owns.
func TestReconstruction_RetryAfterOwnershipChangeDoesNotResurrectShedLeaves(t *testing.T) {
	applier, dir, _, _ := newTestApplier(t, false /* leaves start nil */)

	total := uint32(1) << testD // 16 leaves
	fullRange := []LeafRange{{Start: 0, End: total}}
	noLongerOwned := LeafRange{Start: 0, End: 8}
	stillOwned := LeafRange{Start: 8, End: 16}

	reader := newFakeBackendReader()
	reader.tenantsErr = errors.New("simulated object-store outage")

	rewinder := newFakeRewinder(map[int32]int64{0: 0})
	rewinder.setAtOrBefore(map[int32]int64{0: 0})

	own := newDynamicOwnership(fullRange) // claim time: everything owned
	q := NewReconstructionQueue(dir, applier, rewinder, reader, ReconstructionConfig{Concurrency: 2}, own.fn, rate.NewLimiter(rate.Inf, 0), applier.metrics, log.NewNopLogger())
	q.Enqueue(fullRange)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// First attempt: step 1 claims all 16 leaves, then Tenants() fails --
	// the deferred cleanup Abandons all 16 back to nil, and RunBatch
	// re-enqueues the ORIGINAL, now-stale full range (see RunBatch).
	_, err := q.RunBatch(ctx)
	require.Error(t, err, "first attempt must surface the simulated Tenants() failure")
	for idx := range total {
		require.Equalf(t, LeafNil, dir.State(idx), "leaf %d must be back to nil after the first attempt's failure", idx)
	}
	require.Equal(t, 1, q.PendingRanges(), "RunBatch must re-enqueue the failed attempt's original range for retry")

	// Ownership moves on before the retry: only stillOwned remains this
	// instance's. Also clear the simulated outage and give the batch a
	// real (empty) tenant index so the retry can actually succeed.
	own.set([]LeafRange{stillOwned})
	reader.tenantsErr = nil

	stats, err := q.RunBatch(ctx)
	require.NoError(t, err, "second attempt must succeed now that Tenants() no longer fails")

	for idx := noLongerOwned.Start; idx < noLongerOwned.End; idx++ {
		assert.Equal(t, LeafNil, dir.State(idx), "leaf %d is no longer owned; the retry must not have resurrected it via the stale full-range re-enqueue", idx)
	}
	for idx := stillOwned.Start; idx < stillOwned.End; idx++ {
		assert.Equal(t, LeafComplete, dir.State(idx), "leaf %d is still owned; the retry must claim and complete it", idx)
	}
	assert.EqualValues(t, stillOwned.End-stillOwned.Start, stats.LeavesStarted, "the retry must only claim the still-owned half, not the stale full range it was re-enqueued with")
}

func TestReconstruction_CoalescesBatchIntoOneColumnPass(t *testing.T) {
	applier, dir, _, _ := newTestApplier(t, false /* leaves start nil so BeginConstructing actually fires */)

	tr := testTimeRange()
	reader := newFakeBackendReader()
	reader.setTenantIndex("tenant-a", &backend.TenantIndex{
		CreatedAt: tr.start,
		Meta: []*backend.BlockMeta{
			blockMetaFixture(testUUID(t, 1), "tenant-a", tr, vparquet3.VersionString),
			blockMetaFixture(testUUID(t, 2), "tenant-a", tr, vparquet4.VersionString),
		},
	})

	rewinder := newFakeRewinder(map[int32]int64{0: 3}) // simulateLag left off: Rewind won't disturb current, so catch-up is trivially satisfied
	rewinder.setAtOrBefore(map[int32]int64{0: 0})

	q := NewReconstructionQueue(dir, applier, rewinder, reader, ReconstructionConfig{Concurrency: 4}, ownsEverything(testD), rate.NewLimiter(rate.Inf, 0), applier.metrics, log.NewNopLogger())

	// Several separate Enqueue calls, deliberately covering every leaf
	// between them.
	total := uint32(1) << testD
	q.Enqueue([]LeafRange{{Start: 0, End: total / 2}})
	q.Enqueue([]LeafRange{{Start: total / 2, End: total}})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	stats, err := q.RunBatch(ctx)
	require.NoError(t, err)

	assert.Equal(t, 2, stats.Ranges, "both enqueued ranges must be claimed by the same batch")
	assert.Equal(t, int(total), stats.LeavesStarted, "every leaf must have started constructing")
	assert.Equal(t, 2, stats.Blocks)
	assert.Equal(t, 2, stats.NotApplied, "both fixtures are unsupported-encoding")
	assert.Equal(t, 0, stats.Applied)
	assert.Equal(t, 0, stats.Failed)

	assert.Equal(t, 1, reader.tenantsCallCount(), "tenants must be enumerated exactly once for the whole batch")
	assert.Equal(t, 1, reader.tenantIndexCallCount("tenant-a"), "the tenant index must be read exactly once, not once per range")
	assert.Equal(t, 1, rewinder.rewindCallCount(), "exactly one rewind for the whole coalesced batch")
	assert.Equal(t, map[int32]int64{0: 0}, rewinder.lastRewindOffsets(), "the rewind must target exactly what OffsetsAtOrBefore resolved")

	for idx := uint32(0); idx < total; idx++ {
		assert.Equal(t, LeafComplete, dir.State(idx), "leaf %d must have flipped to complete", idx)
	}
}

// TestReconstruction_LiveWritesAccumulateDuringConstructing is the
// completeness invariant's (§7 #1) most direct exercise at this WP's level,
// complementing WP10's unit-level Directory test: while a batch's column
// pass is genuinely still in flight, a live write lands on the very leaf
// under construction; the leaf must NOT flip to complete until BOTH the
// pass finishes AND the consumer has caught back up past the pre-rewind
// position -- and the live write must still be present once it does.
func TestReconstruction_LiveWritesAccumulateDuringConstructing(t *testing.T) {
	applier, dir, reg, _ := newTestApplier(t, false /* leaf starts nil */)

	tr := testTimeRange()
	reader := newFakeBackendReader()
	reader.setTenantIndex("tenant-a", &backend.TenantIndex{
		CreatedAt: tr.start,
		Meta: []*backend.BlockMeta{
			blockMetaFixture(testUUID(t, 1), "tenant-a", tr, vparquet3.VersionString),
		},
	})
	gate := make(chan struct{})
	reader.blockTenantIndexOn(gate) // holds the "column pass" open until the test says otherwise

	rewinder := newFakeRewinder(map[int32]int64{0: 5})
	rewinder.enableSimulatedLag()
	rewinder.setAtOrBefore(map[int32]int64{0: 1})

	q := NewReconstructionQueue(dir, applier, rewinder, reader, ReconstructionConfig{Concurrency: 2}, ownsEverything(testD), rate.NewLimiter(rate.Inf, 0), applier.metrics, log.NewNopLogger())

	const leafIdx = uint32(3)
	q.Enqueue([]LeafRange{{Start: leafIdx, End: leafIdx + 1}})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	batchDone := make(chan struct{})
	var stats BatchStats
	var batchErr error
	go func() {
		stats, batchErr = q.RunBatch(ctx)
		close(batchDone)
	}()

	require.Eventually(t, func() bool { return dir.State(leafIdx) == LeafConstructing }, 2*time.Second, 5*time.Millisecond,
		"the batch must mark the leaf constructing before doing anything else")

	// A live write lands on the constructing leaf while the pass is still
	// blocked inside TenantIndex.
	const liveFP = uint16(42)
	liveBlock, _ := reg.GetOrCreate(testUUID(t, 500), "tenant-a", tr.start, tr.end)
	require.True(t, dir.InsertLive(leafIdx, liveFP, liveBlock.Handle))

	close(gate) // let the pass itself proceed

	require.Never(t, func() bool { return dir.State(leafIdx) == LeafComplete }, 300*time.Millisecond, 10*time.Millisecond,
		"the leaf must not flip to complete before the post-rewind catch-up finishes, even once the pass itself is done")

	rewinder.advance(0, 5) // simulate replay catching back up past preRewind

	require.Eventually(t, func() bool { return dir.State(leafIdx) == LeafComplete }, 2*time.Second, 5*time.Millisecond,
		"the leaf must flip to complete once BOTH the pass and the catch-up have finished")
	<-batchDone
	require.NoError(t, batchErr)
	assert.Equal(t, 1, stats.LeavesStarted)
	assert.Equal(t, 1, stats.NotApplied)

	handles, ok := dir.Lookup(leafIdx, liveFP)
	require.True(t, ok, "the now-complete leaf must be servable")
	assert.Contains(t, handles, liveBlock.Handle, "the live write that landed during construction must still be present once complete")
}
