package vparquet

import (
	"context"
	"fmt"
	"math"

	"github.com/grafana/tempo/pkg/parquetquery"
	"github.com/grafana/tempo/pkg/traceql"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/pkg/errors"
	"github.com/segmentio/parquet-go"
)

func (b *backendBlock) FetchSeries(ctx context.Context, req traceql.FetchSpansRequest) (traceql.FetchSpansetSeriesResponse, error) {
	err := checkConditions(req.Conditions)
	if err != nil {
		return traceql.FetchSpansetSeriesResponse{}, errors.Wrap(err, "conditions invalid")
	}

	pf, rr, err := b.openForSearch(ctx, common.SearchOptions{})
	if err != nil {
		return traceql.FetchSpansetSeriesResponse{}, err
	}

	iter, err := fetchSeries(ctx, req, pf)
	if err != nil {
		return traceql.FetchSpansetSeriesResponse{}, errors.Wrap(err, "creating fetch iter")
	}

	return traceql.FetchSpansetSeriesResponse{
		Results: iter,
		Bytes:   func() uint64 { return rr.TotalBytesRead.Load() },
	}, nil
}

func fetchSeries(ctx context.Context, req traceql.FetchSpansRequest, pf *parquet.File) (*genIterator[traceql.SpansetSeries], error) {
	// Categorize conditions into span-level or resource-level
	var spanConditions, resourceConditions []traceql.Condition
	for _, cond := range req.Conditions {
		// If no-scoped intrinsic then assign default scope
		scope := cond.Attribute.Scope
		if cond.Attribute.Scope == traceql.AttributeScopeNone {
			if defscope, ok := intrinsicDefaultScope[cond.Attribute.Intrinsic]; ok {
				scope = defscope
			}
		}

		switch scope {
		case traceql.AttributeScopeNone:
			spanConditions = append(spanConditions, cond)
			resourceConditions = append(resourceConditions, cond)
			continue
		case traceql.AttributeScopeSpan:
			spanConditions = append(spanConditions, cond)
			continue
		case traceql.AttributeScopeResource:
			resourceConditions = append(resourceConditions, cond)
			continue
		default:
			return nil, fmt.Errorf("unsupported traceql scope: %s", cond.Attribute)
		}
	}

	// TODO(mapno): Is it ok to reuse the collector for all the iterators,
	//  or is there a better way?
	collector := &spansetSeriesCollector{}

	makeIter := makeIterFunc(ctx, pf.RowGroups(), pf)

	spanColumnIters, _, err := createSpanColumnIterators[*noopCollector](makeIter, spanConditions, true)
	if err != nil {
		return nil, err
	}

	start, end := req.StartTimeUnixNanos, req.EndTimeUnixNanos
	// Time range filtering?
	var startFilter, endFilter parquetquery.Predicate
	if start > 0 && end > 0 {
		// Here's how we detect the span overlaps the time window:
		// Span start <= req.End
		// Span end >= req.Start
		startFilter = parquetquery.NewIntBetweenPredicate(0, int64(end))
		endFilter = parquetquery.NewIntBetweenPredicate(int64(start), math.MaxInt64)
	}

	spanColumnIters = append(spanColumnIters, makeIter(columnPathSpanStartTime, startFilter, columnPathSpanStartTime))
	spanColumnIters = append(spanColumnIters, makeIter(columnPathSpanEndTime, endFilter, columnPathSpanEndTime))

	// TODO(mapno): Use join-left iterator
	spanIterator := parquetquery.NewJoinIterator(DefinitionLevelResourceSpansILSSpan, spanColumnIters, collector)

	resourceColumnIters, err := createResourceColumIterators[*noopCollector](makeIter, resourceConditions, true)
	if err != nil {
		return nil, err
	}
	// Put span iterator last, so it is only read when
	// the resource conditions are met.
	resourceColumnIters = append(resourceColumnIters, spanIterator)

	// Resource spans iterator
	// TODO(mapno): Use left-join iterator
	resourceIterator := parquetquery.NewJoinIterator(DefinitionLevelResourceSpans, resourceColumnIters, collector)

	traceIters := []parquetquery.Iterator{
		resourceIterator,
		// Add static columns that are always return
		makeIter(columnPathTraceID, nil, columnPathTraceID),
		makeIter(columnPathStartTimeUnixNano, nil, columnPathStartTimeUnixNano),
		makeIter(columnPathDurationNanos, nil, columnPathDurationNanos),
		makeIter(columnPathRootSpanName, nil, columnPathRootSpanName),
		makeIter(columnPathRootServiceName, nil, columnPathRootServiceName),
	}

	// Final trace iterator
	// Join iterator means it requires matching resources to have been found
	// traceCollector adds trace-level data to the spansets
	traceIter := parquetquery.NewJoinIterator(DefinitionLevelTrace, traceIters, collector)

	return &genIterator[traceql.SpansetSeries]{traceIter}, nil
}

var _ traceql.SpansetSeriesIterator = (*genIterator[traceql.SpansetSeries])(nil)

type genIterator[T any] struct {
	iter parquetquery.Iterator
}

func (i *genIterator[T]) Next(context.Context) (*T, error) {
	res, err := i.iter.Next()
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, nil
	}

	// The value is in the OtherEntries
	value := res.OtherEntries[0].Value.(*T)

	return value, nil
}

var _ parquetquery.GroupPredicate = (*spansetSeriesCollector)(nil)

// spansetSeriesCollector is a parquetquery.GroupPredicate that collects values for intrinsic and well-known columns
type spansetSeriesCollector struct{}

func (a *spansetSeriesCollector) KeepGroup(res *parquetquery.IteratorResult) bool {
	var spansetSeries *traceql.SpansetSeries
	for _, oe := range res.OtherEntries {
		if series, ok := oe.Value.(*traceql.SpansetSeries); ok {
			spansetSeries = series
		}
	}

	for _, kv := range res.Entries {
		switch kv.Key {
		// Trace well-known columns
		case columnPathRootServiceName:
			spansetSeries.ServiceName = kv.Value.String()
		case columnPathRootSpanName:
			spansetSeries.RootSpanName = kv.Value.String()
		// Resource well-known columns
		case columnPathResourceServiceName:
			spansetSeries.ServiceName = kv.Value.String()
		case columnPathResourceCluster:
			spansetSeries.Cluster = kv.Value.String()
		case columnPathResourceNamespace:
			spansetSeries.Namespace = kv.Value.String()
		case columnPathResourcePod:
			spansetSeries.Pod = kv.Value.String()
		case columnPathResourceContainer:
			spansetSeries.Container = kv.Value.String()
		case columnPathResourceK8sClusterName:
			spansetSeries.K8sCluster = kv.Value.String()
		case columnPathResourceK8sNamespaceName:
			spansetSeries.K8sNamespace = kv.Value.String()
		case columnPathResourceK8sPodName:
			spansetSeries.K8sPod = kv.Value.String()
		case columnPathResourceK8sContainerName:
			spansetSeries.K8sContainer = kv.Value.String()
		}
	}
	res.Entries = res.Entries[:0]
	res.OtherEntries = res.OtherEntries[:0]
	res.AppendOtherValue("series", spansetSeries)

	return true
}

var _ parquetquery.GroupPredicate = (*noopCollector)(nil)

type noopCollector struct{}

// TODO(mapno): Should it return false?
func (n noopCollector) KeepGroup(*parquetquery.IteratorResult) bool { return true }
