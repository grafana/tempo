package pool

import (
	"fmt"
	"sync"

	"github.com/golang/protobuf/proto"
	"github.com/uber-go/atomic"
)

// jpe: work queue length metric

type JobFunc func(payload interface{}) (proto.Message, error)

type job struct {
	payload interface{}
	fn      JobFunc

	wg      *sync.WaitGroup
	results chan proto.Message
	stopped *atomic.Bool
	err     *atomic.Error
}

type Pool struct {
	cfg  *Config
	size *atomic.Int32

	workQueue chan *job
}

func NewPool(cfg *Config) *Pool {
	if cfg == nil {
		cfg = defaultConfig()
	}

	q := make(chan *job, cfg.QueueDepth)
	p := &Pool{
		cfg:       cfg,
		workQueue: q,
		size:      atomic.NewInt32(0),
	}

	for i := 0; i < cfg.MaxWorkers; i++ {
		go p.worker(q)
	}

	return p
}

func (p *Pool) RunJobs(payloads []interface{}, fn JobFunc) (proto.Message, error) {

	// sanity check before we even attempt to start adding jobs
	if int(p.size.Load())+len(payloads) > p.cfg.QueueDepth {
		return nil, fmt.Errorf("queue doesn't have room for %d jobs", len(payloads))
	}

	totalJobs := len(payloads)
	results := make(chan proto.Message, 1)
	wg := &sync.WaitGroup{}
	stopped := atomic.NewBool(false)
	err := atomic.NewError(nil)

	wg.Add(totalJobs)
	// add each job one at a time.  these might still fail
	for _, payload := range payloads {
		j := &job{
			fn:      fn,
			payload: payload,
			wg:      wg,
			results: results,
			stopped: stopped,
			err:     err,
		}

		select {
		case p.workQueue <- j:
			p.size.Inc()
		default:
			stopped.Store(true)
			return nil, fmt.Errorf("failed to add a job due to queue being full")
		}
	}

	allDone := make(chan struct{})
	go func() {
		wg.Wait()
		allDone <- struct{}{}
	}()

	select {
	case msg := <-results:
		wg.Done()
		stopped.Store(true)
		return msg, nil
	case <-allDone:
		return nil, err.Load()
	}
}

// jpe: call/test this
func (p *Pool) Shutdown() {
	close(p.workQueue)
}

func (p *Pool) worker(j <-chan *job) {
	for job := range j {
		p.size.Dec()

		if job.stopped.Load() {
			job.wg.Done()
			continue
		}

		msg, err := job.fn(job.payload)

		if msg != nil {
			select {
			case job.results <- msg:
				// purposefully not flagging job.wg.Done().  this prevents a race between the wg and the results chan
				continue
			default:
				fmt.Println("asdfasdfasdf")
				// this is weird.  found the id twice?
			}
		}
		if err != nil {
			job.err.Store(err)
		}
		job.wg.Done()
	}
}

// default is concurrency disabled
func defaultConfig() *Config {
	return &Config{
		MaxWorkers: 30,
		QueueDepth: 10000,
	}
}
