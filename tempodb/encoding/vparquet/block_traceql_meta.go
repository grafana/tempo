package vparquet

import (
	"context"
	"time"

	"github.com/grafana/tempo/pkg/parquetquery"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/traceql"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/segmentio/parquet-go"
)

// FetchMetadata for the given spansets from the block.
func (b *backendBlock) FetchMetadata(ctx context.Context, ss []traceql.Spanset, opts common.SearchOptions) ([]traceql.SpansetMetadata, error) {
	pf, _, err := b.openForSearch(ctx, opts)
	if err != nil {
		return nil, err
	}

	return fetchMetadata(ctx, ss, pf, opts)
}

// jpe does this correclty merge the span attributes into the output metadata?
func fetchMetadata(ctx context.Context, ss []traceql.Spanset, pf *parquet.File, opts common.SearchOptions) ([]traceql.SpansetMetadata, error) {
	rgs := pf.RowGroups() // jpe consolidate?
	if opts.TotalPages > 0 {
		// Read UP TO TotalPages.  The sharding calculations
		// are just estimates, so it may not line up with the
		// actual number of pages in this file.
		if opts.StartPage+opts.TotalPages > len(rgs) {
			opts.TotalPages = len(rgs) - opts.StartPage
		}
		rgs = rgs[opts.StartPage : opts.StartPage+opts.TotalPages]
	}
	makeIter := makeIterFunc(ctx, rgs, pf)

	// collect rownumbers
	rowNums := []parquetquery.RowNumber{} // jpe prealloc
	for _, ss := range ss {
		for _, span := range ss.Spans {
			rowNums = append(rowNums, span.RowNum)
		}
	}

	// span level iterator
	iters := make([]parquetquery.Iterator, 0, 4)
	iters = append(iters, &rowNumberIterator{rowNumbers: rowNums})
	iters = append(iters, makeIter(columnPathSpanStartTime, nil, columnPathSpanStartTime))
	iters = append(iters, makeIter(columnPathSpanEndTime, nil, columnPathSpanEndTime))
	iters = append(iters, makeIter(columnPathSpanID, nil, columnPathSpanID))
	spanIterator := parquetquery.NewJoinIterator(DefinitionLevelResourceSpans, iters, &spanCollector{})

	// now wrap in a trace level iterator
	traceIters := []parquetquery.Iterator{
		spanIterator,
		// Add static columns that are always return
		makeIter(columnPathTraceID, nil, columnPathTraceID),
		makeIter(columnPathStartTimeUnixNano, nil, columnPathStartTimeUnixNano),
		makeIter(columnPathDurationNanos, nil, columnPathDurationNanos),
		makeIter(columnPathRootSpanName, nil, columnPathRootSpanName),
		makeIter(columnPathRootServiceName, nil, columnPathRootServiceName),
	}
	traceIter := parquetquery.NewJoinIterator(DefinitionLevelTrace, traceIters, &traceCollector{})

	// jpe and return meta? should this function return an iterator as well?
	meta := []traceql.SpansetMetadata{}
	for {
		res, err := traceIter.Next()
		if res == nil {
			break
		}
		if err != nil {
			return nil, err
		}

		spansetMetaData := res.OtherEntries[0].Value.(*traceql.SpansetMetadata) // jpe one of these assertions is going to break, use key instead of index
		meta = append(meta, *spansetMetaData)
	}

	return meta, nil
}

// This turns groups of span values into Span objects
type spanMetaCollector struct {
	minAttributes   int                                     // jpe wut
	durationFilters []*parquetquery.GenericPredicate[int64] // jpe wut
}

var _ parquetquery.GroupPredicate = (*spanMetaCollector)(nil)

// jpe - how do we restrict to 3 spans per trace?
func (c *spanMetaCollector) KeepGroup(res *parquetquery.IteratorResult) bool {

	span := &traceql.SpanMetadata{
		Attributes: make(map[traceql.Attribute]traceql.Static),
	}

	for _, e := range res.OtherEntries { // jpe - this is likely not going to be copied through correctly
		span.Attributes[newSpanAttr(e.Key)] = e.Value.(traceql.Static)
	}

	// Merge all individual columns into the span
	for _, kv := range res.Entries {
		switch kv.Key {
		case columnPathSpanID:
			span.ID = kv.Value.ByteArray()
		case columnPathSpanStartTime:
			span.StartTimeUnixNanos = kv.Value.Uint64()
		case columnPathSpanEndTime:
			span.EndtimeUnixNanos = kv.Value.Uint64()
		case columnPathSpanName:
			span.Attributes[traceql.NewIntrinsic(traceql.IntrinsicName)] = traceql.NewStaticString(kv.Value.String())
		//case columnPathSpanDuration:
		//	span.Attributes[traceql.NewIntrinsic(traceql.IntrinsicDuration)] = traceql.NewStaticDuration(time.Duration(kv.Value.Uint64()))
		case columnPathSpanStatusCode:
			// Map OTLP status code back to TraceQL enum.
			// For other values, use the raw integer.
			var status traceql.Status
			switch kv.Value.Uint64() {
			case uint64(v1.Status_STATUS_CODE_UNSET):
				status = traceql.StatusUnset
			case uint64(v1.Status_STATUS_CODE_OK):
				status = traceql.StatusOk
			case uint64(v1.Status_STATUS_CODE_ERROR):
				status = traceql.StatusError
			default:
				status = traceql.Status(kv.Value.Uint64())
			}
			span.Attributes[traceql.NewIntrinsic(traceql.IntrinsicStatus)] = traceql.NewStaticStatus(status)
		default:
			// TODO - This exists for span-level dedicated columns like http.status_code
			// Are nils possible here?
			switch kv.Value.Kind() {
			case parquet.Boolean:
				span.Attributes[newSpanAttr(kv.Key)] = traceql.NewStaticBool(kv.Value.Boolean())
			case parquet.Int32, parquet.Int64:
				span.Attributes[newSpanAttr(kv.Key)] = traceql.NewStaticInt(int(kv.Value.Int64()))
			case parquet.Float:
				span.Attributes[newSpanAttr(kv.Key)] = traceql.NewStaticFloat(kv.Value.Double())
			case parquet.ByteArray:
				span.Attributes[newSpanAttr(kv.Key)] = traceql.NewStaticString(kv.Value.String())
			}
		}
	}

	// Save computed duration if any filters present and at least one is passed.
	if len(c.durationFilters) > 0 {
		duration := span.EndtimeUnixNanos - span.StartTimeUnixNanos
		for _, f := range c.durationFilters {
			if f == nil || f.Fn(int64(duration)) {
				span.Attributes[traceql.NewIntrinsic(traceql.IntrinsicDuration)] = traceql.NewStaticDuration(time.Duration(duration))
				break
			}
		}
	}

	if c.minAttributes > 0 {
		count := 0
		for _, v := range span.Attributes {
			if v.Type != traceql.TypeNil {
				count++
			}
		}
		if count < c.minAttributes {
			return false
		}
	}

	res.Entries = res.Entries[:0]
	res.OtherEntries = res.OtherEntries[:0]
	res.AppendOtherValue("span", span)

	return true
}

// traceMetaCollector receives rows from the resource-level matches.
// It adds trace-level attributes into the spansets before
// they are returned
type traceMetaCollector struct {
}

var _ parquetquery.GroupPredicate = (*traceMetaCollector)(nil)

func (c *traceMetaCollector) KeepGroup(res *parquetquery.IteratorResult) bool {
	finalSpanset := &traceql.SpansetMetadata{}

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

	for _, e := range res.OtherEntries {
		if spanset, ok := e.Value.(*traceql.SpansetMetadata); ok {
			finalSpanset.Spans = append(finalSpanset.Spans, spanset.Spans...)
		}
	}

	res.Entries = res.Entries[:0]
	res.OtherEntries = res.OtherEntries[:0]
	res.AppendOtherValue("spanset", finalSpanset)

	return true
}

// jpe - how to marry in attribute data?
