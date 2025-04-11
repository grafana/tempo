package provider

import (
	"container/heap"
	"context"
	"flag"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/google/uuid"
	"github.com/grafana/dskit/backoff"
	"github.com/grafana/tempo/modules/backendscheduler/work"
	"github.com/grafana/tempo/modules/backendscheduler/work/tenantselector"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/modules/storage"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/tempodb"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/blockselector"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("modules/backendscheduler/provider/compaction")

type CompactionConfig struct {
	MeasureInterval  time.Duration           `yaml:"measure_interval"`
	Compactor        tempodb.CompactorConfig `yaml:"compaction"`
	MaxJobsPerTenant int                     `yaml:"max_jobs_per_tenant"`
	Backoff          backoff.Config          `yaml:"backoff"`
}

func (cfg *CompactionConfig) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	f.DurationVar(&cfg.MeasureInterval, prefix+"backend-scheduler.compaction-provider.measure-interval", time.Minute, "Interval at which to metric tenant blocklist")
	f.IntVar(&cfg.MaxJobsPerTenant, prefix+"backend-scheduler.max-jobs-per-tenant", 1000, "Maximum number of jobs to run per tenant before moving on to the next tenant")

	// Backoff
	f.DurationVar(&cfg.Backoff.MinBackoff, prefix+".backoff-min-period", 100*time.Millisecond, "Minimum delay when backing off.")
	f.DurationVar(&cfg.Backoff.MaxBackoff, prefix+".backoff-max-period", 10*time.Second, "Maximum delay when backing off.")
	f.IntVar(&cfg.Backoff.MaxRetries, prefix+".backoff-retries", 0, "Number of times to backoff and retry before failing.")

	cfg.Compactor = tempodb.CompactorConfig{}
	cfg.Compactor.RegisterFlagsAndApplyDefaults(util.PrefixConfig(prefix, "compaction"), f)
}

type CompactionProvider struct {
	cfg    CompactionConfig
	logger log.Logger

	// Dependencies needed for compaction job selection
	store     storage.Store
	overrides overrides.Interface

	// Scheduler calls required for this provider
	sched Scheduler

	// Dependencies needed for tenant selection
	curPriority       *tenantselector.PriorityQueue
	curTenant         *tenantselector.Item
	curSelector       blockselector.CompactionBlockSelector
	curTenantJobCount int
}

func NewCompactionProvider(
	cfg CompactionConfig,
	logger log.Logger,
	store storage.Store,
	overrides overrides.Interface,
	scheduler Scheduler,
) *CompactionProvider {
	return &CompactionProvider{
		cfg:         cfg,
		logger:      logger,
		store:       store,
		overrides:   overrides,
		curPriority: tenantselector.NewPriorityQueue(),
		sched:       scheduler,
	}
}

func (p *CompactionProvider) Start(ctx context.Context) <-chan *work.Job {
	jobs := make(chan *work.Job, 1)

	go func() {
		defer close(jobs)

		measureTicker := time.NewTicker(p.cfg.MeasureInterval)
		defer measureTicker.Stop()

		level.Info(p.logger).Log("msg", "compaction provider started")

		var (
			job *work.Job
			b   = backoff.New(ctx, p.cfg.Backoff)
		)

		reset := func() {
			job = nil
			b.Reset()
		}

		for {
			select {
			case <-ctx.Done():
				level.Info(p.logger).Log("msg", "compaction provider stopping")
				return
			case <-measureTicker.C:
				// Measure the tenants to get their current compaction status
				p.measureTenants()
			default:
				ctx, span := tracer.Start(ctx, "compaction-provider-poll")

				attributes := []attribute.KeyValue{
					attribute.Int("max_jobs_per_tenant", p.cfg.MaxJobsPerTenant),
					attribute.Int("jobs_in_queue", len(jobs)),
					attribute.Int("jobs_in_scheduler", len(p.sched.ListJobs())),
				}

				span.SetAttributes(attributes...)

				if job == nil {
					// we don't have a job, get the next one
					job = p.nextCompactionJob(ctx)
					if job == nil {
						span.AddEvent("backoff")
						b.Wait()
					}
				}

				if job != nil {
					span.SetAttributes(attribute.String("job_id", job.ID))
					// we have a job that we need to send
					select {
					case jobs <- job:
						// job sent, reset the job
						span.AddEvent("job-sent")
						reset()
					default:
						// Channel full, try again next tick

						span.SetAttributes(
							attribute.String("job_id", job.ID),
							attribute.String("job_type", job.Type.String()),
							attribute.Int("jobs_in_queue", len(jobs)),
							attribute.Int("jobs_in_scheduler", len(p.sched.ListJobs())),
						)

						span.AddEvent("channel-full")
						b.Wait()
					}
				}
				span.End()
			}
		}
	}()

	return jobs
}

// returns the next compaction job for the tenant
func (p *CompactionProvider) nextCompactionJob(ctx context.Context) *work.Job {
	resetCount := 0

	ctx, span := tracer.Start(ctx, "next-compaction-job")
	defer span.End()

	reset := func() {
		resetCount++
		p.curSelector = nil
		p.curTenant = nil
		p.curTenantJobCount = 0
		span.AddEvent("reset")
	}

	// Only reset up to the number of tenants to ensure we visit them all at least once.
	for resetCount <= len(p.store.Tenants()) {
		// do we have an current tenant?
		if p.curSelector == nil {
			if !p.prepareNextTenant(ctx) {
				return nil
			}
			continue
		}

		if p.curTenantJobCount >= p.cfg.MaxJobsPerTenant {
			reset()
			continue
		}

		job := p.createJob(ctx)
		if job == nil {
			// we don't have a job, reset the curTenant
			// and try again
			span.AddEvent("no-job-created")
			reset()
			continue
		}

		if job != nil {
			// we have a job, return it
			span.SetAttributes(attribute.String("job_id", job.ID))
			span.AddEvent("job-created")
			return job
		}

	}

	return nil
}

func (p *CompactionProvider) prepareNextTenant(ctx context.Context) bool {
	if p.curPriority.Len() == 0 {
		p.prioritizeTenants(ctx)
		if p.curPriority.Len() == 0 {
			return false
		}
	}

	p.curTenant = heap.Pop(p.curPriority).(*tenantselector.Item)
	if p.curTenant == nil {
		return false
	}

	p.curSelector, _ = p.newBlockSelector(p.curTenant.Value())
	return true
}

func (p *CompactionProvider) createJob(ctx context.Context) *work.Job {
	_, span := tracer.Start(ctx, "create-job")
	defer span.End()

	span.SetAttributes(attribute.String("tenant_id", p.curTenant.Value()))

	input, ok := p.getNextBlockIDs(ctx)
	if !ok {
		span.AddEvent("not-enough-input-blocks", trace.WithAttributes(
			attribute.Int("input_blocks", len(input)),
		))
		return nil
	}

	p.curTenantJobCount++
	return &work.Job{
		ID:   uuid.New().String(),
		Type: tempopb.JobType_JOB_TYPE_COMPACTION,
		JobDetail: tempopb.JobDetail{
			Tenant:     p.curTenant.Value(),
			Compaction: &tempopb.CompactionDetail{Input: input},
		},
	}
}

func (p *CompactionProvider) getNextBlockIDs(_ context.Context) ([]string, bool) {
	ids := make([]string, 0, blockselector.DefaultMaxInputBlocks)

	toBeCompacted, _ := p.curSelector.BlocksToCompact()

	if len(toBeCompacted) == 0 {
		return nil, false
	}

	for _, b := range toBeCompacted {
		if p.sched.HasBlocks([]string{b.BlockID.String()}) {
			continue
		}

		ids = append(ids, b.BlockID.String())
	}

	return ids, len(ids) >= blockselector.DefaultMinInputBlocks
}

// prioritizeTenants prioritizes tenants based on the number of outstanding blocks.
func (p *CompactionProvider) prioritizeTenants(ctx context.Context) {
	tenants := []tenantselector.Tenant{}

	_, span := tracer.Start(ctx, "prioritize-tenants")
	defer span.End()

	p.curPriority = tenantselector.NewPriorityQueue() // wipe and restart

	for _, tenantID := range p.store.Tenants() {
		if p.overrides.CompactionDisabled(tenantID) {
			continue
		}

		var (
			outstandingBlocks           = 0
			toBeCompacted               []*backend.BlockMeta
			blockSelector, blocklistLen = p.newBlockSelector(tenantID)
		)

		// Measure the outstanding blocks
		for {
			toBeCompacted, _ = blockSelector.BlocksToCompact()
			if len(toBeCompacted) == 0 {
				span.AddEvent("no-more-blocks-to-compact", trace.WithAttributes(
					attribute.String("tenant_id", tenantID),
					attribute.Int("outstanding_blocks", len(toBeCompacted)),
				))
				break
			}

			outstandingBlocks += len(toBeCompacted)

			span.AddEvent("found-blocks-to-compact", trace.WithAttributes(
				attribute.String("tenant_id", tenantID),
				attribute.Int("outstanding_blocks", len(toBeCompacted)),
			))
		}

		tenants = append(tenants, tenantselector.Tenant{
			ID:                         tenantID,
			BlocklistLength:            blocklistLen,
			OutstandingBlocklistLength: outstandingBlocks,
		})
	}

	var (
		ts       = tenantselector.NewBlockListWeightedTenantSelector(tenants)
		item     *tenantselector.Item
		priority int
	)

	for _, tenant := range tenants {
		priority = ts.PriorityForTenant(tenant.ID)
		item = tenantselector.NewItem(tenant.ID, priority)
		heap.Push(p.curPriority, item)
	}
}

func (p *CompactionProvider) measureTenants() {
	for _, tenant := range p.store.Tenants() {
		blockSelector, _ := p.newBlockSelector(tenant)

		yes := func(_ string) bool {
			return true
		}

		tempodb.MeasureOutstandingBlocks(tenant, blockSelector, yes)
	}
}

func (p *CompactionProvider) newBlockSelector(tenantID string) (blockselector.CompactionBlockSelector, int) {
	var (
		fullBlocklist = p.store.BlockMetas(tenantID)
		window        = p.overrides.MaxCompactionRange(tenantID)
		blocklist     = make([]*backend.BlockMeta, 0, len(fullBlocklist))
	)

	// NOTE: we want to skip blocks which have already been oeprated on based on
	// the scheduler.  This is required because the blocklist may not have been
	// updated yet.
	for _, b := range fullBlocklist {
		if p.sched.HasBlocks([]string{b.BlockID.String()}) {
			continue
		}

		blocklist = append(blocklist, b)
	}

	if window == 0 {
		window = p.cfg.Compactor.MaxCompactionRange
	}

	return blockselector.NewTimeWindowBlockSelector(
		blocklist,
		window,
		p.cfg.Compactor.MaxCompactionObjects,
		p.cfg.Compactor.MaxBlockBytes,
		blockselector.DefaultMinInputBlocks,
		blockselector.DefaultMaxInputBlocks,
	), len(blocklist)
}
