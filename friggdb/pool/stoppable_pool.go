package pool

import (
	"fmt"
	"sync"

	"github.com/uber-go/atomic"
)

type StoppableJobFunc func(payload interface{}, stopCh <-chan struct{}) error

type Stopper struct {
	wg     *sync.WaitGroup
	stopCh chan struct{}
	err    *atomic.Error
}

type stoppableJob struct {
	payload interface{}
	fn      StoppableJobFunc

	wg     *sync.WaitGroup
	stopCh chan struct{}
	err    *atomic.Error
}

func (p *Pool) RunStoppableJobs(payloads []interface{}, fn StoppableJobFunc) (*Stopper, error) {
	totalJobs := len(payloads)

	// sanity check before we even attempt to start adding jobs
	if int(p.size.Load())+totalJobs > p.cfg.QueueDepth {
		return nil, fmt.Errorf("queue doesn't have room for %d jobs", len(payloads))
	}

	stopCh := make(chan struct{})
	wg := &sync.WaitGroup{}
	err := atomic.NewError(nil)

	stopper := &Stopper{
		wg:     wg,
		stopCh: stopCh,
		err:    err,
	}

	wg.Add(totalJobs)
	// add each job one at a time.  even though we checked length above these might still fail
	for _, payload := range payloads {
		j := &stoppableJob{
			fn:      fn,
			payload: payload,
			wg:      wg,
			stopCh:  stopCh,
			err:     err,
		}

		select {
		case p.workQueue <- j:
			p.size.Inc()
		default:
			close(stopCh)
			return nil, fmt.Errorf("failed to add a job to work queue")
		}
	}

	return stopper, nil
}

func (s *Stopper) Stop() error {
	close(s.stopCh)
	s.wg.Wait()

	return s.err.Load()
}

func runStoppableJob(job *stoppableJob) {
	select {
	case <-job.stopCh:
		job.wg.Done()
		return
	default:
		err := job.fn(job.payload, job.stopCh)
		job.err.Store(err)
		job.wg.Done()
	}
}
