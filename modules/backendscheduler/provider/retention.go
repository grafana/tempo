package provider

import (
	"context"
	"flag"
	"sync"
	"time"

	kitlogger "github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/google/uuid"
	"github.com/grafana/tempo/modules/backendscheduler/work"
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
	cfg    RetentionConfig
	store  TenantLister
	sched  Scheduler
	logger kitlogger.Logger

	// pendingChannelTenants tracks tenants whose retention job has been sent to
	// the jobs channel but has not yet been picked up by Next() and recorded in
	// the work cache.  This mirrors the compaction provider's outstandingJobs
	// cache: jobs in the channel are invisible to ListJobs(), so without this we
	// could re-emit a job for the same tenant on the very next tick.
	//
	// Entries are added when a job is sent and removed in runningRetentionState()
	// when the corresponding job ID appears in ListJobs().
	pendingChannelMu      sync.Mutex
	pendingChannelTenants map[string]string // tenant → job ID
}

func NewRetentionProvider(cfg RetentionConfig, logger kitlogger.Logger, store TenantLister, scheduler Scheduler) *RetentionProvider {
	return &RetentionProvider{
		cfg:                   cfg,
		store:                 store,
		sched:                 scheduler,
		logger:                logger,
		pendingChannelTenants: make(map[string]string),
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
	// Snapshot running-tenant state once.  This also evicts tenants from
	// pendingChannelTenants whose jobs have now been recorded in the work cache.
	runningTenants, pendingTenants, legacyRunning := p.runningRetentionState()
	if legacyRunning {
		return
	}

	for _, tenantID := range p.store.Tenants() {
		if _, running := runningTenants[tenantID]; running {
			continue
		}
		if _, pending := pendingTenants[tenantID]; pending {
			// Job for this tenant is already in the channel awaiting pick-up.
			continue
		}
		// Re-check pending redaction right before the blocking send so that
		// redaction jobs added after the snapshot are still caught here.
		if p.sched.HasPendingJobs(tenantID, tempopb.JobType_JOB_TYPE_REDACTION) {
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

		// Record in the pending-channel set before the send so that the
		// next tick sees this tenant as already having a job in flight.
		p.pendingChannelMu.Lock()
		p.pendingChannelTenants[tenantID] = job.ID
		p.pendingChannelMu.Unlock()

		select {
		case jobs <- job:
		case <-ctx.Done():
			// Remove from pending set on cancellation — the job was never consumed.
			p.pendingChannelMu.Lock()
			delete(p.pendingChannelTenants, tenantID)
			p.pendingChannelMu.Unlock()
			return
		}
	}
}

// runningRetentionState returns:
//   - runningTenants: set of tenants with a currently-running retention job in
//     the work cache.
//   - pendingTenants: snapshot of pendingChannelTenants after evicting any
//     entries whose job has been recorded in the work cache (i.e. picked up by
//     Next()).
//   - legacyRunning: true if a global (tenant="") retention job is in flight,
//     which indicates an old scheduler binary; we pause emitting per-tenant jobs
//     until it clears to avoid double-running retention during a rollout.
func (p *RetentionProvider) runningRetentionState() (runningTenants map[string]struct{}, pendingTenants map[string]string, legacyRunning bool) {
	runningTenants = make(map[string]struct{})

	// Collect job IDs that are now in the work cache so we can evict them from
	// pendingChannelTenants.
	recordedJobIDs := make(map[string]struct{})

	for _, j := range p.sched.ListJobs() {
		if j.GetType() != tempopb.JobType_JOB_TYPE_RETENTION {
			continue
		}
		recordedJobIDs[j.ID] = struct{}{}
		switch j.GetStatus() {
		case tempopb.JobStatus_JOB_STATUS_RUNNING, tempopb.JobStatus_JOB_STATUS_UNSPECIFIED:
			tenant := j.Tenant()
			if tenant == "" {
				// A legacy global retention job is in flight; wait for it to finish
				// before emitting per-tenant jobs to avoid double-running retention
				// during a rollout.
				legacyRunning = true
				return
			}
			runningTenants[tenant] = struct{}{}
		}
	}

	// Evict pendingChannelTenants entries that have been recorded in the work
	// cache (Next() has picked them up).  Return a copy so the caller can check
	// without holding the lock.
	p.pendingChannelMu.Lock()
	for tenantID, jobID := range p.pendingChannelTenants {
		if _, recorded := recordedJobIDs[jobID]; recorded {
			delete(p.pendingChannelTenants, tenantID)
		}
	}
	pendingTenants = make(map[string]string, len(p.pendingChannelTenants))
	for k, v := range p.pendingChannelTenants {
		pendingTenants[k] = v
	}
	p.pendingChannelMu.Unlock()

	return
}
