package ingester

import (
	"context"
	"encoding/binary"
	"io/ioutil"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/grafana/tempo/tempodb/wal"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	err = i.CutBlockIfReady(0, 0, 0, false)
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
	assert.Nil(t, i.completingBlock)
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

	err = i.CutBlockIfReady(0, 0, 0, false)
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
		_ = i.CutBlockIfReady(0, 0, 0, false)
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

func TestInstanceCutCompleteTraces(t *testing.T) {
	tempDir, _ := ioutil.TempDir("/tmp", "")
	defer os.RemoveAll(tempDir)

	id := make([]byte, 16)
	rand.Read(id)
	tracepb := test.MakeTrace(10, id)
	pastTrace := &trace{
		traceID:    id,
		trace:      tracepb,
		lastAppend: time.Now().Add(-time.Hour),
	}

	id = make([]byte, 16)
	rand.Read(id)
	nowTrace := &trace{
		traceID:    id,
		trace:      tracepb,
		lastAppend: time.Now().Add(time.Hour),
	}

	tt := []struct {
		name             string
		cutoff           time.Duration
		immediate        bool
		input            []*trace
		expectedExist    []*trace
		expectedNotExist []*trace
	}{
		{
			name:      "empty",
			cutoff:    0,
			immediate: false,
		},
		{
			name:             "cut immediate",
			cutoff:           0,
			immediate:        true,
			input:            []*trace{pastTrace, nowTrace},
			expectedNotExist: []*trace{pastTrace, nowTrace},
		},
		{
			name:             "cut recent",
			cutoff:           0,
			immediate:        false,
			input:            []*trace{pastTrace, nowTrace},
			expectedExist:    []*trace{nowTrace},
			expectedNotExist: []*trace{pastTrace},
		},
		{
			name:             "cut all time",
			cutoff:           2 * time.Hour,
			immediate:        false,
			input:            []*trace{pastTrace, nowTrace},
			expectedNotExist: []*trace{pastTrace, nowTrace},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			instance := defaultInstance(t, tempDir)

			for _, trace := range tc.input {
				fp := instance.tokenForTraceID(trace.traceID)
				instance.traces[fp] = trace
			}

			err := instance.CutCompleteTraces(tc.cutoff, tc.immediate)
			require.NoError(t, err)

			assert.Equal(t, len(tc.expectedExist), len(instance.traces))
			for _, expectedExist := range tc.expectedExist {
				_, ok := instance.traces[instance.tokenForTraceID(expectedExist.traceID)]
				assert.True(t, ok)
			}

			for _, expectedNotExist := range tc.expectedNotExist {
				_, ok := instance.traces[instance.tokenForTraceID(expectedNotExist.traceID)]
				assert.False(t, ok)
			}
		})
	}
}

func TestInstanceCutBlockIfReady(t *testing.T) {
	tempDir, _ := ioutil.TempDir("/tmp", "")
	defer os.RemoveAll(tempDir)

	tt := []struct {
		name               string
		maxTracesPerBlock  int
		maxBlockLifetime   time.Duration
		maxBlockBytes      uint64
		immediate          bool
		pushCount          int
		expectedToCutBlock bool
	}{
		{
			name:               "empty",
			expectedToCutBlock: false,
		},
		{
			name:               "doesnt cut anything",
			pushCount:          1,
			expectedToCutBlock: false,
		},
		{
			name:               "cut immediate",
			immediate:          true,
			pushCount:          1,
			expectedToCutBlock: true,
		},
		{
			name:               "cut based on trace count",
			maxTracesPerBlock:  1,
			pushCount:          2,
			expectedToCutBlock: true,
		},
		{
			name:               "cut based on block lifetime",
			maxBlockLifetime:   time.Microsecond,
			pushCount:          1,
			expectedToCutBlock: true,
		},
		{
			name:               "cut based on block size",
			maxBlockBytes:      10,
			pushCount:          10,
			expectedToCutBlock: true,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			instance := defaultInstance(t, tempDir)

			for i := 0; i < tc.pushCount; i++ {
				request := test.MakeRequest(10, []byte{})
				err := instance.Push(context.Background(), request)
				require.NoError(t, err)
			}

			// Defaults
			if tc.maxBlockBytes == 0 {
				tc.maxBlockBytes = 1000
			}
			if tc.maxBlockLifetime == 0 {
				tc.maxBlockLifetime = time.Hour
			}
			if tc.maxTracesPerBlock == 0 {
				tc.maxTracesPerBlock = 1000
			}

			lastCutTime := instance.lastBlockCut

			// Cut all traces to headblock for testing
			err := instance.CutCompleteTraces(0, true)
			require.NoError(t, err)

			err = instance.CutBlockIfReady(tc.maxTracesPerBlock, tc.maxBlockLifetime, tc.maxBlockBytes, tc.immediate)
			require.NoError(t, err)

			// Wait for goroutine to finish flushing to avoid test flakiness
			if tc.expectedToCutBlock {
				time.Sleep(time.Millisecond * 250)
			}

			assert.Equal(t, tc.expectedToCutBlock, instance.lastBlockCut.After(lastCutTime))
		})
	}
}

func defaultInstance(t assert.TestingT, tempDir string) *instance {
	limits, err := overrides.NewOverrides(overrides.Limits{})
	assert.NoError(t, err, "unexpected error creating limits")
	limiter := NewLimiter(limits, &ringCountMock{count: 1}, 1)

	wal, err := wal.New(&wal.Config{
		Filepath:        tempDir,
		IndexDownsample: 2,
		BloomFP:         .01,
	})
	assert.NoError(t, err, "unexpected error creating wal")

	instance, err := newInstance("fake", limiter, wal)
	assert.NoError(t, err, "unexpected error creating new instance")

	return instance
}

func BenchmarkInstancePush(b *testing.B) {
	tempDir, err := ioutil.TempDir("/tmp", "")
	assert.NoError(b, err, "unexpected error getting temp dir")
	defer os.RemoveAll(tempDir)

	instance := defaultInstance(b, tempDir)
	request := test.MakeRequest(10, []byte{})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Rotate trace ID
		binary.LittleEndian.PutUint32(request.Batch.InstrumentationLibrarySpans[0].Spans[0].TraceId, uint32(i))
		err = instance.Push(context.Background(), request)
		assert.NoError(b, err)
	}
}

func BenchmarkInstancePushExistingTrace(b *testing.B) {
	tempDir, err := ioutil.TempDir("/tmp", "")
	assert.NoError(b, err, "unexpected error getting temp dir")
	defer os.RemoveAll(tempDir)

	instance := defaultInstance(b, tempDir)
	request := test.MakeRequest(10, []byte{})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err = instance.Push(context.Background(), request)
		assert.NoError(b, err)
	}
}

func BenchmarkInstanceFindTraceByID(b *testing.B) {
	tempDir, err := ioutil.TempDir("/tmp", "")
	assert.NoError(b, err, "unexpected error getting temp dir")
	defer os.RemoveAll(tempDir)

	instance := defaultInstance(b, tempDir)
	traceID := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	request := test.MakeRequest(10, traceID)
	err = instance.Push(context.Background(), request)
	assert.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		trace, err := instance.FindTraceByID(traceID)
		assert.NotNil(b, trace)
		assert.NoError(b, err)
	}
}
