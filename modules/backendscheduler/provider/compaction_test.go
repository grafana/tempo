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

	b := p.prepareNextTenant(ctx, false)
	require.False(t, b, "no tenant should be found")
	require.Nil(t, p.curTenant, "a tenant should not be set")
	require.Nil(t, p.curSelector, "a block selector should not be set")

	writeTenantBlocks(ctx, t, backend.NewWriter(ww), tenant, 1)
	time.Sleep(150 * time.Millisecond)

	b = p.prepareNextTenant(ctx, false)
	require.False(t, b, "no tenant with a single block should be found")
	require.Nil(t, p.curTenant, "a tenant should not be set")
	require.Nil(t, p.curSelector, "a block selector should not be set")

	writeTenantBlocks(ctx, t, backend.NewWriter(ww), tenant, 1)
	time.Sleep(150 * time.Millisecond)

	b = p.prepareNextTenant(ctx, false)
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

// TestCompactionProvider_SkipsAllCompactionDuringRedaction verifies that newBlockSelector
// returns an empty selector when any redaction jobs exist for the tenant. Once a
// SubmitRedaction call creates pending redaction jobs, all new compaction is gated until
// the redaction batch completes. Busy-block filtering (BusyBlocksForTenant) handles
// exclusion of specific blocks across non-redaction workloads.
func TestCompactionProvider_SkipsAllCompactionDuringRedaction(t *testing.T) {
	const testTenant = "test-tenant"
	cfg := CompactionConfig{}
	cfg.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})
	cfg.MaxJobsPerTenant = 1000
	cfg.MeasureInterval = 100 * time.Millisecond
	cfg.MinCycleInterval = 100 * time.Millisecond

	tmpDir := t.TempDir()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	store, _, ww := newStore(ctx, t, tmpDir)
	defer store.Shutdown()

	writeTenantBlocks(ctx, t, backend.NewWriter(ww), testTenant, 5)
	time.Sleep(150 * time.Millisecond)

	blockMetas := store.BlockMetas(testTenant)
	require.GreaterOrEqual(t, len(blockMetas), 2, "need at least 2 blocks to mark pending")

	// Mark two blocks as having pending redaction jobs.
	pendingBlock1 := blockMetas[0].BlockID.String()
	pendingBlock2 := blockMetas[1].BlockID.String()

	w := work.New(work.Config{})
	pendingJobs := []*work.Job{
		{
			ID:   "redact-1",
			Type: tempopb.JobType_JOB_TYPE_REDACTION,
			JobDetail: tempopb.JobDetail{
				Tenant: testTenant,
				Redaction: &tempopb.RedactionDetail{
					BlockId: pendingBlock1,
				},
			},
		},
		{
			ID:   "redact-2",
			Type: tempopb.JobType_JOB_TYPE_REDACTION,
			JobDetail: tempopb.JobDetail{
				Tenant: testTenant,
				Redaction: &tempopb.RedactionDetail{
					BlockId: pendingBlock2,
				},
			},
		},
	}
	require.NoError(t, w.AddPendingJobs(pendingJobs))
	// In production, pending redaction jobs are always accompanied by a batch.
	// TenantPending (the compaction guard) checks for an active batch, so the test
	// must add one to match the production invariant.
	require.NoError(t, w.AddBatch(&tempopb.RedactionBatch{
		BatchId:  "batch-1",
		TenantId: testTenant,
	}))

	limits, err := overrides.NewOverrides(overrides.Config{Defaults: overrides.Overrides{}}, nil, prometheus.DefaultRegisterer)
	require.NoError(t, err)

	p := NewCompactionProvider(
		cfg,
		test.NewTestingLogger(t),
		store,
		limits,
		w,
	)

	// When a redaction batch is active the gate fires: all compaction is blocked,
	// not just the two specific blocks.
	selector, blocklistLen := p.newBlockSelector(testTenant)
	require.NotNil(t, selector)
	require.Equal(t, 0, blocklistLen, "all compaction should be blocked while redaction jobs exist")
	require.Empty(t, collectAllMetas(selector), "no blocks should be offered for compaction during redaction")
}

// TestCompactionProvider_MeasureTenantsIgnoresTenantPending verifies that
// newBlockSelectorForMeasurement returns the real block count even when
// TenantPending is true. This ensures the outstanding-blocks metric (and
// therefore autoscaling) is not disrupted by an active redaction batch.
func TestCompactionProvider_MeasureTenantsIgnoresTenantPending(t *testing.T) {
	const testTenant = "test-tenant"
	cfg := CompactionConfig{}
	cfg.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})

	tmpDir := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	store, _, ww := newStore(ctx, t, tmpDir)
	defer store.Shutdown()

	writeTenantBlocks(ctx, t, backend.NewWriter(ww), testTenant, 5)
	time.Sleep(150 * time.Millisecond)

	w := work.New(work.Config{})
	// Activate a redaction batch so TenantPending returns true.
	require.NoError(t, w.AddBatch(&tempopb.RedactionBatch{
		BatchId:  "batch-1",
		TenantId: testTenant,
	}))
	require.True(t, w.TenantPending(testTenant))

	limits, err := overrides.NewOverrides(overrides.Config{Defaults: overrides.Overrides{}}, nil, prometheus.DefaultRegisterer)
	require.NoError(t, err)

	p := NewCompactionProvider(cfg, test.NewTestingLogger(t), store, limits, w)

	// newBlockSelector must return 0 blocks (compaction gated during redaction).
	_, blocklistLen := p.newBlockSelector(testTenant)
	require.Equal(t, 0, blocklistLen, "newBlockSelector should return 0 blocks while TenantPending")

	// newBlockSelectorForMeasurement must return the actual block count so that
	// the outstanding-blocks metric is not suppressed during redaction.
	_, measureLen := p.newBlockSelectorForMeasurement(testTenant)
	require.Greater(t, measureLen, 0, "newBlockSelectorForMeasurement should return blocks even while TenantPending")
}

func TestCompactionProvider_InFlightJobsPreventDuplicates(t *testing.T) {
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
	}

	// All received jobs should have their blocks reported as busy (registered in-flight
	// before entering the channel pipeline).
	for _, job := range receivedJobs {
		for _, blockID := range job.GetCompactionInput() {
			require.True(t, w.IsBlockBusy(tenant, blockID), "block should be busy while job is in-flight")
		}
	}

	// Add the first job to the work queue; its blocks move from in-flight to active.
	firstJob := receivedJobs[0]
	err = w.AddJob(firstJob)
	require.NoError(t, err, "should be able to add first job to work queue")

	// After AddJob, first job's blocks are in the active map — still reported as busy.
	for _, blockID := range firstJob.GetCompactionInput() {
		require.True(t, w.IsBlockBusy(tenant, blockID), "block should be busy while job is active")
	}

	// Get a new block selector for the tenant and verify the first job's blocks are excluded.
	twbs, _ := p.newBlockSelector(tenant)
	require.NotNil(t, twbs)

	metas := collectAllMetas(twbs)
	for _, blockID := range firstJob.GetCompactionInput() {
		u := backend.MustParse(blockID)
		meta, found := foundMetaInMetas(metas, u)
		require.False(t, found, "Block %s from job %s should not be present in the block selector after being added to work queue", blockID, firstJob.ID)
		require.Nil(t, meta, "Block %s from job %s should not be present in the block selector after being added to work queue", blockID, firstJob.ID)
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
				BloomFP:             0.01,
				BloomShardSizeBytes: 100_000,
				Version:             encoding.LatestEncoding().Version(),
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
