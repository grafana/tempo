package provider

import (
	"context"
	"flag"
	"os"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/grafana/tempo/v3/modules/backendscheduler/work"
	"github.com/grafana/tempo/v3/pkg/tempopb"
	"github.com/stretchr/testify/require"
)

func TestRedactionProvider_DrainsPendingQueue(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cfg := RedactionConfig{}
	cfg.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})
	cfg.PollInterval = 20 * time.Millisecond

	workCfg := work.Config{}
	workCfg.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})
	w := work.New(workCfg)

	// Add pending redaction jobs
	jobs := []*work.Job{
		createRedactionJob("r1", "tenant-a", "block-1", nil),
		createRedactionJob("r2", "tenant-a", "block-2", nil),
	}
	require.NoError(t, w.AddPendingJobs(jobs))

	logger := log.NewLogfmtLogger(os.Stderr)
	p := NewRedactionProvider(cfg, logger, w)
	jobChan := p.Start(ctx)

	var received []*work.Job
	for len(received) < 2 {
		select {
		case j := <-jobChan:
			require.NotNil(t, j)
			require.Equal(t, tempopb.JobType_JOB_TYPE_REDACTION, j.Type)
			received = append(received, j)
		case <-ctx.Done():
			t.Fatal("timeout waiting for jobs")
		}
	}

	// Both jobs should have been popped from the pending queue.
	require.Len(t, received, 2)
	require.Empty(t, w.ListAllPendingJobs())
}

// TestRedactionProvider_ReleasesInFlightOnShutdown verifies a job dequeued but never received (no worker
// asked for it) does not stay counted in flight once the provider stops.
func TestRedactionProvider_ReleasesInFlightOnShutdown(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := RedactionConfig{}
	cfg.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})
	cfg.PollInterval = 20 * time.Millisecond

	workCfg := work.Config{}
	workCfg.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})
	w := work.New(workCfg)
	require.NoError(t, w.AddPendingJobs([]*work.Job{createRedactionJob("r1", "tenant-a", "block-1", nil)}))

	p := NewRedactionProvider(cfg, log.NewLogfmtLogger(os.Stderr), w)
	_ = p.Start(ctx) // nobody receives, so the provider blocks on the send

	require.Eventually(t, func() bool { return len(w.ListAllPendingJobs()) == 0 }, 2*time.Second, 5*time.Millisecond)
	require.True(t, w.HasJobsForTenant("tenant-a", tempopb.JobType_JOB_TYPE_REDACTION),
		"the dequeued job is counted in flight while blocked on the send")

	cancel()

	require.Eventually(t, func() bool {
		return !w.HasJobsForTenant("tenant-a", tempopb.JobType_JOB_TYPE_REDACTION)
	}, 2*time.Second, 5*time.Millisecond, "shutdown must release the in-flight count")
}

func createRedactionJob(id, tenantID, blockID string, traceIDs [][]byte) *work.Job {
	return &work.Job{
		ID:   id,
		Type: tempopb.JobType_JOB_TYPE_REDACTION,
		JobDetail: tempopb.JobDetail{
			Tenant: tenantID,
			Redaction: &tempopb.RedactionDetail{
				BlockId:  blockID,
				TraceIds: traceIDs,
			},
		},
	}
}
