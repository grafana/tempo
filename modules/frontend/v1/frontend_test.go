package v1

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/grafana/dskit/httpgrpc"
	"github.com/grafana/dskit/user"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/grafana/tempo/v3/modules/frontend/pipeline"
	"github.com/grafana/tempo/v3/modules/frontend/queue"
	"github.com/grafana/tempo/v3/modules/frontend/v1/frontendv1pb"
)

// mockProcessServer is a minimal Frontend_ProcessServer that drives Process()
// through the batching happy path: it completes the GET_ID handshake advertising
// REQUEST_BATCHING, then answers every batch it is sent with a matching batch of
// 200 responses. Once it has answered for totalRequests requests it cancels the
// server context so the next GetNextRequestForQuerier unblocks Process's loop.
type mockProcessServer struct {
	grpc.ServerStream
	ctx    context.Context
	cancel context.CancelFunc

	totalRequests int
	// returned for every request batch
	sendErr error
	// if set, the batches are answered once release is closed, or the server context is done
	release chan struct{}
	// if set, receives the size of every batch sent to the querier
	sent chan int

	mu            sync.Mutex
	recvCount     int
	lastBatchSize int
	served        int
}

// newHeldProcessServer returns a mockProcessServer that holds every batch it receives until
// it is released, so tests can observe the frontend while jobs execute.
func newHeldProcessServer(ctx context.Context) *mockProcessServer {
	return &mockProcessServer{ctx: ctx, release: make(chan struct{}), sent: make(chan int, 100)}
}

func (m *mockProcessServer) Context() context.Context { return m.ctx }

func (m *mockProcessServer) Send(msg *frontendv1pb.FrontendToClient) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	switch msg.Type {
	case frontendv1pb.Type_HTTP_REQUEST:
		m.lastBatchSize = 1
	case frontendv1pb.Type_HTTP_REQUEST_BATCH:
		m.lastBatchSize = len(msg.HttpRequestBatch)
	default:
		// GET_ID handshake, nothing to record.
		return nil
	}

	if m.sent != nil {
		m.sent <- m.lastBatchSize
	}

	return m.sendErr
}

func (m *mockProcessServer) Recv() (*frontendv1pb.ClientToFrontend, error) {
	m.mu.Lock()
	m.recvCount++
	if m.recvCount == 1 {
		m.mu.Unlock()
		// Handshake: advertise batching support so Process uses MaxBatchSize.
		return &frontendv1pb.ClientToFrontend{
			ClientID: "test-querier",
			Features: int32(frontendv1pb.Feature_REQUEST_BATCHING),
		}, nil
	}
	size := m.lastBatchSize
	m.mu.Unlock()

	if m.release != nil {
		select {
		case <-m.release:
		case <-m.ctx.Done():
			return nil, m.ctx.Err()
		}
	}

	batch := make([]*httpgrpc.HTTPResponse, size)
	for i := range batch {
		batch[i] = &httpgrpc.HTTPResponse{Code: http.StatusOK}
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.served += size
	if m.cancel != nil && m.served >= m.totalRequests {
		// No more work is coming; unblock the queue so Process returns.
		m.cancel()
	}

	return &frontendv1pb.ClientToFrontend{HttpResponseBatch: batch}, nil
}

// TestProcessRequestBatchReuseRace exercises Process() with batching enabled and a
// deep queue so nearly every iteration dispatches a len>1 batch. That path spawns
// the doneChan watcher goroutine, which outlives reportResponseUpstream and races
// with the next iteration's reqBatch.clear()+add() over the reused backing array.
// Run with -race to detect the data race.
func TestProcessRequestBatchReuseRace(t *testing.T) {
	cfg := Config{
		MaxOutstandingPerTenant: 100000,
		MaxBatchSize:            8,
	}
	f, err := New(cfg, log.NewNopLogger(), prometheus.NewRegistry())
	require.NoError(t, err)

	const totalRequests = 4000

	// Enqueue everything up front so the queue stays deep and GetNextRequestForQuerier
	// returns full (len>1) batches.
	noopSpan := trace.SpanFromContext(context.Background())
	for i := 0; i < totalRequests; i++ {
		httpReq := httptest.NewRequest("GET", "http://example.com", nil)
		r := &request{
			request:   pipeline.NewHTTPRequest(httpReq),
			queueSpan: noopSpan,
			err:       make(chan error, 1),
			response:  make(chan *http.Response, 1),
		}
		require.NoError(t, f.requestQueue.EnqueueRequest("test-tenant", r))
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	srv := &mockProcessServer{
		ctx:           ctx,
		cancel:        cancel,
		totalRequests: totalRequests,
	}

	err = f.Process(srv)
	require.ErrorIs(t, err, context.Canceled)
	require.GreaterOrEqual(t, srv.served, totalRequests)
}

func TestProcessLogsOversizedBatch(t *testing.T) {
	var buf bytes.Buffer
	f, err := New(Config{MaxOutstandingPerTenant: 100, MaxBatchSize: 2}, log.NewLogfmtLogger(log.NewSyncWriter(&buf)), prometheus.NewRegistry())
	require.NoError(t, err)

	noopSpan := trace.SpanFromContext(context.Background())
	for range 2 {
		httpReq := httptest.NewRequest("GET", "http://example.com", nil).WithContext(user.InjectOrgID(context.Background(), "test-tenant"))
		r := &request{
			request:   pipeline.NewHTTPRequest(httpReq),
			queueSpan: noopSpan,
			err:       make(chan error, 1),
			response:  make(chan *http.Response, 1),
		}
		require.NoError(t, f.requestQueue.EnqueueRequest("test-tenant", r))
	}

	sendErr := status.Error(codes.ResourceExhausted, "trying to send message larger than max (200 vs. 100)")
	srv := &mockProcessServer{ctx: context.Background(), sendErr: sendErr}

	err = f.Process(srv)
	require.ErrorIs(t, err, sendErr)
	require.Contains(t, buf.String(), "tenant=test-tenant")
	require.Contains(t, buf.String(), "requests=2")
	require.Contains(t, buf.String(), "bytes=")
}

func TestQueryOp(t *testing.T) {
	for _, tc := range []struct {
		name     string
		shape    pipeline.QueryShape
		expected string
	}{
		{"traces", pipeline.QueryShape{Type: pipeline.QueryTypeTraces}, "traces"},
		{"search", pipeline.QueryShape{Type: pipeline.QueryTypeSearch}, "search"},
		{"metrics", pipeline.QueryShape{Type: pipeline.QueryTypeMetrics}, "metrics"},
		{"metadata", pipeline.QueryShape{Type: pipeline.QueryTypeMetadata}, "metadata"},
		{"unstamped", pipeline.QueryShape{}, "unknown"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := pipeline.NewHTTPRequest(httptest.NewRequest("GET", "/", nil))
			req.SetQueryShape(tc.shape)

			require.Equal(t, tc.expected, queryOp(req))
		})
	}
}

func TestInflightJobsCountsQueuedAndExecuting(t *testing.T) {
	f, reg := newTestFrontend(t, 10, 1)
	done := startRoundTrip(f, tenantRequest(context.Background()))

	require.Eventually(t, func() bool { return gaugeValue(reg, inflightJobsName, pipeline.QueryTypeSearch) == 1 }, time.Second, time.Millisecond, "queued job not counted")

	r := dequeueForQuerier(t, f)
	require.Equal(t, 1.0, gaugeValue(reg, inflightJobsName, pipeline.QueryTypeSearch), "executing job not counted")

	r.response <- &http.Response{StatusCode: http.StatusOK}
	require.NoError(t, (<-done).err)
	require.Equal(t, 0.0, gaugeValue(reg, inflightJobsName, pipeline.QueryTypeSearch))
}

func TestInflightJobsReleasedOnError(t *testing.T) {
	f, reg := newTestFrontend(t, 10, 1)
	done := startRoundTrip(f, tenantRequest(context.Background()))
	require.Eventually(t, func() bool { return gaugeValue(reg, inflightJobsName, pipeline.QueryTypeSearch) == 1 }, time.Second, time.Millisecond)

	dequeueForQuerier(t, f).Fail(errors.New("querier stream closed"))
	require.Error(t, (<-done).err)
	require.Equal(t, 0.0, gaugeValue(reg, inflightJobsName, pipeline.QueryTypeSearch))
}

func TestInflightJobsReleasedOnCancel(t *testing.T) {
	f, reg := newTestFrontend(t, 10, 1)
	ctx, cancel := context.WithCancel(context.Background())
	done := startRoundTrip(f, tenantRequest(ctx))
	require.Eventually(t, func() bool { return gaugeValue(reg, inflightJobsName, pipeline.QueryTypeSearch) == 1 }, time.Second, time.Millisecond)

	cancel()
	require.ErrorIs(t, (<-done).err, context.Canceled)
	require.Equal(t, 0.0, gaugeValue(reg, inflightJobsName, pipeline.QueryTypeSearch))
}

func TestInflightJobsExcludesRejected(t *testing.T) {
	f, reg := newTestFrontend(t, 1, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	startRoundTrip(f, tenantRequest(ctx)) // fills the tenant queue
	require.Eventually(t, func() bool { return gaugeValue(reg, inflightJobsName, pipeline.QueryTypeSearch) == 1 }, time.Second, time.Millisecond)

	res := <-startRoundTrip(f, tenantRequest(context.Background()))
	require.ErrorIs(t, res.err, queue.ErrTooManyRequests)
	require.Equal(t, 1.0, gaugeValue(reg, inflightJobsName, pipeline.QueryTypeSearch))
}

func TestInflightJobsRemovedForInactiveUser(t *testing.T) {
	f, reg := newTestFrontend(t, 10, 1)
	done := startRoundTrip(f, tenantRequest(context.Background()))
	require.Eventually(t, func() bool { return gaugeValue(reg, inflightJobsName, pipeline.QueryTypeSearch) == 1 }, time.Second, time.Millisecond)
	dequeueForQuerier(t, f).response <- &http.Response{StatusCode: http.StatusOK}
	require.NoError(t, (<-done).err)

	f.cleanupInactiveUserMetrics(testTenant)
	require.Equal(t, -1.0, gaugeValue(reg, inflightJobsName, pipeline.QueryTypeSearch))
}

func TestInflightJobsCleanupDuringJobDoesNotGoNegative(t *testing.T) {
	f, reg := newTestFrontend(t, 10, 1)
	done := startRoundTrip(f, tenantRequest(context.Background()))
	require.Eventually(t, func() bool { return gaugeValue(reg, inflightJobsName, pipeline.QueryTypeSearch) == 1 }, time.Second, time.Millisecond)

	f.cleanupInactiveUserMetrics(testTenant)
	dequeueForQuerier(t, f).response <- &http.Response{StatusCode: http.StatusOK}
	require.NoError(t, (<-done).err)
	require.Equal(t, -1.0, gaugeValue(reg, inflightJobsName, pipeline.QueryTypeSearch))
}

const (
	testTenant = "test-tenant"

	inflightJobsName  = "tempo_query_frontend_inflight_jobs"
	executingJobsName = "tempo_query_frontend_executing_jobs"
)

func newTestFrontend(t *testing.T, maxOutstanding, maxBatchSize int) (*Frontend, *prometheus.Registry) {
	reg := prometheus.NewRegistry()
	f, err := New(Config{MaxOutstandingPerTenant: maxOutstanding, MaxBatchSize: maxBatchSize}, log.NewNopLogger(), reg)
	require.NoError(t, err)
	return f, reg
}

func tenantRequest(ctx context.Context) pipeline.Request {
	httpReq := httptest.NewRequest("GET", "http://example.com", nil)
	req := pipeline.NewHTTPRequest(httpReq.WithContext(user.InjectOrgID(ctx, testTenant)))
	req.SetQueryShape(pipeline.QueryShape{Type: pipeline.QueryTypeSearch})
	return req
}

type roundTripResult struct {
	resp *http.Response
	err  error
}

func startRoundTrip(f *Frontend, req pipeline.Request) <-chan roundTripResult {
	done := make(chan roundTripResult, 1)
	go func() {
		resp, err := f.RoundTrip(req)
		done <- roundTripResult{resp: resp, err: err}
	}()
	return done
}

// dequeueForQuerier pulls the next queued job the way a querier worker does.
func dequeueForQuerier(t *testing.T, f *Frontend) *request {
	t.Helper()
	reqs, _, err := f.requestQueue.GetNextRequestForQuerier(context.Background(), queue.FirstUser(), make([]queue.Request, 1))
	require.NoError(t, err)
	return reqs[0].(*request)
}

// gaugeValue returns the value of the {user=testTenant, op} series of a gauge, or -1 when the series is absent.
func gaugeValue(reg *prometheus.Registry, name, op string) float64 {
	mfs, err := reg.Gather()
	if err != nil {
		return -1
	}
	for _, mf := range mfs {
		if mf.GetName() != name {
			continue
		}
		for _, m := range mf.GetMetric() {
			labels := map[string]string{}
			for _, lp := range m.GetLabel() {
				labels[lp.GetName()] = lp.GetValue()
			}
			if labels["user"] == testTenant && labels["op"] == op {
				return m.GetGauge().GetValue()
			}
		}
	}
	return -1
}

// TestQueryOpSurvivesSharding asserts the shape stamped on the parent request is
// carried onto the sharded subrequests that are actually enqueued.
func TestQueryOpSurvivesSharding(t *testing.T) {
	parent := pipeline.NewHTTPRequest(httptest.NewRequest("GET", "/", nil))
	parent.SetQueryShape(pipeline.QueryShape{Type: pipeline.QueryTypeSearch})

	sub := parent.CloneFromHTTPRequest(httptest.NewRequest("GET", "/shard", nil))

	require.Equal(t, "search", queryOp(sub))
}

// enqueueJob enqueues a job of the given op without waiting for the response.
func enqueueJob(ctx context.Context, t *testing.T, f *Frontend, op string) {
	t.Helper()

	req := tenantRequest(ctx)
	req.SetQueryShape(pipeline.QueryShape{Type: op})

	r := &request{
		request:   req,
		queueSpan: trace.SpanFromContext(context.Background()),
		err:       make(chan error, 1),
		response:  make(chan *http.Response, 1),
	}
	require.NoError(t, f.requestQueue.EnqueueRequest(testTenant, r))
}

func startProcess(f *Frontend, srv frontendv1pb.Frontend_ProcessServer) <-chan error {
	done := make(chan error, 1)
	go func() { done <- f.Process(srv) }()
	return done
}

func waitForBatch(t *testing.T, srv *mockProcessServer, size int) {
	t.Helper()
	select {
	case got := <-srv.sent:
		require.Equal(t, size, got)
	case <-time.After(5 * time.Second):
		require.Fail(t, "batch was not sent to the querier")
	}
}

func TestExecutingJobsCountsJobsOfBatchWhileExecuting(t *testing.T) {
	f, reg := newTestFrontend(t, 100, 3)
	enqueueJob(context.Background(), t, f, pipeline.QueryTypeSearch)
	enqueueJob(context.Background(), t, f, pipeline.QueryTypeSearch)
	enqueueJob(context.Background(), t, f, pipeline.QueryTypeTraces)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	srv := newHeldProcessServer(ctx)
	done := startProcess(f, srv)

	waitForBatch(t, srv, 3)
	require.Equal(t, 2.0, gaugeValue(reg, executingJobsName, pipeline.QueryTypeSearch))
	require.Equal(t, 1.0, gaugeValue(reg, executingJobsName, pipeline.QueryTypeTraces))

	require.Equal(t, -1.0, gaugeValue(reg, executingJobsName, pipeline.QueryTypeMetrics), "no job of this op was sent")

	close(srv.release)
	require.Eventually(t, func() bool {
		return gaugeValue(reg, executingJobsName, pipeline.QueryTypeSearch) == 0 && gaugeValue(reg, executingJobsName, pipeline.QueryTypeTraces) == 0
	}, 5*time.Second, time.Millisecond)

	cancel()
	require.ErrorIs(t, <-done, context.Canceled)
}

func TestExecutingJobsDoesNotCountQueuedJobs(t *testing.T) {
	f, reg := newTestFrontend(t, 100, 1)
	enqueueJob(context.Background(), t, f, pipeline.QueryTypeSearch)
	enqueueJob(context.Background(), t, f, pipeline.QueryTypeSearch)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	srv := newHeldProcessServer(ctx)
	done := startProcess(f, srv)

	// the querier works on one job at a time, the second one is still queued
	waitForBatch(t, srv, 1)
	require.Equal(t, 1.0, gaugeValue(reg, executingJobsName, pipeline.QueryTypeSearch))

	srv.release <- struct{}{}
	waitForBatch(t, srv, 1)
	require.Equal(t, 1.0, gaugeValue(reg, executingJobsName, pipeline.QueryTypeSearch))

	close(srv.release)
	require.Eventually(t, func() bool { return gaugeValue(reg, executingJobsName, pipeline.QueryTypeSearch) == 0 }, 5*time.Second, time.Millisecond)

	cancel()
	require.ErrorIs(t, <-done, context.Canceled)
}

func TestExecutingJobsReleased(t *testing.T) {
	sendErr := errors.New("querier stream closed")

	for _, tc := range []struct {
		name    string
		sendErr error
		// ends the job while the querier works on it, nil if sending the batch fails
		trigger func(cancelJob, cancelStream context.CancelFunc)
		wantErr error
	}{
		{name: "send error", sendErr: sendErr, wantErr: sendErr},
		{name: "stream error", trigger: func(_, cancelStream context.CancelFunc) { cancelStream() }, wantErr: context.Canceled},
		{name: "job canceled", trigger: func(cancelJob, _ context.CancelFunc) { cancelJob() }, wantErr: context.Canceled},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f, reg := newTestFrontend(t, 100, 1)
			jobCtx, cancelJob := context.WithCancel(context.Background())
			defer cancelJob()
			enqueueJob(jobCtx, t, f, pipeline.QueryTypeSearch)

			streamCtx, cancelStream := context.WithCancel(context.Background())
			defer cancelStream()
			srv := newHeldProcessServer(streamCtx)
			srv.sendErr = tc.sendErr
			done := startProcess(f, srv)

			if tc.trigger != nil {
				waitForBatch(t, srv, 1)
				require.Equal(t, 1.0, gaugeValue(reg, executingJobsName, pipeline.QueryTypeSearch))
				tc.trigger(cancelJob, cancelStream)
			}

			require.ErrorIs(t, <-done, tc.wantErr)
			// the series exists, so the job was counted, and it is back to 0
			require.Equal(t, 0.0, gaugeValue(reg, executingJobsName, pipeline.QueryTypeSearch))
		})
	}
}

func TestExecutingJobsExcludesExpiredJobs(t *testing.T) {
	f, reg := newTestFrontend(t, 100, 2)
	expiredCtx, cancelExpired := context.WithCancel(context.Background())
	enqueueJob(expiredCtx, t, f, pipeline.QueryTypeSearch)
	enqueueJob(context.Background(), t, f, pipeline.QueryTypeTraces)
	cancelExpired()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	srv := newHeldProcessServer(ctx)
	done := startProcess(f, srv)

	// the expired job is dropped from the batch and never counted
	waitForBatch(t, srv, 1)
	require.Equal(t, -1.0, gaugeValue(reg, executingJobsName, pipeline.QueryTypeSearch))
	require.Equal(t, 1.0, gaugeValue(reg, executingJobsName, pipeline.QueryTypeTraces))

	close(srv.release)
	require.Eventually(t, func() bool { return gaugeValue(reg, executingJobsName, pipeline.QueryTypeTraces) == 0 }, 5*time.Second, time.Millisecond)

	cancel()
	require.ErrorIs(t, <-done, context.Canceled)
}

func TestExecutingJobsCleanupDuringJobDoesNotRecreateSeries(t *testing.T) {
	f, reg := newTestFrontend(t, 100, 1)
	enqueueJob(context.Background(), t, f, pipeline.QueryTypeSearch)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	srv := newHeldProcessServer(ctx)
	done := startProcess(f, srv)

	waitForBatch(t, srv, 1)
	require.Equal(t, 1.0, gaugeValue(reg, executingJobsName, pipeline.QueryTypeSearch))

	f.cleanupInactiveUserMetrics(testTenant)
	require.Zero(t, testutil.CollectAndCount(f.executingJobs))

	// the job ends after the series was removed, which must neither recreate it nor make it negative.
	// the next job is held, so the frontend is done with the first one when its batch is sent.
	srv.release <- struct{}{}
	enqueueJob(context.Background(), t, f, pipeline.QueryTypeTraces)
	waitForBatch(t, srv, 1)
	require.Equal(t, 1.0, gaugeValue(reg, executingJobsName, pipeline.QueryTypeTraces))
	require.Equal(t, 1, testutil.CollectAndCount(f.executingJobs), "only the series of the running job exists")

	cancel()
	require.ErrorIs(t, <-done, context.Canceled)
}
