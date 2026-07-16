package bloomgateway

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"time"

	"github.com/grafana/tempo/tempodb/backend"
)

// snapshotFormatVersion is the on-disk format's own version, checked
// before D/F/seed fingerprint on every Load (DESIGN.md § Snapshots: "any
// mismatch in format version, D, F, or seed fingerprint discards the
// snapshot").
const snapshotFormatVersion uint32 = 1

// snapshotMagic is a fixed sentinel at the very start of every snapshot
// file, checked before anything else -- distinguishes "this isn't a
// bloom-gateway snapshot at all" (ErrSnapshotCorrupt) from "this IS one,
// just an incompatible version" (ErrSnapshotMismatch).
const snapshotMagic uint32 = 0x424c4d31 // ASCII "BLM1"

// snapshotHeaderSize is the fixed-size prefix Load always reads first,
// before deciding whether to read anything else: magic(4) +
// formatVersion(4) + D(1) + F(1) + SeedFingerprint(8) + bodyLength(8).
const snapshotHeaderSize = 4 + 4 + 1 + 1 + 8 + 8

// ErrSnapshotMismatch is returned by Load when the snapshot's format
// version, D, F, or seed fingerprint doesn't match what the caller
// expects -- the caller's cue to discard and reconstruct (DESIGN.md §
// Snapshots). Load returns this after reading ONLY the small fixed
// header, without reading the (potentially many-GiB) body at all, so a
// mismatch is always cheap to detect and never risks decoding state that
// is about to be discarded anyway.
var ErrSnapshotMismatch = errors.New("bloomgateway: snapshot format/d/f/seed-fingerprint mismatch")

// ErrSnapshotCorrupt is returned by Load for every failure mode OTHER than
// a mismatch: a missing/unreadable file, a bad magic number, a truncated
// body, a checksum failure, or a malformed field. Deliberately
// distinguishable from ErrSnapshotMismatch (via errors.Is) even though
// DESIGN.md's own operational handling for both is "discard, reconstruct"
// -- a mismatch is an EXPECTED consequence of a D/F/seed/format-changing
// rollout, worth a routine log line; corruption usually indicates a
// disk/volume problem worth a more alarming one.
var ErrSnapshotCorrupt = errors.New("bloomgateway: snapshot corrupt or truncated")

// LeafSource lets Save stream a snapshot's complete-leaf section one leaf
// at a time, instead of requiring every complete leaf to already be cloned
// into an in-memory map before Save is even called. This exists
// specifically because of a 2026-07-16 production incident (DESIGN.md §
// Snapshots amendment): buildSnapshotState (bloomgateway.go) used to
// CloneLeaf every owned leaf into a map up front, and at production scale
// (~2.1M owned leaves) that doubled live heap and OOM-killed the pod
// mid-assembly, on every snapshot tick, before Save ever got a chance to
// write anything. bloomgateway.go's directoryLeafSource is the production
// implementation (streaming straight from the live Directory); this
// file's own mapLeafSource (below) is the trivial adapter for callers that
// already hold every leaf in memory (State.CompleteLeaves -- Load's own
// return shape, and every State this file's tests build by hand).
type LeafSource interface {
	// Indexes returns the sorted, ascending leaf indexes that were
	// LeafComplete at collection time -- cheap to gather (a Directory.
	// Range pass over indexes only, no cloning).
	Indexes() []uint32
	// Clone returns idx's leaf, deep-copied and safe to serialize
	// independently of any concurrent writer, plus whether idx is STILL
	// complete right now. Save calls this exactly once per index,
	// immediately before serializing that leaf, and discards the result
	// immediately after -- peak additional memory is one leaf, never all
	// of them. ok=false (an index that flipped away from LeafComplete
	// between collection and this call -- an ownership change shedding it,
	// most plausibly) means Save skips idx entirely: safe by the
	// completeness invariant (DESIGN.md § Leaf lifecycle) -- an owned leaf
	// simply missing from the snapshot is re-enqueued for reconstruction
	// on the next load (reconcileStartup), and nothing was ever served, or
	// claimed to have been saved, from anything but complete state.
	Clone(idx uint32) (leaf *Leaf, ok bool)
}

// mapLeafSource adapts a plain, already-hydrated map[uint32]*Leaf --
// State.CompleteLeaves' own shape -- to LeafSource, for every Save caller
// that isn't buildSnapshotState (this file's own tests, chiefly). Clone
// never actually re-clones: a caller that already holds a private map has
// no concurrent writer to race, and every index Indexes() returns came
// from this exact map, so ok is unconditionally true.
type mapLeafSource map[uint32]*Leaf

func (m mapLeafSource) Indexes() []uint32 {
	idxs := make([]uint32, 0, len(m))
	for idx := range m {
		idxs = append(idxs, idx)
	}
	slices.Sort(idxs)
	return idxs
}

func (m mapLeafSource) Clone(idx uint32) (*Leaf, bool) {
	leaf, ok := m[idx]
	return leaf, ok
}

// leavesSource resolves state's leaf data to the single LeafSource
// encodeBody actually iterates, regardless of which of State's two
// mutually-exclusive representations the caller populated (see their own
// field comments on State): Leaves, if set, is preferred -- it is the more
// memory-conscious of the two, and buildSnapshotState never sets
// CompleteLeaves alongside it, so in practice there is no real ambiguity.
func leavesSource(state *State) LeafSource {
	if state.Leaves != nil {
		return state.Leaves
	}
	return mapLeafSource(state.CompleteLeaves)
}

// LeafSink receives each complete leaf Load decodes, immediately after
// decoding it -- Load's streaming restore counterpart to Save's own
// LeafSource above (2026-07-16 restore-side amendment; see Load's own doc
// comment for the production incident this responds to: a 7.44 GiB
// snapshot's ENTIRE complete-leaf section used to be decoded into one
// map -- doubling live heap right as the restored instance's Kafka replay
// burst and reconciliation startup also ramped up -- before reconcile
// Startup ever got a chance to import a single leaf). Called once per
// (idx, leaf) pair in the snapshot's complete-leaf section, in ascending
// idx order (encodeBody's own sorted-write contract) though a caller must
// not rely on that ordering. The leaf handed to Sink is never referenced
// by Load again afterward, so an implementation that stores the pointer
// directly (bloomgateway.go's reconcileStartup, via Directory.Complete) is
// safe -- no copy needed on the way in, mirroring how CloneLeaf's own copy
// on the way OUT (save side) is what makes THAT direction safe instead.
type LeafSink func(idx uint32, leaf *Leaf)

// State is the caller-assembled snapshot payload (DESIGN.md § Snapshots).
// Only complete leaves are included, matching "the leaf directory with
// leaf payloads for complete leaves"; ConstructingRanges round-trip as
// bare ranges only (re-enqueued to the reconstruction queue on load,
// DESIGN.md: "constructing/pending ranges (re-enqueued on load)") -- NEVER
// as their necessarily-partial (and therefore unsafe-to-serve, § Design
// constraints) leaf contents. Assembling a State from the live structures,
// and reconciling a loaded one against current ring ownership, are the
// orchestrator's job (WP20) -- this file is save/load mechanics only.
//
// AMENDMENT A2 note: a block's transient chunk-arrival bitset (the
// event-applier's per-(uuid, chunk_count) bookkeeping of which AddChunk
// indexes have been seen for a still-Pending block) is DELIBERATELY not a
// field here and is never persisted. A restart therefore always loses
// in-progress chunk-arrival bookkeeping for any block that was still
// Pending at snapshot time -- this is not a gap this file works around.
// AMENDMENT A2 assigns the healing path to reconciliation instead: its
// repair-Add condition is "in the tenant index AND (absent from the
// registry OR present with State == BlockPending) past the grace window",
// which re-fetches a stuck-Pending block's trace-ID column and re-applies
// it as a synthetic, single-chunk (chunk_index=0, chunk_count=1) Add. That
// synthetic Add completes immediately and idempotently regardless of
// whatever partial chunk grouping existed before the restart, so no
// bitset needs to survive a restart for correctness -- only for a modest
// amount of avoided re-fetching, which is reconciliation's normal job
// already, not a new failure mode this file introduces.
type State struct {
	D, F            uint8
	SeedFingerprint uint64
	Tokens          []uint32
	Offsets         map[int32]int64
	// CompleteLeaves is populated only by Save callers that already hold
	// every complete leaf in memory (nothing production-sized ever does --
	// see Leaves below) and by this package's own hand-built test States.
	// Load (2026-07-16 restore-side streaming amendment) never populates
	// this anymore -- decodeBody hands each decoded leaf to the caller's
	// LeafSink one at a time instead, for the exact memory reason Leaves
	// exists on the save side (see LeafSink's own doc comment).
	CompleteLeaves map[uint32]*Leaf
	// Leaves is buildSnapshotState's own streaming alternative to
	// CompleteLeaves -- see LeafSource's own doc comment for why it
	// exists. Load never populates this (decodeBody only ever fills
	// CompleteLeaves), and no hand-built test State in this package needs
	// it either.
	Leaves             LeafSource
	ConstructingRanges []LeafRange
	Blocks             []Block
	Tenants            TenantSetSnapshot
}

// Snapshotter serializes/deserializes State to/from local disk.
type Snapshotter struct {
	metrics *metrics
}

// NewSnapshotter builds a Snapshotter.
func NewSnapshotter(m *metrics) *Snapshotter {
	return &Snapshotter{metrics: m}
}

// Save serializes state to path, replacing any existing file atomically:
// write to a temp file in the same directory, then rename into place. A
// crash mid-Save must never leave a half-written file AT path, since Load
// would then see corruption on the very next restart -- exactly when a
// clean snapshot matters most.
//
// Callers must have already stopped mutating the structures state was
// assembled from before calling Save (DESIGN.md § Snapshots consistency:
// "v1 pauses the worker pool between events for the duration of the
// serialization"). Save itself does not know about or touch the worker
// pool -- it only serializes whatever state it's handed, trusting the
// caller's pause. The one exception is state.Leaves (see LeafSource's own
// doc comment): the worker-pool pause does NOT stop the sweep,
// reconstruction, or reconciliation writers, so a leaf Save is about to
// stream can still change out from under it -- handled per-index at
// stream time, not here.
func (sn *Snapshotter) Save(path string, state *State) error {
	start := time.Now()

	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("bloomgateway: snapshot: creating temp file: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath) // no-op once the rename below succeeds; cleans up on any earlier error return

	if err := writeSnapshot(tmp, state); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("bloomgateway: snapshot: closing temp file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("bloomgateway: snapshot: renaming into place: %w", err)
	}

	if sn.metrics != nil {
		sn.metrics.snapshotDurationSeconds.Observe(time.Since(start).Seconds())
		if info, statErr := os.Stat(path); statErr == nil {
			sn.metrics.snapshotBytes.Set(float64(info.Size()))
		}
	}
	return nil
}

// writeSnapshot writes the full on-disk format to f: header (magic,
// format version, D, F, seed fingerprint, body length), body, then a
// trailing CRC32 of the body.
//
// f must support Seek (Save always passes a real *os.File): bodyLength is
// written as an 8-byte placeholder and fixed up once the body's actual
// length is known, and the body itself streams DIRECTLY to f (through a
// buffered writer, coalescing millions of small field writes into large
// syscalls) rather than being buffered whole in memory first. This is
// deliberate, not an over-optimization: State's structures (leaves,
// registry, tenant sets) are already the live, in-memory serving state at
// DESIGN.md's reference ~15-20 GiB/instance scale (§ Sizing) -- buffering
// a second full copy just to learn its length before writing the header
// would transiently double this instance's memory footprint on every
// snapshot.
//
// 2026-07-16 amendment (DESIGN.md § Snapshots): the body's checksum is
// deliberately NOT computed incrementally alongside this write anymore (an
// earlier revision did, via io.MultiWriter). The leaf-entry count inside
// the body (encodeBody's own leaf section, streamCompleteLeaves below) is
// itself a placeholder patched in place once every collected index has
// actually been streamed -- a LeafSource can skip an index that flipped
// away from complete since collection, so the true count isn't known
// until that loop finishes. Patching bytes an incremental hash has
// ALREADY summed would desync the hash from what actually ends up on
// disk. Computing the checksum in one pass AFTER every patch is applied
// -- by reading the finalized body back from f rather than hashing it as
// it's written -- sidesteps that entirely, at the cost of one extra
// sequential disk read no larger than the body itself (bounded, constant
// extra memory: io.CopyN's own internal buffer, not a State-sized one).
func writeSnapshot(f *os.File, state *State) error {
	hw := &snapshotWriter{w: f}
	hw.writeUint32(snapshotMagic)
	hw.writeUint32(snapshotFormatVersion)
	hw.writeUint8(state.D)
	hw.writeUint8(state.F)
	hw.writeUint64(state.SeedFingerprint)

	bodyLengthOffset, err := f.Seek(0, io.SeekCurrent)
	if err != nil {
		return fmt.Errorf("bloomgateway: snapshot: seeking past header: %w", err)
	}
	hw.writeUint64(0) // placeholder; fixed up below once the body's length is known
	if hw.err != nil {
		return fmt.Errorf("bloomgateway: snapshot: writing header: %w", hw.err)
	}

	bodyStart, err := f.Seek(0, io.SeekCurrent)
	if err != nil {
		return fmt.Errorf("bloomgateway: snapshot: seeking to body start: %w", err)
	}

	// The body is the write-heavy part of this format -- at DESIGN.md's
	// reference scale, millions of individual 1-8 byte field writes
	// (encodeBody calls snapshotWriter's primitives once per token,
	// offset, leaf entry, ...). Writing each one straight to f would cost
	// one syscall per field; bufio.Writer coalesces them into large
	// writes. buffered and f are threaded through encodeBody (rather than
	// just the sw wrapper) purely so streamCompleteLeaves can Flush and
	// Seek f directly for its own leaf-count fixup -- see its own doc
	// comment.
	buffered := bufio.NewWriter(f)
	bw := &snapshotWriter{w: buffered}
	if err := encodeBody(bw, buffered, f, state); err != nil {
		return fmt.Errorf("bloomgateway: snapshot: encoding body: %w", err)
	}
	if bw.err != nil {
		return fmt.Errorf("bloomgateway: snapshot: encoding body: %w", bw.err)
	}
	if err := buffered.Flush(); err != nil {
		return fmt.Errorf("bloomgateway: snapshot: flushing body: %w", err)
	}

	bodyEnd, err := f.Seek(0, io.SeekCurrent)
	if err != nil {
		return fmt.Errorf("bloomgateway: snapshot: seeking to body end: %w", err)
	}

	if _, err := f.Seek(bodyLengthOffset, io.SeekStart); err != nil {
		return fmt.Errorf("bloomgateway: snapshot: seeking back to fix up body length: %w", err)
	}
	fixup := &snapshotWriter{w: f}
	fixup.writeUint64(uint64(bodyEnd - bodyStart))
	if fixup.err != nil {
		return fmt.Errorf("bloomgateway: snapshot: fixing up body length: %w", fixup.err)
	}

	// Checksum: computed fresh from the finalized on-disk body (bodyStart
	// through bodyEnd), never from an incremental hash -- see this
	// function's own doc comment for why.
	if _, err := f.Seek(bodyStart, io.SeekStart); err != nil {
		return fmt.Errorf("bloomgateway: snapshot: seeking to body start for checksum: %w", err)
	}
	checksum := crc32.NewIEEE()
	if _, err := io.CopyN(checksum, f, bodyEnd-bodyStart); err != nil {
		return fmt.Errorf("bloomgateway: snapshot: computing checksum: %w", err)
	}

	if _, err := f.Seek(bodyEnd, io.SeekStart); err != nil {
		return fmt.Errorf("bloomgateway: snapshot: seeking to end for trailer: %w", err)
	}
	trailer := &snapshotWriter{w: f}
	trailer.writeUint32(checksum.Sum32())
	if trailer.err != nil {
		return fmt.Errorf("bloomgateway: snapshot: writing checksum trailer: %w", trailer.err)
	}
	return nil
}

// Load decodes path, streaming each complete leaf to sink as it is decoded
// rather than materializing all of them in one map first -- 2026-07-16
// restore-side amendment (DESIGN.md § Snapshots), mirroring Save's own
// streaming redesign (LeafSource's doc comment above tells that half of
// the story). Production incident this responds to: a 7.44 GiB snapshot's
// entire complete-leaf section (2,098,208 leaves) used to be decoded into
// one map before reconcileStartup ever imported a single leaf into the
// live Directory -- on top of the map itself, every decoded leaf's fps/
// handles were FRESH allocations distinct from the map's own bucket
// overhead, so the transient peak during Load was never just "the file's
// size again", it was closer to double the eventual steady-state leaf data.
// Worse, that garbage does not vanish the instant Load returns: it is
// merely unreferenced, waiting for the next GC cycle, and stays physically
// resident for however long that takes -- which is exactly when the
// restored instance's Kafka consumer (16 partitions, tens of minutes of
// buffered replay) and reconciliation startup ALSO ramp up their own
// allocations. Streaming removes the transient entirely: a leaf is
// decoded, handed to sink, and immediately eligible for collection with
// nothing else of comparable size competing for the same headroom.
//
// Returns ErrSnapshotMismatch (format version, D, F, or seed fingerprint)
// after reading ONLY the small fixed header -- WITHOUT reading the
// (potentially many-GiB) body at all, so a mismatch is always cheap to
// detect. Any other failure (missing file, bad magic, truncated body,
// checksum mismatch, malformed field) wraps ErrSnapshotCorrupt instead,
// distinguishable via errors.Is.
//
// Two sequential passes over the body, both bounded, constant-memory:
//  1. verifyBodyChecksum hashes the body as it streams past (io.CopyN into
//     a running CRC32), never buffering it whole, and compares against the
//     trailing checksum.
//  2. Only if that passes, Load seeks back to the body's start and decodes
//     the SAME bytes, calling sink once per complete leaf as it's decoded.
//
// This ordering is load-bearing, not incidental. A single pass that
// decoded-and-called-sink WHILE ALSO computing the checksum would not
// discover the file was corrupt until EOF -- by which point sink may
// already have handed millions of leaves to a caller that imports them
// directly into a live, already-serving Directory (reconcileStartup's own
// production sink). The RETIRED, non-streaming Load avoided this for free
// by buffering the whole body before decoding anything, so corruption was
// always caught before a single field was ever interpreted; streaming
// loses that "buffer first" safety net, so this method re-earns the same
// guarantee explicitly, at the cost of one extra sequential read over the
// body, rather than silently trading it away for the memory win.
func (sn *Snapshotter) Load(path string, wantD, wantF uint8, wantSeedFingerprint uint64, sink LeafSink) (*State, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("%w: opening %s: %v", ErrSnapshotCorrupt, path, err)
	}
	defer f.Close()

	headerBuf := make([]byte, snapshotHeaderSize)
	if _, err := io.ReadFull(f, headerBuf); err != nil {
		return nil, fmt.Errorf("%w: reading header: %v", ErrSnapshotCorrupt, err)
	}
	hr := &snapshotReader{r: &io.LimitedReader{R: bytes.NewReader(headerBuf), N: int64(len(headerBuf))}}

	magic := hr.readUint32()
	version := hr.readUint32()
	gotD := hr.readUint8()
	gotF := hr.readUint8()
	gotSeedFingerprint := hr.readUint64()
	bodyLength := hr.readUint64()
	if hr.err != nil {
		return nil, fmt.Errorf("%w: decoding header: %v", ErrSnapshotCorrupt, hr.err)
	}

	if magic != snapshotMagic {
		return nil, fmt.Errorf("%w: bad magic %#x, want %#x", ErrSnapshotCorrupt, magic, snapshotMagic)
	}

	if version != snapshotFormatVersion || gotD != wantD || gotF != wantF || gotSeedFingerprint != wantSeedFingerprint {
		return nil, fmt.Errorf("%w: got (format_version=%d, d=%d, f=%d, seed_fingerprint=%d), want (format_version=%d, d=%d, f=%d, seed_fingerprint=%d)",
			ErrSnapshotMismatch, version, gotD, gotF, gotSeedFingerprint, snapshotFormatVersion, wantD, wantF, wantSeedFingerprint)
	}

	bodyStart, err := f.Seek(0, io.SeekCurrent)
	if err != nil {
		return nil, fmt.Errorf("%w: seeking to body start: %v", ErrSnapshotCorrupt, err)
	}

	// Pass 1: verify, in full, before any decoding is attempted -- see this
	// method's own doc comment for why.
	if err := verifyBodyChecksum(f, bodyLength); err != nil {
		return nil, err
	}

	// Pass 2: decode + stream, only now that the body is confirmed
	// byte-for-byte intact. bufio coalesces the same many-small-reads
	// pattern encodeBody's writer side already documented needing to
	// coalesce on the way out; io.LimitedReader bounds the total to
	// bodyLength (our OWN trusted header field, not any nested/corrupted
	// value) regardless of what garbage might follow on disk, and exposes
	// its own remaining-bytes count so snapshotReader.readBytes can keep
	// bounds-checking every length-prefixed field before allocating (see
	// its own doc comment) even without a single in-memory buffer to check
	// a slice length against anymore.
	if _, err := f.Seek(bodyStart, io.SeekStart); err != nil {
		return nil, fmt.Errorf("%w: seeking back to body start for decode: %v", ErrSnapshotCorrupt, err)
	}
	sr := &snapshotReader{r: &io.LimitedReader{R: bufio.NewReader(f), N: int64(bodyLength)}}
	state := decodeBody(sr, gotD, gotF, gotSeedFingerprint, sink)
	if sr.err != nil {
		return nil, fmt.Errorf("%w: decoding body: %v", ErrSnapshotCorrupt, sr.err)
	}
	return state, nil
}

// verifyBodyChecksum reads exactly bodyLength body bytes plus the trailing
// 4-byte CRC32 from f, starting at f's current position, hashing the body
// as it streams past (io.CopyN's own internal buffer -- fixed-size,
// unrelated to bodyLength) rather than buffering it whole, then compares
// the result against the stored trailer. See Load's own doc comment for
// why this whole pass exists as a separate, first step before any
// decoding: it is what lets Load keep its "never interpret a byte of a
// corrupt snapshot" guarantee while still never holding the body in memory
// twice (or even once) the way the retired implementation did.
func verifyBodyChecksum(f *os.File, bodyLength uint64) error {
	checksum := crc32.NewIEEE()
	if _, err := io.CopyN(checksum, f, int64(bodyLength)); err != nil {
		return fmt.Errorf("%w: reading body for checksum verification: %v", ErrSnapshotCorrupt, err)
	}

	var trailerBuf [4]byte
	if _, err := io.ReadFull(f, trailerBuf[:]); err != nil {
		return fmt.Errorf("%w: reading checksum trailer: %v", ErrSnapshotCorrupt, err)
	}

	wantChecksum := binary.BigEndian.Uint32(trailerBuf[:])
	if gotChecksum := checksum.Sum32(); gotChecksum != wantChecksum {
		return fmt.Errorf("%w: checksum mismatch: want %#x, got %#x", ErrSnapshotCorrupt, wantChecksum, gotChecksum)
	}
	return nil
}

// encodeBody writes every State field below the header, in a fixed order
// decodeBody must mirror exactly. Map-keyed fields (Offsets,
// Tenants.Buckets) are written in a stable sorted order purely so two Save
// calls over equal input produce byte-identical output; decodeBody does
// not depend on that order in any way.
//
// buffered and f are needed alongside sw purely so streamCompleteLeaves
// (the leaf section, below) can Flush and Seek f directly for its own
// leaf-count fixup; every other field here goes through sw exactly as
// every earlier revision of this function did.
func encodeBody(sw *snapshotWriter, buffered *bufio.Writer, f *os.File, state *State) error {
	sw.writeUint32(uint32(len(state.Tokens)))
	for _, tok := range state.Tokens {
		sw.writeUint32(tok)
	}

	partitions := make([]int32, 0, len(state.Offsets))
	for p := range state.Offsets {
		partitions = append(partitions, p)
	}
	sort.Slice(partitions, func(i, j int) bool { return partitions[i] < partitions[j] })
	sw.writeUint32(uint32(len(partitions)))
	for _, p := range partitions {
		sw.writeUint32(uint32(p)) // int32 -> uint32 bit-pattern round trip; reversed on decode
		sw.writeInt64(state.Offsets[p])
	}

	sw.writeUint32(uint32(len(state.ConstructingRanges)))
	for _, r := range state.ConstructingRanges {
		sw.writeUint32(r.Start)
		sw.writeUint32(r.End)
	}

	sw.writeUint32(uint32(len(state.Blocks)))
	for _, b := range state.Blocks {
		// backend.UUID.Marshal never errors: it always marshals a fixed
		// 16-byte value (google/uuid's own MarshalBinary contract).
		uuidBytes, _ := b.UUID.Marshal()
		sw.writeBytes(uuidBytes)
		sw.writeString(b.TenantID)
		sw.writeTime(b.StartTime)
		sw.writeTime(b.EndTime)
		sw.writeUint8(uint8(b.State))
		sw.writeUint32(uint32(b.Handle))
		sw.writeTime(b.DeletedAt)
	}

	if err := streamCompleteLeaves(sw, buffered, f, leavesSource(state)); err != nil {
		return err
	}

	tenantIDs := make([]string, 0, len(state.Tenants.Buckets))
	for id := range state.Tenants.Buckets {
		tenantIDs = append(tenantIDs, id)
	}
	sort.Strings(tenantIDs)
	sw.writeUint32(uint32(len(tenantIDs)))
	for _, id := range tenantIDs {
		buckets := state.Tenants.Buckets[id]
		sw.writeString(id)

		bucketKeys := make([]bucketKey, 0, len(buckets))
		for k := range buckets {
			bucketKeys = append(bucketKeys, k)
		}
		sort.Slice(bucketKeys, func(i, j int) bool { return bucketKeys[i] < bucketKeys[j] })
		sw.writeUint32(uint32(len(bucketKeys)))
		for _, k := range bucketKeys {
			sw.writeInt64(int64(k))
			sw.writeBlob(buckets[k])
		}
	}
	return nil
}

// streamCompleteLeaves writes the on-disk complete-leaf section -- a count
// followed by (idx, entries...) per leaf, byte-for-byte the same shape
// this file always used -- but sources each leaf from src ONE AT A TIME
// (LeafSource's own doc comment), so Save never needs more than one
// leaf's worth of additional memory regardless of how many complete
// leaves this instance owns (the 2026-07-16 OOM this streaming save
// replaces, DESIGN.md § Snapshots amendment).
//
// Unlike every other section in encodeBody, this one's count cannot be
// written correctly up front: src.Clone may report an index src.Indexes()
// already collected as no longer complete (the documented race window --
// see LeafSource's own comment), and a skipped index must not be counted.
// The count is therefore written as a zero placeholder and patched in
// place once the real count is known: buffered.Flush so f's Seek-observed
// position is accurate (nothing buffered-but-unflushed left to skew it),
// f.Seek back to the placeholder, rewrite it, f.Seek forward again to
// resume the buffered stream exactly where it left off. Safe because
// nothing else writes to f concurrently during one Save call
// (os.CreateTemp's own private, unshared temp file) -- see writeSnapshot's
// own doc comment for why the checksum is computed separately, after this
// patch, rather than incrementally alongside it.
func streamCompleteLeaves(sw *snapshotWriter, buffered *bufio.Writer, f *os.File, src LeafSource) error {
	if sw.err != nil {
		return nil // already broken; writeSnapshot's own bw.err check surfaces it
	}

	if err := buffered.Flush(); err != nil {
		return fmt.Errorf("flushing before leaf count placeholder: %w", err)
	}
	countOffset, err := f.Seek(0, io.SeekCurrent)
	if err != nil {
		return fmt.Errorf("seeking to leaf count placeholder: %w", err)
	}
	sw.writeUint32(0) // placeholder; patched below once every collected index has been attempted

	indexes := src.Indexes()
	written := uint32(0)
	for _, idx := range indexes {
		if sw.err != nil {
			break
		}
		leaf, ok := src.Clone(idx)
		if !ok || leaf == nil {
			// !ok: race window -- idx flipped away from LeafComplete
			// between collection and this Clone call. Safe to skip
			// outright -- see LeafSource's own doc comment.
			//
			// leaf == nil (with ok true): only reachable via a buggy
			// LeafSource implementation, or the theoretically-constructible
			// nil-complete Directory slot (Directory.Complete never
			// nil-checks its own reference before reporting complete).
			// Must degrade the same way as the race window -- leaf absent
			// from the snapshot, reconstructed on restore per the
			// completeness invariant (§ Leaf lifecycle) -- rather than a
			// nil-deref panic mid-save.
			continue
		}
		sw.writeUint32(idx)
		// fps/handles: same-package direct field access (leaf.go's own
		// type doc only restricts CONCURRENT access without the
		// directory's stripe lock; src.Clone already returned an
		// independent deep copy under that lock, so this read is safe
		// with no further synchronization).
		sw.writeUint32(uint32(len(leaf.fps)))
		for i := range leaf.fps {
			sw.writeUint16(leaf.fps[i])
			sw.writeUint32(uint32(leaf.handles[i]))
		}
		written++
	}
	if sw.err != nil {
		return nil // surfaced by writeSnapshot's own bw.err check, as above
	}

	if err := buffered.Flush(); err != nil {
		return fmt.Errorf("flushing leaf section: %w", err)
	}
	resumeOffset, err := f.Seek(0, io.SeekCurrent)
	if err != nil {
		return fmt.Errorf("seeking to leaf section end: %w", err)
	}

	if _, err := f.Seek(countOffset, io.SeekStart); err != nil {
		return fmt.Errorf("seeking back to fix up leaf count: %w", err)
	}
	fixup := &snapshotWriter{w: f}
	fixup.writeUint32(written)
	if fixup.err != nil {
		return fmt.Errorf("fixing up leaf count: %w", fixup.err)
	}

	if _, err := f.Seek(resumeOffset, io.SeekStart); err != nil {
		return fmt.Errorf("seeking back to resume after leaf count fixup: %w", err)
	}
	return nil
}

// decodeBody is encodeBody's exact inverse, reading incrementally from sr
// (an io.Reader-backed snapshotReader, 2026-07-16 restore-side streaming
// amendment -- see Load's own doc comment) rather than an in-memory byte
// slice. d/f/seedFingerprint come from the already-validated header
// (Load), not from the body itself -- there is no redundant copy of them
// in the body encoding.
//
// Every field EXCEPT the complete-leaf section is still decoded into an
// ordinary State field, exactly as before streaming: tokens, offsets,
// constructing ranges, blocks, and tenant sets are all small relative to
// leaf data (DESIGN.md § Sizing) and stay fully materialized. The
// complete-leaf section is the one exception -- streamCompleteLeavesIn
// (below) calls sink once per decoded leaf instead of building a map, and
// state.CompleteLeaves is left nil (see its own field comment on State).
func decodeBody(sr *snapshotReader, d, f uint8, seedFingerprint uint64, sink LeafSink) *State {
	state := &State{D: d, F: f, SeedFingerprint: seedFingerprint}

	numTokens := sr.readUint32()
	if !sr.checkCount(numTokens, 4, "tokens") { // 4 bytes/token: one uint32 each
		return state
	}
	state.Tokens = make([]uint32, numTokens)
	for i := range state.Tokens {
		state.Tokens[i] = sr.readUint32()
	}

	numOffsets := sr.readUint32()
	if !sr.checkCount(numOffsets, 12, "offsets") { // 12 bytes/entry: uint32 partition + int64 offset
		return state
	}
	state.Offsets = make(map[int32]int64, numOffsets)
	for i := uint32(0); i < numOffsets && sr.err == nil; i++ {
		p := int32(sr.readUint32())
		state.Offsets[p] = sr.readInt64()
	}

	numRanges := sr.readUint32()
	if !sr.checkCount(numRanges, 8, "constructing ranges") { // 8 bytes/range: two uint32s
		return state
	}
	state.ConstructingRanges = make([]LeafRange, numRanges)
	for i := range state.ConstructingRanges {
		state.ConstructingRanges[i] = LeafRange{Start: sr.readUint32(), End: sr.readUint32()}
	}

	numBlocks := sr.readUint32()
	// 28 bytes/block MINIMUM (not exact -- Block's own encoding is
	// variable-length, see checkCount's own doc comment): 16 (UUID) + 4
	// (TenantID's own blob length prefix, an empty string's smallest
	// legal encoding) + 1 (StartTime, smallest legal encoding is the
	// zero-time tag alone) + 1 (EndTime, same) + 1 (State) + 4 (Handle) +
	// 1 (DeletedAt, same as Start/EndTime) = 28; matches encodeBody's own
	// per-block field sequence below exactly.
	if !sr.checkCount(numBlocks, 28, "blocks") {
		return state
	}
	state.Blocks = make([]Block, numBlocks)
	for i := range state.Blocks {
		uuidBytes := sr.readBytes(16)
		var u backend.UUID
		if sr.err == nil {
			if err := u.Unmarshal(uuidBytes); err != nil {
				sr.err = fmt.Errorf("decoding block %d uuid: %w", i, err)
			}
		}
		state.Blocks[i] = Block{
			UUID:      u,
			TenantID:  sr.readString(),
			StartTime: sr.readTime(),
			EndTime:   sr.readTime(),
			State:     BlockState(sr.readUint8()),
			Handle:    Handle(sr.readUint32()),
			DeletedAt: sr.readTime(),
		}
	}

	streamCompleteLeavesIn(sr, sink)

	numTenants := sr.readUint32()
	// 8 bytes/tenant MINIMUM: tenantID's own blob length prefix (4, an
	// empty string's smallest legal encoding) + numBuckets's own 4-byte
	// field, present unconditionally regardless of its value -- the
	// buckets a tenant claims are bounded separately, immediately below.
	if !sr.checkCount(numTenants, 8, "tenants") {
		return state
	}
	state.Tenants = TenantSetSnapshot{Buckets: make(map[string]map[bucketKey][]byte, numTenants)}
	for i := uint32(0); i < numTenants && sr.err == nil; i++ {
		tenantID := sr.readString()
		numBuckets := sr.readUint32()
		// 12 bytes/bucket MINIMUM: bucketKey (8, int64) + the bucket
		// blob's own length prefix (4, an empty blob's smallest legal
		// encoding).
		if !sr.checkCount(numBuckets, 12, "tenant buckets") {
			return state
		}
		buckets := make(map[bucketKey][]byte, numBuckets)
		for j := uint32(0); j < numBuckets; j++ {
			k := bucketKey(sr.readInt64())
			// readBlob alone is enough here (no separate "copy" variant
			// needed, unlike before streaming): every read now allocates
			// independently -- see readBlob's own doc comment.
			buckets[k] = sr.readBlob()
		}
		state.Tenants.Buckets[tenantID] = buckets
	}

	return state
}

// streamCompleteLeavesIn decodes the complete-leaf section directly from
// sr, calling sink once per (idx, leaf) pair immediately after decoding it
// -- Load's streaming counterpart to Save's own streamCompleteLeaves.
// Never holds more than one leaf's worth of additional memory: fps/handles
// are allocated fresh per leaf (there is no shared backing buffer to slice
// into anymore, unlike the retired in-memory design), handed to sink, and
// then go out of scope from this function's own point of view -- what
// sink does with them afterward (reconcileStartup's own production sink
// imports owned ones straight into the Directory via Directory.Complete,
// pointer-shared, and silently discards the rest) is entirely its own
// business; see LeafSink's own doc comment.
func streamCompleteLeavesIn(sr *snapshotReader, sink LeafSink) {
	numLeaves := sr.readUint32()
	for i := uint32(0); i < numLeaves && sr.err == nil; i++ {
		idx := sr.readUint32()
		numEntries := sr.readUint32()
		// 6 bytes/entry: uint16 fingerprint + uint32 handle. 2026-07-16
		// review finding: this was the reviewer's own directly-reproduced
		// case -- a 54-byte crafted file claiming 50,000,000 entries drove
		// a large allocation below before ever failing; see checkCount's
		// own doc comment.
		if !sr.checkCount(numEntries, 6, "leaf entries") {
			return
		}
		// nil, not an allocated empty slice, for numEntries == 0: matches
		// NewLeaf()'s own zero-value convention (leaf.go) exactly, so an
		// empty leaf round-trips identically regardless of whether it was
		// ever populated -- a bare make([]T, 0) would be functionally
		// equivalent but observably different under reflect.DeepEqual.
		var fps []uint16
		var handles []Handle
		if numEntries > 0 {
			fps = make([]uint16, numEntries)
			handles = make([]Handle, numEntries)
		}
		for j := range fps {
			fps[j] = sr.readUint16()
			handles[j] = Handle(sr.readUint32())
		}
		if sr.err != nil {
			return
		}
		sink(idx, &Leaf{fps: fps, handles: handles})
	}
}

// snapshotWriter is a small sticky-error binary writer: once a write
// fails, every subsequent call becomes a no-op and returns the same
// error, so encodeBody's field-by-field calls never need an if err !=
// nil check after each individual one -- err is checked exactly once, by
// the caller, after encoding finishes.
type snapshotWriter struct {
	w   io.Writer
	err error
}

func (sw *snapshotWriter) writeBytes(b []byte) {
	if sw.err != nil {
		return
	}
	_, sw.err = sw.w.Write(b)
}

func (sw *snapshotWriter) writeUint8(v uint8) { sw.writeBytes([]byte{v}) }

func (sw *snapshotWriter) writeUint16(v uint16) {
	var b [2]byte
	binary.BigEndian.PutUint16(b[:], v)
	sw.writeBytes(b[:])
}

func (sw *snapshotWriter) writeUint32(v uint32) {
	var b [4]byte
	binary.BigEndian.PutUint32(b[:], v)
	sw.writeBytes(b[:])
}

func (sw *snapshotWriter) writeUint64(v uint64) {
	var b [8]byte
	binary.BigEndian.PutUint64(b[:], v)
	sw.writeBytes(b[:])
}

func (sw *snapshotWriter) writeInt64(v int64) { sw.writeUint64(uint64(v)) }

// writeBlob writes a uint32 length prefix followed by b's bytes.
func (sw *snapshotWriter) writeBlob(b []byte) {
	sw.writeUint32(uint32(len(b)))
	sw.writeBytes(b)
}

func (sw *snapshotWriter) writeString(s string) { sw.writeBlob([]byte(s)) }

// writeTime encodes t as a one-byte presence tag (0 = t.IsZero()) plus
// UnixNano when present. time.Time{}.UnixNano() is documented as
// undefined for dates outside roughly [1678, 2262] (Go's time package
// docs), and the zero Time (year 1) falls squarely outside that range --
// it cannot be safely round-tripped through UnixNano alone. This only
// ever matters for Block.StartTime/EndTime/DeletedAt, the only time.Time
// fields this file serializes (DeletedAt is legitimately zero for every
// non-BlockDeleted block, registry.go).
func (sw *snapshotWriter) writeTime(t time.Time) {
	if t.IsZero() {
		sw.writeUint8(0)
		return
	}
	sw.writeUint8(1)
	sw.writeInt64(t.UnixNano())
}

// snapshotReader is snapshotWriter's inverse: a sticky-error binary reader,
// streaming over an io.Reader (2026-07-16 restore-side amendment -- see
// decodeBody's own doc comment for why: Load must never materialize the
// whole, potentially many-GiB body in memory just to decode it, mirroring
// the save side's own streaming redesign). r is always an *io.LimitedReader
// bounding the total remaining body bytes (Load's own construction, for
// both the small fixed header and the body proper) -- every length-
// prefixed BYTE read (readBytes) checks the requested length against r.N
// BEFORE allocating, so a corrupted inner length (e.g. a bogus "4 billion
// trace IDs" claim on a single blob) still fails fast as ErrSnapshotCorrupt
// rather than attempting an allocation sized by untrusted input, exactly
// as the retired in-memory design's own bounds check did against a slice
// length instead. A COUNT read separately from any single blob -- sizing a
// make([]T, count) or make(map[K]V, count) of many small elements, rather
// than one length-prefixed byte read -- is a distinct hazard readBytes
// alone cannot cover (its check is against the ONE read it performs, not
// against a count some LATER, unrelated make() call will use); every such
// count is instead checked by checkCount below, immediately after being
// read and always before the make() it sizes.
type snapshotReader struct {
	r   *io.LimitedReader
	err error
}

// readBytes allocates and reads exactly n fresh bytes. Every field this
// file decodes now allocates independently -- there is no shared backing
// buffer to slice into anymore, unlike the retired in-memory design -- so
// a caller never needs to worry about a returned slice pinning anything
// beyond itself; see readBlob's own doc comment for what this replaces.
func (sr *snapshotReader) readBytes(n int) []byte {
	if sr.err != nil {
		return nil
	}
	if n < 0 || int64(n) > sr.r.N {
		sr.err = fmt.Errorf("%w: need %d bytes, only %d remain in the body", ErrSnapshotCorrupt, n, sr.r.N)
		return nil
	}
	buf := make([]byte, n)
	if _, err := io.ReadFull(sr.r, buf); err != nil {
		sr.err = fmt.Errorf("%w: %v", ErrSnapshotCorrupt, err)
		return nil
	}
	return buf
}

// checkCount reports whether count elements, each requiring at least
// minStride bytes on disk, could possibly be backed by the bytes sr has
// left -- and sets sr.err (returning false) if not. 2026-07-16 review
// finding: decodeBody/streamCompleteLeavesIn each read an untrusted count
// and then make() a slice or map sized DIRECTLY by it (tokens, offsets,
// constructing ranges, blocks, leaf entries, tenants, tenant buckets) --
// readBytes's own before-allocating check (its doc comment) only guards
// a single length-prefixed byte blob, not a count that sizes a LATER
// make() of many elements; without this check, a corrupt or crafted file
// can force an allocation proportional to any uint32 value regardless of
// the file's real size (reproduced directly in review: a 54-byte crafted
// file claiming 50,000,000 entries drove a large allocation before ever
// failing -- at uint32's actual max that scales to tens of GB in one
// make() call, the exact OOM class this whole streaming rewrite exists to
// eliminate). Every call site below invokes this immediately after
// reading its count and always before the corresponding make(), mirroring
// readBytes's own check-before-allocate discipline for the one shape it
// cannot itself cover.
//
// minStride is each element's SMALLEST possible on-disk size, not
// necessarily its exact size: callers whose element encoding is fixed-
// width (tokens, offsets, ranges, leaf entries) pass the exact per-element
// byte count, but callers whose encoding is variable-length (blocks embed
// a string and three optional timestamps; tenants and tenant buckets embed
// a string or blob) pass the minimum any single element could ever
// consume -- every optional/variable part at its smallest legal encoding.
// Either way the check stays sound: a legitimate file can never be
// rejected (every real element consumes at least minStride bytes), while
// a count too large for the remaining body to possibly back, even at the
// most generous minimum, is caught before the allocation. The comparison
// divides rather than multiplies specifically to avoid any risk of the
// bound check's own arithmetic overflowing for a large count.
func (sr *snapshotReader) checkCount(count uint32, minStride int64, what string) bool {
	if sr.err != nil {
		return false
	}
	if minStride <= 0 { // defensive: every real call site passes a positive literal
		return true
	}
	if int64(count) > sr.r.N/minStride {
		sr.err = fmt.Errorf("%w: %s count %d implausible: only %d bytes remain in the body", ErrSnapshotCorrupt, what, count, sr.r.N)
		return false
	}
	return true
}

func (sr *snapshotReader) readUint8() uint8 {
	b := sr.readBytes(1)
	if b == nil {
		return 0
	}
	return b[0]
}

func (sr *snapshotReader) readUint16() uint16 {
	b := sr.readBytes(2)
	if b == nil {
		return 0
	}
	return binary.BigEndian.Uint16(b)
}

func (sr *snapshotReader) readUint32() uint32 {
	b := sr.readBytes(4)
	if b == nil {
		return 0
	}
	return binary.BigEndian.Uint32(b)
}

func (sr *snapshotReader) readUint64() uint64 {
	b := sr.readBytes(8)
	if b == nil {
		return 0
	}
	return binary.BigEndian.Uint64(b)
}

func (sr *snapshotReader) readInt64() int64 { return int64(sr.readUint64()) }

// readBlob reads a uint32 length prefix and returns that many freshly
// allocated bytes -- safe to retain indefinitely, unlike before streaming.
// The retired in-memory design's readBlob returned a slice INTO the
// shared body buffer (no copy), which was fine for a value consumed
// immediately (e.g. UUID.Unmarshal, which copies internally) but NOT for
// one retained long-term (tenant bucket blobs, the one raw []byte this
// file keeps inside the returned *State) without a separate readBlobCopy
// that made an explicit copy. There is no shared buffer to alias anymore
// -- every read already allocates independently -- so that distinction,
// and the second method, are both gone: readBlob alone is enough now for
// every caller, retained or not.
func (sr *snapshotReader) readBlob() []byte {
	n := sr.readUint32()
	if sr.err != nil {
		return nil
	}
	return sr.readBytes(int(n))
}

func (sr *snapshotReader) readString() string {
	b := sr.readBlob()
	if b == nil {
		return ""
	}
	return string(b) // always copies (Go spec: []byte-to-string conversion copies)
}

// readTime is writeTime's exact inverse.
func (sr *snapshotReader) readTime() time.Time {
	tag := sr.readUint8()
	if sr.err != nil || tag == 0 {
		return time.Time{}
	}
	return time.Unix(0, sr.readInt64()).UTC()
}
