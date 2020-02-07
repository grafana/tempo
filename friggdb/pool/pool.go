package pool

import (
	"fmt"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/uber-go/atomic"
)

const (
	queueLengthReportDuration = 15 * time.Second
)

var (
	metricQueryQueueLength = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "friggdb",
		Name:      "query_queue_length",
		Help:      "Current length of the query queue.",
	})

	metricQueryQueueMax = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "friggdb",
		Name:      "query_queue_max",
		Help:      "Maximum number of items in the query queue.",
	})
)

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

	metricQueryQueueMax.Set(float64(cfg.QueueDepth))

	return p
}

func (p *Pool) RunJobs(payloads []interface{}, fn JobFunc) (proto.Message, error) {
	totalJobs := len(payloads)

	// sanity check before we even attempt to start adding jobs
	if int(p.size.Load())+totalJobs > p.cfg.QueueDepth {
		return nil, fmt.Errorf("queue doesn't have room for %d jobs", len(payloads))
	}

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

	allDone := make(chan struct{}, 1)
	go func() {
		wg.Wait()
		allDone <- struct{}{}
	}()

	select {
	case msg := <-results:
		wg.Done()
		stopped.Store(true) // todo: use stopCh instead?
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
				// not signalling done here to dodge race condition between results chan and done
				continue
			default:
				// this is weird.  found the id twice?
			}
		}
		if err != nil {
			job.err.Store(err)
		}
		job.wg.Done()
	}
}

func (p *Pool) reportQueueLength() {
	ticker := time.NewTicker(queueLengthReportDuration)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				metricQueryQueueLength.Set(float64(p.size.Load()))
			}
		}
	}()
} // jpe : shutdown: stopch?

// default is concurrency disabled
func defaultConfig() *Config {
	return &Config{
		MaxWorkers: 30,
		QueueDepth: 10000,
	}
}
