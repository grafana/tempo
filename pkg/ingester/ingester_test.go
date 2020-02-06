package ingester

import (
	"context"
	"crypto/rand"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/cortexproject/cortex/pkg/ring"
	"github.com/cortexproject/cortex/pkg/ring/kv"
	"github.com/cortexproject/cortex/pkg/ring/kv/codec"
	"github.com/cortexproject/cortex/pkg/util/flagext"
	"github.com/go-kit/kit/log"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"
	"github.com/weaveworks/common/user"

	"github.com/grafana/frigg/friggdb"
	"github.com/grafana/frigg/friggdb/backend/local"
	"github.com/grafana/frigg/pkg/friggpb"
	"github.com/grafana/frigg/pkg/ingester/client"
	"github.com/grafana/frigg/pkg/storage"
	"github.com/grafana/frigg/pkg/util/test"
	"github.com/grafana/frigg/pkg/util/validation"
)

func TestPushQuery(t *testing.T) {
	tmpDir, err := ioutil.TempDir("/tmp", "")
	assert.NoError(t, err, "unexpected error getting tempdir")
	defer os.RemoveAll(tmpDir)

	ctx := user.InjectOrgID(context.Background(), "test")
	ingester, traces, traceIDs := defaultIngester(t, tmpDir)

	// now query and get nils (nothing has been flushed)
	for _, traceID := range traceIDs {
		foundTrace, err := ingester.FindTraceByID(ctx, &friggpb.TraceByIDRequest{
			TraceID: traceID,
		})
		assert.NoError(t, err, "unexpected error querying")
		assert.Nil(t, foundTrace.Trace)
	}

	// force cut all traces
	for _, instance := range ingester.instances {
		err = instance.CutCompleteTraces(0, true)
		assert.NoError(t, err, "unexpected error cutting traces")
	}

	// should be able to find them now
	for i, traceID := range traceIDs {
		foundTrace, err := ingester.FindTraceByID(ctx, &friggpb.TraceByIDRequest{
			TraceID: traceID,
		})
		assert.NoError(t, err, "unexpected error querying")
		equal := proto.Equal(traces[i], foundTrace.Trace)
		assert.True(t, equal)
	}
}

func TestWal(t *testing.T) {
	tmpDir, err := ioutil.TempDir("/tmp", "")
	assert.NoError(t, err, "unexpected error getting tempdir")
	defer os.RemoveAll(tmpDir)

	ctx := user.InjectOrgID(context.Background(), "test")
	ingester, traces, traceIDs := defaultIngester(t, tmpDir)

	// now query and get nils (nothing has been flushed)
	for _, traceID := range traceIDs {
		foundTrace, err := ingester.FindTraceByID(ctx, &friggpb.TraceByIDRequest{
			TraceID: traceID,
		})
		assert.NoError(t, err, "unexpected error querying")
		assert.Nil(t, foundTrace.Trace)
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
		foundTrace, err := ingester.FindTraceByID(ctx, &friggpb.TraceByIDRequest{
			TraceID: traceID,
		})
		assert.NoError(t, err, "unexpected error querying")
		equal := proto.Equal(traces[i], foundTrace.Trace)
		assert.True(t, equal)
	}
}

func defaultIngester(t *testing.T, tmpDir string) (*Ingester, []*friggpb.Trace, [][]byte) {
	ingesterConfig := defaultIngesterTestConfig(t)
	limits, err := validation.NewOverrides(defaultLimitsTestConfig())
	assert.NoError(t, err, "unexpected error creating overrides")

	s, err := storage.NewStore(storage.Config{
		Trace: friggdb.Config{
			Backend:                  "local",
			BloomFilterFalsePositive: .01,
			Local: local.Config{
				Path: tmpDir,
			},
			WALFilepath:     tmpDir,
			IndexDownsample: 2,
		},
	}, limits, log.NewNopLogger())
	assert.NoError(t, err, "unexpected error store")

	ingester, err := New(ingesterConfig, client.Config{}, s, limits)
	assert.NoError(t, err, "unexpected error creating ingester")

	// make some fake traceIDs/requests
	traces := make([]*friggpb.Trace, 0)

	traceIDs := make([][]byte, 0)
	for i := 0; i < 10; i++ {
		id := make([]byte, 16)
		rand.Read(id)

		traces = append(traces, test.MakeTrace(10, id))
		traceIDs = append(traceIDs, id)
	}

	ctx := user.InjectOrgID(context.Background(), "test")
	for _, trace := range traces {
		for _, batch := range trace.Batches {
			_, err := ingester.Push(ctx,
				&friggpb.PushRequest{
					Batch: batch,
				})
			assert.NoError(t, err, "unexpected error pushing")
		}
	}

	return ingester, traces, traceIDs
}

func defaultIngesterTestConfig(t *testing.T) Config {
	kvClient, err := kv.NewClient(kv.Config{Store: "inmemory"}, codec.Proto{Factory: ring.ProtoDescFactory})
	assert.NoError(t, err)

	cfg := Config{}
	flagext.DefaultValues(&cfg)
	cfg.FlushCheckPeriod = 99999 * time.Hour
	cfg.MaxTraceIdle = 99999 * time.Hour
	cfg.ConcurrentFlushes = 1
	cfg.LifecyclerConfig.RingConfig.KVStore.Mock = kvClient
	cfg.LifecyclerConfig.NumTokens = 1
	cfg.LifecyclerConfig.ListenPort = func(i int) *int { return &i }(0)
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
