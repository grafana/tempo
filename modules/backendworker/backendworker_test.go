package backendworker

import (
	"context"
	"encoding/binary"
	"flag"
	"fmt"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/gogo/status"
	"github.com/google/uuid"
	"github.com/grafana/dskit/backoff"
	"github.com/grafana/dskit/flagext"
	"github.com/grafana/dskit/kv"
	"github.com/grafana/dskit/services"
	backendscheduler_client "github.com/grafana/tempo/v3/modules/backendscheduler/client"
	"github.com/grafana/tempo/v3/modules/overrides"
	"github.com/grafana/tempo/v3/modules/storage"
	"github.com/grafana/tempo/v3/pkg/model"
	"github.com/grafana/tempo/v3/pkg/tempopb"
	util_log "github.com/grafana/tempo/v3/pkg/util/log"
	"github.com/grafana/tempo/v3/pkg/util/test"
	"github.com/grafana/tempo/v3/tempodb"
	"github.com/grafana/tempo/v3/tempodb/backend"
	"github.com/grafana/tempo/v3/tempodb/backend/local"
	"github.com/grafana/tempo/v3/tempodb/encoding"
	"github.com/grafana/tempo/v3/tempodb/encoding/common"
	"github.com/grafana/tempo/v3/tempodb/wal"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
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

	err = w.processJobs(ctx)
	require.Error(t, err, "no jobs found")

	w.backendScheduler = &mockScheduler{
		next:      nextFuncWithJob(store, tenant),
		updateJob: updateJobNoop,
	}

	err = w.processJobs(ctx)
	require.NoError(t, err)

	err = services.StopAndAwaitTerminated(ctx, w)
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
	ifaces, err := net.Interfaces()
	require.NoError(t, err)
	netWorkInteraces := make([]string, len(ifaces))
	for i, iface := range ifaces {
		netWorkInteraces[i] = iface.Name
	}
	workerConfig.Ring.InstanceInterfaceNames = netWorkInteraces

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

func (i *mockScheduler) SubmitRedaction(_ context.Context, _ *tempopb.SubmitRedactionRequest, _ ...grpc.CallOption) (*tempopb.SubmitRedactionResponse, error) {
	return &tempopb.SubmitRedactionResponse{}, nil
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

	// The store service is never started, so only cancel + Shutdown joins the poller.
	ctx, cancel := context.WithCancel(ctx)
	s.EnablePolling(ctx, &ownsEverythingSharder{})

	t.Cleanup(func() {
		cancel()
		s.Shutdown()
	})
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

// TestProcessRedactionJobMissingBlockObservable verifies that a redaction job
// whose target block is absent from the live blocklist is counted (and logged),
// rather than silently completing as a no-op. The completion stays non-fatal —
// the scheduler's coverage logic is responsible for re-targeting moved blocks —
// but the event must be observable.
func TestProcessRedactionJobMissingBlockObservable(t *testing.T) {
	limitCfg := overrides.Config{}
	limitCfg.RegisterFlagsAndApplyDefaults(&flag.FlagSet{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	workerCfg, schedulerClientCfg, overridesSvc, _, store := setupDependencies(ctx, t, limitCfg)

	w, err := New(workerCfg, schedulerClientCfg, store, overridesSvc, prometheus.NewRegistry())
	require.NoError(t, err)
	w.backendScheduler = &mockScheduler{updateJob: updateJobNoop}

	before := testutil.ToFloat64(metricRedactionBlockMissing.WithLabelValues(tenant))
	err = w.processRedactionJob(ctx, &tempopb.NextJobResponse{
		JobId: "job-missing-block",
		Detail: tempopb.JobDetail{
			Tenant:    tenant,
			Redaction: &tempopb.RedactionDetail{BlockId: uuid.New().String()},
		},
	})
	require.NoError(t, err, "a missing block must complete as a non-fatal no-op")
	after := testutil.ToFloat64(metricRedactionBlockMissing.WithLabelValues(tenant))
	require.Equal(t, before+1, after, "a missing redaction block must be counted, not silently dropped")
}

func TestIsSharded(t *testing.T) {
	tests := []struct {
		name     string
		store    string
		expected bool
	}{
		{
			name:     "empty store is not sharded",
			store:    "",
			expected: false,
		},
		{
			name:     "inmemory store is not sharded",
			store:    "inmemory",
			expected: false,
		},
		{
			name:     "memberlist store is sharded",
			store:    "memberlist",
			expected: true,
		},
		{
			name:     "consul store is sharded",
			store:    "consul",
			expected: true,
		},
		{
			name:     "etcd store is sharded",
			store:    "etcd",
			expected: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := &BackendWorker{
				cfg: Config{
					Ring: RingConfig{
						KVStore: kv.Config{
							Store: tc.store,
						},
					},
				},
			}
			assert.Equal(t, tc.expected, w.isSharded())
		})
	}
}

// captureLogger records what was logged, so a test can assert on the LEVEL a line was
// emitted at rather than only on the behaviour around it.
type captureLogger struct {
	mu    sync.Mutex
	lines []string
}

func (c *captureLogger) Log(kv ...interface{}) error {
	var sb strings.Builder
	for i := 0; i+1 < len(kv); i += 2 {
		fmt.Fprintf(&sb, "%v=%v ", kv[i], kv[i+1])
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	c.lines = append(c.lines, sb.String())

	return nil
}

func (c *captureLogger) captured() string {
	c.mu.Lock()
	defer c.mu.Unlock()

	return strings.Join(c.lines, "\n")
}

// captureSchedulerLogs points the package logger at a captureLogger for one test.
func captureSchedulerLogs(t *testing.T) *captureLogger {
	t.Helper()

	captured := &captureLogger{}
	previous := util_log.Logger
	util_log.Logger = captured
	t.Cleanup(func() { util_log.Logger = previous })

	return captured
}

func backoffWorker() *BackendWorker {
	// Only cfg.Backoff is read by callSchedulerWithBackoff, and MaxRetries bounds the loop
	// so the call returns instead of polling for the life of the test.
	return &BackendWorker{cfg: Config{Backoff: backoff.Config{
		MinBackoff: time.Millisecond,
		MaxBackoff: 2 * time.Millisecond,
		MaxRetries: 2,
	}}}
}

// An idle scheduler answers Next with codes.NotFound once its long-poll times out, which
// processJobs passes through to the backoff loop so polling continues. That is the expected
// state of a cluster with nothing to compact, so it must not be reported as a failure: on a
// 15s poll it otherwise emits an error line and a call retry on every poll, forever.
func TestCallSchedulerWithBackoffDoesNotReportAnEmptyQueueAsAFailure(t *testing.T) {
	captured := captureSchedulerLogs(t)
	before := testutil.ToFloat64(metricWorkerCallRetries.WithLabelValues())

	err := backoffWorker().callSchedulerWithBackoff(context.Background(), func(context.Context) error {
		return status.Error(codes.NotFound, "no jobs found")
	})

	// Still retried and still gave up once the backoff was exhausted: only the reporting
	// changes, not the polling.
	require.Error(t, err)
	require.Equal(t, before, testutil.ToFloat64(metricWorkerCallRetries.WithLabelValues()),
		"an empty job queue must not count as a call retry")
	require.NotContains(t, captured.captured(), "level=error",
		"an empty job queue must not be logged at error level")
	require.Contains(t, captured.captured(), "level=debug",
		"an empty job queue should still be observable at debug level")
}

// The negative control for the test above: a scheduler that is genuinely unreachable must
// still be reported, or demoting the idle case would have hidden a real failure.
func TestCallSchedulerWithBackoffStillReportsRealFailures(t *testing.T) {
	captured := captureSchedulerLogs(t)
	before := testutil.ToFloat64(metricWorkerCallRetries.WithLabelValues())

	err := backoffWorker().callSchedulerWithBackoff(context.Background(), func(context.Context) error {
		return status.Error(codes.Unavailable, "connection refused")
	})

	require.Error(t, err)
	require.Greater(t, testutil.ToFloat64(metricWorkerCallRetries.WithLabelValues()), before,
		"a scheduler that cannot be reached must still count as a call retry")
	require.Contains(t, captured.captured(), "level=error",
		"a scheduler that cannot be reached must still be logged at error level")
}
