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
	cfg.LocalWorkPath = tmpDir + "/work"

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
	err = s.loadWorkCache(ctx)
	require.ErrorIs(t, backend.ErrDoesNotExist, err)

	// Flush the empty work cache
	err = s.flushWorkCache(ctx)
	require.NoError(t, err)

	// No error loading the empty work cache
	err = s.loadWorkCache(ctx)
	require.NoError(t, err)

	u := uuid.NewString()
	// Add a job to the work TestCache
	err = s.work.AddJob(&work.Job{
		ID: u,
	})
	require.NoError(t, err)

	// Flush the work cache to the backend
	err = s.flushWorkCacheToBackend(ctx)
	require.NoError(t, err)

	os.Remove(s.cfg.LocalWorkPath + "/" + backend.WorkFileName)

	// Now the work cache should load from the backend and not be empty.
	err = s.loadWorkCache(ctx)
	require.NoError(t, err)

	job := s.work.GetJob(u)
	require.NotNil(t, job, "Job should be found in the work cache")

	jobs := s.work.ListJobs()
	require.Len(t, jobs, 1, "There should be one job in the work cache")
	require.Equal(t, u, jobs[0].ID, "Job ID should match the one added")

	// Create a new scheduler with a different local work path
	cfg.LocalWorkPath = tmpDir + "/work2"
	s, err = New(cfg, store, limits, rr, ww)
	require.NoError(t, err)

	err = s.loadWorkCache(ctx)
	require.NoError(t, err)

	job = s.work.GetJob(u)
	require.NotNil(t, job, "Job should be found in the work cache")

	jobs = s.work.ListJobs()
	require.Len(t, jobs, 1, "There should be one job in the work cache")
	require.Equal(t, u, jobs[0].ID, "Job ID should match the one added")
}
