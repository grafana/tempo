package work

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/pkg/tempopb"
)

func testBatch(tenant string) *tempopb.RedactionBatch {
	return &tempopb.RedactionBatch{
		BatchId:           "b-" + tenant,
		TenantId:          tenant,
		CreatedAtUnixNano: time.Now().UnixNano(),
	}
}

// TestPersistBatchCancelledCommitsBeforePublishing pins the ordering the cancel path depends on: the
// cancel is written to the manifest (committed) without being made visible in memory (published).
//
// The order matters because the readers of the in-memory flag act destructively — Next() discards a
// dequeued job, checkPendingRescans clears the skipped-block list — and neither can be undone. If the
// flag were published first and the write then failed, those effects would already have happened while
// the operator was told the cancel failed. Publishing only after a successful commit means an
// unpersisted cancel is never observable, so "cancel failed" always means nothing changed.
func TestPersistBatchCancelledCommitsBeforePublishing(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	w := New(Config{})
	tenant := "t1"
	require.NoError(t, w.AddBatch(testBatch(tenant)))

	require.NoError(t, w.PersistBatchCancelled(ctx, tenant, dir))

	require.False(t, w.GetBatch(tenant).Cancelled,
		"persisting must not publish the flag in memory; the caller publishes only after the commit succeeds")

	// The manifest on disk carries the cancel, so a restart reconstructs it.
	reloaded := New(Config{})
	require.NoError(t, reloaded.LoadBatchesFromLocal(ctx, dir))
	b := reloaded.GetBatch(tenant)
	require.NotNil(t, b)
	require.True(t, b.Cancelled, "the cancel is durable once PersistBatchCancelled returns nil")
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
	require.NoError(t, w.AddBatch(testBatch(tenant)))

	require.Error(t, w.PersistBatchCancelled(ctx, tenant, badPath))
	require.False(t, w.GetBatch(tenant).Cancelled, "a failed commit publishes nothing")
	require.NoFileExists(t, filepath.Join(badPath, batchesFileName))
}

// TestPersistBatchCancelledUnknownTenant verifies persisting a cancel for a tenant with no batch is an
// error rather than a manifest write, so a cancel can't invent state for a batch that is already gone.
func TestPersistBatchCancelledUnknownTenant(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	w := New(Config{})

	require.Error(t, w.PersistBatchCancelled(ctx, "absent", dir))
}

// TestSetBatchRescanIfCurrentRejectsCancelled verifies a rescan cannot be armed on a cancelled batch.
// The rescan sweep scans on a snapshot and takes a long time, so a cancel can land mid-scan; committing
// the result unconditionally would re-arm the rescan the cancel had just cleared, holding the tenant's
// compaction for another rescan delay with no work outstanding.
func TestSetBatchRescanIfCurrentRejectsCancelled(t *testing.T) {
	w := New(Config{})
	tenant := "t1"
	b := testBatch(tenant)
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
	require.NoError(t, w.AddBatch(testBatch(tenant)))

	require.False(t, w.SetBatchRescanIfCurrent(tenant, "some-older-batch", []string{"j1"}, 12345),
		"a scan started under a previous batch does not commit onto the current one")
	require.Zero(t, w.GetBatch(tenant).RescanAfterUnixNano)
}

// TestSetBatchRescanIfCurrentAppliesWhenCurrent verifies the normal path still arms the rescan.
func TestSetBatchRescanIfCurrentAppliesWhenCurrent(t *testing.T) {
	w := New(Config{})
	tenant := "t1"
	b := testBatch(tenant)
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
	require.NoError(t, w.AddBatch(testBatch(tenant)))

	require.False(t, w.SetBatchCancelled(tenant, true), "first cancel reports the batch was not cancelled")
	require.True(t, w.SetBatchCancelled(tenant, true), "a repeat cancel reports it was already cancelled")
	require.True(t, w.GetBatch(tenant).Cancelled)
}
