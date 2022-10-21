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

	rootExpr, err := e.parseQuery(searchReq)
	if err != nil {
		return nil, err
	}

	fetchSpansRequest := e.createFetchSpansRequest(searchReq, rootExpr.Pipeline)

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

iter:
	for {
		spanSet, err := iterator.Next(ctx)
		if err != nil {
			span.LogKV("msg", "iterator.Next", "err", err)
			return nil, err
		}

		if spanSet == nil {
			break
		}

		ss, err := rootExpr.Pipeline.evaluate([]Spanset{*spanSet})
		if err != nil {
			span.LogKV("msg", "pipeline.evaluate", "err", err)
			continue
		}

		if len(ss) == 0 {
			continue
		}

		for _, spanSet := range ss {
			traceSearchMetadata, err := e.asTraceSearchMetadata(spanSet)
			if err != nil {
				return nil, err
			}
			res.Traces = append(res.Traces, traceSearchMetadata)

			if len(res.Traces) == int(searchReq.Limit) {
				break iter
			}
		}
	}

	span.SetTag("spansets found", len(res.Traces))

	return res, nil
}

func (e *Engine) parseQuery(searchReq *tempopb.SearchRequest) (*RootExpr, error) {
	ast, err := Parse(searchReq.Query)
	if err != nil {
		// TODO parsing "{}" returns an error, this is a hacky solution but will fail on other valid queries like "{ }"
		if searchReq.Query == "{}" {
			return &RootExpr{Pipeline: Pipeline{[]pipelineElement{}}}, nil
		}
		return nil, err
	}

	return ast, nil
}

// createFetchSpansRequest will flatten the SpansetFilter in simple conditions the storage layer
// can work with.
func (e *Engine) createFetchSpansRequest(searchReq *tempopb.SearchRequest, pipeline Pipeline) FetchSpansRequest {
	// TODO handle SearchRequest.MinDurationMs and MaxDurationMs, this refers to the trace level duration which is not the same as the intrinsic duration

	req := FetchSpansRequest{
		StartTimeUnixNanos: unixMilliToNano(searchReq.Start),
		EndTimeUnixNanos:   unixMilliToNano(searchReq.End),
		Conditions:         nil,
		AllConditions:      true,
	}

	pipeline.extractConditions(&req)
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

func (e *Engine) asTraceSearchMetadata(spanset Spanset) (*tempopb.TraceSearchMetadata, error) {
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
