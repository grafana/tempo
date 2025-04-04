package provider

import (
	"context"
	"flag"
	"time"

	kitlogger "github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/google/uuid"
	"github.com/grafana/tempo/modules/backendscheduler/work"
	"github.com/grafana/tempo/pkg/tempopb"
)

type RetentionConfig struct {
	Interval time.Duration `yaml:"interval"`
}

func (cfg *RetentionConfig) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	f.DurationVar(&cfg.Interval, prefix+"backend-scheduler.retention-interval", time.Hour, "Interval at which to perform tenant retention")
}

type RetentionProvider struct {
	cfg    RetentionConfig
	sched  Scheduler
	logger kitlogger.Logger
}

func NewRetentionProvider(cfg RetentionConfig, logger kitlogger.Logger, scheduler Scheduler) *RetentionProvider {
	return &RetentionProvider{
		cfg:    cfg,
		sched:  scheduler,
		logger: logger,
	}
}

func (p *RetentionProvider) Start(ctx context.Context) <-chan *work.Job {
	jobs := make(chan *work.Job, 1)

	go func() {
		defer close(jobs)
		ticker := time.NewTicker(p.cfg.Interval)
		defer ticker.Stop()

		level.Info(p.logger).Log("msg", "retention provider started")

		for {
			select {
			case <-ctx.Done():
				level.Info(p.logger).Log("msg", "retention provider stopping")
				return
			case <-ticker.C:
				if job := p.nextRetentionJob(); job != nil {
					select {
					case jobs <- job:
					default:
						// Channel full, try again next tick
					}
				}
			}
		}
	}()

	return jobs
}

func (p *RetentionProvider) nextRetentionJob() *work.Job {
	// Check if we already have a retention job running
	for _, j := range p.sched.ListJobs() {
		switch j.GetType() {
		case tempopb.JobType_JOB_TYPE_RETENTION:
			switch j.GetStatus() {
			case tempopb.JobStatus_JOB_STATUS_RUNNING, tempopb.JobStatus_JOB_STATUS_UNSPECIFIED:
				return nil
			}
		}
	}

	return &work.Job{
		ID:   uuid.New().String(),
		Type: tempopb.JobType_JOB_TYPE_RETENTION,
		JobDetail: tempopb.JobDetail{
			Retention: &tempopb.RetentionDetail{},
		},
	}
}
