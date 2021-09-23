package distributor

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/grafana/tempo/pkg/tempofb"
	"github.com/grafana/tempo/pkg/tempopb"
	v1_common "github.com/grafana/tempo/pkg/tempopb/common/v1"
	v1_resource "github.com/grafana/tempo/pkg/tempopb/resource/v1"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/tempodb/search"
)

func TestExtractSearchData(t *testing.T) {
	traceIDA := []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F}

	testCases := []struct {
		name       string
		trace      *tempopb.Trace
		id         []byte
		extractTag extractTagFunc
		searchData *tempofb.SearchEntryMutable
	}{
		{
			name: "extracts search tags",
			trace: &tempopb.Trace{
				Batches: []*v1.ResourceSpans{
					{
						Resource: &v1_resource.Resource{
							Attributes: []*v1_common.KeyValue{
								{
									Key: "foo",
									Value: &v1_common.AnyValue{
										Value: &v1_common.AnyValue_StringValue{StringValue: "bar"},
									},
								},
								{
									Key: "service.name",
									Value: &v1_common.AnyValue{
										Value: &v1_common.AnyValue_StringValue{StringValue: "baz"},
									},
								},
							},
						},
						InstrumentationLibrarySpans: []*v1.InstrumentationLibrarySpans{
							{
								InstrumentationLibrary: &v1_common.InstrumentationLibrary{
									Name: "test",
								},
								Spans: []*v1.Span{
									{
										TraceId: traceIDA,
										Name:    "firstSpan",
									},
								},
							},
						},
					},
				},
			},
			id: traceIDA,
			searchData: &tempofb.SearchEntryMutable{
				TraceID: traceIDA,
				Tags: tempofb.SearchDataMap{
					"foo":                         []string{"bar"},
					search.RootSpanPrefix + "foo": []string{"bar"},
					search.RootSpanNameTag:        []string{"firstSpan"},
					search.SpanNameTag:            []string{"firstSpan"},
					search.RootServiceNameTag:     []string{"baz"},
					search.ServiceNameTag:         []string{"baz"},
				},
				StartTimeUnixNano: 0,
				EndTimeUnixNano:   0,
			},
			extractTag: func(tag string) bool {
				return true
			},
		},
		{
			name: "drops tags in deny list",
			trace: &tempopb.Trace{
				Batches: []*v1.ResourceSpans{
					{
						Resource: &v1_resource.Resource{
							Attributes: []*v1_common.KeyValue{
								{
									Key: "foo",
									Value: &v1_common.AnyValue{
										Value: &v1_common.AnyValue_StringValue{StringValue: "bar"},
									},
								},
								{
									Key: "bar",
									Value: &v1_common.AnyValue{
										Value: &v1_common.AnyValue_StringValue{StringValue: "baz"},
									},
								},
							},
						},
					},
				},
			},
			id: traceIDA,
			searchData: &tempofb.SearchEntryMutable{
				TraceID: traceIDA,
				Tags: tempofb.SearchDataMap{
					"bar": []string{"baz"},
				},
				StartTimeUnixNano: 0,
				EndTimeUnixNano:   0,
			},
			extractTag: func(tag string) bool {
				return tag != "foo"
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.searchData.ToBytes(), extractSearchData(tc.trace, tc.id, tc.extractTag))
		})
	}
}
