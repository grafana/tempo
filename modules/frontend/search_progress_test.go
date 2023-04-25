package frontend

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/stretchr/testify/assert"
)

func TestSearchProgressShouldQuit(t *testing.T) {
	ctx := context.Background()

	// brand-new response should not quit
	sr := newSearchProgress(ctx, 10, 0, 0, 0)
	assert.False(t, sr.shouldQuit())

	// errored response should quit
	sr = newSearchProgress(ctx, 10, 0, 0, 0)
	sr.setError(errors.New("blerg"))
	assert.True(t, sr.shouldQuit())

	// happy status code should not quit
	sr = newSearchProgress(ctx, 10, 0, 0, 0)
	sr.setStatus(200, "")
	assert.False(t, sr.shouldQuit())

	// sad status code should quit
	sr = newSearchProgress(ctx, 10, 0, 0, 0)
	sr.setStatus(400, "")
	assert.True(t, sr.shouldQuit())

	sr = newSearchProgress(ctx, 10, 0, 0, 0)
	sr.setStatus(500, "")
	assert.True(t, sr.shouldQuit())

	// cancelled context should quit
	cancellableContext, cancel := context.WithCancel(ctx)
	sr = newSearchProgress(cancellableContext, 10, 0, 0, 0)
	cancel()
	assert.True(t, sr.shouldQuit())

	// limit reached should quit
	sr = newSearchProgress(ctx, 2, 0, 0, 0)
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

func TestSearchProgressCombineResults(t *testing.T) {
	start := time.Date(1, 2, 3, 4, 5, 6, 7, time.UTC)
	traceID := "traceID"

	sr := newSearchProgress(context.Background(), 10, 0, 0, 0)
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

	expected := &shardedSearchResults{
		response: &tempopb.SearchResponse{
			Traces: []*tempopb.TraceSearchMetadata{
				{TraceID: traceID, StartTimeUnixNano: uint64(start.UnixNano()), DurationMs: uint32(time.Hour.Milliseconds())},
			},
			Metrics: &tempopb.SearchMetrics{
				CompletedJobs: 1,
			},
		},
		finishedRequests: 1,
		statusCode:       200,
	}

	assert.Equal(t, expected, sr.result())

}
