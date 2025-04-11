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
	cfg.Backoff.MinBackoff = 1 * time.Millisecond
	cfg.Backoff.MaxBackoff = 10 * time.Millisecond
	cfg.Backoff.MaxRetries = 1

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
	for i := 0; i < tenantCount; i++ {
		testTenant := tenant + strconv.Itoa(i)
		writeTenantBlocks(ctx, t, backend.NewWriter(ww), testTenant, 5)
	}

	time.Sleep(100 * time.Millisecond)

	limits, err := overrides.NewOverrides(overrides.Config{Defaults: overrides.Overrides{}}, nil, prometheus.DefaultRegisterer)
	require.NoError(t, err)

	w := work.New(work.Config{})

	p := NewCompactionProvider(
		cfg,
		log.NewNopLogger(),
		store,
		limits,
		w,
	)

	time.Sleep(100 * time.Millisecond)

	jobChan := p.Start(ctx)

	var receivedJobs []*work.Job
	for job := range jobChan {
		receivedJobs = append(receivedJobs, job)
	}

	for _, job := range receivedJobs {
		require.NotNil(t, job)
		require.Equal(t, tempopb.JobType_JOB_TYPE_COMPACTION, job.Type)
		require.NotEqual(t, "", job.Tenant())
		require.NotEqual(t, "", job.ID)
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
	for i := 0; i < count; i++ {
		meta := &backend.BlockMeta{
			BlockID:  backend.NewUUID(),
			TenantID: tenant,
		}

		err = w.WriteBlockMeta(ctx, meta)
		require.NoError(t, err)
	}
}

type ownsEverythingSharder struct{}

func (ownsEverythingSharder) Owns(_ string) bool {
	return true
}
