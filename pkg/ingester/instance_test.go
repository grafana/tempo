package ingester

import (
	"context"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/grafana/frigg/pkg/util/test"
	"github.com/grafana/frigg/pkg/util/validation"

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
	i.Push(context.Background(), request)

	i.CutCompleteTraces(0, true)

	ready := i.CompleteBlocksReady(0)
	assert.False(t, ready, "there should be no cut blocks")

	err = i.CutBlockIfReady(5, 0)
	assert.NoError(t, err, "unexpected error cutting block")

	ready = i.CompleteBlocksReady(0)
	assert.True(t, ready, "block should be ready due to time")

	err = i.CutBlockIfReady(0, 30*time.Hour)
	assert.NoError(t, err, "unexpected error cutting block")

	block := i.GetCompleteBlock()
	err = i.ClearCompleteBlock(block)
	assert.NoError(t, err)

	block = i.GetCompleteBlock()
	err = i.ClearCompleteBlock(block)
	assert.NoError(t, err)

	err = i.resetHeadBlock()
	assert.NoError(t, err, "unexpected error resetting block")

	block = i.GetCompleteBlock()
	assert.Nil(t, block)
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
	i.Push(context.Background(), request)

	trace, err := i.FindTraceByID(traceID)
	assert.Nil(t, trace)
	assert.NoError(t, err)

	err = i.CutCompleteTraces(0, true)
	assert.NoError(t, err)

	trace, err = i.FindTraceByID(traceID)
	assert.NotNil(t, trace)
	assert.NoError(t, err)

	err = i.CutBlockIfReady(0, 0)
	assert.NoError(t, err)

	trace, err = i.FindTraceByID(traceID)
	assert.NotNil(t, trace)
	assert.NoError(t, err)
}
