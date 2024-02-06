package distributor

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"testing"
	"time"

	kitlog "github.com/go-kit/log"
	"github.com/gogo/status"
	"github.com/golang/protobuf/proto" // nolint: all  //ProtoReflect
	"github.com/grafana/dskit/flagext"
	dslog "github.com/grafana/dskit/log"
	"github.com/grafana/dskit/ring"
	ring_client "github.com/grafana/dskit/ring/client"
	"github.com/grafana/dskit/user"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/ptrace"
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
	numIngesters       = 5
	noError            = tempopb.PushErrorReason_NO_ERROR
	maxLiveTraceError  = tempopb.PushErrorReason_MAX_LIVE_TRACES
	traceTooLargeError = tempopb.PushErrorReason_TRACE_TOO_LARGE
)

var ctx = user.InjectOrgID(context.Background(), "test")

func batchesToTraces(t *testing.T, batches []*v1.ResourceSpans) ptrace.Traces {
	t.Helper()

	trace := tempopb.Trace{Batches: batches}

	m, err := trace.Marshal()
	require.NoError(t, err)

	traces, err := (&ptrace.ProtoUnmarshaler{}).UnmarshalTraces(m)
	require.NoError(t, err)

	return traces
}

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
					ScopeSpans: []*v1.ScopeSpans{
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
					ScopeSpans: []*v1.ScopeSpans{
						{
							Spans: []*v1.Span{
								{
									TraceId:           traceIDA,
									StartTimeUnixNano: uint64(10 * time.Second),
									EndTimeUnixNano:   uint64(20 * time.Second),
								},
							},
						},
					},
				},
			},
			expectedKeys: []uint32{util.TokenFor(util.FakeTenantID, traceIDA)},
			expectedTraces: []*tempopb.Trace{
				{
					Batches: []*v1.ResourceSpans{
						{
							ScopeSpans: []*v1.ScopeSpans{
								{
									Spans: []*v1.Span{
										{
											TraceId:           traceIDA,
											StartTimeUnixNano: uint64(10 * time.Second),
											EndTimeUnixNano:   uint64(20 * time.Second),
										},
									},
								},
							},
						},
					},
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
					ScopeSpans: []*v1.ScopeSpans{
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
								},
							},
						},
					},
				},
			},
			expectedKeys: []uint32{util.TokenFor(util.FakeTenantID, traceIDA), util.TokenFor(util.FakeTenantID, traceIDB)},
			expectedTraces: []*tempopb.Trace{
				{
					Batches: []*v1.ResourceSpans{
						{
							ScopeSpans: []*v1.ScopeSpans{
								{
									Spans: []*v1.Span{
										{
											TraceId:           traceIDA,
											StartTimeUnixNano: uint64(30 * time.Second),
											EndTimeUnixNano:   uint64(40 * time.Second),
										},
									},
								},
							},
						},
					},
				},
				{
					Batches: []*v1.ResourceSpans{
						{
							ScopeSpans: []*v1.ScopeSpans{
								{
									Spans: []*v1.Span{
										{
											TraceId:           traceIDB,
											StartTimeUnixNano: uint64(50 * time.Second),
											EndTimeUnixNano:   uint64(60 * time.Second),
										},
									},
								},
							},
						},
					},
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
					ScopeSpans: []*v1.ScopeSpans{
						{
							Spans: []*v1.Span{
								{
									TraceId:           traceIDA,
									StartTimeUnixNano: uint64(30 * time.Second),
									EndTimeUnixNano:   uint64(40 * time.Second),
								},
							},
						},
					},
				},
				{
					Resource: &v1_resource.Resource{
						DroppedAttributesCount: 4,
					},
					ScopeSpans: []*v1.ScopeSpans{
						{
							Spans: []*v1.Span{
								{
									TraceId:           traceIDB,
									StartTimeUnixNano: uint64(50 * time.Second),
									EndTimeUnixNano:   uint64(60 * time.Second),
								},
							},
						},
					},
				},
			},
			expectedKeys: []uint32{util.TokenFor(util.FakeTenantID, traceIDA), util.TokenFor(util.FakeTenantID, traceIDB)},
			expectedTraces: []*tempopb.Trace{
				{
					Batches: []*v1.ResourceSpans{
						{
							Resource: &v1_resource.Resource{
								DroppedAttributesCount: 3,
							},
							ScopeSpans: []*v1.ScopeSpans{
								{
									Spans: []*v1.Span{
										{
											TraceId:           traceIDA,
											StartTimeUnixNano: uint64(30 * time.Second),
											EndTimeUnixNano:   uint64(40 * time.Second),
										},
									},
								},
							},
						},
					},
				},
				{
					Batches: []*v1.ResourceSpans{
						{
							Resource: &v1_resource.Resource{
								DroppedAttributesCount: 4,
							},
							ScopeSpans: []*v1.ScopeSpans{
								{
									Spans: []*v1.Span{
										{
											TraceId:           traceIDB,
											StartTimeUnixNano: uint64(50 * time.Second),
											EndTimeUnixNano:   uint64(60 * time.Second),
										},
									},
								},
							},
						},
					},
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
					ScopeSpans: []*v1.ScopeSpans{
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
								},
							},
						},
					},
				},
			},
			expectedKeys: []uint32{util.TokenFor(util.FakeTenantID, traceIDA), util.TokenFor(util.FakeTenantID, traceIDB)},
			expectedTraces: []*tempopb.Trace{
				{
					Batches: []*v1.ResourceSpans{
						{
							Resource: &v1_resource.Resource{
								DroppedAttributesCount: 1,
							},
							ScopeSpans: []*v1.ScopeSpans{
								{
									Spans: []*v1.Span{
										{
											TraceId:           traceIDA,
											StartTimeUnixNano: uint64(30 * time.Second),
											EndTimeUnixNano:   uint64(40 * time.Second),
										},
									},
								},
							},
						},
					},
				},
				{
					Batches: []*v1.ResourceSpans{
						{
							Resource: &v1_resource.Resource{
								DroppedAttributesCount: 1,
							},
							ScopeSpans: []*v1.ScopeSpans{
								{
									Spans: []*v1.Span{
										{
											TraceId:           traceIDB,
											StartTimeUnixNano: uint64(50 * time.Second),
											EndTimeUnixNano:   uint64(60 * time.Second),
										},
									},
								},
							},
						},
					},
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
					ScopeSpans: []*v1.ScopeSpans{
						{
							Scope: &v1_common.InstrumentationScope{
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
								},
							},
						},
					},
				},
			},
			expectedKeys: []uint32{util.TokenFor(util.FakeTenantID, traceIDA), util.TokenFor(util.FakeTenantID, traceIDB)},
			expectedTraces: []*tempopb.Trace{
				{
					Batches: []*v1.ResourceSpans{
						{
							ScopeSpans: []*v1.ScopeSpans{
								{
									Scope: &v1_common.InstrumentationScope{
										Name: "test",
									},
									Spans: []*v1.Span{
										{
											TraceId:           traceIDA,
											StartTimeUnixNano: uint64(30 * time.Second),
											EndTimeUnixNano:   uint64(40 * time.Second),
										},
									},
								},
							},
						},
					},
				},
				{
					Batches: []*v1.ResourceSpans{
						{
							ScopeSpans: []*v1.ScopeSpans{
								{
									Scope: &v1_common.InstrumentationScope{
										Name: "test",
									},
									Spans: []*v1.Span{
										{
											TraceId:           traceIDB,
											StartTimeUnixNano: uint64(50 * time.Second),
											EndTimeUnixNano:   uint64(60 * time.Second),
										},
									},
								},
							},
						},
					},
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
					ScopeSpans: []*v1.ScopeSpans{
						{
							Scope: &v1_common.InstrumentationScope{
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
								},
							},
						},
					},
				},
			},
			expectedKeys: []uint32{util.TokenFor(util.FakeTenantID, traceIDB)},
			expectedTraces: []*tempopb.Trace{
				{
					Batches: []*v1.ResourceSpans{
						{
							Resource: &v1_resource.Resource{
								DroppedAttributesCount: 3,
							},
							ScopeSpans: []*v1.ScopeSpans{
								{
									Scope: &v1_common.InstrumentationScope{
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
										},
									},
								},
							},
						},
					},
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
					ScopeSpans: []*v1.ScopeSpans{
						{
							Scope: &v1_common.InstrumentationScope{
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
								},
							},
						},
					},
				},
				{
					Resource: &v1_resource.Resource{
						DroppedAttributesCount: 4,
					},
					ScopeSpans: []*v1.ScopeSpans{
						{
							Scope: &v1_common.InstrumentationScope{
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
								},
							},
						},
					},
				},
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
							ScopeSpans: []*v1.ScopeSpans{
								{
									Scope: &v1_common.InstrumentationScope{
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
									},
								},
							},
						},
						{
							Resource: &v1_resource.Resource{
								DroppedAttributesCount: 4,
							},
							ScopeSpans: []*v1.ScopeSpans{
								{
									Scope: &v1_common.InstrumentationScope{
										Name: "test2",
									},
									Spans: []*v1.Span{
										{
											TraceId:           traceIDB,
											Name:              "spanB",
											StartTimeUnixNano: uint64(10 * time.Second),
											EndTimeUnixNano:   uint64(30 * time.Second),
										},
									},
								},
							},
						},
					},
				},
				{
					Batches: []*v1.ResourceSpans{
						{
							Resource: &v1_resource.Resource{
								DroppedAttributesCount: 3,
							},
							ScopeSpans: []*v1.ScopeSpans{
								{
									Scope: &v1_common.InstrumentationScope{
										Name: "test",
									},
									Spans: []*v1.Span{
										{
											TraceId:           traceIDA,
											Name:              "spanE",
											StartTimeUnixNano: uint64(70 * time.Second),
											EndTimeUnixNano:   uint64(80 * time.Second),
										},
									},
								},
							},
						},
						{
							Resource: &v1_resource.Resource{
								DroppedAttributesCount: 4,
							},
							ScopeSpans: []*v1.ScopeSpans{
								{
									Scope: &v1_common.InstrumentationScope{
										Name: "test2",
									},
									Spans: []*v1.Span{
										{
											TraceId:           traceIDA,
											Name:              "spanD",
											StartTimeUnixNano: uint64(60 * time.Second),
											EndTimeUnixNano:   uint64(80 * time.Second),
										},
									},
								},
							},
						},
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
	ils := make([][]*v1.ScopeSpans, batches)

	for i := 0; i < batches; i++ {
		for _, t := range traces {
			ils[i] = append(ils[i], t.Batches[i].ScopeSpans...)
		}
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		for _, blerg := range ils {
			_, _, err := requestsByTraceID([]*v1.ResourceSpans{
				{
					ScopeSpans: blerg,
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
			limits := overrides.Config{}
			limits.RegisterFlagsAndApplyDefaults(&flag.FlagSet{})

			// todo:  test limits
			d := prepare(t, limits, nil)

			b := test.MakeBatch(tc.lines, []byte{})
			traces := batchesToTraces(t, []*v1.ResourceSpans{b})
			response, err := d.PushTraces(ctx, traces)

			assert.True(t, proto.Equal(tc.expectedResponse, response))
			assert.Equal(t, tc.expectedError, err)
		})
	}
}

func TestLogSpans(t *testing.T) {
	for i, tc := range []struct {
		LogReceivedSpansEnabled bool
		filterByStatusError     bool
		includeAllAttributes    bool
		batches                 []*v1.ResourceSpans
		expectedLogsSpan        []testLogSpan
	}{
		{
			LogReceivedSpansEnabled: false,
			batches: []*v1.ResourceSpans{
				makeResourceSpans("test", []*v1.ScopeSpans{
					makeScope(
						makeSpan("0a0102030405060708090a0b0c0d0e0f", "dad44adc9a83b370", "Test Span", nil)),
				}),
			},
			expectedLogsSpan: []testLogSpan{},
		},
		{
			LogReceivedSpansEnabled: true,
			filterByStatusError:     false,
			batches: []*v1.ResourceSpans{
				makeResourceSpans("test-service", []*v1.ScopeSpans{
					makeScope(
						makeSpan("0a0102030405060708090a0b0c0d0e0f", "dad44adc9a83b370", "Test Span1", nil),
						makeSpan("e3210a2b38097332d1fe43083ea93d29", "6c21c48da4dbd1a7", "Test Span2", nil)),
					makeScope(
						makeSpan("bb42ec04df789ff04b10ea5274491685", "1b3a296034f4031e", "Test Span3", nil)),
				}),
				makeResourceSpans("test-service2", []*v1.ScopeSpans{
					makeScope(
						makeSpan("b1c792dea27d511c145df8402bdd793a", "56afb9fe18b6c2d6", "Test Span", nil)),
				}),
			},
			expectedLogsSpan: []testLogSpan{
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
				makeResourceSpans("test-service", []*v1.ScopeSpans{
					makeScope(
						makeSpan("0a0102030405060708090a0b0c0d0e0f", "dad44adc9a83b370", "Test Span1", nil),
						makeSpan("e3210a2b38097332d1fe43083ea93d29", "6c21c48da4dbd1a7", "Test Span2", &v1.Status{Code: v1.Status_STATUS_CODE_ERROR})),
					makeScope(
						makeSpan("bb42ec04df789ff04b10ea5274491685", "1b3a296034f4031e", "Test Span3", nil)),
				}),
				makeResourceSpans("test-service2", []*v1.ScopeSpans{
					makeScope(
						makeSpan("b1c792dea27d511c145df8402bdd793a", "56afb9fe18b6c2d6", "Test Span", &v1.Status{Code: v1.Status_STATUS_CODE_ERROR})),
				}),
			},
			expectedLogsSpan: []testLogSpan{
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
			includeAllAttributes:    true,
			batches: []*v1.ResourceSpans{
				makeResourceSpans("test-service", []*v1.ScopeSpans{
					makeScope(
						makeSpan("0a0102030405060708090a0b0c0d0e0f", "dad44adc9a83b370", "Test Span1", nil,
							makeAttribute("tag1", "value1")),
						makeSpan("e3210a2b38097332d1fe43083ea93d29", "6c21c48da4dbd1a7", "Test Span2", &v1.Status{Code: v1.Status_STATUS_CODE_ERROR},
							makeAttribute("tag1", "value1"),
							makeAttribute("tag2", "value2"))),
					makeScope(
						makeSpan("bb42ec04df789ff04b10ea5274491685", "1b3a296034f4031e", "Test Span3", nil)),
				}, makeAttribute("resource_attribute1", "value1")),
				makeResourceSpans("test-service2", []*v1.ScopeSpans{
					makeScope(
						makeSpan("b1c792dea27d511c145df8402bdd793a", "56afb9fe18b6c2d6", "Test Span", &v1.Status{Code: v1.Status_STATUS_CODE_ERROR})),
				}, makeAttribute("resource_attribute2", "value2")),
			},
			expectedLogsSpan: []testLogSpan{
				{
					Name:               "Test Span2",
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
					Name:               "Test Span",
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
			includeAllAttributes:    true,
			batches: []*v1.ResourceSpans{
				makeResourceSpans("test-service", []*v1.ScopeSpans{
					makeScope(
						makeSpan("0a0102030405060708090a0b0c0d0e0f", "dad44adc9a83b370", "Test Span", nil, makeAttribute("tag1", "value1"))),
				}),
			},
			expectedLogsSpan: []testLogSpan{
				{
					Name:            "Test Span",
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
		t.Run(fmt.Sprintf("[%d] TestLogSpans LogReceivedSpansEnabled=%v filterByStatusError=%v includeAllAttributes=%v", i, tc.LogReceivedSpansEnabled, tc.filterByStatusError, tc.includeAllAttributes), func(t *testing.T) {
			limits := overrides.Config{}
			limits.RegisterFlagsAndApplyDefaults(&flag.FlagSet{})

			buf := &bytes.Buffer{}
			logger := kitlog.NewJSONLogger(kitlog.NewSyncWriter(buf))

			d := prepare(t, limits, logger)
			d.cfg.LogReceivedSpans = LogReceivedSpansConfig{
				Enabled:              tc.LogReceivedSpansEnabled,
				FilterByStatusError:  tc.filterByStatusError,
				IncludeAllAttributes: tc.includeAllAttributes,
			}

			traces := batchesToTraces(t, tc.batches)
			_, err := d.PushTraces(ctx, traces)
			if err != nil {
				t.Fatal(err)
			}

			bufJSON := "[" + strings.TrimRight(strings.ReplaceAll(buf.String(), "\n", ","), ",") + "]"
			var actualLogsSpan []testLogSpan
			err = json.Unmarshal([]byte(bufJSON), &actualLogsSpan)
			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, len(tc.expectedLogsSpan), len(actualLogsSpan))
			for i, expectedLogSpan := range tc.expectedLogsSpan {
				assert.EqualValues(t, expectedLogSpan, actualLogsSpan[i])
			}
		})
	}
}

func TestRateLimitRespected(t *testing.T) {
	// prepare test data
	overridesConfig := overrides.Config{
		Defaults: overrides.Overrides{
			Ingestion: overrides.IngestionOverrides{
				RateStrategy:   overrides.LocalIngestionRateStrategy,
				RateLimitBytes: 400,
				BurstSizeBytes: 200,
			},
		},
	}
	buf := &bytes.Buffer{}
	logger := kitlog.NewJSONLogger(kitlog.NewSyncWriter(buf))
	d := prepare(t, overridesConfig, logger)
	batches := []*v1.ResourceSpans{
		makeResourceSpans("test-service", []*v1.ScopeSpans{
			makeScope(
				makeSpan("0a0102030405060708090a0b0c0d0e0f", "dad44adc9a83b370", "Test Span1", nil,
					makeAttribute("tag1", "value1")),
				makeSpan("e3210a2b38097332d1fe43083ea93d29", "6c21c48da4dbd1a7", "Test Span2", &v1.Status{Code: v1.Status_STATUS_CODE_ERROR},
					makeAttribute("tag1", "value1"),
					makeAttribute("tag2", "value2"))),
			makeScope(
				makeSpan("bb42ec04df789ff04b10ea5274491685", "1b3a296034f4031e", "Test Span3", nil)),
		}, makeAttribute("resource_attribute1", "value1")),
		makeResourceSpans("test-service2", []*v1.ScopeSpans{
			makeScope(
				makeSpan("b1c792dea27d511c145df8402bdd793a", "56afb9fe18b6c2d6", "Test Span", &v1.Status{Code: v1.Status_STATUS_CODE_ERROR})),
		}, makeAttribute("resource_attribute2", "value2")),
	}
	traces := batchesToTraces(t, batches)

	// invoke unit
	_, err := d.PushTraces(ctx, traces)

	// validations
	if err == nil {
		t.Fatal("Expected error")
	}
	status, ok := status.FromError(err)
	assert.True(t, ok)
	assert.True(t, status.Code() == codes.ResourceExhausted, "Wrong status code")
}

func TestDiscardCountReplicationFactor(t *testing.T) {
	tt := []struct {
		name                                string
		pushErrorByTrace                    [][]tempopb.PushErrorReason
		replicationFactor                   int
		expectedLiveTracesDiscardedCount    int
		expectedTraceTooLargeDiscardedCount int
	}{
		// trace sizes
		// trace[0] = 5 spans
		// trace[1] = 10 spans
		// trace[2] = 15 spans
		{
			name:                                "no errors, minimum responses",
			pushErrorByTrace:                    [][]tempopb.PushErrorReason{{noError, noError, noError}, {noError, noError, noError}},
			replicationFactor:                   3,
			expectedLiveTracesDiscardedCount:    0,
			expectedTraceTooLargeDiscardedCount: 0,
		},
		{
			name:                                "no error, max responses",
			pushErrorByTrace:                    [][]tempopb.PushErrorReason{{noError, noError, noError}, {noError, noError, noError}, {noError, noError, noError}},
			replicationFactor:                   3,
			expectedLiveTracesDiscardedCount:    0,
			expectedTraceTooLargeDiscardedCount: 0,
		},
		{
			name:                                "one mlt error, minimum responses",
			pushErrorByTrace:                    [][]tempopb.PushErrorReason{{maxLiveTraceError, noError, noError}, {noError, noError, noError}},
			replicationFactor:                   3,
			expectedLiveTracesDiscardedCount:    5,
			expectedTraceTooLargeDiscardedCount: 0,
		},
		{
			name:                                "one mlt error, max responses",
			pushErrorByTrace:                    [][]tempopb.PushErrorReason{{maxLiveTraceError, noError, noError}, {noError, noError, noError}, {noError, noError, noError}},
			replicationFactor:                   3,
			expectedLiveTracesDiscardedCount:    0,
			expectedTraceTooLargeDiscardedCount: 0,
		},
		{
			name:                                "one ttl error, minimum responses",
			pushErrorByTrace:                    [][]tempopb.PushErrorReason{{noError, traceTooLargeError, noError}, {noError, noError, noError}},
			replicationFactor:                   3,
			expectedLiveTracesDiscardedCount:    0,
			expectedTraceTooLargeDiscardedCount: 10,
		},
		{
			name:                                "one ttl error, max responses",
			pushErrorByTrace:                    [][]tempopb.PushErrorReason{{noError, traceTooLargeError, noError}, {noError, noError, noError}, {noError, noError, noError}},
			replicationFactor:                   3,
			expectedLiveTracesDiscardedCount:    0,
			expectedTraceTooLargeDiscardedCount: 0,
		},
		{
			name:                                "two mlt errors, minimum responses",
			pushErrorByTrace:                    [][]tempopb.PushErrorReason{{maxLiveTraceError, noError, noError}, {maxLiveTraceError, noError, noError}},
			replicationFactor:                   3,
			expectedLiveTracesDiscardedCount:    5,
			expectedTraceTooLargeDiscardedCount: 0,
		},
		{
			name:                                "two ttl errors, max responses",
			pushErrorByTrace:                    [][]tempopb.PushErrorReason{{noError, traceTooLargeError, noError}, {noError, traceTooLargeError, noError}, {noError, noError, noError}},
			replicationFactor:                   3,
			expectedLiveTracesDiscardedCount:    0,
			expectedTraceTooLargeDiscardedCount: 10,
		},
		{
			name:                                "three ttl errors, max responses",
			pushErrorByTrace:                    [][]tempopb.PushErrorReason{{noError, traceTooLargeError, noError}, {noError, traceTooLargeError, noError}, {noError, traceTooLargeError, noError}},
			replicationFactor:                   3,
			expectedLiveTracesDiscardedCount:    0,
			expectedTraceTooLargeDiscardedCount: 10,
		},
		{
			name:                                "three mix errors, max responses",
			pushErrorByTrace:                    [][]tempopb.PushErrorReason{{noError, traceTooLargeError, noError}, {noError, maxLiveTraceError, noError}, {noError, traceTooLargeError, noError}},
			replicationFactor:                   3,
			expectedLiveTracesDiscardedCount:    0,
			expectedTraceTooLargeDiscardedCount: 10,
		},
		{
			name:                                "three mix trace errors, max responses",
			pushErrorByTrace:                    [][]tempopb.PushErrorReason{{noError, traceTooLargeError, noError}, {noError, noError, traceTooLargeError}, {noError, maxLiveTraceError, traceTooLargeError}},
			replicationFactor:                   3,
			expectedLiveTracesDiscardedCount:    10,
			expectedTraceTooLargeDiscardedCount: 15,
		},
		{
			name:                                "one ttl error rep factor 5 min (3) response",
			pushErrorByTrace:                    [][]tempopb.PushErrorReason{{noError, traceTooLargeError, noError}, {noError, noError, noError}, {noError, noError, noError}},
			replicationFactor:                   5,
			expectedLiveTracesDiscardedCount:    0,
			expectedTraceTooLargeDiscardedCount: 10,
		},
		{
			name:                                "one error rep factor 5 with 4 responses",
			pushErrorByTrace:                    [][]tempopb.PushErrorReason{{noError, traceTooLargeError, noError}, {noError, noError, noError}, {noError, noError, noError}, {noError, noError, noError}},
			replicationFactor:                   5,
			expectedLiveTracesDiscardedCount:    0,
			expectedTraceTooLargeDiscardedCount: 0,
		},
		{
			name:                                "replication factor 1",
			pushErrorByTrace:                    [][]tempopb.PushErrorReason{{noError, traceTooLargeError, noError}},
			replicationFactor:                   1,
			expectedLiveTracesDiscardedCount:    0,
			expectedTraceTooLargeDiscardedCount: 10,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			traceByID := make([]*rebatchedTrace, 3)
			// batch with 3 traces
			traceByID[0] = &rebatchedTrace{
				spanCount: 5,
			}

			traceByID[1] = &rebatchedTrace{
				spanCount: 15,
			}

			traceByID[2] = &rebatchedTrace{
				spanCount: 10,
			}

			keys := []int{0, 2, 1}

			numSuccessByTraceIndex := make([]int, len(traceByID))
			lastErrorReasonByTraceIndex := make([]tempopb.PushErrorReason, len(traceByID))

			for _, ErrorByTrace := range tc.pushErrorByTrace {
				for ringIndex, err := range ErrorByTrace {
					// translate
					traceIndex := keys[ringIndex]

					currentNumSuccess := numSuccessByTraceIndex[traceIndex]
					if err == tempopb.PushErrorReason_NO_ERROR {
						numSuccessByTraceIndex[traceIndex] = currentNumSuccess + 1
					} else {
						lastErrorReasonByTraceIndex[traceIndex] = err
					}
				}
			}

			liveTraceDiscardedCount, traceTooLongDiscardedCount, _ := countDiscaredSpans(numSuccessByTraceIndex, lastErrorReasonByTraceIndex, traceByID, tc.replicationFactor)

			require.Equal(t, tc.expectedLiveTracesDiscardedCount, liveTraceDiscardedCount)
			require.Equal(t, tc.expectedTraceTooLargeDiscardedCount, traceTooLongDiscardedCount)
		})
	}
}

func TestProcessIngesterPushByteResponse(t *testing.T) {
	// batch has 5 traces [0, 1, 2, 3, 4, 5]
	numOfTraces := 5
	tt := []struct {
		name                   string
		pushErrorByTrace       []tempopb.PushErrorReason
		indexes                []int
		expectedSuccessIndex   []int
		expectedLastErrorIndex []tempopb.PushErrorReason
	}{
		{
			name:                   "explicit no errors, first three traces",
			pushErrorByTrace:       []tempopb.PushErrorReason{noError, noError, noError},
			indexes:                []int{0, 1, 2},
			expectedSuccessIndex:   []int{1, 1, 1, 0, 0},
			expectedLastErrorIndex: make([]tempopb.PushErrorReason, numOfTraces),
		},
		{
			name:                   "no errors, no ErrorsByTrace value",
			pushErrorByTrace:       []tempopb.PushErrorReason{},
			indexes:                []int{1, 2, 3},
			expectedSuccessIndex:   []int{0, 1, 1, 1, 0},
			expectedLastErrorIndex: make([]tempopb.PushErrorReason, numOfTraces),
		},
		{
			name:                   "all errors, first three traces",
			pushErrorByTrace:       []tempopb.PushErrorReason{traceTooLargeError, traceTooLargeError, traceTooLargeError},
			indexes:                []int{0, 1, 2},
			expectedSuccessIndex:   []int{0, 0, 0, 0, 0},
			expectedLastErrorIndex: []tempopb.PushErrorReason{traceTooLargeError, traceTooLargeError, traceTooLargeError, noError, noError},
		},
		{
			name:                   "random errors, random three traces",
			pushErrorByTrace:       []tempopb.PushErrorReason{traceTooLargeError, maxLiveTraceError, noError},
			indexes:                []int{0, 2, 4},
			expectedSuccessIndex:   []int{0, 0, 0, 0, 1},
			expectedLastErrorIndex: []tempopb.PushErrorReason{traceTooLargeError, noError, maxLiveTraceError, noError, noError},
		},
	}

	// prepare test data
	overridesConfig := overrides.Config{}
	buf := &bytes.Buffer{}
	logger := kitlog.NewJSONLogger(kitlog.NewSyncWriter(buf))
	d := prepare(t, overridesConfig, logger)

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			numSuccessByTraceIndex := make([]int, numOfTraces)
			lastErrorReasonByTraceIndex := make([]tempopb.PushErrorReason, numOfTraces)
			pushByteResponse := &tempopb.PushResponse{
				ErrorsByTrace: tc.pushErrorByTrace,
			}
			d.processPushResponse(pushByteResponse, numSuccessByTraceIndex, lastErrorReasonByTraceIndex, numOfTraces, tc.indexes)
			assert.Equal(t, numSuccessByTraceIndex, tc.expectedSuccessIndex)
			assert.Equal(t, lastErrorReasonByTraceIndex, tc.expectedLastErrorIndex)
		})
	}
}

func TestIngesterPushBytes(t *testing.T) {
	// prepare test data
	overridesConfig := overrides.Config{}
	buf := &bytes.Buffer{}
	logger := kitlog.NewJSONLogger(kitlog.NewSyncWriter(buf))
	d := prepare(t, overridesConfig, logger)

	traces := []*rebatchedTrace{
		{
			spanCount: 1,
		},
		{
			spanCount: 5,
		},
		{
			spanCount: 10,
		},
		{
			spanCount: 15,
		},
		{
			spanCount: 20,
		},
	}
	numOfTraces := len(traces)
	numSuccessByTraceIndex := make([]int, numOfTraces)
	lastErrorReasonByTraceIndex := make([]tempopb.PushErrorReason, numOfTraces)

	// 0 = trace_too_large, trace_too_large || discard count: 1
	// 1 = no error, trace_too_large || discard count: 5
	// 2 = no error, no error || discard count: 0
	// 3 = max_live, max_live || discard count: 15
	// 4 = trace_too_large, max_live || discard count: 20
	// total ttl: 6, mlt: 35

	batches := [][]int{
		{0, 1, 2},
		{1, 3},
		{0, 2},
		{3, 4},
		{4},
	}

	errorsByTraces := [][]tempopb.PushErrorReason{
		{traceTooLargeError, noError, noError},
		{traceTooLargeError, maxLiveTraceError},
		{traceTooLargeError, noError},
		{maxLiveTraceError, traceTooLargeError},
		{maxLiveTraceError},
	}

	for i, indexes := range batches {
		pushResponse := &tempopb.PushResponse{
			ErrorsByTrace: errorsByTraces[i],
		}
		d.processPushResponse(pushResponse, numSuccessByTraceIndex, lastErrorReasonByTraceIndex, numOfTraces, indexes)
	}

	maxLiveDiscardedCount, traceTooLargeDiscardedCount, _ := countDiscaredSpans(numSuccessByTraceIndex, lastErrorReasonByTraceIndex, traces, 3)
	assert.Equal(t, traceTooLargeDiscardedCount, 6)
	assert.Equal(t, maxLiveDiscardedCount, 35)
}

type testLogSpan struct {
	Msg                string `json:"msg"`
	Level              string `json:"level"`
	TraceID            string `json:"traceid"`
	SpanID             string `json:"spanid"`
	Name               string `json:"span_name"`
	SpanStatus         string `json:"span_status,omitempty"`
	SpanKind           string `json:"span_kind,omitempty"`
	SpanServiceName    string `json:"span_service_name,omitempty"`
	SpanTag1           string `json:"span_tag1,omitempty"`
	SpanTag2           string `json:"span_tag2,omitempty"`
	ResourceAttribute1 string `json:"span_resource_attribute1,omitempty"`
	ResourceAttribute2 string `json:"span_resource_attribute2,omitempty"`
}

func makeAttribute(key, value string) *v1_common.KeyValue {
	return &v1_common.KeyValue{
		Key:   key,
		Value: &v1_common.AnyValue{Value: &v1_common.AnyValue_StringValue{StringValue: value}},
	}
}

func makeSpan(traceID, spanID, name string, status *v1.Status, attributes ...*v1_common.KeyValue) *v1.Span {
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
		Name:       name,
		TraceId:    traceIDBytes,
		SpanId:     spanIDBytes,
		Status:     status,
		Kind:       v1.Span_SPAN_KIND_SERVER,
		Attributes: attributes,
	}
}

func makeScope(spans ...*v1.Span) *v1.ScopeSpans {
	return &v1.ScopeSpans{
		Scope: &v1_common.InstrumentationScope{
			Name:    "super library",
			Version: "0.0.1",
		},
		Spans: spans,
	}
}

func makeResourceSpans(serviceName string, ils []*v1.ScopeSpans, attributes ...*v1_common.KeyValue) *v1.ResourceSpans {
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
		ScopeSpans: ils,
	}

	rs.Resource.Attributes = append(rs.Resource.Attributes, attributes...)

	return rs
}

func prepare(t *testing.T, limits overrides.Config, logger kitlog.Logger) *Distributor {
	if logger == nil {
		logger = kitlog.NewNopLogger()
	}

	var (
		distributorConfig Config
		clientConfig      ingester_client.Config
	)
	flagext.DefaultValues(&clientConfig)

	overrides, err := overrides.NewOverrides(limits, prometheus.DefaultRegisterer)
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
	distributorConfig.DistributorRing.KVStore.Mock = nil
	distributorConfig.DistributorRing.InstanceInterfaceNames = []string{"eth0", "en0", "lo0"}
	distributorConfig.factory = func(addr string) (ring_client.PoolClient, error) {
		return ingesters[addr], nil
	}

	l := dslog.Level{}
	_ = l.Set("error")
	mw := receiver.MultiTenancyMiddleware()
	d, err := New(distributorConfig, clientConfig, ingestersRing, generator_client.Config{}, nil, overrides, mw, logger, l, prometheus.NewPedanticRegistry())
	require.NoError(t, err)

	return d
}

type mockIngester struct {
	grpc_health_v1.HealthClient
}

var _ tempopb.PusherClient = (*mockIngester)(nil)

func (i *mockIngester) PushBytes(context.Context, *tempopb.PushBytesRequest, ...grpc.CallOption) (*tempopb.PushResponse, error) {
	return &tempopb.PushResponse{}, nil
}

func (i *mockIngester) PushBytesV2(context.Context, *tempopb.PushBytesRequest, ...grpc.CallOption) (*tempopb.PushResponse, error) {
	return &tempopb.PushResponse{}, nil
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

func (r mockRing) Get(key uint32, _ ring.Operation, buf []ring.InstanceDesc, _, _ []string) (ring.ReplicationSet, error) {
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

func (r mockRing) GetAllHealthy(ring.Operation) (ring.ReplicationSet, error) {
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

func (r mockRing) ShuffleShard(string, int) ring.ReadRing {
	return r
}

func (r mockRing) ShuffleShardWithLookback(string, int, time.Duration, time.Time) ring.ReadRing {
	return r
}

func (r mockRing) GetTokenRangesForInstance(_ string) (ring.TokenRanges, error) {
	return nil, nil
}

func (r mockRing) InstancesCount() int {
	return len(r.ingesters)
}

func (r mockRing) HasInstance(string) bool {
	return true
}

func (r mockRing) CleanupShuffleShardCache(string) {
}

func (r mockRing) GetInstanceState(string) (ring.InstanceState, error) {
	return ring.ACTIVE, nil
}
