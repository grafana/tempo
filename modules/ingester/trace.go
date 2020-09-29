package ingester

import (
	"context"
	"time"

	"github.com/grafana/tempo/pkg/tempopb"
)

type trace struct {
	trace      *tempopb.Trace
	token      uint32
	lastAppend time.Time
	traceID    []byte
}

func newTrace(token uint32, traceID []byte) *trace {
	return &trace{
		token:      token,
		trace:      &tempopb.Trace{},
		lastAppend: time.Now(),
		traceID:    traceID,
	}
}

func (t *trace) Push(_ context.Context, req *tempopb.PushRequest) error {
	t.trace.Batches = append(t.trace.Batches, req.Batch)
	t.lastAppend = time.Now()

	return nil
}
