package queue

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/grafana/dskit/services"
	"github.com/grafana/tempo/v3/pkg/util/test"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/atomic"
)

const messages = 50_000

type mockRequest struct {
	weight int
	failed chan error
}

func (r *mockRequest) Invalid() bool { return false }
func (r *mockRequest) Weight() int {
	if r.weight > 0 {
		return r.weight
	}
	return 1
}

func (r *mockRequest) Fail(err error) {
	if r.failed != nil {
		r.failed <- err
	}
}

func TestGetNextForQuerierOneUser(t *testing.T) {
	t.Parallel()
	messages := 10

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stop := make(chan struct{})
	requestsPulled := atomic.NewInt32(0)

	q, start := queueWithListeners(ctx, 100, 1, func(_ []Request) {
		i := requestsPulled.Inc()
		if i == int32(messages) {
			close(stop)
		}
	})
	close(start)

	for j := 0; j < messages; j++ {
		err := q.EnqueueRequest("test", &mockRequest{})
		require.NoError(t, err)
	}

	<-stop

	require.Equal(t, int32(messages), requestsPulled.Load())

	err := q.stopping(nil)
	require.NoError(t, err)
}

func TestGetNextForQuerierRandomUsers(t *testing.T) {
	t.Parallel()
	messages := 100

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stop := make(chan struct{})
	requestsPulled := atomic.NewInt32(0)

	q, start := queueWithListeners(ctx, 100, 1, func(_ []Request) {
		if requestsPulled.Inc() == int32(messages) {
			close(stop)
		}
	})
	close(start)

	for j := 0; j < messages; j++ {
		err := q.EnqueueRequest(test.RandomString(), &mockRequest{})
		require.NoError(t, err)
	}

	<-stop

	require.Equal(t, int32(messages), requestsPulled.Load())

	err := q.stopping(nil)
	require.NoError(t, err)
}

func TestGetNextBatches(t *testing.T) {
	t.Parallel()
	messages := 10

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stop := make(chan struct{})
	requestsPulled := atomic.NewInt32(0)

	q, start := queueWithListeners(ctx, 100, 3, func(r []Request) {
		if requestsPulled.Add(int32(len(r))) == int32(messages) {
			close(stop)
		}
	})
	close(start)

	for j := 0; j < messages; j++ {
		err := q.EnqueueRequest("user", &mockRequest{})
		require.NoError(t, err)
	}

	<-stop

	require.Equal(t, int32(messages), requestsPulled.Load())

	err := q.stopping(nil)
	require.NoError(t, err)
}

func BenchmarkGetNextForQuerier100(b *testing.B) {
	benchmarkGetNextForQuerier(b, 100, messages)
}

func BenchmarkGetNextForQuerier1000(b *testing.B) {
	benchmarkGetNextForQuerier(b, 1000, messages)
}

func BenchmarkGetNextForQuerier5000(b *testing.B) {
	benchmarkGetNextForQuerier(b, 5000, messages)
}

func benchmarkGetNextForQuerier(b *testing.B, listeners int, messages int) {
	const user = "user"

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stop := make(chan struct{})
	requestsPulled := atomic.NewInt32(0)

	q, start := queueWithListeners(ctx, listeners, 1, func(_ []Request) {
		if requestsPulled.Inc() == int32(messages) {
			stop <- struct{}{}
		}
	})
	close(start)

	req := &mockRequest{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < messages; j++ {
			err := q.EnqueueRequest(user, req)
			if err != nil {
				panic(err)
			}
		}

		<-stop
		requestsPulled.Sub(int32(messages))
	}

	err := q.stopping(nil)
	if err != nil {
		panic(err)
	}
}

func newTestRequestQueue() *RequestQueue {
	g := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "test_len",
	}, []string{"user"})
	c := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "test_discarded",
	}, []string{"user"})
	b := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: "test_batch_weight",
	}, []string{"user"})

	return NewRequestQueue(100_000, g, b, c)
}

func queueWithListeners(ctx context.Context, listeners int, batchSize int, listenerFn func(r []Request)) (*RequestQueue, chan struct{}) {
	q := newTestRequestQueue()
	start := make(chan struct{})

	for i := 0; i < listeners; i++ {
		go func() {
			var r []Request
			var err error
			var last UserIndex

			<-start

			batchBuffer := make([]Request, batchSize)
			for {
				r, last, err = q.GetNextRequestForQuerier(ctx, last, batchBuffer)
				if err != nil {
					return
				}
				if listenerFn != nil {
					listenerFn(r)
				}
			}
		}()
	}

	err := services.StartAndAwaitRunning(context.Background(), q)
	if err != nil {
		panic(err)
	}

	return q, start
}

func TestContextCond(t *testing.T) {
	t.Run("wait until broadcast", func(t *testing.T) {
		t.Parallel()
		mtx := &sync.Mutex{}
		cond := contextCond{Cond: sync.NewCond(mtx)}

		doneWaiting := make(chan struct{})

		mtx.Lock()
		go func() {
			cond.Wait(context.Background())
			mtx.Unlock()
			close(doneWaiting)
		}()

		assertChanNotReceived(t, doneWaiting, 100*time.Millisecond, "cond.Wait returned, but it should not because we did not broadcast yet")

		cond.Broadcast()
		assertChanReceived(t, doneWaiting, 250*time.Millisecond, "cond.Wait did not return after broadcast")
	})

	t.Run("wait until context deadline", func(t *testing.T) {
		t.Parallel()
		mtx := &sync.Mutex{}
		cond := contextCond{Cond: sync.NewCond(mtx)}
		doneWaiting := make(chan struct{})

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		mtx.Lock()
		go func() {
			cond.Wait(ctx)
			mtx.Unlock()
			close(doneWaiting)
		}()

		assertChanNotReceived(t, doneWaiting, 100*time.Millisecond, "cond.Wait returned, but it should not because we did not broadcast yet and didn't cancel the context")

		cancel()
		assertChanReceived(t, doneWaiting, 250*time.Millisecond, "cond.Wait did not return after cancelling the context")
	})

	t.Run("wait on already canceled context", func(t *testing.T) {
		// This test represents the racy real world scenario,
		// we don't know whether it's going to wait before the broadcast triggered by the context cancellation.
		t.Parallel()
		mtx := &sync.Mutex{}
		cond := contextCond{Cond: sync.NewCond(mtx)}
		doneWaiting := make(chan struct{})

		alreadyCanceledContext, cancel := context.WithCancel(context.Background())
		cancel()

		mtx.Lock()
		go func() {
			cond.Wait(alreadyCanceledContext)
			mtx.Unlock()
			close(doneWaiting)
		}()

		assertChanReceived(t, doneWaiting, 250*time.Millisecond, "cond.Wait did not return after cancelling the context")
	})

	t.Run("wait on already canceled context, but it takes a while to wait", func(t *testing.T) {
		t.Parallel()
		mtx := &sync.Mutex{}
		cond := contextCond{
			Cond: sync.NewCond(mtx),
			testHookBeforeWaiting: func() {
				// This makes the waiting goroutine so slow that out Wait(ctx) will need to broadcast once it sees it waiting.
				time.Sleep(250 * time.Millisecond)
			},
		}
		doneWaiting := make(chan struct{})

		alreadyCanceledContext, cancel := context.WithCancel(context.Background())
		cancel()

		mtx.Lock()
		go func() {
			cond.Wait(alreadyCanceledContext)
			mtx.Unlock()
			close(doneWaiting)
		}()

		assertChanReceived(t, doneWaiting, time.Second, "cond.Wait did not return after 500ms")
	})

	t.Run("lots of goroutines waiting at the same time, none of them misses it's broadcast from cancel", func(t *testing.T) {
		t.Parallel()
		mtx := &sync.Mutex{}
		cond := contextCond{
			Cond: sync.NewCond(mtx),
			testHookBeforeWaiting: func() {
				// Wait just a little bit to create every goroutine
				time.Sleep(time.Millisecond)
			},
		}
		const goroutines = 100

		doneWaiting := make(chan struct{}, goroutines)
		release := make(chan struct{})

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		for i := 0; i < goroutines; i++ {
			go func() {
				<-release

				mtx.Lock()
				cond.Wait(ctx)
				mtx.Unlock()

				doneWaiting <- struct{}{}
			}()
		}
		go func() {
			<-release
			cancel()
		}()

		close(release)

		assert.Eventually(t, func() bool {
			return len(doneWaiting) == goroutines
		}, time.Second, 10*time.Millisecond)
	})
}

func TestGetBatchBuffer(t *testing.T) {
	tests := []struct {
		name           string
		queueContents  []Request
		requestedCount int
		expectedCount  int
	}{
		{
			name:           "exactly requested count",
			queueContents:  []Request{&mockRequest{}, &mockRequest{}, &mockRequest{}},
			requestedCount: 3,
			expectedCount:  3,
		},
		{
			name:           "less than requested count",
			queueContents:  []Request{&mockRequest{}, &mockRequest{}},
			requestedCount: 3,
			expectedCount:  2,
		},
		{
			name:           "more than requested count",
			queueContents:  []Request{&mockRequest{}, &mockRequest{}, &mockRequest{}, &mockRequest{}},
			requestedCount: 3,
			expectedCount:  3,
		},
		{
			name:           "less than requested count due to biggest weight",
			queueContents:  []Request{&mockRequest{weight: 10}},
			requestedCount: 3,
			expectedCount:  1,
		},
		{
			name:           "empty queue",
			queueContents:  []Request{},
			requestedCount: 3,
			expectedCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			queue := make(chan Request, len(tt.queueContents))
			for _, req := range tt.queueContents {
				queue <- req
			}

			q := &RequestQueue{
				queueLength: prometheus.NewGaugeVec(prometheus.GaugeOpts{
					Name: "test_len",
				}, []string{"user"}),

				batchWeight: prometheus.NewHistogramVec(prometheus.HistogramOpts{
					Name: "test_weight",
				}, []string{"user"}),
			}

			batchBuffer := make([]Request, tt.requestedCount)
			result := q.getBatchBuffer(batchBuffer, "user", queue)

			assert.Equal(t, tt.expectedCount, len(result))
		})
	}
}

func TestStopWithEmptyQueues(t *testing.T) {
	t.Parallel()

	q := newTestRequestQueue()
	require.NoError(t, services.StartAndAwaitRunning(t.Context(), q))

	// a tenant queue outlives the request that created it, so any frontend that served
	// traffic within the last cleanup period is in this state when SIGTERM arrives
	require.NoError(t, q.EnqueueRequest("test", &mockRequest{}))
	batch, _, err := q.GetNextRequestForQuerier(t.Context(), FirstUser(), make([]Request, 1))
	require.NoError(t, err)
	require.Len(t, batch, 1)
	require.Equal(t, 1, tenantQueueCount(q))

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	require.NoError(t, services.StopAndAwaitTerminated(ctx, q))
}

func TestStopWaitsForDispatch(t *testing.T) {
	t.Parallel()

	q := newTestRequestQueue()
	require.NoError(t, services.StartAndAwaitRunning(t.Context(), q))
	require.NoError(t, q.EnqueueRequest("test", &mockRequest{}))

	// the querier only shows up after shutdown has started, so terminating without
	// waiting for it would drop the request
	dequeued := make(chan int, 1)
	go func() {
		time.Sleep(200 * time.Millisecond)
		batch, _, err := q.GetNextRequestForQuerier(t.Context(), FirstUser(), make([]Request, 1))
		if err != nil {
			dequeued <- -1
			return
		}
		dequeued <- len(batch)
	}()

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	start := time.Now()
	require.NoError(t, services.StopAndAwaitTerminated(ctx, q))
	require.Equal(t, 1, <-dequeued)
	require.Less(t, time.Since(start), 2*time.Second, "drain ended on its deadline instead of on dispatch")
}

func TestStopDrainsFullQueue(t *testing.T) {
	t.Parallel()

	const queued = 200

	ctx := t.Context()

	dispatched := atomic.NewInt32(0)
	// listeners stay paused until start closes, so the queue is full when shutdown begins
	q, start := queueWithListeners(ctx, 10, 1, func(r []Request) {
		dispatched.Add(int32(len(r)))
	})

	for range queued {
		require.NoError(t, q.EnqueueRequest("test", &mockRequest{}))
	}

	close(start)
	q.StopAsync()

	awaitCtx, awaitCancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer awaitCancel()
	require.NoError(t, q.AwaitTerminated(awaitCtx))

	require.ErrorIs(t, q.EnqueueRequest("test", &mockRequest{}), ErrStopped)

	// the listener counts after its dequeue returns, so it can trail termination slightly
	require.Eventually(t, func() bool {
		return dispatched.Load() == int32(queued)
	}, 5*time.Second, 10*time.Millisecond, "queued work was dropped instead of drained")
}

func TestStopRefusesNewWork(t *testing.T) {
	t.Parallel()

	q := newTestRequestQueue()
	q.drainTimeout = 5 * time.Second
	require.NoError(t, services.StartAndAwaitRunning(t.Context(), q))

	// no querier takes this, so the drain is still running while we probe below
	require.NoError(t, q.EnqueueRequest("test", &mockRequest{}))

	q.StopAsync()

	// EnqueueRequest must reject now, or new work keeps arriving and the drain never ends
	require.Eventually(t, func() bool {
		return errors.Is(q.EnqueueRequest("test", &mockRequest{}), ErrStopped)
	}, 2*time.Second, 10*time.Millisecond)
}

func TestStopFailsUndispatchedWork(t *testing.T) {
	t.Parallel()

	q := newTestRequestQueue()
	q.drainTimeout = 100 * time.Millisecond
	require.NoError(t, services.StartAndAwaitRunning(t.Context(), q))

	req := &mockRequest{failed: make(chan error, 1)}
	require.NoError(t, q.EnqueueRequest("test", req))

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	require.NoError(t, services.StopAndAwaitTerminated(ctx, q))

	// an abandoned request must be told, or its caller blocks until its own query timeout
	// and holds the http graceful drain open for that long
	select {
	case err := <-req.failed:
		require.ErrorIs(t, err, ErrStopped)
	default:
		t.Fatal("request was left in the queue without being failed")
	}
}

func tenantQueueCount(q *RequestQueue) int {
	q.mtx.RLock()
	defer q.mtx.RUnlock()

	return q.queues.len()
}

func assertChanReceived(t *testing.T, c chan struct{}, timeout time.Duration, msg string) {
	t.Helper()

	select {
	case <-c:
	case <-time.After(timeout):
		t.Fatal(msg)
	}
}

func assertChanNotReceived(t *testing.T, c chan struct{}, wait time.Duration, msg string, args ...interface{}) {
	t.Helper()

	select {
	case <-c:
		t.Fatalf(msg, args...)
	case <-time.After(wait):
		// OK!
	}
}
