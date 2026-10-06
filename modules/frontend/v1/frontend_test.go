package v1

import (
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
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"

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

	mu            sync.Mutex
	recvCount     int
	lastBatchSize int
	served        int
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
	}

	return nil
}

func (m *mockProcessServer) Recv() (*frontendv1pb.ClientToFrontend, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.recvCount++
	if m.recvCount == 1 {
		// Handshake: advertise batching support so Process uses MaxBatchSize.
		return &frontendv1pb.ClientToFrontend{
			ClientID: "test-querier",
			Features: int32(frontendv1pb.Feature_REQUEST_BATCHING),
		}, nil
	}

	batch := make([]*httpgrpc.HTTPResponse, m.lastBatchSize)
	for i := range batch {
		batch[i] = &httpgrpc.HTTPResponse{Code: http.StatusOK}
	}

	m.served += m.lastBatchSize
	if m.served >= m.totalRequests {
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
	f, reg := newInflightTestFrontend(t, 10)
	done := startRoundTrip(f, tenantRequest(context.Background()))

	require.Eventually(t, func() bool { return inflightJobs(reg) == 1 }, time.Second, time.Millisecond, "queued job not counted")

	r := dequeueForQuerier(t, f)
	require.Equal(t, 1.0, inflightJobs(reg), "executing job not counted")

	r.response <- &http.Response{StatusCode: http.StatusOK}
	require.NoError(t, (<-done).err)
	require.Equal(t, 0.0, inflightJobs(reg))
}

func TestInflightJobsReleasedOnError(t *testing.T) {
	f, reg := newInflightTestFrontend(t, 10)
	done := startRoundTrip(f, tenantRequest(context.Background()))
	require.Eventually(t, func() bool { return inflightJobs(reg) == 1 }, time.Second, time.Millisecond)

	dequeueForQuerier(t, f).Fail(errors.New("querier stream closed"))
	require.Error(t, (<-done).err)
	require.Equal(t, 0.0, inflightJobs(reg))
}

func TestInflightJobsReleasedOnCancel(t *testing.T) {
	f, reg := newInflightTestFrontend(t, 10)
	ctx, cancel := context.WithCancel(context.Background())
	done := startRoundTrip(f, tenantRequest(ctx))
	require.Eventually(t, func() bool { return inflightJobs(reg) == 1 }, time.Second, time.Millisecond)

	cancel()
	require.ErrorIs(t, (<-done).err, context.Canceled)
	require.Equal(t, 0.0, inflightJobs(reg))
}

func TestInflightJobsExcludesRejected(t *testing.T) {
	f, reg := newInflightTestFrontend(t, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	startRoundTrip(f, tenantRequest(ctx)) // fills the tenant queue
	require.Eventually(t, func() bool { return inflightJobs(reg) == 1 }, time.Second, time.Millisecond)

	res := <-startRoundTrip(f, tenantRequest(context.Background()))
	require.ErrorIs(t, res.err, queue.ErrTooManyRequests)
	require.Equal(t, 1.0, inflightJobs(reg))
}

func TestInflightJobsRemovedForInactiveUser(t *testing.T) {
	f, reg := newInflightTestFrontend(t, 10)
	done := startRoundTrip(f, tenantRequest(context.Background()))
	require.Eventually(t, func() bool { return inflightJobs(reg) == 1 }, time.Second, time.Millisecond)
	dequeueForQuerier(t, f).response <- &http.Response{StatusCode: http.StatusOK}
	require.NoError(t, (<-done).err)

	f.cleanupInactiveUserMetrics(testTenant)
	require.Equal(t, -1.0, inflightJobs(reg))
}

func TestInflightJobsCleanupDuringJobDoesNotGoNegative(t *testing.T) {
	f, reg := newInflightTestFrontend(t, 10)
	done := startRoundTrip(f, tenantRequest(context.Background()))
	require.Eventually(t, func() bool { return inflightJobs(reg) == 1 }, time.Second, time.Millisecond)

	f.cleanupInactiveUserMetrics(testTenant)
	dequeueForQuerier(t, f).response <- &http.Response{StatusCode: http.StatusOK}
	require.NoError(t, (<-done).err)
	require.Equal(t, -1.0, inflightJobs(reg))
}

const testTenant = "test-tenant"

func newInflightTestFrontend(t *testing.T, maxOutstanding int) (*Frontend, *prometheus.Registry) {
	reg := prometheus.NewRegistry()
	f, err := New(Config{MaxOutstandingPerTenant: maxOutstanding, MaxBatchSize: 1}, log.NewNopLogger(), reg)
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

// inflightJobs returns the {user=testTenant, op="search"} gauge value, or -1 when the series is absent.
func inflightJobs(reg *prometheus.Registry) float64 {
	mfs, err := reg.Gather()
	if err != nil {
		return -1
	}
	for _, mf := range mfs {
		if mf.GetName() != "tempo_query_frontend_inflight_jobs" {
			continue
		}
		for _, m := range mf.GetMetric() {
			labels := map[string]string{}
			for _, lp := range m.GetLabel() {
				labels[lp.GetName()] = lp.GetValue()
			}
			if labels["user"] == testTenant && labels["op"] == pipeline.QueryTypeSearch {
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
