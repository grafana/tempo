package provider

import (
	"context"
	"flag"
	"time"

	kitlogger "github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/google/uuid"
	"github.com/grafana/tempo/modules/backendscheduler/work"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/tempopb"
)

// TenantLister provides the list of tenants known to the blocklist.
// storage.Store satisfies this interface.
type TenantLister interface {
	Tenants() []string
}

type RetentionConfig struct {
	Interval time.Duration `yaml:"interval"`
}

func (cfg *RetentionConfig) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	f.DurationVar(&cfg.Interval, prefix+"backend-scheduler.retention-interval", time.Hour, "Interval at which to perform tenant retention")
}

type RetentionProvider struct {
	cfg       RetentionConfig
	store     TenantLister
	overrides overrides.Interface
	sched     Scheduler
	logger    kitlogger.Logger
}

func NewRetentionProvider(cfg RetentionConfig, logger kitlogger.Logger, store TenantLister, overrides overrides.Interface, scheduler Scheduler) *RetentionProvider {
	return &RetentionProvider{
		cfg:       cfg,
		store:     store,
		overrides: overrides,
		sched:     scheduler,
		logger:    logger,
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
				p.emitRetentionJobs(ctx, jobs)
			}
		}
	}()

	return jobs
}

// emitRetentionJobs sends one retention job per eligible tenant into jobs.
// It uses a blocking send per job so all tenants are served within one tick
// period regardless of downstream backpressure.
func (p *RetentionProvider) emitRetentionJobs(ctx context.Context, jobs chan<- *work.Job) {
	// A legacy global retention job (empty tenant) means an old binary is still
	// running; hold off on per-tenant jobs until the rollout completes.
	if p.sched.HasJobsForTenant("", tempopb.JobType_JOB_TYPE_RETENTION) {
		return
	}

	for _, tenantID := range p.store.Tenants() {
		if p.overrides.CompactionDisabled(tenantID) {
			continue
		}
		// HasJobsForTenant covers pending queue, registered, and active — so this
		// catches jobs in the channel gap after RegisterJob.
		if p.sched.HasJobsForTenant(tenantID, tempopb.JobType_JOB_TYPE_RETENTION) {
			continue
		}
		if p.sched.HasJobsForTenant(tenantID, tempopb.JobType_JOB_TYPE_REDACTION) {
			level.Debug(p.logger).Log("msg", "skipping retention for tenant with pending redaction", "tenant", tenantID)
			continue
		}

		job := &work.Job{
			ID:   uuid.New().String(),
			Type: tempopb.JobType_JOB_TYPE_RETENTION,
			JobDetail: tempopb.JobDetail{
				Tenant:    tenantID,
				Retention: &tempopb.RetentionDetail{},
			},
		}

		// Register before the send so HasJobsForTenant returns true during the
		// channel gap. Cleared automatically by AddJob when promoted to active.
		// Mirrors the compaction provider pattern.
		p.sched.RegisterJob(job)

		select {
		case jobs <- job:
		case <-ctx.Done():
			return
		}
	}
}
