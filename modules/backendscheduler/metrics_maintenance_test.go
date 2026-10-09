package backendscheduler

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/modules/backendscheduler/work"
	"github.com/grafana/tempo/pkg/tempopb"
)

func applyBatch(tenant string) *tempopb.RedactionBatch {
	return &tempopb.RedactionBatch{BatchId: "b-" + tenant, TenantId: tenant}
}

func dryRunBatch(tenant string) *tempopb.RedactionBatch {
	return &tempopb.RedactionBatch{
		BatchId:  "b-" + tenant,
		TenantId: tenant,
		Mode:     tempopb.RedactionMode_REDACTION_MODE_DRY_RUN,
	}
}

// TestRecordMaintenancePausedApplyBatch covers the reason the gauge exists: while an apply-mode
// redaction holds a tenant, that tenant's compaction and retention are paused, and nothing else
// reports it. Compaction job dispatch is not a usable substitute — it keeps draining a queue built
// before the batch was submitted, so it dips rather than stopping.
func TestRecordMaintenancePausedApplyBatch(t *testing.T) {
	metricTenantMaintenancePaused.Reset()
	s := &BackendScheduler{work: work.New(work.Config{})}

	require.NoError(t, s.work.AddBatch(applyBatch("tenant-a")))
	s.recordMaintenancePaused()

	require.Equal(t, 1.0, testutil.ToFloat64(metricTenantMaintenancePaused.WithLabelValues("tenant-a")))
}

// TestRecordMaintenancePausedIgnoresDryRun pins that a dry-run gets no series at all. A dry-run
// rewrites nothing and never pauses maintenance, so reporting it as paused would send an operator
// looking for a compaction stall that is not happening.
func TestRecordMaintenancePausedIgnoresDryRun(t *testing.T) {
	metricTenantMaintenancePaused.Reset()
	s := &BackendScheduler{work: work.New(work.Config{})}

	require.NoError(t, s.work.AddBatch(dryRunBatch("tenant-dry")))
	s.recordMaintenancePaused()

	require.Equal(t, 0, testutil.CollectAndCount(metricTenantMaintenancePaused),
		"a dry-run never pauses maintenance, so it should not appear at all")
}

// TestRecordMaintenancePausedReportsResume is the operator's actual question — has the quiet period
// elapsed and is compaction running again. The series must survive at 0 rather than being deleted:
// a deleted series can only be detected with absent(), whereas a 1 -> 0 edge is directly visible on
// a graph and alertable. This is a deliberate difference from jobs_pending, which deletes drained
// series because its label set is per (tenant, job_type) and unbounded; the set of tenants holding a
// redaction gate in one process lifetime is a handful.
func TestRecordMaintenancePausedReportsResume(t *testing.T) {
	metricTenantMaintenancePaused.Reset()
	s := &BackendScheduler{work: work.New(work.Config{})}

	require.NoError(t, s.work.AddBatch(applyBatch("tenant-a")))
	s.recordMaintenancePaused()
	require.Equal(t, 1.0, testutil.ToFloat64(metricTenantMaintenancePaused.WithLabelValues("tenant-a")))

	// Teardown after quiescence removes the batch, which re-enables compaction and retention.
	s.work.RemoveBatch("tenant-a")
	s.recordMaintenancePaused()

	// Counted before reading the value: ToFloat64(WithLabelValues(...)) would re-create a deleted
	// series at 0 and assert successfully against its own side effect.
	require.Equal(t, 1, testutil.CollectAndCount(metricTenantMaintenancePaused),
		"the series is retained so the 1 -> 0 transition can be seen and alerted on")
	require.Equal(t, 0.0, testutil.ToFloat64(metricTenantMaintenancePaused.WithLabelValues("tenant-a")),
		"resume must be observable as a value, not as a missing series")
}
