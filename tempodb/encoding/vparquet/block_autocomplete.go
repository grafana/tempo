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
	spanIterator := parquetquery.NewJoinIterator(DefinitionLevelResourceSpansILSSpan, spanColumnIters, &spanSeriesCollector{})

	resourceColumnIters, err := createResourceColumIterators[*noopCollector](makeIter, resourceConditions, true)
	if err != nil {
		return nil, err
	}
	// Put span iterator last, so it is only read when
	// the resource conditions are met.
	resourceColumnIters = append(resourceColumnIters, spanIterator)

	// Resource spans iterator
	// TODO(mapno): Use left-join iterator
	resourceIterator := parquetquery.NewJoinIterator(DefinitionLevelResourceSpans, resourceColumnIters, &resourceSeriesCollector{})

	// TODO(mapno): Do we need this?
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
	traceIter := parquetquery.NewJoinIterator(DefinitionLevelTrace, traceIters, &spansetSeriesCollector{})

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

var _ parquetquery.GroupPredicate = (*spanSeriesCollector)(nil)

type spanSeriesCollector struct{}

func (c *spanSeriesCollector) KeepGroup(res *parquetquery.IteratorResult) bool {
	spanSeries := &traceql.SpanSeries{}
	for _, kv := range res.Entries {
		switch kv.Key {
		case LabelHTTPMethod:
			spanSeries.HTTPMethod = kv.Value.String()
		case LabelHTTPUrl:
			spanSeries.HTTPUrl = kv.Value.String()
		case LabelHTTPStatusCode:
			spanSeries.HTTPStatusCode = int(kv.Value.Int32())
		}
	}
	res.Entries = res.Entries[:0]
	res.OtherEntries = res.OtherEntries[:0]
	res.AppendOtherValue("series", spanSeries)

	return true
}

var _ parquetquery.GroupPredicate = (*resourceSeriesCollector)(nil)

type resourceSeriesCollector struct{}

func (c *resourceSeriesCollector) KeepGroup(res *parquetquery.IteratorResult) bool {
	resourceSeries := &traceql.ResourceSeries{}

	spanSeries := make([]traceql.SpanSeries, 0, len(res.OtherEntries))
	for _, oe := range res.OtherEntries {
		switch oe.Value.(type) {
		case *traceql.SpanSeries:
			spanSeries = append(spanSeries, *oe.Value.(*traceql.SpanSeries))
		}
	}
	resourceSeries.SpanSeries = spanSeries

	for _, kv := range res.Entries {
		switch kv.Key {
		case LabelServiceName:
			resourceSeries.ServiceName = kv.Value.String()
		case LabelCluster:
			resourceSeries.Cluster = kv.Value.String()
		case LabelNamespace:
			resourceSeries.Namespace = kv.Value.String()
		case LabelPod:
			resourceSeries.Pod = kv.Value.String()
		case LabelContainer:
			resourceSeries.Container = kv.Value.String()
		case LabelK8sClusterName:
			resourceSeries.K8sCluster = kv.Value.String()
		case LabelK8sNamespaceName:
			resourceSeries.K8sNamespace = kv.Value.String()
		case LabelK8sPodName:
			resourceSeries.K8sPod = kv.Value.String()
		case LabelK8sContainerName:
			resourceSeries.K8sContainer = kv.Value.String()
		}
	}
	res.Entries = res.Entries[:0]
	res.OtherEntries = res.OtherEntries[:0]
	res.AppendOtherValue("series", resourceSeries)
	return true
}

var _ parquetquery.GroupPredicate = (*spansetSeriesCollector)(nil)

// spansetSeriesCollector is a parquetquery.GroupPredicate that collects values for intrinsic and well-known columns
type spansetSeriesCollector struct{}

func (a *spansetSeriesCollector) KeepGroup(res *parquetquery.IteratorResult) bool {
	spansetSeries := &traceql.SpansetSeries{}

	resourceSeries := make([]traceql.ResourceSeries, 0, len(res.OtherEntries))
	for _, oe := range res.OtherEntries {
		switch oe.Value.(type) {
		case *traceql.ResourceSeries:
			resourceSeries = append(resourceSeries, *oe.Value.(*traceql.ResourceSeries))
		}
	}
	spansetSeries.ResourceSeries = resourceSeries

	for _, kv := range res.Entries {
		switch kv.Key {
		// Trace well-known columns
		case columnPathRootServiceName:
			spansetSeries.RootServiceName = kv.Value.String()
		case columnPathRootSpanName:
			spansetSeries.RootSpanName = kv.Value.String()
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
