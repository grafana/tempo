package common

import (
	"context"

	"github.com/grafana/tempo/pkg/parquetquery"
	"github.com/grafana/tempo/pkg/traceql"
)

// spansetOtherEntryKey is the OtherEntries key used by vparquet5's traceCollector
// to store a completed *traceql.Spanset in a parquetquery.IteratorResult.
const spansetOtherEntryKey = "spanset"

// lateMaterializeIter wraps a driving SpansetIterator and a fetch
// parquetquery.Iterator. For each spanset from the driving side it seeks
// the fetch iterator to that trace's row and merges the fetched spanset's
// metadata (TraceID, RootServiceName, etc.) into the driving spanset.
//
// The fetch iterator is built once during translation from a TraceScanNode
// that carries OpNone conditions for search metadata columns. Each result
// produced by the fetch iterator has a *traceql.Spanset in OtherEntries
// (placed there by the vparquet5 traceCollector) with the metadata fields
// already populated.
//
// definitionLevel must be 0 (trace level): SeekTo advances one trace at a
// time, and all spans within a spanset share the same trace row number
// rn[0], so we only need one seek per spanset.
type lateMaterializeIter struct {
	driving         traceql.SpansetIterator
	fetcher         parquetquery.Iterator
	definitionLevel int
}

func newLateMaterializeIter(
	driving traceql.SpansetIterator,
	fetcher parquetquery.Iterator,
	definitionLevel int,
) *lateMaterializeIter {
	return &lateMaterializeIter{
		driving:         driving,
		fetcher:         fetcher,
		definitionLevel: definitionLevel,
	}
}

func (it *lateMaterializeIter) Next(ctx context.Context) (*traceql.Spanset, error) {
	ss, err := it.driving.Next(ctx)
	if ss == nil || err != nil {
		return ss, err
	}

	if len(ss.Spans) == 0 {
		return ss, nil
	}

	// All spans in a spanset share the same trace row index rn[0].
	// Seek the fetch iterator to this trace. Because driving and fetch
	// both iterate in parquet row order, SeekTo always moves forward.
	rn := ss.Spans[0].RowNum()
	if !rn.Valid() {
		return ss, nil
	}

	res, err := it.fetcher.SeekTo(rn, it.definitionLevel)
	if err != nil {
		return nil, err
	}
	if res == nil {
		return ss, nil
	}

	// Verify the fetched result is for the same trace row.
	if parquetquery.EqualRowNumber(it.definitionLevel, res.RowNumber, rn) {
		mergeSpansetMetadata(ss, res)
	}

	return ss, nil
}

func (it *lateMaterializeIter) Close() {
	it.driving.Close()
	it.fetcher.Close()
}

// mergeSpansetMetadata copies trace-level metadata from the fetched
// parquetquery.IteratorResult into the driving spanset. The fetch iterator
// is built from a TraceScanNode, so its traceCollector stores a complete
// *traceql.Spanset in OtherEntries under spansetOtherEntryKey.
func mergeSpansetMetadata(dst *traceql.Spanset, res *parquetquery.IteratorResult) {
	iface := res.OtherValueFromKey(spansetOtherEntryKey)
	if iface == nil {
		return
	}
	src, ok := iface.(*traceql.Spanset)
	if !ok {
		return
	}
	if len(dst.TraceID) == 0 {
		dst.TraceID = src.TraceID
	}
	if len(dst.RootServiceName) == 0 {
		dst.RootServiceName = src.RootServiceName
	}
	if len(dst.RootSpanName) == 0 {
		dst.RootSpanName = src.RootSpanName
	}
	if dst.StartTimeUnixNanos == 0 {
		dst.StartTimeUnixNanos = src.StartTimeUnixNanos
	}
	if dst.DurationNanos == 0 {
		dst.DurationNanos = src.DurationNanos
	}
	if len(dst.ServiceStats) == 0 && len(src.ServiceStats) > 0 {
		dst.ServiceStats = src.ServiceStats
	}
}
