package queue

import (
	"context"
	"testing"

	"github.com/grafana/tempo/pkg/util/test"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"go.uber.org/atomic"
)

const messages = 50_000

type mockRequest struct{}

func (r *mockRequest) Invalid() bool { return false }

func TestGetNextForQuerierOneUser(t *testing.T) {
	messages := 100

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stop := make(chan struct{})
	requestsPulled := atomic.NewInt32(0)

	q := queueWithListeners(100, ctx, func(r Request) {
		if requestsPulled.Inc() == int32(messages) {
			close(stop)
		}
	})

	for j := 0; j < messages; j++ {
		err := q.EnqueueRequest("test", &mockRequest{}, 0, nil)
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

	q := queueWithListeners(100, ctx, func(r Request) {
		if requestsPulled.Inc() == int32(messages) {
			close(stop)
		}
	})

	for j := 0; j < messages; j++ {
		err := q.EnqueueRequest(test.RandomString(), &mockRequest{}, 0, nil)
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

	q := queueWithListeners(listeners, ctx, func(r Request) {
		if requestsPulled.Inc() == int32(messages) {
			stop <- struct{}{}
		}
	})

	req := &mockRequest{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < messages; j++ {
			err := q.EnqueueRequest(user, req, 0, nil)
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

func queueWithListeners(listeners int, ctx context.Context, listenerFn func(r Request)) *RequestQueue {
	g := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "test_len",
	}, []string{"user"})
	c := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "test_discarded",
	}, []string{"user"})

	q := NewRequestQueue(100_000, g, c)

	for i := 0; i < listeners; i++ {
		go func() {
			for {
				r, err := q.GetNextRequestForQuerier(ctx)
				if listenerFn != nil {
					listenerFn(r)
				}
				if err != nil {
					return
				}
			}
		}()
	}

	return q
}
