package vparquet

import (
	"fmt"
	"testing"

	"github.com/dustin/go-humanize"
	"github.com/segmentio/parquet-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/pkg/tempopb"
	v1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
	v1_resource "github.com/grafana/tempo/pkg/tempopb/resource/v1"
	v1_trace "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/util/test"
)

func TestProtoParquetRoundTrip(t *testing.T) {

	// This test round trips a proto trace and checks that the transformation works as expected
	// Proto -> Parquet -> Proto

	traceIDA := []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F}

	expectedTrace := &tempopb.Trace{
		Batches: []*v1_trace.ResourceSpans{
			{
				Resource: &v1_resource.Resource{
					Attributes: []*v1.KeyValue{
						{
							Key: "rs.string.key",
							Value: &v1.AnyValue{
								Value: &v1.AnyValue_StringValue{StringValue: "foo"},
							},
						},
						{
							Key: "rs.int.key",
							Value: &v1.AnyValue{
								Value: &v1.AnyValue_IntValue{IntValue: 1},
							},
						},
						{
							Key: "rs.bool.key",
							Value: &v1.AnyValue{
								Value: &v1.AnyValue_BoolValue{BoolValue: true},
							},
						},
						{
							Key: "rs.kv.key",
							Value: &v1.AnyValue{Value: &v1.AnyValue_KvlistValue{KvlistValue: &v1.KeyValueList{Values: []*v1.KeyValue{
								{Key: "s2", Value: &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: "s3"}}},
								{Key: "i2", Value: &v1.AnyValue{Value: &v1.AnyValue_IntValue{IntValue: 789}}},
							}}}},
						},
						{ // special column
							Key: "service.name",
							Value: &v1.AnyValue{
								Value: &v1.AnyValue_StringValue{StringValue: "bar"},
							},
						},
					},
				},
				ScopeSpans: []*v1_trace.ScopeSpans{
					{
						Scope: &v1.InstrumentationScope{
							Name: "test",
						},
						Spans: []*v1_trace.Span{
							{
								TraceId: traceIDA,
								SpanId:  []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
								Name:    "firstSpan",
								Kind:    v1_trace.Span_SPAN_KIND_CLIENT,
								Status: &v1_trace.Status{
									Code: v1_trace.Status_STATUS_CODE_OK,
								},
								Attributes: []*v1.KeyValue{
									{
										Key: "span.int.key",
										Value: &v1.AnyValue{
											Value: &v1.AnyValue_IntValue{IntValue: 100},
										},
									},
									{
										Key: "span.bool.key",
										Value: &v1.AnyValue{
											Value: &v1.AnyValue_BoolValue{BoolValue: true},
										},
									},
									{ // special column
										Key: "http.url",
										Value: &v1.AnyValue{
											Value: &v1.AnyValue_StringValue{StringValue: "https://grafana.com"},
										},
									},
								},
								Events: []*v1_trace.Span_Event{
									{
										// Basic fields
										TimeUnixNano:           123,
										Name:                   "456",
										DroppedAttributesCount: 789,

										// An attribute of every type
										Attributes: []*v1.KeyValue{
											// String
											{Key: "s", Value: &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: "s2"}}},

											// Int
											{Key: "i", Value: &v1.AnyValue{Value: &v1.AnyValue_IntValue{IntValue: 123}}},

											// Double
											{Key: "d", Value: &v1.AnyValue{Value: &v1.AnyValue_DoubleValue{DoubleValue: 123.456}}},

											// Bool
											{Key: "b", Value: &v1.AnyValue{Value: &v1.AnyValue_BoolValue{BoolValue: true}}},

											// KVList
											{Key: "kv", Value: &v1.AnyValue{Value: &v1.AnyValue_KvlistValue{KvlistValue: &v1.KeyValueList{Values: []*v1.KeyValue{
												{Key: "s2", Value: &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: "s3"}}},
												{Key: "i2", Value: &v1.AnyValue{Value: &v1.AnyValue_IntValue{IntValue: 789}}},
											}}}}},

											// Array
											{Key: "a", Value: &v1.AnyValue{Value: &v1.AnyValue_ArrayValue{ArrayValue: &v1.ArrayValue{Values: []*v1.AnyValue{
												{Value: &v1.AnyValue_StringValue{StringValue: "s4"}},
												{Value: &v1.AnyValue_IntValue{IntValue: 101112}},
											}}}}},
										},
									},
								},
							},
							{
								TraceId: traceIDA,
								Name:    "secondSpan",
								Status: &v1_trace.Status{
									Code: v1_trace.Status_STATUS_CODE_ERROR,
								},
								ParentSpanId: []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
							},
						},
					},
				},
			},
		},
	}

	parquetTrace := traceToParquet(traceIDA, expectedTrace)
	actualTrace := parquetTraceToTempopbTrace(&parquetTrace)
	assert.Equal(t, expectedTrace, actualTrace)
}

func TestProtoToParquetEmptyTrace(t *testing.T) {

	want := Trace{
		TraceID:       make([]byte, 16),
		ResourceSpans: []ResourceSpans{},
	}

	got := traceToParquet(nil, &tempopb.Trace{})

	require.Equal(t, want, got)
}

func BenchmarkProtoToParquet(b *testing.B) {

	batchCount := 100
	spanCounts := []int{
		100, 1000,
		10000,
	}

	for _, spanCount := range spanCounts {
		b.Run("SpanCount:"+humanize.SI(float64(batchCount*spanCount), ""), func(b *testing.B) {

			id := test.ValidTraceID(nil)
			tr := test.MakeTraceWithSpanCount(batchCount, spanCount, id)

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				traceToParquet(id, tr)
			}
		})
	}
}

func BenchmarkEventToParquet(b *testing.B) {
	e := &v1_trace.Span_Event{
		TimeUnixNano: 1000,
		Name:         "blerg",
		Attributes: []*v1.KeyValue{
			// String
			{Key: "s", Value: &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: "s2"}}},

			// Int
			{Key: "i", Value: &v1.AnyValue{Value: &v1.AnyValue_IntValue{IntValue: 123}}},

			// Double
			{Key: "d", Value: &v1.AnyValue{Value: &v1.AnyValue_DoubleValue{DoubleValue: 123.456}}},

			// Bool
			{Key: "b", Value: &v1.AnyValue{Value: &v1.AnyValue_BoolValue{BoolValue: true}}},

			// KVList
			{Key: "kv", Value: &v1.AnyValue{Value: &v1.AnyValue_KvlistValue{KvlistValue: &v1.KeyValueList{Values: []*v1.KeyValue{
				{Key: "s2", Value: &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: "s3"}}},
				{Key: "i2", Value: &v1.AnyValue{Value: &v1.AnyValue_IntValue{IntValue: 789}}},
			}}}}},

			// Array
			{Key: "a", Value: &v1.AnyValue{Value: &v1.AnyValue_ArrayValue{ArrayValue: &v1.ArrayValue{Values: []*v1.AnyValue{
				{Value: &v1.AnyValue_StringValue{StringValue: "s4"}},
				{Value: &v1.AnyValue_IntValue{IntValue: 101112}},
			}}}}},
		},
	}

	for i := 0; i < b.N; i++ {
		eventToParquet(e)
	}
}

func BenchmarkDeconstruct(b *testing.B) {

	batchCount := 100
	spanCounts := []int{
		100, 1000,
		10000,
	}

	poolSizes := []int{
		100_000,
		30_000_000,
	}

	for _, spanCount := range spanCounts {
		for _, poolSize := range poolSizes {
			ss := humanize.SI(float64(batchCount*spanCount), "")
			ps := humanize.SI(float64(poolSize), "")
			b.Run(fmt.Sprintf("SpanCount%v/Pool%v", ss, ps), func(b *testing.B) {

				id := test.ValidTraceID(nil)
				tr := traceToParquet(id, test.MakeTraceWithSpanCount(batchCount, spanCount, id))
				sch := parquet.SchemaOf(tr)

				b.ResetTimer()

				pool := newRowPool(poolSize)

				for i := 0; i < b.N; i++ {
					r2 := sch.Deconstruct(pool.Get(), tr)
					pool.Put(r2)
				}
			})
		}
	}
}

func TestParquetRowSizeEstimate(t *testing.T) {

	batchCount := 100
	spanCounts := []int{
		100, 1000,
		// 10000, this crashes in GitHub
	}

	for _, spanCount := range spanCounts {
		ss := humanize.SI(float64(batchCount*spanCount), "")
		t.Run(fmt.Sprintf("SpanCount%v", ss), func(t *testing.T) {

			id := test.ValidTraceID(nil)
			tr := test.MakeTraceWithSpanCount(batchCount, spanCount, id)
			proto, _ := tr.Marshal()
			fmt.Println("Size of proto is:", len(proto))

			parq := traceToParquet(id, tr)
			sch := parquet.SchemaOf(parq)
			row := sch.Deconstruct(nil, parq)

			fmt.Println("Size of parquet row is:", estimateProtoSize(row))
			fmt.Println("Size of parquet is:", estimateTraceSize(&parq))
		})
	}
}
