package livestore

import (
	"context"

	"github.com/grafana/tempo/pkg/tempopb"
	trace_v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

// protoIterator implements common.Iterator over a slice of pendingTraces.
// It yields (ID, *tempopb.Trace) pairs, feeding into CreateBlock's existing
// proto-to-Parquet path which calls traceToParquetWithMapping() once per trace.
type protoIterator struct {
	traces []*pendingTrace
	idx    int
}

var _ common.Iterator = (*protoIterator)(nil)

func newProtoIterator(traces []*pendingTrace) *protoIterator {
	return &protoIterator{
		traces: traces,
	}
}

func (it *protoIterator) Next(_ context.Context) (common.ID, *tempopb.Trace, error) {
	if it.idx >= len(it.traces) {
		return nil, nil, nil
	}

	t := it.traces[it.idx]
	it.idx++

	trace := &tempopb.Trace{
		ResourceSpans: t.Batches,
	}

	return t.ID, trace, nil
}

func (it *protoIterator) Close() {
	it.traces = nil
}

// traceStartEndSeconds returns the start/end timestamps in seconds for a
// set of resource spans, used when calling CreateBlock.
func traceStartEndSeconds(batches []*trace_v1.ResourceSpans) (startSeconds, endSeconds uint32) {
	var startNanos, endNanos uint64
	for _, rs := range batches {
		for _, ss := range rs.ScopeSpans {
			for _, s := range ss.Spans {
				if startNanos == 0 || s.StartTimeUnixNano < startNanos {
					startNanos = s.StartTimeUnixNano
				}
				if s.EndTimeUnixNano > endNanos {
					endNanos = s.EndTimeUnixNano
				}
			}
		}
	}
	startSeconds = uint32(startNanos / 1_000_000_000)
	endSeconds = uint32(endNanos / 1_000_000_000)
	return
}
