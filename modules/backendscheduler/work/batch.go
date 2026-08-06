package work

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/grafana/tempo/pkg/tempopb"
)

const batchesFileName = "batches.pb"

// batchStore holds in-flight redaction batches, one per tenant at a time.
// Trace IDs are stored here rather than in each pending job so that a submission
// of N block-level jobs does not duplicate the trace ID list N times in memory
// or on disk.
type batchStore struct {
	// writeMu orders manifest writers with respect to each other. mu alone is not enough: it is
	// released before the file write, so two writers can marshal in one order and rename in the other.
	// That was harmless while every writer marshalled the live store — they all wrote the same truth —
	// but persistCancelled deliberately marshals a state the store does not yet hold, so an older
	// snapshot landing last can revert a cancel the handler has already published and reported.
	// Held across marshal-and-write; always acquired before mu, never while holding it.
	writeMu  sync.Mutex
	mu       sync.RWMutex
	byTenant map[string]*tempopb.RedactionBatch
}

func newBatchStore() *batchStore {
	return &batchStore{
		byTenant: make(map[string]*tempopb.RedactionBatch),
	}
}

func (b *batchStore) add(batch *tempopb.RedactionBatch) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if _, exists := b.byTenant[batch.TenantId]; exists {
		return ErrBatchAlreadyExists
	}
	b.byTenant[batch.TenantId] = batch
	return nil
}

// get returns a shallow copy of the tenant's batch taken under the lock, or nil if there is none.
// Copying rather than handing back the live pointer is what lets callers read mutable fields
// (Cancelled, RescanAfterUnixNano, QuiesceUntilUnixNano) off-lock: the locked setters mutate the
// stored batch, so a live pointer would race them. Same contract as list() — see its comment for why
// a shallow copy is a consistent view.
func (b *batchStore) get(tenantID string) *tempopb.RedactionBatch {
	b.mu.RLock()
	defer b.mu.RUnlock()
	batch, ok := b.byTenant[tenantID]
	if !ok {
		return nil
	}
	cp := *batch
	return &cp
}

func (b *batchStore) remove(tenantID string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.byTenant, tenantID)
}

// hasBlockingBatch reports whether the tenant has a batch that must block compaction/retention,
// i.e. an apply-mode redaction. A dry-run writes nothing and is not an exclusive operation, so it
// does not block. Fails safe: an unset/unrecognized mode (zero value APPLY) blocks.
func (b *batchStore) hasBlockingBatch(tenantID string) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	batch, ok := b.byTenant[tenantID]
	return ok && !batch.Mode.IsDryRun()
}

// flush writes all active batches to batches.pb in localPath using proto encoding.
func (b *batchStore) flush(localPath string) error {
	b.writeMu.Lock()
	defer b.writeMu.Unlock()

	// Hold the read lock through Marshal so the locked setters (setRescan, setRescanIfCurrent,
	// setQuiesceUntil, setCancelled) cannot mutate fields that proto.Marshal is reading.
	b.mu.RLock()
	batches := make([]*tempopb.RedactionBatch, 0, len(b.byTenant))
	for _, batch := range b.byTenant {
		batches = append(batches, batch)
	}
	msg := &tempopb.RedactionBatches{Batches: batches}
	data, err := msg.Marshal()
	b.mu.RUnlock()

	if err != nil {
		return fmt.Errorf("marshal batches: %w", err)
	}

	path := filepath.Join(localPath, batchesFileName)
	if err := os.MkdirAll(localPath, 0o700); err != nil {
		return fmt.Errorf("mkdir %s: %w", localPath, err)
	}
	return atomicWriteFile(data, path, batchesFileName)
}

// list returns all active batches.
func (b *batchStore) list() []*tempopb.RedactionBatch {
	b.mu.RLock()
	defer b.mu.RUnlock()
	out := make([]*tempopb.RedactionBatch, 0, len(b.byTenant))
	for _, batch := range b.byTenant {
		// Return a shallow copy taken under the lock, not the live pointer: callers read mutable
		// fields (RescanAfterUnixNano, Cancelled, ...) off tick, which would otherwise race the
		// locked setters (SetBatchRescan/SetBatchCancelled). Scalars are snapshotted by value;
		// slice fields (e.g. SkippedCompactionJobIds) are reassigned rather than mutated in place,
		// so the copied header stays a consistent view. Cheap: no deep copy of trace IDs.
		cp := *batch
		out = append(out, &cp)
	}
	return out
}

// setRescan updates the rescan fields on the batch for tenantID under the write lock.
// Pass nil ids and 0 afterNano to clear the rescan state.
func (b *batchStore) setRescan(tenantID string, ids []string, afterNano int64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if batch, ok := b.byTenant[tenantID]; ok {
		batch.SkippedCompactionJobIds = ids
		batch.RescanAfterUnixNano = afterNano
	}
}

// setRescanIfCurrent arms the rescan only if the tenant's batch is still the one identified by
// batchID and has not been cancelled, reporting whether it applied. The check and the write share one
// lock acquisition, so a cancel cannot land between them.
//
// The rescan sweep computes its result from a snapshot and takes a long time doing it, so the batch can
// change underneath it: a cancel clears the rescan (and must not see it re-armed), and once a cancelled
// batch is removed the operator may resubmit, giving a new batch ID that must not inherit a previous
// batch's scan result.
func (b *batchStore) setRescanIfCurrent(tenantID, batchID string, ids []string, afterNano int64) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	batch, ok := b.byTenant[tenantID]
	if !ok || batch.BatchId != batchID || batch.Cancelled {
		return false
	}
	batch.SkippedCompactionJobIds = ids
	batch.RescanAfterUnixNano = afterNano
	return true
}

func (b *batchStore) setQuiesceUntil(tenantID string, untilUnixNano int64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if batch, ok := b.byTenant[tenantID]; ok {
		batch.QuiesceUntilUnixNano = untilUnixNano
	}
}

// quiescenceState reads a tenant's quiescence-relevant fields under the lock, returning a
// snapshot so callers never touch the live batch pointer's mutable fields unsynchronized.
func (b *batchStore) quiescenceState(tenantID string) (quiesceUntilUnixNano int64, rescanPending, dryRun, ok bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	batch, exists := b.byTenant[tenantID]
	if !exists {
		return 0, false, false, false
	}
	return batch.QuiesceUntilUnixNano, batch.RescanAfterUnixNano > 0, batch.Mode.IsDryRun(), true
}

// setCancelled sets the tenant's batch cancelled flag under the write lock and reports the value it
// replaced. Reading and writing under one lock acquisition lets a caller tell whether it made the
// change or found it already made, which is what makes a retried cancel idempotent and stops two
// concurrent cancels from drawing different conclusions.
func (b *batchStore) setCancelled(tenantID string, cancelled bool) (previous bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	batch, ok := b.byTenant[tenantID]
	if !ok {
		return false
	}
	previous = batch.Cancelled
	batch.Cancelled = cancelled
	return previous
}

// commitCancel makes a tenant's cancel durable and only then publishes it in the store, as one
// operation with respect to every other manifest writer. It reports whether the batch was already
// cancelled, and returns ErrBatchNotFound if the tenant has no batch (including one removed while the
// commit was in flight, in which case the manifest is corrected by the next flush).
//
// Commit must precede publish because the readers of the in-memory flag act irreversibly: Next()
// discards a dequeued job and checkPendingRescans clears the skipped-block list, neither of which can
// be undone if the write then fails. Commit and publish must also be inseparable: in between, the
// manifest says cancelled while the store does not, so any concurrent flush of live state would write
// the cancel back out of existence. Holding writeMu across both closes that window — a flush blocked
// on it proceeds afterwards and marshals a store that now agrees with the disk.
func (b *batchStore) commitCancel(tenantID, localPath string) (alreadyCancelled bool, err error) {
	b.writeMu.Lock()
	defer b.writeMu.Unlock()

	b.mu.RLock()
	target, ok := b.byTenant[tenantID]
	if !ok {
		b.mu.RUnlock()
		return false, ErrBatchNotFound
	}
	alreadyCancelled = target.Cancelled
	batches := make([]*tempopb.RedactionBatch, 0, len(b.byTenant))
	for tenant, batch := range b.byTenant {
		if tenant == tenantID {
			// Serialize a copy carrying the cancel, leaving the stored batch untouched until the write
			// succeeds.
			cp := *target
			cp.Cancelled = true
			batches = append(batches, &cp)
			continue
		}
		batches = append(batches, batch)
	}
	msg := &tempopb.RedactionBatches{Batches: batches}
	data, err := msg.Marshal()
	b.mu.RUnlock()

	if err != nil {
		return false, fmt.Errorf("marshal batches: %w", err)
	}
	if err := os.MkdirAll(localPath, 0o700); err != nil {
		return false, fmt.Errorf("mkdir %s: %w", localPath, err)
	}
	if err := atomicWriteFile(data, filepath.Join(localPath, batchesFileName), batchesFileName); err != nil {
		return false, err
	}

	// Durable: publish it. Still under writeMu, so nothing has flushed the pre-cancel state in between.
	b.mu.Lock()
	batch, stillThere := b.byTenant[tenantID]
	if stillThere {
		batch.Cancelled = true
	}
	b.mu.Unlock()
	if !stillThere {
		// Removed while the commit was in flight; the caller must not report a successful cancel.
		return alreadyCancelled, ErrBatchNotFound
	}
	return alreadyCancelled, nil
}

func (b *batchStore) load(localPath string) error {
	// Ordered with the writers, so a reload cannot read a manifest that a concurrent write is
	// midway through replacing.
	b.writeMu.Lock()
	defer b.writeMu.Unlock()

	path := filepath.Join(localPath, batchesFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read %s: %w", path, err)
	}

	msg := &tempopb.RedactionBatches{}
	if err := msg.Unmarshal(data); err != nil {
		return fmt.Errorf("unmarshal batches: %w", err)
	}

	b.mu.Lock()
	defer b.mu.Unlock()
	b.byTenant = make(map[string]*tempopb.RedactionBatch, len(msg.Batches))
	for _, batch := range msg.Batches {
		b.byTenant[batch.TenantId] = batch
	}
	return nil
}

// --- Work methods delegating to batchStore ---

func (w *Work) AddBatch(batch *tempopb.RedactionBatch) error {
	return w.batches.add(batch)
}

// GetBatch returns a point-in-time shallow copy of the tenant's batch, or nil if there is none.
// Reading its fields is safe without holding any lock; mutating the copy has no effect on the store,
// so use SetBatchRescan/SetBatchCancelled/SetBatchQuiesceUntil to change batch state. Slice fields
// (TraceIds, SkippedCompactionJobIds) share backing arrays with the stored batch and must not be
// mutated in place. Same contract as ListBatches.
func (w *Work) GetBatch(tenantID string) *tempopb.RedactionBatch {
	return w.batches.get(tenantID)
}

func (w *Work) RemoveBatch(tenantID string) {
	w.batches.remove(tenantID)
}

func (w *Work) FlushBatchesToLocal(_ context.Context, localPath string) error {
	return w.batches.flush(localPath)
}

func (w *Work) LoadBatchesFromLocal(_ context.Context, localPath string) error {
	return w.batches.load(localPath)
}

func (w *Work) ListBatches() []*tempopb.RedactionBatch {
	return w.batches.list()
}

func (w *Work) SetBatchRescan(tenantID string, skippedJobIDs []string, rescanAfterUnixNano int64) {
	w.batches.setRescan(tenantID, skippedJobIDs, rescanAfterUnixNano)
}

func (w *Work) SetBatchQuiesceUntil(tenantID string, untilUnixNano int64) {
	w.batches.setQuiesceUntil(tenantID, untilUnixNano)
}

// BatchQuiescenceState returns a locked snapshot of a tenant's quiesce-until deadline, whether a
// rescan is pending, whether the batch is a dry-run, and whether it was cancelled; ok is false
// when no batch exists.
func (w *Work) BatchQuiescenceState(tenantID string) (quiesceUntilUnixNano int64, rescanPending, dryRun, ok bool) {
	return w.batches.quiescenceState(tenantID)
}

// SetBatchCancelled sets the tenant's batch cancelled flag, publishing it to readers, and reports the
// value it replaced so a retry can tell whether it made the change. Publish only after
// PersistBatchCancelled has made the cancel durable: the readers of this flag act irreversibly.
// Purging the batch's queued jobs is a separate step.
func (w *Work) SetBatchCancelled(tenantID string, cancelled bool) (previous bool) {
	return w.batches.setCancelled(tenantID, cancelled)
}

// SetBatchRescanIfCurrent arms the rescan only if the tenant's batch is still the one identified by
// batchID and is not cancelled, reporting whether it applied. Use this to commit a rescan result that
// was computed from an earlier snapshot.
func (w *Work) SetBatchRescanIfCurrent(tenantID, batchID string, skippedJobIDs []string, rescanAfterUnixNano int64) bool {
	return w.batches.setRescanIfCurrent(tenantID, batchID, skippedJobIDs, rescanAfterUnixNano)
}

// CommitBatchCancel makes a tenant's cancel durable and then publishes it, as one operation: on
// success the cancel is both on disk and visible to readers, and on failure neither. Reports whether
// the batch was already cancelled, so a retry is idempotent, and ErrBatchNotFound when the tenant has
// no batch.
func (w *Work) CommitBatchCancel(_ context.Context, tenantID, localPath string) (alreadyCancelled bool, err error) {
	return w.batches.commitCancel(tenantID, localPath)
}
