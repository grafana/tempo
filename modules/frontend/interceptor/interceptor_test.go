package interceptor

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/grafana/tempo/pkg/gogocodec"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/encoding"
)

func TestInterceptorsCancelContextForStreaming(t *testing.T) {
	encoding.RegisterCodec(gogocodec.NewCodec())

	interceptorTimeout := time.Second
	apiTimeout := time.Second * 5

	unaryInt := NewFrontendAPIUnaryTimeout(interceptorTimeout)
	streamInt := NewFrontendAPIStreamTimeout(interceptorTimeout)

	serv := grpc.NewServer(grpc.UnaryInterceptor(unaryInt), grpc.StreamInterceptor(streamInt))
	defer serv.GracefulStop()

	srv := &mockService{apiTimeout}
	tempopb.RegisterStreamingQuerierServer(serv, srv)
	tempopb.RegisterPusherServer(serv, srv)

	listener, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)

	go func() {
		require.NoError(t, serv.Serve(listener))
	}()

	conn, err := grpc.Dial(listener.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(10e6), grpc.MaxCallSendMsgSize(10e6)))
	require.NoError(t, err)
	defer func() {
		require.NoError(t, conn.Close())
	}()

	// test that a streaming client has its context cancelled after the interceptor timeout and before the api timeout
	c := tempopb.NewStreamingQuerierClient(conn)
	client, err := c.Search(context.Background(), &tempopb.SearchRequest{})
	require.NoError(t, err)

	start := time.Now()
	_, err = client.Recv()
	require.EqualError(t, err, "rpc error: code = DeadlineExceeded desc = context deadline exceeded")
	require.LessOrEqual(t, time.Since(start), apiTimeout) // confirm that we didn't wait for the full api timeout

	// test that the pusher client does not have its context cancelled and waits for the full api timeout
	pc := tempopb.NewPusherClient(conn)

	start = time.Now()
	_, err = pc.PushBytesV2(context.Background(), &tempopb.PushBytesRequest{})
	require.NoError(t, err)
	require.GreaterOrEqual(t, time.Since(start), apiTimeout) // confirm that we did wait for the full api timeout
}

type mockService struct {
	apiTimeout time.Duration
}

func (s *mockService) Search(_ *tempopb.SearchRequest, ss tempopb.StreamingQuerier_SearchServer) error {
	select {
	case <-time.After(s.apiTimeout):
	case <-ss.Context().Done():
		return ss.Context().Err()
	}

	return nil
}

func (s *mockService) PushBytes(ctx context.Context, _ *tempopb.PushBytesRequest) (*tempopb.PushResponse, error) {
	select {
	case <-time.After(s.apiTimeout):
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	return &tempopb.PushResponse{}, nil
}

func (s *mockService) PushBytesV2(ctx context.Context, _ *tempopb.PushBytesRequest) (*tempopb.PushResponse, error) {
	select {
	case <-time.After(s.apiTimeout):
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	return &tempopb.PushResponse{}, nil
}

func (s *mockService) SearchTags(*tempopb.SearchTagsRequest, tempopb.StreamingQuerier_SearchTagsServer) error {
	return nil
}

func (s *mockService) SearchTagsV2(*tempopb.SearchTagsRequest, tempopb.StreamingQuerier_SearchTagsV2Server) error {
	return nil
}

func (s *mockService) SearchTagValues(*tempopb.SearchTagValuesRequest, tempopb.StreamingQuerier_SearchTagValuesServer) error {
	return nil
}

func (s *mockService) SearchTagValuesV2(*tempopb.SearchTagValuesRequest, tempopb.StreamingQuerier_SearchTagValuesV2Server) error {
	return nil
}

func (s *mockService) QueryRange(*tempopb.QueryRangeRequest, tempopb.StreamingQuerier_QueryRangeServer) error {
	return nil
}
