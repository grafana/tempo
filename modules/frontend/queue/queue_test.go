package queue

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/grafana/dskit/services"
	"github.com/grafana/tempo/v2/pkg/util/test"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/atomic"
)

const messages = 50_000

type mockRequest struct{}

func (r *mockRequest) Invalid() bool { return false }

func TestGetNextForQuerierOneUser(t *testing.T) {
	messages := 10

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stop := make(chan struct{})
	requestsPulled := atomic.NewInt32(0)

	q, start := queueWithListeners(ctx, 100, 1, func(r []Request) {
		i := requestsPulled.Inc()
		if i == int32(messages) {
			close(stop)
		}
	})
	close(start)

	for j := 0; j < messages; j++ {
		err := q.EnqueueRequest("test", &mockRequest{}, 0)
		require.NoError(t, err)
	}

	<-stop

	require.Equal(t, int32(messages), requestsPulled.Load())

	err := q.stopping(nil)
	require.NoError(t, err)
}

func TestGetNextForQuerierRandomUsers(t *testing.T) {
	messages := 100

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stop := make(chan struct{})
	requestsPulled := atomic.NewInt32(0)

	q, start := queueWithListeners(ctx, 100, 1, func(r []Request) {
		if requestsPulled.Inc() == int32(messages) {
			close(stop)
		}
	})
	close(start)

	for j := 0; j < messages; j++ {
		err := q.EnqueueRequest(test.RandomString(), &mockRequest{}, 0)
		require.NoError(t, err)
	}

	<-stop

	require.Equal(t, int32(messages), requestsPulled.Load())

	err := q.stopping(nil)
	require.NoError(t, err)
}

func TestGetNextBatches(t *testing.T) {
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
		err := q.EnqueueRequest("user", &mockRequest{}, 0)
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

	q, start := queueWithListeners(ctx, listeners, 1, func(r []Request) {
		if requestsPulled.Inc() == int32(messages) {
			stop <- struct{}{}
		}
	})
	close(start)

	req := &mockRequest{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < messages; j++ {
			err := q.EnqueueRequest(user, req, 0)
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

func queueWithListeners(ctx context.Context, listeners int, batchSize int, listenerFn func(r []Request)) (*RequestQueue, chan struct{}) {
	g := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "test_len",
	}, []string{"user"})
	c := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "test_discarded",
	}, []string{"user"})

	q := NewRequestQueue(100_000, 0, g, c)
	start := make(chan struct{})

	for i := 0; i < listeners; i++ {
		go func() {
			var r []Request
			var err error
			var last UserIndex

			<-start

			batchBuffer := make([]Request, batchSize)
			for {
				r, last, err = q.GetNextRequestForQuerier(ctx, last, "", batchBuffer)
				if err != nil {
					return
				}
				if listenerFn != nil {
					listenerFn(r)
				}
			}
		}()
	}

	return q, start
}

func TestRequestQueue_GetNextRequestForQuerier_ShouldGetRequestAfterReshardingBecauseQuerierHasBeenForgotten(t *testing.T) {
	const forgetDelay = 3 * time.Second

	queue := NewRequestQueue(1, forgetDelay,
		prometheus.NewGaugeVec(prometheus.GaugeOpts{}, []string{"user"}),
		prometheus.NewCounterVec(prometheus.CounterOpts{}, []string{"user"}))

	// Start the queue service.
	ctx := context.Background()
	require.NoError(t, services.StartAndAwaitRunning(ctx, queue))
	t.Cleanup(func() {
		require.NoError(t, services.StopAndAwaitTerminated(ctx, queue))
	})

	// Two queriers connect.
	queue.RegisterQuerierConnection("querier-1")
	queue.RegisterQuerierConnection("querier-2")

	// Querier-2 waits for a new request.
	querier2wg := sync.WaitGroup{}
	querier2wg.Add(1)
	go func() {
		defer querier2wg.Done()
		_, _, err := queue.GetNextRequestForQuerier(ctx, FirstUser(), "querier-2", make([]Request, 1))
		require.NoError(t, err)
	}()

	// Querier-1 crashes (no graceful shutdown notification).
	queue.UnregisterQuerierConnection("querier-1")

	// Enqueue a request from an user which would be assigned to querier-1.
	// NOTE: "user-1" hash falls in the querier-1 shard.
	require.NoError(t, queue.EnqueueRequest("user-1", "request", 1))

	startTime := time.Now()
	querier2wg.Wait()
	waitTime := time.Since(startTime)

	// We expect that querier-2 got the request only after querier-1 forget delay is passed.
	assert.GreaterOrEqual(t, waitTime.Milliseconds(), forgetDelay.Milliseconds())
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

func assertChanReceived(t *testing.T, c chan struct{}, timeout time.Duration, msg string) {
	t.Helper()

	select {
	case <-c:
	case <-time.After(timeout):
		t.Fatalf(msg)
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
