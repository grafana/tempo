package ingester

import (
	"context"
	"time"

	"github.com/joe-elliott/frigg/pkg/friggpb"
)

var ()

func init() {

}

type trace struct {
	trace      *friggpb.Trace
	fp         traceFingerprint
	lastAppend time.Time
	traceID    []byte
}

func newTrace(fp traceFingerprint, traceID []byte) *trace {
	return &trace{
		fp:         fp,
		trace:      &friggpb.Trace{},
		lastAppend: time.Now(),
		traceID:    traceID,
	}
}

func (t *trace) Push(_ context.Context, req *friggpb.PushRequest) error {
	t.trace.Batches = append(t.trace.Batches, req)
	t.lastAppend = time.Now()

	return nil
}
