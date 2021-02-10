package ingester

import (
	"context"
	"time"

	"github.com/gogo/status"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/tempopb"
	"google.golang.org/grpc/codes"
)

type trace struct {
	trace        *tempopb.Trace
	token        uint32
	lastAppend   time.Time
	traceID      []byte
	maxSpans     int
	currentSpans int
}

func newTrace(maxSpans int, token uint32, traceID []byte) *trace {
	return &trace{
		token:      token,
		trace:      &tempopb.Trace{},
		lastAppend: time.Now(),
		traceID:    traceID,
		maxSpans:   maxSpans,
	}
}

func (t *trace) Push(_ context.Context, req *tempopb.PushRequest) error {
	if t.maxSpans != 0 {
		// count spans
		spanCount := 0
		for _, ils := range req.Batch.InstrumentationLibrarySpans {
			spanCount += len(ils.Spans)
		}

		if t.currentSpans+spanCount > t.maxSpans {
			return status.Errorf(codes.FailedPrecondition, "%s totalSpans (%d) exceeded while adding %d spans", overrides.ErrorPrefixTraceTooLarge, t.maxSpans, spanCount)
		}

		t.currentSpans += spanCount
	}

	t.trace.Batches = append(t.trace.Batches, req.Batch)
	t.lastAppend = time.Now()

	return nil
}
