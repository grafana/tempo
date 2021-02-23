package distributor

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"testing"
	"time"

	"github.com/cortexproject/cortex/pkg/ring"
	ring_client "github.com/cortexproject/cortex/pkg/ring/client"
	"github.com/cortexproject/cortex/pkg/ring/kv"
	"github.com/cortexproject/cortex/pkg/util/flagext"
	"github.com/gogo/status"
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
		name         string
		request      *tempopb.PushRequest
		expectedKeys []uint32
		expectedReqs []*tempopb.PushRequest
		expectedErr  error
	}{
		{
			name: "empty",
			request: &tempopb.PushRequest{
				Batch: &v1.ResourceSpans{},
			},
			expectedKeys: []uint32{},
			expectedReqs: []*tempopb.PushRequest{},
		},
		{
			name: "bad trace id",
			request: &tempopb.PushRequest{
				Batch: &v1.ResourceSpans{
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
			request: &tempopb.PushRequest{
				Batch: &v1.ResourceSpans{
					InstrumentationLibrarySpans: []*v1.InstrumentationLibrarySpans{
						{
							Spans: []*v1.Span{
								{
									TraceId: traceIDA,
								}}}}},
			},
			expectedKeys: []uint32{util.TokenFor(util.FakeTenantID, traceIDA)},
			expectedReqs: []*tempopb.PushRequest{
				{
					Batch: &v1.ResourceSpans{
						InstrumentationLibrarySpans: []*v1.InstrumentationLibrarySpans{
							{
								Spans: []*v1.Span{
									{
										TraceId: traceIDA,
									}}}}}},
			},
		},
		{
			name: "two traces",
			request: &tempopb.PushRequest{
				Batch: &v1.ResourceSpans{
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
			expectedReqs: []*tempopb.PushRequest{
				{
					Batch: &v1.ResourceSpans{
						InstrumentationLibrarySpans: []*v1.InstrumentationLibrarySpans{
							{
								Spans: []*v1.Span{
									{
										TraceId: traceIDA,
									}}}}},
				},
				{
					Batch: &v1.ResourceSpans{
						InstrumentationLibrarySpans: []*v1.InstrumentationLibrarySpans{
							{
								Spans: []*v1.Span{
									{
										TraceId: traceIDB,
									}}}}}},
			},
		},
		{
			name: "resource copied",
			request: &tempopb.PushRequest{
				Batch: &v1.ResourceSpans{
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
			expectedReqs: []*tempopb.PushRequest{
				{
					Batch: &v1.ResourceSpans{
						Resource: &v1_resource.Resource{
							DroppedAttributesCount: 1,
						},
						InstrumentationLibrarySpans: []*v1.InstrumentationLibrarySpans{
							{
								Spans: []*v1.Span{
									{
										TraceId: traceIDA,
									}}}}},
				},
				{
					Batch: &v1.ResourceSpans{
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
		{
			name: "ils copied",
			request: &tempopb.PushRequest{
				Batch: &v1.ResourceSpans{
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
			expectedReqs: []*tempopb.PushRequest{
				{
					Batch: &v1.ResourceSpans{
						InstrumentationLibrarySpans: []*v1.InstrumentationLibrarySpans{
							{
								InstrumentationLibrary: &v1_common.InstrumentationLibrary{
									Name: "test",
								},
								Spans: []*v1.Span{
									{
										TraceId: traceIDA,
									}}}}},
				},
				{
					Batch: &v1.ResourceSpans{
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
		{
			name: "one trace",
			request: &tempopb.PushRequest{
				Batch: &v1.ResourceSpans{
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
			expectedReqs: []*tempopb.PushRequest{
				{
					Batch: &v1.ResourceSpans{
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
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keys, reqs, err := requestsByTraceID(tt.request, util.FakeTenantID, 1)

			for i, expectedKey := range tt.expectedKeys {
				foundIndex := -1
				for j, key := range keys {
					if expectedKey == key {
						foundIndex = j
					}
				}

				require.NotEqual(t, -1, foundIndex, "expected key %d not found", foundIndex)

				// now confirm that the request at this position is the expected one
				expectedReq := tt.expectedReqs[i]
				actualReq := reqs[foundIndex]
				assert.Equal(t, expectedReq, actualReq)
			}

			assert.Equal(t, tt.expectedErr, err)
		})
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

			request := test.MakeRequest(tc.lines, []byte{})
			response, err := d.Push(ctx, request)

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
	d, err := New(distributorConfig, clientConfig, ingestersRing, overrides, true, l)
	require.NoError(t, err)

	return d
}

type mockIngester struct {
	grpc_health_v1.HealthClient
}

var _ tempopb.PusherClient = (*mockIngester)(nil)

func (i *mockIngester) Push(ctx context.Context, in *tempopb.PushRequest, opts ...grpc.CallOption) (*tempopb.PushResponse, error) {
	return nil, nil
}

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
		Ingesters: buf[:0],
	}
	for i := uint32(0); i < r.replicationFactor; i++ {
		n := (key + i) % uint32(len(r.ingesters))
		result.Ingesters = append(result.Ingesters, r.ingesters[n])
	}
	return result, nil
}

func (r mockRing) GetAllHealthy(op ring.Operation) (ring.ReplicationSet, error) {
	return ring.ReplicationSet{
		Ingesters: r.ingesters,
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
