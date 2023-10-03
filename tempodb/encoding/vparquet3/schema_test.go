package vparquet3

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
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

func TestProtoParquetRoundTrip(t *testing.T) {
	// This test round trips a proto trace and checks that the transformation works as expected
	// Proto -> Parquet -> Proto
	meta := backend.BlockMeta{
		DedicatedColumns: test.MakeDedicatedColumns(),
	}
	traceIDA := []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F}

	expectedTrace := parquetTraceToTempopbTrace(&meta, fullyPopulatedTestTrace(traceIDA))

	parquetTrace, connected := traceToParquet(&meta, traceIDA, expectedTrace, nil)
	require.True(t, connected)
	actualTrace := parquetTraceToTempopbTrace(&meta, parquetTrace)
	assert.Equal(t, expectedTrace, actualTrace)
}

func TestProtoToParquetEmptyTrace(t *testing.T) {
	want := &Trace{
		TraceID:       make([]byte, 16),
		ResourceSpans: nil,
	}

	got, connected := traceToParquet(&backend.BlockMeta{}, nil, &tempopb.Trace{}, nil)
	require.False(t, connected)
	require.Equal(t, want, got)
}

func TestProtoParquetRando(t *testing.T) {
	trp := &Trace{}
	for i := 0; i < 100; i++ {
		batches := rand.Intn(15)
		id := test.ValidTraceID(nil)
		expectedTrace := test.AddDedicatedAttributes(test.MakeTrace(batches, id))

		parqTr, _ := traceToParquet(&backend.BlockMeta{}, id, expectedTrace, trp)
		actualTrace := parquetTraceToTempopbTrace(&backend.BlockMeta{}, parqTr)
		require.Equal(t, expectedTrace, actualTrace)
	}
}

func TestFieldsAreCleared(t *testing.T) {
	meta := backend.BlockMeta{
		DedicatedColumns: test.MakeDedicatedColumns(),
	}

	traceID := []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F}
	complexTrace := parquetTraceToTempopbTrace(&meta, fullyPopulatedTestTrace(traceID))
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
	_, _ = traceToParquet(&meta, traceID, complexTrace, tr)

	parqTr, _ := traceToParquet(&meta, traceID, simpleTrace, tr)
	actualTrace := parquetTraceToTempopbTrace(&meta, parqTr)
	require.Equal(t, simpleTrace, actualTrace)
}

func TestTraceToParquet(t *testing.T) {
	strPtr := func(s string) *string { return &s }
	intPtr := func(i int64) *int64 { return &i }

	meta := backend.BlockMeta{DedicatedColumns: test.MakeDedicatedColumns()}
	traceID := common.ID{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F}

	tsc := []struct {
		name     string
		id       common.ID
		trace    tempopb.Trace
		expected Trace
	}{
		{
			name: "span and resource attributes",
			id:   traceID,
			trace: tempopb.Trace{
				Batches: []*v1_trace.ResourceSpans{{
					Resource: &v1_resource.Resource{
						Attributes: []*v1.KeyValue{
							{Key: "res.attr", Value: &v1.AnyValue{Value: &v1.AnyValue_IntValue{IntValue: 123}}},
							{Key: "service.name", Value: &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: "service-a"}}},
							{Key: "cluster", Value: &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: "cluster-a"}}},
							{Key: "namespace", Value: &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: "namespace-a"}}},
							{Key: "pod", Value: &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: "pod-a"}}},
							{Key: "container", Value: &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: "container-a"}}},
							{Key: "k8s.cluster.name", Value: &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: "k8s-cluster-a"}}},
							{Key: "k8s.namespace.name", Value: &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: "k8s-namespace-a"}}},
							{Key: "k8s.pod.name", Value: &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: "k8s-pod-a"}}},
							{Key: "k8s.container.name", Value: &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: "k8s-container-a"}}},
							{Key: "dedicated.resource.1", Value: &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: "dedicated-resource-attr-value-1"}}},
							{Key: "dedicated.resource.2", Value: &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: "dedicated-resource-attr-value-2"}}},
							{Key: "dedicated.resource.3", Value: &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: "dedicated-resource-attr-value-3"}}},
							{Key: "dedicated.resource.4", Value: &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: "dedicated-resource-attr-value-4"}}},
							{Key: "dedicated.resource.5", Value: &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: "dedicated-resource-attr-value-5"}}},
						},
					},
					ScopeSpans: []*v1_trace.ScopeSpans{{
						Scope: &v1.InstrumentationScope{},
						Spans: []*v1_trace.Span{{
							Name:   "span-a",
							SpanId: common.ID{0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
							Attributes: []*v1.KeyValue{
								{Key: "span.attr", Value: &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: "aaa"}}},
								{Key: "http.method", Value: &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: "POST"}}},
								{Key: "http.url", Value: &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: "https://example.com"}}},
								{Key: "http.status_code", Value: &v1.AnyValue{Value: &v1.AnyValue_IntValue{IntValue: 201}}},
								{Key: "dedicated.span.1", Value: &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: "dedicated-span-attr-value-1"}}},
								{Key: "dedicated.span.2", Value: &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: "dedicated-span-attr-value-2"}}},
								{Key: "dedicated.span.3", Value: &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: "dedicated-span-attr-value-3"}}},
								{Key: "dedicated.span.4", Value: &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: "dedicated-span-attr-value-4"}}},
								{Key: "dedicated.span.5", Value: &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: "dedicated-span-attr-value-5"}}},
							},
						}},
					}},
				}},
			},
			expected: Trace{
				TraceID:         traceID,
				TraceIDText:     "102030405060708090a0b0c0d0e0f",
				RootSpanName:    "span-a",
				RootServiceName: "service-a",
				ResourceSpans: []ResourceSpans{{
					Resource: Resource{
						ServiceName:      "service-a",
						Cluster:          strPtr("cluster-a"),
						Namespace:        strPtr("namespace-a"),
						Pod:              strPtr("pod-a"),
						Container:        strPtr("container-a"),
						K8sClusterName:   strPtr("k8s-cluster-a"),
						K8sNamespaceName: strPtr("k8s-namespace-a"),
						K8sPodName:       strPtr("k8s-pod-a"),
						K8sContainerName: strPtr("k8s-container-a"),
						Attrs: []Attribute{
							{Key: "res.attr", ValueInt: intPtr(int64(123))},
						},
						DedicatedAttributes: DedicatedAttributes{
							String01: strPtr("dedicated-resource-attr-value-1"),
							String02: strPtr("dedicated-resource-attr-value-2"),
							String03: strPtr("dedicated-resource-attr-value-3"),
							String04: strPtr("dedicated-resource-attr-value-4"),
							String05: strPtr("dedicated-resource-attr-value-5"),
						},
					},
					ScopeSpans: []ScopeSpans{{
						Spans: []Span{{
							Name:           "span-a",
							SpanID:         []byte{0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
							NestedSetLeft:  1,
							NestedSetRight: 2,
							HttpMethod:     strPtr("POST"),
							HttpUrl:        strPtr("https://example.com"),
							HttpStatusCode: intPtr(201),
							Attrs: []Attribute{
								{Key: "span.attr", Value: strPtr("aaa")},
							},
							DedicatedAttributes: DedicatedAttributes{
								String01: strPtr("dedicated-span-attr-value-1"),
								String02: strPtr("dedicated-span-attr-value-2"),
								String03: strPtr("dedicated-span-attr-value-3"),
								String04: strPtr("dedicated-span-attr-value-4"),
								String05: strPtr("dedicated-span-attr-value-5"),
							},
						}},
					}},
				}},
			},
		},
		{
			name: "nested set model bounds",
			id:   traceID,
			trace: tempopb.Trace{
				Batches: []*v1_trace.ResourceSpans{{
					Resource: &v1_resource.Resource{
						Attributes: []*v1.KeyValue{
							{Key: "service.name", Value: &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: "service-a"}}},
						},
					},
					ScopeSpans: []*v1_trace.ScopeSpans{{
						Scope: &v1.InstrumentationScope{},
						Spans: []*v1_trace.Span{
							{
								Name:   "span-a",
								SpanId: common.ID{0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
								Attributes: []*v1.KeyValue{
									{Key: "span.attr", Value: &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: "aaa"}}},
								},
							},
							{
								Name:         "span-b",
								SpanId:       common.ID{0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
								ParentSpanId: common.ID{0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
								Attributes: []*v1.KeyValue{
									{Key: "span.attr", Value: &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: "bbb"}}},
								},
							},
							{
								Name:         "span-c",
								SpanId:       common.ID{0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x03},
								ParentSpanId: common.ID{0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
								Attributes: []*v1.KeyValue{
									{Key: "span.attr", Value: &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: "ccc"}}},
								},
							},
						},
					}},
				}},
			},
			expected: Trace{
				TraceID:         traceID,
				TraceIDText:     "102030405060708090a0b0c0d0e0f",
				RootSpanName:    "span-a",
				RootServiceName: "service-a",
				ResourceSpans: []ResourceSpans{{
					Resource: Resource{
						ServiceName: "service-a",
						Attrs:       []Attribute{},
					},
					ScopeSpans: []ScopeSpans{{
						Spans: []Span{
							{
								Name:           "span-a",
								SpanID:         []byte{0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
								NestedSetLeft:  1,
								NestedSetRight: 6,
								Attrs: []Attribute{
									{Key: "span.attr", Value: strPtr("aaa")},
								},
							},
							{
								Name:           "span-b",
								SpanID:         []byte{0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
								ParentSpanID:   []byte{0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
								ParentID:       1,
								NestedSetLeft:  2,
								NestedSetRight: 3,
								Attrs: []Attribute{
									{Key: "span.attr", Value: strPtr("bbb")},
								},
							},
							{
								Name:           "span-c",
								SpanID:         []byte{0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x03},
								ParentSpanID:   []byte{0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
								ParentID:       1,
								NestedSetLeft:  4,
								NestedSetRight: 5,
								Attrs: []Attribute{
									{Key: "span.attr", Value: strPtr("ccc")},
								},
							},
						},
					}},
				}},
			},
		},
	}

	for _, tt := range tsc {
		t.Run(tt.name, func(t *testing.T) {
			var actual Trace
			traceToParquet(&meta, tt.id, &tt.trace, &actual)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func BenchmarkProtoToParquet(b *testing.B) {
	meta := backend.BlockMeta{
		DedicatedColumns: test.MakeDedicatedColumns(),
	}

	batchCount := 100
	spanCounts := []int{
		100, 1000,
		10000,
	}

	for _, spanCount := range spanCounts {
		b.Run("SpanCount:"+humanize.SI(float64(batchCount*spanCount), ""), func(b *testing.B) {
			id := test.ValidTraceID(nil)
			tr := test.AddDedicatedAttributes(test.MakeTraceWithSpanCount(batchCount, spanCount, id))

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, _ = traceToParquet(&meta, id, tr, nil)
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
	meta := backend.BlockMeta{
		DedicatedColumns: test.MakeDedicatedColumns(),
	}

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
				dbt := test.MakeTraceWithSpanCount(batchCount, spanCount, id)
				test.AddDedicatedAttributes(dbt)

				tr, _ := traceToParquet(&meta, id, dbt, nil)
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
