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
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("modules/backendscheduler/provider/compaction")

type CompactionConfig struct {
	MeasureInterval  time.Duration           `yaml:"measure_interval"`
	Compactor        tempodb.CompactorConfig `yaml:"compaction"`
	MaxJobsPerTenant int                     `yaml:"max_jobs_per_tenant"`
	Backoff          backoff.Config          `yaml:"backoff"`
	MinInputBlocks   int                     `yaml:"min_input_blocks"`
	MaxInputBlocks   int                     `yaml:"max_input_blocks"`
}

func (cfg *CompactionConfig) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	f.DurationVar(&cfg.MeasureInterval, prefix+"backend-scheduler.compaction-provider.measure-interval", time.Minute, "Interval at which to metric tenant blocklist")
	f.IntVar(&cfg.MaxJobsPerTenant, prefix+"backend-scheduler.max-jobs-per-tenant", 1000, "Maximum number of jobs to run per tenant before moving on to the next tenant")

	// Compaction
	f.IntVar(&cfg.MinInputBlocks, prefix+".min-input-blocks", blockselector.DefaultMinInputBlocks, "Minimum number of blocks to compact in a single job.")
	f.IntVar(&cfg.MaxInputBlocks, prefix+".max-input-blocks", blockselector.DefaultMaxInputBlocks, "Maximum number of blocks to compact in a single job.")

	// Backoff
	f.DurationVar(&cfg.Backoff.MinBackoff, prefix+".backoff-min-period", 100*time.Millisecond, "Minimum delay when backing off.")
	f.DurationVar(&cfg.Backoff.MaxBackoff, prefix+".backoff-max-period", 10*time.Second, "Maximum delay when backing off.")
	cfg.Backoff.MaxRetries = 0

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
	curPriority *tenantselector.PriorityQueue
	curTenant   *tenantselector.Item
	curSelector blockselector.CompactionBlockSelector
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

		level.Info(p.logger).Log("msg", "compaction provider started")

		var (
			job               *work.Job
			b                 = backoff.New(ctx, p.cfg.Backoff)
			curTenantJobCount int
			span              trace.Span
			loopCtx           context.Context
			spanStarted       bool
		)

		reset := func() {
			metricTenantReset.WithLabelValues(p.curTenant.Value()).Inc()
			span.AddEvent("tenant reset", trace.WithAttributes(
				attribute.String("tenant_id", p.curTenant.Value()),
				attribute.Int("job_count", curTenantJobCount),
			))
			p.curSelector = nil
			p.curTenant = nil
			curTenantJobCount = 0
			span.End()
			spanStarted = false
		}

		for {
			if ctx.Err() != nil {
				level.Info(p.logger).Log("msg", "compaction provider stopping")
				return
			}

			if !spanStarted {
				loopCtx, span = tracer.Start(ctx, "compactionProviderLoop")
				spanStarted = true
			}

			if p.curSelector == nil {
				b.Wait()
				if !p.prepareNextTenant(loopCtx) {
					level.Info(p.logger).Log("msg", "received empty tenant", "waiting", b.NextDelay())
					metricTenantBackoff.Inc()
					span.AddEvent("tenant not prepared")
				}
				continue
			}

			if curTenantJobCount >= p.cfg.MaxJobsPerTenant {
				level.Info(p.logger).Log("msg", "max jobs per tenant reached, skipping to next tenant")
				span.AddEvent("max jobs per tenant reached")
				reset()
				continue
			}

			job = p.createJob(loopCtx)
			if job == nil {
				level.Info(p.logger).Log("msg", "tenant exhausted, skipping to next tenant after delay", "waiting", b.NextDelay())
				// we don't have a job, reset the curTenant and try again
				metricTenantEmptyJob.Inc()
				// Avoid CPU spin
				b.Wait()
				reset()
				continue
			}

			select {
			case <-ctx.Done():
				level.Info(p.logger).Log("msg", "compaction provider stopping")
				span.AddEvent("context done")
				span.End()
				return
			case jobs <- job:
				metricJobsCreated.WithLabelValues(p.curTenant.Value()).Inc()
				curTenantJobCount++
				b.Reset()
				span.AddEvent("job created", trace.WithAttributes(
					attribute.String("job_id", job.ID),
					attribute.String("tenant_id", p.curTenant.Value()),
				))
				span.End()
				spanStarted = false
			}
		}
	}()

	// Measure the tenants to get their current compaction status in a separate
	// goroutine to avoid blocking the main job loop.
	go func() {
		measureTicker := time.NewTicker(p.cfg.MeasureInterval)
		defer measureTicker.Stop()
		for {
			select {
			case <-ctx.Done():
				level.Info(p.logger).Log("msg", "compaction provider measure ticker stopping")
				return
			case <-measureTicker.C:
				p.measureTenants()
			}
		}
	}()

	return jobs
}

func (p *CompactionProvider) prepareNextTenant(ctx context.Context) bool {
	_, span := tracer.Start(ctx, "prepareNextTenant")
	defer span.End()

	if p.curPriority.Len() == 0 {
		p.prioritizeTenants(ctx)
		if p.curPriority.Len() == 0 {
			return false
		}
	}

	p.curTenant = heap.Pop(p.curPriority).(*tenantselector.Item)
	if p.curTenant == nil {
		span.AddEvent("no more tenants to compact")
		return false
	}

	p.curSelector, _ = p.newBlockSelector(p.curTenant.Value())
	return true
}

func (p *CompactionProvider) createJob(ctx context.Context) *work.Job {
	_, span := tracer.Start(ctx, "createJob")
	defer span.End()

	span.SetAttributes(attribute.String("tenant_id", p.curTenant.Value()))

	input, ok := p.getNextBlockIDs(ctx)
	if !ok {
		span.AddEvent("not-enough-input-blocks", trace.WithAttributes(
			attribute.Int("input_blocks", len(input)),
		))

		span.SetStatus(codes.Error, "not enough input blocks for compaction")
		return nil
	}

	span.AddEvent("input blocks selected", trace.WithAttributes(
		attribute.Int("input_blocks", len(input)),
		attribute.StringSlice("input_block_ids", input),
	))
	span.SetStatus(codes.Ok, "compaction job created")
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
	ids := make([]string, 0, p.cfg.MaxInputBlocks)

	toBeCompacted, _ := p.curSelector.BlocksToCompact()

	if len(toBeCompacted) == 0 {
		return nil, false
	}

	for _, b := range toBeCompacted {
		ids = append(ids, b.BlockID.String())
	}

	return ids, len(ids) >= p.cfg.MinInputBlocks
}

// prioritizeTenants prioritizes tenants based on the number of outstanding blocks.
func (p *CompactionProvider) prioritizeTenants(ctx context.Context) {
	tenants := []tenantselector.Tenant{}

	_, span := tracer.Start(ctx, "prioritizeTenants")
	defer span.End()

	p.curPriority = tenantselector.NewPriorityQueue() // wipe and restart

	var (
		blocklistLen      int
		blockSelector     blockselector.CompactionBlockSelector
		outstandingBlocks int
		toBeCompacted     []*backend.BlockMeta
	)

	for _, tenantID := range p.store.Tenants() {
		if p.overrides.CompactionDisabled(tenantID) {
			continue
		}

		outstandingBlocks = 0
		clear(toBeCompacted)

		blockSelector, blocklistLen = p.newBlockSelector(tenantID)

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

		if priority >= p.cfg.MinInputBlocks {
			item = tenantselector.NewItem(tenant.ID, priority)
			heap.Push(p.curPriority, item)
		}
	}
}

func (p *CompactionProvider) measureTenants() {
	_, span := tracer.Start(context.Background(), "measureTenants")
	defer span.End()

	owns := func(_ string) bool {
		return true
	}

	var blockSelector blockselector.CompactionBlockSelector
	for _, tenant := range p.store.Tenants() {
		blockSelector, _ = p.newBlockSelector(tenant)
		tempodb.MeasureOutstandingBlocks(tenant, blockSelector, owns)
	}
}

func (p *CompactionProvider) newBlockSelector(tenantID string) (blockselector.CompactionBlockSelector, int) {
	var (
		fullBlocklist = p.store.BlockMetas(tenantID)
		window        = p.overrides.MaxCompactionRange(tenantID)
		blocklist     = make([]*backend.BlockMeta, 0, len(fullBlocklist))
	)

	// Query the work for the jobs and build up a list of UUIDs which we can match against and skip on the selector.
	var (
		perTenantBlockIDs = make(map[backend.UUID]struct{})
		bid               backend.UUID
		err               error
	)

	for _, job := range p.sched.ListJobs() {
		if job.Tenant() != tenantID {
			continue
		}

		if job.GetType() == tempopb.JobType_JOB_TYPE_UNSPECIFIED {
			continue
		}

		// NOTE: We check the compaction job input, but not the output.  This is to
		// allow the output of one job to become the input of another.
		for _, blockID := range job.GetCompactionInput() {
			bid, err = backend.ParseUUID(blockID)
			if err != nil {
				level.Error(p.logger).Log("msg", "failed to parse block ID", "block_id", blockID, "err", err)
				continue
			}
			perTenantBlockIDs[bid] = struct{}{}
		}
	}

	for _, block := range fullBlocklist {
		if _, ok := perTenantBlockIDs[block.BlockID]; ok {
			// Skip blocks that are already in the input list from another job
			continue
		}

		blocklist = append(blocklist, block)
	}

	if window == 0 {
		window = p.cfg.Compactor.MaxCompactionRange
	}

	return blockselector.NewTimeWindowBlockSelector(
		blocklist,
		window,
		p.cfg.Compactor.MaxCompactionObjects,
		p.cfg.Compactor.MaxBlockBytes,
		p.cfg.MinInputBlocks,
		p.cfg.MaxInputBlocks,
	), len(blocklist)
}
