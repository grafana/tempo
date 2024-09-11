package v1

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"github.com/grafana/dskit/httpgrpc"
	"github.com/grafana/dskit/multierror"
)

type requestBatch struct {
	// requests that represent that communicate back with the upstream pipeline
	pipelineRequests []*request
	// requests that are actually sent to the queriers
	wireRequests []*httpgrpc.HTTPRequest
}

// jpe - necessary?
type buffer struct {
	buff []byte
	io.ReadCloser
}

func (b *buffer) Bytes() []byte {
	return b.buff
}

func (b *requestBatch) clear() {
	b.pipelineRequests = b.pipelineRequests[:0]
	b.wireRequests = b.wireRequests[:0]
}

func (b *requestBatch) add(r *request) error {
	b.pipelineRequests = append(b.pipelineRequests, r)

	req, err := httpgrpc.FromHTTPRequest(r.request.HTTPRequest())
	if err != nil {
		return err
	}

	b.wireRequests = append(b.wireRequests, req)

	return nil
}

func (b *requestBatch) httpGrpcRequests() []*httpgrpc.HTTPRequest {
	return b.wireRequests
}

func (b *requestBatch) len() int {
	return len(b.pipelineRequests)
}

func (b *requestBatch) contextError() error {
	multiErr := multierror.New()

	for _, r := range b.pipelineRequests {
		if err := r.OriginalContext().Err(); err != nil {
			multiErr.Add(err)
		}
	}

	return multiErr.Err()
}

// doneChan() returns a channel that can be used to watch for context cancellation
// across the entire batch. it only closes the returned channel if all contexts
// in the batch are done. the consequence of this is that if the batch is broken
// across upstream http queries then the queriers may continue working on a query
// they don't need to. this should be rare. nearly always all jobs in one batch
// will belong to the same upstream http query.
func (b *requestBatch) doneChan(stop <-chan struct{}) <-chan struct{} {
	if len(b.pipelineRequests) == 1 {
		return b.pipelineRequests[0].OriginalContext().Done()
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		// tests each request context and only closes done if all are done.
		// technically it is only testing one a time, but the loop will only complete
		// if all are done.
		for _, r := range b.pipelineRequests {
			select {
			case <-r.OriginalContext().Done():
			case <-stop:
				return
			}
		}
	}()

	return done
}

// reportErrorToPipeline sends errors back up the query frontend http pipeline
func (b *requestBatch) reportErrorToPipeline(err error) {
	for _, r := range b.pipelineRequests {
		r.err <- err
	}
}

// reportResultsToPipeline sends errors back up the query frontend http pipeline
func (b *requestBatch) reportResultsToPipeline(responses []*httpgrpc.HTTPResponse) error {
	if len(responses) != len(b.pipelineRequests) {
		return fmt.Errorf("incorrect number of responses to pipeline %d != %d", len(responses), len(b.pipelineRequests))
	}

	for i, r := range b.pipelineRequests {
		r.response <- httpGRPCResponseToHTTPResponse(responses[i])
	}

	return nil
}

func httpGRPCResponseToHTTPResponse(resp *httpgrpc.HTTPResponse) *http.Response {
	// translate back
	httpResp := &http.Response{
		StatusCode:    int(resp.Code),
		Body:          &buffer{buff: resp.Body, ReadCloser: io.NopCloser(bytes.NewReader(resp.Body))},
		Header:        http.Header{},
		ContentLength: int64(len(resp.Body)),
	}
	for _, h := range resp.Headers {
		httpResp.Header[h.Key] = h.Values
	}

	return httpResp
}
