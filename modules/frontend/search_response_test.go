package frontend

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/stretchr/testify/assert"
)

func TestSearchResponseShouldQuit(t *testing.T) {
	ctx := context.Background()

	cancelFunc := func() {}

	// brand-new response should not quit
	sr := newSearchResponse(ctx, 10, cancelFunc)
	assert.False(t, sr.shouldQuit())

	// errored response should quit
	sr = newSearchResponse(ctx, 10, cancelFunc)
	sr.setError(errors.New("blerg"))
	assert.True(t, sr.shouldQuit())

	// happy status code should not quit
	sr = newSearchResponse(ctx, 10, cancelFunc)
	sr.setStatus(200, "")
	assert.False(t, sr.shouldQuit())

	// sad status code should quit
	sr = newSearchResponse(ctx, 10, cancelFunc)
	sr.setStatus(400, "")
	assert.True(t, sr.shouldQuit())

	sr = newSearchResponse(ctx, 10, cancelFunc)
	sr.setStatus(500, "")
	assert.True(t, sr.shouldQuit())

	// cancelled context should quit
	cancellableContext, cancel := context.WithCancel(ctx)
	sr = newSearchResponse(cancellableContext, 10, cancelFunc)
	cancel()
	assert.True(t, sr.shouldQuit())

	// limit reached should quit
	sr = newSearchResponse(ctx, 2, cancelFunc)
	sr.addResponse(&tempopb.SearchResponse{
		Traces: []*tempopb.TraceSearchMetadata{
			{
				TraceID: "something",
			},
		},
		Metrics: &tempopb.SearchMetrics{},
	})
	assert.False(t, sr.shouldQuit())
	sr.addResponse(&tempopb.SearchResponse{
		Traces: []*tempopb.TraceSearchMetadata{
			{
				TraceID: "something",
			},
			{
				TraceID: "something",
			},
		},
		Metrics: &tempopb.SearchMetrics{},
	})
	assert.False(t, sr.shouldQuit())
	sr.addResponse(&tempopb.SearchResponse{
		Traces: []*tempopb.TraceSearchMetadata{
			{
				TraceID: "otherthing",
			},
			{
				TraceID: "thingthatsdifferent",
			},
		},
		Metrics: &tempopb.SearchMetrics{},
	})
	assert.True(t, sr.shouldQuit())
}

func TestCancelFuncEvents(t *testing.T) {
	ctx := context.Background()

	called := false
	cancelFunc := func() {
		called = true
	}

	// setError should call cancelFunc
	sr := newSearchResponse(ctx, 1, cancelFunc)
	sr.setError(errors.New("an err"))
	assert.True(t, called)

	// setStatus with 200 should not call cancelFunc
	called = false
	sr = newSearchResponse(ctx, 1, cancelFunc)
	sr.setStatus(200, "OK")
	assert.False(t, called) // cancelFunc should NOT be called

	// setStatus with 500 should call cancelFunc
	called = false
	sr = newSearchResponse(ctx, 1, cancelFunc)
	sr.setStatus(500, "")
	assert.True(t, true)

	// setStatus with 500 should call cancelFunc
	called = false
	sr = newSearchResponse(ctx, 1, cancelFunc)
	sr.setStatus(500, "")
	assert.True(t, true)

	// shouldQuit true should call cancelFunc
	// AND internalShouldQuit should NOT call cancelFunc
	called = false
	sr = newSearchResponse(ctx, 1, cancelFunc)
	// directly set statusCode because setStatus also calls cancelFunc
	sr.statusCode = 500
	assert.True(t, sr.internalShouldQuit())
	assert.False(t, called) // cancelFunc is not called till we call shouldQuit
	assert.True(t, sr.shouldQuit())
	assert.True(t, called) // shouldQuit calls cancelFunc

	// addResponse within limit should NOT call cancelFunc
	called = false
	sr = newSearchResponse(ctx, 100, cancelFunc)
	sr.addResponse(&tempopb.SearchResponse{
		Traces: []*tempopb.TraceSearchMetadata{
			{
				TraceID: "something",
			},
		},
		Metrics: &tempopb.SearchMetrics{},
	})
	assert.False(t, sr.shouldQuit())
	assert.False(t, called)

	// addResponse over limit should call cancelFunc
	called = false
	sr = newSearchResponse(ctx, 1, cancelFunc)
	sr.addResponse(&tempopb.SearchResponse{
		Traces: []*tempopb.TraceSearchMetadata{
			{
				TraceID: "something",
			},
		},
		Metrics: &tempopb.SearchMetrics{},
	})
	assert.False(t, sr.internalShouldQuit()) // internalShouldQuit should return false
	assert.False(t, called)                  // first response should not call cancelFunc

	sr.addResponse(&tempopb.SearchResponse{
		Traces: []*tempopb.TraceSearchMetadata{
			{
				TraceID: "something else", // needs unique traceID to hit limits
			},
		},
		Metrics: &tempopb.SearchMetrics{},
	})
	assert.True(t, sr.internalShouldQuit()) // internalShouldQuit should return true
	assert.True(t, called)                  // addResponse calls cancelFunc (without calling shouldQuit)
}

func TestSearchResponseCombineResults(t *testing.T) {
	start := time.Date(1, 2, 3, 4, 5, 6, 7, time.UTC)
	traceID := "traceID"

	sr := newSearchResponse(context.Background(), 10, func() {})
	sr.addResponse(&tempopb.SearchResponse{
		Traces: []*tempopb.TraceSearchMetadata{
			{
				TraceID:           traceID,
				StartTimeUnixNano: uint64(start.Add(time.Second).UnixNano()),
				DurationMs:        uint32(time.Second.Milliseconds()),
			}, // 1 second after start and shorter duration
			{
				TraceID:           traceID,
				StartTimeUnixNano: uint64(start.UnixNano()),
				DurationMs:        uint32(time.Hour.Milliseconds()),
			}, // earliest start time and longer duration
			{
				TraceID:           traceID,
				StartTimeUnixNano: uint64(start.Add(time.Hour).UnixNano()),
				DurationMs:        uint32(time.Millisecond.Milliseconds()),
			}, // 1 hour after start and shorter duration
		},
		Metrics: &tempopb.SearchMetrics{},
	})

	expected := &tempopb.SearchResponse{
		Traces: []*tempopb.TraceSearchMetadata{
			{TraceID: traceID, StartTimeUnixNano: uint64(start.UnixNano()), DurationMs: uint32(time.Hour.Milliseconds())},
		},
		Metrics: &tempopb.SearchMetrics{},
	}

	assert.Equal(t, expected, sr.result())

}
