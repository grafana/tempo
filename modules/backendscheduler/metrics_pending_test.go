package backendscheduler

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/modules/backendscheduler/work"
	"github.com/grafana/tempo/pkg/tempopb"
)

// TestRecordPendingJobs covers publishing queue depth, and in particular that a drained queue stops
// being reported while a surviving one keeps its value.
//
// The gauge is keyed by tenant, and a tenant whose queue empties stops appearing in the snapshot — it
// never reports zero. Without removing those series their last non-zero value would persist for the
// process lifetime: a finished redaction would look permanently backlogged, and an autoscaler
// triggering on this would hold the scale-up forever.
//
// Removal is deliberately done by deleting drained label sets rather than resetting the vector. A
// scrape landing inside a Reset would observe series missing and read the total lower than it is,
// which is the spurious scale-down this metric exists to prevent.
func TestRecordPendingJobs(t *testing.T) {
	metricJobsPending.Reset() // isolate from other tests in this package
	s := &BackendScheduler{work: work.New(work.Config{})}

	redaction := func(id, tenant, block string) *work.Job {
		return &work.Job{
			ID:   id,
			Type: tempopb.JobType_JOB_TYPE_REDACTION,
			JobDetail: tempopb.JobDetail{
				Tenant:    tenant,
				Redaction: &tempopb.RedactionDetail{BlockId: block},
			},
		}
	}

	require.NoError(t, s.work.AddPendingJobs([]*work.Job{
		redaction("r1", "tenant-a", "block-1"),
		redaction("r2", "tenant-a", "block-2"),
		redaction("r3", "tenant-b", "block-1"),
		{
			ID:   "c1",
			Type: tempopb.JobType_JOB_TYPE_COMPACTION,
			JobDetail: tempopb.JobDetail{
				Tenant:     "tenant-a",
				Compaction: &tempopb.CompactionDetail{Input: []string{"block-9"}},
			},
		},
	}))

	s.recordPendingJobs()
	require.Equal(t, 2.0, testutil.ToFloat64(metricJobsPending.WithLabelValues("tenant-a", "JOB_TYPE_REDACTION")))
	require.Equal(t, 1.0, testutil.ToFloat64(metricJobsPending.WithLabelValues("tenant-b", "JOB_TYPE_REDACTION")))
	require.Equal(t, 1.0, testutil.ToFloat64(metricJobsPending.WithLabelValues("tenant-a", "JOB_TYPE_COMPACTION")))
	require.Equal(t, 3, testutil.CollectAndCount(metricJobsPending), "one series per tenant and type with queued work")

	// Drain every redaction job. NextPendingJob is type-scoped, so the compaction job stays queued
	// and tenant-a keeps exactly one series while tenant-b loses its only one.
	for s.work.NextPendingJob(tempopb.JobType_JOB_TYPE_REDACTION) != nil { //nolint:revive // drain
	}

	s.recordPendingJobs()
	require.Equal(t, 1, testutil.CollectAndCount(metricJobsPending),
		"drained queues must lose their series rather than keep their last value")
	require.Equal(t, 1.0, testutil.ToFloat64(metricJobsPending.WithLabelValues("tenant-a", "JOB_TYPE_COMPACTION")),
		"the queue that was never drained keeps its value; only drained series are removed")

	// Publishing again with nothing changed must be stable, not oscillate as labels are re-recorded.
	s.recordPendingJobs()
	require.Equal(t, 1, testutil.CollectAndCount(metricJobsPending), "a repeat publish must be idempotent")
	require.Equal(t, 1.0, testutil.ToFloat64(metricJobsPending.WithLabelValues("tenant-a", "JOB_TYPE_COMPACTION")))
}
