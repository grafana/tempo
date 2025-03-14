package backendworker

import (
	"context"
	"encoding/binary"
	"flag"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/google/uuid"
	"github.com/grafana/dskit/flagext"
	backendscheduler_client "github.com/grafana/tempo/modules/backendscheduler/client"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/modules/storage"
	"github.com/grafana/tempo/pkg/model"
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
	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"
)

var tenant = "test-tenant"

func TestWorker(t *testing.T) {
	limitCfg := overrides.Config{}
	limitCfg.RegisterFlagsAndApplyDefaults(&flag.FlagSet{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	workerCfg, schedulerClientCfg, overridesSvc, scheduler, store := setupDependencies(ctx, t, limitCfg)

	w, err := New(workerCfg, schedulerClientCfg, store, overridesSvc, prometheus.DefaultRegisterer)
	require.NoError(t, err)
	require.NotNil(t, w)

	w.backendScheduler = scheduler

	err = w.processCompactionJobs(ctx)
	require.Error(t, err, "no jobs found")

	w.backendScheduler = &mockScheduler{
		next:      nextFuncWithJob(store, tenant),
		updateJob: updateJobNoop,
	}

	err = w.processCompactionJobs(ctx)
	require.NoError(t, err)
}

func setupDependencies(ctx context.Context, t *testing.T, limits overrides.Config) (Config, backendscheduler_client.Config, overrides.Service, *mockScheduler, storage.Store) {
	t.Helper()

	var (
		workerConfig Config
		clientConfig backendscheduler_client.Config
	)
	flagext.DefaultValues(&clientConfig)

	f := flag.NewFlagSet("", flag.PanicOnError)
	workerConfig.RegisterFlagsAndApplyDefaults("backendworker", f)

	workerConfig.BackendSchedulerAddr = "localhost:1234"
	workerConfig.Ring.KVStore.Store = "inmemory"
	workerConfig.Ring.KVStore.Mock = nil
	workerConfig.Ring.InstanceInterfaceNames = []string{"eth0", "en0", "lo0", "enp102s0f4u1u3"}

	overrides, err := overrides.NewOverrides(limits, nil, prometheus.DefaultRegisterer)
	require.NoError(t, err)

	scheduler := &mockScheduler{
		next:      nextNoop,
		updateJob: updateJobNoop,
	}

	store, _, _ := newStore(ctx, t, t.TempDir())
	cutTestBlocks(t, store, tenant, 10, 10)

	time.Sleep(200 * time.Millisecond)

	return workerConfig, clientConfig, overrides, scheduler, store
}

var _ tempopb.BackendSchedulerClient = (*mockScheduler)(nil)

type mockScheduler struct {
	grpc_health_v1.HealthClient
	// next mock to be overridden in test scenarios if needed
	next func(ctx context.Context, in *tempopb.NextJobRequest, opts ...grpc.CallOption) (*tempopb.NextJobResponse, error)
	// next mock to be overridden in test scenarios if needed
	updateJob func(ctx context.Context, in *tempopb.UpdateJobStatusRequest, opts ...grpc.CallOption) (*tempopb.UpdateJobStatusResponse, error)
}

func (i *mockScheduler) Next(ctx context.Context, req *tempopb.NextJobRequest, _ ...grpc.CallOption) (*tempopb.NextJobResponse, error) {
	return i.next(ctx, req)
}

func (i *mockScheduler) UpdateJob(ctx context.Context, req *tempopb.UpdateJobStatusRequest, _ ...grpc.CallOption) (*tempopb.UpdateJobStatusResponse, error) {
	return i.updateJob(ctx, req)
}

func nextNoop(_ context.Context, _ *tempopb.NextJobRequest, _ ...grpc.CallOption) (*tempopb.NextJobResponse, error) {
	return &tempopb.NextJobResponse{}, nil
}

func updateJobNoop(_ context.Context, _ *tempopb.UpdateJobStatusRequest, _ ...grpc.CallOption) (*tempopb.UpdateJobStatusResponse, error) {
	return &tempopb.UpdateJobStatusResponse{}, nil
}

func nextFuncWithJob(store storage.Store, tenant string) func(context.Context, *tempopb.NextJobRequest, ...grpc.CallOption) (*tempopb.NextJobResponse, error) {
	var input []string

	metas := store.BlockMetas(tenant)
	for _, meta := range metas {
		input = append(input, meta.BlockID.String())
		if len(input) == 4 {
			break
		}
	}

	if len(input) == 0 {
		return nextNoop
	}

	return func(_ context.Context, _ *tempopb.NextJobRequest, _ ...grpc.CallOption) (*tempopb.NextJobResponse, error) {
		return &tempopb.NextJobResponse{
			JobId: uuid.New().String(),
			Type:  tempopb.JobType_JOB_TYPE_COMPACTION,
			Detail: tempopb.JobDetail{
				Tenant: tenant,
				Compaction: &tempopb.CompactionDetail{
					Input: input,
				},
			},
		}, nil
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

func cutTestBlocks(t testing.TB, w tempodb.Writer, tenantID string, blockCount int, recordCount int) []common.BackendBlock {
	blocks := make([]common.BackendBlock, 0)
	dec := model.MustNewSegmentDecoder(model.CurrentEncoding)

	wal := w.WAL()
	for i := 0; i < blockCount; i++ {
		meta := &backend.BlockMeta{BlockID: backend.NewUUID(), TenantID: tenantID}
		head, err := wal.NewBlock(meta, model.CurrentEncoding)
		require.NoError(t, err)

		for j := 0; j < recordCount; j++ {
			id := makeTraceID(i, j)
			tr := test.MakeTrace(1, id)
			now := uint32(time.Now().Unix())
			writeTraceToWal(t, head, dec, id, tr, now, now)
		}

		b, err := w.CompleteBlock(context.Background(), head)
		require.NoError(t, err)
		blocks = append(blocks, b)
	}

	return blocks
}

func makeTraceID(i int, j int) []byte {
	id := make([]byte, 16)
	binary.LittleEndian.PutUint64(id, uint64(i))
	binary.LittleEndian.PutUint64(id[8:], uint64(j))
	return id
}

func writeTraceToWal(t require.TestingT, b common.WALBlock, dec model.SegmentDecoder, id common.ID, tr *tempopb.Trace, start, end uint32) {
	b1, err := dec.PrepareForWrite(tr, 0, 0)
	require.NoError(t, err)

	b2, err := dec.ToObject([][]byte{b1})
	require.NoError(t, err)

	err = b.Append(id, b2, start, end, true)
	require.NoError(t, err, "unexpected error writing req")
}

type ownsEverythingSharder struct{}

func (ownsEverythingSharder) Owns(_ string) bool {
	return true
}
