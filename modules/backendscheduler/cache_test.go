package backendscheduler

import (
	"context"
	"flag"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/grafana/tempo/modules/backendscheduler/work"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

func TestCache(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})
	cfg.ProviderConfig.Retention.Interval = 100 * time.Millisecond
	cfg.Work.LocalWorkPath = tmpDir + "/work"

	var (
		ctx, cancel   = context.WithCancel(context.Background())
		store, rr, ww = newStore(ctx, t, tmpDir)
	)

	defer func() {
		cancel()
		store.Shutdown()
	}()

	limits, err := overrides.NewOverrides(overrides.Config{Defaults: overrides.Overrides{}}, nil, prometheus.DefaultRegisterer)
	require.NoError(t, err)

	s, err := New(cfg, store, limits, rr, ww)
	require.NoError(t, err)

	// No file should exist yet
	err = s.work.LoadFromLocal(ctx)
	require.NoError(t, err)

	// Flush the empty work cache
	err = s.work.FlushToLocal(ctx, nil)
	require.NoError(t, err)

	// No error loading the empty work cache
	err = s.work.LoadFromLocal(ctx)
	require.NoError(t, err)

	u := uuid.NewString()
	// Add a job to the work TestCache
	err = s.work.AddJob(ctx, &work.Job{ID: u}, uuid.NewString())
	require.NoError(t, err)

	// Flush the work cache to the backend
	err = s.flushWorkCacheToBackend(ctx)
	require.NoError(t, err)

	os.Remove(s.cfg.Work.LocalWorkPath + "/" + backend.WorkFileName)

	// Test backend fallback
	err = s.loadWorkCacheFromBackend(ctx)
	require.NoError(t, err)

	job := s.work.GetJob(u)
	require.NotNil(t, job, "Job should be found in the work cache")

	jobs := s.work.ListJobs()
	require.Len(t, jobs, 1, "There should be one job in the work cache")
	require.Equal(t, u, jobs[0].ID, "Job ID should match the one added")

	// Create a new scheduler with a different local work path
	cfg.Work.LocalWorkPath = tmpDir + "/work2"
	s, err = New(cfg, store, limits, rr, ww)
	require.NoError(t, err)

	// Test loading from backend when no local cache exists
	err = s.loadWorkCacheFromBackend(ctx)
	require.NoError(t, err)

	job = s.work.GetJob(u)
	require.NotNil(t, job, "Job should be found in the work cache")

	jobs = s.work.ListJobs()
	require.Len(t, jobs, 1, "There should be one job in the work cache")
	require.Equal(t, u, jobs[0].ID, "Job ID should match the one added")
}
