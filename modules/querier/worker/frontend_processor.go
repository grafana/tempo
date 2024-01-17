package worker

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/grafana/dskit/backoff"
	"github.com/grafana/dskit/httpgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/grafana/tempo/modules/frontend/v1/frontendv1pb"
)

var processorBackoffConfig = backoff.Config{
	MinBackoff: 50 * time.Millisecond,
	MaxBackoff: 1 * time.Second,
}

func newFrontendProcessor(cfg Config, handler RequestHandler, log log.Logger) processor {
	return &frontendProcessor{
		log:            log,
		handler:        handler,
		maxMessageSize: cfg.GRPCClientConfig.MaxSendMsgSize,
		querierID:      cfg.QuerierID,
	}
}

// Handles incoming queries from frontend.
type frontendProcessor struct {
	handler        RequestHandler
	maxMessageSize int
	querierID      string

	log log.Logger
}

// notifyShutdown implements processor.
func (fp *frontendProcessor) notifyShutdown(ctx context.Context, conn *grpc.ClientConn, address string) {
	client := frontendv1pb.NewFrontendClient(conn)

	req := &frontendv1pb.NotifyClientShutdownRequest{ClientID: fp.querierID}
	if _, err := client.NotifyClientShutdown(ctx, req); err != nil {
		// Since we're shutting down there's nothing we can do except logging it.
		level.Warn(fp.log).Log("msg", "failed to notify querier shutdown to query-frontend", "address", address, "err", err)
	}
}

// runOne loops, trying to establish a stream to the frontend to begin request processing.
func (fp *frontendProcessor) processQueriesOnSingleStream(ctx context.Context, conn *grpc.ClientConn, address string) {
	client := frontendv1pb.NewFrontendClient(conn)

	backoff := backoff.New(ctx, processorBackoffConfig)
	for backoff.Ongoing() {
		c, err := client.Process(ctx)
		if err != nil {
			level.Error(fp.log).Log("msg", "error contacting frontend", "address", address, "err", err)
			backoff.Wait()
			continue
		}

		if err := fp.process(c); err != nil {
			if status.Code(err) != codes.Canceled {
				level.Error(fp.log).Log("msg", "error processing requests", "address", address, "err", err)
				backoff.Wait()
			}
			continue
		}

		backoff.Reset()
	}
}

// process loops processing requests on an established stream.
func (fp *frontendProcessor) process(c frontendv1pb.Frontend_ProcessClient) error {
	// Build a child context so we can cancel a query when the stream is closed.
	ctx, cancel := context.WithCancel(c.Context())
	defer cancel()

	for {
		request, err := c.Recv()
		if err != nil {
			return err
		}

		switch request.Type {
		case frontendv1pb.Type_HTTP_REQUEST:
			// Handle the request on a "background" goroutine, so we go back to
			// blocking on c.Recv().  This allows us to detect the stream closing
			// and cancel the query.  We don't actually handle queries in parallel
			// here, as we're running in lock step with the server - each Recv is
			// paired with a Send.
			go func() {
				resp := fp.runRequest(ctx, request.HttpRequest)
				err := fp.handleSendError(c.Send(&frontendv1pb.ClientToFrontend{
					HttpResponse: resp,
				}))
				if err != nil {
					level.Error(fp.log).Log("msg", "error running requests", "err", err)
				}
			}()

		case frontendv1pb.Type_GET_ID:
			err := fp.handleSendError(c.Send(&frontendv1pb.ClientToFrontend{
				ClientID: fp.querierID,
				Features: int32(frontendv1pb.Feature_REQUEST_BATCHING),
			}))
			if err != nil {
				return err
			}

		case frontendv1pb.Type_HTTP_REQUEST_BATCH:
			go func() {
				resp := fp.runRequests(ctx, request.HttpRequestBatch)
				err := fp.handleSendError(c.Send(&frontendv1pb.ClientToFrontend{
					HttpResponseBatch: resp,
				}))
				if err != nil {
					level.Error(fp.log).Log("msg", "error running  batched requests", "err", err)
				}
			}()

		default:
			return fmt.Errorf("unknown request type: %v", request.Type)
		}
	}
}

func (fp *frontendProcessor) runRequests(ctx context.Context, requests []*httpgrpc.HTTPRequest) []*httpgrpc.HTTPResponse {
	wg := sync.WaitGroup{}

	responses := make([]*httpgrpc.HTTPResponse, len(requests))
	for i, request := range requests {
		wg.Add(1)
		go func(i int, request *httpgrpc.HTTPRequest) {
			defer wg.Done()
			responses[i] = fp.runRequest(ctx, request)
		}(i, request)
	}
	wg.Wait()
	return responses
}

func (fp *frontendProcessor) runRequest(ctx context.Context, request *httpgrpc.HTTPRequest) *httpgrpc.HTTPResponse {
	response, err := fp.handler.Handle(ctx, request)
	if err != nil {
		var ok bool
		response, ok = httpgrpc.HTTPResponseFromError(err)
		if !ok {
			response = &httpgrpc.HTTPResponse{
				Code: http.StatusInternalServerError,
				Body: []byte(err.Error()),
			}
		}
	}

	// Ensure responses that are too big are not retried.
	if len(response.Body) >= fp.maxMessageSize {
		errMsg := fmt.Sprintf("response larger than the max (%d vs %d)", len(response.Body), fp.maxMessageSize)
		response = &httpgrpc.HTTPResponse{
			Code: http.StatusRequestEntityTooLarge,
			Body: []byte(errMsg),
		}
		level.Error(fp.log).Log("msg", "error processing query", "err", errMsg)
	}

	return response
}

func (fp *frontendProcessor) handleSendError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, context.Canceled) {
		level.Debug(fp.log).Log("msg", "error processing requests", "err", err)
		return nil
	}

	// io.EOF errors are caused by the client. the real error will be returned on the next Recv.
	// https: //github.com/grpc/grpc-go/blob/db32c5bfeb563e7ce6661b37d6a55688cbeb4a20/stream.go#L108
	if errors.Is(err, io.EOF) {
		return nil
	}

	return err
}
