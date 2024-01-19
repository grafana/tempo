package pipeline

import (
	"context"
	"io"
	"net/http"
	"strings"
)

var _ Responses = syncResponse{}

type syncResponse struct { // jpe do i need this anymore? maybe to bridge sync to async?
	r *http.Response
}

func NewBadRequest(err error) Responses {
	return NewSyncResponse(&http.Response{
		StatusCode: http.StatusBadRequest,
		Status:     http.StatusText(http.StatusBadRequest),
		Body:       io.NopCloser(strings.NewReader(err.Error())),
	})
}

func NewSyncResponse(r *http.Response) Responses { // jpe - add to pipeline.go? better naming? note similarities to AsyncResponse. name is bad. this is an async response
	return syncResponse{ // jpe make sure doesn't allocate
		r: r,
	}
}

func (s syncResponse) Next(ctx context.Context) (*http.Response, error, bool) {
	if err := ctx.Err(); err != nil {
		return nil, err, false
	}

	return s.r, nil, true
}

var _ Responses = &asyncResponse{}

type asyncResponse struct { // jpe response fan in
	respChan chan Responses

	curResponses Responses
}

func newAsyncResponse() *asyncResponse {
	return &asyncResponse{
		respChan: make(chan Responses), // jpe - buffered?
	}
}

func (a *asyncResponse) send(r Responses) {
	a.respChan <- r
}

func (a *asyncResponse) done() {
	close(a.respChan)
}

func (a *asyncResponse) Next(ctx context.Context) (*http.Response, error, bool) {
	if a.curResponses == nil {
		select {
		case <-ctx.Done():
			return nil, ctx.Err(), false
		case r := <-a.respChan:
			a.curResponses = r
		default:
			// no more responses, bail
			return nil, nil, true
		}
	}

	return a.curResponses.Next(ctx)
}

// func (a *asyncResponse) All() *http.Response { // - jpe - just remove all?
// 	var resp *http.Response
// 	for r := range a.respChan {
// 		a.c.AddRequest(r, "") // jpe - add tenant somehow? it's a one off for the trace by id path so find a way to add there?
// 		//		resp = r.Combine(resp) - jpe take combiner?
// 	}
// 	return resp
// }
