package provider

import (
	"context"
	"flag"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/grafana/tempo/modules/backendscheduler/work"
	"github.com/grafana/tempo/pkg/tempopb"
)

// RedactionConfig holds configuration for the redaction provider.
type RedactionConfig struct {
	// PollInterval is how often to try popping a pending redaction job when the queue is empty.
	PollInterval time.Duration `yaml:"poll_interval"`
}

func (cfg *RedactionConfig) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	f.DurationVar(&cfg.PollInterval, prefix+"backend-scheduler.redaction-provider.poll-interval", 2*time.Second, "Interval at which to check for pending redaction jobs when the queue is empty")
}

// RedactionProvider drains the pending redaction queue and sends jobs to the scheduler.
type RedactionProvider struct {
	cfg    RedactionConfig
	sched  Scheduler
	logger log.Logger
}

// NewRedactionProvider returns a provider that pops pending redaction jobs and feeds them to the merged job channel.
func NewRedactionProvider(cfg RedactionConfig, logger log.Logger, scheduler Scheduler) *RedactionProvider {
	return &RedactionProvider{
		cfg:    cfg,
		sched:  scheduler,
		logger: logger,
	}
}

// Start implements Provider. It drains the pending redaction queue at a manageable rate.
func (p *RedactionProvider) Start(ctx context.Context) <-chan *work.Job {
	jobs := make(chan *work.Job, 1)

	go func() {
		defer close(jobs)
		ticker := time.NewTicker(p.cfg.PollInterval)
		defer ticker.Stop()

		level.Info(p.logger).Log("msg", "redaction provider started")

		for {
			select {
			case <-ctx.Done():
				level.Info(p.logger).Log("msg", "redaction provider stopping")
				return
			default:
				job := p.sched.PopNextPendingJob(tempopb.JobType_JOB_TYPE_REDACTION)
				if job != nil {
					select {
					case jobs <- job:
					case <-ctx.Done():
						return
					}
					continue
				}
				// No job available; wait before trying again
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
				}
			}
		}
	}()

	return jobs
}
