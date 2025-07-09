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
	cfg.Backoff.MinBackoff = 10 * time.Millisecond
	cfg.Backoff.MaxBackoff = 100 * time.Millisecond

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
	cfg.Backoff.MinBackoff = 30 * time.Millisecond // twice the poll cycle
	cfg.Backoff.MaxBackoff = 50 * time.Millisecond

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

	s.EnablePolling(ctx, &ownsEverythingSharder{})

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
