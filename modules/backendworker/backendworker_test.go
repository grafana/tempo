package backendworker

import (
	"context"
	"encoding/binary"
	"flag"
	"net"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/google/uuid"
	"github.com/grafana/dskit/flagext"
	"github.com/grafana/dskit/kv"
	"github.com/grafana/dskit/services"
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
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/kfake"
	"github.com/twmb/franz-go/pkg/kgo"
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

	// A fresh registry, not prometheus.DefaultRegisterer: New now
	// unconditionally registers bloomgatewayevents metrics (even when the
	// producer is disabled, as it is here), and DefaultRegisterer is a
	// process-global singleton that a repeat run (e.g. go test -count=2)
	// would hit twice, panicking on duplicate registration.
	w, err := New(workerCfg, schedulerClientCfg, store, overridesSvc, prometheus.NewRegistry())
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
	s.EnablePolling(ctx, &ownsEverythingSharder{}, false)

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

// recordingStore wraps a real storage.Store and records whether/what
// SetCompactionNotifier was called with, so New's enabled/disabled wiring
// can be asserted without driving a full compaction.
type recordingStore struct {
	storage.Store
	notifierSet bool
	notifier    tempodb.CompactionNotifier
}

func (r *recordingStore) SetCompactionNotifier(n tempodb.CompactionNotifier) {
	r.notifierSet = true
	r.notifier = n
	r.Store.SetCompactionNotifier(n)
}

// newBloomGatewayKfakeCluster starts a real in-process Kafka broker seeded
// with topic at numPartitions, mirroring
// pkg/bloomgatewayevents/publisher_test.go's newKfakeCluster (unexported in
// that package, so duplicated here rather than imported).
func newBloomGatewayKfakeCluster(t testing.TB, numPartitions int32, topic string) string {
	t.Helper()
	cluster, err := kfake.NewCluster(kfake.NumBrokers(1), kfake.SeedTopics(numPartitions, topic))
	require.NoError(t, err)
	t.Cleanup(cluster.Close)
	addrs := cluster.ListenAddrs()
	require.Len(t, addrs, 1)
	return addrs[0]
}

// pollBloomGatewayRecords polls reader until at least want records arrive
// or timeout elapses, mirroring
// pkg/bloomgatewayevents/publisher_test.go's pollRecords.
func pollBloomGatewayRecords(t testing.TB, reader *kgo.Client, want int, timeout time.Duration) []*kgo.Record {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var got []*kgo.Record
	for len(got) < want && ctx.Err() == nil {
		fetches := reader.PollFetches(ctx)
		fetches.EachRecord(func(r *kgo.Record) { got = append(got, r) })
	}
	return got
}

// TestBackendWorker_NotifierInstalledWhenEnabled proves New installs a
// notifier on the store when the producer is enabled. The Kafka address is
// never dialed synchronously (bloomgatewayevents.New / ingest.NewWriterClient
// connect lazily, per pkg/bloomgatewayevents's own
// TestPublisher_PublishAdd_UnreachableBroker_DropsSilently), so this needs
// no real broker.
func TestBackendWorker_NotifierInstalledWhenEnabled(t *testing.T) {
	limitCfg := overrides.Config{}
	limitCfg.RegisterFlagsAndApplyDefaults(&flag.FlagSet{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	workerCfg, schedulerClientCfg, overridesSvc, _, store := setupDependencies(ctx, t, limitCfg)

	workerCfg.Producer.RegisterFlagsAndApplyDefaults("producer", flag.NewFlagSet("test", flag.ContinueOnError))
	workerCfg.Producer.Enabled = true

	rs := &recordingStore{Store: store}
	w, err := New(workerCfg, schedulerClientCfg, rs, overridesSvc, prometheus.NewRegistry())
	require.NoError(t, err)
	require.NotNil(t, w)
	t.Cleanup(w.publisher.Close)

	assert.True(t, rs.notifierSet, "an enabled producer must install a notifier on the store")
	assert.NotNil(t, rs.notifier)
}

// TestBackendWorker_NotifierNotInstalledWhenDisabled proves the converse:
// a disabled producer (the zero-value default) must leave the store's
// notifier unset, so compaction pays no bloom-gateway overhead.
func TestBackendWorker_NotifierNotInstalledWhenDisabled(t *testing.T) {
	limitCfg := overrides.Config{}
	limitCfg.RegisterFlagsAndApplyDefaults(&flag.FlagSet{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	workerCfg, schedulerClientCfg, overridesSvc, _, store := setupDependencies(ctx, t, limitCfg)
	// workerCfg.Producer is left at its zero value: Enabled defaults to
	// false, which Validate() always accepts, so no Kafka setup is needed
	// to prove the disabled path installs nothing.

	rs := &recordingStore{Store: store}
	w, err := New(workerCfg, schedulerClientCfg, rs, overridesSvc, prometheus.NewRegistry())
	require.NoError(t, err)
	require.NotNil(t, w)
	t.Cleanup(w.publisher.Close)

	assert.False(t, rs.notifierSet, "a disabled producer must not install a notifier on the store")
}

// TestBackendWorker_CompactionPublishesAdds drives one real compaction job
// end to end -- BackendWorker.processCompactionJob -> tempodb's
// CompactWithConfig -> the installed Notifier -> Publisher -- against a
// real (fake) Kafka broker, proving the full chain rather than just the
// wiring.
func TestBackendWorker_CompactionPublishesAdds(t *testing.T) {
	limitCfg := overrides.Config{}
	limitCfg.RegisterFlagsAndApplyDefaults(&flag.FlagSet{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	workerCfg, schedulerClientCfg, overridesSvc, _, store := setupDependencies(ctx, t, limitCfg)

	const topic = "backendworker-compaction"
	const numPartitions = int32(4)
	addr := newBloomGatewayKfakeCluster(t, numPartitions, topic)

	reader, err := kgo.NewClient(kgo.SeedBrokers(addr), kgo.ConsumeTopics(topic))
	require.NoError(t, err)
	defer reader.Close()

	workerCfg.Producer.RegisterFlagsAndApplyDefaults("producer", flag.NewFlagSet("test", flag.ContinueOnError))
	workerCfg.Producer.Enabled = true
	workerCfg.Producer.Kafka.Address = addr
	workerCfg.Producer.Kafka.Topic = topic
	workerCfg.Producer.Kafka.AutoCreateTopicDefaultPartitions = int(numPartitions)
	// The topic is already seeded at exactly numPartitions; auto-create
	// must stay off or kfake's old-style AlterConfigs wipes the broker
	// config set on seed (same quirk documented in
	// pkg/bloomgatewayevents/publisher_test.go's newTestConfig).
	workerCfg.Producer.Kafka.AutoCreateTopicEnabled = false

	w, err := New(workerCfg, schedulerClientCfg, store, overridesSvc, prometheus.NewRegistry())
	require.NoError(t, err)
	defer w.publisher.Close()

	var gotOutput []string
	w.backendScheduler = &mockScheduler{
		next: nextFuncWithJob(store, tenant),
		updateJob: func(_ context.Context, req *tempopb.UpdateJobStatusRequest, _ ...grpc.CallOption) (*tempopb.UpdateJobStatusResponse, error) {
			if req.Compaction != nil {
				gotOutput = req.Compaction.Output
			}
			return &tempopb.UpdateJobStatusResponse{}, nil
		},
	}

	err = w.processJobs(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, gotOutput, "compaction must have produced at least one output block")

	recs := pollBloomGatewayRecords(t, reader, len(gotOutput), 10*time.Second)
	require.NotEmpty(t, recs)

	var gotBlockIDs []string
	for _, r := range recs {
		event := &tempopb.BloomGatewayEvent{}
		require.NoError(t, event.Unmarshal(r.Value))
		require.Equal(t, tempopb.BloomGatewayEventType_BLOOM_GATEWAY_EVENT_TYPE_ADD_CHUNK, event.Type)
		require.NotNil(t, event.AddChunk)
		assert.Equal(t, tenant, event.AddChunk.TenantId)
		assert.EqualValues(t, 1, event.AddChunk.ChunkCount, "a handful of trace IDs is far under ChunkSize, so each output block must be exactly one chunk")
		gotBlockIDs = append(gotBlockIDs, event.AddChunk.BlockId)
	}
	assert.ElementsMatch(t, gotOutput, gotBlockIDs)
}
