package pool

import (
	"context"
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

type JobFunc func(ctx context.Context, payload interface{}) ([]byte, string, error)

type result struct {
	data []byte
	enc  string
}

type job struct {
	ctx     context.Context
	cancel  context.CancelFunc
	payload interface{}
	fn      JobFunc

	wg        *sync.WaitGroup
	resultsCh chan result
	stop      *atomic.Bool
	err       chan error
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

// RunJobs executes the provided function for the given payloads in parallel
// It returns a slice of errors from executing the fn and an error for pool related errors.
func (p *Pool) RunJobs(ctx context.Context, payloads []interface{}, fn JobFunc) ([][]byte, []string, []error, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	totalJobs := len(payloads)

	// sanity check before we even attempt to start adding jobs
	if int(p.size.Load())+totalJobs > p.cfg.QueueDepth {
		return nil, nil, nil, fmt.Errorf("queue doesn't have room for %d jobs", len(payloads))
	}

	resultsCh := make(chan result, totalJobs) // way for jobs to send back results
	errCh := make(chan error, totalJobs)      // way for jobs to send back an error
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
			err:       errCh,
		}

		select {
		case p.workQueue <- j:
			p.size.Inc()
		default:
			wg.Done()
			stop.Store(true)
			return nil, nil, nil, fmt.Errorf("failed to add a job to work queue")
		}
	}

	// wait for all jobs to finish
	wg.Wait()

	// close resultsCh and errCh
	close(resultsCh)
	close(errCh)

	// read all from results channel
	var data [][]byte
	var enc []string
	if l := len(resultsCh); l > 0 {
		data = make([][]byte, 0, l)
		enc = make([]string, 0, l)
	}
	for res := range resultsCh {
		data = append(data, res.data)
		enc = append(enc, res.enc)
	}

	fnErrs := make([]error, 0, len(errCh))
	for err := range errCh {
		fnErrs = append(fnErrs, err)
	}

	return data, enc, fnErrs, nil
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
	if data != nil {
		// Commenting out job cancellations for now because of a resource leak suspected in the GCS golang client.
		// Issue logged here: https://github.com/googleapis/google-cloud-go/issues/3018
		// job.cancel()
		select {
		case job.resultsCh <- result{
			data: data,
			enc:  enc,
		}:
		default: // if we hit default it means that something else already returned a good result.  /shrug
		}
	}
	// todo: return result or err, not both
	if err != nil {
		job.err <- err
	}
}
