package v1

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/go-kit/log"
	"github.com/grafana/dskit/httpgrpc"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"

	"github.com/grafana/tempo/modules/frontend/pipeline"
	"github.com/grafana/tempo/modules/frontend/v1/frontendv1pb"
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
