package flushqueues

import (
	"sync"

	"github.com/cortexproject/cortex/pkg/util"
	"github.com/prometheus/client_golang/prometheus"
)

type FlushQueues struct {
	queues     []*util.PriorityQueue
	index      int
	activeKeys sync.Map
}

// New creates a new set of flush queues with a prom gauge to track current depth
func New(queues int, metric prometheus.Gauge) *FlushQueues {
	f := &FlushQueues{
		queues: make([]*util.PriorityQueue, queues),
	}

	for j := 0; j < queues; j++ {
		f.queues[j] = util.NewPriorityQueue(metric)
	}

	return f
}

// Enqueue adds the op to the next queue and prevents any other items to be added with this key
func (f *FlushQueues) Enqueue(op util.Op, force bool) {
	_, ok := f.activeKeys.Load(op.Key())
	if ok && !force {
		return
	}

	f.activeKeys.Store(op.Key(), struct{}{})
	f.index++
	flushQueueIndex := f.index % len(f.queues)
	f.queues[flushQueueIndex].Enqueue(op)
}

// Dequeue removes the next op from the requested queue
func (f *FlushQueues) Dequeue(q int) util.Op {
	return f.queues[q].Dequeue()
}

// ClearKey unblocks the requested key.  This should be called only after a flush has been successful
func (f *FlushQueues) ClearKey(key string) {
	f.activeKeys.Delete(key)
}

// Stop closes all queues
func (f *FlushQueues) Stop() error {
	for _, q := range f.queues {
		q.Close()
	}

	return nil
}
