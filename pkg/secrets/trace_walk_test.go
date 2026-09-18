package secrets

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/grafana/tempo/pkg/tempopb"
	commonv1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
	resourcev1 "github.com/grafana/tempo/pkg/tempopb/resource/v1"
	tracev1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
)

func TestWalkPushSpansRequestVisitsEveryTraceValueClass(t *testing.T) {
	request := &tempopb.PushSpansRequest{Batches: []*tracev1.ResourceSpans{{
		SchemaUrl: "resource-schema-secret",
		Resource: &resourcev1.Resource{
			Attributes: []*commonv1.KeyValue{
				{Key: "resource.key", Value: stringValue("resource-secret")},
				{Key: "password", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_IntValue{IntValue: 42}}},
			},
			EntityRefs: []*commonv1.EntityRef{{SchemaUrl: "entity-schema-secret", Type: "entity-type-secret", IdKeys: []string{"entity-id-secret"}, DescriptionKeys: []string{"entity-description-secret"}}},
		},
		ScopeSpans: []*tracev1.ScopeSpans{{
			SchemaUrl: "scope-schema-secret",
			Scope: &commonv1.InstrumentationScope{
				Name:       "scope-name-secret",
				Version:    "scope-version-secret",
				Attributes: []*commonv1.KeyValue{{Key: "scope.key", Value: bytesValue("scope-secret")}},
			},
			Spans: []*tracev1.Span{{
				TraceId:    []byte{1},
				SpanId:     []byte{2},
				TraceState: "trace-state-secret",
				Name:       "span-name-secret",
				Status:     &tracev1.Status{Message: "status-secret"},
				Attributes: []*commonv1.KeyValue{{Key: "span.key", Value: arrayValue(stringValue("span-secret"))}, {Key: "map.password", Value: kvValue("map.child", "event-secret")}},
				Events:     []*tracev1.Span_Event{{Name: "event-name-secret", Attributes: []*commonv1.KeyValue{{Key: "event.key", Value: kvValue("nested.key", "event-secret")}}}},
				Links:      []*tracev1.Span_Link{{TraceState: "link-state-secret", Attributes: []*commonv1.KeyValue{{Key: "link.key", Value: stringValue("link-secret")}}}},
			}},
		}},
	}}}

	var visited []string
	WalkPushSpansRequest(request, func(field TraceField) bool {
		visited = append(visited, field.Value)
		return true
	})

	assert.ElementsMatch(t, []string{
		"resource-schema-secret",
		"resource-secret",
		"42",
		"entity-schema-secret",
		"entity-type-secret",
		"scope-schema-secret",
		"scope-name-secret",
		"scope-version-secret",
		"scope-secret",
		"trace-state-secret",
		"span-name-secret",
		"status-secret",
		"span-secret",
		"event-name-secret",
		"event-secret",
		"event-secret",
		"link-state-secret",
		"link-secret",
	}, visited)
}

func TestWalkPushSpansRequestStopsWhenVisitorRejectsField(t *testing.T) {
	request := &tempopb.PushSpansRequest{Batches: []*tracev1.ResourceSpans{{
		SchemaUrl:  "first",
		ScopeSpans: []*tracev1.ScopeSpans{{SchemaUrl: "second"}},
	}}}
	visits := 0
	WalkPushSpansRequest(request, func(TraceField) bool {
		visits++
		return false
	})
	assert.Equal(t, 1, visits)
}

func stringValue(value string) *commonv1.AnyValue {
	return &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: value}}
}

func bytesValue(value string) *commonv1.AnyValue {
	return &commonv1.AnyValue{Value: &commonv1.AnyValue_BytesValue{BytesValue: []byte(value)}}
}

func arrayValue(values ...*commonv1.AnyValue) *commonv1.AnyValue {
	return &commonv1.AnyValue{Value: &commonv1.AnyValue_ArrayValue{ArrayValue: &commonv1.ArrayValue{Values: values}}}
}

func kvValue(key, value string) *commonv1.AnyValue {
	return &commonv1.AnyValue{Value: &commonv1.AnyValue_KvlistValue{KvlistValue: &commonv1.KeyValueList{Values: []*commonv1.KeyValue{{Key: key, Value: stringValue(value)}}}}}
}
