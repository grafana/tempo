package ingester

import (
	"context"
	"io/ioutil"
	"math/rand"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/grafana/tempo/pkg/util/validation"

	"github.com/stretchr/testify/assert"
)

type ringCountMock struct {
	count int
}

func (m *ringCountMock) HealthyInstancesCount() int {
	return m.count
}

func TestInstance(t *testing.T) {
	limits, err := validation.NewOverrides(validation.Limits{})
	assert.NoError(t, err, "unexpected error creating limits")
	limiter := NewLimiter(limits, &ringCountMock{count: 1}, 1)

	tempDir, err := ioutil.TempDir("/tmp", "")
	assert.NoError(t, err, "unexpected error getting temp dir")
	defer os.RemoveAll(tempDir)

	ingester, _, _ := defaultIngester(t, tempDir)
	wal := ingester.wal

	request := test.MakeRequest(10, []byte{})

	i, err := newInstance("fake", limiter, wal)
	assert.NoError(t, err, "unexpected error creating new instance")
	err = i.Push(context.Background(), request)
	assert.NoError(t, err)

	err = i.CutCompleteTraces(0, true)
	assert.NoError(t, err)

	ready, err := i.CutBlockIfReady(5, 0, false)
	assert.True(t, ready, "there should be no cut blocks")
	assert.NoError(t, err, "unexpected error cutting block")

	ready, err = i.CutBlockIfReady(0, 30*time.Hour, false)
	assert.True(t, ready, "there should be no cut blocks")
	assert.NoError(t, err, "unexpected error cutting block")

	block := i.GetBlockToBeFlushed()
	assert.NotNil(t, block)

	err = ingester.store.WriteBlock(context.Background(), block)
	assert.NoError(t, err)

	err = i.ClearCompleteBlocks(0)
	assert.NoError(t, err)
	assert.Len(t, i.completeBlocks, 1)

	err = i.resetHeadBlock()
	assert.NoError(t, err, "unexpected error resetting block")
}

func TestInstanceFind(t *testing.T) {
	limits, err := validation.NewOverrides(validation.Limits{})
	assert.NoError(t, err, "unexpected error creating limits")
	limiter := NewLimiter(limits, &ringCountMock{count: 1}, 1)

	tempDir, err := ioutil.TempDir("/tmp", "")
	assert.NoError(t, err, "unexpected error getting temp dir")
	defer os.RemoveAll(tempDir)

	ingester, _, _ := defaultIngester(t, tempDir)
	wal := ingester.wal

	request := test.MakeRequest(10, []byte{})
	traceID := request.Batch.Spans[0].TraceId

	i, err := newInstance("fake", limiter, wal)
	assert.NoError(t, err, "unexpected error creating new instance")
	err = i.Push(context.Background(), request)
	assert.NoError(t, err)

	trace, err := i.FindTraceByID(traceID)
	assert.NotNil(t, trace)
	assert.NoError(t, err)

	err = i.CutCompleteTraces(0, true)
	assert.NoError(t, err)

	trace, err = i.FindTraceByID(traceID)
	assert.NotNil(t, trace)
	assert.NoError(t, err)

	ready, err := i.CutBlockIfReady(0, 0, false)
	assert.True(t, ready)
	assert.NoError(t, err)

	trace, err = i.FindTraceByID(traceID)
	assert.NotNil(t, trace)
	assert.NoError(t, err)
}

func BenchmarkShardedTraceMap(b *testing.B) {
	// benchmark instance.Push and instance.FindTraceByID
	i := &instance{}
	for s := range i.traceMapShards {
		i.traceMapShards[s] = &traceMapShard{
			tracesMtx: new(sync.RWMutex),
			traces:    make(map[traceFingerprint]*trace),
		}
	}

	id := make([]byte, 16)
	_, err := rand.Read(id)
	assert.NoError(b, err)

	trace := test.MakeTrace(10000, id)

	for n := 0; n < b.N; n++ {
		for _, batch := range trace.Batches {
			err := i.Push(context.Background(),
				&tempopb.PushRequest{
					Batch: batch,
				})
			assert.NoError(b, err, "unexpected error pushing")
		}

		t, err := i.FindTraceByID(id)
		assert.NotNil(b, t)
		assert.NoError(b, err)
	}
}
