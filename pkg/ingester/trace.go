package ingester

import (
	"context"
	"time"

	"github.com/grafana/tempo/pkg/tempopb"
)

type trace struct {
	trace      *tempopb.Trace
	fp         traceFingerprint
	lastAppend time.Time
	traceID    []byte
}

func newTrace(fp traceFingerprint, traceID []byte) *trace {
	return &trace{
		fp:         fp,
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
