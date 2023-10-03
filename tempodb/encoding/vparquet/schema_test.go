package vparquet

import (
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"testing"

	"github.com/dustin/go-humanize"
	"github.com/parquet-go/parquet-go"
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
	expectedTrace := parquetTraceToTempopbTrace(fullyPopulatedTestTrace(traceIDA))

	parquetTrace := traceToParquet(traceIDA, expectedTrace, nil)
	actualTrace := parquetTraceToTempopbTrace(parquetTrace)
	assert.Equal(t, expectedTrace, actualTrace)
}

func TestProtoToParquetEmptyTrace(t *testing.T) {
	want := &Trace{
		TraceID:       make([]byte, 16),
		ResourceSpans: nil,
	}

	got := traceToParquet(nil, &tempopb.Trace{}, nil)
	require.Equal(t, want, got)
}

func TestProtoParquetRando(t *testing.T) {
	trp := &Trace{}
	for i := 0; i < 100; i++ {
		batches := rand.Intn(15)
		id := test.ValidTraceID(nil)
		expectedTrace := test.MakeTrace(batches, id)

		parqTr := traceToParquet(id, expectedTrace, trp)
		actualTrace := parquetTraceToTempopbTrace(parqTr)
		require.Equal(t, expectedTrace, actualTrace)
	}
}

func TestFieldsAreCleared(t *testing.T) {
	traceID := []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F}
	complexTrace := parquetTraceToTempopbTrace(fullyPopulatedTestTrace(traceID))
	simpleTrace := &tempopb.Trace{
		Batches: []*v1_trace.ResourceSpans{
			{
				Resource: &v1_resource.Resource{
					Attributes: []*v1.KeyValue{
						{Key: "i", Value: &v1.AnyValue{Value: &v1.AnyValue_IntValue{IntValue: 123}}},
					},
				},
				ScopeSpans: []*v1_trace.ScopeSpans{
					{
						Scope: &v1.InstrumentationScope{},
						Spans: []*v1_trace.Span{
							{
								TraceId: traceID,
								Status:  &v1_trace.Status{},
								Attributes: []*v1.KeyValue{
									{
										Key: "a",
										Value: &v1.AnyValue{
											Value: &v1.AnyValue_StringValue{StringValue: "b"},
										},
									},
								},
								Events: []*v1_trace.Span_Event{
									{
										// An attribute of every type
										Attributes: []*v1.KeyValue{
											// String
											{Key: "i", Value: &v1.AnyValue{Value: &v1.AnyValue_IntValue{IntValue: 123}}},
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

	// first convert a trace that sets all fields and then convert
	// a minimal trace to make sure nothing bleeds through
	tr := &Trace{}
	_ = traceToParquet(traceID, complexTrace, tr)
	parqTr := traceToParquet(traceID, simpleTrace, tr)
	actualTrace := parquetTraceToTempopbTrace(parqTr)
	require.Equal(t, simpleTrace, actualTrace)
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
				_ = traceToParquet(id, tr, nil)
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

	ee := &Event{}
	for i := 0; i < b.N; i++ {
		eventToParquet(e, ee)
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
				tr := traceToParquet(id, test.MakeTraceWithSpanCount(batchCount, spanCount, id), nil)
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
	// use this test to parse actual Parquet files and compare the two methods of estimating row size
	s := []string{}

	for _, s := range s {
		estimateRowSize(t, s)
	}
}

func estimateRowSize(t *testing.T, name string) {
	f, err := os.OpenFile(name, os.O_RDONLY, 0o644)
	require.NoError(t, err)

	fi, err := f.Stat()
	require.NoError(t, err)

	pf, err := parquet.OpenFile(f, fi.Size())
	require.NoError(t, err)

	r := parquet.NewGenericReader[*Trace](pf)
	row := make([]*Trace, 1)

	totalProtoSize := int64(0)
	totalTraceSize := int64(0)
	for {
		_, err := r.Read(row)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			require.NoError(t, err)
		}

		tr := row[0]
		sch := parquet.SchemaOf(tr)
		row := sch.Deconstruct(nil, tr)

		totalProtoSize += int64(estimateMarshalledSizeFromParquetRow(row))
		totalTraceSize += int64(estimateMarshalledSizeFromTrace(tr))
	}

	fmt.Println(pf.Size(), ",", len(pf.RowGroups()), ",", totalProtoSize, ",", totalTraceSize)
}

func TestExtendReuseSlice(t *testing.T) {
	tcs := []struct {
		sz       int
		in       []int
		expected []int
	}{
		{
			sz:       0,
			in:       []int{1, 2, 3},
			expected: []int{},
		},
		{
			sz:       2,
			in:       []int{1, 2, 3},
			expected: []int{1, 2},
		},
		{
			sz:       5,
			in:       []int{1, 2, 3},
			expected: []int{1, 2, 3, 0, 0},
		},
	}

	for _, tc := range tcs {
		t.Run(fmt.Sprintf("%v", tc.sz), func(t *testing.T) {
			out := extendReuseSlice(tc.sz, tc.in)
			assert.Equal(t, tc.expected, out)
		})
	}
}

func BenchmarkExtendReuseSlice(b *testing.B) {
	in := []int{1, 2, 3}
	for i := 0; i < b.N; i++ {
		_ = extendReuseSlice(100, in)
	}
}
