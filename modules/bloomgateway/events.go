package bloomgateway

import (
	"fmt"
	"sync"
	"time"

	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/tempodb/backend"
)

// supportedEventVersion is the only BloomGatewayEvent envelope version this
// build understands. DecodeEvent rejects any other value (including the
// proto3 default 0, i.e. an envelope whose producer never set the field)
// BEFORE dispatching on Type — an unrecognized version is dropped and
// counted, never partially applied (DESIGN.md § Write path event types:
// "versioned envelope"; plan §2's explicit version discipline).
const supportedEventVersion uint32 = 1

// Applier decodes a BloomGatewayEvent and applies it idempotently against the
// directory, registry, and tenant sets — the safety-invariant core of the
// write path (DESIGN.md § Event processing). The same Applier is used for
// live Kafka events (WP14) and for synthetic reconstruction/reconciliation
// Adds (WP18/WP19), so live and rebuilt state converge on exactly one apply
// path, as the design requires.
//
// Every operation here is idempotent: at-least-once redelivery, out-of-order
// chunks, replay, and concurrent workers all converge to the same state
// (invariant #5).
type Applier struct {
	dir      *Directory
	reg      *Registry
	tenants  *TenantSet
	hashSeed uint64 // already domain-separated via HashSeed (hash.go); NOT the raw seed
	d, f     uint8
	metrics  *metrics

	// chunkMu guards chunkProgress only. It is deliberately NOT held across
	// the completion side effects (AddBlock/CommitLive), so a single global
	// mutex over all in-flight blocks' chunk bookkeeping never serializes the
	// registry/tenant work of unrelated blocks completing in parallel during
	// replay.
	chunkMu       sync.Mutex
	chunkProgress map[chunkKey]*chunkProgress

	// applyDeleteRaceHook, if non-nil, is invoked by ApplyDelete right after
	// MarkDeleted succeeds and before RemoveBlock. It exists solely so
	// events_test.go can deterministically drive a concurrent completing
	// chunk into that exact gap: the gap is a couple of instructions wide in
	// real code, so real goroutine scheduling alone does not reliably land
	// there. Matches modules/frontend/queue's testHookBeforeWaiting
	// convention; always nil in production.
	applyDeleteRaceHook func()
}

// chunkKey identifies one chunking of one block. Keyed by (uuid, chunkCount),
// not uuid alone (AMENDMENT A2): a synthetic single-chunk repair/reconstruction
// Add (chunkCount=1) must be able to complete a block immediately and
// idempotently even if a partial grouping from the live stream (a different
// chunkCount) is already in flight for the same UUID. The two groupings are
// independent; whichever fills first commits the block, and the other is
// discarded on completion.
type chunkKey struct {
	uuid  backend.UUID
	count uint32
}

// chunkProgress tracks which chunk indices of one chunkKey have been applied.
// remaining is the count of not-yet-seen indices; completion fires when it
// reaches zero. Tracking seen-ness per index (rather than a bare counter)
// makes redelivery of the same chunk_index idempotent — re-seeing an already
// seen index does not decrement remaining.
type chunkProgress struct {
	seen      []bool
	remaining int
}

// NewApplier builds an Applier. hashSeed must be the already-derived
// HashSeed(rawSeed) value (hash.go / AMENDMENT A4), never the raw configured
// secret — Address is called with this value verbatim.
func NewApplier(dir *Directory, reg *Registry, tenants *TenantSet, hashSeed uint64, d, f uint8, m *metrics) *Applier {
	return &Applier{
		dir:           dir,
		reg:           reg,
		tenants:       tenants,
		hashSeed:      hashSeed,
		d:             d,
		f:             f,
		metrics:       m,
		chunkProgress: make(map[chunkKey]*chunkProgress),
	}
}

// DecodeEvent unmarshals a raw Kafka record value into a BloomGatewayEvent and
// validates the envelope version. A non-nil error means "drop and count" to
// the caller (WP14's worker) — the event must never be partially applied.
// Version is checked before Type is ever examined, per the plan's version
// discipline.
func (a *Applier) DecodeEvent(raw []byte) (*tempopb.BloomGatewayEvent, error) {
	event := &tempopb.BloomGatewayEvent{}
	if err := event.Unmarshal(raw); err != nil {
		return nil, fmt.Errorf("bloomgateway: decode event: %w", err)
	}
	if event.Version != supportedEventVersion {
		return nil, fmt.Errorf("bloomgateway: unsupported event version %d (want %d)", event.Version, supportedEventVersion)
	}
	return event, nil
}

// Apply dispatches a decoded event to ApplyAddChunk or ApplyDelete. A nil
// sub-message for the declared type, or an unspecified/unknown type, is an
// error (drop and count) — never a silent no-op that would hide a producer
// bug.
func (a *Applier) Apply(event *tempopb.BloomGatewayEvent) error {
	switch event.Type {
	case tempopb.BloomGatewayEventType_BLOOM_GATEWAY_EVENT_TYPE_ADD_CHUNK:
		if event.AddChunk == nil {
			return fmt.Errorf("bloomgateway: ADD_CHUNK event with nil add_chunk payload")
		}
		return a.ApplyAddChunk(event.AddChunk)
	case tempopb.BloomGatewayEventType_BLOOM_GATEWAY_EVENT_TYPE_DELETE:
		if event.Delete == nil {
			return fmt.Errorf("bloomgateway: DELETE event with nil delete payload")
		}
		return a.ApplyDelete(event.Delete)
	default:
		return fmt.Errorf("bloomgateway: unspecified or unknown event type %v", event.Type)
	}
}

// ApplyAddChunk applies one AddChunk. Every step is idempotent (invariant
// #5). The high-level shape (DESIGN.md § Event processing):
//
//  1. Validate the whole chunk up front — parse block_id, and require every
//     trace ID to be 1..16 bytes. A single malformed trace ID drops the
//     ENTIRE chunk (none of its IDs applied, the chunk NOT marked seen), so a
//     producer bug can never half-apply a chunk (invariant: all-or-nothing).
//  2. GetOrCreate the block (any chunk may be the first — arrival order is
//     not guaranteed). If it is already BlockDeleted, return immediately
//     without touching leaves or chunk bookkeeping: tombstone terminality
//     (invariant #4), which is what makes late/replayed chunks harmless.
//  3. Insert (fp, handle) for every trace ID, IGNORING each InsertLive's
//     applied result — a block whose IDs all hash to unowned (nil) leaves
//     still commits (invariant #3, unconditional commit).
//  4. Record the chunk index; iff this call fills its (uuid, count) grouping,
//     commit the block exactly once (§ commitBlock).
func (a *Applier) ApplyAddChunk(chunk *tempopb.BloomGatewayAddChunk) (err error) {
	start := time.Now()
	defer func() {
		a.metrics.addApplyDurationSeconds.Observe(time.Since(start).Seconds())
		if err != nil {
			a.metrics.addsTotal.WithLabelValues("dropped").Inc()
		} else {
			a.metrics.addsTotal.WithLabelValues("applied").Inc()
		}
	}()

	uuid, perr := backend.ParseUUID(chunk.BlockId)
	if perr != nil {
		return fmt.Errorf("bloomgateway: add chunk: parse block_id: %w", perr)
	}
	// Validate the whole chunk before mutating anything (all-or-nothing).
	// PadTraceIDTo16Bytes never errors (it truncates >16B, left-pads <16B),
	// so a length check here is the only place a malformed trace ID — an
	// empty one (would pad to the all-zero ID) or an over-long one (would be
	// silently truncated, mis-hashing vs. what the producer sent) — can be
	// caught and turned into a whole-chunk drop.
	if verr := validateTraceIDs(chunk.TraceIds); verr != nil {
		return verr
	}

	blockStart := time.Unix(0, chunk.StartTimeUnixNano)
	blockEnd := time.Unix(0, chunk.EndTimeUnixNano)
	block, _ := a.reg.GetOrCreate(uuid, chunk.TenantId, blockStart, blockEnd)

	// Tombstone fast-path: a Deleted block is terminal, so skip leaf inserts
	// and bookkeeping entirely. A stale "not deleted" read here is safe — the
	// authoritative terminality check is inside commitBlock's CommitLive,
	// which no-ops under the registry lock if the block became Deleted after
	// this point (so we can never resurrect a swept block into A_T).
	//
	// Also drop any chunkProgress grouping already recorded for uuid: this
	// fast-path returns before recordChunk ever runs, so it is the only
	// place that can reclaim a grouping left behind by a chunk that landed
	// before the block was Deleted mid-chunking — otherwise it would leak
	// for the life of the process (this bookkeeping is deliberately not
	// swept or persisted, AMENDMENT A2).
	if state, _ := a.reg.State(uuid); state == BlockDeleted {
		a.chunkMu.Lock()
		a.dropProgressForUUIDLocked(uuid)
		a.chunkMu.Unlock()
		return nil
	}

	for _, id := range chunk.TraceIds {
		leafIdx, fp := Address(id, a.hashSeed, a.d, a.f)
		// fp < 2^f and Config.Validate enforces f <= 16, so the cast is
		// lossless; leaf storage is fp16 (leaf.go).
		a.dir.InsertLive(leafIdx, uint16(fp), block.Handle)
	}
	a.metrics.addChunksTotal.Inc()

	a.recordChunk(uuid, chunk.TenantId, chunk.ChunkIndex, chunk.ChunkCount, block)
	return nil
}

// recordChunk marks (chunkIndex) seen within the (uuid, chunkCount) grouping
// and, iff that call is the one that fills the grouping, commits the block
// exactly once. The fill detection and grouping teardown happen under
// chunkMu, so among concurrent workers completing the same grouping, exactly
// one observes remaining hit zero and proceeds to commit (invariant #5's
// concurrent-worker case; the trickiest concurrency in the module).
func (a *Applier) recordChunk(uuid backend.UUID, tenantID string, chunkIndex, chunkCount uint32, block *Block) {
	// Only Pending blocks need chunk accounting. Once a block is resolved
	// (Live / LiveUnsupportedEncoding / Deleted) its completion has already
	// happened (or is moot), so a late or duplicate chunk must NOT create a
	// fresh grouping that could linger forever — AMENDMENT A2's "late live
	// chunks after commit ... harmlessly no-op". The leaf inserts above
	// already ran (idempotent); there is simply nothing left to complete.
	//
	// Drop any lingering grouping(s) for uuid here too: the block can have
	// been resolved by a different chunkCount grouping (AMENDMENT A2's
	// synthetic single-chunk repair) or by a Delete that raced ahead of this
	// grouping's own completion, either of which can leave this uuid's entry
	// orphaned with no other call site left to reclaim it.
	if state, _ := a.reg.State(uuid); state != BlockPending {
		a.chunkMu.Lock()
		a.dropProgressForUUIDLocked(uuid)
		a.chunkMu.Unlock()
		return
	}

	a.chunkMu.Lock()
	completed := false
	if chunkCount > 0 && chunkIndex < chunkCount {
		key := chunkKey{uuid: uuid, count: chunkCount}
		p := a.chunkProgress[key]
		if p == nil {
			p = &chunkProgress{seen: make([]bool, chunkCount), remaining: int(chunkCount)}
			a.chunkProgress[key] = p
		}
		if !p.seen[chunkIndex] {
			p.seen[chunkIndex] = true
			p.remaining--
		}
		if p.remaining == 0 {
			completed = true
			a.dropProgressForUUIDLocked(uuid)
		}
	}
	a.chunkMu.Unlock()

	if completed {
		a.commitBlock(uuid, tenantID, block, false)
	}
}

// dropProgressForUUIDLocked removes every grouping for uuid. Called on
// completion so that a partial grouping under a different chunkCount (a
// pathological producer sending mixed counts, or a live grouping that a
// synthetic count=1 repair completed ahead of) cannot linger. Caller holds
// chunkMu.
func (a *Applier) dropProgressForUUIDLocked(uuid backend.UUID) {
	for k := range a.chunkProgress {
		if k.uuid == uuid {
			delete(a.chunkProgress, k)
		}
	}
}

// commitBlock performs a block's one Pending -> Live transition and its A_T
// insertion, resolving any race with a concurrent Delete or
// CommitUnsupportedEncoding demotion so that the block ends up in A_T if and
// only if it is confirmed Live (never Deleted, never LiveUnsupportedEncoding).
//
// The ordering — AddBlock, then CommitLive, then undo-unless-confirmed-Live —
// is chosen so that every interleaving converges to the terminal correct
// state:
//   - vs. ApplyDelete (MarkDeleted, then RemoveBlock, in that order — see
//     ApplyDelete's doc for why that order matters): if MarkDeleted already
//     ran, our CommitLive no-ops (BlockDeleted is terminal) and our
//     post-check sees a non-Live state and undoes the add. If MarkDeleted
//     has not run yet, our CommitLive may win the race and set Live;
//     ApplyDelete's own RemoveBlock — issued only after its MarkDeleted, so
//     always the last word of that Delete — then removes our handle
//     regardless of what our post-check saw.
//   - vs. CommitUnsupportedEncoding (CommitLive(uuid, true), then
//     RemoveBlock): a still-Pending block can be demoted straight to
//     BlockLiveUnsupportedEncoding without ever passing through BlockLive, so
//     our own CommitLive(uuid, false) can land second and hit the registry's
//     illegal LiveUnsupportedEncoding->Live transition (rejected, state left
//     unchanged). The post-check must therefore undo on ANY state other than
//     confirmed Live — checking only for BlockDeleted misses exactly this
//     case, leaving the handle stuck in A_T forever with nothing else ever
//     calling RemoveBlock for it again.
//
// unsupportedEncoding routes through CommitLive(uuid, true) for the
// LiveUnsupportedEncoding path (AMENDMENT A1); such a block is never added to
// A_T (and is removed from it, if a prior live Add had added it).
func (a *Applier) commitBlock(uuid backend.UUID, tenantID string, block *Block, unsupportedEncoding bool) {
	if unsupportedEncoding {
		// LiveUnsupportedEncoding blocks are never rejectable: ensure the
		// handle is absent from A_T (a no-op unless a prior live Add had made
		// this block Live and added it — the demotion case).
		_ = a.reg.CommitLive(uuid, true)
		a.tenants.RemoveBlock(tenantID, block.Handle)
		return
	}

	a.tenants.AddBlock(tenantID, block.Handle, block.StartTime, block.EndTime)
	_ = a.reg.CommitLive(uuid, false) // no-op/error if a Delete or demotion already won the race
	if state, _ := a.reg.State(uuid); state != BlockLive {
		// Anything other than a confirmed Live means some other transition
		// won the race — BlockDeleted (a Delete; its RemoveBlock may have run
		// before our AddBlock) or BlockLiveUnsupportedEncoding (a concurrent
		// CommitUnsupportedEncoding demotion) — so undo the speculative add.
		// Idempotent if the other side already removed it.
		a.tenants.RemoveBlock(tenantID, block.Handle)
	}
}

// ApplyDelete applies a Delete. A no-op for an unknown block (§ Event
// processing: "Delete for an unknown block is a no-op"). Otherwise it marks
// the registry entry Deleted (terminal), removes the block's handle from
// A_T, and drops any leftover chunkProgress grouping for it.
//
// MarkDeleted runs BEFORE RemoveBlock (not after): this makes RemoveBlock
// this Delete's last, unconditional, idempotent action, issued only once the
// registry has durably recorded the terminal state. That ordering is what
// closes the race against commitBlock's own
// AddBlock/CommitLive/undo-unless-Live sequence (see commitBlock's doc) — a
// completing chunk's commitBlock can land at any point relative to this
// Delete's two effects, but because RemoveBlock here always comes last, it
// can never have already been "used up" as a no-op (H not yet added) before
// a concurrent AddBlock lands with nothing left to clean it up afterward. A
// RemoveBlock-then-MarkDeleted ordering leaves exactly that gap open.
//
// A Delete that races ahead of a still-Pending block's remaining chunks
// correctly prevents that block from ever becoming rejectable: MarkDeleted is
// terminal, so the later completing chunk's CommitLive no-ops (invariant #4).
func (a *Applier) ApplyDelete(del *tempopb.BloomGatewayDelete) error {
	uuid, err := backend.ParseUUID(del.BlockId)
	if err != nil {
		return fmt.Errorf("bloomgateway: delete: parse block_id: %w", err)
	}

	block, ok := a.reg.LookupUUID(uuid)
	if !ok {
		return nil // unknown block: no-op
	}

	if err := a.reg.MarkDeleted(uuid); err != nil {
		return err
	}
	if a.applyDeleteRaceHook != nil {
		a.applyDeleteRaceHook()
	}
	a.tenants.RemoveBlock(block.TenantID, block.Handle)

	// Reclaim any chunkProgress grouping orphaned by this block being
	// Deleted before its chunking completed — otherwise, if no further
	// chunk for it is ever redelivered, nothing else would ever reclaim it.
	a.chunkMu.Lock()
	a.dropProgressForUUIDLocked(uuid)
	a.chunkMu.Unlock()

	a.metrics.deletesTotal.Inc()
	return nil
}

// CommitUnsupportedEncoding registers (or demotes) a block whose parquet
// encoding this reader cannot column-project (§0 D7 / AMENDMENT A1). It is
// only ever called by the reconstruction/reconciliation column-fetch helper
// (WP18/WP19), never from the Kafka path: live Adds carry pre-enumerated
// trace IDs and are encoding-agnostic, so an unsupported encoding is strictly
// a backfill-time concern.
//
// The block enters (or is demoted to) BlockLiveUnsupportedEncoding and is
// guaranteed absent from A_T, so it can never be rejected and instead rides
// the existing "unknown to gateway, must be searched" wire semantics for free
// (invariant #10). Idempotent; a no-op if the block is already Deleted.
func (a *Applier) CommitUnsupportedEncoding(uuid backend.UUID, tenantID string, blockStart, blockEnd time.Time) error {
	block, _ := a.reg.GetOrCreate(uuid, tenantID, blockStart, blockEnd)

	before, _ := a.reg.State(uuid)
	if err := a.reg.CommitLive(uuid, true); err != nil {
		return err
	}
	// Whether newly registered or demoted from Live, ensure the handle is not
	// rejectable.
	a.tenants.RemoveBlock(tenantID, block.Handle)

	// The block is now resolved (LiveUnsupportedEncoding, or already Deleted
	// and left so by CommitLive) — no chunk completion can ever be pending
	// for it again. Drop any partial (uuid, count) grouping a prior
	// stuck-Pending live stream left behind, exactly as recordChunk and
	// ApplyDelete do at their own resolution points; otherwise a block
	// demoted mid-chunking by reconciliation would leak its grouping for the
	// life of the process.
	a.chunkMu.Lock()
	a.dropProgressForUUIDLocked(uuid)
	a.chunkMu.Unlock()

	// Count each block once, as it enters the unsupported state (not on
	// idempotent repeat calls). Decrementing on Delete/reclaim is the sweep's
	// / delete path's responsibility (WP16/WP19) — noted as a known gap here
	// so the gauge's downward maintenance lands with the code that owns those
	// transitions, rather than being silently omitted.
	if after, _ := a.reg.State(uuid); before != BlockLiveUnsupportedEncoding && after == BlockLiveUnsupportedEncoding {
		a.metrics.unsupportedEncodingBlocks.WithLabelValues(tenantID).Inc()
	}
	return nil
}

// validateTraceIDs returns an error if any trace ID is empty or longer than
// 16 bytes — the all-or-nothing chunk-validation predicate (see
// ApplyAddChunk). A valid trace ID is 1..16 bytes; canonicalization to the
// 16-byte form happens inside Address (via util.PadTraceIDTo16Bytes), exactly
// once per ID, and must not be duplicated here.
func validateTraceIDs(traceIDs [][]byte) error {
	for i, id := range traceIDs {
		if len(id) == 0 || len(id) > 16 {
			return fmt.Errorf("bloomgateway: add chunk: trace id at index %d has invalid length %d (want 1..16)", i, len(id))
		}
	}
	return nil
}
