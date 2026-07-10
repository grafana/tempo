package bloomgateway

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
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
	D, F               uint8
	SeedFingerprint    uint64
	Tokens             []uint32
	Offsets            map[int32]int64
	CompleteLeaves     map[uint32]*Leaf
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
// caller's pause.
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
// length is known, and the body itself streams DIRECTLY to f (through an
// io.MultiWriter that also feeds a running CRC32) rather than being
// buffered whole in memory first. This is deliberate, not an
// over-optimization: State's structures (leaves, registry, tenant sets)
// are already the live, in-memory serving state at DESIGN.md's reference
// ~15-20 GiB/instance scale (§ Sizing) -- buffering a second full copy
// just to learn its length before writing the header would transiently
// double this instance's memory footprint on every snapshot.
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
	// writes. The checksum still sees EXACTLY the same bytes f does,
	// since bufio's underlying writer here is the MultiWriter, not f
	// directly -- Flush below is what guarantees f's own Seek-observed
	// position matches what's actually landed on disk, before this
	// function relies on that position for the length fixup.
	checksum := crc32.NewIEEE()
	buffered := bufio.NewWriter(io.MultiWriter(f, checksum))
	bw := &snapshotWriter{w: buffered}
	encodeBody(bw, state)
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

// Load decodes path. Returns ErrSnapshotMismatch (format version, D, F, or
// seed fingerprint) after reading ONLY the small fixed header -- WITHOUT
// reading the (potentially many-GiB) body at all, so a mismatch is always
// cheap to detect and never risks a partial/corrupt read of state that's
// about to be discarded anyway (mismatch table tested first in
// snapshot_test.go, before the happy-path round-trip). Any other failure
// (missing file, bad magic, truncated body, checksum mismatch, malformed
// field) wraps ErrSnapshotCorrupt instead, distinguishable via errors.Is.
func (sn *Snapshotter) Load(path string, wantD, wantF uint8, wantSeedFingerprint uint64) (*State, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("%w: opening %s: %v", ErrSnapshotCorrupt, path, err)
	}
	defer f.Close()

	headerBuf := make([]byte, snapshotHeaderSize)
	if _, err := io.ReadFull(f, headerBuf); err != nil {
		return nil, fmt.Errorf("%w: reading header: %v", ErrSnapshotCorrupt, err)
	}
	hr := &snapshotReader{b: headerBuf}

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

	// Only now -- format/D/F/seed confirmed compatible -- read the rest of
	// the file, bounded by bodyLength+4 (our OWN trusted header field, not
	// any nested/corrupted value): io.LimitReader guarantees this read
	// never pulls in more bytes than the header itself already claims,
	// regardless of what garbage might follow on disk.
	rest, err := io.ReadAll(io.LimitReader(f, int64(bodyLength)+4))
	if err != nil {
		return nil, fmt.Errorf("%w: reading body: %v", ErrSnapshotCorrupt, err)
	}
	if uint64(len(rest)) < bodyLength+4 {
		return nil, fmt.Errorf("%w: truncated: want %d body+checksum bytes, got %d", ErrSnapshotCorrupt, bodyLength+4, len(rest))
	}

	body := rest[:bodyLength]
	wantChecksum := binary.BigEndian.Uint32(rest[bodyLength : bodyLength+4])
	if gotChecksum := crc32.ChecksumIEEE(body); gotChecksum != wantChecksum {
		return nil, fmt.Errorf("%w: checksum mismatch: want %#x, got %#x", ErrSnapshotCorrupt, wantChecksum, gotChecksum)
	}

	// Every subsequent length-prefixed read below is bounds-checked against
	// len(body) BEFORE ever slicing (snapshotReader.readBytes) -- a
	// corrupted inner length (e.g. a bogus "4 billion trace IDs" claim)
	// therefore fails fast as ErrSnapshotCorrupt rather than attempting an
	// allocation sized by untrusted input.
	sr := &snapshotReader{b: body}
	state := decodeBody(sr, gotD, gotF, gotSeedFingerprint)
	if sr.err != nil {
		return nil, fmt.Errorf("%w: decoding body: %v", ErrSnapshotCorrupt, sr.err)
	}
	return state, nil
}

// encodeBody writes every State field below the header, in a fixed order
// decodeBody must mirror exactly. Map-keyed fields (Offsets,
// CompleteLeaves, Tenants.Buckets) are written in a stable sorted order
// purely so two Save calls over equal input produce byte-identical
// output; decodeBody does not depend on that order in any way.
func encodeBody(sw *snapshotWriter, state *State) {
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

	leafIndexes := make([]uint32, 0, len(state.CompleteLeaves))
	for idx := range state.CompleteLeaves {
		leafIndexes = append(leafIndexes, idx)
	}
	sort.Slice(leafIndexes, func(i, j int) bool { return leafIndexes[i] < leafIndexes[j] })
	sw.writeUint32(uint32(len(leafIndexes)))
	for _, idx := range leafIndexes {
		leaf := state.CompleteLeaves[idx]
		sw.writeUint32(idx)
		// fps/handles: same-package direct field access (leaf.go's own
		// type doc only restricts CONCURRENT access without the
		// directory's stripe lock; Save's own documented precondition --
		// the caller has already paused all mutation -- makes this a
		// safe, single-threaded, already-quiesced read, exactly like
		// Directory.Leaf's "safe for a caller that already knows no
		// concurrent mutation is possible" case).
		sw.writeUint32(uint32(len(leaf.fps)))
		for i := range leaf.fps {
			sw.writeUint16(leaf.fps[i])
			sw.writeUint32(uint32(leaf.handles[i]))
		}
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
}

// decodeBody is encodeBody's exact inverse. d/f/seedFingerprint come from
// the already-validated header (Load), not from the body itself -- there
// is no redundant copy of them in the body encoding.
func decodeBody(sr *snapshotReader, d, f uint8, seedFingerprint uint64) *State {
	state := &State{D: d, F: f, SeedFingerprint: seedFingerprint}

	numTokens := sr.readUint32()
	state.Tokens = make([]uint32, numTokens)
	for i := range state.Tokens {
		state.Tokens[i] = sr.readUint32()
	}

	numOffsets := sr.readUint32()
	state.Offsets = make(map[int32]int64, numOffsets)
	for i := uint32(0); i < numOffsets && sr.err == nil; i++ {
		p := int32(sr.readUint32())
		state.Offsets[p] = sr.readInt64()
	}

	numRanges := sr.readUint32()
	state.ConstructingRanges = make([]LeafRange, numRanges)
	for i := range state.ConstructingRanges {
		state.ConstructingRanges[i] = LeafRange{Start: sr.readUint32(), End: sr.readUint32()}
	}

	numBlocks := sr.readUint32()
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

	numLeaves := sr.readUint32()
	state.CompleteLeaves = make(map[uint32]*Leaf, numLeaves)
	for i := uint32(0); i < numLeaves && sr.err == nil; i++ {
		idx := sr.readUint32()
		numEntries := sr.readUint32()
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
		state.CompleteLeaves[idx] = &Leaf{fps: fps, handles: handles}
	}

	numTenants := sr.readUint32()
	state.Tenants = TenantSetSnapshot{Buckets: make(map[string]map[bucketKey][]byte, numTenants)}
	for i := uint32(0); i < numTenants && sr.err == nil; i++ {
		tenantID := sr.readString()
		numBuckets := sr.readUint32()
		buckets := make(map[bucketKey][]byte, numBuckets)
		for j := uint32(0); j < numBuckets; j++ {
			k := bucketKey(sr.readInt64())
			// readBlobCopy (not readBlob) is load-bearing: readBlob would
			// return a slice INTO the shared body buffer, which would
			// then pin that entire (potentially many-GiB) buffer in
			// memory for as long as this one small bucket blob stays
			// reachable inside the returned State -- an explicit copy is
			// what lets the raw body buffer actually be garbage
			// collected once decode finishes.
			buckets[k] = sr.readBlobCopy()
		}
		state.Tenants.Buckets[tenantID] = buckets
	}

	return state
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

// snapshotReader is snapshotWriter's inverse: a sticky-error binary reader
// over an in-memory body. Every read is bounds-checked against len(b)
// BEFORE any slice operation (readBytes), so a corrupted length-prefixed
// field fails fast as an error rather than ever attempting to slice past
// the buffer.
type snapshotReader struct {
	b   []byte
	pos int
	err error
}

// readBytes returns a slice INTO sr.b (no copy) -- callers that retain the
// result beyond the current decode step (i.e. anywhere in the returned
// *State) must copy it explicitly; see readBlobCopy.
func (sr *snapshotReader) readBytes(n int) []byte {
	if sr.err != nil {
		return nil
	}
	if n < 0 || sr.pos+n > len(sr.b) {
		sr.err = fmt.Errorf("%w: need %d bytes at offset %d, only %d remain", ErrSnapshotCorrupt, n, sr.pos, len(sr.b)-sr.pos)
		return nil
	}
	out := sr.b[sr.pos : sr.pos+n]
	sr.pos += n
	return out
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

// readBlob reads a uint32 length prefix and returns that many bytes AS A
// SLICE INTO sr.b (no copy) -- fine for a value consumed immediately and
// not retained (e.g. UUID.Unmarshal, which copies internally; string(b),
// which always copies per the language spec), but NOT fine for a raw
// []byte retained long-term; see readBlobCopy for that case.
func (sr *snapshotReader) readBlob() []byte {
	n := sr.readUint32()
	if sr.err != nil {
		return nil
	}
	return sr.readBytes(int(n))
}

// readBlobCopy is readBlob plus an explicit copy, for the one field this
// file retains as a raw []byte inside the returned *State (tenant bucket
// blobs) -- see its call site's comment in decodeBody for why the copy is
// load-bearing, not defensive-programming boilerplate.
func (sr *snapshotReader) readBlobCopy() []byte {
	b := sr.readBlob()
	if b == nil {
		return nil
	}
	out := make([]byte, len(b))
	copy(out, b)
	return out
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
