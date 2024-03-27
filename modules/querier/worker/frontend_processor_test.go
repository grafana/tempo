package worker

import (
	"context"
	"errors"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/grafana/dskit/grpcclient"
	"github.com/grafana/dskit/httpgrpc"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
)

type RequestHandlerFunc func(context.Context, *httpgrpc.HTTPRequest) (*httpgrpc.HTTPResponse, error)

// ServeHTTP calls f(w, r).
func (f RequestHandlerFunc) Handle(c context.Context, r *httpgrpc.HTTPRequest) (*httpgrpc.HTTPResponse, error) {
	return f(c, r)
}

func TestRunRequests(t *testing.T) {
	handler := func(ctx context.Context, r *httpgrpc.HTTPRequest) (*httpgrpc.HTTPResponse, error) {
		time.Sleep(time.Millisecond)
		return &httpgrpc.HTTPResponse{
			Body: r.Body,
		}, nil
	}

	inf := newFrontendProcessor(Config{GRPCClientConfig: grpcclient.Config{MaxSendMsgSize: 10}}, RequestHandlerFunc(handler), log.NewNopLogger())
	fp := inf.(*frontendProcessor)

	totalRequests := byte(10)
	reqs := []*httpgrpc.HTTPRequest{}
	for i := byte(0); i < totalRequests; i++ {
		reqs = append(reqs, &httpgrpc.HTTPRequest{
			Body: []byte{i},
		})
	}

	resps := fp.runRequests(context.Background(), reqs)
	require.Len(t, resps, int(totalRequests))

	for i, resp := range resps {
		require.Equal(t, []byte{byte(i)}, resp.Body)
	}

	// check that counter metric is working
	m := &dto.Metric{}
	err := fp.metricRequestsTotal.Write(m)
	require.NoError(t, err)
	require.Equal(t, float64(totalRequests), m.Counter.GetValue())
}

func TestHandleSendError(t *testing.T) {
	inf := newFrontendProcessor(Config{}, nil, log.NewNopLogger())
	fp := inf.(*frontendProcessor)

	err := fp.handleSendError(nil)
	require.NoError(t, err)

	err = fp.handleSendError(context.Canceled)
	require.NoError(t, err)

	err = fp.handleSendError(fmt.Errorf("%w", context.Canceled))
	require.NoError(t, err)

	err = fp.handleSendError(io.EOF)
	require.NoError(t, err)

	err = fp.handleSendError(errors.New("foo"))
	require.Error(t, err)
}
