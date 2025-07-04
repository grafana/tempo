package ingester

import (
	"context"
	"crypto/rand"
	"flag"
	"os"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/gogo/protobuf/proto"
	"github.com/google/uuid"
	"github.com/grafana/dskit/flagext"
	"github.com/grafana/dskit/kv/consul"
	"github.com/grafana/dskit/ring"
	"github.com/grafana/dskit/user"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
	"github.com/grafana/tempo/tempodb/encoding/vparquet4"
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
				err := instance.CutCompleteTraces(0, 0, true)
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
	pushBatchV2(t, ingester, testTrace.ResourceSpans[0], traceID)

	// force cut all traces
	for _, instance := range ingester.instances {
		err = instance.CutCompleteTraces(0, 0, true)
		require.NoError(t, err, "unexpected error cutting traces")
	}

	// push the 2nd batch
	pushBatchV2(t, ingester, testTrace.ResourceSpans[1], traceID)

	// make sure the trace comes back whole
	foundTrace, err := ingester.FindTraceByID(ctx, &tempopb.TraceByIDRequest{
		TraceID: traceID,
	})
	require.NoError(t, err, "unexpected error querying")
	require.True(t, proto.Equal(testTrace, foundTrace.Trace))

	// force cut all traces
	for _, instance := range ingester.instances {
		err = instance.CutCompleteTraces(0, 0, true)
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
		err := instance.CutCompleteTraces(0, 0, true)
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
		err := instance.CutCompleteTraces(0, 0, true)
		require.NoError(t, err, "unexpected error cutting traces")

		blockID, err := instance.CutBlockIfReady(0, 0, true)
		require.NoError(t, err)

		err = instance.CompleteBlock(context.Background(), blockID)
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
	require.NoError(t, inst.CutCompleteTraces(0, 0, true))

	// search WAL
	ctx := user.InjectOrgID(context.Background(), "test")
	searchReq := &tempopb.SearchRequest{Query: "{ }"}
	results, err := inst.Search(ctx, searchReq)
	require.NoError(t, err)
	require.Equal(t, 1, len(results.Traces))

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
	require.Equal(t, 1, len(results.Traces))
}

func TestRediscoverLocalBlocks(t *testing.T) {
	tmpDir := t.TempDir()

	ctx := user.InjectOrgID(context.Background(), "test")
	ingester, traces, traceIDs := defaultIngester(t, tmpDir)

	// force cut all traces
	for _, instance := range ingester.instances {
		err := instance.CutCompleteTraces(0, 0, true)
		require.NoError(t, err, "unexpected error cutting traces")
	}

	// force complete all blocks
	for _, instance := range ingester.instances {
		blockID, err := instance.CutBlockIfReady(0, 0, true)
		require.NoError(t, err)

		err = instance.CompleteBlock(context.Background(), blockID)
		require.NoError(t, err)

		err = instance.ClearCompletingBlock(blockID)
		require.NoError(t, err)
	}

	// create new ingester.  this should rediscover local blocks
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
}

func TestRediscoverDropsInvalidBlocks(t *testing.T) {
	tmpDir := t.TempDir()

	ctx := user.InjectOrgID(context.Background(), "test")
	ingester, _, _ := defaultIngester(t, tmpDir)

	// force cut all traces
	for _, instance := range ingester.instances {
		err := instance.CutCompleteTraces(0, 0, true)
		require.NoError(t, err, "unexpected error cutting traces")
	}

	// force complete all blocks
	for _, instance := range ingester.instances {
		blockID, err := instance.CutBlockIfReady(0, 0, true)
		require.NoError(t, err)

		err = instance.CompleteBlock(context.Background(), blockID)
		require.NoError(t, err)

		err = instance.ClearCompletingBlock(blockID)
		require.NoError(t, err)
	}

	// create new ingester. this should rediscover local blocks. there should be 1 block
	ingester, _, _ = defaultIngester(t, tmpDir)

	instance, ok := ingester.instances["test"]
	require.True(t, ok)
	require.Len(t, instance.completeBlocks, 1)

	// now mangle a complete block
	instance, ok = ingester.instances["test"]
	require.True(t, ok)
	require.Len(t, instance.completeBlocks, 1)

	// this cheats by reaching into the internals of the block and overwriting the parquet file directly. if this test starts failing
	// it could be b/c the block internals changed and this no longer breaks a block
	block := instance.completeBlocks[0]
	err := block.writer.Write(ctx, vparquet4.DataFileName, uuid.UUID(block.BlockMeta().BlockID), "test", []byte("mangled"), nil)
	require.NoError(t, err)

	// create new ingester. this should rediscover local blocks. there should be 0 blocks
	ingester, _, _ = defaultIngester(t, tmpDir)

	instance, ok = ingester.instances["test"]
	require.True(t, ok)
	require.Len(t, instance.completeBlocks, 0)
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
	err := inst.CutCompleteTraces(0, 0, true)
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

func TestIngesterStartingReadOnly(t *testing.T) {
	ctx := user.InjectOrgID(context.Background(), "test")

	limits, err := overrides.NewOverrides(defaultOverridesConfig(), nil, prometheus.DefaultRegisterer)
	require.NoError(t, err)

	// Create ingester but without starting it
	ingester, err := New(
		defaultIngesterTestConfig(),
		defaultIngesterStore(t, t.TempDir()),
		limits,
		prometheus.NewPedanticRegistry(),
		false)
	require.NoError(t, err)

	_, err = ingester.PushBytesV2(ctx, &tempopb.PushBytesRequest{})
	require.ErrorIs(t, err, ErrStarting)
}

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

func TestDedicatedColumns(t *testing.T) {
	tmpDir, err := os.MkdirTemp("/tmp", "")
	require.NoError(t, err, "unexpected error getting tempdir")
	t.Cleanup(func() { _ = os.RemoveAll(tmpDir) })

	cfg := overrides.Config{}
	cfg.RegisterFlagsAndApplyDefaults(&flag.FlagSet{})
	cfg.Defaults.Storage.DedicatedColumns = backend.DedicatedColumns{{Scope: "span", Name: "foo", Type: "string"}}

	i := defaultIngesterWithOverrides(t, tmpDir, cfg)
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
	require.NoError(t, inst.CutCompleteTraces(0, 0, true))

	assert.Equal(t, cfg.Defaults.Storage.DedicatedColumns, inst.headBlock.BlockMeta().DedicatedColumns)

	// TODO: This search should find a match once the read path is supported
	ctx := user.InjectOrgID(context.Background(), "test")
	searchReq := &tempopb.SearchRequest{Query: "{span.foo=\"bar\"}"}
	results, err := inst.Search(ctx, searchReq)
	require.NoError(t, err)
	assert.Len(t, results.Traces, 0)

	blockID, err := inst.CutBlockIfReady(0, 0, true)
	require.NoError(t, err)

	// TODO: This check should be included as part of the read path
	inst.blocksMtx.RLock()
	for _, b := range inst.completingBlocks {
		assert.Equal(t, cfg.Defaults.Storage.DedicatedColumns, b.BlockMeta().DedicatedColumns)
	}
	inst.blocksMtx.RUnlock()

	// Complete block
	err = inst.CompleteBlock(context.Background(), blockID)
	require.NoError(t, err)

	// TODO: This check should be included as part of the read path
	inst.blocksMtx.RLock()
	for _, b := range inst.completeBlocks {
		assert.Equal(t, cfg.Defaults.Storage.DedicatedColumns, b.BlockMeta().DedicatedColumns)
	}
	inst.blocksMtx.RUnlock()
}

func defaultIngesterModule(t testing.TB, tmpDir string) *Ingester {
	return defaultIngesterWithOverrides(t, tmpDir, defaultOverridesConfig())
}

func defaultIngesterStore(t testing.TB, tmpDir string) storage.Store {
	s, err := storage.NewStore(storage.Config{
		Trace: tempodb.Config{
			Backend: backend.Local,
			Local: &local.Config{
				Path: tmpDir,
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
				Filepath: tmpDir,
			},
		},
	}, nil, log.NewNopLogger())
	require.NoError(t, err, "unexpected error store")

	return s
}

func defaultIngesterWithOverrides(t testing.TB, tmpDir string, o overrides.Config) *Ingester {
	ingesterConfig := defaultIngesterTestConfig()
	limits, err := overrides.NewOverrides(o, nil, prometheus.DefaultRegisterer)
	require.NoError(t, err, "unexpected error creating overrides")

	s := defaultIngesterStore(t, tmpDir)

	ingester, err := New(ingesterConfig, s, limits, prometheus.NewPedanticRegistry(), false)
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
		for _, batch := range trace.ResourceSpans {
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

	cfg.FlushOpTimeout = 99999 * time.Hour
	cfg.FlushCheckPeriod = 99999 * time.Hour
	cfg.FlushObjectStorage = true
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

func defaultOverridesConfig() overrides.Config {
	config := overrides.Config{}
	config.RegisterFlagsAndApplyDefaults(&flag.FlagSet{})
	return config
}

func pushBatchV2(t testing.TB, i *Ingester, batch *v1.ResourceSpans, id []byte) {
	ctx := user.InjectOrgID(context.Background(), "test")

	_, err := i.PushBytesV2(ctx, makePushBytesRequest(id, batch))
	require.NoError(t, err)
}

func pushBatchV1(t testing.TB, i *Ingester, batch *v1.ResourceSpans, id []byte) {
	ctx := user.InjectOrgID(context.Background(), "test")

	batchDecoder := model.MustNewSegmentDecoder(model_v1.Encoding)

	pbTrace := &tempopb.Trace{
		ResourceSpans: []*v1.ResourceSpans{batch},
	}

	buffer, err := batchDecoder.PrepareForWrite(pbTrace, 0, 0)
	require.NoError(t, err)

	_, err = i.PushBytes(ctx, &tempopb.PushBytesRequest{
		Traces: []tempopb.PreallocBytes{
			{
				Slice: buffer,
			},
		},
		Ids: [][]byte{
			id,
		},
	})
	require.NoError(t, err)
}
