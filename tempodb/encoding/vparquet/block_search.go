package vparquet

import (
	"context"
	"fmt"
	"io"
	"math"
	"strconv"
	"time"

	"github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
	"github.com/segmentio/parquet-go"

	tempo_io "github.com/grafana/tempo/pkg/io"
	pq "github.com/grafana/tempo/pkg/parquetquery"
	"github.com/grafana/tempo/pkg/tempopb"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

// These are reserved search parameters
const (
	LabelDuration = "duration"

	StatusCodeTag   = "status.code"
	StatusCodeUnset = "unset"
	StatusCodeOK    = "ok"
	StatusCodeError = "error"
)

var StatusCodeMapping = map[string]int{
	StatusCodeUnset: int(v1.Status_STATUS_CODE_UNSET),
	StatusCodeOK:    int(v1.Status_STATUS_CODE_OK),
	StatusCodeError: int(v1.Status_STATUS_CODE_ERROR),
}

// openForSearch consolidates all the logic regarding opening a parquet file in object storage
func (b *backendBlock) openForSearch(ctx context.Context, opts common.SearchOptions, o ...parquet.FileOption) (*parquet.File, *BackendReaderAt, error) {
	backendReaderAt := NewBackendReaderAt(ctx, b.r, DataFileName, b.meta.BlockID, b.meta.TenantID)

	// backend reader
	readerAt := io.ReaderAt(backendReaderAt)

	// buffering
	if opts.ReadBufferSize > 0 {
		//   only use buffered reader at if the block is small, otherwise it's far more effective to use larger
		//   buffers in the parquet sdk
		if opts.ReadBufferCount*opts.ReadBufferSize > int(b.meta.Size) {
			readerAt = tempo_io.NewBufferedReaderAt(readerAt, int64(b.meta.Size), opts.ReadBufferSize, opts.ReadBufferCount)
		} else {
			o = append(o, parquet.ReadBufferSize(opts.ReadBufferSize))
		}
	}

	// optimized reader
	readerAt = newParquetOptimizedReaderAt(readerAt, int64(b.meta.Size), b.meta.FooterSize)

	// cached reader
	if opts.CacheControl.ColumnIndex || opts.CacheControl.Footer || opts.CacheControl.OffsetIndex {
		readerAt = newCachedReaderAt(readerAt, backendReaderAt, opts.CacheControl)
	}

	span, _ := opentracing.StartSpanFromContext(ctx, "parquet.OpenFile")
	defer span.Finish()
	pf, err := parquet.OpenFile(readerAt, int64(b.meta.Size), o...)

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

	pf, rr, err := b.openForSearch(derivedCtx, opts, parquet.SkipPageIndex(true))
	if err != nil {
		return nil, fmt.Errorf("unexpected error opening parquet file: %w", err)
	}
	defer func() { span.SetTag("inspectedBytes", rr.TotalBytesRead.Load()) }()

	// Get list of row groups to inspect. Ideally we use predicate pushdown
	// here to keep only row groups that can potentially satisfy the request
	// conditions, but don't have it figured out yet.
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

	results, err := searchParquetFile(derivedCtx, pf, req, rgs)
	if err != nil {
		return nil, err
	}
	results.Metrics.InspectedBlocks++
	results.Metrics.InspectedBytes += rr.TotalBytesRead.Load()
	results.Metrics.InspectedTraces += uint32(b.meta.TotalObjects)

	return results, nil
}

func (b *backendBlock) SearchTags(ctx context.Context, cb common.TagCallback, opts common.SearchOptions) error {
	span, derivedCtx := opentracing.StartSpanFromContext(ctx, "parquet.backendBlock.SearchTags",
		opentracing.Tags{
			"blockID":   b.meta.BlockID,
			"tenantID":  b.meta.TenantID,
			"blockSize": b.meta.Size,
		})
	defer span.Finish()

	pf, rr, err := b.openForSearch(derivedCtx, opts)
	if err != nil {
		return fmt.Errorf("unexpected error opening parquet file: %w", err)
	}
	defer func() { span.SetTag("inspectedBytes", rr.TotalBytesRead.Load()) }()

	// find indexes of generic attribute columns
	resourceKeyIdx, _ := pq.GetColumnIndexByPath(pf, FieldResourceAttrKey)
	spanKeyIdx, _ := pq.GetColumnIndexByPath(pf, FieldSpanAttrKey)
	if resourceKeyIdx == -1 || spanKeyIdx == -1 {
		return fmt.Errorf("resource or span attributes col not found (%d, %d)", resourceKeyIdx, spanKeyIdx)
	}
	standardAttrIdxs := []int{
		resourceKeyIdx,
		spanKeyIdx,
	}

	// find indexes of all special columns
	specialAttrIdxs := map[int]string{}
	for lbl, col := range labelMappings {
		idx, _ := pq.GetColumnIndexByPath(pf, col)
		if idx == -1 {
			continue
		}

		specialAttrIdxs[idx] = lbl
	}

	// now search all row groups
	rgs := pf.RowGroups()
	for _, rg := range rgs {
		// search all special attributes
		for idx, lbl := range specialAttrIdxs {
			cc := rg.ColumnChunks()[idx]
			pgs := cc.Pages()
			for {
				pg, err := pgs.ReadPage()
				if err == io.EOF || pg == nil {
					break
				}
				if err != nil {
					return err
				}

				// if a special attribute has any non-null values, include it
				if pg.NumNulls() < pg.NumValues() {
					cb(lbl)
					delete(specialAttrIdxs, idx) // remove from map so we won't search again
					break
				}
			}
		}

		// search other attributes
		for _, idx := range standardAttrIdxs {
			cc := rg.ColumnChunks()[idx]
			pgs := cc.Pages()
			for {
				pg, err := pgs.ReadPage()
				if err == io.EOF || pg == nil {
					break
				}
				if err != nil {
					return err
				}

				dict := pg.Dictionary()
				if dict == nil {
					continue
				}

				for i := 0; i < dict.Len(); i++ {
					s := string(dict.Index(int32(i)).ByteArray())
					cb(s)
				}
			}
		}
	}

	return nil
}

func (b *backendBlock) SearchTagValues(ctx context.Context, tag string, cb common.TagCallback, opts common.SearchOptions) error {
	span, derivedCtx := opentracing.StartSpanFromContext(ctx, "parquet.backendBlock.SearchTagValues",
		opentracing.Tags{
			"blockID":   b.meta.BlockID,
			"tenantID":  b.meta.TenantID,
			"blockSize": b.meta.Size,
		})
	defer span.Finish()

	pf, rr, err := b.openForSearch(derivedCtx, opts)
	if err != nil {
		return fmt.Errorf("unexpected error opening parquet file: %w", err)
	}
	defer func() { span.SetTag("inspectedBytes", rr.TotalBytesRead.Load()) }()

	// labelMappings will indicate whether this is a search for a special or standard
	// column
	column := labelMappings[tag]
	if column == "" {
		err = searchStandardTagValues(ctx, tag, pf, cb)
		if err != nil {
			return fmt.Errorf("unexpected error searching standard tags: %w", err)
		}
		return nil
	}

	err = searchSpecialTagValues(ctx, column, pf, cb)
	if err != nil {
		return fmt.Errorf("unexpected error searching special tags: %w", err)
	}
	return nil
}

func makePipelineWithRowGroups(ctx context.Context, req *tempopb.SearchRequest, pf *parquet.File, rgs []parquet.RowGroup) pq.Iterator {
	makeIter := makeIterFunc(ctx, rgs, pf)

	// Wire up iterators
	var resourceIters []pq.Iterator
	var traceIters []pq.Iterator

	otherAttrConditions := map[string]string{}

	for k, v := range req.Tags {
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
				break
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
			pq.NewJoinIterator(DefinitionLevelResourceSpansILSSpan, []pq.Iterator{
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
		traceIters = append(traceIters, makeIter("DurationNanos", durFilter, "Duration"))
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

func searchParquetFile(ctx context.Context, pf *parquet.File, req *tempopb.SearchRequest, rgs []parquet.RowGroup) (*tempopb.SearchResponse, error) {

	// Search happens in 2 phases for an optimization.
	// Phase 1 is iterate all columns involved in the request.
	// Only if there are any matches do we enter phase 2, which
	// is to load the display-related columns.

	// Find matches
	matchingRows, err := searchRaw(ctx, pf, req, rgs)
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

func searchRaw(ctx context.Context, pf *parquet.File, req *tempopb.SearchRequest, rgs []parquet.RowGroup) ([]pq.RowNumber, error) {
	iter := makePipelineWithRowGroups(ctx, req, pf, rgs)
	if iter == nil {
		return nil, errors.New("make pipeline returned a nil iterator")
	}
	defer iter.Close()

	// Collect matches, row numbers only.
	var matchingRows []pq.RowNumber
	for {
		match, err := iter.Next()
		if err != nil {
			return nil, errors.Wrap(err, "searchRaw next failed")
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
		makeIter("DurationNanos", nil, "DurationNanos"),
	}, nil)
	defer iter2.Close()

	for {
		match, err := iter2.Next()
		if err != nil {
			return nil, errors.Wrap(err, "rawToResults next failed")
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
			DurationMs:        uint32(matchMap["DurationNanos"][0].Int64() / int64(time.Millisecond)),
		}
		results = append(results, result)
	}

	return results, nil
}

// searchStandardTagValues searches a parquet file for "standard" tags. i.e. tags that don't have unique
// columns and are contained in labelMappings
func searchStandardTagValues(ctx context.Context, tag string, pf *parquet.File, cb common.TagCallback) error {
	rgs := pf.RowGroups()
	makeIter := makeIterFunc(ctx, rgs, pf)

	keyPred := pq.NewStringInPredicate([]string{tag})

	iter := pq.NewJoinIterator(DefinitionLevelResourceAttrs, []pq.Iterator{
		makeIter(FieldResourceAttrKey, keyPred, "keys"),
		makeIter(FieldResourceAttrVal, nil, "values"),
	}, nil)
	for {
		match, err := iter.Next()
		if err != nil {
			return errors.Wrap(err, "iter.Next on failed on resource lookup")
		}
		if match == nil {
			break
		}
		m := match.ToMap()
		for _, s := range m["values"] {
			cb(s.String())
		}
	}
	iter.Close()

	iter = pq.NewJoinIterator(DefinitionLevelResourceSpansILSSpan, []pq.Iterator{
		makeIter(FieldSpanAttrKey, keyPred, "keys"),
		makeIter(FieldSpanAttrVal, nil, "values"),
	}, nil)
	for {
		match, err := iter.Next()
		if err != nil {
			return errors.Wrap(err, "iter.Next on failed on span lookup")
		}
		if match == nil {
			break
		}
		m := match.ToMap()
		for _, s := range m["values"] {
			cb(s.String())
		}
	}
	iter.Close()

	return nil
}

// searchSpecialTagValues searches a parquet file for all values for the provided column. It first attempts
// to only pull all values from the column's dictionary. If this fails it falls back to scanning the entire path.
func searchSpecialTagValues(ctx context.Context, column string, pf *parquet.File, cb common.TagCallback) error {
	pred := newReportValuesPredicate(cb)
	rgs := pf.RowGroups()

	iter := makeIterFunc(ctx, rgs, pf)(column, pred, "")
	defer iter.Close()
	for {
		match, err := iter.Next()
		if err != nil {
			return errors.Wrap(err, "iter.Next failed")
		}
		if match == nil {
			break
		}
	}

	return nil
}

func makeIterFunc(ctx context.Context, rgs []parquet.RowGroup, pf *parquet.File) func(name string, predicate pq.Predicate, selectAs string) pq.Iterator {
	return func(name string, predicate pq.Predicate, selectAs string) pq.Iterator {
		index, _ := pq.GetColumnIndexByPath(pf, name)
		if index == -1 {
			// TODO - don't panic, error instead
			panic("column not found in parquet file:" + name)
		}
		return pq.NewColumnIterator(ctx, rgs, index, name, 1000, predicate, selectAs)
	}
}

type rowNumberIterator struct {
	rowNumbers []pq.RowNumber
}

var _ pq.Iterator = (*rowNumberIterator)(nil)

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

	for at, _ = r.Next(); r != nil && pq.CompareRowNumbers(definitionLevel, at.RowNumber, to) < 0; {
		at, _ = r.Next()
	}

	return at, nil
}

func (r *rowNumberIterator) Close() {}

// reportValuesPredicate is a "fake" predicate that uses existing iterator logic to find all values in a given column
type reportValuesPredicate struct {
	cb common.TagCallback
}

func newReportValuesPredicate(cb common.TagCallback) *reportValuesPredicate {
	return &reportValuesPredicate{cb: cb}
}

// KeepColumnChunk always returns true b/c we always have to dig deeper to find all values
func (r *reportValuesPredicate) KeepColumnChunk(cc parquet.ColumnChunk) bool {
	return true
}

// KeepPage checks to see if the page has a dictionary. if it does then we can report the values contained in it
// and return false b/c we don't have to go to the actual columns to retrieve values. if there is no dict we return
// true so the iterator will call KeepValue on all values in the column
func (r *reportValuesPredicate) KeepPage(pg parquet.Page) bool {
	if dict := pg.Dictionary(); dict != nil {
		for i := 0; i < dict.Len(); i++ {
			s := string(dict.Index(int32(i)).ByteArray())
			r.cb(s)
		}

		return false
	}

	return true
}

// KeepValue is only called if this column does not have a dictionary. Just report everything to r.cb and
// return false so the iterator do any extra work.
func (r *reportValuesPredicate) KeepValue(v parquet.Value) bool {
	r.cb(v.String())

	return false
}
