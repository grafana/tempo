package provider

import (
	"context"
	"flag"
	"os"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/grafana/tempo/modules/backendscheduler/work"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

// staticTenantLister implements TenantLister with a fixed tenant list.
type staticTenantLister struct{ tenants []string }

func (s *staticTenantLister) Tenants() []string { return s.tenants }

func TestRetentionProvider(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	cfg := RetentionConfig{}
	cfg.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})
	cfg.Interval = 10 * time.Millisecond

	workCfg := work.Config{}
	workCfg.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})
	w := work.New(workCfg)

	tenants := &staticTenantLister{tenants: []string{"tenant-a", "tenant-b"}}

	logger := log.NewLogfmtLogger(os.Stderr)

	limits, err := overrides.NewOverrides(overrides.Config{Defaults: overrides.Overrides{}}, nil, prometheus.NewRegistry())
	require.NoError(t, err)
	p := NewRetentionProvider(cfg, logger, tenants, limits, w)

	jobChan := p.Start(ctx)

	seen := make(map[string]bool)
	for job := range jobChan {
		require.NotNil(t, job)
		require.Equal(t, tempopb.JobType_JOB_TYPE_RETENTION, job.Type)
		require.NotEmpty(t, job.Tenant(), "retention jobs must have a tenant set")
		seen[job.Tenant()] = true

		err := w.AddJob(job)
		require.NoError(t, err)
		job.Start() // mark running so the provider skips this tenant next tick
	}

	// Every tenant should have received exactly one retention job before the
	// context deadline.
	for _, tenantID := range tenants.tenants {
		require.True(t, seen[tenantID], "expected retention job for tenant %s", tenantID)
	}
}

func TestRetentionProviderSkipsRedactionPending(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	cfg := RetentionConfig{}
	cfg.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})
	cfg.Interval = 10 * time.Millisecond

	workCfg := work.Config{}
	workCfg.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})
	w := work.New(workCfg)

	// Submit a redaction for tenant-a the way production does: a batch plus a
	// pending redaction job. The batch is what marks the tenant pending.
	require.NoError(t, w.AddBatch(&tempopb.RedactionBatch{
		BatchId:  "batch-1",
		TenantId: "tenant-a",
		TraceIds: [][]byte{[]byte("trace-1")},
	}))
	err := w.AddPendingJobs([]*work.Job{{
		ID:   "redact-1",
		Type: tempopb.JobType_JOB_TYPE_REDACTION,
		JobDetail: tempopb.JobDetail{
			Tenant:    "tenant-a",
			BatchId:   "batch-1",
			Redaction: &tempopb.RedactionDetail{BlockId: "block-1"},
		},
	}})
	require.NoError(t, err)

	tenants := &staticTenantLister{tenants: []string{"tenant-a", "tenant-b"}}
	logger := log.NewLogfmtLogger(os.Stderr)

	limits, err := overrides.NewOverrides(overrides.Config{Defaults: overrides.Overrides{}}, nil, prometheus.NewRegistry())
	require.NoError(t, err)
	p := NewRetentionProvider(cfg, logger, tenants, limits, w)
	jobChan := p.Start(ctx)

	seen := make(map[string]bool)
	for job := range jobChan {
		require.NotNil(t, job)
		seen[job.Tenant()] = true
		err := w.AddJob(job)
		require.NoError(t, err)
		job.Start()
	}

	require.False(t, seen["tenant-a"], "retention must be skipped for tenant with pending redaction")
	require.True(t, seen["tenant-b"], "retention must run for tenant without pending redaction")
}

func TestRetentionProviderRolloutCompat(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	cfg := RetentionConfig{}
	cfg.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})
	cfg.Interval = 10 * time.Millisecond

	workCfg := work.Config{}
	workCfg.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})
	w := work.New(workCfg)

	// Simulate a legacy global retention job (tenant="") still running.
	globalJob := &work.Job{
		ID:   "retention-global",
		Type: tempopb.JobType_JOB_TYPE_RETENTION,
		JobDetail: tempopb.JobDetail{
			Tenant:    "",
			Retention: &tempopb.RetentionDetail{},
		},
	}
	require.NoError(t, w.AddJob(globalJob))
	globalJob.Start()

	tenants := &staticTenantLister{tenants: []string{"tenant-a"}}
	logger := log.NewLogfmtLogger(os.Stderr)

	limits, err := overrides.NewOverrides(overrides.Config{Defaults: overrides.Overrides{}}, nil, prometheus.NewRegistry())
	require.NoError(t, err)
	p := NewRetentionProvider(cfg, logger, tenants, limits, w)
	jobChan := p.Start(ctx)

	var receivedJobs []*work.Job
	for job := range jobChan {
		receivedJobs = append(receivedJobs, job)
	}

	// No per-tenant jobs should be emitted while a global job is in flight.
	require.Empty(t, receivedJobs, "no jobs should be emitted while a global retention job is running")
}

// TestRetentionProviderGatesOnActiveBatch covers the rescan-wait window: a
// redaction batch is active (TenantPending true) but no redaction jobs are in
// flight, so HasJobsForTenant(REDACTION) is false. Retention must still skip the
// tenant — otherwise it can compact-out a skipped block before the rescan covers
// it.
func TestRetentionProviderGatesOnActiveBatch(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	cfg := RetentionConfig{}
	cfg.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})
	cfg.Interval = 10 * time.Millisecond

	workCfg := work.Config{}
	workCfg.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})
	w := work.New(workCfg)

	// Active redaction batch for tenant-a in its rescan-wait window: the batch
	// exists with a pending rescan, but there are NO redaction jobs in flight.
	require.NoError(t, w.AddBatch(&tempopb.RedactionBatch{
		BatchId:                 "batch-1",
		TenantId:                "tenant-a",
		TraceIds:                [][]byte{[]byte("trace-1")},
		SkippedCompactionJobIds: []string{"comp-1"},
		RescanAfterUnixNano:     time.Now().Add(time.Hour).UnixNano(),
	}))
	require.True(t, w.TenantPending("tenant-a"))
	require.False(t, w.HasJobsForTenant("tenant-a", tempopb.JobType_JOB_TYPE_REDACTION),
		"precondition: no redaction jobs in flight during the rescan-wait window")

	tenants := &staticTenantLister{tenants: []string{"tenant-a", "tenant-b"}}
	logger := log.NewLogfmtLogger(os.Stderr)

	limits, err := overrides.NewOverrides(overrides.Config{Defaults: overrides.Overrides{}}, nil, prometheus.NewRegistry())
	require.NoError(t, err)
	p := NewRetentionProvider(cfg, logger, tenants, limits, w)
	jobChan := p.Start(ctx)

	seen := make(map[string]bool)
	for job := range jobChan {
		seen[job.Tenant()] = true
		require.NoError(t, w.AddJob(job))
		job.Start()
	}

	require.False(t, seen["tenant-a"], "retention must be skipped while a redaction batch is active")
	require.True(t, seen["tenant-b"], "retention must still run for tenants without an active batch")
}
