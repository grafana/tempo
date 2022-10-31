package worker

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/gogo/status"
	"github.com/grafana/dskit/backoff"
	"github.com/weaveworks/common/httpgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	"github.com/grafana/tempo/modules/frontend/v1/frontendv1pb"
	querier_stats "github.com/grafana/tempo/modules/querier/stats"
	"github.com/grafana/tempo/pkg/scheduler/queue"
)

var (
	processorBackoffConfig = backoff.Config{
		MinBackoff: 50 * time.Millisecond,
		MaxBackoff: 1 * time.Second,
	}
)

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

// processQueriesOnSingleStream loops, trying to establish a stream to the frontend to begin request processing.
func (fp *frontendProcessor) processQueriesOnSingleStream(ctx context.Context, conn *grpc.ClientConn, address string) {
	if ctx.Err() != nil {
		level.Debug(fp.log).Log("msg", "returning early for context err", "address", address, "context.Err", ctx.Err())
		return
	}

	client := frontendv1pb.NewFrontendClient(conn)

	doneErr := func(err error) error {
		st, ok := status.FromError(err)
		if !ok {
			switch err.Error() {
			case context.Canceled.Error():
				return nil
			}
		}

		switch st.Code() {
		case codes.Canceled:
			return nil
		case codes.Unknown:
			switch st.Message() {
			case queue.ErrStopped.Error():
				return nil
			}

		default:
			level.Warn(fp.log).Log("msg", "unhandled status code", "code", st.Code())
		}

		return err
	}

	backoff := backoff.New(ctx, processorBackoffConfig)
	for backoff.Ongoing() {
		c, err := client.Process(ctx)
		if err != nil {
			if doneErr(err) != nil {
				level.Error(fp.log).Log("msg", "error contacting frontend", "address", address, "err", err)
			}

			backoff.Wait()
			continue
		}

		if err := fp.process(c); err != nil {
			if doneErr(err) != nil {
				level.Error(fp.log).Log("msg", "error processing requests", "address", address, "err", err)
			}

			backoff.Wait()
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
			go fp.runRequest(ctx, request.HttpRequest, request.StatsEnabled, func(response *httpgrpc.HTTPResponse, stats *querier_stats.Stats) error {
				return c.Send(&frontendv1pb.ClientToFrontend{
					HttpResponse: response,
					Stats:        stats,
				})
			})

		case frontendv1pb.Type_GET_ID:
			err := c.Send(&frontendv1pb.ClientToFrontend{ClientID: fp.querierID})
			if err != nil {
				return err
			}

		default:
			return fmt.Errorf("unknown request type: %v", request.Type)
		}
	}
}

func (fp *frontendProcessor) runRequest(ctx context.Context, request *httpgrpc.HTTPRequest, statsEnabled bool, sendHTTPResponse func(response *httpgrpc.HTTPResponse, stats *querier_stats.Stats) error) {
	var stats *querier_stats.Stats
	if statsEnabled {
		stats, ctx = querier_stats.ContextWithEmptyStats(ctx)
	}

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

	if err := sendHTTPResponse(response, stats); err != nil {
		level.Error(fp.log).Log("msg", "error processing requests", "err", err)
	}
}
