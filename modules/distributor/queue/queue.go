package queue

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-kit/log"
	"github.com/grafana/dskit/user"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	pushesTotalMetrics = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Subsystem: "distributor",
		Name:      "queue_pushes_total",
		Help:      "Total number of successful requests queued up for a tenant to the generatorForwarder",
	}, []string{"name", "tenant"})
	pushesFailuresTotalMetric = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Subsystem: "distributor",
		Name:      "queue_pushes_failures_total",
		Help:      "Total number of failed pushes to the queue for a tenant to the generatorForwarder",
	}, []string{"name", "tenant"})
	lengthMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "tempo",
		Subsystem: "distributor",
		Name:      "queue_length",
		Help:      "Number of queued requests for a tenant",
	}, []string{"name", "tenant"})
)

type ProcessFunc[T any] func(ctx context.Context, data T)

// Queue represents a single tenant's queue.
type Queue[T any] struct {
	// wg is used to wait for the workers to drain the queue while stopping
	wg sync.WaitGroup

	logger log.Logger

	name           string
	tenantID       string
	workerCount    int
	size           int
	reqChan        chan T
	fn             ProcessFunc[T]
	workersCloseCh chan struct{}

	pushesTotalMetrics         *prometheus.CounterVec
	pushesFailuresTotalMetrics *prometheus.CounterVec
	lengthMetric               *prometheus.GaugeVec

	readOnly *atomic.Bool
}

func New[T any](cfg Config, logger log.Logger, fn ProcessFunc[T]) *Queue[T] {
	return &Queue[T]{
		logger:                     logger,
		name:                       cfg.Name,
		tenantID:                   cfg.TenantID,
		workerCount:                cfg.WorkerCount,
		size:                       cfg.Size,
		reqChan:                    make(chan T, cfg.Size),
		fn:                         fn,
		workersCloseCh:             make(chan struct{}),
		pushesTotalMetrics:         pushesTotalMetrics,
		pushesFailuresTotalMetrics: pushesFailuresTotalMetric,
		lengthMetric:               lengthMetric,
		readOnly:                   &atomic.Bool{},
	}
}

// Push pushes data onto a queue.
// If the queue is full, the data is dropped
func (m *Queue[T]) Push(ctx context.Context, data T) error {
	if m.readOnly.Load() {
		return fmt.Errorf("queue is read-only")
	}

	m.pushesTotalMetrics.WithLabelValues(m.name, m.tenantID).Inc()

	select {
	case <-ctx.Done():
		m.pushesFailuresTotalMetrics.WithLabelValues(m.name, m.tenantID).Inc()
		return fmt.Errorf("failed to push data to queue for tenant=%s and queue_name=%s: %w", m.tenantID, m.name, ctx.Err())
	default:
	}

	select {
	case m.reqChan <- data:
		m.lengthMetric.WithLabelValues(m.name, m.tenantID).Inc()
		return nil
	default:
	}

	// Fail fast if the queue is full
	m.pushesFailuresTotalMetrics.WithLabelValues(m.name, m.tenantID).Inc()

	return fmt.Errorf("failed to push data to queue for tenant=%s and queue_name=%s: queue is full", m.tenantID, m.name)
}

func (m *Queue[T]) StartWorkers() {
	for i := 0; i < m.workerCount; i++ {
		m.wg.Add(1)

		go m.worker()
	}
}

// TenantID returns the tenant id.
func (m *Queue[T]) TenantID() string {
	return m.tenantID
}

// Size returns the size of the queue.
func (m *Queue[T]) Size() int {
	return m.size
}

// WorkerCount returns the number of expected workers.
func (m *Queue[T]) WorkerCount() int {
	return m.workerCount
}

// ShouldUpdate returns true if the queue size or worker count (alive or total) has changed
func (m *Queue[T]) ShouldUpdate(size, workerCount int) bool {
	return m.size != size || m.workerCount != workerCount
}

func (m *Queue[T]) Shutdown(ctx context.Context) error {
	// Call to stopWorkers only once
	if m.readOnly.CompareAndSwap(false, true) {
		return m.stopWorkers(ctx)
	}

	return nil
}

func (m *Queue[T]) worker() {
	defer m.wg.Done()

	for {
		select {
		case req := <-m.reqChan:
			m.lengthMetric.WithLabelValues(m.name, m.tenantID).Dec()
			m.forwardRequest(req)
		case <-m.workersCloseCh:
			// Forces to always trying to pull from the queue before exiting
			// This is important during shutdown to ensure that the queue is drained
			select {
			case req, ok := <-m.reqChan:
				if !ok { // closed
					return
				}

				m.lengthMetric.WithLabelValues(m.name, m.tenantID).Dec()
				m.forwardRequest(req)
			default:
				return
			}
		}
	}
}

func (m *Queue[T]) forwardRequest(req T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ctx = user.InjectOrgID(ctx, m.tenantID)

	m.fn(ctx, req)
}

func (m *Queue[T]) stopWorkers(ctx context.Context) error {
	// Close workersCloseCh and wait for all workers to return
	close(m.workersCloseCh)

	doneCh := make(chan struct{})
	go func() {
		m.wg.Wait()
		close(doneCh)
	}()

	select {
	case <-ctx.Done():
		return fmt.Errorf("failed to stop tenant %s Queue: %w", m.tenantID, ctx.Err())
	case <-doneCh:
		return nil
	}
}
