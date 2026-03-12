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
	// RescanDelay is how long to wait after submission before rescanning for output blocks
	// produced by compaction jobs that were active at submission time. Should be set to at
	// least 2× the tempodb blocklist_poll interval so the new blocks are visible. The
	// scheduler's work.prune_age must exceed this value.
	RescanDelay time.Duration `yaml:"rescan_delay"`
	// MaxRescanGenerations limits how many times the rescan loop may re-arm itself when
	// newly-produced output blocks are themselves still being compacted. Each generation
	// waits one RescanDelay before retrying.
	MaxRescanGenerations int `yaml:"max_rescan_generations"`
}

func (cfg *RedactionConfig) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	f.DurationVar(&cfg.PollInterval, prefix+"backend-scheduler.redaction-provider.poll-interval", 2*time.Second, "Interval at which to check for pending redaction jobs when the queue is empty")
	f.DurationVar(&cfg.RescanDelay, prefix+"backend-scheduler.redaction-provider.rescan-delay", 5*time.Minute, "How long to wait before rescanning for output blocks from compaction jobs active at submission time")
	f.IntVar(&cfg.MaxRescanGenerations, prefix+"backend-scheduler.redaction-provider.max-rescan-generations", 5, "Maximum number of rescan generations before giving up and requiring operator resubmission")
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
				job := p.sched.NextPendingJob(tempopb.JobType_JOB_TYPE_REDACTION)
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
