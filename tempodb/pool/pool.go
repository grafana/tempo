package pool

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/atomic"
)

const (
	queueLengthReportDuration = 15 * time.Second
)

var (
	errPoolShutdown = errors.New("pool is shut down")

	metricQueryQueueLength = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "tempodb",
		Name:      "work_queue_length",
		Help:      "Current length of the work queue.",
	})

	metricQueryQueueMax = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "tempodb",
		Name:      "work_queue_max",
		Help:      "Maximum number of items in the work queue.",
	})
)

type JobFunc func(ctx context.Context, payload interface{}) (interface{}, error)

type result struct {
	data interface{}
	err  error
}

type job struct {
	ctx     context.Context
	payload interface{}
	fn      JobFunc

	resultsCh chan result
	stop      *atomic.Bool
}

type Pool struct {
	cfg  *Config
	size *atomic.Int32

	workQueue    chan *job
	shutdownCh   chan struct{}
	shutdownOnce sync.Once
}

func NewPool(cfg *Config) *Pool {
	if cfg == nil {
		cfg = defaultConfig()
	}

	q := make(chan *job, cfg.QueueDepth)
	p := &Pool{
		cfg:        cfg,
		workQueue:  q,
		size:       atomic.NewInt32(0),
		shutdownCh: make(chan struct{}),
	}

	for i := 0; i < cfg.MaxWorkers; i++ {
		go p.worker(q)
	}

	p.reportQueueLength()

	metricQueryQueueMax.Set(float64(cfg.QueueDepth))

	return p
}

func (p *Pool) RunJobs(ctx context.Context, payloads []interface{}, fn JobFunc) ([]interface{}, []error, error) {
	select {
	case <-p.shutdownCh:
		return nil, nil, errPoolShutdown
	default:
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	totalJobs := len(payloads)

	// sanity check before we even attempt to start adding jobs
	if int(p.size.Load())+totalJobs > p.cfg.QueueDepth {
		return nil, nil, fmt.Errorf("queue doesn't have room for %d jobs", len(payloads))
	}

	resultsCh := make(chan result, totalJobs) // way for jobs to send back results
	stop := atomic.NewBool(false)             // way to signal to the jobs to quit

	// add each job one at a time.  even though we checked length above these might still fail
	for _, payload := range payloads {
		j := &job{
			ctx:       ctx,
			fn:        fn,
			payload:   payload,
			resultsCh: resultsCh,
			stop:      stop,
		}

		select {
		case <-p.shutdownCh:
			stop.Store(true)
			return nil, nil, errPoolShutdown
		case p.workQueue <- j:
			p.size.Inc()
		default:
			stop.Store(true)
			return nil, nil, fmt.Errorf("failed to add a job to work queue")
		}
	}

	// Each job reports completion, including jobs without a result. Leave the
	// channel open so an in-flight job can finish after shutdown returns an error.
	var data []interface{}
	var funcErrs []error
	for range totalJobs {
		var completed result
		select {
		case <-p.shutdownCh:
			stop.Store(true)
			return nil, nil, errPoolShutdown
		case completed = <-resultsCh:
		}
		if completed.err != nil {
			funcErrs = append(funcErrs, completed.err)
		} else if completed.data != nil {
			data = append(data, completed.data)
		}
	}

	return data, funcErrs, nil
}

func (p *Pool) Shutdown() {
	p.shutdownOnce.Do(func() {
		close(p.shutdownCh)
	})
}

func (p *Pool) worker(j <-chan *job) {
	for {
		select {
		case <-p.shutdownCh:
			return
		case j, ok := <-j:
			if !ok {
				return
			}
			runJob(j)
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
			case <-p.shutdownCh:
				return
			}
		}
	}()
}

func runJob(job *job) {
	var completed result
	defer func() {
		// The buffer holds one completion per job, even after the caller returns.
		job.resultsCh <- completed
	}()

	// bail in case we have been asked to stop
	if job.ctx.Err() != nil {
		return
	}

	// bail in case not all jobs could be enqueued
	if job.stop.Load() {
		return
	}

	completed.data, completed.err = job.fn(job.ctx, job.payload)
}
