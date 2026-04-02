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

func (b *batchStore) get(tenantID string) *tempopb.RedactionBatch {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.byTenant[tenantID]
}

func (b *batchStore) remove(tenantID string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.byTenant, tenantID)
}

func (b *batchStore) hasActive(tenantID string) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	_, ok := b.byTenant[tenantID]
	return ok
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
		out = append(out, batch)
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

// GetBatch returns a live pointer into batchStore. Callers must treat the returned
// value as read-only; use SetBatchRescan to mutate rescan fields under the write lock.
// TODO: return a copy or add narrower accessor methods so the read-only contract is
// enforced by the type system rather than convention.
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
