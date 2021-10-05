package pool

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/uber-go/atomic"
	"go.uber.org/multierr"
)

const (
	queueLengthReportDuration = 15 * time.Second
)

var (
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

// JobStats are metrics of running the provided func for the given payloads
type JobStats struct {
	// fnErrs contain the errors of executing the func
	fnErrs []error
}

func (s *JobStats) Errs() error {
	if len(s.fnErrs) != 0 {
		return multierr.Combine(s.fnErrs...)
	}
	return nil
}

func (s *JobStats) ErrsCount() int {
	return len(s.fnErrs)
}

var jobStats = JobStats{}

type JobFunc func(ctx context.Context, payload interface{}) ([]byte, string, error)

type result struct {
	data []byte
	enc  string
	err  error
}

type job struct {
	ctx     context.Context
	cancel  context.CancelFunc
	payload interface{}
	fn      JobFunc

	wg        *sync.WaitGroup
	resultsCh chan result
	stop      *atomic.Bool
}

type Pool struct {
	cfg  *Config
	size *atomic.Int32

	workQueue  chan *job
	shutdownCh chan struct{}
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

func (p *Pool) RunJobs(ctx context.Context, payloads []interface{}, fn JobFunc) ([][]byte, []string, JobStats, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	totalJobs := len(payloads)

	// sanity check before we even attempt to start adding jobs
	if int(p.size.Load())+totalJobs > p.cfg.QueueDepth {
		return nil, nil, jobStats, fmt.Errorf("queue doesn't have room for %d jobs", len(payloads))
	}

	resultsCh := make(chan result, totalJobs) // way for jobs to send back results
	stop := atomic.NewBool(false)             // way to signal to the jobs to quit
	wg := &sync.WaitGroup{}                   // way to wait for all jobs to complete

	// add each job one at a time.  even though we checked length above these might still fail
	for _, payload := range payloads {
		wg.Add(1)
		j := &job{
			ctx:       ctx,
			cancel:    cancel,
			fn:        fn,
			payload:   payload,
			wg:        wg,
			resultsCh: resultsCh,
			stop:      stop,
		}

		select {
		case p.workQueue <- j:
			p.size.Inc()
		default:
			wg.Done()
			stop.Store(true)
			return nil, nil, jobStats, fmt.Errorf("failed to add a job to work queue")
		}
	}

	// wait for all jobs to finish
	wg.Wait()

	// close resultsCh
	close(resultsCh)

	// read all from results channel
	var data [][]byte
	var enc []string
	var stats JobStats
	for res := range resultsCh {
		if res.err != nil {
			stats.fnErrs = append(stats.fnErrs, res.err)
		} else {
			data = append(data, res.data)
			enc = append(enc, res.enc)
		}
	}

	return data, enc, stats, nil
}

func (p *Pool) Shutdown() {
	close(p.workQueue)
	close(p.shutdownCh)
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
	defer job.wg.Done()

	// bail in case not all jobs could be enqueued
	if job.stop.Load() {
		return
	}

	data, enc, err := job.fn(job.ctx, job.payload)
	if data != nil || err != nil {
		// Commenting out job cancellations for now because of a resource leak suspected in the GCS golang client.
		// Issue logged here: https://github.com/googleapis/google-cloud-go/issues/3018
		// job.cancel()
		select {
		case job.resultsCh <- result{
			data: data,
			enc:  enc,
			err:  err,
		}:
		default: // if we hit default it means that something else already returned a good result.  /shrug
		}
	}
}
