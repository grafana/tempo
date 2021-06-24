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
	traceBytes   *tempopb.TraceBytes
	lastAppend   time.Time
	traceID      []byte
	maxBytes     int
	currentBytes int

	// List of flatbuffers
	searchData [][]byte
}

func newTrace(maxBytes int, traceID []byte) *trace {
	return &trace{
		traceBytes: &tempopb.TraceBytes{
			Traces: make([][]byte, 0, 10), // 10 for luck
		},
		lastAppend: time.Now(),
		traceID:    traceID,
		maxBytes:   maxBytes,
	}
}

func (t *trace) Push(_ context.Context, trace []byte, searchData []byte) error {
	t.lastAppend = time.Now()
	if t.maxBytes != 0 {
		reqSize := len(trace)
		if t.currentBytes+reqSize > t.maxBytes {
			return status.Errorf(codes.FailedPrecondition, "%s max size of trace (%d) exceeded while adding %d bytes", overrides.ErrorPrefixTraceTooLarge, t.maxBytes, reqSize)
		}

		t.currentBytes += reqSize
	}

	t.traceBytes.Traces = append(t.traceBytes.Traces, trace)

	t.searchData = append(t.searchData, searchData)

	return nil
}

/*func (t *trace) pushHeader(trace []byte) {
	// Unmarshal data so we can process it
	x := &tempopb.Trace{}
	proto.Unmarshal(trace, x)

	for _, b := range x.Batches {
		for _, i := range b.InstrumentationLibrarySpans {
			for _, s := range i.Spans {

			}
		}
	}
}
*/
