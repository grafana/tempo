package queue

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/weaveworks/common/user"
	"go.uber.org/atomic"
)

type request[T any] struct {
	data T
}

type ProcessFunc[T any] func(ctx context.Context, data T) error

// Queue represents a single tenant's queue.
type Queue[T any] struct {
	// wg is used to wait for the workers to drain the queue while stopping
	wg sync.WaitGroup

	logger log.Logger

	name           string
	tenantID       string
	workerCount    int
	size           int
	reqChan        chan *request[T]
	fn             ProcessFunc[T]
	workersCloseCh chan struct{}

	pushesTotalMetrics         *prometheus.CounterVec
	pushesFailuresTotalMetrics *prometheus.CounterVec
	lengthMetric               *prometheus.GaugeVec

	readOnly *atomic.Bool
}

func New[T any](cfg Config, logger log.Logger, reg prometheus.Registerer, fn ProcessFunc[T]) *Queue[T] {
	pushesTotalMetrics := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Subsystem: "distributor",
		Name:      "queue_pushes_total",
		Help:      "Total number of successful requests queued up for a tenant to the generatorForwarder",
	}, []string{"name", "tenant"})
	pushesFailuresTotalMetric := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Subsystem: "distributor",
		Name:      "queue_pushes_failures_total",
		Help:      "Total number of failed pushes to the queue for a tenant to the generatorForwarder",
	}, []string{"name", "tenant"})
	lengthMetric := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "tempo",
		Subsystem: "distributor",
		Name:      "queue_length",
		Help:      "Number of queued requests for a tenant",
	}, []string{"name", "tenant"})

	var alreadyRegisteredErr prometheus.AlreadyRegisteredError
	err := reg.Register(pushesTotalMetrics)
	if err != nil && !errors.As(err, &alreadyRegisteredErr) {
		_ = level.Warn(logger).Log("msg", "failed to register queue_pushes_total metric", "err", err)
	}
	err = reg.Register(pushesFailuresTotalMetric)
	if err != nil && !errors.As(err, &alreadyRegisteredErr) {
		_ = level.Warn(logger).Log("msg", "failed to register queue_pushes_failures_total metric", "err", err)
	}

	err = reg.Register(lengthMetric)
	if err != nil && !errors.As(err, &alreadyRegisteredErr) {
		_ = level.Warn(logger).Log("msg", "failed to register queue_length metric", "err", err)
	}

	return &Queue[T]{
		logger:                     logger,
		name:                       cfg.Name,
		tenantID:                   cfg.TenantID,
		workerCount:                cfg.WorkerCount,
		size:                       cfg.Size,
		reqChan:                    make(chan *request[T], cfg.Size),
		fn:                         fn,
		workersCloseCh:             make(chan struct{}),
		pushesTotalMetrics:         pushesTotalMetrics,
		pushesFailuresTotalMetrics: pushesFailuresTotalMetric,
		lengthMetric:               lengthMetric,
		readOnly:                   atomic.NewBool(false),
	}
}

// Push pushes data onto a queue.
// If the queue is full, the data is dropped
func (m *Queue[T]) Push(ctx context.Context, data T) error {
	if m.readOnly.Load() {
		return fmt.Errorf("queue is read-only")
	}

	m.pushesTotalMetrics.WithLabelValues(m.name, m.tenantID).Inc()

	req := &request[T]{
		data: data,
	}

	select {
	case <-ctx.Done():
		m.pushesFailuresTotalMetrics.WithLabelValues(m.name, m.tenantID).Inc()
		return fmt.Errorf("failed to push data to queue for tenant=%s and queue_name=%s: %w", m.tenantID, m.name, ctx.Err())
	default:
	}

	select {
	case m.reqChan <- req:
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

// Name returns queue name.
func (m *Queue[T]) Name() string {
	return m.name
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
	if m.readOnly.CAS(false, true) {
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
				if !ok { //closed
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

func (m *Queue[T]) forwardRequest(req *request[T]) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ctx = user.InjectOrgID(ctx, m.tenantID)

	if err := m.fn(ctx, req.data); err != nil {
		_ = level.Error(m.logger).Log("msg", "pushing with forwarder failed", "err", err)
	}
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
