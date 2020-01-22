package ingester

import (
	"context"
	"crypto/rand"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/cortexproject/cortex/pkg/chunk"
	"github.com/cortexproject/cortex/pkg/ring"
	"github.com/cortexproject/cortex/pkg/ring/kv"
	"github.com/cortexproject/cortex/pkg/ring/kv/codec"
	"github.com/cortexproject/cortex/pkg/util/flagext"
	"github.com/stretchr/testify/assert"
	"github.com/weaveworks/common/user"

	"github.com/joe-elliott/frigg/pkg/friggpb"
	"github.com/joe-elliott/frigg/pkg/ingester/client"
	"github.com/joe-elliott/frigg/pkg/util/test"
	"github.com/joe-elliott/frigg/pkg/util/validation"
)

func TestPushQuery(t *testing.T) {
	ingesterConfig := defaultIngesterTestConfig(t)
	limits, err := validation.NewOverrides(defaultLimitsTestConfig())
	assert.NoError(t, err, "unexpected error creating overrides")

	tmpDir, err := ioutil.TempDir("/tmp", "")
	assert.NoError(t, err, "unexpected error getting tempdir")
	defer os.RemoveAll(tmpDir)
	ingesterConfig.WALConfig.Filepath = tmpDir

	store := &mockStore{}
	ingester, err := New(ingesterConfig, client.Config{}, store, limits)

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
		for _, req := range trace.Batches {
			_, err = ingester.Push(ctx, req)
			assert.NoError(t, err, "unexpected error pushing")
		}
	}

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
		assert.NoError(t, err, "unexpected cutting traces")
	}

	// should be able to find them now
	for i, traceID := range traceIDs {
		foundTrace, err := ingester.FindTraceByID(ctx, &friggpb.TraceByIDRequest{
			TraceID: traceID,
		})
		assert.NoError(t, err, "unexpected error querying")
		assert.Equal(t, traces[i], foundTrace.Trace)
	}
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

type mockStore struct {
}

func (s *mockStore) Put(ctx context.Context, chunks []chunk.Chunk) error {
	return nil
}
