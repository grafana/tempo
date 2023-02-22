package traceql

import (
	"context"
	"fmt"
	"time"

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

	span.SetTag("pipeline", rootExpr.Pipeline)
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

	spansetsEvaluated := 0

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
			return nil, err
		}
		spansetsEvaluated++

		if len(ss) == 0 {
			continue
		}

		for _, spanSet := range ss {
			res.Traces = append(res.Traces, e.asTraceSearchMetadata(spanSet))

			if len(res.Traces) == int(searchReq.Limit) {
				break iter
			}
		}
	}

	span.SetTag("spansets_evaluated", spansetsEvaluated)
	span.SetTag("spansets_found", len(res.Traces))

	return res, nil
}

func (e *Engine) parseQuery(searchReq *tempopb.SearchRequest) (*RootExpr, error) {
	r, err := Parse(searchReq.Query)
	if err != nil {
		return nil, err
	}
	return r, r.validate()
}

// createFetchSpansRequest will flatten the SpansetFilter in simple conditions the storage layer
// can work with.
func (e *Engine) createFetchSpansRequest(searchReq *tempopb.SearchRequest, pipeline Pipeline) FetchSpansRequest {
	// TODO handle SearchRequest.MinDurationMs and MaxDurationMs, this refers to the trace level duration which is not the same as the intrinsic duration

	req := FetchSpansRequest{
		StartTimeUnixNanos: unixSecToNano(searchReq.Start),
		EndTimeUnixNanos:   unixSecToNano(searchReq.End),
		Conditions:         nil,
		AllConditions:      true,
	}

	pipeline.extractConditions(&req)
	return req
}

func (e *Engine) asTraceSearchMetadata(spanset Spanset) *tempopb.TraceSearchMetadata {
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
			SpanID:            util.SpanIDToHexString(span.ID),
			StartTimeUnixNano: span.StartTimeUnixNanos,
			DurationNanos:     span.EndtimeUnixNanos - span.StartTimeUnixNanos,
			Attributes:        nil,
		}

		if name, ok := span.Attributes[NewIntrinsic(IntrinsicName)]; ok {
			tempopbSpan.Name = name.S
		}

		for attribute, static := range span.Attributes {
			if attribute.Intrinsic == IntrinsicName || attribute.Intrinsic == IntrinsicDuration {
				continue
			}

			staticAnyValue := static.asAnyValue()

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

	return metadata
}

func unixSecToNano(ts uint32) uint64 {
	return uint64(ts) * uint64(time.Second/time.Nanosecond)
}

func (s Static) asAnyValue() *common_v1.AnyValue {
	switch s.Type {
	case TypeInt:
		return &common_v1.AnyValue{
			Value: &common_v1.AnyValue_IntValue{
				IntValue: int64(s.N),
			},
		}
	case TypeString:
		return &common_v1.AnyValue{
			Value: &common_v1.AnyValue_StringValue{
				StringValue: s.S,
			},
		}
	case TypeFloat:
		return &common_v1.AnyValue{
			Value: &common_v1.AnyValue_DoubleValue{
				DoubleValue: s.F,
			},
		}
	case TypeBoolean:
		return &common_v1.AnyValue{
			Value: &common_v1.AnyValue_BoolValue{
				BoolValue: s.B,
			},
		}
	case TypeDuration:
		return &common_v1.AnyValue{
			Value: &common_v1.AnyValue_StringValue{
				StringValue: s.D.String(),
			},
		}
	case TypeStatus:
		return &common_v1.AnyValue{
			Value: &common_v1.AnyValue_StringValue{
				StringValue: s.Status.String(),
			},
		}
	case TypeNil:
		return &common_v1.AnyValue{
			Value: &common_v1.AnyValue_StringValue{
				StringValue: "nil",
			},
		}
	}

	return &common_v1.AnyValue{
		Value: &common_v1.AnyValue_StringValue{
			StringValue: fmt.Sprintf("error formatting val: static has unexpected type %v", s.Type),
		},
	}
}
