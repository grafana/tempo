package traceql

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"time"

	"github.com/opentracing/opentracing-go"

	"github.com/grafana/tempo/pkg/tempopb"
	common_v1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
	"github.com/grafana/tempo/pkg/util"
)

const (
	DefaultSpansPerSpanSet int = 3
)

type Engine struct{}

func NewEngine() *Engine {
	return &Engine{}
}

func (e *Engine) Compile(query string) (func(input []*Spanset) (result []*Spanset, err error), *FetchSpansRequest, error) {
	expr, err := Parse(query)
	if err != nil {
		return nil, nil, err
	}

	req := &FetchSpansRequest{
		AllConditions: true,
	}
	expr.Pipeline.extractConditions(req)

	return expr.Pipeline.evaluate, req, nil
}

func (e *Engine) ExecuteSearch(ctx context.Context, searchReq *tempopb.SearchRequest, spanSetFetcher SpansetFetcher) (*tempopb.SearchResponse, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "traceql.Engine.ExecuteSearch")
	defer span.Finish()

	rootExpr, err := e.parseQuery(searchReq)
	if err != nil {
		return nil, err
	}

	fetchSpansRequest := e.createFetchSpansRequest(searchReq, rootExpr.Pipeline)

	span.SetTag("pipeline", rootExpr.Pipeline)
	span.SetTag("fetchSpansRequest", fetchSpansRequest)

	// calculate search meta conditions.
	metaConditions := SearchMetaConditionsWithout(fetchSpansRequest.Conditions)

	spansetsEvaluated := 0
	// set up the expression evaluation as a filter to reduce data pulled
	fetchSpansRequest.SecondPassConditions = append(fetchSpansRequest.SecondPassConditions, metaConditions...)
	fetchSpansRequest.SecondPass = func(inSS *Spanset) ([]*Spanset, error) {
		if len(inSS.Spans) == 0 {
			return nil, nil
		}

		evalSS, err := rootExpr.Pipeline.evaluate([]*Spanset{inSS})
		if err != nil {
			span.LogKV("msg", "pipeline.evaluate", "err", err)
			return nil, err
		}

		spansetsEvaluated++
		if len(evalSS) == 0 {
			return nil, nil
		}

		// reduce all evalSS to their max length to reduce meta data lookups
		for i := range evalSS {
			l := len(evalSS[i].Spans)
			evalSS[i].AddAttribute(attributeMatched, NewStaticInt(l))

			spansPerSpanSet := int(searchReq.SpansPerSpanSet)
			if spansPerSpanSet == 0 {
				spansPerSpanSet = DefaultSpansPerSpanSet
			}
			if l > spansPerSpanSet {
				evalSS[i].Spans = evalSS[i].Spans[:spansPerSpanSet]
			}
		}

		return evalSS, nil
	}

	fetchSpansResponse, err := spanSetFetcher.Fetch(ctx, fetchSpansRequest)
	if err != nil {
		return nil, err
	}
	iterator := fetchSpansResponse.Results
	defer iterator.Close()

	res := &tempopb.SearchResponse{
		Traces:  nil,
		Metrics: &tempopb.SearchMetrics{},
	}
	combiner := NewMetadataCombiner()
	for {
		spanset, err := iterator.Next(ctx)
		if err != nil && !errors.Is(err, io.EOF) {
			span.LogKV("msg", "iterator.Next", "err", err)
			return nil, err
		}
		if spanset == nil {
			break
		}
		combiner.AddMetadata(e.asTraceSearchMetadata(spanset))

		if combiner.Count() >= int(searchReq.Limit) && searchReq.Limit > 0 {
			break
		}
	}
	res.Traces = combiner.Metadata()

	span.SetTag("spansets_evaluated", spansetsEvaluated)
	span.SetTag("spansets_found", len(res.Traces))

	// Bytes can be nil when callback is no set
	if fetchSpansResponse.Bytes != nil {
		// InspectedBytes is used to compute query throughput and SLO metrics
		res.Metrics.InspectedBytes = fetchSpansResponse.Bytes()
		span.SetTag("inspectedBytes", res.Metrics.InspectedBytes)
	}

	return res, nil
}

func (e *Engine) ExecuteTagValues(
	ctx context.Context,
	tag Attribute,
	query string,
	cb func(v Static) bool,
	fetcher SpansetFetcher,
) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "traceql.Engine.ExecuteTagValues")
	defer span.Finish()

	span.SetTag("sanitized query", query)

	rootExpr, err := Parse(query)
	if err != nil {
		return err
	}
	if err := rootExpr.validate(); err != nil {
		return err
	}

	searchReq := &tempopb.SearchRequest{
		Start: 0, // TODO: Should add Start and End
		End:   math.MaxUint32,
	}

	fetchSpansRequest := e.createFetchSpansRequest(searchReq, rootExpr.Pipeline)
	// TODO: remove other conditions for the wantAttr we're searching for
	// for _, cond := range fetchSpansRequest.Conditions {
	// 	if cond.Attribute == wantAttr {
	// 		return fmt.Errorf("cannot search for tag values for tag that is already used in query")
	// 	}
	// }
	fetchSpansRequest.Conditions = append(fetchSpansRequest.Conditions, Condition{
		Attribute: tag,
		Op:        OpNone,
	})

	span.SetTag("pipeline", rootExpr.Pipeline)
	span.SetTag("fetchSpansRequest", fetchSpansRequest)

	var collectAttributeValue func(s Span) bool
	switch tag.Scope {
	case AttributeScopeResource,
		AttributeScopeSpan: // If tag is scoped, we can check the map directly
		collectAttributeValue = func(s Span) bool {
			if v, ok := s.Attributes()[tag]; ok {
				return cb(v)
			}
			return false
		}
	case AttributeScopeNone:
		// If tag is unscoped, it can either be an intrinsic (eg. `name`) or an unscoped attribute (eg. `.namespace`)
		//
		// If the tag is intrinsic Attribute.Intrinsic is set to the Intrinsic it corresponds,
		// so we can check against `!= IntrinsicNone` and use tag directly.
		//
		// If the tag is unscoped, we need to check resource and span scoped manually by building a new Attribute with each scope.
		collectAttributeValue = func(s Span) bool {
			if tag.Intrinsic != IntrinsicNone { // it's intrinsic
				if v, ok := s.Attributes()[tag]; ok {
					return cb(v)
				}
			} else { // it's unscoped
				for _, scope := range []AttributeScope{AttributeScopeResource, AttributeScopeSpan} {
					scopedAttr := Attribute{Scope: scope, Parent: tag.Parent, Name: tag.Name}
					if v, ok := s.Attributes()[scopedAttr]; ok {
						return cb(v)
					}
				}
			}

			return false
		}
	default:
		return fmt.Errorf("unknown attribute scope: %s", tag)
	}

	fetchSpansResponse, err := fetcher.Fetch(ctx, fetchSpansRequest)
	if err != nil {
		return err
	}
	iterator := fetchSpansResponse.Results
	defer iterator.Close()

	for {
		spanset, err := iterator.Next(ctx)
		if err != nil && !errors.Is(err, io.EOF) {
			span.LogKV("msg", "iterator.Next", "err", err)
			return err
		}
		if spanset == nil {
			break
		}
		if len(spanset.Spans) == 0 {
			continue
		}

		evalSS, err := rootExpr.Pipeline.evaluate([]*Spanset{spanset})
		if err != nil {
			span.LogKV("msg", "pipeline.evaluate", "err", err)
			return err
		}

		if len(evalSS) == 0 {
			continue
		}

		for _, ss := range evalSS {
			for _, s := range ss.Spans {
				if collectAttributeValue(s) {
					return nil // exit early if we've exceed max bytes
				}
			}
		}

	}

	return nil
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

func (e *Engine) asTraceSearchMetadata(spanset *Spanset) *tempopb.TraceSearchMetadata {
	metadata := &tempopb.TraceSearchMetadata{
		TraceID:           util.TraceIDToHexString(spanset.TraceID),
		RootServiceName:   spanset.RootServiceName,
		RootTraceName:     spanset.RootSpanName,
		StartTimeUnixNano: spanset.StartTimeUnixNanos,
		DurationMs:        uint32(spanset.DurationNanos / 1_000_000),
		SpanSet:           &tempopb.SpanSet{},
	}

	for _, span := range spanset.Spans {
		tempopbSpan := &tempopb.Span{
			SpanID:            util.SpanIDToHexString(span.ID()),
			StartTimeUnixNano: span.StartTimeUnixNanos(),
			DurationNanos:     span.DurationNanos(),
			Attributes:        nil,
		}

		atts := span.Attributes()

		if name, ok := atts[NewIntrinsic(IntrinsicName)]; ok {
			tempopbSpan.Name = name.S
		}

		for attribute, static := range atts {
			if attribute.Intrinsic == IntrinsicName ||
				attribute.Intrinsic == IntrinsicDuration ||
				attribute.Intrinsic == IntrinsicTraceDuration ||
				attribute.Intrinsic == IntrinsicTraceRootService ||
				attribute.Intrinsic == IntrinsicTraceRootSpan {
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
	}

	// create a new slice and add the spanset to it. eventually we will deprecate
	//  metadata.SpanSet. populating both the SpanSet and the []SpanSets is for
	//  backwards compatibility with Grafana. since this method only translates one
	//  spanset into a TraceSearchMetadata Spansets[0] == Spanset. Higher up the chain
	//  we will combine Spansets with the same trace id.
	metadata.SpanSets = []*tempopb.SpanSet{metadata.SpanSet}

	// add attributes
	for _, att := range spanset.Attributes {
		if att.Name == attributeMatched {
			metadata.SpanSet.Matched = uint32(att.Val.N)
			continue
		}

		staticAnyValue := att.Val.asAnyValue()
		keyValue := &common_v1.KeyValue{
			Key:   att.Name,
			Value: staticAnyValue,
		}
		metadata.SpanSet.Attributes = append(metadata.SpanSet.Attributes, keyValue)
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
	case TypeKind:
		return &common_v1.AnyValue{
			Value: &common_v1.AnyValue_StringValue{
				StringValue: s.Kind.String(),
			},
		}
	}

	return &common_v1.AnyValue{
		Value: &common_v1.AnyValue_StringValue{
			StringValue: fmt.Sprintf("error formatting val: static has unexpected type %v", s.Type),
		},
	}
}
