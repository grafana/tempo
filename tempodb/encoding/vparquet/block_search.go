package vparquet

import (
	"context"
	"fmt"
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
	LabelName = "name"

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

	or := NewParquetOptimizedReaderAt(br, rr, int64(b.meta.Size), uint32(b.meta.FooterSize))

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

	traceID, _ := util.ExtractTraceID(ctx)
	fmt.Println("Searched parquet file:", traceID, b.meta.BlockID, opts.StartPage, opts.TotalPages, results)

	return results, nil
}

func makePipelineWithRowGroups(ctx context.Context, req *tempopb.SearchRequest, pf *parquet.File, rgs []parquet.RowGroup) (pq.Iterator, parquetSearchMetrics) {

	makeIter := func(name string, predicate pq.Predicate, selectAs string) pq.Iterator {
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
		switch k {
		case LabelServiceName:
			resourceIters = append(resourceIters, makeIter("rs.Resource.ServiceName", pq.NewSubstringPredicate(v), ""))
		case LabelCluster:
			resourceIters = append(resourceIters, makeIter("rs.Resource.Cluster", pq.NewSubstringPredicate(v), ""))
		case LabelNamespace:
			resourceIters = append(resourceIters, makeIter("rs.Resource.Namespace", pq.NewSubstringPredicate(v), ""))
		case LabelPod:
			resourceIters = append(resourceIters, makeIter("rs.Resource.Pod", pq.NewSubstringPredicate(v), ""))
		case LabelContainer:
			resourceIters = append(resourceIters, makeIter("rs.Resource.Container", pq.NewSubstringPredicate(v), ""))
		case LabelK8sClusterName:
			resourceIters = append(resourceIters, makeIter("rs.Resource.K8sClusterName", pq.NewSubstringPredicate(v), ""))
		case LabelK8sNamespaceName:
			resourceIters = append(resourceIters, makeIter("rs.Resource.K8sNamespaceName", pq.NewSubstringPredicate(v), ""))
		case LabelK8sPodName:
			resourceIters = append(resourceIters, makeIter("rs.Resource.K8sPodName", pq.NewSubstringPredicate(v), ""))
		case LabelK8sContainerName:
			resourceIters = append(resourceIters, makeIter("rs.Resource.K8sContainerName", pq.NewSubstringPredicate(v), ""))
		case LabelName:
			resourceIters = append(resourceIters, makeIter("rs.ils.Spans.Name", pq.NewSubstringPredicate(v), ""))
		case LabelHTTPMethod:
			resourceIters = append(resourceIters, makeIter("rs.ils.Spans.HttpMethod", pq.NewSubstringPredicate(v), ""))
		case LabelHTTPUrl:
			resourceIters = append(resourceIters, makeIter("rs.ils.Spans.HttpUrl", pq.NewSubstringPredicate(v), ""))
		case LabelHTTPStatusCode:
			if i, err := strconv.Atoi(v); err == nil {
				resourceIters = append(resourceIters, makeIter("rs.ils.Spans.HttpStatusCode", pq.NewIntBetweenPredicate(int64(i), int64(i)), ""))
				break
			}
			// Non-numeric string field
			otherAttrConditions[k] = v
		case StatusCodeTag:
			code := StatusCodeMapping[v]
			resourceIters = append(resourceIters, makeIter("rs.ils.Spans.StatusCode", pq.NewIntBetweenPredicate(int64(code), int64(code)), ""))
		default:
			otherAttrConditions[k] = v
		}
	}

	// Generic attribute conditions?
	if len(otherAttrConditions) > 0 {
		// We are looking for one or more foo=bar attributes that aren't
		// projected to their own columns, they are in the generic Key/Value
		// columns at the resource or span levels.  We want to search
		// both locations. But we also only want to read the columns once.

		var keys []string
		var vals []string
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
		makeIter("StartTimeUnixNano"),
		makeIter("DurationNanos"),
	}, nil)

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
