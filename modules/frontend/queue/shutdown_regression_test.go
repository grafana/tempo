package queue

import (
	"context"
	"testing"
	"time"

	"github.com/grafana/dskit/services"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

func TestShutdownDrainsWithoutCleanupTimer(t *testing.T) {
	for _, before := range []bool{true, false} {
		name := "drain_during_stop"
		if before {
			name = "already_drained"
		}
		t.Run(name, func(t *testing.T) {
			q := NewRequestQueue(10,
				prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "length"}, []string{"user"}),
				prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: "weight"}, []string{"user"}),
				prometheus.NewCounterVec(prometheus.CounterOpts{Name: "discarded"}, []string{"user"}))
			q.Service = services.NewTimerService(time.Hour, nil, q.cleanupQueues, q.stopping)
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			require.NoError(t, services.StartAndAwaitRunning(ctx, q))
			require.NoError(t, q.EnqueueRequest("tenant", &mockRequest{}))
			drain := func() {
				got, _, err := q.GetNextRequestForQuerier(ctx, FirstUser(), make([]Request, 1))
				require.NoError(t, err)
				require.Len(t, got, 1)
			}
			if before {
				drain()
			}
			waiting := make(chan struct{}, 1)
			q.cond.testHookBeforeWaiting = func() {
				select {
				case waiting <- struct{}{}:
				default:
				}
			}
			t.Cleanup(func() {
				q.mtx.Lock()
				q.queues.deleteQueue("tenant")
				q.cond.Broadcast()
				q.mtx.Unlock()
			})
			q.StopAsync()
			if !before {
				select {
				case <-waiting:
				case <-ctx.Done():
					t.Fatal("shutdown did not wait for queued work")
				}
				drain()
			}
			require.NoError(t, q.AwaitTerminated(ctx))
		})
	}
}
