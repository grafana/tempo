package backendscheduler

import (
	"context"
	"flag"
	"testing"
	"time"

	"github.com/grafana/dskit/user"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/tempodb/backend"
)

// newStatusScheduler builds a scheduler over a store holding blockCount blocks for tenant, all
// within the last few days so any window in the tests can select them.
func newStatusScheduler(ctx context.Context, t *testing.T, tenant string, blockCount int) *BackendScheduler {
	t.Helper()

	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})
	tmpDir := t.TempDir()
	cfg.LocalWorkPath = tmpDir

	store, rr, ww := newStore(ctx, t, tmpDir)
	t.Cleanup(store.Shutdown)

	base := time.Now().Add(-3 * 24 * time.Hour)
	ranges := make([][2]time.Time, blockCount)
	for i := range ranges {
		start := base.Add(time.Duration(i) * time.Hour)
		ranges[i] = [2]time.Time{start, start.Add(time.Hour)}
	}
	writeTenantBlocksWithRanges(ctx, t, backend.NewWriter(ww), tenant, ranges)
	require.Eventually(t, func() bool { return len(store.BlockMetas(tenant)) == blockCount },
		5*time.Second, 50*time.Millisecond, "blocks must be polled before submitting")

	limits, err := overrides.NewOverrides(overrides.Config{Defaults: overrides.Overrides{}}, nil, prometheus.NewRegistry())
	require.NoError(t, err)
	s, err := New(cfg, store, limits, rr, ww)
	require.NoError(t, err)
	return s
}

func submitAll(ctx context.Context, t *testing.T, s *BackendScheduler, tenant string) *tempopb.SubmitRedactionResponse {
	t.Helper()
	resp, err := s.SubmitRedaction(user.InjectOrgID(ctx, tenant), &tempopb.SubmitRedactionRequest{
		Selector: &tempopb.SubmitRedactionRequest_Query{
			Query: &tempopb.TraceQLSelector{Query: `{resource.namespace = "checkout"}`},
		},
	})
	require.NoError(t, err)
	return resp
}

// TestGetRedactionStatusNoBatch pins that absence of a batch is a normal answer, not an error: a
// finished redaction leaves no batch behind, so an operator polling after completion must get a
// clean "not active" rather than NotFound.
func TestGetRedactionStatusNoBatch(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s := newStatusScheduler(ctx, t, "t-status-none", 1)

	resp, err := s.GetRedactionStatus(user.InjectOrgID(ctx, "t-status-none"), &tempopb.GetRedactionStatusRequest{})
	require.NoError(t, err)
	require.False(t, resp.Active)
	require.Empty(t, resp.BatchId)
}

// TestGetRedactionStatusRequiresOrgID pins that the tenant comes from the header. There is no
// tenant field on the request, so a caller without an org ID must be refused rather than
// defaulting to some tenant's redaction state.
func TestGetRedactionStatusRequiresOrgID(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s := newStatusScheduler(ctx, t, "t-status-auth", 1)

	_, err := s.GetRedactionStatus(ctx, &tempopb.GetRedactionStatusRequest{})
	require.Error(t, err)
	require.Equal(t, codes.Unauthenticated, status.Code(err))
}

// TestGetRedactionStatusReportsSubmission covers the fresh-submission shape: everything the
// operator needs to confirm they submitted what they meant, with the whole batch outstanding.
func TestGetRedactionStatusReportsSubmission(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	const tenant = "t-status-fresh"
	s := newStatusScheduler(ctx, t, tenant, 3)

	sub := submitAll(ctx, t, s, tenant)
	require.Equal(t, int32(3), sub.JobsCreated)

	resp, err := s.GetRedactionStatus(user.InjectOrgID(ctx, tenant), &tempopb.GetRedactionStatusRequest{})
	require.NoError(t, err)
	require.True(t, resp.Active)
	require.Equal(t, sub.BatchId, resp.BatchId)
	require.Equal(t, tempopb.RedactionMode_REDACTION_MODE_APPLY, resp.Mode)
	require.Equal(t, tempopb.RedactionPhase_REDACTION_PHASE_RUNNING, resp.Phase)
	require.Equal(t, sub.JobsCreated, resp.JobsCreated, "the denominator is the submission's job count")
	require.Equal(t, sub.JobsCreated, resp.JobsRemaining, "nothing has run yet")
	require.Zero(t, resp.JobsRunning)
	require.Zero(t, resp.JobsFailed)
	require.NotZero(t, resp.CreatedAtUnixNano)
}

// TestGetRedactionStatusProgressSurvivesPrune is the point of reporting remaining rather than
// completions: a completed job that has since left the job map -- which is what Prune does to it
// after PruneAge -- must still count as progress, so a long redaction never appears to regress.
func TestGetRedactionStatusProgressSurvivesPrune(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	const tenant = "t-status-progress"
	s := newStatusScheduler(ctx, t, tenant, 3)

	sub := submitAll(ctx, t, s, tenant)

	// Take one job all the way to completion.
	j := s.work.NextPendingJob(tempopb.JobType_JOB_TYPE_REDACTION)
	require.NotNil(t, j)
	j.SetWorkerID("w1")
	require.NoError(t, s.work.AddJob(j))
	s.work.StartJob(j.ID)
	s.work.CompleteJob(j.ID)

	resp, err := s.GetRedactionStatus(user.InjectOrgID(ctx, tenant), &tempopb.GetRedactionStatusRequest{})
	require.NoError(t, err)
	require.Equal(t, sub.JobsCreated-1, resp.JobsRemaining, "one of three is done")
	require.Equal(t, sub.JobsCreated, resp.JobsCreated, "the denominator does not move")

	// Drop the completed job from the map, which is the state Prune leaves behind once the job is
	// older than PruneAge. Removing it directly keeps the premise deterministic -- driving Prune
	// itself would mean either waiting out PruneAge or shrinking it below the minimum the config
	// validates against RescanDelay.
	s.work.RemoveJob(j.ID)
	require.Nil(t, s.work.GetJob(j.ID), "completed job is gone from the job map")

	after, err := s.GetRedactionStatus(user.InjectOrgID(ctx, tenant), &tempopb.GetRedactionStatusRequest{})
	require.NoError(t, err)
	require.Equal(t, resp.JobsRemaining, after.JobsRemaining, "pruning a completed job is not a regression")
	require.Equal(t, resp.JobsCreated, after.JobsCreated)
}
