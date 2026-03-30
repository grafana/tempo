package trace

import (
	"testing"

	"github.com/grafana/tempo/pkg/tempopb"
	commonv1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
	resourcev1 "github.com/grafana/tempo/pkg/tempopb/resource/v1"
	tracev1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/stretchr/testify/assert"
)

func TestSortTrace(t *testing.T) {
	tests := []struct {
		input    *tempopb.Trace
		expected *tempopb.Trace
	}{
		{
			input:    &tempopb.Trace{},
			expected: &tempopb.Trace{},
		},
		{
			input: testTraceForSorting(),
			expected: &tempopb.Trace{
				ResourceSpans: []*tracev1.ResourceSpans{
					{
						Resource: &resourcev1.Resource{
							Attributes: []*commonv1.KeyValue{
								{Key: "a", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "VA"}}},
								{Key: "b", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "VB"}}},
							},
						},
						ScopeSpans: []*tracev1.ScopeSpans{
							{
								Scope: &commonv1.InstrumentationScope{
									Name: "scope2",
									Attributes: []*commonv1.KeyValue{
										{Key: "a", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "VA"}}},
										{Key: "b", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "VB"}}},
									},
								},
								Spans: []*tracev1.Span{
									{
										StartTimeUnixNano: 1,
										Attributes: []*commonv1.KeyValue{
											{Key: "a", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "VA"}}},
											{Key: "b", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "VB"}}},
										},
										Events: []*tracev1.Span_Event{
											{
												TimeUnixNano: 11,
												Name:         "eventA2",
												Attributes: []*commonv1.KeyValue{
													{Key: "a", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "VA"}}},
													{Key: "b", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "VB"}}},
												},
											},
											{
												TimeUnixNano: 21,
												Name:         "eventB2",
												Attributes: []*commonv1.KeyValue{
													{Key: "a", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "VA"}}},
													{Key: "b", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "VB"}}},
												},
											},
										},
										Links: []*tracev1.Span_Link{
											{
												TraceId: []byte{0x11},
												SpanId:  []byte{0x11},
												Attributes: []*commonv1.KeyValue{
													{Key: "a", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "VA"}}},
													{Key: "b", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "VB"}}},
												},
											},
											{
												TraceId: []byte{0x12},
												SpanId:  []byte{0x12},
												Attributes: []*commonv1.KeyValue{
													{Key: "a", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "VA"}}},
													{Key: "b", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "VB"}}},
												},
											},
										},
									},
								},
							},
						},
					},
					{
						Resource: nil, // For testing of nil resource handling.
						ScopeSpans: []*tracev1.ScopeSpans{
							{
								Scope: &commonv1.InstrumentationScope{
									Name: "scope1",
									Attributes: []*commonv1.KeyValue{
										{Key: "a", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "va"}}},
										{Key: "b", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "vb"}}},
									},
								},
								Spans: []*tracev1.Span{
									{
										StartTimeUnixNano: 2,
										Attributes: []*commonv1.KeyValue{
											{Key: "a", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "va"}}},
											{Key: "b", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "vb"}}},
										},
										Events: []*tracev1.Span_Event{
											{
												TimeUnixNano: 10,
												Name:         "eventA",
												Attributes: []*commonv1.KeyValue{
													{Key: "a", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "va"}}},
													{Key: "b", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "vb"}}},
												},
											},
											{
												TimeUnixNano: 20,
												Name:         "eventB",
												Attributes: []*commonv1.KeyValue{
													{Key: "a", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "va"}}},
													{Key: "b", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "vb"}}},
												},
											},
										},
										Links: []*tracev1.Span_Link{
											{
												TraceId: []byte{0x01},
												SpanId:  []byte{0x01},
												Attributes: []*commonv1.KeyValue{
													{Key: "a", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "va"}}},
													{Key: "b", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "vb"}}},
												},
											},
											{
												TraceId: []byte{0x02},
												SpanId:  []byte{0x02},
												Attributes: []*commonv1.KeyValue{
													{Key: "a", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "va"}}},
													{Key: "b", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "vb"}}},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		SortTraceAndAttributes(tt.input)
		assert.Equal(t, tt.expected, tt.input)
	}
}

func testTraceForSorting() *tempopb.Trace {
	return &tempopb.Trace{
		ResourceSpans: []*tracev1.ResourceSpans{
			{
				Resource: nil, // For testing of nil resource handling.
				ScopeSpans: []*tracev1.ScopeSpans{
					{
						Scope: &commonv1.InstrumentationScope{
							Name: "scope1",
							Attributes: []*commonv1.KeyValue{
								{Key: "b", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "vb"}}},
								{Key: "a", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "va"}}},
							},
						},
						Spans: []*tracev1.Span{
							{
								StartTimeUnixNano: 2,
								Attributes: []*commonv1.KeyValue{
									{Key: "b", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "vb"}}},
									{Key: "a", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "va"}}},
								},
								Events: []*tracev1.Span_Event{
									{
										TimeUnixNano: 20,
										Name:         "eventB",
										Attributes: []*commonv1.KeyValue{
											{Key: "b", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "vb"}}},
											{Key: "a", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "va"}}},
										},
									},
									{
										TimeUnixNano: 10,
										Name:         "eventA",
										Attributes: []*commonv1.KeyValue{
											{Key: "b", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "vb"}}},
											{Key: "a", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "va"}}},
										},
									},
								},
								Links: []*tracev1.Span_Link{
									{
										TraceId: []byte{0x02},
										SpanId:  []byte{0x02},
										Attributes: []*commonv1.KeyValue{
											{Key: "b", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "vb"}}},
											{Key: "a", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "va"}}},
										},
									},
									{
										TraceId: []byte{0x01},
										SpanId:  []byte{0x01},
										Attributes: []*commonv1.KeyValue{
											{Key: "b", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "vb"}}},
											{Key: "a", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "va"}}},
										},
									},
								},
							},
						},
					},
				},
			},
			{
				Resource: &resourcev1.Resource{
					Attributes: []*commonv1.KeyValue{
						{Key: "b", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "VB"}}},
						{Key: "a", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "VA"}}},
					},
				},
				ScopeSpans: []*tracev1.ScopeSpans{
					{
						Scope: &commonv1.InstrumentationScope{
							Name: "scope2",
							Attributes: []*commonv1.KeyValue{
								{Key: "b", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "VB"}}},
								{Key: "a", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "VA"}}},
							},
						},
						Spans: []*tracev1.Span{
							{
								StartTimeUnixNano: 1,
								Attributes: []*commonv1.KeyValue{
									{Key: "b", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "VB"}}},
									{Key: "a", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "VA"}}},
								},
								Events: []*tracev1.Span_Event{
									{
										TimeUnixNano: 21,
										Name:         "eventB2",
										Attributes: []*commonv1.KeyValue{
											{Key: "b", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "VB"}}},
											{Key: "a", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "VA"}}},
										},
									},
									{
										TimeUnixNano: 11,
										Name:         "eventA2",
										Attributes: []*commonv1.KeyValue{
											{Key: "b", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "VB"}}},
											{Key: "a", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "VA"}}},
										},
									},
								},
								Links: []*tracev1.Span_Link{
									{
										TraceId: []byte{0x12},
										SpanId:  []byte{0x12},
										Attributes: []*commonv1.KeyValue{
											{Key: "b", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "VB"}}},
											{Key: "a", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "VA"}}},
										},
									},
									{
										TraceId: []byte{0x11},
										SpanId:  []byte{0x11},
										Attributes: []*commonv1.KeyValue{
											{Key: "b", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "VB"}}},
											{Key: "a", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "VA"}}},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func BenchmarkSortTraceAndAttributes(b *testing.B) {
	tr := testTraceForSorting()
	for i := 0; i < b.N; i++ {
		SortTraceAndAttributes(tr)
	}
}
