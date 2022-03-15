package distributor

import (
	"context"
	"time"

	"github.com/go-kit/log/level"
	"github.com/grafana/dskit/services"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/pkg/util/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// TODO: Move to a separate package

var (
	metricCircularQueueOverwrites = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "distributor_forwarder_queue_overwrites",
		Help:      "Total number of overwrites in the forwarder queue",
	}, []string{"tenant"})
)

type pushRingRequest struct {
	userID string
	keys   []uint32
	traces []*rebatchedTrace
}

// forwarder queues up traces to be sent to the metrics-generators
type forwarder struct {
	services.Service

	queue *util.CircularQueue

	forwardFunc func(ctx context.Context, userID string, keys []uint32, traces []*rebatchedTrace) error
}

func newForwarder(fn func(ctx context.Context, userID string, keys []uint32, traces []*rebatchedTrace) error) *forwarder {
	rf := &forwarder{
		// TODO: Make this configurable
		queue:       util.NewCircularQueue(10),
		forwardFunc: fn,
	}

	// TODO: Make this configurable
	rf.Service = services.NewTimerService(time.Millisecond*10, nil, rf.forward, nil).WithName("forwarder")

	return rf
}

func (rf *forwarder) forward(ctx context.Context) error {
	for rf.queue.CanRead() {
		req := rf.queue.Read().(*pushRingRequest)
		err := rf.forwardFunc(ctx, req.userID, req.keys, req.traces)
		if err != nil {
			level.Error(log.Logger).Log("msg", "forwarding traces failed", "err", err)
		}
	}
	return nil
}

// ForwardTraces queues up traces to be sent to the metrics-generators
func (rf *forwarder) ForwardTraces(_ context.Context, userID string, keys []uint32, traces []*rebatchedTrace) {
	overwrite := rf.queue.Write(&pushRingRequest{
		userID: userID,
		keys:   keys,
		traces: traces,
	})
	if overwrite {
		metricCircularQueueOverwrites.WithLabelValues(userID).Inc()
	}
}
