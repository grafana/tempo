package v1

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/grafana/dskit/httpgrpc"
	"github.com/grafana/tempo/modules/frontend/pipeline"
	"github.com/stretchr/testify/require"
)

func TestRequestBatchBasics(t *testing.T) {
	rb := &requestBatch{}

	const totalRequests = 3

	for i := byte(0); i < totalRequests; i++ {
		req := httptest.NewRequest("GET", "http://example.com", bytes.NewReader([]byte{i}))
		_ = rb.add(&request{
			request: pipeline.NewHTTPRequest(req),
		})
	}

	// confirm len
	require.Equal(t, totalRequests, rb.len())

	// confirm grpc requests are ordered
	grpcRequests := rb.httpGrpcRequests()
	require.Equal(t, totalRequests, len(grpcRequests))
	for i := byte(0); i < byte(len(grpcRequests)); i++ {
		require.Equal(t, []byte{i}, grpcRequests[i].Body)
	}

	// clear and confirm len
	rb.clear()
	require.Equal(t, 0, rb.len())

	grpcRequests = rb.httpGrpcRequests()
	require.Equal(t, 0, len(grpcRequests))
}

func TestRequestBatchContextError(t *testing.T) {
	rb := &requestBatch{}
	ctx := context.Background()
	const totalRequests = 3

	req := httptest.NewRequest("GET", "http://example.com", nil)
	prequest := pipeline.NewHTTPRequest(req)
	prequest.WithContext(ctx)

	for i := 0; i < totalRequests-1; i++ {
		_ = rb.add(&request{request: prequest})
	}

	// add a cancel context
	cancelCtx, cancel := context.WithCancel(ctx)
	prequest = pipeline.NewHTTPRequest(req)
	prequest.WithContext(cancelCtx)

	_ = rb.add(&request{request: prequest})

	// confirm ok
	require.NoError(t, rb.contextError())

	// cancel anc confirm error
	cancel()
	require.Error(t, rb.contextError())
}

func TestDoneChanCloses(_ *testing.T) {
	rb := &requestBatch{}

	const totalRequests = 3

	ctx := context.Background()
	cancelCtx, cancel := context.WithCancel(ctx)

	req := httptest.NewRequest("GET", "http://example.com", nil)
	prequest := pipeline.NewHTTPRequest(req)
	prequest.WithContext(cancelCtx)

	for i := 0; i < totalRequests-1; i++ {
		_ = rb.add(&request{request: prequest})
	}

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		done := rb.doneChan(make(<-chan struct{}))
		<-done
		wg.Done()
	}()

	cancel()
	wg.Wait()
	// this test won't return unless doneChan closes
}

func TestDoneChanClosesOnStop(_ *testing.T) {
	rb := &requestBatch{}

	const totalRequests = 3
	req := httptest.NewRequest("GET", "http://example.com", nil)

	for i := 0; i < totalRequests-1; i++ {
		_ = rb.add(&request{
			request: pipeline.NewHTTPRequest(req),
		})
	}

	stop := make(chan struct{})
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		done := rb.doneChan(stop)
		<-done
		wg.Done()
	}()

	close(stop)
	wg.Wait()
	// this test won't return unless doneChan closes on stop
}

func TestErrorsPropagateUpstream(t *testing.T) {
	rb := &requestBatch{}

	const totalRequests = 3
	wg := &sync.WaitGroup{}

	for i := 0; i < totalRequests-1; i++ {
		errChan := make(chan error)

		wg.Add(1)
		go func() {
			err := <-errChan
			require.ErrorContains(t, err, "foo")
			wg.Done()
		}()

		_ = rb.add(&request{
			err: errChan,
		})
	}

	rb.reportErrorToPipeline(errors.New("foo"))
	wg.Wait()
	// this test won't return unless all errChan's receive an error
}

func TestResponsesPropagateUpstream(t *testing.T) {
	rb := &requestBatch{}

	const totalRequests = 3
	wg := &sync.WaitGroup{}

	for i := int32(0); i < totalRequests; i++ {
		responseChan := make(chan *http.Response)

		wg.Add(1)
		go func(expectedCode int32) {
			resp := <-responseChan
			require.Equal(t, expectedCode, resp.StatusCode)
			wg.Done()
		}(i)

		_ = rb.add(&request{
			response: responseChan,
		})
	}

	// if the reported results mismatches the actual length we should error
	err := rb.reportResultsToPipeline(make([]*httpgrpc.HTTPResponse, totalRequests+1))
	require.Error(t, err)

	responses := make([]*httpgrpc.HTTPResponse, totalRequests)
	for i := int32(0); i < totalRequests; i++ {
		responses[i] = &httpgrpc.HTTPResponse{
			Code: i,
		}
	}
	err = rb.reportResultsToPipeline(responses)
	require.NoError(t, err)

	wg.Wait()
	// this test won't return unless all responseChan's receive a response
}
