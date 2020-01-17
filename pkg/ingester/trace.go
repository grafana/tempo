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
	batches    []*friggpb.PushRequest
	fp         traceFingerprint
	lastAppend time.Time
}

func newTrace(fp traceFingerprint) *trace {
	return &trace{
		fp:         fp,
		batches:    []*friggpb.PushRequest{},
		lastAppend: time.Now(),
	}
}

func (t *trace) Push(_ context.Context, req *friggpb.PushRequest) error {
	t.batches = append(t.batches, req)
	t.lastAppend = time.Now()

	return nil
}
