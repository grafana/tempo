package frontend

import "testing"

func TestStreamingSearchHandler(t *testing.T) {}

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
// 	}, next, o, nil, log.NewNopLogger(), "", nil)
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
