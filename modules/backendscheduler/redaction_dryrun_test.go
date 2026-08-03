package backendscheduler

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/pkg/tempopb"
)

// TestDryRunBatchDoesNotBlockCompaction verifies a dry-run batch is not an exclusive tenant
// operation: it must not gate compaction/retention (TenantPending false), though it still
// occupies the tenant's single batch slot (HasBatch true).
func TestDryRunBatchDoesNotBlockCompaction(t *testing.T) {
	_, s := newQuiescenceScheduler(t)
	tenant := "t-dry"
	require.NoError(t, s.work.AddBatch(&tempopb.RedactionBatch{
		BatchId: "b", TenantId: tenant, CreatedAtUnixNano: time.Now().UnixNano(),
		Mode: tempopb.RedactionMode_REDACTION_MODE_DRY_RUN,
	}))
	require.False(t, s.work.TenantPending(tenant), "a dry-run batch must not block compaction/retention")
	require.NotNil(t, s.work.GetBatch(tenant), "the dry-run batch still occupies the tenant's batch slot")
}

// TestApplyBatchBlocksCompaction guards the apply path: an apply batch remains an exclusive
// operation that gates compaction (TenantPending true).
func TestApplyBatchBlocksCompaction(t *testing.T) {
	_, s := newQuiescenceScheduler(t)
	tenant := "t-apply"
	require.NoError(t, s.work.AddBatch(&tempopb.RedactionBatch{
		BatchId: "b", TenantId: tenant, CreatedAtUnixNano: time.Now().UnixNano(),
		Mode: tempopb.RedactionMode_REDACTION_MODE_APPLY,
	}))
	require.True(t, s.work.TenantPending(tenant), "an apply batch must block compaction")
	require.NotNil(t, s.work.GetBatch(tenant))
}

// TestDryRunBatchRemovedImmediatelyOnCompletion verifies a completed dry-run batch is removed
// at once rather than entering quiescence: a dry-run writes nothing, so there is no
// cleanup-window race to hold compaction for, and the tenant's slot frees immediately.
func TestDryRunBatchRemovedImmediatelyOnCompletion(t *testing.T) {
	ctx, s := newQuiescenceScheduler(t)
	tenant := "t-dry-done"
	require.NoError(t, s.work.AddBatch(&tempopb.RedactionBatch{
		BatchId: "b", TenantId: tenant, CreatedAtUnixNano: time.Now().UnixNano(),
		Mode: tempopb.RedactionMode_REDACTION_MODE_DRY_RUN,
	}))
	// No jobs and no rescan pending → the batch is done.
	s.cleanupBatchIfDone(ctx, tenant)
	require.Nil(t, s.work.GetBatch(tenant), "a completed dry-run batch is removed immediately, not quiesced")
}
