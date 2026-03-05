package vparquet5

import (
	"context"

	"github.com/parquet-go/parquet-go"
	"github.com/grafana/tempo/pkg/parquetquery"
	"github.com/grafana/tempo/pkg/traceql"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

// blockScanBackend implements common.ScanBackend for a single vparquet5 block.
// It delegates to the existing create*Iterator functions, passing the correct
// makeIter/makeNilIter closures and dedicated-column metadata.
type blockScanBackend struct {
	makeIter         makeIterFn
	makeNilIter      makeIterFn
	dedicatedColumns backend.DedicatedColumns
}

var _ common.ScanBackend = (*blockScanBackend)(nil)

// NewBlockScanBackend creates a blockScanBackend for the given parquet file,
// row groups, and dedicated-column mapping.
func NewBlockScanBackend(ctx context.Context, pf *parquet.File, rgs []parquet.RowGroup, dc backend.DedicatedColumns) *blockScanBackend {
	return &blockScanBackend{
		makeIter:         makeIterFunc(ctx, rgs, pf),
		makeNilIter:      makeNilIterFunc(ctx, rgs, pf),
		dedicatedColumns: dc,
	}
}

// NewBlockScanBackendFromBlock opens the parquet file for a given backendBlock
// and returns a blockScanBackend ready to use.  The caller should close the
// returned BackendReaderAt when done to release resources.
func NewBlockScanBackendFromBlock(ctx context.Context, b *backendBlock, opts common.SearchOptions) (*blockScanBackend, *BackendReaderAt, error) {
	pf, rr, err := b.openForSearch(ctx, opts)
	if err != nil {
		return nil, nil, err
	}
	rgs := rowGroupsFromFile(pf, opts)
	return NewBlockScanBackend(ctx, pf, rgs, b.meta.DedicatedColumns), rr, nil
}

// SpanIter delegates to createSpanIterator with span-scope conditions from the node.
func (b *blockScanBackend) SpanIter(
	ctx context.Context,
	node *traceql.SpanScanNode,
	child parquetquery.Iterator,
) (parquetquery.Iterator, error) {
	var inner []parquetquery.Iterator
	if child != nil {
		inner = []parquetquery.Iterator{child}
	}
	return createSpanIterator(b.makeIter, b.makeNilIter, inner, node.Conditions, node.AllConditions, b.dedicatedColumns, false, nil)
}

// InstrumentationScopeIter delegates to createInstrumentationIterator.
func (b *blockScanBackend) InstrumentationScopeIter(
	ctx context.Context,
	node *traceql.InstrumentationScopeScanNode,
	child parquetquery.Iterator,
) (parquetquery.Iterator, error) {
	return createInstrumentationIterator(b.makeIter, b.makeNilIter, child, node.Conditions, false, false)
}

// ResourceIter delegates to createResourceIterator.
func (b *blockScanBackend) ResourceIter(
	ctx context.Context,
	node *traceql.ResourceScanNode,
	child parquetquery.Iterator,
) (parquetquery.Iterator, error) {
	// requireAtLeastOneMatchOverall is false — the TraceScanNode handles that.
	return createResourceIterator(b.makeIter, b.makeNilIter, child, node.Conditions, false, node.AllConditions, b.dedicatedColumns, false)
}

// TraceIter delegates to createTraceIterator and wraps the result in a spansetIterator.
// primary: if non-nil, it is the second-pass row source (surviving span row numbers).
func (b *blockScanBackend) TraceIter(
	ctx context.Context,
	node *traceql.TraceScanNode,
	primary parquetquery.Iterator,
	child parquetquery.Iterator,
) (traceql.SpansetIterator, error) {
	if primary != nil {
		// Second-pass path: primary drives the row selection.
		iter, err := createTraceIterator(b.makeIter, primary, node.Conditions, 0, 0, node.AllConditions, false, nil)
		if err != nil {
			return nil, err
		}
		return newSpansetIterator(newRebatchIterator(iter)), nil
	}

	// First-pass path: child is the resource-level iterator.
	iter, err := createTraceIterator(b.makeIter, child, node.Conditions, 0, 0, node.AllConditions, false, nil)
	if err != nil {
		return nil, err
	}
	return newSpansetIterator(newRebatchIterator(iter)), nil
}

// EventIter delegates to createEventIterator.
func (b *blockScanBackend) EventIter(
	ctx context.Context,
	node *traceql.EventScanNode,
	child parquetquery.Iterator,
) (parquetquery.Iterator, error) {
	return createEventIterator(b.makeIter, b.makeNilIter, node.Conditions, false, b.dedicatedColumns, false)
}

// OpenScanBackend implements common.ScanBackendOpener.
// It opens the parquet file for the block and returns a blockScanBackend ready to use.
// The caller must call the returned cleanup function when done to release file resources.
func (b *backendBlock) OpenScanBackend(ctx context.Context, opts common.SearchOptions) (common.ScanBackend, func(), error) {
	backend, rr, err := NewBlockScanBackendFromBlock(ctx, b, opts)
	if err != nil {
		return nil, nil, err
	}
	cleanup := func() { _ = rr }
	return backend, cleanup, nil
}

// TraceIterRaw is like TraceIter but returns the raw parquetquery.Iterator
// without wrapping in SpansetIterator. Used by the fetch side of ProjectNode.
func (b *blockScanBackend) TraceIterRaw(
	ctx context.Context,
	node *traceql.TraceScanNode,
	primary parquetquery.Iterator,
	child parquetquery.Iterator,
) (parquetquery.Iterator, error) {
	source := child
	if primary != nil {
		source = primary
	}
	return createTraceIterator(b.makeIter, source, node.Conditions, 0, 0, node.AllConditions, false, nil)
}

// LinkIter delegates to createLinkIterator.
func (b *blockScanBackend) LinkIter(
	ctx context.Context,
	node *traceql.LinkScanNode,
	child parquetquery.Iterator,
) (parquetquery.Iterator, error) {
	return createLinkIterator(b.makeIter, b.makeNilIter, node.Conditions, false, false)
}
