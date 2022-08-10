package vparquet

import (
	"context"
	"fmt"
	"io"
	"math"
	"strconv"
	"time"

	"github.com/opentracing/opentracing-go"
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
	StatusCodeUnset = "unset"
	StatusCodeOK    = "ok"
	StatusCodeError = "error"
)

var StatusCodeMapping = map[string]int{
	StatusCodeUnset: int(v1.Status_STATUS_CODE_UNSET),
	StatusCodeOK:    int(v1.Status_STATUS_CODE_OK),
	StatusCodeError: int(v1.Status_STATUS_CODE_ERROR),
}

func (b *backendBlock) Search(ctx context.Context, req *tempopb.SearchRequest, opts common.SearchOptions) (_ *tempopb.SearchResponse, err error) {
	span, derivedCtx := opentracing.StartSpanFromContext(ctx, "parquet.backendBlock.Search",
		opentracing.Tags{
			"blockID":   b.meta.BlockID,
			"tenantID":  b.meta.TenantID,
			"blockSize": b.meta.Size,
		})
	defer span.Finish()

	rr := NewBackendReaderAt(derivedCtx, b.r, DataFileName, b.meta.BlockID, b.meta.TenantID)
	defer func() { span.SetTag("inspectedBytes", rr.TotalBytesRead) }()

	br := tempo_io.NewBufferedReaderAt(rr, int64(b.meta.Size), opts.ReadBufferSize, opts.ReadBufferCount)

	or := newParquetOptimizedReaderAt(br, rr, int64(b.meta.Size), b.meta.FooterSize, opts.CacheControl)

	span2, _ := opentracing.StartSpanFromContext(derivedCtx, "parquet.OpenFile")
	pf, err := parquet.OpenFile(or, int64(b.meta.Size), parquet.SkipPageIndex(true))
	span2.Finish()
	if err != nil {
		return nil, err
	}

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

	// TODO: error handling
	results := searchParquetFile(derivedCtx, pf, req, rgs)
	results.Metrics.InspectedBlocks++
	results.Metrics.InspectedBytes += rr.TotalBytesRead

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

	rr := NewBackendReaderAt(derivedCtx, b.r, DataFileName, b.meta.BlockID, b.meta.TenantID)
	defer func() { span.SetTag("inspectedBytes", rr.TotalBytesRead) }()

	// jpe - use buffered reader at?
	or := newParquetOptimizedReaderAt(rr, rr, int64(b.meta.Size), b.meta.FooterSize, opts.CacheControl)

	span2, _ := opentracing.StartSpanFromContext(derivedCtx, "parquet.OpenFile")
	pf, err := parquet.OpenFile(or, int64(b.meta.Size), parquet.SkipPageIndex(true))
	span2.Finish()
	if err != nil {
		return err
	}

	// find indexes of generic attribute columns
	resourceKeyIdx, _ := pq.GetColumnIndexByPath(pf, "rs.Resource.Attrs.Key") // jpe make these 2 constants somewhere?
	spanKeyIdx, _ := pq.GetColumnIndexByPath(pf, "rs.ils.Spans.Attrs.Key")
	if resourceKeyIdx == -1 || spanKeyIdx == -1 {
		return fmt.Errorf("resource or span attributes col not found (%d, %d)", resourceKeyIdx, spanKeyIdx)
	}
	idxs := []int{
		resourceKeyIdx,
		spanKeyIdx,
	}

	// find indexes of all special columns
	unfoundIdxs := map[int]string{}
	for lbl, col := range labelMappings {
		idx, _ := pq.GetColumnIndexByPath(pf, col)
		if idx == -1 {
			continue
		}

		unfoundIdxs[idx] = lbl
	}

	// now search all row groups
	rgs := pf.RowGroups()
	for _, rg := range rgs {
		// search all special attributes
		for idx, lbl := range unfoundIdxs {
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
					delete(unfoundIdxs, idx)
					break
				}
			}
		}

		// search other attributes
		for _, idx := range idxs {
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

	rr := NewBackendReaderAt(derivedCtx, b.r, DataFileName, b.meta.BlockID, b.meta.TenantID)
	defer func() { span.SetTag("inspectedBytes", rr.TotalBytesRead) }()

	// jpe - leave off b/c this is local only at first?
	br := tempo_io.NewBufferedReaderAt(rr, int64(b.meta.Size), opts.ReadBufferSize, opts.ReadBufferCount)

	or := newParquetOptimizedReaderAt(br, rr, int64(b.meta.Size), b.meta.FooterSize, opts.CacheControl)

	span2, _ := opentracing.StartSpanFromContext(derivedCtx, "parquet.OpenFile")
	pf, err := parquet.OpenFile(or, int64(b.meta.Size), parquet.SkipPageIndex(true))
	span2.Finish()
	if err != nil {
		return err
	}

	// find column index
	columnName := labelMappings[tag]

	// jpe - make this less shitty - if this is a generic column we need to plow through the generic column values
	if columnName == "" {
		rgs := pf.RowGroups()
		makeIter := func(name string, predicate pq.Predicate, selectAs string) pq.Iterator { // jpe put someplace?
			index, _ := pq.GetColumnIndexByPath(pf, name)
			if index == -1 {
				// TODO - don't panic, error instead
				panic("column not found in parquet file:" + name)
			}
			return pq.NewColumnIterator(ctx, rgs, index, name, 1000, predicate, selectAs)
		}

		keyPred := pq.NewStringInPredicate([]string{tag})

		iter := pq.NewJoinIterator(DefinitionLevelResourceAttrs, []pq.Iterator{
			makeIter("rs.Resource.Attrs.Key", keyPred, "keys"),
			makeIter("rs.Resource.Attrs.Value", nil, "values"),
		}, nil)

		for {
			match := iter.Next()
			if match == nil {
				break
			}
			m := match.ToMap()
			for _, s := range m["values"] {
				cb(s.String())
			}
		}

		iter = pq.NewJoinIterator(DefinitionLevelResourceSpansILSSpan, []pq.Iterator{
			makeIter("rs.ils.Spans.Attrs.Key", keyPred, "keys"),
			makeIter("rs.ils.Spans.Attrs.Value", nil, "values"),
		}, nil)

		for {
			match := iter.Next()
			if match == nil {
				break
			}
			m := match.ToMap()
			for _, s := range m["values"] {
				cb(s.String())
			}
		}

		return nil
	}

	// this is a special column
	idx, _ := pq.GetColumnIndexByPath(pf, columnName)
	if idx == -1 {
		return fmt.Errorf("column not found (%s, %s)", tag, columnName)
	}

	// now search all row groups
	rgs := pf.RowGroups()
	for _, rg := range rgs {
		// search all special attributes
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

			// if this column has a dictionary we are in luck!
			dict := pg.Dictionary()
			if dict == nil {
				continue
			}

			// jpe - handle non string cols like http status code and span status
			for i := 0; i < dict.Len(); i++ {
				s := string(dict.Index(int32(i)).ByteArray())
				cb(s)
			}
		}
	}

	return nil
}

func makePipelineWithRowGroups(ctx context.Context, req *tempopb.SearchRequest, pf *parquet.File, rgs []parquet.RowGroup) (pq.Iterator, parquetSearchMetrics) {

	makeIter := func(name string, predicate pq.Predicate, selectAs string) pq.Iterator { // jpe put someplace?
		index, _ := pq.GetColumnIndexByPath(pf, name)
		if index == -1 {
			// TODO - don't panic, error instead
			panic("column not found in parquet file:" + name)
		}
		return pq.NewColumnIterator(ctx, rgs, index, name, 1000, predicate, selectAs)
	}

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

		traceIters = append(traceIters, makeIter(column, pq.NewSubstringPredicate(v), ""))
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
				makeIter("rs.Resource.Attrs.Key", keyPred, "keys"),
				makeIter("rs.Resource.Attrs.Value", valPred, "values"),
			}, nil),
			// This iterator finds all keys/values at the span level
			pq.NewJoinIterator(DefinitionLevelResourceSpansILSSpan, []pq.Iterator{
				makeIter("rs.ils.Spans.Attrs.Key", keyPred, "keys"),
				makeIter("rs.ils.Spans.Attrs.Value", valPred, "values"),
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
		return makeIter("TraceID", nil, ""), parquetSearchMetrics{}

	case 1:
		// There is only 1 iterator already, no need to wrap it up
		return traceIters[0], parquetSearchMetrics{}

	default:
		// Join all conditions
		return pq.NewJoinIterator(DefinitionLevelTrace, traceIters, nil), parquetSearchMetrics{}
	}
}

func searchParquetFile(ctx context.Context, pf *parquet.File, req *tempopb.SearchRequest, rgs []parquet.RowGroup) *tempopb.SearchResponse {

	// Search happens in 2 phases for an optimization.
	// Phase 1 is iterate all columns involved in the request.
	// Only if there are any matches do we enter phase 2, which
	// is to load the display-related columns.

	// Find matches
	matchingRows := searchRaw(ctx, pf, req, rgs)
	if len(matchingRows) == 0 {
		return &tempopb.SearchResponse{Metrics: &tempopb.SearchMetrics{}}
	}

	// We have some results, now load the display columns
	results := rawToResults(ctx, pf, rgs, matchingRows)

	return &tempopb.SearchResponse{
		Traces:  results,
		Metrics: &tempopb.SearchMetrics{},
	}
}

func searchRaw(ctx context.Context, pf *parquet.File, req *tempopb.SearchRequest, rgs []parquet.RowGroup) []pq.RowNumber {
	iter, _ := makePipelineWithRowGroups(ctx, req, pf, rgs)
	if iter == nil {
		panic("make pipeline failed")
	}
	defer iter.Close()

	// Collect matches, row numbers only.
	var matchingRows []pq.RowNumber
	for {
		match := iter.Next()
		if match == nil {
			break
		}
		matchingRows = append(matchingRows, match.RowNumber)
		if req.Limit > 0 && len(matchingRows) >= int(req.Limit) {
			break
		}
	}

	return matchingRows
}

func rawToResults(ctx context.Context, pf *parquet.File, rgs []parquet.RowGroup, rowNumbers []pq.RowNumber) []*tempopb.TraceSearchMetadata {
	makeIter := func(name string) pq.Iterator {
		index, _ := pq.GetColumnIndexByPath(pf, name)
		if index == -1 {
			// TODO - don't panic, error instead
			panic("column not found in parquet file:" + name)
		}
		return pq.NewColumnIterator(ctx, rgs, index, name, 1000, nil, name)
	}

	results := []*tempopb.TraceSearchMetadata{}
	iter2 := pq.NewJoinIterator(DefinitionLevelTrace, []pq.Iterator{
		&rowNumberIterator{rowNumbers: rowNumbers},
		makeIter("TraceID"),
		makeIter("RootServiceName"),
		makeIter("RootSpanName"),
		makeIter("StartTimeUnixNano"), // jpe use constants?
		makeIter("DurationNanos"),
	}, nil)
	defer iter2.Close()

	for {
		match := iter2.Next()
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

	return results
}

type parquetSearchMetrics struct {
	// TODO:  this isn't accurate, figure out a good way to measure this
	//pTraceID *pq.InstrumentedPredicate
}

func (p *parquetSearchMetrics) ToProto() *tempopb.SearchMetrics {
	return &tempopb.SearchMetrics{
		//InspectedTraces: uint32(p.pTraceID.InspectedValues.Load()),
	}
}

type rowNumberIterator struct {
	rowNumbers []pq.RowNumber
}

var _ pq.Iterator = (*rowNumberIterator)(nil)

func (r *rowNumberIterator) Next() *pq.IteratorResult {
	if len(r.rowNumbers) == 0 {
		return nil
	}

	res := &pq.IteratorResult{RowNumber: r.rowNumbers[0]}
	r.rowNumbers = r.rowNumbers[1:]
	return res
}

func (r *rowNumberIterator) SeekTo(to pq.RowNumber, definitionLevel int) *pq.IteratorResult {
	var at *pq.IteratorResult

	for at = r.Next(); r != nil && pq.CompareRowNumbers(definitionLevel, at.RowNumber, to) < 0; {
		at = r.Next()
	}

	return at
}

func (r *rowNumberIterator) Close() {}
