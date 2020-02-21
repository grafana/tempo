package pool

import (
	"fmt"
	"sync"
	"time"

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
		Name:      "work_queue_length",
		Help:      "Current length of the work queue.",
	})

	metricQueryQueueMax = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "friggdb",
		Name:      "work_queue_max",
		Help:      "Maximum number of items in the work queue.",
	})
)

type JobFunc func(payload interface{}) ([]byte, error)

type job struct {
	payload interface{}
	fn      JobFunc

	wg        *sync.WaitGroup
	resultsCh chan []byte
	stopCh    chan struct{}
	err       *atomic.Error
}

type Pool struct {
	cfg  *Config
	size *atomic.Int32

	workQueue chan interface{}
	stopCh    chan struct{}
}

func NewPool(cfg *Config) *Pool {
	if cfg == nil {
		cfg = defaultConfig()
	}

	q := make(chan interface{}, cfg.QueueDepth)
	p := &Pool{
		cfg:       cfg,
		workQueue: q,
		size:      atomic.NewInt32(0),
		stopCh:    make(chan struct{}),
	}

	for i := 0; i < cfg.MaxWorkers; i++ {
		go p.worker(q)
	}

	go p.reportQueueLength()

	metricQueryQueueMax.Set(float64(cfg.QueueDepth))

	return p
}

func (p *Pool) RunJobs(payloads []interface{}, fn JobFunc) ([]byte, error) {
	totalJobs := len(payloads)

	// sanity check before we even attempt to start adding jobs
	if int(p.size.Load())+totalJobs > p.cfg.QueueDepth {
		return nil, fmt.Errorf("queue doesn't have room for %d jobs", len(payloads))
	}

	resultsCh := make(chan []byte, 1)
	stopCh := make(chan struct{}, 0)
	wg := &sync.WaitGroup{}
	err := atomic.NewError(nil)

	wg.Add(totalJobs)
	// add each job one at a time.  even though we checked length above these might still fail
	for _, payload := range payloads {
		j := &job{
			fn:        fn,
			payload:   payload,
			wg:        wg,
			resultsCh: resultsCh,
			stopCh:    stopCh,
			err:       err,
		}

		select {
		case p.workQueue <- j:
			p.size.Inc()
		default:
			close(stopCh)
			return nil, fmt.Errorf("failed to add a job to work queue")
		}
	}

	allDone := make(chan struct{}, 1)
	go func() {
		wg.Wait()
		allDone <- struct{}{}
	}()

	select {
	case msg := <-resultsCh:
		close(stopCh)
		wg.Done()
		return msg, nil
	case <-allDone:
		return nil, err.Load()
	}
}

func (p *Pool) Shutdown() {
	close(p.workQueue)
	close(p.stopCh)
}

func (p *Pool) worker(j <-chan interface{}) {
	for {
		select {
		case <-p.stopCh:
			return
		case j, ok := <-j:
			if !ok {
				return
			}

			switch typedJob := j.(type) {
			case *stoppableJob:
				runStoppableJob(typedJob)
			case *job:
				runJob(typedJob)
			default:
				panic("unexpected job type")
			}

			p.size.Dec()
		}
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
			case <-p.stopCh:
				return
			}
		}
	}()
}

func runJob(job *job) {
	select {
	case <-job.stopCh:
		job.wg.Done()
		return
	default:
		msg, err := job.fn(job.payload)

		if msg != nil {
			select {
			case job.resultsCh <- msg:
				// not signalling done here to dodge race condition between results chan and done
				return
			default:
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
