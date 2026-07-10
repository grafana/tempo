package vparquet5

import (
	"context"
	"fmt"
	"io"

	pq "github.com/grafana/tempo/pkg/parquetquery"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

// traceIDIterator projects only the TraceID column of a vparquet5 block. It
// is built on the same openForSearch shape as Search/FindTraceByID (skip
// bloom filters + page index, async read mode), not the openForIteration
// shape rawIterator uses -- that decodes every column of every row and is
// tuned for compaction's sequential full-block scan, exactly the cost a
// trace-ID-only enumeration means to avoid.
type traceIDIterator struct {
	rr   *BackendReaderAt
	iter *pq.SyncIterator
}

var _ common.TraceIDIterator = (*traceIDIterator)(nil)

// openTraceIDReader opens a traceIDIterator over b. ctx is used both for
// the parquet-open span and, because BackendReaderAt captures it once at
// construction, for every backend read the iterator triggers lazily as
// pages are pulled during Next -- there is no deeper per-Next ctx plumbing
// available here. Next itself still checks ctx on every call so a caller
// cancelling between reads gets a prompt error rather than one more row.
func (b *backendBlock) openTraceIDReader(ctx context.Context) (*traceIDIterator, error) {
	pf, rr, err := b.openForSearch(ctx, common.DefaultSearchOptions())
	if err != nil {
		return nil, fmt.Errorf("unexpected error opening parquet file: %w", err)
	}

	colIndex, _, maxDef := pq.GetColumnIndexByPath(pf, TraceIDColumnName)
	if colIndex == -1 {
		return nil, fmt.Errorf("unable to get index for column: %s", TraceIDColumnName)
	}

	// No predicate: every row matches, the same "iterate the column, keep
	// everything" shape as makePipelineWithRowGroups' empty-request case
	// and searchSpecialTagValues. Deliberately no SyncIteratorOptIntern:
	// not recommended for a high-cardinality column like TraceID.
	iter := pq.NewSyncIterator(
		ctx, pf.RowGroups(), colIndex,
		pq.SyncIteratorOptColumnName(TraceIDColumnName),
		pq.SyncIteratorOptSelectAs(TraceIDColumnName),
		pq.SyncIteratorOptMaxDefinitionLevel(maxDef),
	)

	return &traceIDIterator{rr: rr, iter: iter}, nil
}

// Next implements common.TraceIDIterator.
func (t *traceIDIterator) Next(ctx context.Context) (common.ID, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	res, err := t.iter.Next()
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, io.EOF
	}

	for _, e := range res.Entries {
		if e.Key == TraceIDColumnName {
			// Already a detached Clone() for ByteArray columns (see
			// SyncIterator.makeResultClone) and already the canonical
			// 16-byte zero-padded form -- every write path pads via
			// util.PadTraceIDTo16Bytes before storing (schema.go). No
			// re-padding needed here.
			return common.ID(e.Value.ByteArray()), nil
		}
	}

	return nil, fmt.Errorf("trace ID projection: column %q missing from result", TraceIDColumnName)
}

// BytesRead reports the wire bytes read from the backend so far. Not part
// of common.TraceIDIterator; exposed for BenchmarkTraceIDProjection to
// re-derive DESIGN.md's per-block trace-ID-column sizing assumption.
func (t *traceIDIterator) BytesRead() uint64 {
	return t.rr.BytesRead()
}

// Close implements common.TraceIDIterator. Safe to call more than once.
func (t *traceIDIterator) Close() {
	if t.iter == nil {
		return
	}
	t.iter.Close()
	t.iter = nil
}
