package flushqueues

import (
	"github.com/cortexproject/cortex/pkg/util"
	"github.com/prometheus/client_golang/prometheus"
)

type FlushQueues struct {
	queues []*util.PriorityQueue
	index  int
}

func New(queues int, metric prometheus.Gauge) *FlushQueues {
	f := &FlushQueues{
		queues: make([]*util.PriorityQueue, queues),
	}

	for j := 0; j < queues; j++ {
		f.queues[j] = util.NewPriorityQueue(metric)
	}

	return f
}

func (f *FlushQueues) Enqueue(op util.Op) {
	f.index++
	flushQueueIndex := f.index % len(f.queues)
	f.queues[flushQueueIndex].Enqueue(op)
}

func (f *FlushQueues) Dequeue(q int) util.Op {
	return f.queues[q].Dequeue()
}

func (f *FlushQueues) Stop() error {
	for _, q := range f.queues {
		q.Close()
	}

	return nil
}
