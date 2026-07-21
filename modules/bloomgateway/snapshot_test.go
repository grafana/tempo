package bloomgateway

import (
	"encoding/binary"
	"errors"
	"hash/crc32"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// loadForTest adapts Load's new sink-based streaming API back to the
// "just give me the whole State, CompleteLeaves included" shape this
// file's own tests are written against -- symmetric to the save side's own
// mapLeafSource (snapshot.go): most of this file's assertions only care
// about Load's error, or about the returned *State's non-leaf fields, or
// want to assert on the full decoded leaf set at once
// (assertStatesEqual), none of which needs streaming's one-at-a-time
// memory discipline at this file's own tiny test scale. Collects every
// decoded leaf into an ordinary map and hangs it off the returned
// *State.CompleteLeaves (nil on a Load error, matching Load's own
// contract), so every existing call site here only needed its bare
// "sn.Load(...)" call rewritten to "loadForTest(t, sn, ...)" -- the rest
// of each test is unchanged.
func loadForTest(t *testing.T, sn *Snapshotter, path string, wantD, wantF uint8, wantSeedFingerprint uint64) (*State, error) {
	t.Helper()
	leaves := make(map[uint32]*Leaf)
	state, err := sn.Load(path, wantD, wantF, wantSeedFingerprint, func(idx uint32, leaf *Leaf) {
		leaves[idx] = leaf
	})
	if state != nil {
		state.CompleteLeaves = leaves
	}
	return state, err
}

// newTestState builds a small but representative State exercising every
// field: multiple tokens, offsets (including a large one, to catch any
// accidental truncation), constructing ranges, blocks in every BlockState
// (including Pending, whose StartTime/EndTime/DeletedAt are legitimately
// zero), non-empty complete leaves, and a non-empty tenant snapshot.
func newTestState(t *testing.T) *State {
	t.Helper()

	leafA := NewLeaf()
	leafA.InsertIfAbsent(10, Handle(1))
	leafA.InsertIfAbsent(20, Handle(2))

	tenants := NewTenantSet()
	tenants.AddBlock("tenant-a", Handle(1), hourMark(0), hourMark(0).Add(time.Minute))
	tenants.AddBlock("tenant-b", Handle(2), hourMark(5), hourMark(5).Add(time.Minute))
	tenantSnap, err := tenants.Export()
	require.NoError(t, err)

	return &State{
		D:               testD,
		F:               testF,
		SeedFingerprint: SeedFingerprint([]byte("snapshot-test-seed")),
		Tokens:          []uint32{1, 2, 3, 4294967295},
		Offsets:         map[int32]int64{0: 100, 1: 200, 15: 9223372036854775807},
		CompleteLeaves:  map[uint32]*Leaf{3: leafA, 7: NewLeaf()},
		ConstructingRanges: []LeafRange{
			{Start: 8, End: 9},
			{Start: 100, End: 200},
		},
		Blocks: []Block{
			{UUID: testUUID(t, 1), TenantID: "tenant-a", StartTime: hourMark(0), EndTime: hourMark(0).Add(time.Minute), State: BlockLive, Handle: Handle(1)},
			{UUID: testUUID(t, 2), TenantID: "tenant-b", StartTime: hourMark(5), EndTime: hourMark(5).Add(time.Minute), State: BlockLiveUnsupportedEncoding, Handle: Handle(2)},
			{UUID: testUUID(t, 3), TenantID: "tenant-a", StartTime: hourMark(9), EndTime: hourMark(9).Add(time.Minute), State: BlockDeleted, Handle: Handle(3), DeletedAt: hourMark(10)},
			{UUID: testUUID(t, 4), TenantID: "tenant-c", State: BlockPending, Handle: Handle(4)}, // Pending: StartTime/EndTime/DeletedAt legitimately zero
		},
		Tenants: tenantSnap,
	}
}

// assertStatesEqual compares two States field by field. time.Time fields
// are compared via Equal (the semantic-instant check Go's own docs
// recommend over == or a blanket reflect.DeepEqual), sidestepping any
// internal-representation quirk between a hand-built input time and one
// reconstructed via time.Unix(0, ns).UTC() on decode; every other field is
// compared with testify's ordinary Equal.
func assertStatesEqual(t *testing.T, want, got *State) {
	t.Helper()
	assert.Equal(t, want.D, got.D)
	assert.Equal(t, want.F, got.F)
	assert.Equal(t, want.SeedFingerprint, got.SeedFingerprint)
	assert.Equal(t, want.Tokens, got.Tokens)
	assert.Equal(t, want.Offsets, got.Offsets)
	assert.Equal(t, want.ConstructingRanges, got.ConstructingRanges)
	assert.Equal(t, want.Tenants, got.Tenants)

	require.Len(t, got.Blocks, len(want.Blocks))
	for i := range want.Blocks {
		wb, gb := want.Blocks[i], got.Blocks[i]
		assert.Equal(t, wb.UUID, gb.UUID, "block %d UUID", i)
		assert.Equal(t, wb.TenantID, gb.TenantID, "block %d TenantID", i)
		assert.True(t, wb.StartTime.Equal(gb.StartTime), "block %d StartTime: want %v got %v", i, wb.StartTime, gb.StartTime)
		assert.True(t, wb.EndTime.Equal(gb.EndTime), "block %d EndTime: want %v got %v", i, wb.EndTime, gb.EndTime)
		assert.Equal(t, wb.State, gb.State, "block %d State", i)
		assert.Equal(t, wb.Handle, gb.Handle, "block %d Handle", i)
		if wb.DeletedAt.IsZero() {
			assert.True(t, gb.DeletedAt.IsZero(), "block %d DeletedAt should still be zero", i)
		} else {
			assert.True(t, wb.DeletedAt.Equal(gb.DeletedAt), "block %d DeletedAt: want %v got %v", i, wb.DeletedAt, gb.DeletedAt)
		}
	}

	require.Len(t, got.CompleteLeaves, len(want.CompleteLeaves))
	for idx, wantLeaf := range want.CompleteLeaves {
		gotLeaf, ok := got.CompleteLeaves[idx]
		require.True(t, ok, "leaf %d missing after round trip", idx)
		assert.Equal(t, wantLeaf.fps, gotLeaf.fps, "leaf %d fps", idx)
		assert.Equal(t, wantLeaf.handles, gotLeaf.handles, "leaf %d handles", idx)
	}
}

// TestSnapshot_Load_MismatchTable is deliberately the FIRST real test in
// this file (before the happy-path round trip below): WP17's own stated
// risk is that a false "match" on load would silently load state hashed
// under different parameters, corrupting every subsequent lookup with no
// visible error, so the mismatch table is exercised before anything else
// gets a chance to look like it's working. Covers each of {D mismatch, F
// mismatch, seed-fingerprint mismatch, format-version bump} in isolation.
func TestSnapshot_Load_MismatchTable(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "snapshot.bin")

	const (
		wantD               = testD
		wantF               = testF
		wantSeedFingerprint = uint64(0xABCD)
	)

	sn := NewSnapshotter(newMetrics(prometheus.NewRegistry()))
	require.NoError(t, sn.Save(path, &State{
		D: wantD, F: wantF, SeedFingerprint: wantSeedFingerprint,
		Offsets: map[int32]int64{}, CompleteLeaves: map[uint32]*Leaf{},
		Tenants: TenantSetSnapshot{Buckets: map[string]map[bucketKey][]byte{}},
	}))

	tests := []struct {
		name            string
		d, f            uint8
		seedFingerprint uint64
	}{
		{"d mismatch", wantD + 1, wantF, wantSeedFingerprint},
		{"f mismatch", wantD, wantF + 1, wantSeedFingerprint},
		{"seed fingerprint mismatch", wantD, wantF, wantSeedFingerprint + 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := loadForTest(t, sn, path, tt.d, tt.f, tt.seedFingerprint)
			require.Error(t, err)
			assert.ErrorIs(t, err, ErrSnapshotMismatch)
			assert.False(t, errors.Is(err, ErrSnapshotCorrupt), "a mismatch must never also look like corruption")
		})
	}

	t.Run("format version bump", func(t *testing.T) {
		// Corrupt the on-disk format-version field directly (the 4 bytes
		// right after the 4-byte magic) -- the only way to exercise a
		// version mismatch without a second, hypothetical format version
		// actually existing in this codebase.
		raw, err := os.ReadFile(path)
		require.NoError(t, err)
		bumped := append([]byte(nil), raw...)
		binary.BigEndian.PutUint32(bumped[4:8], snapshotFormatVersion+1)
		bumpedPath := filepath.Join(dir, "bumped.bin")
		require.NoError(t, os.WriteFile(bumpedPath, bumped, 0o600))

		_, err = loadForTest(t, sn, bumpedPath, wantD, wantF, wantSeedFingerprint)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrSnapshotMismatch)
	})
}

// TestSnapshot_Load_CorruptionDistinctFromMismatch is the second test in
// this file, still before the happy-path round trip: every corruption mode
// here must wrap ErrSnapshotCorrupt and must NEVER also satisfy
// errors.Is(err, ErrSnapshotMismatch) -- the two are meant to be
// distinguishable so a caller can log/alert on them differently (a
// mismatch is an expected rollout consequence; corruption usually is not).
func TestSnapshot_Load_CorruptionDistinctFromMismatch(t *testing.T) {
	dir := t.TempDir()
	sn := NewSnapshotter(newMetrics(prometheus.NewRegistry()))
	state := newTestState(t)

	buildValidFile := func(t *testing.T, name string) string {
		t.Helper()
		path := filepath.Join(dir, name)
		require.NoError(t, sn.Save(path, state))
		return path
	}

	assertCorruptNotMismatch := func(t *testing.T, err error) {
		t.Helper()
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrSnapshotCorrupt)
		assert.False(t, errors.Is(err, ErrSnapshotMismatch), "corruption must never also look like a mismatch")
	}

	t.Run("missing file", func(t *testing.T) {
		_, err := loadForTest(t, sn, filepath.Join(dir, "does-not-exist.bin"), state.D, state.F, state.SeedFingerprint)
		assertCorruptNotMismatch(t, err)
	})

	t.Run("empty file", func(t *testing.T) {
		path := filepath.Join(dir, "empty.bin")
		require.NoError(t, os.WriteFile(path, nil, 0o600))
		_, err := loadForTest(t, sn, path, state.D, state.F, state.SeedFingerprint)
		assertCorruptNotMismatch(t, err)
	})

	t.Run("truncated: missing the trailing checksum entirely", func(t *testing.T) {
		path := buildValidFile(t, "valid-for-truncation.bin")
		raw, err := os.ReadFile(path)
		require.NoError(t, err)
		require.Greater(t, len(raw), snapshotHeaderSize+4)

		truncPath := filepath.Join(dir, "truncated.bin")
		require.NoError(t, os.WriteFile(truncPath, raw[:len(raw)-4], 0o600))

		_, err = loadForTest(t, sn, truncPath, state.D, state.F, state.SeedFingerprint)
		assertCorruptNotMismatch(t, err)
	})

	t.Run("bad magic", func(t *testing.T) {
		path := buildValidFile(t, "valid-for-magic.bin")
		raw, err := os.ReadFile(path)
		require.NoError(t, err)
		corrupted := append([]byte(nil), raw...)
		binary.BigEndian.PutUint32(corrupted[:4], 0xdeadbeef)
		corruptPath := filepath.Join(dir, "bad-magic.bin")
		require.NoError(t, os.WriteFile(corruptPath, corrupted, 0o600))

		_, err = loadForTest(t, sn, corruptPath, state.D, state.F, state.SeedFingerprint)
		assertCorruptNotMismatch(t, err)
	})

	t.Run("corrupted trailing checksum", func(t *testing.T) {
		path := buildValidFile(t, "valid-for-checksum.bin")
		raw, err := os.ReadFile(path)
		require.NoError(t, err)
		corrupted := append([]byte(nil), raw...)
		corrupted[len(corrupted)-1] ^= 0xff // flip a bit inside the trailing CRC32 itself
		corruptPath := filepath.Join(dir, "bad-checksum.bin")
		require.NoError(t, os.WriteFile(corruptPath, corrupted, 0o600))

		_, err = loadForTest(t, sn, corruptPath, state.D, state.F, state.SeedFingerprint)
		assertCorruptNotMismatch(t, err)
	})

	t.Run("corrupted body byte: passes the length check, fails the checksum", func(t *testing.T) {
		path := buildValidFile(t, "valid-for-body.bin")
		raw, err := os.ReadFile(path)
		require.NoError(t, err)
		corrupted := append([]byte(nil), raw...)
		corrupted[snapshotHeaderSize+2] ^= 0xff // flip a byte well inside the body, same overall length
		corruptPath := filepath.Join(dir, "bad-body.bin")
		require.NoError(t, os.WriteFile(corruptPath, corrupted, 0o600))

		_, err = loadForTest(t, sn, corruptPath, state.D, state.F, state.SeedFingerprint)
		assertCorruptNotMismatch(t, err)
	})
}

// bodyLengthHeaderOffset mirrors writeSnapshot's own header field order
// (magic, version, D, F, seedFingerprint, THEN bodyLength) -- the same
// arithmetic snapshotHeaderSize itself uses (snapshot.go), one field (8
// bytes) short of it.
const bodyLengthHeaderOffset = 4 + 4 + 1 + 1 + 8

// buildOversizedCountSnapshot legitimately saves precursor (the real Save
// path, so its body's layout is exactly what the real encoder produces --
// nothing hand-rolled to keep in sync by hand), then overwrites the count
// field byteOffset bytes into the body with absurdCount and TRUNCATES the
// body immediately after that field. Nothing legitimate needs to follow
// it: once checkCount's guard fires, sr.err short-circuits every
// subsequent read as a free no-op (snapshotReader's own sticky-error
// discipline), so bytes beyond this point can never be observed
// regardless of whether they exist. bodyLength and the trailing checksum
// are recomputed to match the shorter, corrupted body, so the file stays
// checksum-valid and reaches decodeBody at all -- this test's target is
// the allocation-bounds guard inside decodeBody/streamCompleteLeavesIn,
// not the checksum guard TestSnapshot_Load_CorruptionDistinctFromMismatch
// already covers.
func buildOversizedCountSnapshot(t *testing.T, sn *Snapshotter, precursor *State, byteOffset int, absurdCount uint32) string {
	t.Helper()
	dir := t.TempDir()
	validPath := filepath.Join(dir, "valid.bin")
	require.NoError(t, sn.Save(validPath, precursor))

	raw, err := os.ReadFile(validPath)
	require.NoError(t, err)
	require.Greater(t, len(raw), snapshotHeaderSize+4)
	body := raw[snapshotHeaderSize : len(raw)-4]
	require.GreaterOrEqual(t, len(body), byteOffset+4, "precursor's real body isn't long enough to contain the target count field at the expected offset")

	truncatedBody := append([]byte(nil), body[:byteOffset]...)
	var countBuf [4]byte
	binary.BigEndian.PutUint32(countBuf[:], absurdCount)
	truncatedBody = append(truncatedBody, countBuf[:]...)

	crafted := make([]byte, snapshotHeaderSize+len(truncatedBody)+4)
	copy(crafted, raw[:snapshotHeaderSize])
	binary.BigEndian.PutUint64(crafted[bodyLengthHeaderOffset:snapshotHeaderSize], uint64(len(truncatedBody)))
	copy(crafted[snapshotHeaderSize:], truncatedBody)
	binary.BigEndian.PutUint32(crafted[snapshotHeaderSize+len(truncatedBody):], crc32.ChecksumIEEE(truncatedBody))

	craftedPath := filepath.Join(dir, "crafted.bin")
	require.NoError(t, os.WriteFile(craftedPath, crafted, 0o600))
	return craftedPath
}

// TestSnapshot_Load_OversizedCountRejectedBeforeAllocating is this review
// round's own regression test (2026-07-16, MAJOR finding): decodeBody and
// streamCompleteLeavesIn each read an untrusted count off the wire and
// then make() a slice or map sized DIRECTLY by it (tokens, offsets,
// constructing ranges, blocks, leaf entries, tenants, tenant buckets) --
// readBytes's own before-allocating check (its doc comment) only guards a
// single length-prefixed byte blob, not a count that sizes a LATER make()
// of many elements. Reproduces the reviewer's own finding directly: a
// legitimately-saved, otherwise-minimal snapshot with exactly ONE count
// field overwritten to an absurd value must be rejected as
// ErrSnapshotCorrupt WITHOUT the process growing anywhere near
// proportionally to the claimed count -- verified both by the returned
// error and, directly, by the process's own cumulative allocation during
// the call (runtime.MemStats.TotalAlloc, which only ever grows and is
// therefore immune to GC-timing flakiness, unlike HeapAlloc).
func TestSnapshot_Load_OversizedCountRejectedBeforeAllocating(t *testing.T) {
	sn := NewSnapshotter(newMetrics(prometheus.NewRegistry()))
	seed := SeedFingerprint([]byte("snapshot-test-seed"))

	// emptyState's body is exactly 24 bytes: six top-level uint32 counts
	// (tokens, offsets, constructing ranges, blocks, leaves, tenants),
	// each legitimately zero, encodeBody's own field order -- so each
	// sits at a fixed, independently-derived offset (a multiple of 4)
	// with nothing variable-length before it to shift its position.
	emptyState := &State{
		D: testD, F: testF, SeedFingerprint: seed,
		Offsets: map[int32]int64{},
		Tenants: TenantSetSnapshot{Buckets: map[string]map[bucketKey][]byte{}},
	}
	// oneLeafState has exactly one complete leaf with zero real entries,
	// so its own numEntries count field exists (nested inside the leaf
	// section, unlike the six above) at a fixed offset: right after
	// tokens/offsets/ranges/blocks/numLeaves=1/idx.
	oneLeafState := &State{
		D: testD, F: testF, SeedFingerprint: seed,
		Offsets:        map[int32]int64{},
		CompleteLeaves: map[uint32]*Leaf{0: NewLeaf()},
		Tenants:        TenantSetSnapshot{Buckets: map[string]map[bucketKey][]byte{}},
	}
	// oneEmptyTenantState has exactly one tenant (empty ID) with zero
	// buckets, so numBuckets -- otherwise per-tenant and variably placed
	// -- exists at a fixed offset: an empty tenant ID encodes as a bare
	// 4-byte zero length prefix (no content bytes), immediately after
	// numTenants=1's own 4 bytes, followed by this tenant's numBuckets.
	oneEmptyTenantState := &State{
		D: testD, F: testF, SeedFingerprint: seed,
		Offsets: map[int32]int64{},
		Tenants: TenantSetSnapshot{Buckets: map[string]map[bucketKey][]byte{"": {}}},
	}

	const absurdCount = 50_000_000 // the reviewer's own reproduction figure

	tests := []struct {
		name       string
		precursor  *State
		byteOffset int // offset of the target count field within the body
	}{
		{"tokens", emptyState, 0},
		{"offsets", emptyState, 4},
		{"constructing ranges", emptyState, 8},
		{"blocks", emptyState, 12},
		{"tenants", emptyState, 20},
		{"leaf entries", oneLeafState, 24},
		{"tenant buckets", oneEmptyTenantState, 28},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path := buildOversizedCountSnapshot(t, sn, tc.precursor, tc.byteOffset, absurdCount)

			var before, after runtime.MemStats
			runtime.ReadMemStats(&before)
			_, err := loadForTest(t, sn, path, testD, testF, seed)
			runtime.ReadMemStats(&after)

			require.Error(t, err)
			assert.ErrorIs(t, err, ErrSnapshotCorrupt)

			// A real (unguarded) allocation for 50,000,000 elements, at
			// any of this test's field strides (4-28 bytes/element), is
			// hundreds of MB at minimum -- bounding observed growth to a
			// few MiB leaves huge margin above ordinary test/runtime
			// noise while remaining utterly incompatible with that
			// allocation having been attempted.
			const maxAllowedGrowth = 8 << 20 // 8 MiB
			grew := after.TotalAlloc - before.TotalAlloc
			assert.Less(t, grew, uint64(maxAllowedGrowth), "Load allocated %d bytes rejecting an absurd %q count -- the bound must reject BEFORE allocating, not after", grew, tc.name)
		})
	}
}

// TestSnapshot_SaveLoad_RoundTrip is the happy path -- deliberately placed
// after the mismatch table and corruption tests above, per this WP's own
// stated risk ("test the mismatch table before the happy-path round-trip").
func TestSnapshot_SaveLoad_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "snapshot.bin")

	want := newTestState(t)
	sn := NewSnapshotter(newMetrics(prometheus.NewRegistry()))
	require.NoError(t, sn.Save(path, want))

	got, err := loadForTest(t, sn, path, want.D, want.F, want.SeedFingerprint)
	require.NoError(t, err)
	assertStatesEqual(t, want, got)
}

// TestSnapshot_ConstructingRangesRoundTripAsRangesOnly is the named test
// plan item: ConstructingRanges round-trip as bare ranges, never as
// partial leaf content. The type itself ([]LeafRange, not map[uint32]*Leaf
// or similar) already makes carrying leaf payloads for them a compile-time
// impossibility; this test locks in the round trip and the accompanying
// absence of any leaf payload for those ranges.
func TestSnapshot_ConstructingRangesRoundTripAsRangesOnly(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "snapshot.bin")

	want := &State{
		D: testD, F: testF, SeedFingerprint: 42,
		Offsets:            map[int32]int64{},
		CompleteLeaves:     map[uint32]*Leaf{},
		ConstructingRanges: []LeafRange{{Start: 3, End: 5}, {Start: 100, End: 250}},
		Tenants:            TenantSetSnapshot{Buckets: map[string]map[bucketKey][]byte{}},
	}

	sn := NewSnapshotter(newMetrics(prometheus.NewRegistry()))
	require.NoError(t, sn.Save(path, want))

	got, err := loadForTest(t, sn, path, want.D, want.F, want.SeedFingerprint)
	require.NoError(t, err)
	assert.Equal(t, want.ConstructingRanges, got.ConstructingRanges)
	assert.Empty(t, got.CompleteLeaves, "no leaf payload must appear for a range that was only ever constructing")
}

// TestSnapshot_Save_AtomicallyReplacesExistingFile confirms the
// temp-file-then-rename mechanics: a second Save at the same path fully
// replaces the first's contents (not appends/corrupts it), and leaves no
// stray ".tmp-*" file behind in the directory.
func TestSnapshot_Save_AtomicallyReplacesExistingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "snapshot.bin")
	sn := NewSnapshotter(newMetrics(prometheus.NewRegistry()))

	first := &State{
		D: testD, F: testF, SeedFingerprint: 1,
		Offsets: map[int32]int64{}, CompleteLeaves: map[uint32]*Leaf{},
		Tenants: TenantSetSnapshot{Buckets: map[string]map[bucketKey][]byte{}},
	}
	require.NoError(t, sn.Save(path, first))

	second := newTestState(t)
	require.NoError(t, sn.Save(path, second))

	got, err := loadForTest(t, sn, path, second.D, second.F, second.SeedFingerprint)
	require.NoError(t, err)
	assertStatesEqual(t, second, got)

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	assert.Len(t, entries, 1, "no stray temp file should remain after Save")
	assert.Equal(t, "snapshot.bin", entries[0].Name())
}

// TestSnapshot_Save_UpdatesMetrics is a smoke test for the two metrics
// Save is documented to touch (metrics.go: snapshotDurationSeconds is
// specifically "duration of one snapshot save", snapshotBytes the size of
// the file just written) -- Load deliberately updates neither (no
// "snapshot load duration" series exists in DESIGN.md's own § Metrics
// table).
func TestSnapshot_Save_UpdatesMetrics(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := newMetrics(reg)
	sn := NewSnapshotter(m)

	path := filepath.Join(t.TempDir(), "snapshot.bin")
	require.NoError(t, sn.Save(path, newTestState(t)))

	count, err := testutil.GatherAndCount(reg, "tempo_bloom_gateway_snapshot_duration_seconds")
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	gotBytes := testutil.ToFloat64(m.snapshotBytes)
	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, float64(info.Size()), gotBytes)
}

// flippingLeafSource is a LeafSource test double that reports a chosen
// subset of indexes as no-longer-complete when Clone is called --
// deterministically reproducing the race window streamCompleteLeaves must
// handle (an index buildSnapshotState's own dir.Range pass collected as
// complete, but shed -- or otherwise no longer complete -- by the time
// Save's LeafSource actually clones it) without depending on real
// goroutine timing.
type flippingLeafSource struct {
	indexes []uint32
	leaves  map[uint32]*Leaf
	flipped map[uint32]bool // indexes to report as no-longer-complete
}

func (s *flippingLeafSource) Indexes() []uint32 { return s.indexes }

func (s *flippingLeafSource) Clone(idx uint32) (*Leaf, bool) {
	if s.flipped[idx] {
		return nil, false
	}
	return s.leaves[idx], true
}

// TestSnapshot_Save_SkipsLeafThatFlipsAwayFromCompleteBeforeClone is the
// named test plan item for the 2026-07-16 streaming-save fix (DESIGN.md §
// Snapshots amendment): an index collected as complete must be skipped --
// never serialized under stale or zeroed content -- if it is no longer
// complete by the time Save's LeafSource actually clones it, and the
// resulting snapshot must still load cleanly. This doubles as the
// regression test for the leaf-count placeholder/patch and the
// re-derived-from-disk checksum (writeSnapshot's own doc comment): if
// either were wrong, either Load would fail outright on this file, or the
// skipped index's absence wouldn't be reflected correctly.
func TestSnapshot_Save_SkipsLeafThatFlipsAwayFromCompleteBeforeClone(t *testing.T) {
	path := filepath.Join(t.TempDir(), "snapshot.bin")

	leafA := NewLeaf()
	leafA.InsertIfAbsent(10, Handle(1))
	leafB := NewLeaf()
	leafB.InsertIfAbsent(20, Handle(2))

	src := &flippingLeafSource{
		indexes: []uint32{3, 7, 12},
		leaves:  map[uint32]*Leaf{3: leafA, 7: leafB, 12: NewLeaf()},
		flipped: map[uint32]bool{7: true}, // simulates idx 7 being shed between collection and clone
	}

	state := &State{
		D: testD, F: testF, SeedFingerprint: 4242,
		Offsets: map[int32]int64{},
		Leaves:  src,
		Tenants: TenantSetSnapshot{Buckets: map[string]map[bucketKey][]byte{}},
	}

	sn := NewSnapshotter(newMetrics(prometheus.NewRegistry()))
	require.NoError(t, sn.Save(path, state))

	got, err := loadForTest(t, sn, path, state.D, state.F, state.SeedFingerprint)
	require.NoError(t, err, "a skipped index must not corrupt the leaf count or checksum")

	require.Len(t, got.CompleteLeaves, 2, "only the two non-flipped indexes must be present")
	gotA, ok := got.CompleteLeaves[3]
	require.True(t, ok)
	assert.Equal(t, leafA.fps, gotA.fps)
	assert.Equal(t, leafA.handles, gotA.handles)

	gotEmpty, ok := got.CompleteLeaves[12]
	require.True(t, ok)
	assert.Empty(t, gotEmpty.fps)

	_, stillThere := got.CompleteLeaves[7]
	assert.False(t, stillThere, "index 7 flipped away from complete before Clone and must be entirely absent from the loaded snapshot")
}

// TestSnapshot_LargeStateSaveLoadTiming builds a CI-budget-friendly (not
// DESIGN.md's reference ~15-20 GiB/instance) but non-trivial State,
// exercises a real Save/Load round trip, and logs the actual wall time and
// on-disk size -- the plan's own test-plan item ("measures actual
// Save/Load wall time and bytes-on-disk for snapshot_duration_seconds/
// snapshot_bytes sizing"), reported as a regular test (t.Logf), not a
// go-tool benchmark: WP17 is not one of the implementation plan's three
// NAMED hot-path benchmarks (BenchmarkLeafInsert, BenchmarkQuery,
// BenchmarkSweepPass), unlike WP15/WP16.
func TestSnapshot_LargeStateSaveLoadTiming(t *testing.T) {
	const (
		numLeaves      = 200
		entriesPerLeaf = referenceEntriesPerLeaf // leaf_bench_test.go's DESIGN.md § Sizing reference population
	)

	rng := rand.New(rand.NewSource(7))
	leaves := make(map[uint32]*Leaf, numLeaves)
	for i := uint32(0); i < numLeaves; i++ {
		leaves[i] = newPopulatedLeaf(entriesPerLeaf, rng)
	}

	numBlocks := numLeaves * 4
	blocks := make([]Block, numBlocks)
	for i := range blocks {
		blocks[i] = Block{
			UUID:      testUUID(t, i),
			TenantID:  "bench-tenant",
			StartTime: time.Unix(int64(i), 0).UTC(),
			EndTime:   time.Unix(int64(i)+60, 0).UTC(),
			State:     BlockLive,
			Handle:    Handle(i + 1),
		}
	}

	state := &State{
		D: 20, F: 16, SeedFingerprint: 123456789,
		Tokens:         []uint32{1, 2, 3},
		Offsets:        map[int32]int64{0: 100, 1: 200},
		CompleteLeaves: leaves,
		Blocks:         blocks,
		Tenants:        TenantSetSnapshot{Buckets: map[string]map[bucketKey][]byte{}},
	}

	path := filepath.Join(t.TempDir(), "large-snapshot.bin")
	sn := NewSnapshotter(newMetrics(prometheus.NewRegistry()))

	saveStart := time.Now()
	require.NoError(t, sn.Save(path, state))
	saveElapsed := time.Since(saveStart)

	info, err := os.Stat(path)
	require.NoError(t, err)

	loadStart := time.Now()
	got, err := loadForTest(t, sn, path, state.D, state.F, state.SeedFingerprint)
	loadElapsed := time.Since(loadStart)
	require.NoError(t, err)

	require.Len(t, got.CompleteLeaves, numLeaves)
	require.Len(t, got.Blocks, numBlocks)

	totalEntries := numLeaves * entriesPerLeaf
	t.Logf("snapshot timing (CI-scaled: %d leaves x %d entries = %d entries, %d blocks): save=%s load=%s bytes=%d (%.2f B/entry)",
		numLeaves, entriesPerLeaf, totalEntries, numBlocks, saveElapsed, loadElapsed, info.Size(), float64(info.Size())/float64(totalEntries))
}
