package frontend

import (
	"context"
	"io"
	"net/http"

	"github.com/cortexproject/cortex/pkg/util"
	"github.com/weaveworks/common/httpgrpc"
	"github.com/weaveworks/common/httpgrpc/server"
)

const (
	// StatusClientClosedRequest is the status code for when a client request cancellation of an http request
	StatusClientClosedRequest = 499
)

var (
	errCanceled              = httpgrpc.Errorf(StatusClientClosedRequest, context.Canceled.Error())
	errDeadlineExceeded      = httpgrpc.Errorf(http.StatusGatewayTimeout, context.DeadlineExceeded.Error())
	errRequestEntityTooLarge = httpgrpc.Errorf(http.StatusRequestEntityTooLarge, "http: request body too large")
)

type HandlerBlerg struct { //jpe better name
	roundTripper http.RoundTripper
}

func NewHandler(rt http.RoundTripper) http.Handler {
	return &HandlerBlerg{
		roundTripper: rt,
	}
}

func (f *HandlerBlerg) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer func() {
		_ = r.Body.Close()
	}()

	resp, err := f.roundTripper.RoundTrip(r)
	if err != nil {
		writeError(w, err)
		return
	}

	w.WriteHeader(resp.StatusCode)
	// we don't check for copy error as there is no much we can do at this point
	_, _ = io.Copy(w, resp.Body)
}

func writeError(w http.ResponseWriter, err error) {
	switch err {
	case context.Canceled:
		err = errCanceled
	case context.DeadlineExceeded:
		err = errDeadlineExceeded
	default:
		if util.IsRequestBodyTooLarge(err) {
			err = errRequestEntityTooLarge
		}
	}
	// use server.WriteError b/c it will handle the httpgrpc error bridge
	server.WriteError(w, err)
}
