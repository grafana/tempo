package common

import (
	"context"

	"github.com/grafana/tempo/pkg/parquetquery"
	"github.com/grafana/tempo/pkg/traceql"
)

// ScanBackend is implemented by each storage encoding (e.g., vparquet5).
// Each method maps 1:1 to a parquet iterator level and produces the iterator
// for that level using the given plan node's conditions.
//
// The child parameter carries the iterator from the level immediately below
// (nil for the innermost level). Returning nil from a method means that level
// is skipped (no iterator needed for the current conditions).
//
// TraceIter converts the full parquet iterator chain into a spanset-level
// iterator. primary is the second-pass row source — when non-nil, surviving
// span row numbers from the first pass are re-fed through the scan to fetch
// additional columns declared in a ProjectNode.
type ScanBackend interface {
	SpanIter(
		ctx context.Context,
		node *traceql.SpanScanNode,
		child parquetquery.Iterator,
	) (parquetquery.Iterator, error)

	InstrumentationScopeIter(
		ctx context.Context,
		node *traceql.InstrumentationScopeScanNode,
		child parquetquery.Iterator,
	) (parquetquery.Iterator, error)

	ResourceIter(
		ctx context.Context,
		node *traceql.ResourceScanNode,
		child parquetquery.Iterator,
	) (parquetquery.Iterator, error)

	// TraceIter converts the parquet iterator chain to a spanset-level iterator.
	// primary: if non-nil, used as the second-pass row source (surviving span row numbers).
	TraceIter(
		ctx context.Context,
		node *traceql.TraceScanNode,
		primary parquetquery.Iterator,
		child parquetquery.Iterator,
	) (traceql.SpansetIterator, error)

	EventIter(
		ctx context.Context,
		node *traceql.EventScanNode,
		child parquetquery.Iterator,
	) (parquetquery.Iterator, error)

	LinkIter(
		ctx context.Context,
		node *traceql.LinkScanNode,
		child parquetquery.Iterator,
	) (parquetquery.Iterator, error)

	// SpanMerger returns a function that merges span-level data from a fetch-side
	// Spanset into the matching driving-side spans (matched by parquet row number).
	// Called by lateMaterializeIter after trace-level metadata is merged.
	// Backends that do not need span-level merging may return nil.
	SpanMerger() func(dst, src *traceql.Spanset)

	// TraceIterRaw is like TraceIter but returns the raw parquetquery.Iterator
	// without wrapping in SpansetIterator. Used by the fetch side of
	// ProjectNode where SeekTo capability is needed.
	TraceIterRaw(
		ctx context.Context,
		node *traceql.TraceScanNode,
		primary parquetquery.Iterator,
		child parquetquery.Iterator,
	) (parquetquery.Iterator, error)
}
