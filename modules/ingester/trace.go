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
	lastAppend   time.Time
	traceID      []byte
	maxBytes     int
	currentBytes int
}

func newTrace(maxBytes int, traceID []byte) *trace {
	return &trace{
		trace:      &tempopb.Trace{},
		lastAppend: time.Now(),
		traceID:    traceID,
		maxBytes:   maxBytes,
	}
}

func (t *trace) Push(_ context.Context, req *tempopb.PushRequest) error {
	t.lastAppend = time.Now()
	if t.maxBytes != 0 {
		reqSize := req.Size()
		if t.currentBytes+reqSize > t.maxBytes {
			return status.Errorf(codes.FailedPrecondition, "%s max size of trace (%d) exceeded while adding %d bytes", overrides.ErrorPrefixTraceTooLarge, t.maxBytes, reqSize)
		}

		t.currentBytes += reqSize
	}

	t.trace.Batches = append(t.trace.Batches, req.Batch)

	return nil
}
