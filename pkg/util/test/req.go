package test

import (
	crand "crypto/rand"
	"encoding/json"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/grafana/tempo/pkg/tempopb"
	v1_common "github.com/grafana/tempo/pkg/tempopb/common/v1"
	v1_resource "github.com/grafana/tempo/pkg/tempopb/resource/v1"
	v1_trace "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/stretchr/testify/require"
)

func MakeSpan(traceID []byte) *v1_trace.Span {
	return MakeSpanWithAttributeCount(traceID, rand.Int()%10+1)
}

func MakeSpanWithAttributeCount(traceID []byte, count int) *v1_trace.Span {
	attributes := make([]*v1_common.KeyValue, 0, count)
	for i := 0; i < count; i++ {
		attributes = append(attributes, &v1_common.KeyValue{
			Key:   RandomString(),
			Value: &v1_common.AnyValue{Value: &v1_common.AnyValue_StringValue{StringValue: RandomString()}},
		})
	}

	now := time.Now()
	s := &v1_trace.Span{
		Name:         "test",
		TraceId:      traceID,
		SpanId:       make([]byte, 8),
		ParentSpanId: make([]byte, 8),
		Kind:         v1_trace.Span_SPAN_KIND_CLIENT,
		Status: &v1_trace.Status{
			Code:    1,
			Message: "OK",
		},
		StartTimeUnixNano:      uint64(now.UnixNano()),
		EndTimeUnixNano:        uint64(now.Add(time.Second).UnixNano()),
		Attributes:             attributes,
		DroppedLinksCount:      rand.Uint32(),
		DroppedAttributesCount: rand.Uint32(),
	}
	_, err := crand.Read(s.SpanId)
	if err != nil {
		panic(err)
	}

	// add link
	if rand.Intn(5) == 0 {
		s.Links = append(s.Links, &v1_trace.Span_Link{
			TraceId:    traceID,
			SpanId:     make([]byte, 8),
			TraceState: "state",
			Attributes: []*v1_common.KeyValue{
				{
					Key: "linkkey",
					Value: &v1_common.AnyValue{
						Value: &v1_common.AnyValue_StringValue{
							StringValue: "linkvalue",
						},
					},
				},
			},
		})
	}

	// add attr
	if rand.Intn(2) == 0 {
		s.Attributes = append(s.Attributes, &v1_common.KeyValue{
			Key: "key",
			Value: &v1_common.AnyValue{
				Value: &v1_common.AnyValue_StringValue{
					StringValue: "value",
				},
			},
		})
	}

	// add event
	if rand.Intn(3) == 0 {
		s.Events = append(s.Events, &v1_trace.Span_Event{
			TimeUnixNano:           rand.Uint64(),
			Name:                   "event",
			DroppedAttributesCount: rand.Uint32(),
			Attributes: []*v1_common.KeyValue{
				{
					Key: "eventkey",
					Value: &v1_common.AnyValue{
						Value: &v1_common.AnyValue_StringValue{
							StringValue: "eventvalue",
						},
					},
				},
			},
		})
	}

	return s
}

func MakeBatch(spans int, traceID []byte) *v1_trace.ResourceSpans {
	traceID = ValidTraceID(traceID)

	batch := &v1_trace.ResourceSpans{
		Resource: &v1_resource.Resource{
			Attributes: []*v1_common.KeyValue{
				{
					Key: "service.name",
					Value: &v1_common.AnyValue{
						Value: &v1_common.AnyValue_StringValue{
							StringValue: "test-service",
						},
					},
				},
			},
		},
	}
	var ss *v1_trace.ScopeSpans

	for i := 0; i < spans; i++ {
		// occasionally make a new ss
		if ss == nil || rand.Int()%3 == 0 {
			ss = &v1_trace.ScopeSpans{
				Scope: &v1_common.InstrumentationScope{
					Name:    "super library",
					Version: "0.0.1",
				},
			}

			batch.ScopeSpans = append(batch.ScopeSpans, ss)
		}

		ss.Spans = append(ss.Spans, MakeSpan(traceID))

		// first span should not have parent
		if len(ss.Spans) == 1 {
			(*ss.Spans[0]).ParentSpanId = nil //[]byte{}
		}
	}
	return batch
}

func MakeTrace(requests int, traceID []byte) *tempopb.Trace {
	traceID = ValidTraceID(traceID)

	trace := &tempopb.Trace{
		Batches: make([]*v1_trace.ResourceSpans, 0),
	}

	for i := 0; i < requests; i++ {
		trace.Batches = append(trace.Batches, MakeBatch(rand.Int()%20+1, traceID))
	}

	return trace
}

func MakeTraceWithSpanCount(requests int, spansEach int, traceID []byte) *tempopb.Trace {
	trace := &tempopb.Trace{
		Batches: make([]*v1_trace.ResourceSpans, 0),
	}

	for i := 0; i < requests; i++ {
		trace.Batches = append(trace.Batches, MakeBatch(spansEach, traceID))
	}

	return trace
}

var (
	dedicatedColumnsResource = backend.DedicatedColumns{
		{Scope: "resource", Name: "dedicated.resource.1", Type: "string"},
		{Scope: "resource", Name: "dedicated.resource.2", Type: "string"},
		{Scope: "resource", Name: "dedicated.resource.3", Type: "string"},
		{Scope: "resource", Name: "dedicated.resource.4", Type: "string"},
		{Scope: "resource", Name: "dedicated.resource.5", Type: "string"},
	}
	dedicatedColumnsSpan = backend.DedicatedColumns{
		{Scope: "span", Name: "dedicated.span.1", Type: "string"},
		{Scope: "span", Name: "dedicated.span.2", Type: "string"},
		{Scope: "span", Name: "dedicated.span.3", Type: "string"},
		{Scope: "span", Name: "dedicated.span.4", Type: "string"},
		{Scope: "span", Name: "dedicated.span.5", Type: "string"},
	}
)

// AddDedicatedAttributes adds resource and span attributes to a trace that are stored in dedicated
// columns when a backend.BlockMeta is created with the column assignments from MakeDedicatedColumns.
func AddDedicatedAttributes(trace *tempopb.Trace) *tempopb.Trace {
	spanAttrs := make([]*v1_common.KeyValue, 0, len(dedicatedColumnsSpan))
	for i, c := range dedicatedColumnsSpan {
		spanAttrs = append(spanAttrs, &v1_common.KeyValue{
			Key: c.Name,
			Value: &v1_common.AnyValue{
				Value: &v1_common.AnyValue_StringValue{
					StringValue: fmt.Sprintf("dedicated-span-attr-value-%d", i+1),
				},
			},
		})
	}
	resourceAttrs := make([]*v1_common.KeyValue, 0, len(dedicatedColumnsResource))
	for i, c := range dedicatedColumnsResource {
		resourceAttrs = append(resourceAttrs, &v1_common.KeyValue{
			Key: c.Name,
			Value: &v1_common.AnyValue{
				Value: &v1_common.AnyValue_StringValue{
					StringValue: fmt.Sprintf("dedicated-resource-attr-value-%d", i+1),
				},
			},
		})
	}

	for _, batch := range trace.Batches {
		attr := make([]*v1_common.KeyValue, 0, len(resourceAttrs)+len(batch.Resource.Attributes))
		attr = append(attr, resourceAttrs...)
		batch.Resource.Attributes = append(attr, batch.Resource.Attributes...)

		for _, ss := range batch.ScopeSpans {
			for _, span := range ss.Spans {
				attr = make([]*v1_common.KeyValue, 0, len(spanAttrs)+len(span.Attributes))
				attr = append(attr, spanAttrs...)
				span.Attributes = append(attr, span.Attributes...)
			}
		}
	}

	return trace
}

// MakeDedicatedColumns creates a dedicated column assignment that matches the attributes
// generated by AddDedicatedAttributes.
func MakeDedicatedColumns() backend.DedicatedColumns {
	columns := make(backend.DedicatedColumns, 0, len(dedicatedColumnsResource)+len(dedicatedColumnsSpan))
	columns = append(columns, dedicatedColumnsResource...)
	columns = append(columns, dedicatedColumnsSpan...)
	return columns
}

func ValidTraceID(traceID []byte) []byte {
	if len(traceID) == 0 {
		traceID = make([]byte, 16)
		_, err := crand.Read(traceID)
		if err != nil {
			panic(err)
		}
	}

	for len(traceID) < 16 {
		traceID = append(traceID, 0)
	}

	return traceID
}

func RandomString() string {
	letters := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

	s := make([]rune, 10)
	for i := range s {
		s[i] = letters[rand.Intn(len(letters))]
	}
	return string(s)
}

func TracesEqual(t *testing.T, t1 *tempopb.Trace, t2 *tempopb.Trace) {
	if !proto.Equal(t1, t2) {
		wantJSON, _ := json.MarshalIndent(t1, "", "  ")
		gotJSON, _ := json.MarshalIndent(t2, "", "  ")

		require.Equal(t, string(wantJSON), string(gotJSON))
	}
}
