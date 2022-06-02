package distributor

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/go-kit/log"
	kitlog "github.com/go-kit/log"
	"github.com/gogo/status"
	"github.com/golang/protobuf/proto"
	"github.com/grafana/dskit/flagext"
	"github.com/grafana/dskit/kv"
	"github.com/grafana/dskit/ring"
	ring_client "github.com/grafana/dskit/ring/client"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/weaveworks/common/logging"
	"github.com/weaveworks/common/user"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health/grpc_health_v1"

	"github.com/grafana/tempo/modules/distributor/receiver"
	generator_client "github.com/grafana/tempo/modules/generator/client"
	ingester_client "github.com/grafana/tempo/modules/ingester/client"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/tempopb"
	v1_common "github.com/grafana/tempo/pkg/tempopb/common/v1"
	v1_resource "github.com/grafana/tempo/pkg/tempopb/resource/v1"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/pkg/util/test"
)

const (
	numIngesters = 5
)

var (
	ctx = user.InjectOrgID(context.Background(), "test")
)

func TestRequestsByTraceID(t *testing.T) {
	traceIDA := []byte{0x0A, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F}
	traceIDB := []byte{0x0B, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F}

	tests := []struct {
		name           string
		batches        []*v1.ResourceSpans
		expectedKeys   []uint32
		expectedTraces []*tempopb.Trace
		expectedIDs    [][]byte
		expectedErr    error
		expectedStarts []uint32
		expectedEnds   []uint32
	}{
		{
			name: "empty",
			batches: []*v1.ResourceSpans{
				{},
				{},
			},
			expectedKeys:   []uint32{},
			expectedTraces: []*tempopb.Trace{},
			expectedIDs:    [][]byte{},
			expectedStarts: []uint32{},
			expectedEnds:   []uint32{},
		},
		{
			name: "bad trace id",
			batches: []*v1.ResourceSpans{
				{
					InstrumentationLibrarySpans: []*v1.InstrumentationLibrarySpans{
						{
							Spans: []*v1.Span{
								{
									TraceId: []byte{0x01},
								},
							},
						},
					},
				},
			},
			expectedErr: status.Errorf(codes.InvalidArgument, "trace ids must be 128 bit"),
		},
		{
			name: "one span",
			batches: []*v1.ResourceSpans{
				{
					InstrumentationLibrarySpans: []*v1.InstrumentationLibrarySpans{
						{
							Spans: []*v1.Span{
								{
									TraceId:           traceIDA,
									StartTimeUnixNano: uint64(10 * time.Second),
									EndTimeUnixNano:   uint64(20 * time.Second),
								}}}}},
			},
			expectedKeys: []uint32{util.TokenFor(util.FakeTenantID, traceIDA)},
			expectedTraces: []*tempopb.Trace{
				{
					Batches: []*v1.ResourceSpans{
						{
							InstrumentationLibrarySpans: []*v1.InstrumentationLibrarySpans{
								{
									Spans: []*v1.Span{
										{
											TraceId:           traceIDA,
											StartTimeUnixNano: uint64(10 * time.Second),
											EndTimeUnixNano:   uint64(20 * time.Second),
										}}}}}},
				},
			},
			expectedIDs: [][]byte{
				traceIDA,
			},
			expectedStarts: []uint32{10},
			expectedEnds:   []uint32{20},
		},
		{
			name: "two traces, one batch",
			batches: []*v1.ResourceSpans{
				{
					InstrumentationLibrarySpans: []*v1.InstrumentationLibrarySpans{
						{
							Spans: []*v1.Span{
								{
									TraceId:           traceIDA,
									StartTimeUnixNano: uint64(30 * time.Second),
									EndTimeUnixNano:   uint64(40 * time.Second),
								},
								{
									TraceId:           traceIDB,
									StartTimeUnixNano: uint64(50 * time.Second),
									EndTimeUnixNano:   uint64(60 * time.Second),
								}}}}},
			},
			expectedKeys: []uint32{util.TokenFor(util.FakeTenantID, traceIDA), util.TokenFor(util.FakeTenantID, traceIDB)},
			expectedTraces: []*tempopb.Trace{
				{
					Batches: []*v1.ResourceSpans{
						{
							InstrumentationLibrarySpans: []*v1.InstrumentationLibrarySpans{
								{
									Spans: []*v1.Span{
										{
											TraceId:           traceIDA,
											StartTimeUnixNano: uint64(30 * time.Second),
											EndTimeUnixNano:   uint64(40 * time.Second),
										}}}}}},
				},
				{
					Batches: []*v1.ResourceSpans{
						{
							InstrumentationLibrarySpans: []*v1.InstrumentationLibrarySpans{
								{
									Spans: []*v1.Span{
										{
											TraceId:           traceIDB,
											StartTimeUnixNano: uint64(50 * time.Second),
											EndTimeUnixNano:   uint64(60 * time.Second),
										}}}}}},
				},
			},
			expectedIDs: [][]byte{
				traceIDA,
				traceIDB,
			},
			expectedStarts: []uint32{30, 50},
			expectedEnds:   []uint32{40, 60},
		},
		{
			name: "two traces, distinct batches",
			batches: []*v1.ResourceSpans{
				{
					Resource: &v1_resource.Resource{
						DroppedAttributesCount: 3,
					},
					InstrumentationLibrarySpans: []*v1.InstrumentationLibrarySpans{
						{
							Spans: []*v1.Span{
								{
									TraceId:           traceIDA,
									StartTimeUnixNano: uint64(30 * time.Second),
									EndTimeUnixNano:   uint64(40 * time.Second),
								}}}}},
				{
					Resource: &v1_resource.Resource{
						DroppedAttributesCount: 4,
					},
					InstrumentationLibrarySpans: []*v1.InstrumentationLibrarySpans{
						{
							Spans: []*v1.Span{
								{
									TraceId:           traceIDB,
									StartTimeUnixNano: uint64(50 * time.Second),
									EndTimeUnixNano:   uint64(60 * time.Second),
								}}}}},
			},
			expectedKeys: []uint32{util.TokenFor(util.FakeTenantID, traceIDA), util.TokenFor(util.FakeTenantID, traceIDB)},
			expectedTraces: []*tempopb.Trace{
				{
					Batches: []*v1.ResourceSpans{
						{
							Resource: &v1_resource.Resource{
								DroppedAttributesCount: 3,
							},
							InstrumentationLibrarySpans: []*v1.InstrumentationLibrarySpans{
								{
									Spans: []*v1.Span{
										{
											TraceId:           traceIDA,
											StartTimeUnixNano: uint64(30 * time.Second),
											EndTimeUnixNano:   uint64(40 * time.Second),
										}}}}}},
				},
				{
					Batches: []*v1.ResourceSpans{
						{
							Resource: &v1_resource.Resource{
								DroppedAttributesCount: 4,
							},
							InstrumentationLibrarySpans: []*v1.InstrumentationLibrarySpans{
								{
									Spans: []*v1.Span{
										{
											TraceId:           traceIDB,
											StartTimeUnixNano: uint64(50 * time.Second),
											EndTimeUnixNano:   uint64(60 * time.Second),
										}}}}}},
				},
			},
			expectedIDs: [][]byte{
				traceIDA,
				traceIDB,
			},
			expectedStarts: []uint32{30, 50},
			expectedEnds:   []uint32{40, 60},
		},
		{
			name: "resource copied",
			batches: []*v1.ResourceSpans{
				{
					Resource: &v1_resource.Resource{
						DroppedAttributesCount: 1,
					},
					InstrumentationLibrarySpans: []*v1.InstrumentationLibrarySpans{
						{
							Spans: []*v1.Span{
								{
									TraceId:           traceIDA,
									StartTimeUnixNano: uint64(30 * time.Second),
									EndTimeUnixNano:   uint64(40 * time.Second),
								},
								{
									TraceId:           traceIDB,
									StartTimeUnixNano: uint64(50 * time.Second),
									EndTimeUnixNano:   uint64(60 * time.Second),
								}}}}},
			},
			expectedKeys: []uint32{util.TokenFor(util.FakeTenantID, traceIDA), util.TokenFor(util.FakeTenantID, traceIDB)},
			expectedTraces: []*tempopb.Trace{
				{
					Batches: []*v1.ResourceSpans{
						{
							Resource: &v1_resource.Resource{
								DroppedAttributesCount: 1,
							},
							InstrumentationLibrarySpans: []*v1.InstrumentationLibrarySpans{
								{
									Spans: []*v1.Span{
										{
											TraceId:           traceIDA,
											StartTimeUnixNano: uint64(30 * time.Second),
											EndTimeUnixNano:   uint64(40 * time.Second),
										}}}}}},
				},
				{
					Batches: []*v1.ResourceSpans{
						{
							Resource: &v1_resource.Resource{
								DroppedAttributesCount: 1,
							},
							InstrumentationLibrarySpans: []*v1.InstrumentationLibrarySpans{
								{
									Spans: []*v1.Span{
										{
											TraceId:           traceIDB,
											StartTimeUnixNano: uint64(50 * time.Second),
											EndTimeUnixNano:   uint64(60 * time.Second),
										}}}}}},
				},
			},
			expectedIDs: [][]byte{
				traceIDA,
				traceIDB,
			},
			expectedStarts: []uint32{30, 50},
			expectedEnds:   []uint32{40, 60},
		},
		{
			name: "ils copied",
			batches: []*v1.ResourceSpans{
				{
					InstrumentationLibrarySpans: []*v1.InstrumentationLibrarySpans{
						{
							InstrumentationLibrary: &v1_common.InstrumentationLibrary{
								Name: "test",
							},
							Spans: []*v1.Span{
								{
									TraceId:           traceIDA,
									StartTimeUnixNano: uint64(30 * time.Second),
									EndTimeUnixNano:   uint64(40 * time.Second),
								},
								{
									TraceId:           traceIDB,
									StartTimeUnixNano: uint64(50 * time.Second),
									EndTimeUnixNano:   uint64(60 * time.Second),
								}}}}},
			},
			expectedKeys: []uint32{util.TokenFor(util.FakeTenantID, traceIDA), util.TokenFor(util.FakeTenantID, traceIDB)},
			expectedTraces: []*tempopb.Trace{
				{
					Batches: []*v1.ResourceSpans{
						{
							InstrumentationLibrarySpans: []*v1.InstrumentationLibrarySpans{
								{
									InstrumentationLibrary: &v1_common.InstrumentationLibrary{
										Name: "test",
									},
									Spans: []*v1.Span{
										{
											TraceId:           traceIDA,
											StartTimeUnixNano: uint64(30 * time.Second),
											EndTimeUnixNano:   uint64(40 * time.Second),
										}}}}}},
				},
				{
					Batches: []*v1.ResourceSpans{
						{
							InstrumentationLibrarySpans: []*v1.InstrumentationLibrarySpans{
								{
									InstrumentationLibrary: &v1_common.InstrumentationLibrary{
										Name: "test",
									},
									Spans: []*v1.Span{
										{
											TraceId:           traceIDB,
											StartTimeUnixNano: uint64(50 * time.Second),
											EndTimeUnixNano:   uint64(60 * time.Second),
										}}}}}},
				},
			},
			expectedIDs: [][]byte{
				traceIDA,
				traceIDB,
			},
			expectedStarts: []uint32{30, 50},
			expectedEnds:   []uint32{40, 60},
		},
		{
			name: "one trace",
			batches: []*v1.ResourceSpans{
				{
					Resource: &v1_resource.Resource{
						DroppedAttributesCount: 3,
					},
					InstrumentationLibrarySpans: []*v1.InstrumentationLibrarySpans{
						{
							InstrumentationLibrary: &v1_common.InstrumentationLibrary{
								Name: "test",
							},
							Spans: []*v1.Span{
								{
									TraceId:           traceIDB,
									Name:              "spanA",
									StartTimeUnixNano: uint64(30 * time.Second),
									EndTimeUnixNano:   uint64(40 * time.Second),
								},
								{
									TraceId:           traceIDB,
									Name:              "spanB",
									StartTimeUnixNano: uint64(50 * time.Second),
									EndTimeUnixNano:   uint64(60 * time.Second),
								}}}}},
			},
			expectedKeys: []uint32{util.TokenFor(util.FakeTenantID, traceIDB)},
			expectedTraces: []*tempopb.Trace{
				{
					Batches: []*v1.ResourceSpans{
						{
							Resource: &v1_resource.Resource{
								DroppedAttributesCount: 3,
							},
							InstrumentationLibrarySpans: []*v1.InstrumentationLibrarySpans{
								{
									InstrumentationLibrary: &v1_common.InstrumentationLibrary{
										Name: "test",
									},
									Spans: []*v1.Span{
										{
											TraceId:           traceIDB,
											Name:              "spanA",
											StartTimeUnixNano: uint64(30 * time.Second),
											EndTimeUnixNano:   uint64(40 * time.Second),
										},
										{
											TraceId:           traceIDB,
											Name:              "spanB",
											StartTimeUnixNano: uint64(50 * time.Second),
											EndTimeUnixNano:   uint64(60 * time.Second),
										}}}}}},
				},
			},
			expectedIDs: [][]byte{
				traceIDB,
			},
			expectedStarts: []uint32{30},
			expectedEnds:   []uint32{60},
		},
		{
			name: "two traces - two batches - don't combine across batches",
			batches: []*v1.ResourceSpans{
				{
					Resource: &v1_resource.Resource{
						DroppedAttributesCount: 3,
					},
					InstrumentationLibrarySpans: []*v1.InstrumentationLibrarySpans{
						{
							InstrumentationLibrary: &v1_common.InstrumentationLibrary{
								Name: "test",
							},
							Spans: []*v1.Span{
								{
									TraceId:           traceIDB,
									Name:              "spanA",
									StartTimeUnixNano: uint64(30 * time.Second),
									EndTimeUnixNano:   uint64(40 * time.Second),
								},
								{
									TraceId:           traceIDB,
									Name:              "spanC",
									StartTimeUnixNano: uint64(20 * time.Second),
									EndTimeUnixNano:   uint64(50 * time.Second),
								},
								{
									TraceId:           traceIDA,
									Name:              "spanE",
									StartTimeUnixNano: uint64(70 * time.Second),
									EndTimeUnixNano:   uint64(80 * time.Second),
								}}}}},
				{
					Resource: &v1_resource.Resource{
						DroppedAttributesCount: 4,
					},
					InstrumentationLibrarySpans: []*v1.InstrumentationLibrarySpans{
						{
							InstrumentationLibrary: &v1_common.InstrumentationLibrary{
								Name: "test2",
							},
							Spans: []*v1.Span{
								{
									TraceId:           traceIDB,
									Name:              "spanB",
									StartTimeUnixNano: uint64(10 * time.Second),
									EndTimeUnixNano:   uint64(30 * time.Second),
								},
								{
									TraceId:           traceIDA,
									Name:              "spanD",
									StartTimeUnixNano: uint64(60 * time.Second),
									EndTimeUnixNano:   uint64(80 * time.Second),
								}}}}},
			},
			expectedKeys: []uint32{
				util.TokenFor(util.FakeTenantID, traceIDB),
				util.TokenFor(util.FakeTenantID, traceIDA),
			},
			expectedTraces: []*tempopb.Trace{
				{
					Batches: []*v1.ResourceSpans{
						{
							Resource: &v1_resource.Resource{
								DroppedAttributesCount: 3,
							},
							InstrumentationLibrarySpans: []*v1.InstrumentationLibrarySpans{
								{
									InstrumentationLibrary: &v1_common.InstrumentationLibrary{
										Name: "test",
									},
									Spans: []*v1.Span{
										{
											TraceId:           traceIDB,
											Name:              "spanA",
											StartTimeUnixNano: uint64(30 * time.Second),
											EndTimeUnixNano:   uint64(40 * time.Second),
										},
										{
											TraceId:           traceIDB,
											Name:              "spanC",
											StartTimeUnixNano: uint64(20 * time.Second),
											EndTimeUnixNano:   uint64(50 * time.Second),
										}}}}},
						{
							Resource: &v1_resource.Resource{
								DroppedAttributesCount: 4,
							},
							InstrumentationLibrarySpans: []*v1.InstrumentationLibrarySpans{
								{
									InstrumentationLibrary: &v1_common.InstrumentationLibrary{
										Name: "test2",
									},
									Spans: []*v1.Span{
										{
											TraceId:           traceIDB,
											Name:              "spanB",
											StartTimeUnixNano: uint64(10 * time.Second),
											EndTimeUnixNano:   uint64(30 * time.Second),
										}}}}},
					},
				},
				{
					Batches: []*v1.ResourceSpans{
						{
							Resource: &v1_resource.Resource{
								DroppedAttributesCount: 3,
							},
							InstrumentationLibrarySpans: []*v1.InstrumentationLibrarySpans{
								{
									InstrumentationLibrary: &v1_common.InstrumentationLibrary{
										Name: "test",
									},
									Spans: []*v1.Span{
										{
											TraceId:           traceIDA,
											Name:              "spanE",
											StartTimeUnixNano: uint64(70 * time.Second),
											EndTimeUnixNano:   uint64(80 * time.Second),
										}}}}},
						{
							Resource: &v1_resource.Resource{
								DroppedAttributesCount: 4,
							},
							InstrumentationLibrarySpans: []*v1.InstrumentationLibrarySpans{
								{
									InstrumentationLibrary: &v1_common.InstrumentationLibrary{
										Name: "test2",
									},
									Spans: []*v1.Span{
										{
											TraceId:           traceIDA,
											Name:              "spanD",
											StartTimeUnixNano: uint64(60 * time.Second),
											EndTimeUnixNano:   uint64(80 * time.Second),
										}}}}},
					},
				},
			},
			expectedIDs: [][]byte{
				traceIDB,
				traceIDA,
			},
			expectedStarts: []uint32{10, 60},
			expectedEnds:   []uint32{50, 80},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keys, rebatchedTraces, err := requestsByTraceID(tt.batches, util.FakeTenantID, 1)
			require.Equal(t, len(keys), len(rebatchedTraces))

			for i, expectedKey := range tt.expectedKeys {
				foundIndex := -1
				for j, key := range keys {
					if expectedKey == key {
						foundIndex = j
					}
				}
				require.NotEqual(t, -1, foundIndex, "expected key %d not found", foundIndex)

				// now confirm that the request at this position is the expected one
				expectedReq := tt.expectedTraces[i]
				actualReq := rebatchedTraces[foundIndex].trace
				assert.Equal(t, expectedReq, actualReq)
				assert.Equal(t, tt.expectedIDs[i], rebatchedTraces[foundIndex].id)
				assert.Equal(t, tt.expectedStarts[i], rebatchedTraces[foundIndex].start)
				assert.Equal(t, tt.expectedEnds[i], rebatchedTraces[foundIndex].end)
			}

			assert.Equal(t, tt.expectedErr, err)
		})
	}
}

func BenchmarkTestsByRequestID(b *testing.B) {
	spansPer := 100
	batches := 10
	traces := []*tempopb.Trace{
		test.MakeTraceWithSpanCount(batches, spansPer, []byte{0x0A, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F}),
		test.MakeTraceWithSpanCount(batches, spansPer, []byte{0x0B, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F}),
		test.MakeTraceWithSpanCount(batches, spansPer, []byte{0x0C, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F}),
		test.MakeTraceWithSpanCount(batches, spansPer, []byte{0x0D, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F}),
	}
	ils := make([][]*v1.InstrumentationLibrarySpans, batches)

	for i := 0; i < batches; i++ {
		for _, t := range traces {
			ils[i] = append(ils[i], t.Batches[i].InstrumentationLibrarySpans...)
		}
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		for _, blerg := range ils {
			_, _, err := requestsByTraceID([]*v1.ResourceSpans{
				{
					InstrumentationLibrarySpans: blerg,
				},
			}, "test", spansPer*len(traces))
			require.NoError(b, err)
		}
	}
}

func TestDistributor(t *testing.T) {
	for i, tc := range []struct {
		lines            int
		expectedResponse *tempopb.PushResponse
		expectedError    error
	}{
		{
			lines:            10,
			expectedResponse: nil,
		},
		{
			lines:            100,
			expectedResponse: nil,
		},
	} {
		t.Run(fmt.Sprintf("[%d](samples=%v)", i, tc.lines), func(t *testing.T) {
			limits := &overrides.Limits{}
			flagext.DefaultValues(limits)

			// todo:  test limits
			d := prepare(t, limits, nil, nil)

			b := test.MakeBatch(tc.lines, []byte{})
			response, err := d.PushBatches(ctx, []*v1.ResourceSpans{b})

			assert.True(t, proto.Equal(tc.expectedResponse, response))
			assert.Equal(t, tc.expectedError, err)
		})
	}
}

func TestLogSpans(t *testing.T) {
	for i, tc := range []struct {
		LogReceivedTraces       bool // Backwards compatibility with old config
		LogReceivedSpansEnabled bool
		filterByStatusError     bool
		includeAttributes       bool
		batches                 []*v1.ResourceSpans
		expectedLogSpans        []logSpan
	}{
		{
			LogReceivedSpansEnabled: false,
			batches: []*v1.ResourceSpans{
				makeResourceSpans("test", []*v1.InstrumentationLibrarySpans{
					makeInstrumentationLibrary(
						makeSpan("0a0102030405060708090a0b0c0d0e0f", "dad44adc9a83b370", nil)),
				}),
			},
			expectedLogSpans: []logSpan{},
		},
		{
			LogReceivedTraces: true,
			batches: []*v1.ResourceSpans{
				makeResourceSpans("test", []*v1.InstrumentationLibrarySpans{
					makeInstrumentationLibrary(
						makeSpan("0a0102030405060708090a0b0c0d0e0f", "dad44adc9a83b370", nil)),
				}),
			},
			expectedLogSpans: []logSpan{
				{
					Msg:     "received",
					Level:   "info",
					TraceID: "0a0102030405060708090a0b0c0d0e0f",
					SpanID:  "dad44adc9a83b370",
				},
			},
		},
		{
			LogReceivedSpansEnabled: true,
			filterByStatusError:     false,
			batches: []*v1.ResourceSpans{
				makeResourceSpans("test-service", []*v1.InstrumentationLibrarySpans{
					makeInstrumentationLibrary(
						makeSpan("0a0102030405060708090a0b0c0d0e0f", "dad44adc9a83b370", nil),
						makeSpan("e3210a2b38097332d1fe43083ea93d29", "6c21c48da4dbd1a7", nil)),
					makeInstrumentationLibrary(
						makeSpan("bb42ec04df789ff04b10ea5274491685", "1b3a296034f4031e", nil)),
				}),
				makeResourceSpans("test-service2", []*v1.InstrumentationLibrarySpans{
					makeInstrumentationLibrary(
						makeSpan("b1c792dea27d511c145df8402bdd793a", "56afb9fe18b6c2d6", nil)),
				}),
			},
			expectedLogSpans: []logSpan{
				{
					Msg:     "received",
					Level:   "info",
					TraceID: "0a0102030405060708090a0b0c0d0e0f",
					SpanID:  "dad44adc9a83b370",
				},
				{
					Msg:     "received",
					Level:   "info",
					TraceID: "e3210a2b38097332d1fe43083ea93d29",
					SpanID:  "6c21c48da4dbd1a7",
				},
				{
					Msg:     "received",
					Level:   "info",
					TraceID: "bb42ec04df789ff04b10ea5274491685",
					SpanID:  "1b3a296034f4031e",
				},
				{
					Msg:     "received",
					Level:   "info",
					TraceID: "b1c792dea27d511c145df8402bdd793a",
					SpanID:  "56afb9fe18b6c2d6",
				},
			},
		},
		{
			LogReceivedSpansEnabled: true,
			filterByStatusError:     true,
			batches: []*v1.ResourceSpans{
				makeResourceSpans("test-service", []*v1.InstrumentationLibrarySpans{
					makeInstrumentationLibrary(
						makeSpan("0a0102030405060708090a0b0c0d0e0f", "dad44adc9a83b370", nil),
						makeSpan("e3210a2b38097332d1fe43083ea93d29", "6c21c48da4dbd1a7", &v1.Status{Code: v1.Status_STATUS_CODE_ERROR})),
					makeInstrumentationLibrary(
						makeSpan("bb42ec04df789ff04b10ea5274491685", "1b3a296034f4031e", nil)),
				}),
				makeResourceSpans("test-service2", []*v1.InstrumentationLibrarySpans{
					makeInstrumentationLibrary(
						makeSpan("b1c792dea27d511c145df8402bdd793a", "56afb9fe18b6c2d6", &v1.Status{Code: v1.Status_STATUS_CODE_ERROR})),
				}),
			},
			expectedLogSpans: []logSpan{
				{
					Msg:     "received",
					Level:   "info",
					TraceID: "e3210a2b38097332d1fe43083ea93d29",
					SpanID:  "6c21c48da4dbd1a7",
				},
				{
					Msg:     "received",
					Level:   "info",
					TraceID: "b1c792dea27d511c145df8402bdd793a",
					SpanID:  "56afb9fe18b6c2d6",
				},
			},
		},
		{
			LogReceivedSpansEnabled: true,
			filterByStatusError:     true,
			includeAttributes:       true,
			batches: []*v1.ResourceSpans{
				makeResourceSpans("test-service", []*v1.InstrumentationLibrarySpans{
					makeInstrumentationLibrary(
						makeSpan("0a0102030405060708090a0b0c0d0e0f", "dad44adc9a83b370", nil,
							makeAttribute("tag1", "value1")),
						makeSpan("e3210a2b38097332d1fe43083ea93d29", "6c21c48da4dbd1a7", &v1.Status{Code: v1.Status_STATUS_CODE_ERROR},
							makeAttribute("tag1", "value1"),
							makeAttribute("tag2", "value2"))),
					makeInstrumentationLibrary(
						makeSpan("bb42ec04df789ff04b10ea5274491685", "1b3a296034f4031e", nil)),
				}, makeAttribute("resource_attribute1", "value1")),
				makeResourceSpans("test-service2", []*v1.InstrumentationLibrarySpans{
					makeInstrumentationLibrary(
						makeSpan("b1c792dea27d511c145df8402bdd793a", "56afb9fe18b6c2d6", &v1.Status{Code: v1.Status_STATUS_CODE_ERROR})),
				}, makeAttribute("resource_attribute2", "value2")),
			},
			expectedLogSpans: []logSpan{
				{
					Msg:                "received",
					Level:              "info",
					TraceID:            "e3210a2b38097332d1fe43083ea93d29",
					SpanID:             "6c21c48da4dbd1a7",
					SpanServiceName:    "test-service",
					SpanStatus:         "STATUS_CODE_ERROR",
					SpanKind:           "SPAN_KIND_SERVER",
					SpanTag1:           "value1",
					SpanTag2:           "value2",
					ResourceAttribute1: "value1",
				},
				{
					Msg:                "received",
					Level:              "info",
					TraceID:            "b1c792dea27d511c145df8402bdd793a",
					SpanID:             "56afb9fe18b6c2d6",
					SpanServiceName:    "test-service2",
					SpanStatus:         "STATUS_CODE_ERROR",
					SpanKind:           "SPAN_KIND_SERVER",
					ResourceAttribute2: "value2",
				},
			},
		},
		{
			LogReceivedSpansEnabled: true,
			filterByStatusError:     false,
			includeAttributes:       true,
			batches: []*v1.ResourceSpans{
				makeResourceSpans("test-service", []*v1.InstrumentationLibrarySpans{
					makeInstrumentationLibrary(
						makeSpan("0a0102030405060708090a0b0c0d0e0f", "dad44adc9a83b370", nil, makeAttribute("tag1", "value1"))),
				}),
			},
			expectedLogSpans: []logSpan{
				{
					Msg:             "received",
					Level:           "info",
					TraceID:         "0a0102030405060708090a0b0c0d0e0f",
					SpanID:          "dad44adc9a83b370",
					SpanServiceName: "test-service",
					SpanStatus:      "STATUS_CODE_OK",
					SpanKind:        "SPAN_KIND_SERVER",
					SpanTag1:        "value1",
				},
			},
		},
	} {
		t.Run(fmt.Sprintf("[%d] TestLogSpans LogReceivedTraces=%v LogReceivedSpansEnabled=%v filterByStatusError=%v includeAttributes=%v", i, tc.LogReceivedTraces, tc.LogReceivedSpansEnabled, tc.filterByStatusError, tc.includeAttributes), func(t *testing.T) {
			limits := &overrides.Limits{}
			flagext.DefaultValues(limits)

			buf := &bytes.Buffer{}
			logger := kitlog.NewJSONLogger(kitlog.NewSyncWriter(buf))

			d := prepare(t, limits, nil, logger)
			d.cfg.LogReceivedTraces = tc.LogReceivedTraces
			d.cfg.LogReceivedSpans = LogReceivedSpansConfig{
				Enabled:             tc.LogReceivedSpansEnabled,
				FilterByStatusError: tc.filterByStatusError,
				IncludeAttributes:   tc.includeAttributes,
			}

			_, err := d.PushBatches(ctx, tc.batches)
			if err != nil {
				t.Fatal(err)
			}

			bufJSON := "[" + strings.TrimRight(strings.ReplaceAll(buf.String(), "\n", ","), ",") + "]"
			var actualLogsSpan []logSpan
			err = json.Unmarshal([]byte(bufJSON), &actualLogsSpan)
			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, len(tc.expectedLogSpans), len(actualLogsSpan))
			for i, expectedLogSpan := range tc.expectedLogSpans {
				assert.EqualValues(t, expectedLogSpan, actualLogsSpan[i])
			}
		})
	}
}

type logSpan struct {
	Msg                string `json:"msg"`
	Level              string `json:"level"`
	TraceID            string `json:"traceid"`
	SpanID             string `json:"spanid"`
	SpanStatus         string `json:"span_status,omitempty"`
	SpanKind           string `json:"span_kind,omitempty"`
	SpanServiceName    string `json:"span_service_name,omitempty"`
	SpanTag1           string `json:"span_tag1,omitempty"`
	SpanTag2           string `json:"span_tag2,omitempty"`
	ResourceAttribute1 string `json:"span_resource_attribute1,omitempty"`
	ResourceAttribute2 string `json:"span_resource_attribute2,omitempty"`
}

func makeAttribute(key string, value string) *v1_common.KeyValue {
	return &v1_common.KeyValue{
		Key:   key,
		Value: &v1_common.AnyValue{Value: &v1_common.AnyValue_StringValue{StringValue: value}},
	}
}

func makeSpan(traceID string, spanID string, status *v1.Status, attributes ...*v1_common.KeyValue) *v1.Span {
	if status == nil {
		status = &v1.Status{Code: v1.Status_STATUS_CODE_OK}
	}

	traceIDBytes, err := hex.DecodeString(traceID)
	if err != nil {
		panic(err)
	}
	spanIDBytes, err := hex.DecodeString(spanID)
	if err != nil {
		panic(err)
	}

	return &v1.Span{
		TraceId:    traceIDBytes,
		SpanId:     spanIDBytes,
		Status:     status,
		Kind:       v1.Span_SPAN_KIND_SERVER,
		Attributes: attributes,
	}
}

func makeInstrumentationLibrary(spans ...*v1.Span) *v1.InstrumentationLibrarySpans {
	return &v1.InstrumentationLibrarySpans{
		InstrumentationLibrary: &v1_common.InstrumentationLibrary{
			Name:    "super library",
			Version: "0.0.1",
		},
		Spans: spans,
	}
}

func makeResourceSpans(serviceName string, ils []*v1.InstrumentationLibrarySpans, attributes ...*v1_common.KeyValue) *v1.ResourceSpans {
	rs := &v1.ResourceSpans{
		Resource: &v1_resource.Resource{
			Attributes: []*v1_common.KeyValue{
				{
					Key: "service.name",
					Value: &v1_common.AnyValue{
						Value: &v1_common.AnyValue_StringValue{
							StringValue: serviceName,
						},
					},
				},
			},
		},
		InstrumentationLibrarySpans: ils,
	}

	rs.Resource.Attributes = append(rs.Resource.Attributes, attributes...)

	return rs
}

func prepare(t *testing.T, limits *overrides.Limits, kvStore kv.Client, logger log.Logger) *Distributor {
	if logger == nil {
		logger = log.NewNopLogger()
	}

	var (
		distributorConfig Config
		clientConfig      ingester_client.Config
	)
	flagext.DefaultValues(&clientConfig)

	overrides, err := overrides.NewOverrides(*limits)
	require.NoError(t, err)

	// Mock the ingesters ring
	ingesters := map[string]*mockIngester{}
	for i := 0; i < numIngesters; i++ {
		ingesters[fmt.Sprintf("ingester%d", i)] = &mockIngester{}
	}

	ingestersRing := &mockRing{
		replicationFactor: 3,
	}
	for addr := range ingesters {
		ingestersRing.ingesters = append(ingestersRing.ingesters, ring.InstanceDesc{
			Addr: addr,
		})
	}

	distributorConfig.DistributorRing.HeartbeatPeriod = 100 * time.Millisecond
	distributorConfig.DistributorRing.InstanceID = strconv.Itoa(rand.Int())
	distributorConfig.DistributorRing.KVStore.Mock = kvStore
	distributorConfig.DistributorRing.InstanceInterfaceNames = []string{"eth0", "en0", "lo0"}
	distributorConfig.factory = func(addr string) (ring_client.PoolClient, error) {
		return ingesters[addr], nil
	}

	l := logging.Level{}
	_ = l.Set("error")
	mw := receiver.MultiTenancyMiddleware()
	d, err := New(distributorConfig, clientConfig, ingestersRing, generator_client.Config{}, nil, overrides, mw, logger, l, false, false, prometheus.NewPedanticRegistry())
	require.NoError(t, err)

	return d
}

type mockIngester struct {
	grpc_health_v1.HealthClient
}

var _ tempopb.PusherClient = (*mockIngester)(nil)

func (i *mockIngester) PushBytes(ctx context.Context, in *tempopb.PushBytesRequest, opts ...grpc.CallOption) (*tempopb.PushResponse, error) {
	return nil, nil
}

func (i *mockIngester) PushBytesV2(ctx context.Context, in *tempopb.PushBytesRequest, opts ...grpc.CallOption) (*tempopb.PushResponse, error) {
	return nil, nil
}

func (i *mockIngester) Close() error {
	return nil
}

// Copied from Cortex; TODO(twilkie) - factor this our and share it.
// mockRing doesn't do virtual nodes, just returns mod(key) + replicationFactor
// ingesters.
type mockRing struct {
	prometheus.Counter
	ingesters         []ring.InstanceDesc
	replicationFactor uint32
}

var _ ring.ReadRing = (*mockRing)(nil)

func (r mockRing) Get(key uint32, op ring.Operation, buf []ring.InstanceDesc, _, _ []string) (ring.ReplicationSet, error) {
	result := ring.ReplicationSet{
		MaxErrors: 1,
		Instances: buf[:0],
	}
	for i := uint32(0); i < r.replicationFactor; i++ {
		n := (key + i) % uint32(len(r.ingesters))
		result.Instances = append(result.Instances, r.ingesters[n])
	}
	return result, nil
}

func (r mockRing) GetAllHealthy(op ring.Operation) (ring.ReplicationSet, error) {
	return ring.ReplicationSet{
		Instances: r.ingesters,
		MaxErrors: 1,
	}, nil
}

func (r mockRing) GetReplicationSetForOperation(op ring.Operation) (ring.ReplicationSet, error) {
	return r.GetAllHealthy(op)
}

func (r mockRing) ReplicationFactor() int {
	return int(r.replicationFactor)
}

func (r mockRing) ShuffleShard(identifier string, size int) ring.ReadRing {
	return r
}

func (r mockRing) ShuffleShardWithLookback(string, int, time.Duration, time.Time) ring.ReadRing {
	return r
}

func (r mockRing) InstancesCount() int {
	return len(r.ingesters)
}

func (r mockRing) HasInstance(instanceID string) bool {
	return true
}

func (r mockRing) CleanupShuffleShardCache(identifier string) {
}

func (r mockRing) GetInstanceState(instanceID string) (ring.InstanceState, error) {
	return ring.ACTIVE, nil
}
