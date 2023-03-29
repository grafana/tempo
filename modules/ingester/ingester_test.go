package ingester

import (
	"context"
	"crypto/rand"
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
	"github.com/grafana/tempo/pkg/model"
	"github.com/grafana/tempo/pkg/model/trace"
	model_v1 "github.com/grafana/tempo/pkg/model/v1"
	model_v2 "github.com/grafana/tempo/pkg/model/v2"
	"github.com/grafana/tempo/pkg/tempopb"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/grafana/tempo/tempodb"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/local"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/grafana/tempo/tempodb/wal"
)

func TestPushQueryAllEncodings(t *testing.T) {
	for _, e := range model.AllEncodings {
		t.Run(e, func(t *testing.T) {
			var push func(testing.TB, *Ingester, *v1.ResourceSpans, []byte)

			switch e {
			case model_v1.Encoding:
				push = pushBatchV1
			case model_v2.Encoding:
				push = pushBatchV2
			default:
				t.Fatal("unsupported encoding", e)
			}

			tmpDir := t.TempDir()
			ctx := user.InjectOrgID(context.Background(), "test")
			ingester, traces, traceIDs := defaultIngesterWithPush(t, tmpDir, push)

			// live trace search
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

			// head block search
			for i, traceID := range traceIDs {
				foundTrace, err := ingester.FindTraceByID(ctx, &tempopb.TraceByIDRequest{
					TraceID: traceID,
				})
				require.NoError(t, err, "unexpected error querying")
				trace.SortTrace(foundTrace.Trace)
				equal := proto.Equal(traces[i], foundTrace.Trace)
				require.True(t, equal)
			}
		})
	}
}

func TestFullTraceReturned(t *testing.T) {
	ctx := user.InjectOrgID(context.Background(), "test")
	ingester, _, _ := defaultIngester(t, t.TempDir())

	traceID := make([]byte, 16)
	_, err := rand.Read(traceID)
	require.NoError(t, err)
	testTrace := test.MakeTrace(2, traceID) // 2 batches
	trace.SortTrace(testTrace)

	// push the first batch
	pushBatchV2(t, ingester, testTrace.Batches[0], traceID)

	// force cut all traces
	for _, instance := range ingester.instances {
		err = instance.CutCompleteTraces(0, true)
		require.NoError(t, err, "unexpected error cutting traces")
	}

	// push the 2nd batch
	pushBatchV2(t, ingester, testTrace.Batches[1], traceID)

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
	tmpDir := t.TempDir()

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
		require.NotNil(t, foundTrace.Trace)
		trace.SortTrace(foundTrace.Trace)
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

		trace.SortTrace(foundTrace.Trace)
		test.TracesEqual(t, traces[i], foundTrace.Trace)
	}
}

func TestWalDropsZeroLength(t *testing.T) {
	tmpDir := t.TempDir()
	ingester, _, _ := defaultIngester(t, tmpDir)

	// force cut all traces and wipe wal
	for _, instance := range ingester.instances {
		err := instance.CutCompleteTraces(0, true)
		require.NoError(t, err, "unexpected error cutting traces")

		blockID, err := instance.CutBlockIfReady(0, 0, true)
		require.NoError(t, err)

		err = instance.CompleteBlock(blockID)
		require.NoError(t, err)

		err = instance.ClearCompletingBlock(blockID)
		require.NoError(t, err)

		err = ingester.local.ClearBlock(blockID, instance.instanceID)
		require.NoError(t, err)
	}

	// create new ingester. we should have no tenants b/c we all our wals should have been 0 length
	ingester, _, _ = defaultIngesterWithPush(t, tmpDir, func(t testing.TB, i *Ingester, rs *v1.ResourceSpans, b []byte) {})
	require.Equal(t, 0, len(ingester.instances))
}

func TestSearchWAL(t *testing.T) {
	tmpDir, err := os.MkdirTemp("/tmp", "")
	require.NoError(t, err, "unexpected error getting tempdir")
	defer os.RemoveAll(tmpDir)

	i := defaultIngesterModule(t, tmpDir)
	inst, _ := i.getOrCreateInstance("test")
	require.NotNil(t, inst)

	dec := model.MustNewSegmentDecoder(model.CurrentEncoding)

	// create some search data
	id := make([]byte, 16)
	_, err = rand.Read(id)
	require.NoError(t, err)
	trace := test.MakeTrace(10, id)
	b1, err := dec.PrepareForWrite(trace, 0, 0)
	require.NoError(t, err)

	// push to instance
	require.NoError(t, inst.PushBytes(context.Background(), id, b1))

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
	tmpDir := t.TempDir()

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
		trace.SortTrace(foundTrace.Trace)
		equal := proto.Equal(traces[i], foundTrace.Trace)
		require.True(t, equal)
	}
}

func defaultIngesterModule(t testing.TB, tmpDir string) *Ingester {
	ingesterConfig := defaultIngesterTestConfig()
	limits, err := overrides.NewOverrides(defaultLimitsTestConfig())
	require.NoError(t, err, "unexpected error creating overrides")

	s, err := storage.NewStore(storage.Config{
		Trace: tempodb.Config{
			Backend: "local",
			Local: &local.Config{
				Path: tmpDir,
			},
			Block: &common.BlockConfig{
				IndexDownsampleBytes: 2,
				BloomFP:              0.01,
				BloomShardSizeBytes:  100_000,
				Version:              encoding.DefaultEncoding().Version(),
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

func defaultIngester(t testing.TB, tmpDir string) (*Ingester, []*tempopb.Trace, [][]byte) {
	return defaultIngesterWithPush(t, tmpDir, pushBatchV2)
}

func defaultIngesterWithPush(t testing.TB, tmpDir string, push func(testing.TB, *Ingester, *v1.ResourceSpans, []byte)) (*Ingester, []*tempopb.Trace, [][]byte) {
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
			push(t, ingester, batch, traceIDs[i])
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

func pushBatchV2(t testing.TB, i *Ingester, batch *v1.ResourceSpans, id []byte) {
	ctx := user.InjectOrgID(context.Background(), "test")
	batchDecoder := model.MustNewSegmentDecoder(model_v2.Encoding)

	pbTrace := &tempopb.Trace{
		Batches: []*v1.ResourceSpans{batch},
	}

	buffer, err := batchDecoder.PrepareForWrite(pbTrace, 0, 0)
	require.NoError(t, err)

	_, err = i.PushBytesV2(ctx, &tempopb.PushBytesRequest{
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

func pushBatchV1(t testing.TB, i *Ingester, batch *v1.ResourceSpans, id []byte) {
	ctx := user.InjectOrgID(context.Background(), "test")

	batchDecoder := model.MustNewSegmentDecoder(model_v1.Encoding)

	pbTrace := &tempopb.Trace{
		Batches: []*v1.ResourceSpans{batch},
	}

	buffer, err := batchDecoder.PrepareForWrite(pbTrace, 0, 0)
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
