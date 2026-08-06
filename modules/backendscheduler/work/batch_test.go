package work

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/pkg/tempopb"
)

// testTenant is the tenant every batch_test case operates on; batch state is per-tenant and none of
// these tests need a second one.
const testTenant = "t1"

func testBatch() *tempopb.RedactionBatch {
	return &tempopb.RedactionBatch{
		BatchId:           "b-" + testTenant,
		TenantId:          testTenant,
		CreatedAtUnixNano: time.Now().UnixNano(),
	}
}

// TestCommitBatchCancelIsDurableAndVisible verifies a successful cancel is both on disk and visible in
// memory. Neither half is optional: the readers of the in-memory flag act irreversibly (Next() discards
// a dequeued job, checkPendingRescans clears the skipped-block list), so the flag must not become
// visible until the write has succeeded — and the two must be inseparable, because in between the
// manifest and the store disagree and any concurrent flush of live state would undo the write.
func TestCommitBatchCancelIsDurableAndVisible(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	w := New(Config{})
	require.NoError(t, w.AddBatch(testBatch()))

	already, err := w.CommitBatchCancel(ctx, testTenant, dir)
	require.NoError(t, err)
	require.False(t, already, "the first cancel reports the batch was not already cancelled")

	require.True(t, w.GetBatch(testTenant).Cancelled, "a committed cancel is visible to readers")

	reloaded := New(Config{})
	require.NoError(t, reloaded.LoadBatchesFromLocal(ctx, dir))
	b := reloaded.GetBatch(testTenant)
	require.NotNil(t, b)
	require.True(t, b.Cancelled, "a committed cancel survives a restart")
}

// TestCommitBatchCancelIsIdempotent verifies a retried cancel succeeds and says so, which is what lets
// an operator re-run the command after an interrupted attempt without a confusing failure.
func TestCommitBatchCancelIsIdempotent(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	w := New(Config{})
	require.NoError(t, w.AddBatch(testBatch()))

	_, err := w.CommitBatchCancel(ctx, testTenant, dir)
	require.NoError(t, err)

	already, err := w.CommitBatchCancel(ctx, testTenant, dir)
	require.NoError(t, err)
	require.True(t, already, "a repeat cancel reports the batch was already cancelled")
}

// TestPersistBatchCancelledFailureChangesNothing verifies a failed commit leaves no trace: not in
// memory, and not on disk. That is what lets the RPC report failure honestly and be safely retried.
func TestPersistBatchCancelledFailureChangesNothing(t *testing.T) {
	ctx := context.Background()
	// A child of a regular file, so MkdirAll inside the write fails.
	notADir := filepath.Join(t.TempDir(), "notadir")
	require.NoError(t, os.WriteFile(notADir, []byte("x"), 0o600))
	badPath := filepath.Join(notADir, "sub")

	w := New(Config{})
	tenant := "t1"
	require.NoError(t, w.AddBatch(testBatch()))

	_, err := w.CommitBatchCancel(ctx, tenant, badPath)
	require.Error(t, err)
	require.False(t, w.GetBatch(tenant).Cancelled, "a failed commit publishes nothing")
	require.NoFileExists(t, filepath.Join(badPath, batchesFileName))
}

// TestPersistBatchCancelledUnknownTenant verifies persisting a cancel for a tenant with no batch is an
// error rather than a manifest write, so a cancel can't invent state for a batch that is already gone.
func TestPersistBatchCancelledUnknownTenant(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	w := New(Config{})

	_, err := w.CommitBatchCancel(ctx, "absent", dir)
	require.ErrorIs(t, err, ErrBatchNotFound)
}

// TestManifestWritesAreSerialized verifies concurrent manifest writers cannot reorder, so a durable
// cancel is never reverted by an older snapshot.
//
// Every other writer marshals the live store, so two of them racing wrote the same truth and a lost
// update was harmless. persistCancelled breaks that: it deliberately marshals a copy that differs from
// the store, so if a plain flush marshals before the cancel but renames after it, the file reverts to
// not-cancelled — while the handler has already published the flag and reported success. A restart then
// runs the rescan and rewrites blocks the operator asked to stop.
//
// The write must therefore be ordered with the marshal, not just protected during it. Meaningful under
// -race; the assertion is that the last cancel to be committed is what the manifest ends up holding.
func TestManifestWritesAreSerialized(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	w := New(Config{})
	tenant := "t1"
	require.NoError(t, w.AddBatch(testBatch()))

	// One goroutine repeatedly writes the manifest from the live store (Cancelled=false, as the
	// maintenance tick and UpdateJob both do); the other commits the cancel.
	var wg sync.WaitGroup
	stop := make(chan struct{})
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
				_ = w.FlushBatchesToLocal(ctx, dir)
			}
		}
	}()

	_, err := w.CommitBatchCancel(ctx, tenant, dir)
	require.NoError(t, err)
	close(stop)
	wg.Wait()

	// Whatever interleaving occurred, a writer that started before the commit must not have landed
	// after it: the committed cancel is still on disk.
	reloaded := New(Config{})
	require.NoError(t, reloaded.LoadBatchesFromLocal(ctx, dir))
	b := reloaded.GetBatch(tenant)
	require.NotNil(t, b, "the batch must not be erased by a concurrent writer")
	require.True(t, b.Cancelled, "a committed cancel must not be reverted by an older snapshot")
}

// TestSetBatchRescanIfCurrentRejectsCancelled verifies a rescan cannot be armed on a cancelled batch.
// The rescan sweep scans on a snapshot and takes a long time, so a cancel can land mid-scan; committing
// the result unconditionally would re-arm the rescan the cancel had just cleared, holding the tenant's
// compaction for another rescan delay with no work outstanding.
func TestSetBatchRescanIfCurrentRejectsCancelled(t *testing.T) {
	w := New(Config{})
	tenant := "t1"
	b := testBatch()
	require.NoError(t, w.AddBatch(b))
	w.SetBatchCancelled(tenant, true)

	require.False(t, w.SetBatchRescanIfCurrent(tenant, b.BatchId, []string{"j1"}, 12345),
		"a cancelled batch never re-arms its rescan")
	require.Zero(t, w.GetBatch(tenant).RescanAfterUnixNano)
	require.Empty(t, w.GetBatch(tenant).SkippedCompactionJobIds)
}

// TestSetBatchRescanIfCurrentRejectsReplacedBatch verifies a rescan result cannot be committed onto a
// different batch than the one it was computed from. Cancel removes the batch once quiescence ends so
// the operator can resubmit, and a resubmission gets a new batch ID; applying an in-flight scan's
// result to it would arm a rescan for blocks the new batch never scheduled.
func TestSetBatchRescanIfCurrentRejectsReplacedBatch(t *testing.T) {
	w := New(Config{})
	tenant := "t1"
	require.NoError(t, w.AddBatch(testBatch()))

	require.False(t, w.SetBatchRescanIfCurrent(tenant, "some-older-batch", []string{"j1"}, 12345),
		"a scan started under a previous batch does not commit onto the current one")
	require.Zero(t, w.GetBatch(tenant).RescanAfterUnixNano)
}

// TestSetBatchRescanIfCurrentAppliesWhenCurrent verifies the normal path still arms the rescan.
func TestSetBatchRescanIfCurrentAppliesWhenCurrent(t *testing.T) {
	w := New(Config{})
	tenant := "t1"
	b := testBatch()
	require.NoError(t, w.AddBatch(b))

	require.True(t, w.SetBatchRescanIfCurrent(tenant, b.BatchId, []string{"j1"}, 12345))
	got := w.GetBatch(tenant)
	require.Equal(t, int64(12345), got.RescanAfterUnixNano)
	require.Equal(t, []string{"j1"}, got.SkippedCompactionJobIds)
}

// TestSetBatchCancelledReportsPreviousValue verifies the setter reports what it replaced, so a retried
// cancel is idempotent and two concurrent cancels can tell which of them actually made the change.
func TestSetBatchCancelledReportsPreviousValue(t *testing.T) {
	w := New(Config{})
	tenant := "t1"
	require.NoError(t, w.AddBatch(testBatch()))

	require.False(t, w.SetBatchCancelled(tenant, true), "first cancel reports the batch was not cancelled")
	require.True(t, w.SetBatchCancelled(tenant, true), "a repeat cancel reports it was already cancelled")
	require.True(t, w.GetBatch(tenant).Cancelled)
}
