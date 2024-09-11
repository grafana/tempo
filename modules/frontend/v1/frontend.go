package v1

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/grafana/dskit/flagext"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/grafana/dskit/httpgrpc"
	"github.com/grafana/dskit/services"
	"github.com/grafana/dskit/tenant"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/grafana/tempo/modules/frontend/pipeline"
	"github.com/grafana/tempo/modules/frontend/queue"
	"github.com/grafana/tempo/modules/frontend/v1/frontendv1pb"
	"github.com/grafana/tempo/pkg/util"
)

var tracer = otel.Tracer("modules/frontend/v1")

// Config for a Frontend.
type Config struct {
	MaxOutstandingPerTenant int                    `yaml:"max_outstanding_per_tenant"`
	MaxBatchSize            int                    `yaml:"max_batch_size"`
	LogQueryRequestHeaders  flagext.StringSliceCSV `yaml:"log_query_request_headers"`
}

// RegisterFlags adds the flags required to config this to the given FlagSet.
func (cfg *Config) RegisterFlags(f *flag.FlagSet) {
	f.IntVar(&cfg.MaxOutstandingPerTenant, "querier.max-outstanding-requests-per-tenant", 2000, "Maximum number of outstanding requests per tenant per frontend; requests beyond this error with HTTP 429.")
	f.Var(&cfg.LogQueryRequestHeaders, "query-frontend.log-query-request-headers", "Comma-separated list of request header names to include in query logs. Applies to both query stats and slow queries logs.")
}

// Frontend queues HTTP requests, dispatches them to backends, and handles retries
// for requests which failed.
type Frontend struct {
	services.Service

	cfg Config
	log log.Logger

	requestQueue *queue.RequestQueue
	activeUsers  *util.ActiveUsersCleanupService

	connectedQuerierWorkers *atomic.Int32

	// Subservices manager.
	subservices        *services.Manager
	subservicesWatcher *services.FailureWatcher

	// Metrics.
	queueLength       *prometheus.GaugeVec
	discardedRequests *prometheus.CounterVec
	numClients        prometheus.GaugeFunc
	queueDuration     prometheus.Histogram
	actualBatchSize   prometheus.Histogram
}

// jpe - can i get rid of this?
type request struct {
	enqueueTime time.Time
	queueSpan   trace.Span

	request  pipeline.Request
	err      chan error
	response chan *http.Response
}

func (r *request) Weight() int {
	return r.request.Weight()
}

func (r *request) OriginalContext() context.Context {
	return r.request.Context()
}

// New creates a new frontend. Frontend implements service, and must be started and stopped.
func New(cfg Config, log log.Logger, registerer prometheus.Registerer) (*Frontend, error) {
	const batchBucketCount = 5
	if cfg.MaxBatchSize <= 0 {
		return nil, errors.New("max_batch_size must be positive")
	}
	batchBucketSize := float64(cfg.MaxBatchSize) / float64(batchBucketCount)

	f := &Frontend{
		cfg: cfg,
		log: log,
		queueLength: promauto.With(registerer).NewGaugeVec(prometheus.GaugeOpts{
			Name: "tempo_query_frontend_queue_length",
			Help: "Number of queries in the queue.",
		}, []string{"user"}),
		discardedRequests: promauto.With(registerer).NewCounterVec(prometheus.CounterOpts{
			Name: "tempo_query_frontend_discarded_requests_total",
			Help: "Total number of query requests discarded.",
		}, []string{"user"}),
		queueDuration: promauto.With(registerer).NewHistogram(prometheus.HistogramOpts{
			Name:                            "tempo_query_frontend_queue_duration_seconds",
			Help:                            "Time spend by requests queued.",
			Buckets:                         prometheus.DefBuckets,
			NativeHistogramBucketFactor:     1.1,
			NativeHistogramMaxBucketNumber:  100,
			NativeHistogramMinResetDuration: 1 * time.Hour,
		}),
		actualBatchSize: promauto.With(registerer).NewHistogram(prometheus.HistogramOpts{
			Name:    "tempo_query_frontend_actual_batch_size",
			Help:    "Batch size.",
			Buckets: prometheus.LinearBuckets(1, batchBucketSize, batchBucketCount),
		}),
		connectedQuerierWorkers: &atomic.Int32{},
	}

	f.requestQueue = queue.NewRequestQueue(cfg.MaxOutstandingPerTenant, f.queueLength, f.discardedRequests)
	f.activeUsers = util.NewActiveUsersCleanupWithDefaultValues(f.cleanupInactiveUserMetrics)

	var err error
	f.subservices, err = services.NewManager(f.requestQueue, f.activeUsers)
	if err != nil {
		return nil, err
	}

	f.numClients = promauto.With(registerer).NewGaugeFunc(prometheus.GaugeOpts{
		Name: "tempo_query_frontend_connected_clients",
		Help: "Number of worker clients currently connected to the frontend.",
	}, func() float64 {
		return float64(f.connectedQuerierWorkers.Load())
	})

	f.Service = services.NewBasicService(f.starting, f.running, f.stopping)
	return f, nil
}

func (f *Frontend) starting(ctx context.Context) error {
	f.subservicesWatcher = services.NewFailureWatcher()
	f.subservicesWatcher.WatchManager(f.subservices)

	if err := services.StartManagerAndAwaitHealthy(ctx, f.subservices); err != nil {
		return fmt.Errorf("unable to start frontend subservices: %w", err)
	}

	return nil
}

func (f *Frontend) running(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case err := <-f.subservicesWatcher.Chan():
			return fmt.Errorf("frontend subservice failed: %w", err)
		}
	}
}

func (f *Frontend) stopping(_ error) error {
	// This will also stop the requests queue, which stop accepting new requests and errors out any pending requests.
	return services.StopManagerAndAwaitStopped(context.Background(), f.subservices)
}

func (f *Frontend) cleanupInactiveUserMetrics(user string) {
	f.queueLength.DeleteLabelValues(user)
	f.discardedRequests.DeleteLabelValues(user)
}

// jpe - convert to/from grpc madness her
//   - rewrite modules/frontend code to take a pipeline.RoundTripper

// func (f *Frontend) RoundTripGRPC(ctx context.Context, req *httpgrpc.HTTPRequest) (*httpgrpc.HTTPResponse, error) {
func (f *Frontend) RoundTrip(req pipeline.Request) (*http.Response, error) {
	// Propagate trace context in gRPC too - this will be ignored if using HTTP.
	//  jpe - move this until after the conversion to httpgrpc.HTTPRequest
	// carrier := (*httpgrpcutil.HttpgrpcHeadersCarrier)(req)
	// otel.GetTextMapPropagator().Inject(ctx, carrier)

	request := request{
		request: req,

		// Buffer of 1 to ensure response can be written by the server side
		// of the Process stream, even if this goroutine goes away due to
		// client context cancellation.
		err:      make(chan error, 1),
		response: make(chan *http.Response, 1),
	}

	ctx := req.Context()
	if err := f.queueRequest(ctx, &request); err != nil {
		return nil, err
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()

	case resp := <-request.response:
		return resp, nil

	case err := <-request.err:
		return nil, err
	}
}

// Process allows backends to pull requests from the frontend.
func (f *Frontend) Process(server frontendv1pb.Frontend_ProcessServer) error {
	_, querierFeatures, err := getQuerierInfo(server)
	if err != nil {
		return err
	}

	f.connectedQuerierWorkers.Add(1)
	defer f.connectedQuerierWorkers.Add(-1)

	lastUserIndex := queue.FirstUser()

	reqBatch := &requestBatch{}
	batchSize := 1
	if querierSupportsBatching(querierFeatures) {
		batchSize = f.cfg.MaxBatchSize
	}
	for {
		reqSlice := make([]queue.Request, batchSize)
		reqSlice, idx, err := f.requestQueue.GetNextRequestForQuerier(server.Context(), lastUserIndex, reqSlice)
		if err != nil {
			return err
		}
		lastUserIndex = idx

		reqBatch.clear()
		for _, reqWrapper := range reqSlice {
			req := reqWrapper.(*request)

			f.queueDuration.Observe(time.Since(req.enqueueTime).Seconds())
			req.queueSpan.End()

			// only add if not expired
			if req.OriginalContext().Err() != nil {
				continue
			}

			err = reqBatch.add(req)
			if err != nil {
				return fmt.Errorf("unexpected error adding request to batch: %w", err)
			}
		}

		// if all requests are expired then continue requesting jobs for this user. this nicely
		// drains a large expired query for a tenant and allows them to execute a real query
		if reqBatch.len() == 0 {
			lastUserIndex = lastUserIndex.ReuseLastUser()
			continue
		}

		f.actualBatchSize.Observe(float64(reqBatch.len()))

		// Handle the stream sending & receiving on a goroutine so we can
		// monitoring the contexts in a select and cancel things appropriately.
		resps := make(chan *frontendv1pb.ClientToFrontend, 1)
		errs := make(chan error, 1)
		go func() {
			// todo: we are still sending the old Type_HTTP_REQUEST for backwards compat
			// with queriers that don't support the new Type_HTTP_REQUEST_BATCH. this feature
			// was introduced in 2.2. We should remove this in a few versions
			if reqBatch.len() == 1 {
				err = server.Send(&frontendv1pb.FrontendToClient{
					Type:        frontendv1pb.Type_HTTP_REQUEST,
					HttpRequest: reqBatch.httpGrpcRequests()[0],
				})
			} else {
				err = server.Send(&frontendv1pb.FrontendToClient{
					Type:             frontendv1pb.Type_HTTP_REQUEST_BATCH,
					HttpRequestBatch: reqBatch.httpGrpcRequests(),
				})
			}
			if err != nil {
				errs <- err
				return
			}

			resp, err := server.Recv()
			if err != nil {
				errs <- err
				return
			}

			resps <- resp
		}()

		err = reportResponseUpstream(reqBatch, errs, resps)
		if err != nil {
			return err
		}
	}
}

func reportResponseUpstream(reqBatch *requestBatch, errs chan error, resps chan *frontendv1pb.ClientToFrontend) error {
	stopCh := make(chan struct{})
	defer close(stopCh)

	select {
	// If the upstream request is cancelled, we need to cancel the
	// downstream req.  Only way we can do that is to close the stream.
	// The worker client is expecting this semantics.
	case <-reqBatch.doneChan(stopCh):
		return reqBatch.contextError()

	// Is there was an error handling this request due to network IO,
	// then error out this upstream request _and_ stream.
	// The assumption appears to be that the querier will reestablish in the event of this kind
	// of error.
	case err := <-errs:
		reqBatch.reportErrorToPipeline(err)
		return err

	// Happy path :D
	case resp := <-resps:
		// todo: like above support for batches and single requests
		// can be removed in a few versions once all queriers support batching
		var err error
		if len(resp.HttpResponseBatch) == 0 {
			err = reqBatch.reportResultsToPipeline([]*httpgrpc.HTTPResponse{resp.HttpResponse})
		} else {
			err = reqBatch.reportResultsToPipeline(resp.HttpResponseBatch)
		}
		if err != nil {
			return fmt.Errorf("unexpected error reporting results upstream: %w", err)
		}
	}

	return nil
}

func (f *Frontend) NotifyClientShutdown(_ context.Context, req *frontendv1pb.NotifyClientShutdownRequest) (*frontendv1pb.NotifyClientShutdownResponse, error) {
	level.Info(f.log).Log("msg", "received shutdown notification from querier", "querier", req.GetClientID())

	return &frontendv1pb.NotifyClientShutdownResponse{}, nil
}

func getQuerierInfo(server frontendv1pb.Frontend_ProcessServer) (string, int32, error) {
	err := server.Send(&frontendv1pb.FrontendToClient{
		Type: frontendv1pb.Type_GET_ID,
		// Old queriers don't support GET_ID, and will try to use the request.
		// To avoid confusing them, include dummy request.
		HttpRequest: &httpgrpc.HTTPRequest{
			Method: "GET",
			Url:    "/invalid_request_sent_by_frontend",
		},
	})
	if err != nil {
		return "", int32(frontendv1pb.Feature_NONE), err
	}

	resp, err := server.Recv()
	if err != nil {
		return "", int32(frontendv1pb.Feature_NONE), err
	}

	// Old queriers will return empty string, which is fine. All old queriers will be
	// treated as single querier with lot of connections.
	// (Note: if resp is nil, GetClientID() returns "")
	return resp.GetClientID(), resp.Features, err
}

func (f *Frontend) queueRequest(ctx context.Context, req *request) error {
	tenantIDs, err := tenant.TenantIDs(ctx)
	if err != nil {
		return err
	}

	now := time.Now()
	req.enqueueTime = now
	_, req.queueSpan = tracer.Start(ctx, "queued")

	joinedTenantID := tenant.JoinTenantIDs(tenantIDs)
	f.activeUsers.UpdateUserTimestamp(joinedTenantID, now)

	return f.requestQueue.EnqueueRequest(joinedTenantID, req)
}

// CheckReady determines if the query frontend is ready.  Function parameters/return
// chosen to match the same method in the ingester
func (f *Frontend) CheckReady(_ context.Context) error {
	// if we have more than one querier connected we will consider ourselves ready
	connectedClients := f.connectedQuerierWorkers.Load()
	if connectedClients > 0 {
		return nil
	}

	msg := fmt.Sprintf("not ready: number of queriers connected to query-frontend is %d", int64(connectedClients))
	level.Info(f.log).Log("msg", msg)
	return errors.New(msg)
}

func querierSupportsBatching(features int32) bool {
	return features&int32(frontendv1pb.Feature_REQUEST_BATCHING) != 0
}
