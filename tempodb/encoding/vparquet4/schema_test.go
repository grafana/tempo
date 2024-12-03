package vparquet4

import (
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"sort"
	"testing"

	"github.com/dustin/go-humanize"
	"github.com/gogo/protobuf/proto"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
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

	//assert.Equal(t, expectedTrace, actualTrace)
	tempopbTraceEqual(t, expectedTrace, actualTrace)
}

func TestProtoToParquetEmptyTrace(t *testing.T) {
	want := &Trace{
		TraceID:       make([]byte, 16),
		ResourceSpans: nil,
		ServiceStats:  map[string]ServiceStats{},
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
		ResourceSpans: []*v1_trace.ResourceSpans{
			{
				Resource: &v1_resource.Resource{
					Attributes: []*v1.KeyValue{
						{Key: LabelServiceName, Value: &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: "service1"}}},
						{Key: "i", Value: &v1.AnyValue{Value: &v1.AnyValue_DoubleValue{DoubleValue: 123.456}}},
					},
				},
				ScopeSpans: []*v1_trace.ScopeSpans{
					{
						Scope: &v1.InstrumentationScope{},
						Spans: []*v1_trace.Span{
							{
								TraceId: traceID,
								Status: &v1_trace.Status{
									Code: v1_trace.Status_STATUS_CODE_ERROR,
								},
								Attributes: []*v1.KeyValue{
									// an attribute for every type in order to make sure attributes are reused with different
									// type combinations
									{Key: "a", Value: &v1.AnyValue{Value: &v1.AnyValue_IntValue{IntValue: 11}}},
									{Key: "b", Value: &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: "bbb"}}},
									{Key: "c", Value: &v1.AnyValue{Value: &v1.AnyValue_BoolValue{BoolValue: true}}},
									{Key: "d", Value: &v1.AnyValue{Value: &v1.AnyValue_DoubleValue{DoubleValue: 111.11}}},
								},
								Events: []*v1_trace.Span_Event{
									{
										Attributes: []*v1.KeyValue{
											{Key: "event-attr", Value: &v1.AnyValue{Value: &v1.AnyValue_IntValue{IntValue: 123}}},
										},
									},
								},
								Links: []*v1_trace.Span_Link{
									{
										Attributes: []*v1.KeyValue{
											{Key: "link-attr", Value: &v1.AnyValue{Value: &v1.AnyValue_IntValue{IntValue: 123}}},
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

	expectedTrace := &Trace{
		TraceID:         traceID,
		TraceIDText:     "102030405060708090a0b0c0d0e0f",
		RootServiceName: "service1",
		ServiceStats:    map[string]ServiceStats{"service1": {SpanCount: 1, ErrorCount: 1}},
		ResourceSpans: []ResourceSpans{{
			Resource: Resource{
				ServiceName: "service1",
				Attrs: []Attribute{
					attr("i", 123.456),
				},
			},
			ScopeSpans: []ScopeSpans{{
				Spans: []Span{{
					StatusCode:     2,
					ParentID:       -1,
					NestedSetLeft:  1,
					NestedSetRight: 2,
					Attrs: []Attribute{
						attr("a", 11),
						attr("b", "bbb"),
						attr("c", true),
						attr("d", 111.11),
					},
					Events: []Event{{
						Attrs: []Attribute{
							attr("event-attr", 123),
						},
					}},
					Links: []Link{{
						Attrs: []Attribute{
							attr("link-attr", 123),
						},
					}},
				}},
			}},
		}},
	}

	// first convert a trace that sets all fields and then convert
	// a minimal trace to make sure nothing bleeds through
	tr := &Trace{}
	_, _ = traceToParquet(&meta, traceID, complexTrace, tr)
	actualTrace, _ := traceToParquet(&meta, traceID, simpleTrace, tr)

	traceEqual(t, expectedTrace, actualTrace)
}

func TestTraceToParquet(t *testing.T) {
	meta := backend.BlockMeta{DedicatedColumns: test.MakeDedicatedColumns()}
	traceID := common.ID{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F}

	tsc := []struct {
		name     string
		id       common.ID
		trace    tempopb.Trace
		expected Trace
	}{
		{
			name: "span scope and resource attributes",
			id:   traceID,
			trace: tempopb.Trace{
				ResourceSpans: []*v1_trace.ResourceSpans{{
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
							{Key: "res.string.array", Value: &v1.AnyValue{Value: &v1.AnyValue_ArrayValue{ArrayValue: &v1.ArrayValue{
								Values: []*v1.AnyValue{
									{Value: &v1.AnyValue_StringValue{StringValue: "one"}},
									{Value: &v1.AnyValue_StringValue{StringValue: "two"}},
									{Value: &v1.AnyValue_StringValue{StringValue: "three"}},
								},
							}}}},
							{Key: "res.int.array", Value: &v1.AnyValue{Value: &v1.AnyValue_ArrayValue{ArrayValue: &v1.ArrayValue{
								Values: []*v1.AnyValue{
									{Value: &v1.AnyValue_IntValue{IntValue: 1}},
									{Value: &v1.AnyValue_IntValue{IntValue: 2}},
								},
							}}}},
							{Key: "res.double.array", Value: &v1.AnyValue{Value: &v1.AnyValue_ArrayValue{ArrayValue: &v1.ArrayValue{
								Values: []*v1.AnyValue{
									{Value: &v1.AnyValue_DoubleValue{DoubleValue: 1.1}},
									{Value: &v1.AnyValue_DoubleValue{DoubleValue: 2.2}},
									{Value: &v1.AnyValue_DoubleValue{DoubleValue: 3.3}},
								},
							}}}},
							{Key: "res.bool.array", Value: &v1.AnyValue{Value: &v1.AnyValue_ArrayValue{ArrayValue: &v1.ArrayValue{
								Values: []*v1.AnyValue{
									{Value: &v1.AnyValue_BoolValue{BoolValue: true}},
									{Value: &v1.AnyValue_BoolValue{BoolValue: false}},
									{Value: &v1.AnyValue_BoolValue{BoolValue: true}},
									{Value: &v1.AnyValue_BoolValue{BoolValue: true}},
								},
							}}}},
						},
					},
					ScopeSpans: []*v1_trace.ScopeSpans{{
						Scope: &v1.InstrumentationScope{
							Name:                   "scope-a",
							Version:                "scope-a-version",
							DroppedAttributesCount: 101,
							Attributes: []*v1.KeyValue{
								{Key: "scope.attr.str", Value: &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: "scope-val-1"}}},
								{Key: "scope.attr.int", Value: &v1.AnyValue{Value: &v1.AnyValue_IntValue{IntValue: 102}}},
								{Key: "scope.attr.float", Value: &v1.AnyValue{Value: &v1.AnyValue_DoubleValue{DoubleValue: 1.234}}},
								{Key: "scope.attr.bool", Value: &v1.AnyValue{Value: &v1.AnyValue_BoolValue{BoolValue: true}}},
								{Key: "scope.string.array", Value: &v1.AnyValue{Value: &v1.AnyValue_ArrayValue{ArrayValue: &v1.ArrayValue{
									Values: []*v1.AnyValue{
										{Value: &v1.AnyValue_StringValue{StringValue: "one"}},
										{Value: &v1.AnyValue_StringValue{StringValue: "two"}},
									},
								}}}},
							},
						},
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
								{Key: "span.string.array", Value: &v1.AnyValue{Value: &v1.AnyValue_ArrayValue{ArrayValue: &v1.ArrayValue{
									Values: []*v1.AnyValue{
										{Value: &v1.AnyValue_StringValue{StringValue: "one"}},
										{Value: &v1.AnyValue_StringValue{StringValue: "two"}},
									},
								}}}},
								{Key: "span.int.array", Value: &v1.AnyValue{Value: &v1.AnyValue_ArrayValue{ArrayValue: &v1.ArrayValue{
									Values: []*v1.AnyValue{
										{Value: &v1.AnyValue_IntValue{IntValue: 1}},
										{Value: &v1.AnyValue_IntValue{IntValue: 2}},
										{Value: &v1.AnyValue_IntValue{IntValue: 3}},
									},
								}}}},
								{Key: "span.double.array", Value: &v1.AnyValue{Value: &v1.AnyValue_ArrayValue{ArrayValue: &v1.ArrayValue{
									Values: []*v1.AnyValue{
										{Value: &v1.AnyValue_DoubleValue{DoubleValue: 1.1}},
										{Value: &v1.AnyValue_DoubleValue{DoubleValue: 2.2}},
									},
								}}}},
								{Key: "span.bool.array", Value: &v1.AnyValue{Value: &v1.AnyValue_ArrayValue{ArrayValue: &v1.ArrayValue{
									Values: []*v1.AnyValue{
										{Value: &v1.AnyValue_BoolValue{BoolValue: true}},
										{Value: &v1.AnyValue_BoolValue{BoolValue: false}},
										{Value: &v1.AnyValue_BoolValue{BoolValue: true}},
										{Value: &v1.AnyValue_BoolValue{BoolValue: false}},
									},
								}}}},
								{Key: "span.unsupported.array", Value: &v1.AnyValue{Value: &v1.AnyValue_ArrayValue{ArrayValue: &v1.ArrayValue{
									Values: []*v1.AnyValue{
										{Value: &v1.AnyValue_BoolValue{BoolValue: true}},
										{Value: &v1.AnyValue_IntValue{IntValue: 1}},
										{Value: &v1.AnyValue_BoolValue{BoolValue: true}},
									},
								}}}},
								{Key: "span.unsupported.kvlist", Value: &v1.AnyValue{Value: &v1.AnyValue_KvlistValue{KvlistValue: &v1.KeyValueList{
									Values: []*v1.KeyValue{
										{Key: "key-a", Value: &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: "val-a"}}},
										{Key: "key-b", Value: &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: "val-b"}}},
									},
								}}}},
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
				ServiceStats: map[string]ServiceStats{
					"service-a": {
						SpanCount:  1,
						ErrorCount: 0,
					},
				},
				ResourceSpans: []ResourceSpans{{
					Resource: Resource{
						ServiceName:      "service-a",
						Cluster:          ptr("cluster-a"),
						Namespace:        ptr("namespace-a"),
						Pod:              ptr("pod-a"),
						Container:        ptr("container-a"),
						K8sClusterName:   ptr("k8s-cluster-a"),
						K8sNamespaceName: ptr("k8s-namespace-a"),
						K8sPodName:       ptr("k8s-pod-a"),
						K8sContainerName: ptr("k8s-container-a"),
						Attrs: []Attribute{
							attr("res.attr", 123),
							attr("res.string.array", []string{"one", "two", "three"}),
							attr("res.int.array", []int64{1, 2}),
							attr("res.double.array", []float64{1.1, 2.2, 3.3}),
							attr("res.bool.array", []bool{true, false, true, true}),
						},
						DedicatedAttributes: DedicatedAttributes{
							String01: ptr("dedicated-resource-attr-value-1"),
							String02: ptr("dedicated-resource-attr-value-2"),
							String03: ptr("dedicated-resource-attr-value-3"),
							String04: ptr("dedicated-resource-attr-value-4"),
							String05: ptr("dedicated-resource-attr-value-5"),
						},
					},
					ScopeSpans: []ScopeSpans{{
						Scope: InstrumentationScope{
							Name:                   "scope-a",
							Version:                "scope-a-version",
							DroppedAttributesCount: 101,
							Attrs: []Attribute{
								attr("scope.attr.str", "scope-val-1"),
								attr("scope.attr.int", 102),
								attr("scope.attr.float", 1.234),
								attr("scope.attr.bool", true),
								attr("scope.string.array", []string{"one", "two"}),
							},
						},
						Spans: []Span{{
							Name:           "span-a",
							SpanID:         []byte{0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
							NestedSetLeft:  1,
							NestedSetRight: 2,
							ParentID:       -1,
							HttpMethod:     ptr("POST"),
							HttpUrl:        ptr("https://example.com"),
							HttpStatusCode: ptr(int64(201)),
							Attrs: []Attribute{
								attr("span.attr", "aaa"),
								attr("span.string.array", []string{"one", "two"}),
								attr("span.int.array", []int64{1, 2, 3}),
								attr("span.double.array", []float64{1.1, 2.2}),
								attr("span.bool.array", []bool{true, false, true, false}),
								{Key: "span.unsupported.array", ValueUnsupported: ptr("{\"arrayValue\":{\"values\":[{\"boolValue\":true},{\"intValue\":\"1\"},{\"boolValue\":true}]}}"), IsArray: false},
								{Key: "span.unsupported.kvlist", ValueUnsupported: ptr("{\"kvlistValue\":{\"values\":[{\"key\":\"key-a\",\"value\":{\"stringValue\":\"val-a\"}},{\"key\":\"key-b\",\"value\":{\"stringValue\":\"val-b\"}}]}}"), IsArray: false},
							},
							DroppedAttributesCount: 0,
							DedicatedAttributes: DedicatedAttributes{
								String01: ptr("dedicated-span-attr-value-1"),
								String02: ptr("dedicated-span-attr-value-2"),
								String03: ptr("dedicated-span-attr-value-3"),
								String04: ptr("dedicated-span-attr-value-4"),
								String05: ptr("dedicated-span-attr-value-5"),
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
				ResourceSpans: []*v1_trace.ResourceSpans{{
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
				ServiceStats: map[string]ServiceStats{
					"service-a": {
						SpanCount:  3,
						ErrorCount: 0,
					},
				},
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
								ParentID:       -1,
								Attrs: []Attribute{
									attr("span.attr", "aaa"),
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
									attr("span.attr", "bbb"),
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
									attr("span.attr", "ccc"),
								},
							},
						},
					}},
				}},
			},
		},
		{
			name: "service stats",
			id:   traceID,
			trace: tempopb.Trace{
				ResourceSpans: []*v1_trace.ResourceSpans{{
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
								Status: &v1_trace.Status{
									Code: v1_trace.Status_STATUS_CODE_ERROR,
								},
							},
							{
								Name:         "span-b",
								SpanId:       common.ID{0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
								ParentSpanId: common.ID{0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
							},
							{
								Name:         "span-c",
								SpanId:       common.ID{0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x03},
								ParentSpanId: common.ID{0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
							},
						},
					}},
				}, {
					Resource: &v1_resource.Resource{
						Attributes: []*v1.KeyValue{
							{Key: "service.name", Value: &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: "service-b"}}},
						},
					},
					ScopeSpans: []*v1_trace.ScopeSpans{{
						Scope: &v1.InstrumentationScope{},
						Spans: []*v1_trace.Span{
							{
								Name:         "span-d",
								SpanId:       common.ID{0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x04},
								ParentSpanId: common.ID{0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x03},
							},
							{
								Name:         "span-e",
								SpanId:       common.ID{0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x05},
								ParentSpanId: common.ID{0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x04},
								Status: &v1_trace.Status{
									Code: v1_trace.Status_STATUS_CODE_ERROR,
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
				ServiceStats: map[string]ServiceStats{
					"service-a": {
						SpanCount:  3,
						ErrorCount: 1,
					},
					"service-b": {
						SpanCount:  2,
						ErrorCount: 1,
					},
				},
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
								NestedSetRight: 10,
								ParentID:       -1,
								StatusCode:     int(v1_trace.Status_STATUS_CODE_ERROR),
							},
							{
								Name:           "span-b",
								SpanID:         []byte{0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
								ParentSpanID:   []byte{0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
								ParentID:       1,
								NestedSetLeft:  2,
								NestedSetRight: 3,
							},
							{
								Name:           "span-c",
								SpanID:         []byte{0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x03},
								ParentSpanID:   []byte{0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
								ParentID:       1,
								NestedSetLeft:  4,
								NestedSetRight: 9,
							},
						},
					}},
				}, {
					Resource: Resource{
						ServiceName: "service-b",
						Attrs:       []Attribute{},
					},
					ScopeSpans: []ScopeSpans{{
						Spans: []Span{
							{
								Name:           "span-d",
								SpanID:         []byte{0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x04},
								ParentSpanID:   []byte{0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x03},
								NestedSetLeft:  5,
								NestedSetRight: 8,
								ParentID:       4,
							},
							{
								Name:           "span-e",
								SpanID:         []byte{0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x05},
								ParentSpanID:   []byte{0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x04},
								ParentID:       5,
								NestedSetLeft:  6,
								NestedSetRight: 7,
								StatusCode:     int(v1_trace.Status_STATUS_CODE_ERROR),
							},
						},
					}},
				}},
			},
		},
		{
			name: "links and events attributes",
			id:   traceID,
			trace: tempopb.Trace{
				ResourceSpans: []*v1_trace.ResourceSpans{{
					Resource: &v1_resource.Resource{
						Attributes: []*v1.KeyValue{
							{Key: "service.name", Value: &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: "service-a"}}},
						},
					},
					ScopeSpans: []*v1_trace.ScopeSpans{{
						Scope: &v1.InstrumentationScope{},
						Spans: []*v1_trace.Span{
							{
								Name:              "span-with-link",
								SpanId:            common.ID{0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01}, // 01
								StartTimeUnixNano: 1500,
								EndTimeUnixNano:   3000,
								Links: []*v1_trace.Span_Link{{
									TraceId: traceID,
									SpanId:  common.ID{0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02}, // 02
									Attributes: []*v1.KeyValue{
										{Key: "link.attr", Value: &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: "aaa"}}},
									},
									TraceState: "link trace state",
								}},
							},
							{
								Name:              "span-with-event",
								SpanId:            common.ID{0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02}, // 02
								StartTimeUnixNano: 1000,
								EndTimeUnixNano:   4000,
								Events: []*v1_trace.Span_Event{{
									TimeUnixNano: 2000,
									Name:         "event name",
									Attributes: []*v1.KeyValue{
										{Key: "event.attr", Value: &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: "bbb"}}},
									},
								}},
							},
						},
					}},
				}},
			},
			expected: Trace{
				TraceID:           traceID,
				TraceIDText:       "102030405060708090a0b0c0d0e0f",
				RootSpanName:      "span-with-event",
				RootServiceName:   "service-a",
				StartTimeUnixNano: 1000,
				EndTimeUnixNano:   4000,
				DurationNano:      3000,
				ServiceStats: map[string]ServiceStats{
					"service-a": {
						SpanCount:  2,
						ErrorCount: 0,
					},
				},
				ResourceSpans: []ResourceSpans{{
					Resource: Resource{
						ServiceName: "service-a",
						Attrs:       []Attribute{},
					},
					ScopeSpans: []ScopeSpans{{
						Spans: []Span{
							{
								Name:              "span-with-link",
								SpanID:            []byte{0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
								NestedSetLeft:     1,
								NestedSetRight:    2,
								ParentID:          -1,
								StartTimeUnixNano: 1500,
								DurationNano:      1500,
								Links: []Link{{
									TraceID: traceID,
									SpanID:  []byte{0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
									Attrs: []Attribute{
										attr("link.attr", "aaa"),
									},
									TraceState: "link trace state",
								}},
							},
							{
								Name:              "span-with-event",
								SpanID:            []byte{0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
								NestedSetLeft:     3,
								NestedSetRight:    4,
								ParentID:          -1,
								StartTimeUnixNano: 1000,
								DurationNano:      3000,
								Events: []Event{{
									TimeSinceStartNano: 1000,
									Name:               "event name",
									Attrs: []Attribute{
										attr("event.attr", "bbb"),
									},
								}},
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
			traceEqual(t, &tt.expected, &actual)
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
	s := &v1_trace.Span{
		StartTimeUnixNano: 100,
	}
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
		eventToParquet(e, ee, s.StartTimeUnixNano)
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
	f, err := os.OpenFile(name, os.O_RDONLY, 0o600)
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

func tempopbTraceEqual(t *testing.T, expected, actual *tempopb.Trace) {
	sortAttributesTempopb(expected)
	sortAttributesTempopb(actual)

	if !proto.Equal(expected, actual) {
		t.Log(cmp.Diff(expected, actual))
		assert.Fail(t, "expected and actual are not equal")
	}
}

func sortAttributesTempopb(t *tempopb.Trace) {
	// sort attributes for comparison
	for _, rs := range t.ResourceSpans {
		sort.Slice(rs.Resource.Attributes, func(i, j int) bool {
			if rs.Resource.Attributes[i].Key == rs.Resource.Attributes[j].Key {
				return rs.Resource.Attributes[i].Value.String() < rs.Resource.Attributes[j].Value.String()
			}

			return rs.Resource.Attributes[i].Key < rs.Resource.Attributes[j].Key
		})
		for _, ss := range rs.ScopeSpans {
			sort.Slice(ss.Scope.Attributes, func(i, j int) bool {
				if rs.Resource.Attributes[i].Key == rs.Resource.Attributes[j].Key {
					return rs.Resource.Attributes[i].Value.String() < rs.Resource.Attributes[j].Value.String()
				}

				return ss.Scope.Attributes[i].Key < ss.Scope.Attributes[j].Key
			})
		}
	}
}

// traceEqual asserts similar to assert.Equal but treats empty / nil slices and maps as equal
func traceEqual(t *testing.T, expected, actual *Trace, messages ...interface{}) {
	sortAttributes(expected)
	sortAttributes(actual)

	if !cmp.Equal(expected, actual, cmpopts.EquateEmpty()) {
		t.Log(cmp.Diff(expected, actual))
		assert.Fail(t, "expected and actual are not equal", messages...)
	}
}

func sortAttributes(t *Trace) {
	// sort attributes for comparison
	for _, rs := range t.ResourceSpans {
		sort.Slice(rs.Resource.Attrs, func(i, j int) bool {
			return rs.Resource.Attrs[i].Key < rs.Resource.Attrs[j].Key
		})
		for _, ss := range rs.ScopeSpans {
			sort.Slice(ss.Scope.Attrs, func(i, j int) bool {
				return ss.Scope.Attrs[i].Key < ss.Scope.Attrs[j].Key
			})
		}
	}
}
