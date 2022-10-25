package traceql

import (
	"context"
	"fmt"

	"github.com/opentracing/opentracing-go"

	"github.com/grafana/tempo/pkg/tempopb"
	common_v1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
	"github.com/grafana/tempo/pkg/util"
)

type Engine struct {
	spansPerSpanSet int
}

func NewEngine() *Engine {
	return &Engine{
		spansPerSpanSet: 3, // TODO make configurable
	}
}

func (e *Engine) Execute(ctx context.Context, searchReq *tempopb.SearchRequest, spanSetFetcher SpansetFetcher) (*tempopb.SearchResponse, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "traceql.Engine.Execute")
	defer span.Finish()

	// TODO this engine implementation assumes each query will contain exactly and at most one SpansetFilter, these queries
	//  can be processed in a single pass. When we deal with more complicated queries we will probably have to do multiple
	//  passes and this implementation will become a subcomponent of that.

	spanSetFilter, err := e.parseQueryAndExtractSpanSetFilter(searchReq)
	if err != nil {
		return nil, err
	}

	fetchSpansRequest := e.createFetchSpansRequest(searchReq, spanSetFilter)

	span.SetTag("fetchSpansRequest", fetchSpansRequest)

	fetchSpansResponse, err := spanSetFetcher.Fetch(ctx, fetchSpansRequest)
	if err != nil {
		return nil, err
	}
	iterator := fetchSpansResponse.Results

	res := &tempopb.SearchResponse{
		Traces: nil,
		// TODO capture and update metrics
		Metrics: &tempopb.SearchMetrics{},
	}

	for {
		spanSet, err := iterator.Next(ctx)
		if err != nil {
			span.LogKV("msg", "iterator.Next", "err", err)
			return nil, err
		}

		if spanSet == nil {
			break
		}

		span.LogKV("msg", "iterator.Next", "rootSpanName", spanSet.RootSpanName, "rootServiceName", spanSet.RootServiceName, "spans", len(spanSet.Spans))

		spanSet = e.validateSpanSet(spanSetFilter, spanSet)
		if spanSet == nil {
			continue
		}

		span.LogKV("msg", "validateSpanSet", "spans", len(spanSet.Spans))

		traceSearchMetadata, err := e.asTraceSearchMetadata(spanSet)
		if err != nil {
			return nil, err
		}
		res.Traces = append(res.Traces, traceSearchMetadata)

		if len(res.Traces) == int(searchReq.Limit) {
			break
		}
	}

	span.SetTag("traces_found", len(res.Traces))

	return res, nil
}

func (e *Engine) parseQueryAndExtractSpanSetFilter(searchReq *tempopb.SearchRequest) (*SpansetFilter, error) {
	// Parse TraceQL query
	ast, err := Parse(searchReq.Query)
	if err != nil {
		// TODO parsing "{}" returns an error, this is a hacky solution but will fail on other valid queries like "{ }"
		if searchReq.Query == "{}" {
			return &SpansetFilter{Expression: NewStaticBool(true)}, nil
		}
		return nil, err
	}

	if len(ast.Pipeline.Elements) != 1 {
		return nil, fmt.Errorf("queries with multiple pipeline elements aren't supported yet")
	}

	element := ast.Pipeline.Elements[0]

	spanSetFilter, ok := element.(SpansetFilter)
	if !ok {
		return nil, fmt.Errorf("queries with %T are not supported yet", element)
	}

	return &spanSetFilter, err
}

// createFetchSpansRequest will flatten the SpansetFilter in simple conditions the storage layer
// can work with.
func (e *Engine) createFetchSpansRequest(searchReq *tempopb.SearchRequest, spanSetFilter *SpansetFilter) FetchSpansRequest {
	// TODO handle SearchRequest.MinDurationMs and MaxDurationMs, this refers to the trace level duration which is not the same as the intrinsic duration

	req := FetchSpansRequest{
		StartTimeUnixNanos: unixMilliToNano(searchReq.Start),
		EndTimeUnixNanos:   unixMilliToNano(searchReq.End),
		Conditions:         nil,
		AllConditions:      true,
	}
	spanSetFilter.extractConditions(&req)
	return req
}

// validateSpanSet will validate the Spanset fulfills the SpansetFilter.
func (e *Engine) validateSpanSet(spanSetFilter *SpansetFilter, spanSet *Spanset) *Spanset {
	newSpanSet := &Spanset{
		TraceID:         spanSet.TraceID,
		RootSpanName:    spanSet.RootSpanName,
		RootServiceName: spanSet.RootServiceName,
		Spans:           nil,
	}

	for _, span := range spanSet.Spans {
		matches, _ := spanSetFilter.matches(span)
		if !matches {
			continue
		}

		newSpanSet.Spans = append(newSpanSet.Spans, span)
	}

	if len(newSpanSet.Spans) == 0 {
		return nil
	}

	return newSpanSet
}

func (e *Engine) asTraceSearchMetadata(spanset *Spanset) (*tempopb.TraceSearchMetadata, error) {
	metadata := &tempopb.TraceSearchMetadata{
		TraceID:           util.TraceIDToHexString(spanset.TraceID),
		RootServiceName:   spanset.RootServiceName,
		RootTraceName:     spanset.RootSpanName,
		StartTimeUnixNano: spanset.StartTimeUnixNanos,
		DurationMs:        uint32(spanset.DurationNanos / 1_000_000),
		SpanSet: &tempopb.SpanSet{
			Matched: uint32(len(spanset.Spans)),
		},
	}

	for _, span := range spanset.Spans {
		tempopbSpan := &tempopb.Span{
			SpanID:            util.TraceIDToHexString(span.ID),
			StartTimeUnixNano: span.StartTimeUnixNanos,
			DurationNanos:     span.EndtimeUnixNanos - span.StartTimeUnixNanos,
			Attributes:        nil,
		}

		for attribute, static := range span.Attributes {
			staticAnyValue, err := asAnyValue(static)
			if err != nil {
				return nil, err
			}

			keyValue := &common_v1.KeyValue{
				Key:   attribute.Name,
				Value: staticAnyValue,
			}

			tempopbSpan.Attributes = append(tempopbSpan.Attributes, keyValue)
		}

		metadata.SpanSet.Spans = append(metadata.SpanSet.Spans, tempopbSpan)

		if e.spansPerSpanSet != 0 && len(metadata.SpanSet.Spans) == e.spansPerSpanSet {
			break
		}
	}

	return metadata, nil
}

func unixMilliToNano(ts uint32) uint64 {
	return uint64(ts) * 1000
}

func asAnyValue(static Static) (*common_v1.AnyValue, error) {
	switch static.Type {
	case TypeInt:
		return &common_v1.AnyValue{
			Value: &common_v1.AnyValue_IntValue{
				IntValue: int64(static.N),
			},
		}, nil
	case TypeString:
		return &common_v1.AnyValue{
			Value: &common_v1.AnyValue_StringValue{
				StringValue: static.S,
			},
		}, nil
	case TypeFloat:
		return &common_v1.AnyValue{
			Value: &common_v1.AnyValue_DoubleValue{
				DoubleValue: static.F,
			},
		}, nil
	case TypeBoolean:
		return &common_v1.AnyValue{
			Value: &common_v1.AnyValue_BoolValue{
				BoolValue: static.B,
			},
		}, nil
	case TypeDuration:
		return &common_v1.AnyValue{
			Value: &common_v1.AnyValue_StringValue{
				StringValue: static.D.String(),
			},
		}, nil
	case TypeStatus:
		return &common_v1.AnyValue{
			Value: &common_v1.AnyValue_StringValue{
				StringValue: static.Status.String(),
			},
		}, nil
	default:
		return nil, fmt.Errorf("static has unexpected type %v", static.Type)
	}
}
