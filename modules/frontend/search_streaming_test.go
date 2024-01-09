package frontend

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/golang/protobuf/jsonpb" //nolint:all //deprecated
	"github.com/google/uuid"
	"github.com/grafana/dskit/user"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/stretchr/testify/require"
	"go.uber.org/atomic"
	"google.golang.org/grpc/metadata"
)

type mockStreamingServer struct {
	lastResponse atomic.Pointer[*tempopb.SearchResponse]
	responses    atomic.Int32
	ctx          context.Context
	cb           func(int, *tempopb.SearchResponse)
}

func (m *mockStreamingServer) Send(r *tempopb.SearchResponse) error {
	if r != nil && len(r.Traces) > 0 {
		m.lastResponse.Store(&r)
	}
	m.responses.Inc()
	if m.cb != nil {
		m.cb(int(m.responses.Load()), r)
	}
	return nil
}
func (m *mockStreamingServer) Context() context.Context     { return m.ctx }
func (m *mockStreamingServer) SendHeader(metadata.MD) error { return nil }
func (m *mockStreamingServer) SetHeader(metadata.MD) error  { return nil }
func (m *mockStreamingServer) SendMsg(interface{}) error    { return nil }
func (m *mockStreamingServer) RecvMsg(interface{}) error    { return nil }
func (m *mockStreamingServer) SetTrailer(metadata.MD)       {}

func newMockStreamingServer(cb func(int, *tempopb.SearchResponse)) *mockStreamingServer {
	return &mockStreamingServer{
		ctx: user.InjectOrgID(context.Background(), "fake-tenant"),
		cb:  cb,
	}
}

func TestStreamingSearchHandlerSucceeds(t *testing.T) {
	traceResp := []*tempopb.TraceSearchMetadata{
		{
			TraceID:         "1234",
			RootServiceName: "root",
		},
	}

	next := RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
		response := &tempopb.SearchResponse{
			Traces:  traceResp,
			Metrics: &tempopb.SearchMetrics{},
		}
		resString, err := (&jsonpb.Marshaler{}).MarshalToString(response)
		require.NoError(t, err)

		return &http.Response{
			Body:       io.NopCloser(strings.NewReader(resString)),
			StatusCode: 200,
		}, nil
	})

	srv := newMockStreamingServer(nil)
	handler := testHandler(t, next)
	err := handler(&tempopb.SearchRequest{
		Start: 1000,
		End:   1500,
		Query: "{}",
	}, srv)
	require.NoError(t, err)
	// confirm final result is expected
	require.Equal(t,
		*srv.lastResponse.Load(),
		&tempopb.SearchResponse{
			Traces: traceResp,
			Metrics: &tempopb.SearchMetrics{
				TotalBlocks:     1,
				CompletedJobs:   2,
				TotalJobs:       2,
				TotalBlockBytes: 209715200,
			},
		},
	)
}

func TestStreamingSearchHandlerStreams(t *testing.T) {
	traceResp := []*tempopb.TraceSearchMetadata{
		{
			TraceID:         "1234",
			RootServiceName: "root",
		},
	}

	next := RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
		time.Sleep(1 * time.Second) // forces the streaming responses to work

		response := &tempopb.SearchResponse{
			Traces:  traceResp,
			Metrics: &tempopb.SearchMetrics{},
		}
		resString, err := (&jsonpb.Marshaler{}).MarshalToString(response)
		require.NoError(t, err)

		return &http.Response{
			Body:       io.NopCloser(strings.NewReader(resString)),
			StatusCode: 200,
		}, nil
	})

	srv := newMockStreamingServer(
		func(n int, r *tempopb.SearchResponse) {
			if len(r.Traces) > 0 {
				// if we have some traces confirm it's what is expected
				require.Equal(t, r,
					&tempopb.SearchResponse{
						Traces: traceResp,
						Metrics: &tempopb.SearchMetrics{
							TotalBlocks:     1,
							CompletedJobs:   r.Metrics.CompletedJobs,
							TotalJobs:       2,
							TotalBlockBytes: 209715200,
						},
					},
				)
			}
		},
	)
	handler := testHandler(t, next)
	err := handler(&tempopb.SearchRequest{
		Start: 1000,
		End:   1500,
		Query: "{}",
	}, srv)
	require.NoError(t, err)
	require.GreaterOrEqual(t, srv.responses.Load(), int32(3)) // confirm that our server got 3 or more sends
}

func TestStreamingSearchHandlerCancels(t *testing.T) {
	next := RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
		time.Sleep(24 * time.Hour) // will break any test time limits
		return nil, nil
	})

	var cancel context.CancelFunc
	srv := newMockStreamingServer(nil)
	srv.ctx, cancel = context.WithCancel(srv.ctx)

	go func() {
		time.Sleep(500 * time.Millisecond)
		cancel()
	}()

	handler := testHandler(t, next)
	err := handler(&tempopb.SearchRequest{
		Start: 1000,
		End:   1500,
		Query: "{}",
	}, srv)
	require.Equal(t, context.Canceled, err)
}

func TestStreamingSearchHandlerFailsDueToStatusCode(t *testing.T) {
	next := RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			Body:       io.NopCloser(strings.NewReader("error")),
			StatusCode: 500,
		}, nil
	})

	srv := newMockStreamingServer(nil)
	handler := testHandler(t, next)
	err := handler(&tempopb.SearchRequest{
		Start: 1000,
		End:   1500,
		Query: "{}",
	}, srv)
	require.Error(t, err)
}

func TestStreamingSearchHandlerFailsDueToError(t *testing.T) {
	next := RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
		return nil, errors.New("error")
	})

	srv := newMockStreamingServer(nil)
	handler := testHandler(t, next)
	err := handler(&tempopb.SearchRequest{
		Start: 1000,
		End:   1500,
		Query: "{}",
	}, srv)
	require.Error(t, err)
}

func testHandler(t *testing.T, next http.RoundTripper) streamingSearchHandler {
	t.Helper()

	o, err := overrides.NewOverrides(overrides.Config{})
	require.NoError(t, err)

	handler := newSearchStreamingGRPCHandler(Config{
		Search: SearchConfig{
			Sharder: SearchSharderConfig{
				ConcurrentRequests:    1, // 1 concurrent request to force order
				TargetBytesPerRequest: defaultTargetBytesPerRequest,
			},
		},
	}, o, next, &mockReader{
		metas: []*backend.BlockMeta{ // one block with 2 records that are each the target bytes per request will force 2 sub queries
			{
				StartTime:    time.Unix(1100, 0),
				EndTime:      time.Unix(1200, 0),
				Size:         defaultTargetBytesPerRequest * 2,
				TotalRecords: 2,
				BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000000"),
			},
		},
	}, &frontendCache{}, "", log.NewNopLogger())

	return handler
}

func TestDiffSearchProgress(t *testing.T) {
	ctx := context.Background()
	diffProgress := newDiffSearchProgress(ctx, 0, 0, 0, 0)

	// first request should be empty
	require.Equal(t, &tempopb.SearchResponse{
		Traces:  []*tempopb.TraceSearchMetadata{},
		Metrics: &tempopb.SearchMetrics{},
	}, diffProgress.result().response)

	diffProgress.addResponse(&tempopb.SearchResponse{
		Traces: []*tempopb.TraceSearchMetadata{
			{
				TraceID:         "1234",
				RootServiceName: "root",
			},
		},
		Metrics: &tempopb.SearchMetrics{
			InspectedTraces: 1,
			InspectedBytes:  2,
		},
	})

	// now we should get the same metadata as above
	require.Equal(t, &tempopb.SearchResponse{
		Traces: []*tempopb.TraceSearchMetadata{
			{
				TraceID:         "1234",
				RootServiceName: "root",
			},
		},
		Metrics: &tempopb.SearchMetrics{
			CompletedJobs:   1,
			InspectedTraces: 1,
			InspectedBytes:  2,
		},
	}, diffProgress.result().response)

	// metrics, but the trace hasn't change
	require.Equal(t, &tempopb.SearchResponse{
		Traces: []*tempopb.TraceSearchMetadata{},
		Metrics: &tempopb.SearchMetrics{
			CompletedJobs:   1,
			InspectedTraces: 1,
			InspectedBytes:  2,
		},
	}, diffProgress.result().response)

	// new traces
	diffProgress.addResponse(&tempopb.SearchResponse{
		Traces: []*tempopb.TraceSearchMetadata{
			{
				TraceID:           "5678",
				RootServiceName:   "root",
				StartTimeUnixNano: 1, // forces order
			},
			{
				TraceID:           "9011",
				RootServiceName:   "root",
				StartTimeUnixNano: 2,
			},
		},
		Metrics: &tempopb.SearchMetrics{
			InspectedTraces: 1,
			InspectedBytes:  2,
		},
	})

	require.Equal(t, &tempopb.SearchResponse{
		Traces: []*tempopb.TraceSearchMetadata{
			{
				TraceID:           "9011",
				RootServiceName:   "root",
				StartTimeUnixNano: 2,
			},
			{
				TraceID:           "5678",
				RootServiceName:   "root",
				StartTimeUnixNano: 1,
			},
		},
		Metrics: &tempopb.SearchMetrics{
			CompletedJobs:   2,
			InspectedTraces: 2,
			InspectedBytes:  4,
		},
	}, diffProgress.result().response)

	// write over existing trace
	diffProgress.addResponse(&tempopb.SearchResponse{
		Traces: []*tempopb.TraceSearchMetadata{
			{
				TraceID:    "1234",
				DurationMs: 100,
			},
		},
		Metrics: &tempopb.SearchMetrics{
			InspectedTraces: 1,
			InspectedBytes:  2,
		},
	})

	require.Equal(t, &tempopb.SearchResponse{
		Traces: []*tempopb.TraceSearchMetadata{
			{
				TraceID:         "1234",
				RootServiceName: "root",
				DurationMs:      100,
			},
		},
		Metrics: &tempopb.SearchMetrics{
			CompletedJobs:   3,
			InspectedTraces: 3,
			InspectedBytes:  6,
		},
	}, diffProgress.result().response)
}
