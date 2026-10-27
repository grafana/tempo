package hedgedmetrics

import (
	"context"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/cristalhq/hedgedhttp"
	"github.com/grafana/tempo/v3/pkg/util/test"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

func TestHedgedCounter(t *testing.T) {
	type stats struct{ actual, requested uint64 }
	tests := []struct {
		name  string
		stats []stats
		want  []float64
	}{
		{
			name:  "no requests",
			stats: []stats{{0, 0}},
			want:  []float64{0},
		},
		{
			name:  "increasing",
			stats: []stats{{5, 5}, {15, 10}, {28, 20}, {38, 25}, {38, 25}},
			want:  []float64{0, 5, 8, 13, 13},
		},
		{
			name:  "requested ahead of actual",
			stats: []stats{{0, 1}, {9, 10}, {12, 10}, {12, 13}, {15, 13}},
			want:  []float64{0, 0, 2, 2, 2},
		},
		{
			name:  "hedged count dips while sub-requests are in flight",
			stats: []stats{{110, 100}, {108, 100}, {111, 101}, {125, 110}},
			want:  []float64{10, 10, 10, 15},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctr := prometheus.NewCounter(prometheus.CounterOpts{Name: test.RandomString()})
			hc := &hedgedCounter{counter: ctr}
			mock := &MockStatsProvider{}

			for i, s := range tt.stats {
				mock.SetStats(s.actual, s.requested)
				hc.update(mock)
				require.Equal(t, tt.want[i], ctrVal(t, ctr), "step %d", i)
			}
		})
	}
}

type okRoundTripper struct{}

func (okRoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
}

// TestHedgedCounterConcurrentRequests reads real hedgedhttp stats while requests are in flight. No
// request is ever hedged, so the counter must stay at zero.
func TestHedgedCounterConcurrentRequests(t *testing.T) {
	rt, stats, err := hedgedhttp.NewRoundTripperAndStats(time.Hour, 2, okRoundTripper{})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	var wg sync.WaitGroup
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for ctx.Err() == nil {
				req, err := http.NewRequest(http.MethodGet, "http://localhost", nil)
				if err != nil {
					t.Error(err)
					return
				}
				resp, err := rt.RoundTrip(req)
				if err != nil {
					t.Error(err)
					return
				}
				if err := resp.Body.Close(); err != nil {
					t.Error(err)
					return
				}
			}
		}()
	}

	ctr := prometheus.NewCounter(prometheus.CounterOpts{Name: test.RandomString()})
	hc := &hedgedCounter{counter: ctr}
	for ctx.Err() == nil {
		hc.update(stats)
	}
	wg.Wait()

	require.Positive(t, stats.RequestedRoundTrips())
	require.Equal(t, 0.0, ctrVal(t, ctr))
}

// MockStatsProvider is StatsProvider for testing
type MockStatsProvider struct {
	mu                  sync.Mutex
	actualRoundTrips    uint64
	requestedRoundTrips uint64
}

func (m *MockStatsProvider) ActualRoundTrips() uint64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.actualRoundTrips
}

func (m *MockStatsProvider) RequestedRoundTrips() uint64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.requestedRoundTrips
}

func (m *MockStatsProvider) SetStats(actual, requested uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.actualRoundTrips = actual
	m.requestedRoundTrips = requested
}

func TestPublish(t *testing.T) {
	ctr := prometheus.NewCounter(prometheus.CounterOpts{Name: test.RandomString()})
	stats := &MockStatsProvider{}

	publishWithDuration(stats, ctr, 10*time.Millisecond)

	require.Equal(t, 0.0, ctrVal(t, ctr))

	// Set initial stats values
	stats.SetStats(5, 5)
	time.Sleep(30 * time.Millisecond)
	require.Equal(t, 0.0, ctrVal(t, ctr))

	stats.SetStats(15, 10)
	time.Sleep(30 * time.Millisecond)
	require.Equal(t, 5.0, ctrVal(t, ctr))

	stats.SetStats(28, 20)
	time.Sleep(30 * time.Millisecond)
	require.Equal(t, 8.0, ctrVal(t, ctr))

	stats.SetStats(38, 25)
	time.Sleep(30 * time.Millisecond)
	require.Equal(t, 13.0, ctrVal(t, ctr))

	time.Sleep(30 * time.Millisecond)

	// counter doesn't increase if stats stay same
	time.Sleep(30 * time.Millisecond)
	require.Equal(t, 13.0, ctrVal(t, ctr))
}

// hedgedhttp increments RequestedRoundTrips before dispatching a request and ActualRoundTrips only once the
// sub-request goroutine starts, so a snapshot taken under load can see more requested than actual round trips.
func TestPublishInFlightRequests(t *testing.T) {
	ctr := prometheus.NewCounter(prometheus.CounterOpts{Name: test.RandomString()})
	stats := &MockStatsProvider{}

	publishWithDuration(stats, ctr, 10*time.Millisecond)

	stats.SetStats(15, 10)
	time.Sleep(30 * time.Millisecond)
	require.Equal(t, 5.0, ctrVal(t, ctr))

	// a request was requested but its first sub-request has not started yet
	stats.SetStats(15, 16)
	time.Sleep(30 * time.Millisecond)
	require.Equal(t, 5.0, ctrVal(t, ctr))

	// some hedged sub-requests have not started yet
	stats.SetStats(20, 17)
	time.Sleep(30 * time.Millisecond)
	require.Equal(t, 5.0, ctrVal(t, ctr))

	stats.SetStats(25, 17)
	time.Sleep(30 * time.Millisecond)
	require.Equal(t, 8.0, ctrVal(t, ctr))
}

func ctrVal(t *testing.T, ctr prometheus.Counter) float64 {
	t.Helper()

	val, err := test.GetCounterValue(ctr)
	require.NoError(t, err)

	return val
}
