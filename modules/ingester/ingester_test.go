package ingester

import (
	"context"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/gogo/protobuf/proto"
	"github.com/grafana/dskit/flagext"
	"github.com/grafana/dskit/kv/consul"
	"github.com/grafana/dskit/ring"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"github.com/weaveworks/common/user"

	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/modules/storage"
	"github.com/grafana/tempo/pkg/model/trace"
	"github.com/grafana/tempo/pkg/tempofb"
	"github.com/grafana/tempo/pkg/tempopb"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/grafana/tempo/tempodb"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/local"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/wal"
)

func TestPushQuery(t *testing.T) {
	tmpDir, err := os.MkdirTemp("/tmp", "")
	require.NoError(t, err, "unexpected error getting tempdir")
	defer os.RemoveAll(tmpDir)

	ctx := user.InjectOrgID(context.Background(), "test")
	ingester, traces, traceIDs := defaultIngester(t, tmpDir)

	for pos, traceID := range traceIDs {
		foundTrace, err := ingester.FindTraceByID(ctx, &tempopb.TraceByIDRequest{
			TraceID: traceID,
		})
		require.NoError(t, err, "unexpected error querying")
		require.Equal(t, foundTrace.Trace, traces[pos])
	}

	// force cut all traces
	for _, instance := range ingester.instances {
		err = instance.CutCompleteTraces(0, true)
		require.NoError(t, err, "unexpected error cutting traces")
	}

	// should be able to find them now
	for i, traceID := range traceIDs {
		foundTrace, err := ingester.FindTraceByID(ctx, &tempopb.TraceByIDRequest{
			TraceID: traceID,
		})
		require.NoError(t, err, "unexpected error querying")
		equal := proto.Equal(traces[i], foundTrace.Trace)
		require.True(t, equal)
	}
}

func TestFullTraceReturned(t *testing.T) {
	tmpDir, err := os.MkdirTemp("/tmp", "")
	require.NoError(t, err, "unexpected error getting tempdir")
	defer os.RemoveAll(tmpDir)

	ctx := user.InjectOrgID(context.Background(), "test")
	ingester, _, _ := defaultIngester(t, tmpDir)

	traceID := make([]byte, 16)
	_, err = rand.Read(traceID)
	require.NoError(t, err)
	testTrace := test.MakeTrace(2, traceID) // 2 batches
	trace.SortTrace(testTrace)

	// push the first batch
	pushBatch(t, ingester, testTrace.Batches[0], traceID)

	// force cut all traces
	for _, instance := range ingester.instances {
		err = instance.CutCompleteTraces(0, true)
		require.NoError(t, err, "unexpected error cutting traces")
	}

	// push the 2nd batch
	pushBatch(t, ingester, testTrace.Batches[1], traceID)

	// make sure the trace comes back whole
	foundTrace, err := ingester.FindTraceByID(ctx, &tempopb.TraceByIDRequest{
		TraceID: traceID,
	})
	require.NoError(t, err, "unexpected error querying")
	require.True(t, proto.Equal(testTrace, foundTrace.Trace))

	// force cut all traces
	for _, instance := range ingester.instances {
		err = instance.CutCompleteTraces(0, true)
		require.NoError(t, err, "unexpected error cutting traces")
	}

	// make sure the trace comes back whole
	foundTrace, err = ingester.FindTraceByID(ctx, &tempopb.TraceByIDRequest{
		TraceID: traceID,
	})
	require.NoError(t, err, "unexpected error querying")
	require.True(t, proto.Equal(testTrace, foundTrace.Trace))
}

func TestWal(t *testing.T) {
	tmpDir, err := os.MkdirTemp("/tmp", "")
	require.NoError(t, err, "unexpected error getting tempdir")
	defer os.RemoveAll(tmpDir)

	ctx := user.InjectOrgID(context.Background(), "test")
	ingester, traces, traceIDs := defaultIngester(t, tmpDir)

	for pos, traceID := range traceIDs {
		foundTrace, err := ingester.FindTraceByID(ctx, &tempopb.TraceByIDRequest{
			TraceID: traceID,
		})
		require.NoError(t, err, "unexpected error querying")
		require.Equal(t, foundTrace.Trace, traces[pos])
	}

	// force cut all traces
	for _, instance := range ingester.instances {
		err := instance.CutCompleteTraces(0, true)
		require.NoError(t, err, "unexpected error cutting traces")
	}

	// create new ingester.  this should replay wal!
	ingester, _, _ = defaultIngester(t, tmpDir)

	// should be able to find old traces that were replayed
	for i, traceID := range traceIDs {
		foundTrace, err := ingester.FindTraceByID(ctx, &tempopb.TraceByIDRequest{
			TraceID: traceID,
		})
		require.NoError(t, err, "unexpected error querying")
		equal := proto.Equal(traces[i], foundTrace.Trace)
		require.True(t, equal)
	}

	// a block that has been replayed should have a flush queue entry to complete it
	// wait for the flush queues to be empty and then confirm there is a complete block
	for !ingester.flushQueues.IsEmpty() {
		time.Sleep(100 * time.Millisecond)
	}

	require.Len(t, ingester.instances["test"].completingBlocks, 0)
	require.Len(t, ingester.instances["test"].completeBlocks, 1)

	// should be able to find old traces that were replayed
	for i, traceID := range traceIDs {
		foundTrace, err := ingester.FindTraceByID(ctx, &tempopb.TraceByIDRequest{
			TraceID: traceID,
		})
		require.NoError(t, err, "unexpected error querying")
		equal := proto.Equal(traces[i], foundTrace.Trace)
		require.True(t, equal)
	}
}

func TestSearchWAL(t *testing.T) {
	tmpDir, err := os.MkdirTemp("/tmp", "")
	require.NoError(t, err, "unexpected error getting tempdir")
	defer os.RemoveAll(tmpDir)

	i := defaultIngesterModule(t, tmpDir)
	inst, _ := i.getOrCreateInstance("test")
	require.NotNil(t, inst)

	// create some search data
	id := make([]byte, 16)
	_, err = rand.Read(id)
	require.NoError(t, err)
	trace := test.MakeTrace(10, id)
	traceBytes, err := trace.Marshal()
	require.NoError(t, err)
	entry := &tempofb.SearchEntryMutable{}
	entry.TraceID = id
	entry.AddTag("foo", "bar")
	searchBytes := entry.ToBytes()

	// push to instance
	require.NoError(t, inst.PushBytes(context.Background(), id, traceBytes, searchBytes))

	// Write wal
	require.NoError(t, inst.CutCompleteTraces(0, true))

	// search WAL
	ctx := user.InjectOrgID(context.Background(), "test")
	searchReq := &tempopb.SearchRequest{Tags: map[string]string{
		"foo": "bar",
	}}
	results, err := inst.Search(ctx, searchReq)
	require.NoError(t, err)
	require.Equal(t, uint32(1), results.Metrics.InspectedTraces)

	// Shutdown
	require.NoError(t, i.stopping(nil))

	// the below tests sometimes fail in CI. Awful bandaid sleep slammed in here
	// to reduce occurrence of this issue. todo: fix it properly
	time.Sleep(500 * time.Millisecond)

	// replay wal
	i = defaultIngesterModule(t, tmpDir)
	inst, ok := i.getInstanceByID("test")
	require.True(t, ok)

	results, err = inst.Search(ctx, searchReq)
	require.NoError(t, err)
	require.Equal(t, uint32(1), results.Metrics.InspectedTraces)
}

// TODO - This test is flaky and commented out until it's fixed
// TestWalReplayDeletesLocalBlocks simulates the condition where an ingester restarts after a wal is completed
// to the local disk, but before the wal is deleted. On startup both blocks exist, and the ingester now errs
// on the side of caution and chooses to replay the wal instead of rediscovering the local block.
/*func TestWalReplayDeletesLocalBlocks(t *testing.T) {
	tmpDir := t.TempDir()

	i, _, _ := defaultIngester(t, tmpDir)
	inst, ok := i.getInstanceByID("test")
	require.True(t, ok)

	// Write wal
	err := inst.CutCompleteTraces(0, true)
	require.NoError(t, err)
	blockID, err := inst.CutBlockIfReady(0, 0, true)
	require.NoError(t, err)

	// Complete block
	err = inst.CompleteBlock(blockID)
	require.NoError(t, err)

	// Shutdown
	err = i.stopping(nil)
	require.NoError(t, err)

	// At this point both wal and complete block exist.
	// The wal still exists because we manually completed it and
	// didn't delete it with ClearCompletingBlock()
	require.Len(t, inst.completingBlocks, 1)
	require.Len(t, inst.completeBlocks, 1)
	require.Equal(t, blockID, inst.completingBlocks[0].BlockID())
	require.Equal(t, blockID, inst.completeBlocks[0].BlockMeta().BlockID)

	// Simulate a restart by creating a new ingester.
	i, _, _ = defaultIngester(t, tmpDir)
	inst, ok = i.getInstanceByID("test")
	require.True(t, ok)

	// After restart we only have the 1 wal block
	// TODO - fix race conditions here around access inst fields outside of mutex
	require.Len(t, inst.completingBlocks, 1)
	require.Len(t, inst.completeBlocks, 0)
	require.Equal(t, blockID, inst.completingBlocks[0].BlockID())

	// Shutdown, cleanup
	err = i.stopping(nil)
	require.NoError(t, err)
}
*/

func TestFlush(t *testing.T) {
	tmpDir, err := os.MkdirTemp("/tmp", "")
	require.NoError(t, err, "unexpected error getting tempdir")
	defer os.RemoveAll(tmpDir)

	ctx := user.InjectOrgID(context.Background(), "test")
	ingester, traces, traceIDs := defaultIngester(t, tmpDir)

	for pos, traceID := range traceIDs {
		foundTrace, err := ingester.FindTraceByID(ctx, &tempopb.TraceByIDRequest{
			TraceID: traceID,
		})
		require.NoError(t, err, "unexpected error querying")
		require.Equal(t, traces[pos], foundTrace.Trace)
	}

	// stopping the ingester should force cut all live traces to disk
	require.NoError(t, ingester.stopping(nil))

	// create new ingester.  this should replay wal!
	ingester, _, _ = defaultIngester(t, tmpDir)

	// should be able to find old traces that were replayed
	for i, traceID := range traceIDs {
		foundTrace, err := ingester.FindTraceByID(ctx, &tempopb.TraceByIDRequest{
			TraceID: traceID,
		})
		require.NoError(t, err, "unexpected error querying")
		equal := proto.Equal(traces[i], foundTrace.Trace)
		require.True(t, equal)
	}
}

func defaultIngesterModule(t *testing.T, tmpDir string) *Ingester {
	ingesterConfig := defaultIngesterTestConfig()
	limits, err := overrides.NewOverrides(defaultLimitsTestConfig())
	require.NoError(t, err, "unexpected error creating overrides")

	s, err := storage.NewStore(storage.Config{
		Trace: tempodb.Config{
			Backend: "local",
			Local: &local.Config{
				Path: tmpDir,
			},
			Block: &encoding.BlockConfig{
				IndexDownsampleBytes: 2,
				BloomFP:              0.01,
				BloomShardSizeBytes:  100_000,
				Encoding:             backend.EncLZ4_1M,
				IndexPageSizeBytes:   1000,
			},
			WAL: &wal.Config{
				Filepath: tmpDir,
			},
		},
	}, log.NewNopLogger())
	require.NoError(t, err, "unexpected error store")

	ingester, err := New(ingesterConfig, s, limits, prometheus.NewPedanticRegistry())
	require.NoError(t, err, "unexpected error creating ingester")
	ingester.replayJitter = false

	err = ingester.starting(context.Background())
	require.NoError(t, err, "unexpected error starting ingester")

	return ingester
}

func defaultIngester(t *testing.T, tmpDir string) (*Ingester, []*tempopb.Trace, [][]byte) {
	ingester := defaultIngesterModule(t, tmpDir)

	// make some fake traceIDs/requests
	traces := make([]*tempopb.Trace, 0)

	traceIDs := make([][]byte, 0)
	for i := 0; i < 10; i++ {
		id := make([]byte, 16)
		_, err := rand.Read(id)
		require.NoError(t, err)

		testTrace := test.MakeTrace(10, id)
		trace.SortTrace(testTrace)

		traces = append(traces, testTrace)
		traceIDs = append(traceIDs, id)
	}

	for i, trace := range traces {
		for _, batch := range trace.Batches {
			pushBatch(t, ingester, batch, traceIDs[i])
		}
	}

	return ingester, traces, traceIDs
}

func defaultIngesterTestConfig() Config {
	cfg := Config{}

	flagext.DefaultValues(&cfg.LifecyclerConfig)
	mockStore, _ := consul.NewInMemoryClient(
		ring.GetCodec(),
		log.NewNopLogger(),
		nil,
	)

	cfg.FlushCheckPeriod = 99999 * time.Hour
	cfg.MaxTraceIdle = 99999 * time.Hour
	cfg.ConcurrentFlushes = 1
	cfg.LifecyclerConfig.RingConfig.KVStore.Mock = mockStore
	cfg.LifecyclerConfig.NumTokens = 1
	cfg.LifecyclerConfig.ListenPort = 0
	cfg.LifecyclerConfig.Addr = "localhost"
	cfg.LifecyclerConfig.ID = "localhost"
	cfg.LifecyclerConfig.FinalSleep = 0

	return cfg
}

func defaultLimitsTestConfig() overrides.Limits {
	limits := overrides.Limits{}
	flagext.DefaultValues(&limits)
	return limits
}

func pushBatch(t *testing.T, i *Ingester, batch *v1.ResourceSpans, id []byte) {
	ctx := user.InjectOrgID(context.Background(), "test")

	pbTrace := &tempopb.Trace{
		Batches: []*v1.ResourceSpans{batch},
	}

	buffer := tempopb.SliceFromBytePool(pbTrace.Size())
	_, err := pbTrace.MarshalToSizedBuffer(buffer)
	require.NoError(t, err)

	_, err = i.PushBytes(ctx, &tempopb.PushBytesRequest{
		Traces: []tempopb.PreallocBytes{
			{
				Slice: buffer,
			},
		},
		Ids: []tempopb.PreallocBytes{
			{
				Slice: id,
			},
		},
	})
	require.NoError(t, err)
}
