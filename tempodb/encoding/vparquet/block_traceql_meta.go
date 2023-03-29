package vparquet

import (
	"context"
	"fmt"

	"github.com/grafana/tempo/pkg/parquetquery"
	pq "github.com/grafana/tempo/pkg/parquetquery"
	"github.com/grafana/tempo/pkg/traceql"
	"github.com/grafana/tempo/pkg/util"
)

func createSpansetMetaIterator(makeIter makeIterFn, ss *spansetIterator, spanStartEndRetreived bool) (*spansetMetadataIterator, error) {
	// span level iterator
	iters := make([]parquetquery.Iterator, 0, 4)
	iters = append(iters, &spansToMetaIterator{ss})
	if !spanStartEndRetreived {
		iters = append(iters, makeIter(columnPathSpanStartTime, nil, columnPathSpanStartTime))
		iters = append(iters, makeIter(columnPathSpanEndTime, nil, columnPathSpanEndTime))
	}
	iters = append(iters, makeIter(columnPathSpanID, nil, columnPathSpanID))
	spanIterator := parquetquery.NewJoinIterator(DefinitionLevelResourceSpansILSSpan, iters, &spanMetaCollector{})

	// trace level iterator
	traceIters := []parquetquery.Iterator{
		spanIterator,
		// Add static columns that are always return
		makeIter(columnPathTraceID, nil, columnPathTraceID),
		makeIter(columnPathStartTimeUnixNano, nil, columnPathStartTimeUnixNano),
		makeIter(columnPathDurationNanos, nil, columnPathDurationNanos),
		makeIter(columnPathRootSpanName, nil, columnPathRootSpanName),
		makeIter(columnPathRootServiceName, nil, columnPathRootServiceName),
	}
	traceIter := parquetquery.NewJoinIterator(DefinitionLevelTrace, traceIters, &traceMetaCollector{})

	return newSpansetMetadataIterator(traceIter), nil
}

// spansToMetaIterator operates similarly to the rowNumberIterator except it takes a spanIterator
// and drains it. It is the bridge between the "data" iterators and "metadata" iterators
type spansToMetaIterator struct {
	iter *spansetIterator
}

var _ pq.Iterator = (*spansToMetaIterator)(nil)

func (i *spansToMetaIterator) String() string {
	return fmt.Sprintf("spansToMetaIterator: \n\t%s", util.TabOut(i.iter))
}

func (i *spansToMetaIterator) Next() (*pq.IteratorResult, error) {
	// now go to our iterator
	next, err := i.iter.Next()
	if err != nil {
		return nil, err
	}
	if next == nil {
		return nil, nil
	}

	res := &pq.IteratorResult{RowNumber: next.rowNum}
	res.AppendOtherValue(otherEntrySpanKey, next)

	return res, nil
}

func (i *spansToMetaIterator) SeekTo(to pq.RowNumber, definitionLevel int) (*pq.IteratorResult, error) {
	var at *pq.IteratorResult

	for at, _ = i.Next(); i != nil && at != nil && pq.CompareRowNumbers(definitionLevel, at.RowNumber, to) < 0; {
		at, _ = i.Next()
	}

	return at, nil
}

func (i *spansToMetaIterator) Close() {
	i.iter.Close()
}

// spanMetaCollector collects iterator results with the expectation that they were created
// using the iterators defined above
type spanMetaCollector struct {
}

var _ parquetquery.GroupPredicate = (*spanMetaCollector)(nil)

func (c *spanMetaCollector) String() string {
	return "spanMetaCollector()"
}

func (c *spanMetaCollector) KeepGroup(res *parquetquery.IteratorResult) bool {
	// extract the span from the iterator result and steal it's attributes
	// this is where we convert a traceql.Span to a traceql.SpanMetadata
	span, ok := res.OtherValueFromKey(otherEntrySpanKey).(*span)
	if !ok {
		return false // something very wrong happened. should we panic?
	}

	// span start/end time may come from span attributes or it may come from
	// the iterator results. if we find it in the iterator results, use that
	for _, kv := range res.Entries {
		switch kv.Key {
		case columnPathSpanID:
			span.id = kv.Value.ByteArray()
		case columnPathSpanStartTime:
			span.startTimeUnixNanos = kv.Value.Uint64()
		case columnPathSpanEndTime:
			span.endtimeUnixNanos = kv.Value.Uint64()
		}
	}

	res.Entries = res.Entries[:0]
	res.OtherEntries = res.OtherEntries[:0]
	res.AppendOtherValue(otherEntrySpanKey, span)

	return true
}

// traceMetaCollector receives rows from the resource-level matches.
// It adds trace-level attributes into the spansets before
// they are returned
type traceMetaCollector struct {
}

var _ parquetquery.GroupPredicate = (*traceMetaCollector)(nil)

func (c *traceMetaCollector) String() string {
	return "traceMetaCollector{}"
}

func (c *traceMetaCollector) KeepGroup(res *parquetquery.IteratorResult) bool {
	finalSpanset := &traceql.Spanset{}

	for _, e := range res.Entries {
		switch e.Key {
		case columnPathTraceID:
			finalSpanset.TraceID = e.Value.ByteArray()
		case columnPathStartTimeUnixNano:
			finalSpanset.StartTimeUnixNanos = e.Value.Uint64()
		case columnPathDurationNanos:
			finalSpanset.DurationNanos = e.Value.Uint64()
		case columnPathRootSpanName:
			finalSpanset.RootSpanName = e.Value.String()
		case columnPathRootServiceName:
			finalSpanset.RootServiceName = e.Value.String()
		}
	}

	// we're copying spans directly from the spanMetaIterator into the traceMetaIterator
	//  this skips the step of the batchIterator present on the normal fetch path
	for _, e := range res.OtherEntries {
		if span, ok := e.Value.(*span); ok {
			finalSpanset.Spans = append(finalSpanset.Spans, span)
		}
	}

	res.Entries = res.Entries[:0]
	res.OtherEntries = res.OtherEntries[:0]
	res.AppendOtherValue(otherEntrySpansetKey, finalSpanset)

	return true
}

// spansetMetadataIterator turns the parquet iterator into the final
// traceql iterator.  Every row it receives is one spanset.
type spansetMetadataIterator struct {
	iter parquetquery.Iterator
}

var _ traceql.SpansetIterator = (*spansetMetadataIterator)(nil)

func newSpansetMetadataIterator(iter parquetquery.Iterator) *spansetMetadataIterator {
	return &spansetMetadataIterator{
		iter: iter,
	}
}

func (i *spansetMetadataIterator) Next(ctx context.Context) (*traceql.Spanset, error) {
	res, err := i.iter.Next()
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, nil
	}

	// The spanset is in the OtherEntries
	iface := res.OtherValueFromKey(otherEntrySpansetKey)
	if iface == nil {
		return nil, fmt.Errorf("engine assumption broken: spanset not found in other entries")
	}
	ss, ok := iface.(*traceql.Spanset)
	if !ok {
		return nil, fmt.Errorf("engine assumption broken: spanset is not of type *traceql.Spanset")
	}

	return ss, nil
}

func (i *spansetMetadataIterator) Close() {
	i.iter.Close()
}
