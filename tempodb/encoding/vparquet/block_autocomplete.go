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

	// seriesColumns := map[string]bool{
	// 	columnPathSpanID:             true,
	// 	columnPathSpanName:           true,
	// 	columnPathSpanStartTime:      true,
	// 	columnPathSpanEndTime:        true,
	// 	columnPathSpanStatusCode:     true,
	// 	columnPathSpanHTTPStatusCode: true,
	// 	columnPathSpanHTTPMethod:     true,
	// 	columnPathSpanHTTPURL:        true,
	// }

	makeIter := makeIterFunc(ctx, pf.RowGroups(), pf)

	var (
		columnSelectAs    = map[string]string{}
		columnPredicates  = map[string][]parquetquery.Predicate{}
		genericConditions []traceql.Condition
		spanIters         []parquetquery.Iterator
	)

	addPredicate := func(columnPath string, p parquetquery.Predicate) {
		columnPredicates[columnPath] = append(columnPredicates[columnPath], p)
	}

	for _, spanCond := range spanConditions {
		// Well-known selector?
		if entry, ok := wellKnownColumnLookups[spanCond.Attribute.Name]; ok && entry.level != traceql.AttributeScopeSpan {
			if spanCond.Op == traceql.OpNone {
				addPredicate(entry.columnPath, nil) // No filtering
				columnSelectAs[entry.columnPath] = spanCond.Attribute.Name
				continue
			}

			// Compatible type?
			if entry.typ == operandType(spanCond.Operands) {
				pred, err := createPredicate(spanCond.Op, spanCond.Operands)
				if err != nil {
					return nil, errors.Wrap(err, "creating predicate")
				}
				spanIters = append(spanIters, makeIter(entry.columnPath, pred, spanCond.Attribute.Name))
				continue
			}
		}
		// Else: generic attribute lookup
		genericConditions = append(genericConditions, spanCond)
	}

	// Span attributes iterator
	// TODO(mapno): Use a different parquetquery.GroupPredicate that doesn't collect values
	spanAttributesIter, err := createAttributeIterator(makeIter, genericConditions, DefinitionLevelResourceSpansILSSpanAttrs,
		columnPathResourceAttrKey, columnPathResourceAttrString, columnPathResourceAttrInt, columnPathResourceAttrDouble, columnPathResourceAttrBool)
	if err != nil {
		return nil, errors.Wrap(err, "creating span attribute iterator")
	}

	if len(spanIters) > 0 {
		spanIters = append(spanIters, spanAttributesIter)
	}
	for columnPath, predicates := range columnPredicates {
		spanIters = append(spanIters, makeIter(columnPath, parquetquery.NewOrPredicate(predicates...), columnSelectAs[columnPath]))
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

	spanIters = append(spanIters, makeIter(columnPathSpanStartTime, startFilter, columnPathSpanStartTime))
	spanIters = append(spanIters, makeIter(columnPathSpanEndTime, endFilter, columnPathSpanEndTime))

	groupPredicate := &spansetSeriesCollector{}
	// Scope span iterator
	// TODO(mapno): Use left join
	spanJoinIterator := parquetquery.NewJoinIterator(DefinitionLevelResourceSpansILSSpan, spanIters, groupPredicate)

	// Reset vars
	columnSelectAs = map[string]string{}
	columnPredicates = map[string][]parquetquery.Predicate{}
	genericConditions = []traceql.Condition{}
	resourceIters := []parquetquery.Iterator{}

	for _, cond := range resourceConditions {

		// Well-known selector?
		if entry, ok := wellKnownColumnLookups[cond.Attribute.Name]; ok && entry.level != traceql.AttributeScopeSpan {
			if cond.Op == traceql.OpNone {
				addPredicate(entry.columnPath, nil) // No filtering
				columnSelectAs[entry.columnPath] = cond.Attribute.Name
				continue
			}

			// Compatible type?
			if entry.typ == operandType(cond.Operands) {
				pred, err := createPredicate(cond.Op, cond.Operands)
				if err != nil {
					return nil, errors.Wrap(err, "creating predicate")
				}
				resourceIters = append(resourceIters, makeIter(entry.columnPath, pred, cond.Attribute.Name))
				continue
			}
		}

		// Else: generic attribute lookup
		genericConditions = append(genericConditions, cond)
	}

	resourceAttributesIter, err := createAttributeIterator(makeIter, genericConditions, DefinitionLevelResourceAttrs,
		columnPathResourceAttrKey, columnPathResourceAttrString, columnPathResourceAttrInt, columnPathResourceAttrDouble, columnPathResourceAttrBool)
	if err != nil {
		return nil, errors.Wrap(err, "creating span attribute iterator")
	}
	if resourceAttributesIter != nil {
		resourceIters = append(resourceIters, resourceAttributesIter)
	}

	// Put span iterator last, so it is only read when
	// the resource conditions are met.
	resourceIters = append(resourceIters, spanJoinIterator)

	// Resource spans iterator
	// TODO(mapno): Use left join
	resourceIter := parquetquery.NewJoinIterator(DefinitionLevelResourceSpans, resourceIters, groupPredicate)

	traceIters := []parquetquery.Iterator{
		resourceIter,
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
	traceIter := parquetquery.NewJoinIterator(DefinitionLevelTrace, traceIters, groupPredicate)

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
