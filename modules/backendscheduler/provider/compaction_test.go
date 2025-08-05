package provider

import (
	"context"
	"flag"
	"strconv"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/grafana/tempo/modules/backendscheduler/work"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/modules/storage"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/grafana/tempo/tempodb"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/local"
	"github.com/grafana/tempo/tempodb/blockselector"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/grafana/tempo/tempodb/wal"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

var tenant = "test-tenant"

func TestCompactionProvider(t *testing.T) {
	cfg := CompactionConfig{}
	cfg.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})
	cfg.MaxJobsPerTenant = 2
	cfg.MeasureInterval = 100 * time.Millisecond
	cfg.MinCycleInterval = 100 * time.Millisecond

	tmpDir := t.TempDir()

	var (
		ctx, cancel  = context.WithTimeout(context.Background(), 5*time.Second)
		store, _, ww = newStore(ctx, t, tmpDir)
	)

	defer func() {
		cancel()
		store.Shutdown()
	}()

	// Push some data to a few tenants
	tenantCount := 3
	for i := range tenantCount {
		testTenant := tenant + strconv.Itoa(i)
		writeTenantBlocks(ctx, t, backend.NewWriter(ww), testTenant, 5)
	}

	time.Sleep(100 * time.Millisecond)

	limits, err := overrides.NewOverrides(overrides.Config{Defaults: overrides.Overrides{}}, nil, prometheus.DefaultRegisterer)
	require.NoError(t, err)

	w := work.New(work.Config{})

	p := NewCompactionProvider(
		cfg,
		test.NewTestingLogger(t),
		store,
		limits,
		w,
	)

	jobChan := p.Start(ctx)

	var receivedJobs []*work.Job
	for job := range jobChan {
		receivedJobs = append(receivedJobs, job)
		err = w.AddJob(job)
		require.NoError(t, err)
	}

	for _, job := range receivedJobs {
		require.NotNil(t, job)
		require.Equal(t, tempopb.JobType_JOB_TYPE_COMPACTION, job.Type)
		require.Greater(t, len(job.GetCompactionInput()), 0)
		require.NotEqual(t, "", job.Tenant())
		require.NotEqual(t, "", job.ID)

		// Check that the newBlockSelector does not include the input blocks
		for i := range tenantCount {
			testTenant := tenant + strconv.Itoa(i)
			twbs, _ := p.newBlockSelector(testTenant)
			require.NotNil(t, twbs)

			metas := collectAllMetas(twbs)

			// All blocks which have been received were addded to the work queue.
			// New instances of the block selector should yield zero blocks.
			require.Len(t, metas, 0)
		}
	}
}

func TestCompactionProvider_EmptyStart(t *testing.T) {
	cfg := CompactionConfig{}
	cfg.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})
	cfg.MaxJobsPerTenant = 1
	cfg.MeasureInterval = 100 * time.Millisecond
	cfg.MinCycleInterval = 100 * time.Millisecond

	tmpDir := t.TempDir()

	var (
		ctx, cancel  = context.WithTimeout(context.Background(), 5*time.Second)
		store, _, ww = newStore(ctx, t, tmpDir)
	)

	defer func() {
		cancel()
		store.Shutdown()
	}()

	limits, err := overrides.NewOverrides(overrides.Config{Defaults: overrides.Overrides{}}, nil, prometheus.DefaultRegisterer)
	require.NoError(t, err)

	w := work.New(work.Config{})

	p := NewCompactionProvider(
		cfg,
		test.NewTestingLogger(t),
		store,
		limits,
		w,
	)

	b := p.prepareNextTenant(ctx)
	require.False(t, b, "no tenant should be found")
	require.Nil(t, p.curTenant, "a tenant should not be set")
	require.Nil(t, p.curSelector, "a block selector should not be set")

	writeTenantBlocks(ctx, t, backend.NewWriter(ww), tenant, 1)
	time.Sleep(150 * time.Millisecond)

	b = p.prepareNextTenant(ctx)
	require.False(t, b, "no tenant with a single block should be found")
	require.Nil(t, p.curTenant, "a tenant should not be set")
	require.Nil(t, p.curSelector, "a block selector should not be set")

	writeTenantBlocks(ctx, t, backend.NewWriter(ww), tenant, 1)
	time.Sleep(150 * time.Millisecond)

	b = p.prepareNextTenant(ctx)
	require.True(t, b, "tenant with two blocks should be found")
	require.NotNil(t, p.curTenant, "a tenant should be set")
	require.NotNil(t, p.curSelector, "a block selector should be set")

	jobChan := p.Start(ctx)
	require.NotNil(t, jobChan, "job channel should not be nil")

	blocksSeen := make(map[string]struct{})
	var metas []*backend.BlockMeta

	for job := range jobChan {
		require.NotNil(t, job, "job should not be nil")
		require.Equal(t, tempopb.JobType_JOB_TYPE_COMPACTION, job.Type, "job type should be compaction")
		require.Equal(t, tenant, job.Tenant(), "job tenant should match the current tenant")
		require.Greater(t, len(job.GetCompactionInput()), 0, "compaction input should not be empty")

		metas = store.BlockMetas(tenant)

		// Add each job to the work queue, so we don't process the blocks within.
		err = w.AddJob(job)
		require.NoError(t, err, "should be able to add job to work queue")

		for _, blockID := range job.GetCompactionInput() {
			_, seen := blocksSeen[blockID]
			require.False(t, seen, "block ID %s should not have been seen before", blockID)
			blocksSeen[blockID] = struct{}{}

			u := backend.MustParse(blockID)
			if _, ok := foundMetaInMetas(metas, u); ok {
				// metas = append(metas, meta)
				err = store.MarkBlockCompacted(tenant, u)
				require.NoError(t, err, "should be able to mark block %s compacted for tenant %s", blockID, tenant)
			}
		}
		require.Greater(t, len(metas), 0, "there should be some block metas to compact")

	}
}

func TestCompactionProvider_RecentJobsCache(t *testing.T) {
	// Create a test provider
	provider := &CompactionProvider{
		logger:          log.NewNopLogger(),
		sched:           &mockScheduler{},
		outstandingJobs: make(map[string][]backend.UUID),
	}

	var (
		block1 = backend.NewUUID()
		block2 = backend.NewUUID()
		block3 = backend.NewUUID()
		block4 = backend.NewUUID()
	)

	// Cache starts empty
	require.Len(t, provider.outstandingJobs, 0, "Cache should start empty")

	job1 := &work.Job{
		ID:   "job1",
		Type: tempopb.JobType_JOB_TYPE_COMPACTION,
		JobDetail: tempopb.JobDetail{
			Compaction: &tempopb.CompactionDetail{
				Input: []string{block1.String(), block2.String()},
			},
		},
	}

	provider.addToRecentJobs(job1)
	require.Len(t, provider.outstandingJobs, 1, "Cache should contain one job")
	require.Contains(t, provider.outstandingJobs, "job1", "Cache should contain job1")
	require.Equal(t, []backend.UUID{block1, block2}, provider.outstandingJobs["job1"], "Cache should contain correct blocks")

	job2 := &work.Job{
		ID:   "job2",
		Type: tempopb.JobType_JOB_TYPE_COMPACTION,
		JobDetail: tempopb.JobDetail{
			Compaction: &tempopb.CompactionDetail{
				Input: []string{block3.String(), block4.String()},
			},
		},
	}

	provider.addToRecentJobs(job2)
	require.Len(t, provider.outstandingJobs, 2, "Cache should contain two jobs")
	require.Equal(t, []backend.UUID{block3, block4}, provider.outstandingJobs["job2"], "Cache should contain correct blocks for job2")

	// Non-compaction job should not be added to cache
	retentionJob := &work.Job{
		ID:   "retention1",
		Type: tempopb.JobType_JOB_TYPE_RETENTION,
	}

	provider.addToRecentJobs(retentionJob)
	require.Len(t, provider.outstandingJobs, 2, "Non-compaction jobs should not be added to cache")

	// Empty compaction input should not be added
	emptyJob := &work.Job{
		ID:   "empty-job",
		Type: tempopb.JobType_JOB_TYPE_COMPACTION,
		JobDetail: tempopb.JobDetail{
			Compaction: &tempopb.CompactionDetail{
				Input: []string{}, // Empty input
			},
		},
	}

	provider.addToRecentJobs(emptyJob)
	require.Len(t, provider.outstandingJobs, 2, "Jobs with empty input should not be added to cache")
}

func TestCompactionProvider_RecentJobsCachePreventseDuplicatesAndCleansUp(t *testing.T) {
	const tenant = "test-tenant"
	cfg := CompactionConfig{}
	cfg.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})
	cfg.MaxJobsPerTenant = 1000 // Set high enough so we don't hit the limit
	cfg.MeasureInterval = 100 * time.Millisecond
	cfg.MinCycleInterval = 100 * time.Millisecond

	tmpDir := t.TempDir()

	var (
		ctx, cancel  = context.WithTimeout(context.Background(), 5*time.Second)
		store, _, ww = newStore(ctx, t, tmpDir)
	)

	defer func() {
		cancel()
		store.Shutdown()
	}()

	// Push some data to one tenant - enough blocks to create multiple compaction jobs
	writeTenantBlocks(ctx, t, backend.NewWriter(ww), tenant, 10)

	time.Sleep(100 * time.Millisecond)

	limits, err := overrides.NewOverrides(overrides.Config{Defaults: overrides.Overrides{}}, nil, prometheus.DefaultRegisterer)
	require.NoError(t, err)

	w := work.New(work.Config{})

	p := NewCompactionProvider(
		cfg,
		test.NewTestingLogger(t),
		store,
		limits,
		w,
	)

	jobChan := p.Start(ctx)

	var receivedJobs []*work.Job
	blockIDs := make(map[string]int)

	for job := range jobChan {
		receivedJobs = append(receivedJobs, job)
		// Collect the jobs but do not add them to the work queue yet

		for _, blockID := range job.GetCompactionInput() {
			// Count how many times each block ID appears across all jobs
			blockIDs[blockID]++
			require.Equal(t, 1, blockIDs[blockID], "Block ID %s should only appear once across all jobs", blockID)
		}
	}

	for _, job := range receivedJobs {
		require.NotNil(t, job)
		require.Equal(t, tempopb.JobType_JOB_TYPE_COMPACTION, job.Type)
		require.Greater(t, len(job.GetCompactionInput()), 0)
		require.Equal(t, tenant, job.Tenant(), "Job tenant should match the test tenant")
		require.NotEqual(t, "", job.ID)

		// // Check that the newBlockSelector does not include the input blocks
		// twbs, _ := p.newBlockSelector(tenant)
		// require.NotNil(t, twbs)
		//
		// metas := collectAllMetas(twbs)
		//
		// // All blocks which have been received were addded to the work queue.
		// // New instances of the block selector should yield zero blocks.
		// require.Len(t, metas, 0)
	}

	// Verify that all received jobs are present in the recent jobs cache
	require.Len(t, p.outstandingJobs, len(receivedJobs), "Recent jobs cache should contain all received jobs")
	for _, job := range receivedJobs {
		require.Contains(t, p.outstandingJobs, job.ID, "Recent jobs cache should contain job %s", job.ID)

		require.Equal(t, len(job.GetCompactionInput()), len(p.outstandingJobs[job.ID]), "Recent jobs cache should contain correct number of blocks for job %s", job.ID)
		for _, blockID := range job.GetCompactionInput() {
			u := backend.MustParse(blockID)
			require.Equal(t, u.String(), blockID, "Block ID %s should match the UUID format", blockID)
		}
	}

	// Now add a job to the work queue and verify it is removed from the recent jobs cache
	firstJob := receivedJobs[0]
	err = w.AddJob(firstJob)
	require.NoError(t, err, "should be able to add first job to work queue")

	// Get a new block selector for the tenant
	twbs, _ := p.newBlockSelector(tenant)
	require.NotNil(t, twbs)

	// Verify the recent jobs cache was cleaned up
	require.NotContains(t, p.outstandingJobs, firstJob.ID, "Job should be removed from recent jobs cache after being added to work queue")

	metas := collectAllMetas(twbs)
	// Verify that the blocks in the first job are not present in the metas
	for _, blockID := range firstJob.GetCompactionInput() {
		u := backend.MustParse(blockID)
		meta, found := foundMetaInMetas(metas, u)
		require.False(t, found, "Block %s from job %s should not be present in the block selector after being added to work queue", blockID, firstJob.ID)
		require.Nil(t, meta, "Block %s from job %s should not be present in the block selector after being added to work queue", blockID, firstJob.ID)
	}

	// Verify that the remaining jobs are still in the cache
	for _, job := range receivedJobs[1:] {
		require.Contains(t, p.outstandingJobs, job.ID, "Recent jobs cache should still contain job %s", job.ID)
		require.Equal(t, len(job.GetCompactionInput()), len(p.outstandingJobs[job.ID]), "Recent jobs cache should contain correct number of blocks for job %s", job.ID)
		for _, blockID := range job.GetCompactionInput() {
			u := backend.MustParse(blockID)
			require.Equal(t, u.String(), blockID, "Block ID %s should match the UUID format", blockID)
		}

	}
}

func newStore(ctx context.Context, t testing.TB, tmpDir string) (storage.Store, backend.RawReader, backend.RawWriter) {
	rr, ww, _, err := local.New(&local.Config{
		Path: tmpDir + "/traces",
	})
	require.NoError(t, err)

	return newStoreWithLogger(ctx, t, test.NewTestingLogger(t), tmpDir), rr, ww
}

func newStoreWithLogger(ctx context.Context, t testing.TB, log log.Logger, tmpDir string) storage.Store {
	s, err := storage.NewStore(storage.Config{
		Trace: tempodb.Config{
			Backend: backend.Local,
			Local: &local.Config{
				Path: tmpDir + "/traces",
			},
			Block: &common.BlockConfig{
				IndexDownsampleBytes: 2,
				BloomFP:              0.01,
				BloomShardSizeBytes:  100_000,
				Version:              encoding.LatestEncoding().Version(),
				Encoding:             backend.EncLZ4_1M,
				IndexPageSizeBytes:   1000,
			},
			WAL: &wal.Config{
				Filepath: tmpDir + "/wal",
			},
			BlocklistPoll: 100 * time.Millisecond,
		},
	}, nil, log)
	require.NoError(t, err)

	s.EnablePolling(ctx, &ownsEverythingSharder{}, false)

	return s
}

func writeTenantBlocks(ctx context.Context, t *testing.T, w backend.Writer, tenant string, count int) {
	var err error
	for range count {
		meta := &backend.BlockMeta{
			BlockID:  backend.NewUUID(),
			TenantID: tenant,
			Version:  encoding.LatestEncoding().Version(),
		}

		err = w.WriteBlockMeta(ctx, meta)
		require.NoError(t, err)
	}
}

func foundMetaInMetas(metas []*backend.BlockMeta, u backend.UUID) (*backend.BlockMeta, bool) {
	for _, m := range metas {
		if m.BlockID == u {
			return m, true
		}
	}
	return nil, false
}

type ownsEverythingSharder struct{}

func (ownsEverythingSharder) Owns(_ string) bool {
	return true
}

func collectAllMetas(bs blockselector.CompactionBlockSelector) []*backend.BlockMeta {
	metas, _ := bs.BlocksToCompact()

	for newMetas, _ := bs.BlocksToCompact(); len(newMetas) > 0; {
		metas = append(metas, newMetas...)
	}

	return metas
}

// mockScheduler implements the Scheduler interface for testing
type mockScheduler struct {
	jobs []*work.Job
}

func (m *mockScheduler) ListJobs() []*work.Job {
	return m.jobs
}

// func (m *mockScheduler) addJob(job *work.Job) {
// 	m.jobs = append(m.jobs, job)
// }
