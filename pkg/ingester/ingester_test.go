package ingester

import (
	"context"
	"io/ioutil"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/cortexproject/cortex/pkg/ring"
	"github.com/cortexproject/cortex/pkg/ring/kv/consul"
	"github.com/cortexproject/cortex/pkg/util/flagext"
	"github.com/go-kit/kit/log"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"
	"github.com/weaveworks/common/user"

	"github.com/grafana/tempo/pkg/ingester/client"
	"github.com/grafana/tempo/pkg/storage"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/grafana/tempo/pkg/util/validation"
	"github.com/grafana/tempo/tempodb"
	"github.com/grafana/tempo/tempodb/backend/local"
	"github.com/grafana/tempo/tempodb/wal"
)

func TestPushQuery(t *testing.T) {
	tmpDir, err := ioutil.TempDir("/tmp", "")
	assert.NoError(t, err, "unexpected error getting tempdir")
	defer os.RemoveAll(tmpDir)

	ctx := user.InjectOrgID(context.Background(), "test")
	ingester, traces, traceIDs := defaultIngester(t, tmpDir)

	for pos, traceID := range traceIDs {
		foundTrace, err := ingester.FindTraceByID(ctx, &tempopb.TraceByIDRequest{
			TraceID: traceID,
		})
		assert.NoError(t, err, "unexpected error querying")
		assert.Equal(t, foundTrace.Trace, traces[pos])
	}

	// force cut all traces
	for _, instance := range ingester.instances {
		err = instance.CutCompleteTraces(0, true)
		assert.NoError(t, err, "unexpected error cutting traces")
	}

	// should be able to find them now
	for i, traceID := range traceIDs {
		foundTrace, err := ingester.FindTraceByID(ctx, &tempopb.TraceByIDRequest{
			TraceID: traceID,
		})
		assert.NoError(t, err, "unexpected error querying")
		equal := proto.Equal(traces[i], foundTrace.Trace)
		assert.True(t, equal)
	}
}

func TestFullTraceReturned(t *testing.T) {
	tmpDir, err := ioutil.TempDir("/tmp", "")
	assert.NoError(t, err, "unexpected error getting tempdir")
	defer os.RemoveAll(tmpDir)

	ctx := user.InjectOrgID(context.Background(), "test")
	ingester, _, _ := defaultIngester(t, tmpDir)

	traceID := make([]byte, 16)
	_, err = rand.Read(traceID)
	assert.NoError(t, err)
	trace := test.MakeTrace(2, traceID) // 2 batches

	// push the first batch
	_, err = ingester.Push(ctx,
		&tempopb.PushRequest{
			Batch: trace.Batches[0],
		})
	assert.NoError(t, err, "unexpected error pushing")

	// force cut all traces
	for _, instance := range ingester.instances {
		err = instance.CutCompleteTraces(0, true)
		assert.NoError(t, err, "unexpected error cutting traces")
	}

	// push the 2nd batch
	_, err = ingester.Push(ctx,
		&tempopb.PushRequest{
			Batch: trace.Batches[1],
		})
	assert.NoError(t, err, "unexpected error pushing")

	// force cut all traces
	for _, instance := range ingester.instances {
		err = instance.CutCompleteTraces(0, true)
		assert.NoError(t, err, "unexpected error cutting traces")
	}

	// make sure the trace comes back whole
	foundTrace, err := ingester.FindTraceByID(ctx, &tempopb.TraceByIDRequest{
		TraceID: traceID,
	})
	assert.NoError(t, err, "unexpected error querying")
	equal := proto.Equal(trace, foundTrace.Trace)
	assert.True(t, equal)
}

func TestWal(t *testing.T) {
	tmpDir, err := ioutil.TempDir("/tmp", "")
	assert.NoError(t, err, "unexpected error getting tempdir")
	defer os.RemoveAll(tmpDir)

	ctx := user.InjectOrgID(context.Background(), "test")
	ingester, traces, traceIDs := defaultIngester(t, tmpDir)

	for pos, traceID := range traceIDs {
		foundTrace, err := ingester.FindTraceByID(ctx, &tempopb.TraceByIDRequest{
			TraceID: traceID,
		})
		assert.NoError(t, err, "unexpected error querying")
		assert.Equal(t, foundTrace.Trace, traces[pos])
	}

	// force cut all traces
	for _, instance := range ingester.instances {
		err := instance.CutCompleteTraces(0, true)
		assert.NoError(t, err, "unexpected error cutting traces")
	}

	// create new ingester.  this should replay wal!
	ingester, _, _ = defaultIngester(t, tmpDir)

	// should be able to find old traces that were replayed
	for i, traceID := range traceIDs {
		foundTrace, err := ingester.FindTraceByID(ctx, &tempopb.TraceByIDRequest{
			TraceID: traceID,
		})
		assert.NoError(t, err, "unexpected error querying")
		equal := proto.Equal(traces[i], foundTrace.Trace)
		assert.True(t, equal)
	}
}

func defaultIngester(t *testing.T, tmpDir string) (*Ingester, []*tempopb.Trace, [][]byte) {
	ingesterConfig := defaultIngesterTestConfig(t)
	limits, err := validation.NewOverrides(defaultLimitsTestConfig())
	assert.NoError(t, err, "unexpected error creating overrides")

	s, err := storage.NewStore(storage.Config{
		Trace: tempodb.Config{
			Backend: "local",
			Local: &local.Config{
				Path: tmpDir,
			},
			WAL: &wal.Config{
				Filepath:        tmpDir,
				IndexDownsample: 2,
				BloomFP:         .01,
			},
		},
	}, limits, log.NewNopLogger())
	assert.NoError(t, err, "unexpected error store")

	ingester, err := New(ingesterConfig, client.Config{}, s, limits)
	assert.NoError(t, err, "unexpected error creating ingester")

	// make some fake traceIDs/requests
	traces := make([]*tempopb.Trace, 0)

	traceIDs := make([][]byte, 0)
	for i := 0; i < 10; i++ {
		id := make([]byte, 16)
		_, err = rand.Read(id)
		assert.NoError(t, err)

		traces = append(traces, test.MakeTrace(10, id))
		traceIDs = append(traceIDs, id)
	}

	ctx := user.InjectOrgID(context.Background(), "test")
	for _, trace := range traces {
		for _, batch := range trace.Batches {
			_, err := ingester.Push(ctx,
				&tempopb.PushRequest{
					Batch: batch,
				})
			assert.NoError(t, err, "unexpected error pushing")
		}
	}

	return ingester, traces, traceIDs
}

func defaultIngesterTestConfig(t *testing.T) Config {
	cfg := Config{}
	flagext.DefaultValues(&cfg)
	cfg.FlushCheckPeriod = 99999 * time.Hour
	cfg.MaxTraceIdle = 99999 * time.Hour
	cfg.ConcurrentFlushes = 1
	cfg.LifecyclerConfig.RingConfig.KVStore.Mock = consul.NewInMemoryClient(ring.GetCodec())
	cfg.LifecyclerConfig.NumTokens = 1
	cfg.LifecyclerConfig.ListenPort = 0
	cfg.LifecyclerConfig.Addr = "localhost"
	cfg.LifecyclerConfig.ID = "localhost"
	cfg.LifecyclerConfig.FinalSleep = 0

	return cfg
}

func defaultLimitsTestConfig() validation.Limits {
	limits := validation.Limits{}
	flagext.DefaultValues(&limits)
	return limits
}
