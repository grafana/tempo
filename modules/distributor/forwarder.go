package distributor

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-kit/log/level"
	"github.com/grafana/dskit/services"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/util/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/atomic"
	"go.uber.org/multierr"
)

const (
	defaultWorkerCount = 2
	defaultQueueSize   = 100
)

var (
	metricForwarderPushes = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "distributor_forwarder_pushes_total",
		Help:      "Total number of successful requests queued up for a tenant to the forwarder",
	}, []string{"tenant"})
	metricForwarderPushesFailures = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "distributor_forwarder_pushes_failures_total",
		Help:      "Total number of failed pushes to the queue for a tenant to the forwarder",
	}, []string{"tenant"})
	metricForwarderQueueLength = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "tempo",
		Name:      "distributor_forwarder_queue_length",
		Help:      "Number of queued requests for a tenant",
	}, []string{"tenant"})
)

type forwardFunc func(ctx context.Context, tenantID string, keys []uint32, traces []*rebatchedTrace) error

type request struct {
	keys   []uint32
	traces []*rebatchedTrace
}

// forwarder queues up traces to be sent to the metrics-generators
type forwarder struct {
	services.Service

	// per-tenant queue managers
	queueManagers map[string]*queueManager
	mutex         sync.RWMutex

	forwardFunc forwardFunc

	o                 *overrides.Overrides
	overridesInterval time.Duration
	shutdown          chan interface{}
}

func newForwarder(fn forwardFunc, o *overrides.Overrides) *forwarder {
	rf := &forwarder{
		queueManagers:     make(map[string]*queueManager),
		mutex:             sync.RWMutex{},
		forwardFunc:       fn,
		o:                 o,
		overridesInterval: time.Minute,
		shutdown:          make(chan interface{}),
	}

	rf.Service = services.NewIdleService(rf.start, rf.stop)

	return rf
}

// SendTraces queues up traces to be sent to the metrics-generators
func (f *forwarder) SendTraces(ctx context.Context, tenantID string, keys []uint32, traces []*rebatchedTrace) {
	select {
	case <-f.shutdown:
		return
	default:
	}

	qm := f.getOrCreateQueueManager(tenantID)
	err := qm.pushToQueue(ctx, &request{keys: keys, traces: traces})
	if err != nil {
		level.Error(log.Logger).Log("msg", "failed to push traces to queue", "tenant", tenantID, "err", err)
		metricForwarderPushesFailures.WithLabelValues(tenantID).Inc()
	}

	metricForwarderPushes.WithLabelValues(tenantID).Inc()
}

// getQueueManagerConfig returns queueSize and workerCount for the given tenant
func (f *forwarder) getQueueManagerConfig(tenantID string) (queueSize, workerCount int) {
	queueSize = f.o.MetricsGeneratorForwarderQueueSize(tenantID)
	if queueSize == 0 {
		queueSize = defaultQueueSize
	}

	workerCount = f.o.MetricsGeneratorForwarderWorkers(tenantID)
	if workerCount == 0 {
		workerCount = defaultWorkerCount
	}
	return queueSize, workerCount
}

func (f *forwarder) getOrCreateQueueManager(tenantID string) *queueManager {
	qm, ok := f.getQueueManager(tenantID)
	if ok {
		return qm
	}

	f.mutex.Lock()
	defer f.mutex.Unlock()

	queueSize, workerCount := f.getQueueManagerConfig(tenantID)
	f.queueManagers[tenantID] = newQueueManager(tenantID, queueSize, workerCount, f.forwardFunc)

	return f.queueManagers[tenantID]
}

func (f *forwarder) getQueueManager(tenantID string) (*queueManager, bool) {
	f.mutex.RLock()
	defer f.mutex.RUnlock()

	qm, ok := f.queueManagers[tenantID]
	return qm, ok
}

// watchOverrides watches the overrides for changes
// and updates the queueManagers accordingly
func (f *forwarder) watchOverrides() {
	for {
		select {
		case <-time.After(f.overridesInterval):
			f.mutex.Lock()
			for tenantID, tm := range f.queueManagers {
				queueSize, workerCount := f.getQueueManagerConfig(tenantID)

				// if the queue size or worker count has changed, shutdown the queue manager and create a new one
				if tm.shouldUpdate(queueSize, workerCount) {
					go func() {
						// shutdown the queue manager
						// this will block until all workers have finished and the queue is drained
						if err := tm.shutdown(); err != nil {
							level.Error(log.Logger).Log("msg", "error shutting down queue manager", "tenant", tenantID, "err", err)
						}
					}()
					delete(f.queueManagers, tenantID)
					f.queueManagers[tenantID] = newQueueManager(tenantID, queueSize, workerCount, f.forwardFunc)
				}
			}
			f.mutex.Unlock()
		case <-f.shutdown:
			return
		}
	}
}

func (f *forwarder) start(_ context.Context) error {
	go f.watchOverrides()

	return nil

}

func (f *forwarder) stop(_ error) error {
	close(f.shutdown)
	var errs []error
	for _, tm := range f.queueManagers {
		if err := tm.shutdown(); err != nil {
			errs = append(errs, err)
		}
	}
	return multierr.Combine(errs...)
}

// queueManager manages a single tenant's queue
type queueManager struct {
	// wg is used to wait for the workers to drain the queue while stopping
	wg sync.WaitGroup

	tenantID         string
	workerCount      int
	workerAliveCount *atomic.Int32
	queueSize        int
	reqChan          chan *request
	fn               forwardFunc
	workersCloseCh   chan struct{}

	readOnly *atomic.Bool
}

func newQueueManager(tenantID string, queueSize, workerCount int, fn forwardFunc) *queueManager {
	m := &queueManager{
		tenantID:         tenantID,
		workerCount:      workerCount,
		queueSize:        queueSize,
		workerAliveCount: atomic.NewInt32(0),
		reqChan:          make(chan *request, queueSize),
		fn:               fn,
		workersCloseCh:   make(chan struct{}),
		readOnly:         atomic.NewBool(false),
	}

	m.startWorkers()

	return m
}

// pushToQueue a trace to the queue
// if the queue is full, the trace is dropped
func (m *queueManager) pushToQueue(ctx context.Context, req *request) error {
	if m.readOnly.Load() {
		return fmt.Errorf("queue is read-only")
	}

	select {
	case m.reqChan <- req:
		metricForwarderQueueLength.WithLabelValues(m.tenantID).Inc()
	case <-ctx.Done():
		return fmt.Errorf("failed to pushToQueue traces to tenant %s queue: %w", m.tenantID, ctx.Err())
	default:
		// Fail fast if the queue is full
		return fmt.Errorf("failed to pushToQueue traces to tenant %s queue: queue is full", m.tenantID)

	}

	return nil
}

func (m *queueManager) startWorkers() {
	for i := 0; i < m.workerCount; i++ {
		m.wg.Add(1)
		m.workerAliveCount.Inc()

		go m.worker()
	}
}

func (m *queueManager) worker() {
	defer func() {
		m.wg.Done()
		m.workerAliveCount.Dec()
	}()

	for {
		select {
		case req := <-m.reqChan:
			metricForwarderQueueLength.WithLabelValues(m.tenantID).Dec()
			m.forwardRequest(context.Background(), req)
		default:
			// Forces to always trying to pull from the queue before exiting
			// This is important during shutdown to ensure that the queue is drained
			select {
			case req := <-m.reqChan:
				metricForwarderQueueLength.WithLabelValues(m.tenantID).Dec()
				m.forwardRequest(context.Background(), req)
			case <-m.workersCloseCh:
				// If the queue isn't empty, force to start the loop from the beginning
				if len(m.reqChan) > 0 {
					continue
				}
				return
			}
		}
	}
}

func (m *queueManager) forwardRequest(ctx context.Context, req *request) {
	if err := m.fn(ctx, m.tenantID, req.keys, req.traces); err != nil {
		level.Error(log.Logger).Log("msg", "pushing to metrics-generators failed", "err", err)
	}
}

func (m *queueManager) shutdown() error {
	// Call to stopWorkers only once
	if m.readOnly.CAS(false, true) {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		return m.stopWorkers(ctx)
	}

	return nil
}

func (m *queueManager) stopWorkers(ctx context.Context) error {
	// Close workersCloseCh and wait for all workers to return
	close(m.workersCloseCh)

	doneCh := make(chan struct{})
	go func() {
		m.wg.Wait()
		close(doneCh)
	}()

	select {
	case <-ctx.Done():
		return fmt.Errorf("failed to stop tenant %s queueManager: %w", m.tenantID, ctx.Err())
	case <-doneCh:
		return nil
	}
}

// shouldUpdate returns true if the queue size or worker count (alive or total) has changed
func (m *queueManager) shouldUpdate(numWorkers int, queueSize int) bool {
	// TODO: worker alive count could be 0 and shutting down the queue manager would be impossible
	//  it'd be better if we were able to spawn new workers instead of just closing the queueManager
	//  and creating a new one
	return m.queueSize != queueSize || m.workerCount != numWorkers || m.workerAliveCount.Load() != int32(numWorkers)
}
