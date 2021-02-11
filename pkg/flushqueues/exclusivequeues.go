package flushqueues

import (
	"sync"

	"github.com/cortexproject/cortex/pkg/util"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/uber-go/atomic"
)

type ExclusiveQueues struct {
	queues     []*util.PriorityQueue
	index      *atomic.Int32
	activeKeys sync.Map
}

// New creates a new set of flush queues with a prom gauge to track current depth
func New(queues int, metric prometheus.Gauge) *ExclusiveQueues {
	f := &ExclusiveQueues{
		queues: make([]*util.PriorityQueue, queues),
		index:  atomic.NewInt32(0),
	}

	for j := 0; j < queues; j++ {
		f.queues[j] = util.NewPriorityQueue(metric)
	}

	return f
}

// Enqueue adds the op to the next queue and prevents any other items to be added with this key
func (f *ExclusiveQueues) Enqueue(op util.Op) {
	_, ok := f.activeKeys.Load(op.Key())
	if ok {
		return
	}

	f.activeKeys.Store(op.Key(), struct{}{})
	f.Requeue(op)
}

// Dequeue removes the next op from the requested queue.  After dequeueing the calling
//  process either needs to call ClearKey or Requeue
func (f *ExclusiveQueues) Dequeue(q int) util.Op {
	return f.queues[q].Dequeue()
}

// Requeue adds an op that is presumed to already be covered by activeKeys
func (f *ExclusiveQueues) Requeue(op util.Op) {
	flushQueueIndex := int(f.index.Inc()) % len(f.queues)
	f.queues[flushQueueIndex].Enqueue(op)
}

// Clear unblocks the requested op.  This should be called only after a flush has been successful
func (f *ExclusiveQueues) Clear(op util.Op) {
	f.activeKeys.Delete(op.Key())
}

func (f *ExclusiveQueues) IsEmpty() bool {
	for _, queue := range f.queues {
		if queue.Length() > 0 {
			return false
		}
	}
	return true
}

// Stop closes all queues
func (f *ExclusiveQueues) Stop() {
	for _, q := range f.queues {
		q.Close()
	}
}
