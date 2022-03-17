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
	"go.uber.org/multierr"
)

var (
	metricForwarderDroppedPushes = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "distributor_forwarder_dropped_pushes",
		Help:      "Total number of dropped pushes for a tenant to the metrics-generator's",
	}, []string{"tenant"})
)

type forwardFunc func(ctx context.Context, userID string, keys []uint32, traces []*rebatchedTrace) error

type pushRingRequest struct {
	keys   []uint32
	traces []*rebatchedTrace
}

// forwarder queues up traces to be sent to the metrics-generators
type forwarder struct {
	services.Service

	// per-tenant queue managers
	queueManagers map[string]*queueManager
	mutex         sync.Mutex

	forwardFunc forwardFunc

	o        *overrides.Overrides
	shutdown chan interface{}
}

func newForwarder(fn forwardFunc, o *overrides.Overrides) *forwarder {
	rf := &forwarder{
		queueManagers: make(map[string]*queueManager),
		mutex:         sync.Mutex{},
		forwardFunc:   fn,
		o:             o,
		shutdown:      make(chan interface{}),
	}

	rf.Service = services.NewBasicService(rf.start, rf.running, rf.stop)

	return rf
}

// SendTraces queues up traces to be sent to the metrics-generators
func (rf *forwarder) SendTraces(ctx context.Context, userID string, keys []uint32, traces []*rebatchedTrace) {
	rf.mutex.Lock()
	defer rf.mutex.Unlock()

	if _, ok := rf.queueManagers[userID]; !ok {
		rf.queueManagers[userID] = newQueueManager(userID, rf.forwardFunc, rf.o)
	}

	err := rf.queueManagers[userID].push(ctx, &pushRingRequest{keys: keys, traces: traces})
	if err != nil {
		metricForwarderDroppedPushes.WithLabelValues(userID).Inc()
	}
}

// watchOverrides watches the overrides for changes
// and updates the queueManagers accordingly
func (rf *forwarder) watchOverrides() {
	for {
		select {
		case <-time.After(time.Second * 5):
			rf.mutex.Lock()
			for _, tm := range rf.queueManagers {
				tm.watchOverrides(rf.o)
			}
			rf.mutex.Unlock()
		case <-rf.shutdown:
			return
		}
	}
}

func (rf *forwarder) start(_ context.Context) error {
	go rf.watchOverrides()

	return nil

}

func (rf *forwarder) running(ctx context.Context) error {
	<-ctx.Done()
	return nil
}

func (rf *forwarder) stop(_ error) error {
	close(rf.shutdown)
	var errs []error
	for _, tm := range rf.queueManagers {
		if err := tm.shutdown(); err != nil {
			errs = append(errs, err)
		}
	}
	return multierr.Combine(errs...)
}

// queueManager manages a single tenant's queue
type queueManager struct {
	mutex sync.Mutex
	wg    sync.WaitGroup

	tenantID       string
	numWorkers     int
	queueSize      int
	reqChan        chan *pushRingRequest
	fn             forwardFunc
	workersCloseCh chan struct{}

	shutdownCh chan struct{}
}

func newQueueManager(tenantID string, fn forwardFunc, o *overrides.Overrides) *queueManager {
	m := &queueManager{
		mutex:      sync.Mutex{},
		tenantID:   tenantID,
		fn:         fn,
		shutdownCh: make(chan struct{}),
	}

	m.watchOverrides(o)
	return m
}

// push a trace to the queue
// if the queue is full, the trace is dropped
func (m *queueManager) push(ctx context.Context, req *pushRingRequest) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	select {
	case m.reqChan <- req:
		// TODO: Record some metric
	case <-ctx.Done():
		return fmt.Errorf("failed to push traces to tenant %s queue", m.tenantID)
	}

	return nil
}

// watchOverrides watches the overrides for the tenant and updates the queue and workers
func (m *queueManager) watchOverrides(o *overrides.Overrides) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	numWorkers := o.MetricsGeneratorSendWorkers(m.tenantID)
	queueSize := o.MetricsGeneratorSendQueueSize(m.tenantID)

	if !m.shouldUpdate(numWorkers, queueSize) {
		return
	}

	if err := m.stopWorkers(); err != nil {
		level.Error(log.Logger).Log("msg", "failed to stop tenant workers", "err", err)
		return
	}

	m.updateConfig(numWorkers, queueSize)

	m.startWorkers()
}

func (m *queueManager) startWorkers() {
	m.workersCloseCh = make(chan struct{})
	m.workerLoop(m.numWorkers)
}

func (m *queueManager) shutdown() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// TODO: Flush the queue before exiting
	return m.stopWorkers()
}

func (m *queueManager) updateConfig(numWorkers int, queueSize int) {
	m.numWorkers = numWorkers
	m.reqChan = make(chan *pushRingRequest, queueSize)
}

func (m *queueManager) stopWorkers() error {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	if m.workersCloseCh == nil {
		return nil
	}

	// Close workersCloseCh and wait for all workers to return
	close(m.workersCloseCh)

	doneCh := make(chan struct{})
	go func() {
		m.wg.Wait()
		close(doneCh)
	}()

	select {
	case <-ctx.Done():
		return fmt.Errorf("failed to stop tenant %s queueManager", m.tenantID)
	case <-doneCh:
		return nil
	}
}

func (m *queueManager) workerLoop(n int) {
	for i := 0; i < n; i++ {
		m.wg.Add(1)
		go func() {
			defer m.wg.Done()
			for {
				select {
				case req := <-m.reqChan:
					// TODO: add timeout to context?
					if err := m.fn(context.Background(), m.tenantID, req.keys, req.traces); err != nil {
						level.Error(log.Logger).Log("msg", "pushing to metrics-generators failed", "err", err)
					}
				case <-m.workersCloseCh:
					return
				}
			}
		}()
	}
}

func (m *queueManager) shouldUpdate(numWorkers int, queueSize int) bool {
	// TODO: check if new workers are needed too (e.g. one panicked)
	//  maybe change sync.WaitGroup for an atomic int?
	return m.queueSize != queueSize || m.numWorkers != numWorkers
}
