package ingester

import (
	"context"
	"time"

	cortex_util "github.com/cortexproject/cortex/pkg/util/log"
	"github.com/go-kit/kit/log/level"
	"github.com/gogo/status"
	"google.golang.org/grpc/codes"

	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/tempopb"
)

type trace struct {
	traceBytes   *tempopb.TraceBytes
	lastAppend   time.Time
	traceID      []byte
	maxBytes     int
	currentBytes int

	// List of flatbuffers
	searchData         [][]byte
	maxSearchBytes     int
	currentSearchBytes int
}

func newTrace(traceID []byte, maxBytes int, maxSearchBytes int) *trace {
	return &trace{
		traceBytes: &tempopb.TraceBytes{
			Traces: make([][]byte, 0, 10), // 10 for luck
		},
		lastAppend:     time.Now(),
		traceID:        traceID,
		maxBytes:       maxBytes,
		maxSearchBytes: maxSearchBytes,
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

	if searchData != nil {
		searchDataSize := len(searchData)
		if t.maxSearchBytes != 0 { // disable limit if set to 0
			if t.currentSearchBytes+searchDataSize > t.maxSearchBytes {
				// todo: info level since we are not expecting this limit to be hit, but calibrate accordingly in the future
				level.Info(cortex_util.Logger).Log("msg", "size of search data exceeded max search bytes limit", "maxSearchBytes", t.maxSearchBytes)
				return nil
			}
			t.currentSearchBytes += searchDataSize
		}
		t.searchData = append(t.searchData, searchData)
	}

	return nil
}
