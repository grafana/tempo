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
	// Hold the read lock through Marshal so that clearRescan mutations cannot race
	// with field reads inside proto.Marshal.
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

func (b *batchStore) setQuiesceUntil(tenantID string, untilUnixNano int64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if batch, ok := b.byTenant[tenantID]; ok {
		batch.QuiesceUntilUnixNano = untilUnixNano
	}
}

// quiescenceState reads a tenant's quiescence-relevant fields under the lock, returning a
// snapshot so callers never touch the live batch pointer's mutable fields unsynchronized.
func (b *batchStore) quiescenceState(tenantID string) (quiesceUntilUnixNano int64, rescanPending, dryRun, cancelled, ok bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	batch, exists := b.byTenant[tenantID]
	if !exists {
		return 0, false, false, false, false
	}
	return batch.QuiesceUntilUnixNano, batch.RescanAfterUnixNano > 0, batch.Mode.IsDryRun(), batch.Cancelled, true
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

// persistCancelled writes the manifest with tenantID's batch marked cancelled, WITHOUT changing the
// in-memory store. On success the cancel is durable but not yet visible to readers; the caller
// publishes it with setCancelled.
//
// Committing before publishing is required because the readers of the in-memory flag act
// irreversibly: a dequeued job is discarded at assignment, and a cancelled batch's skipped-block list
// is cleared rather than rescanned. Publishing first would let those happen during the write, so a
// write failure could not be undone and the operator would be told the cancel failed after work had
// already been dropped. Returns an error if the tenant has no batch.
func (b *batchStore) persistCancelled(tenantID, localPath string) error {
	b.mu.RLock()
	target, ok := b.byTenant[tenantID]
	if !ok {
		b.mu.RUnlock()
		return fmt.Errorf("no batch for tenant %s", tenantID)
	}
	batches := make([]*tempopb.RedactionBatch, 0, len(b.byTenant))
	for tenant, batch := range b.byTenant {
		if tenant == tenantID {
			// Serialize a copy carrying the cancel, leaving the stored batch untouched.
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
		return fmt.Errorf("marshal batches: %w", err)
	}
	if err := os.MkdirAll(localPath, 0o700); err != nil {
		return fmt.Errorf("mkdir %s: %w", localPath, err)
	}
	return atomicWriteFile(data, filepath.Join(localPath, batchesFileName), batchesFileName)
}

// load reads batches.pb from localPath. Missing file is not an error (clean start).
func (b *batchStore) load(localPath string) error {
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
func (w *Work) BatchQuiescenceState(tenantID string) (quiesceUntilUnixNano int64, rescanPending, dryRun, cancelled, ok bool) {
	return w.batches.quiescenceState(tenantID)
}

// SetBatchCancelled sets the tenant's batch cancelled flag. A cancelled batch has its remaining
// pending jobs purged separately; in-flight jobs finish, then it is removed immediately (no
// quiescence). Pass false to revert (e.g. when a cancel could not be persisted).
func (w *Work) SetBatchCancelled(tenantID string, cancelled bool) (previous bool) {
	return w.batches.setCancelled(tenantID, cancelled)
}

// PersistBatchCancelled makes a tenant's cancel durable without publishing it in memory, so that no
// reader can act on a cancel that is not yet persisted. Publish it with SetBatchCancelled once this
// returns nil; if it returns an error, nothing changed and the operation is safe to retry.
func (w *Work) PersistBatchCancelled(_ context.Context, tenantID, localPath string) error {
	return w.batches.persistCancelled(tenantID, localPath)
}
