package bloomgateway

import (
	"encoding/binary"
	"errors"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
			_, err := sn.Load(path, tt.d, tt.f, tt.seedFingerprint)
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

		_, err = sn.Load(bumpedPath, wantD, wantF, wantSeedFingerprint)
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
		_, err := sn.Load(filepath.Join(dir, "does-not-exist.bin"), state.D, state.F, state.SeedFingerprint)
		assertCorruptNotMismatch(t, err)
	})

	t.Run("empty file", func(t *testing.T) {
		path := filepath.Join(dir, "empty.bin")
		require.NoError(t, os.WriteFile(path, nil, 0o600))
		_, err := sn.Load(path, state.D, state.F, state.SeedFingerprint)
		assertCorruptNotMismatch(t, err)
	})

	t.Run("truncated: missing the trailing checksum entirely", func(t *testing.T) {
		path := buildValidFile(t, "valid-for-truncation.bin")
		raw, err := os.ReadFile(path)
		require.NoError(t, err)
		require.Greater(t, len(raw), snapshotHeaderSize+4)

		truncPath := filepath.Join(dir, "truncated.bin")
		require.NoError(t, os.WriteFile(truncPath, raw[:len(raw)-4], 0o600))

		_, err = sn.Load(truncPath, state.D, state.F, state.SeedFingerprint)
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

		_, err = sn.Load(corruptPath, state.D, state.F, state.SeedFingerprint)
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

		_, err = sn.Load(corruptPath, state.D, state.F, state.SeedFingerprint)
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

		_, err = sn.Load(corruptPath, state.D, state.F, state.SeedFingerprint)
		assertCorruptNotMismatch(t, err)
	})
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

	got, err := sn.Load(path, want.D, want.F, want.SeedFingerprint)
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

	got, err := sn.Load(path, want.D, want.F, want.SeedFingerprint)
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

	got, err := sn.Load(path, second.D, second.F, second.SeedFingerprint)
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
	got, err := sn.Load(path, state.D, state.F, state.SeedFingerprint)
	loadElapsed := time.Since(loadStart)
	require.NoError(t, err)

	require.Len(t, got.CompleteLeaves, numLeaves)
	require.Len(t, got.Blocks, numBlocks)

	totalEntries := numLeaves * entriesPerLeaf
	t.Logf("snapshot timing (CI-scaled: %d leaves x %d entries = %d entries, %d blocks): save=%s load=%s bytes=%d (%.2f B/entry)",
		numLeaves, entriesPerLeaf, totalEntries, numBlocks, saveElapsed, loadElapsed, info.Size(), float64(info.Size())/float64(totalEntries))
}
