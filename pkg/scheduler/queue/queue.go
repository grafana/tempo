package queue

import (
	"context"
	"sync"
	"time"

	"github.com/grafana/dskit/services"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/atomic"
)

const (
	// How frequently to check for disconnected queriers that should be forgotten.
	forgetCheckPeriod = 5 * time.Second
)

var (
	ErrTooManyRequests = errors.New("too many outstanding requests")
	ErrStopped         = errors.New("queue is stopped")
)

// UserIndex is opaque type that allows to resume iteration over users between successive calls
// of RequestQueue.GetNextRequestForQuerier method.
type UserIndex struct {
	last int
}

// Modify index to start iteration on the same user, for which last queue was returned.
func (ui UserIndex) ReuseLastUser() UserIndex {
	if ui.last >= 0 {
		return UserIndex{last: ui.last - 1}
	}
	return ui
}

// FirstUser returns UserIndex that starts iteration over user queues from the very first user.
func FirstUser() UserIndex {
	return UserIndex{last: -1}
}

// Request stored into the queue.
type Request interface{}

// RequestQueue holds incoming requests in per-user queues. It also assigns each user specified number of queriers,
// and when querier asks for next request to handle (using GetNextRequestForQuerier), it returns requests
// in a fair fashion.
type RequestQueue struct {
	services.Service

	connectedQuerierWorkers *atomic.Int32

	mtx     sync.Mutex
	cond    *sync.Cond // Notified when request is enqueued or dequeued, or querier is disconnected.
	queues  *queues
	stopped bool

	outgoing chan Request

	queueLength       *prometheus.GaugeVec   // Per user and reason.
	discardedRequests *prometheus.CounterVec // Per user.
}

func NewRequestQueue(maxOutstandingPerTenant int, forgetDelay time.Duration, queueLength *prometheus.GaugeVec, discardedRequests *prometheus.CounterVec) *RequestQueue {
	q := &RequestQueue{
		queues:                  newUserQueues(maxOutstandingPerTenant, forgetDelay),
		connectedQuerierWorkers: atomic.NewInt32(0),
		queueLength:             queueLength,
		discardedRequests:       discardedRequests,
		outgoing:                make(chan Request, 1),
	}

	q.cond = sync.NewCond(&q.mtx)

	q.Service = services.NewIdleService(func(ctx context.Context) (err error) { return nil }, q.stopping)

	go q.queueWorker()
	return q
}

// EnqueueRequest puts the request into the queue. MaxQueries is user-specific value that specifies how many queriers can
// this user use (zero or negative = all queriers). It is passed to each EnqueueRequest, because it can change
// between calls.
//
// If request is successfully enqueued, successFn is called with the lock held, before any querier can receive the request.
func (q *RequestQueue) EnqueueRequest(userID string, req Request, maxQueriers int, successFn func()) error {
	q.mtx.Lock()
	defer q.mtx.Unlock()

	if q.stopped {
		return ErrStopped
	}

	queue := q.queues.getOrAddQueue(userID, maxQueriers)
	if queue == nil {
		// This can only happen if userID is "".
		return errors.New("no queue found")
	}

	select {
	case queue <- req:
		q.queueLength.WithLabelValues(userID).Inc()
		q.cond.Broadcast()
		// Call this function while holding a lock. This guarantees that no querier can fetch the request before function returns.
		if successFn != nil {
			successFn()
		}
		return nil
	default:
		q.discardedRequests.WithLabelValues(userID).Inc()
		return ErrTooManyRequests
	}
}

func (q *RequestQueue) queueWorker() {
	last := FirstUser()

FindQueue:
	q.mtx.Lock()
	// We need to wait if there are no users, or no pending requests for given querier.
	for q.queues.len() == 0 && !q.stopped {
		q.cond.Wait()
	}

	if q.stopped {
		q.mtx.Unlock()
		return
	}

	for {
		queue, userID, idx := q.queues.getNextQueue(last.last)
		last.last = idx

		if queue == nil {
			break
		}

		// Pick next request from the queue.
		request := <-queue
		if len(queue) == 0 {
			q.queues.deleteQueue(userID)
		}

		q.queueLength.WithLabelValues(userID).Dec()

		// Tell close() we've processed a request.
		q.cond.Broadcast()

		q.outgoing <- request
	}

	q.mtx.Unlock()
	// There are no unexpired requests, so we can get back
	// and wait for more requests.
	goto FindQueue
}

// GetNextRequestForQuerier find next user queue and takes the next request off of it. Will block if there are no requests.
// By passing user index from previous call of this method, querier guarantees that it iterates over all users fairly.
// If querier finds that request from the user is already expired, it can get a request for the same user by using UserIndex.ReuseLastUser.
func (q *RequestQueue) GetNextRequestForQuerier(ctx context.Context, last UserIndex) (Request, UserIndex, error) { // jpe ditch last user index
	// select ctx.Done against outgoing queue
	select {
	case <-ctx.Done():
		return nil, last, ctx.Err()
	case req := <-q.outgoing:
		return req, last, nil
	}
}

func (q *RequestQueue) stopping(_ error) error {
	q.mtx.Lock()
	defer q.mtx.Unlock()

	for q.queues.len() > 0 && q.connectedQuerierWorkers.Load() > 0 {
		q.cond.Wait()
	}

	// Only stop after dispatching enqueued requests.
	q.stopped = true

	// If there are still goroutines in GetNextRequestForQuerier method, they get notified.
	q.cond.Broadcast()

	return nil
}

func (q *RequestQueue) RegisterQuerierConnection(querier string) {
	q.connectedQuerierWorkers.Inc()
}

func (q *RequestQueue) UnregisterQuerierConnection(querier string) {
	q.connectedQuerierWorkers.Dec()
}

// When querier is waiting for next request, this unblocks the method.
func (q *RequestQueue) QuerierDisconnecting() {
	q.cond.Broadcast()
}

func (q *RequestQueue) GetConnectedQuerierWorkersMetric() float64 {
	return float64(q.connectedQuerierWorkers.Load())
}
