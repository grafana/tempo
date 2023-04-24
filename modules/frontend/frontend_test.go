package frontend

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockNextTripperware struct{}

func (s *mockNextTripperware) RoundTrip(_ *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewReader([]byte("next"))),
	}, nil
}

func TestFrontendRoundTripsSearch(t *testing.T) {
	next := &mockNextTripperware{}
	f, err := New(Config{
		TraceByID: TraceByIDConfig{
			QueryShards: minQueryShards,
			SLO:         testSLOcfg,
		},
		Search: SearchConfig{
			Sharder: SearchSharderConfig{
				ConcurrentRequests:    defaultConcurrentRequests,
				TargetBytesPerRequest: defaultTargetBytesPerRequest,
			},
			SLO: testSLOcfg,
		},
	}, next, nil, nil, "", log.NewNopLogger(), nil)
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "/", nil)

	// search is a blind passthrough. easy!
	res := httptest.NewRecorder()
	f.SearchHandler.ServeHTTP(res, req)
	assert.Equal(t, res.Body.String(), "next")
}

func TestFrontendBadConfigFails(t *testing.T) {
	f, err := New(Config{
		TraceByID: TraceByIDConfig{
			QueryShards: minQueryShards - 1,
		},
		Search: SearchConfig{
			Sharder: SearchSharderConfig{
				ConcurrentRequests:    defaultConcurrentRequests,
				TargetBytesPerRequest: defaultTargetBytesPerRequest,
			},
			SLO: testSLOcfg,
		},
	}, nil, nil, nil, "", log.NewNopLogger(), nil)
	assert.EqualError(t, err, "frontend query shards should be between 2 and 256 (both inclusive)")
	assert.Nil(t, f)

	f, err = New(Config{
		TraceByID: TraceByIDConfig{
			QueryShards: maxQueryShards + 1,
			SLO:         testSLOcfg,
		},
		Search: SearchConfig{
			Sharder: SearchSharderConfig{
				ConcurrentRequests:    defaultConcurrentRequests,
				TargetBytesPerRequest: defaultTargetBytesPerRequest,
			},
			SLO: testSLOcfg,
		},
	}, nil, nil, nil, "", log.NewNopLogger(), nil)
	assert.EqualError(t, err, "frontend query shards should be between 2 and 256 (both inclusive)")
	assert.Nil(t, f)

	f, err = New(Config{
		TraceByID: TraceByIDConfig{
			QueryShards: maxQueryShards,
			SLO:         testSLOcfg,
		},
		Search: SearchConfig{
			Sharder: SearchSharderConfig{
				ConcurrentRequests:    0,
				TargetBytesPerRequest: defaultTargetBytesPerRequest,
			},
			SLO: testSLOcfg,
		},
	}, nil, nil, nil, "", log.NewNopLogger(), nil)
	assert.EqualError(t, err, "frontend search concurrent requests should be greater than 0")
	assert.Nil(t, f)

	f, err = New(Config{
		TraceByID: TraceByIDConfig{
			QueryShards: maxQueryShards,
			SLO:         testSLOcfg,
		},
		Search: SearchConfig{
			Sharder: SearchSharderConfig{
				ConcurrentRequests:    defaultConcurrentRequests,
				TargetBytesPerRequest: 0,
			},
			SLO: testSLOcfg,
		},
	}, nil, nil, nil, "", log.NewNopLogger(), nil)
	assert.EqualError(t, err, "frontend search target bytes per request should be greater than 0")
	assert.Nil(t, f)

	f, err = New(Config{
		TraceByID: TraceByIDConfig{
			QueryShards: maxQueryShards,
			SLO:         testSLOcfg,
		},
		Search: SearchConfig{
			Sharder: SearchSharderConfig{
				ConcurrentRequests:    defaultConcurrentRequests,
				TargetBytesPerRequest: defaultTargetBytesPerRequest,
				QueryIngestersUntil:   time.Minute,
				QueryBackendAfter:     time.Hour,
			},
			SLO: testSLOcfg,
		},
	}, nil, nil, nil, "", log.NewNopLogger(), nil)
	assert.EqualError(t, err, "query backend after should be less than or equal to query ingester until")
	assert.Nil(t, f)
}

// jpe do something with this test?
// var fakeGRPCAuthStreamMiddleware = func(srv interface{}, ss grpc.ServerStream, _ *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
// 	ctx := user.InjectOrgID(ss.Context(), util.FakeTenantID)
// 	return handler(srv, serverStream{
// 		ctx:          ctx,
// 		ServerStream: ss,
// 	})
// }

// type serverStream struct {
// 	ctx context.Context
// 	grpc.ServerStream
// }

// func (ss serverStream) Context() context.Context {
// 	return ss.ctx
// }

// func TestFrontendStreamingSearch(t *testing.T) {
// 	o, err := overrides.NewOverrides(overrides.Limits{})
// 	require.NoError(t, err)

// 	next := &mockNextTripperware{}
// 	f, err := New(Config{
// 		TraceByID: TraceByIDConfig{
// 			QueryShards: minQueryShards,
// 			SLO:         testSLOcfg,
// 		},
// 		Search: SearchConfig{
// 			Sharder: SearchSharderConfig{
// 				ConcurrentRequests:    defaultConcurrentRequests,
// 				TargetBytesPerRequest: defaultTargetBytesPerRequest,
// 			},
// 			SLO: testSLOcfg,
// 		},
// 	}, next, o, nil, log.NewNopLogger(), nil)
// 	require.NoError(t, err)

// 	srv, err := server.New(server.Config{
// 		GRPCListenPort:           9090,
// 		GRPCServerMaxSendMsgSize: 100 * 1024 * 1024,
// 		GPRCServerMaxRecvMsgSize: 100 * 1024 * 1024,
// 		GRPCStreamMiddleware: []grpc.StreamServerInterceptor{
// 			fakeGRPCAuthStreamMiddleware,
// 		},
// 	})
// 	require.NoError(t, err)

// 	tempopb.RegisterStreamingQuerierServer(srv.GRPC, f)

// 	// start grpc server
// 	go func() {
// 		err := srv.Run()
// 		require.NoError(t, err)
// 	}()

// 	// create a fake grpc client
// 	conn, err := grpc.DialContext(context.Background(), "localhost:9090", grpc.WithInsecure(), grpc.WithBlock())
// 	require.NoError(t, err)
// 	defer conn.Close()

// 	client := tempopb.NewStreamingQuerierClient(conn)
// 	c, err := client.Search(context.Background(), &tempopb.SearchRequest{
// 		Start: 0,
// 		End:   10,
// 		Query: "{}",
// 	})
// 	require.NoError(t, err)

// 	for {
// 		_, err := c.Recv()
// 		if err == io.EOF {
// 			break
// 		}
// 		require.NoError(t, err)
// 	}
// }
