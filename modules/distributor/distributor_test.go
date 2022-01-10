package distributor

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"testing"
	"time"

	"github.com/gogo/status"
	"github.com/grafana/dskit/flagext"
	"github.com/grafana/dskit/kv"
	"github.com/grafana/dskit/ring"
	ring_client "github.com/grafana/dskit/ring/client"
	v1_common "github.com/grafana/tempo/pkg/tempopb/common/v1"
	v1_resource "github.com/grafana/tempo/pkg/tempopb/resource/v1"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"

	"github.com/golang/protobuf/proto"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/weaveworks/common/logging"
	"github.com/weaveworks/common/user"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health/grpc_health_v1"

	"github.com/grafana/tempo/modules/distributor/receiver"
	ingester_client "github.com/grafana/tempo/modules/ingester/client"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/tempopb"
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
									TraceId: traceIDA,
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
											TraceId: traceIDA,
										}}}}}},
				},
			},
			expectedIDs: [][]byte{
				traceIDA,
			},
		},
		{
			name: "two traces, one batch",
			batches: []*v1.ResourceSpans{
				{
					InstrumentationLibrarySpans: []*v1.InstrumentationLibrarySpans{
						{
							Spans: []*v1.Span{
								{
									TraceId: traceIDA,
								},
								{
									TraceId: traceIDB,
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
											TraceId: traceIDA,
										}}}}}},
				},
				{
					Batches: []*v1.ResourceSpans{
						{
							InstrumentationLibrarySpans: []*v1.InstrumentationLibrarySpans{
								{
									Spans: []*v1.Span{
										{
											TraceId: traceIDB,
										}}}}}},
				},
			},
			expectedIDs: [][]byte{
				traceIDA,
				traceIDB,
			},
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
									TraceId: traceIDA,
								}}}}},
				{
					Resource: &v1_resource.Resource{
						DroppedAttributesCount: 4,
					},
					InstrumentationLibrarySpans: []*v1.InstrumentationLibrarySpans{
						{
							Spans: []*v1.Span{
								{
									TraceId: traceIDB,
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
											TraceId: traceIDA,
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
											TraceId: traceIDB,
										}}}}}},
				},
			},
			expectedIDs: [][]byte{
				traceIDA,
				traceIDB,
			},
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
									TraceId: traceIDA,
								},
								{
									TraceId: traceIDB,
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
											TraceId: traceIDA,
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
											TraceId: traceIDB,
										}}}}}},
				},
			},
			expectedIDs: [][]byte{
				traceIDA,
				traceIDB,
			},
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
									TraceId: traceIDA,
								},
								{
									TraceId: traceIDB,
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
											TraceId: traceIDA,
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
											TraceId: traceIDB,
										}}}}}},
				},
			},
			expectedIDs: [][]byte{
				traceIDA,
				traceIDB,
			},
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
									TraceId: traceIDB,
									Name:    "spanA",
								},
								{
									TraceId: traceIDB,
									Name:    "spanB",
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
											TraceId: traceIDB,
											Name:    "spanA",
										},
										{
											TraceId: traceIDB,
											Name:    "spanB",
										}}}}}},
				},
			},
			expectedIDs: [][]byte{
				traceIDB,
			},
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
									TraceId: traceIDB,
									Name:    "spanA",
								},
								{
									TraceId: traceIDB,
									Name:    "spanC",
								},
								{
									TraceId: traceIDA,
									Name:    "spanE",
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
									TraceId: traceIDB,
									Name:    "spanB",
								},
								{
									TraceId: traceIDA,
									Name:    "spanD",
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
											TraceId: traceIDB,
											Name:    "spanA",
										},
										{
											TraceId: traceIDB,
											Name:    "spanC",
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
											TraceId: traceIDB,
											Name:    "spanB",
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
											TraceId: traceIDA,
											Name:    "spanE",
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
											TraceId: traceIDA,
											Name:    "spanD",
										}}}}},
					},
				},
			},
			expectedIDs: [][]byte{
				traceIDB,
				traceIDA,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keys, reqs, ids, err := requestsByTraceID(tt.batches, util.FakeTenantID, 1)
			require.Equal(t, len(keys), len(reqs))

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
				actualReq := reqs[foundIndex]
				assert.Equal(t, expectedReq, actualReq)
				assert.Equal(t, tt.expectedIDs[i], ids[foundIndex])
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
			_, _, _, err := requestsByTraceID([]*v1.ResourceSpans{
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
			d := prepare(t, limits, nil)

			b := test.MakeBatch(tc.lines, []byte{})
			response, err := d.PushBatches(ctx, []*v1.ResourceSpans{b})

			assert.True(t, proto.Equal(tc.expectedResponse, response))
			assert.Equal(t, tc.expectedError, err)
		})
	}
}

func prepare(t *testing.T, limits *overrides.Limits, kvStore kv.Client) *Distributor {
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
	d, err := New(distributorConfig, clientConfig, ingestersRing, overrides, mw, l, false, prometheus.NewPedanticRegistry())
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
