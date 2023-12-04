package vparquet3

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"strconv"
	"time"

	"github.com/opentracing/opentracing-go"
	"github.com/parquet-go/parquet-go"

	pq "github.com/grafana/tempo/pkg/parquetquery"
	"github.com/grafana/tempo/pkg/tempopb"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/traceql"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

// These are reserved search parameters
const (
	LabelDuration = "duration"

	StatusCodeTag   = "status.code"
	StatusCodeUnset = "unset"
	StatusCodeOK    = "ok"
	StatusCodeError = "error"

	KindUnspecified = "unspecified"
	KindInternal    = "internal"
	KindClient      = "client"
	KindServer      = "server"
	KindProducer    = "producer"
	KindConsumer    = "consumer"

	EnvVarAsyncIteratorName  = "VPARQUET_ASYNC_ITERATOR"
	EnvVarAsyncIteratorValue = "1"
)

var StatusCodeMapping = map[string]int{
	StatusCodeUnset: int(v1.Status_STATUS_CODE_UNSET),
	StatusCodeOK:    int(v1.Status_STATUS_CODE_OK),
	StatusCodeError: int(v1.Status_STATUS_CODE_ERROR),
}

var KindMapping = map[string]int{
	KindUnspecified: int(v1.Span_SPAN_KIND_UNSPECIFIED),
	KindInternal:    int(v1.Span_SPAN_KIND_INTERNAL),
	KindClient:      int(v1.Span_SPAN_KIND_CLIENT),
	KindServer:      int(v1.Span_SPAN_KIND_SERVER),
	KindProducer:    int(v1.Span_SPAN_KIND_PRODUCER),
	KindConsumer:    int(v1.Span_SPAN_KIND_CONSUMER),
}

// openForSearch consolidates all the logic for opening a parquet file
func (b *backendBlock) openForSearch(ctx context.Context, opts common.SearchOptions) (*parquet.File, *BackendReaderAt, error) {
	b.openMtx.Lock()
	defer b.openMtx.Unlock()

	// TODO: ctx is also cached when we cache backendReaderAt, not ideal but leaving it as is for now
	backendReaderAt := NewBackendReaderAt(ctx, b.r, DataFileName, b.meta)
	// no searches currently require bloom filters or the page index. so just add them statically
	o := []parquet.FileOption{
		parquet.SkipBloomFilters(true),
		parquet.SkipPageIndex(true),
		parquet.FileReadMode(parquet.ReadModeAsync),
	}

	// if the read buffer size provided is <= 0 then we'll use the parquet default
	readBufferSize := opts.ReadBufferSize
	if readBufferSize <= 0 {
		readBufferSize = parquet.DefaultFileConfig().ReadBufferSize
	}

	o = append(o, parquet.ReadBufferSize(readBufferSize))

	// cached reader
	cachedReaderAt := newCachedReaderAt(backendReaderAt, readBufferSize, int64(b.meta.Size), b.meta.FooterSize) // most reads to the backend are going to be readbuffersize so use it as our "page cache" size

	span, _ := opentracing.StartSpanFromContext(ctx, "parquet.OpenFile")
	defer span.Finish()
	pf, err := parquet.OpenFile(cachedReaderAt, int64(b.meta.Size), o...)

	return pf, backendReaderAt, err
}

func (b *backendBlock) Search(ctx context.Context, req *tempopb.SearchRequest, opts common.SearchOptions) (_ *tempopb.SearchResponse, err error) {
	span, derivedCtx := opentracing.StartSpanFromContext(ctx, "parquet.backendBlock.Search",
		opentracing.Tags{
			"blockID":   b.meta.BlockID,
			"tenantID":  b.meta.TenantID,
			"blockSize": b.meta.Size,
		})
	defer span.Finish()

	pf, rr, err := b.openForSearch(derivedCtx, opts)
	if err != nil {
		return nil, fmt.Errorf("unexpected error opening parquet file: %w", err)
	}
	defer func() { span.SetTag("inspectedBytes", rr.BytesRead()) }()

	// Get list of row groups to inspect. Ideally we use predicate pushdown
	// here to keep only row groups that can potentially satisfy the request
	// conditions, but don't have it figured out yet.
	rgs := rowGroupsFromFile(pf, opts)
	results, err := searchParquetFile(derivedCtx, pf, req, rgs, b.meta.DedicatedColumns)
	if err != nil {
		return nil, err
	}
	results.Metrics.InspectedBytes += rr.BytesRead()
	results.Metrics.InspectedTraces += uint32(b.meta.TotalObjects)

	return results, nil
}

func makePipelineWithRowGroups(ctx context.Context, req *tempopb.SearchRequest, pf *parquet.File, rgs []parquet.RowGroup, dc backend.DedicatedColumns) pq.Iterator {
	makeIter := makeIterFunc(ctx, rgs, pf)

	// Wire up iterators
	var resourceIters []pq.Iterator
	var traceIters []pq.Iterator

	// Dedicated column mappings
	spanAndResourceColumnMapping := dedicatedColumnsToColumnMapping(dc)

	otherAttrConditions := map[string]string{}

	for k, v := range req.Tags {
		// dedicated attribute columns
		if c, ok := spanAndResourceColumnMapping.get(k); ok {
			resourceIters = append(resourceIters, makeIter(c.ColumnPath, pq.NewSubstringPredicate(v), ""))
			continue
		}

		column := labelMappings[k]
		// if we don't have a column mapping then pass it forward to otherAttribute handling
		if column == "" {
			otherAttrConditions[k] = v
			continue
		}

		// most columns are just a substring predicate over the column, but we have
		// special handling for http status code and span status
		if k == LabelHTTPStatusCode {
			if i, err := strconv.Atoi(v); err == nil {
				resourceIters = append(resourceIters, makeIter(column, pq.NewIntBetweenPredicate(int64(i), int64(i)), ""))
				continue
			}
			// Non-numeric string field
			otherAttrConditions[k] = v
			continue
		}
		if k == LabelStatusCode {
			code := StatusCodeMapping[v]
			resourceIters = append(resourceIters, makeIter(column, pq.NewIntBetweenPredicate(int64(code), int64(code)), ""))
			continue
		}

		if k == LabelRootServiceName || k == LabelRootSpanName {
			traceIters = append(traceIters, makeIter(column, pq.NewSubstringPredicate(v), ""))
		} else {
			resourceIters = append(resourceIters, makeIter(column, pq.NewSubstringPredicate(v), ""))
		}
	}

	// Generic attribute conditions?
	if len(otherAttrConditions) > 0 {
		// We are looking for one or more foo=bar attributes that aren't
		// projected to their own columns, they are in the generic Key/Value
		// columns at the resource or span levels.  We want to search
		// both locations. But we also only want to read the columns once.

		keys := make([]string, 0, len(otherAttrConditions))
		vals := make([]string, 0, len(otherAttrConditions))
		for k, v := range otherAttrConditions {
			keys = append(keys, k)
			vals = append(vals, v)
		}

		keyPred := pq.NewStringInPredicate(keys)
		valPred := pq.NewStringInPredicate(vals)

		// This iterator combines the results from the resource
		// and span searches, and checks if all conditions were satisfied
		// on each ResourceSpans.  This is a single-pass over the attribute columns.
		j := pq.NewUnionIterator(DefinitionLevelResourceSpans, []pq.Iterator{
			// This iterator finds all keys/values at the resource level
			pq.NewJoinIterator(DefinitionLevelResourceAttrs, []pq.Iterator{
				makeIter(FieldResourceAttrKey, keyPred, "keys"),
				makeIter(FieldResourceAttrVal, valPred, "values"),
			}, nil),
			// This iterator finds all keys/values at the span level
			pq.NewJoinIterator(DefinitionLevelResourceSpansILSSpanAttrs, []pq.Iterator{
				makeIter(FieldSpanAttrKey, keyPred, "keys"),
				makeIter(FieldSpanAttrVal, valPred, "values"),
			}, nil),
		}, pq.NewKeyValueGroupPredicate(keys, vals))

		resourceIters = append(resourceIters, j)
	}

	// Multiple resource-level filters get joined and wrapped
	// up to trace-level. A single filter can be used as-is
	if len(resourceIters) == 1 {
		traceIters = append(traceIters, resourceIters[0])
	}
	if len(resourceIters) > 1 {
		traceIters = append(traceIters, pq.NewJoinIterator(DefinitionLevelTrace, resourceIters, nil))
	}

	// Duration filtering?
	if req.MinDurationMs > 0 || req.MaxDurationMs > 0 {
		min := int64(0)
		if req.MinDurationMs > 0 {
			min = (time.Millisecond * time.Duration(req.MinDurationMs)).Nanoseconds()
		}
		max := int64(math.MaxInt64)
		if req.MaxDurationMs > 0 {
			max = (time.Millisecond * time.Duration(req.MaxDurationMs)).Nanoseconds()
		}
		durFilter := pq.NewIntBetweenPredicate(min, max)
		traceIters = append(traceIters, makeIter("DurationNano", durFilter, "Duration"))
	}

	// Time range filtering?
	if req.Start > 0 && req.End > 0 {
		// Here's how we detect the trace overlaps the time window:

		// Trace start <= req.End
		startFilter := pq.NewIntBetweenPredicate(0, time.Unix(int64(req.End), 0).UnixNano())
		traceIters = append(traceIters, makeIter("StartTimeUnixNano", startFilter, "StartTime"))

		// Trace end >= req.Start, only if column exists
		if pq.HasColumn(pf, "EndTimeUnixNano") {
			endFilter := pq.NewIntBetweenPredicate(time.Unix(int64(req.Start), 0).UnixNano(), math.MaxInt64)
			traceIters = append(traceIters, makeIter("EndTimeUnixNano", endFilter, ""))
		}
	}

	switch len(traceIters) {

	case 0:
		// Empty request, in this case every trace matches so we can
		// simply iterate any column.
		return makeIter("TraceID", nil, "")

	case 1:
		// There is only 1 iterator already, no need to wrap it up
		return traceIters[0]

	default:
		// Join all conditions
		return pq.NewJoinIterator(DefinitionLevelTrace, traceIters, nil)
	}
}

func searchParquetFile(ctx context.Context, pf *parquet.File, req *tempopb.SearchRequest, rgs []parquet.RowGroup, dc backend.DedicatedColumns) (*tempopb.SearchResponse, error) {
	// Search happens in 2 phases for an optimization.
	// Phase 1 is iterate all columns involved in the request.
	// Only if there are any matches do we enter phase 2, which
	// is to load the display-related columns.

	// Find matches
	matchingRows, err := searchRaw(ctx, pf, req, rgs, dc)
	if err != nil {
		return nil, err
	}
	if len(matchingRows) == 0 {
		return &tempopb.SearchResponse{Metrics: &tempopb.SearchMetrics{}}, nil
	}

	// We have some results, now load the display columns
	results, err := rawToResults(ctx, pf, rgs, matchingRows)
	if err != nil {
		return nil, err
	}

	return &tempopb.SearchResponse{
		Traces:  results,
		Metrics: &tempopb.SearchMetrics{},
	}, nil
}

func searchRaw(ctx context.Context, pf *parquet.File, req *tempopb.SearchRequest, rgs []parquet.RowGroup, dc backend.DedicatedColumns) ([]pq.RowNumber, error) {
	iter := makePipelineWithRowGroups(ctx, req, pf, rgs, dc)
	if iter == nil {
		return nil, errors.New("make pipeline returned a nil iterator")
	}
	defer iter.Close()

	// Collect matches, row numbers only.
	var matchingRows []pq.RowNumber
	for {
		match, err := iter.Next()
		if err != nil {
			return nil, fmt.Errorf("searchRaw next failed: %w", err)
		}
		if match == nil {
			break
		}
		matchingRows = append(matchingRows, match.RowNumber)
		if req.Limit > 0 && len(matchingRows) >= int(req.Limit) {
			break
		}
	}

	return matchingRows, nil
}

func rawToResults(ctx context.Context, pf *parquet.File, rgs []parquet.RowGroup, rowNumbers []pq.RowNumber) ([]*tempopb.TraceSearchMetadata, error) {
	makeIter := makeIterFunc(ctx, rgs, pf)

	results := []*tempopb.TraceSearchMetadata{}
	iter2 := pq.NewJoinIterator(DefinitionLevelTrace, []pq.Iterator{
		&rowNumberIterator{rowNumbers: rowNumbers},
		makeIter("TraceID", nil, "TraceID"),
		makeIter("RootServiceName", nil, "RootServiceName"),
		makeIter("RootSpanName", nil, "RootSpanName"),
		makeIter("StartTimeUnixNano", nil, "StartTimeUnixNano"),
		makeIter("DurationNano", nil, "DurationNano"),
	}, nil)
	defer iter2.Close()

	for {
		match, err := iter2.Next()
		if err != nil {
			return nil, fmt.Errorf("rawToResults next failed: %w", err)
		}
		if match == nil {
			break
		}

		matchMap := match.ToMap()
		result := &tempopb.TraceSearchMetadata{
			TraceID:           util.TraceIDToHexString(matchMap["TraceID"][0].Bytes()),
			RootServiceName:   matchMap["RootServiceName"][0].String(),
			RootTraceName:     matchMap["RootSpanName"][0].String(),
			StartTimeUnixNano: matchMap["StartTimeUnixNano"][0].Uint64(),
			DurationMs:        uint32(matchMap["DurationNano"][0].Int64() / int64(time.Millisecond)),
		}
		results = append(results, result)
	}

	return results, nil
}

func makeIterFunc(ctx context.Context, rgs []parquet.RowGroup, pf *parquet.File) func(name string, predicate pq.Predicate, selectAs string) pq.Iterator {
	async := os.Getenv(EnvVarAsyncIteratorName) == EnvVarAsyncIteratorValue

	return func(name string, predicate pq.Predicate, selectAs string) pq.Iterator {
		index, _ := pq.GetColumnIndexByPath(pf, name)
		if index == -1 {
			// TODO - don't panic, error instead
			panic("column not found in parquet file:" + name)
		}

		if async {
			return pq.NewColumnIterator(ctx, rgs, index, name, 1000, predicate, selectAs)
		}

		return pq.NewSyncIterator(ctx, rgs, index, name, 1000, predicate, selectAs)
	}
}

type rowNumberIterator struct {
	rowNumbers []pq.RowNumber
}

var _ pq.Iterator = (*rowNumberIterator)(nil)

func (r *rowNumberIterator) String() string {
	return "rowNumberIterator()"
}

func (r *rowNumberIterator) Next() (*pq.IteratorResult, error) {
	if len(r.rowNumbers) == 0 {
		return nil, nil
	}

	res := &pq.IteratorResult{RowNumber: r.rowNumbers[0]}
	r.rowNumbers = r.rowNumbers[1:]
	return res, nil
}

func (r *rowNumberIterator) SeekTo(to pq.RowNumber, definitionLevel int) (*pq.IteratorResult, error) {
	var at *pq.IteratorResult

	for at, _ = r.Next(); r != nil && at != nil && pq.CompareRowNumbers(definitionLevel, at.RowNumber, to) < 0; {
		at, _ = r.Next()
	}

	return at, nil
}

func (r *rowNumberIterator) Close() {}

// reportValuesPredicate is a "fake" predicate that uses existing iterator logic to find all values in a given column
type reportValuesPredicate struct {
	cb common.TagCallbackV2
}

func newReportValuesPredicate(cb common.TagCallbackV2) *reportValuesPredicate {
	return &reportValuesPredicate{cb: cb}
}

func (r *reportValuesPredicate) String() string {
	return "reportValuesPredicate{}"
}

// KeepColumnChunk checks to see if the page has a dictionary. if it does then we can report the values contained in it
// and return false b/c we don't have to go to the actual columns to retrieve values. if there is no dict we return
// true so the iterator will call KeepValue on all values in the column
func (r *reportValuesPredicate) KeepColumnChunk(cc *pq.ColumnChunkHelper) bool {
	if d := cc.Dictionary(); d != nil {
		for i := 0; i < d.Len(); i++ {
			v := d.Index(int32(i))
			if callback(r.cb, v) {
				break
			}
		}

		// No need to check the pages since this was a dictionary
		// column.
		return false
	}

	return true
}

// KeepPage always returns true because if we get this far we need to
// inspect each individual value.
func (r *reportValuesPredicate) KeepPage(parquet.Page) bool {
	return true
}

// KeepValue is only called if this column does not have a dictionary. Just report everything to r.cb and
// return false so the iterator do any extra work.
func (r *reportValuesPredicate) KeepValue(v parquet.Value) bool {
	callback(r.cb, v)

	return false
}

func callback(cb common.TagCallbackV2, v parquet.Value) (stop bool) {
	switch v.Kind() {

	case parquet.Boolean:
		return cb(traceql.NewStaticBool(v.Boolean()))

	case parquet.Int32, parquet.Int64:
		return cb(traceql.NewStaticInt(int(v.Int64())))

	case parquet.Float, parquet.Double:
		return cb(traceql.NewStaticFloat(v.Double()))

	case parquet.ByteArray, parquet.FixedLenByteArray:
		return cb(traceql.NewStaticString(v.String()))

	default:
		// Skip nils or unsupported type
		return false
	}
}

func rowGroupsFromFile(pf *parquet.File, opts common.SearchOptions) []parquet.RowGroup {
	rgs := pf.RowGroups()
	if opts.TotalPages > 0 {
		// Read UP TO TotalPages.  The sharding calculations
		// are just estimates, so it may not line up with the
		// actual number of pages in this file.
		if opts.StartPage+opts.TotalPages > len(rgs) {
			opts.TotalPages = len(rgs) - opts.StartPage
		}
		rgs = rgs[opts.StartPage : opts.StartPage+opts.TotalPages]
	}

	return rgs
}
