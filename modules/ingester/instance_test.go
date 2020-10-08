package ingester

import (
	"context"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util/test"

	"github.com/stretchr/testify/assert"
)

type ringCountMock struct {
	count int
}

func (m *ringCountMock) HealthyInstancesCount() int {
	return m.count
}

func TestInstance(t *testing.T) {
	limits, err := overrides.NewOverrides(overrides.Limits{})
	assert.NoError(t, err, "unexpected error creating limits")
	limiter := NewLimiter(limits, &ringCountMock{count: 1}, 1)

	tempDir, err := ioutil.TempDir("/tmp", "")
	assert.NoError(t, err, "unexpected error getting temp dir")
	defer os.RemoveAll(tempDir)

	ingester, _, _ := defaultIngester(t, tempDir)
	wal := ingester.store.WAL()

	request := test.MakeRequest(10, []byte{})

	i, err := newInstance("fake", limiter, wal)
	assert.NoError(t, err, "unexpected error creating new instance")
	err = i.Push(context.Background(), request)
	assert.NoError(t, err)

	err = i.CutCompleteTraces(0, true)
	assert.NoError(t, err)

	err = i.CutBlockIfReady(0, 0, false)
	assert.NoError(t, err, "unexpected error cutting block")

	// try a few times while the block gets completed
	block := i.GetBlockToBeFlushed()
	for j := 0; j < 5; j++ {
		if block != nil {
			continue
		}
		time.Sleep(100 * time.Millisecond)
		block = i.GetBlockToBeFlushed()
	}
	assert.NotNil(t, block)
	assert.Nil(t, i.completingBlock, 1)
	assert.Len(t, i.completeBlocks, 1)

	err = ingester.store.WriteBlock(context.Background(), block)
	assert.NoError(t, err)

	err = i.ClearFlushedBlocks(30 * time.Hour)
	assert.NoError(t, err)
	assert.Len(t, i.completeBlocks, 1)

	err = i.ClearFlushedBlocks(0)
	assert.NoError(t, err)
	assert.Len(t, i.completeBlocks, 0)

	err = i.resetHeadBlock()
	assert.NoError(t, err, "unexpected error resetting block")
}

func TestInstanceFind(t *testing.T) {
	limits, err := overrides.NewOverrides(overrides.Limits{})
	assert.NoError(t, err, "unexpected error creating limits")
	limiter := NewLimiter(limits, &ringCountMock{count: 1}, 1)

	tempDir, err := ioutil.TempDir("/tmp", "")
	assert.NoError(t, err, "unexpected error getting temp dir")
	defer os.RemoveAll(tempDir)

	ingester, _, _ := defaultIngester(t, tempDir)
	wal := ingester.store.WAL()

	request := test.MakeRequest(10, []byte{})
	traceID := test.MustTraceID(request)

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

	err = i.CutBlockIfReady(0, 0, false)
	assert.NoError(t, err)

	trace, err = i.FindTraceByID(traceID)
	assert.NotNil(t, trace)
	assert.NoError(t, err)
}

func TestInstanceDoesNotRace(t *testing.T) {
	limits, err := overrides.NewOverrides(overrides.Limits{})
	assert.NoError(t, err, "unexpected error creating limits")
	limiter := NewLimiter(limits, &ringCountMock{count: 1}, 1)

	tempDir, err := ioutil.TempDir("/tmp", "")
	assert.NoError(t, err, "unexpected error getting temp dir")
	defer os.RemoveAll(tempDir)

	ingester, _, _ := defaultIngester(t, tempDir)
	wal := ingester.store.WAL()

	i, err := newInstance("fake", limiter, wal)
	assert.NoError(t, err, "unexpected error creating new instance")

	end := make(chan struct{})

	concurrent := func(f func()) {
		for {
			select {
			case <-end:
				return
			default:
				f()
			}
		}
	}
	go concurrent(func() {
		request := test.MakeRequest(10, []byte{})
		_ = i.Push(context.Background(), request)
	})

	go concurrent(func() {
		_ = i.CutCompleteTraces(0, true)
	})

	go concurrent(func() {
		_ = i.CutBlockIfReady(0, 0, false)
	})

	go concurrent(func() {
		block := i.GetBlockToBeFlushed()
		if block != nil {
			_ = ingester.store.WriteBlock(context.Background(), block)
		}
	})

	go concurrent(func() {
		_ = i.ClearFlushedBlocks(0)
	})

	go concurrent(func() {
		_, _ = i.FindTraceByID([]byte{0x01})
	})

	time.Sleep(100 * time.Millisecond)
	close(end)
}

func TestInstanceLimits(t *testing.T) {
	limits, err := overrides.NewOverrides(overrides.Limits{
		MaxSpansPerTrace: 10,
	})
	assert.NoError(t, err, "unexpected error creating limits")
	limiter := NewLimiter(limits, &ringCountMock{count: 1}, 1)

	tempDir, err := ioutil.TempDir("/tmp", "")
	assert.NoError(t, err, "unexpected error getting temp dir")
	defer os.RemoveAll(tempDir)

	ingester, _, _ := defaultIngester(t, tempDir)
	wal := ingester.store.WAL()

	i, err := newInstance("fake", limiter, wal)
	assert.NoError(t, err, "unexpected error creating new instance")

	type push struct {
		req          *tempopb.PushRequest
		expectsError bool
	}

	tests := []struct {
		name   string
		pushes []push
	}{
		{
			name: "succeeds",
			pushes: []push{
				{
					req: test.MakeRequest(3, []byte{}),
				},
				{
					req: test.MakeRequest(5, []byte{}),
				},
				{
					req: test.MakeRequest(9, []byte{}),
				},
			},
		},
		{
			name: "one fails",
			pushes: []push{
				{
					req: test.MakeRequest(3, []byte{}),
				},
				{
					req:          test.MakeRequest(15, []byte{}),
					expectsError: true,
				},
				{
					req: test.MakeRequest(9, []byte{}),
				},
			},
		},
		{
			name: "multiple pushes same trace",
			pushes: []push{
				{
					req: test.MakeRequest(5, []byte{0x01}),
				},
				{
					req:          test.MakeRequest(7, []byte{0x01}),
					expectsError: true,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for j, push := range tt.pushes {
				err := i.Push(context.Background(), push.req)

				assert.Equalf(t, push.expectsError, err != nil, "push %d failed", j)
			}
		})
	}
}
