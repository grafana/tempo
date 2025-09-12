package provider

import (
	"container/heap"
	"context"
	"flag"
	"sync"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/google/uuid"
	"github.com/grafana/tempo/modules/backendscheduler/work"
	"github.com/grafana/tempo/modules/backendscheduler/work/tenantselector"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/modules/storage"
	"github.com/grafana/tempo/pkg/boundedwaitgroup"
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
	MeasureInterval      time.Duration           `yaml:"measure_interval"`
	Compactor            tempodb.CompactorConfig `yaml:"compaction"`
	MaxJobsPerTenant     int                     `yaml:"max_jobs_per_tenant"`
	MinInputBlocks       int                     `yaml:"min_input_blocks"`
	MaxInputBlocks       int                     `yaml:"max_input_blocks"`
	MaxCompactionLevel   int                     `yaml:"max_compaction_level"`
	MinCycleInterval     time.Duration           `yaml:"min_cycle_interval"`
	MaxConcurrentTenants int                     `yaml:"max_concurrent_tenants"`
}

func (cfg *CompactionConfig) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	f.DurationVar(&cfg.MeasureInterval, prefix+"backend-scheduler.compaction-provider.measure-interval", time.Minute, "Interval at which to metric tenant blocklist")
	f.IntVar(&cfg.MaxJobsPerTenant, prefix+"backend-scheduler.max-jobs-per-tenant", 1000, "Maximum number of jobs to run per tenant before moving on to the next tenant")

	// Compaction
	f.IntVar(&cfg.MinInputBlocks, prefix+".min-input-blocks", blockselector.DefaultMinInputBlocks, "Minimum number of blocks to compact in a single job.")
	f.IntVar(&cfg.MaxInputBlocks, prefix+".max-input-blocks", blockselector.DefaultMaxInputBlocks, "Maximum number of blocks to compact in a single job.")
	f.IntVar(&cfg.MaxCompactionLevel, prefix+".max-compaction-level", blockselector.DefaultMaxCompactionLevel, "Maximum compaction level to include in compaction jobs. 0 means no limit.")

	// Tenant prioritization
	f.DurationVar(&cfg.MinCycleInterval, prefix+".min-cycle-interval", 30*time.Second, "Minimum time between tenant prioritization cycles to prevent excessive CPU usage when no work is available.")

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
	priority           *tenantselector.PriorityQueue
	lastPrioritizeTime time.Time

	// Recent jobs cache for duplicate block ID prevention.
	outstandingJobs    map[string][]backend.UUID
	outstandingJobsMtx sync.Mutex
}

func NewCompactionProvider(
	cfg CompactionConfig,
	logger log.Logger,
	store storage.Store,
	overrides overrides.Interface,
	scheduler Scheduler,
) *CompactionProvider {
	return &CompactionProvider{
		cfg:             cfg,
		logger:          logger,
		store:           store,
		overrides:       overrides,
		priority:        tenantselector.NewPriorityQueue(),
		sched:           scheduler,
		outstandingJobs: make(map[string][]backend.UUID),
	}
}

func (p *CompactionProvider) Start(ctx context.Context) <-chan *work.Job {
	var (
		// jobs is the main output channel for jobs created by this provider
		jobs = make(chan *work.Job, 1)
		// instanceJobs is the channel used by instances to send jobs to the main jobs channel
		instanceJobs = make(chan *work.Job, 1)
	)

	go func() {
		defer close(jobs)

		level.Info(p.logger).Log("msg", "compaction provider started")

		level.Info(p.logger).Log("msg", "compaction provider waiting for poll notification")
		<-p.store.PollNotification(ctx)

		var job *work.Job

		for {
			if ctx.Err() != nil {
				level.Info(p.logger).Log("msg", "compaction provider stopping")
				return
			}

			select {
			case <-ctx.Done():
			case job = <-instanceJobs:
				if job == nil {
					metricTenantEmptyJob.Inc()
					continue
				}
			}

			// Job successfully created, add to recent jobs cache before we send it.
			p.addToRecentJobs(ctx, job)

			select {
			case <-ctx.Done():
				level.Info(p.logger).Log("msg", "compaction provider stopping")
				return
			case jobs <- job:
			}
		}
	}()

	tenantCh := make(chan *tenantselector.Item, 1)

	// Manage the instances which operate on a tenant
	go func() {
		var (
			wg     = boundedwaitgroup.New(uint(p.cfg.MaxConcurrentTenants))
			tenant *tenantselector.Item
		)

		for {
			select {
			case <-ctx.Done():
				level.Info(p.logger).Log("msg", "compaction provider instance manager stopping")
				wg.Wait()
				return
			case tenant = <-tenantCh:
				if tenant == nil {
					continue
				}

				// Merge instance jobs into main jobs channel
				wg.Add(1)
				go func() {
					defer wg.Done()

					tenantID := tenant.Value()

					level.Info(p.logger).Log("msg", "starting compaction instance for tenant", "tenant_id", tenantID)

					var (
						selector, _ = p.newBlockSelector(tenantID)
						i           = newInstance(tenantID, selector, p.cfg, p.logger)
						ch          = i.run(ctx)
					)

					for {
						select {
						case <-ctx.Done():
							level.Debug(p.logger).Log("msg", "compaction provider instance job merger stopping")
							return
						case job, ok := <-ch:
							if !ok {
								level.Info(p.logger).Log("msg", "compaction provider instance job channel closed")
								return
							}

							select {
							case <-ctx.Done():
								level.Info(p.logger).Log("msg", "compaction provider instance job merger stopping")
								return
							case instanceJobs <- job:
								metricJobsCreated.WithLabelValues(tenantID).Inc()
							}
						}
					}
				}()
			}
		}
	}()

	go func() {
		defer close(tenantCh)

		for {
			select {
			case <-ctx.Done():
				return
			case <-p.store.PollNotification(ctx):
				p.prioritizeTenants(ctx)
			case tenantCh <- p.getNextTenant(ctx):
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

func (p *CompactionProvider) prepareNextTenant(ctx context.Context, drained bool) bool {
	_, span := tracer.Start(ctx, "prepareNextTenant")
	defer span.End()

	if p.priority.Len() == 0 {
		// Rate limit calls to prioritizeTenants to prevent excessive CPU usage
		// when cycling through tenants with no available work.
		//
		// We only expect new work for tenants after a the next blocklist poll.  If
		// we have been drained, wait for the next poll.

		if drained {
			level.Debug(p.logger).Log("msg", "rate limiting tenant prioritization; waiting for next poll")
			select {
			case <-ctx.Done():
				return false
			case <-p.store.PollNotification(ctx):
				// We waited for the poll, but we may have been cancelled in the meantime.
				if ctx.Err() != nil {
					return false
				}

				// Continue to prioritizeTenants
			}
		}

		if elapsed := time.Since(p.lastPrioritizeTime); elapsed < p.cfg.MinCycleInterval {
			level.Debug(p.logger).Log("msg", "rate limiting tenant prioritization; waiting")
			time.Sleep(p.cfg.MinCycleInterval - elapsed)

			// Continue to prioritizeTenants
		}

		p.prioritizeTenants(ctx)
		p.lastPrioritizeTime = time.Now()
		if p.priority.Len() == 0 {
			return false
		}
	}

	return true
}

func (p *CompactionProvider) getNextTenant(ctx context.Context) *tenantselector.Item {
	_, span := tracer.Start(ctx, "getNextTenant")
	defer span.End()

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			tenant := heap.Pop(p.priority).(*tenantselector.Item)
			if tenant != nil {
				return tenant
			}

			//  TODO: If the tenant is nil, then we need to wait for the
			//  prioritization to happen, which is on every poll.  Should we wait for
			//  poll here also?  This seems insufficient, because if we have
			//  exhausted all tenants, we will just spin.  We need to wait for
			//  either a poll or a timer.

			// TODO: we need to reimplement the drain logic.  If we have not
			// completely drained the all tenants, but skipped some because we
			// exceeded the max jobs, then we should come back to that tenant, but
			// the only way to do that is to reprioritize.

			// TODO: this is to say, that if we try to prioritiuze right here, and
			// then read a tenant, but we are still nil, then we need to wait for the
			// next poll.

			time.Sleep(100 * time.Millisecond)
		}
	}
}

// prioritizeTenants prioritizes tenants based on the number of outstanding blocks.
func (p *CompactionProvider) prioritizeTenants(ctx context.Context) {
	tenants := []tenantselector.Tenant{}

	_, span := tracer.Start(ctx, "prioritizeTenants")
	defer span.End()

	p.priority = tenantselector.NewPriorityQueue() // wipe and restart

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
			heap.Push(p.priority, item)
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
		inProgressBlockIDs = make(map[backend.UUID]struct{})
		bid                backend.UUID
		err                error
	)

	for _, job := range p.sched.ListJobs() {
		// Clean up recent jobs cache when we find a job which has been persisted
		p.outstandingJobsMtx.Lock()
		delete(p.outstandingJobs, job.ID)
		p.outstandingJobsMtx.Unlock()

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
			inProgressBlockIDs[bid] = struct{}{}
		}
	}

	// Also include blocks from recent jobs cache to prevent duplicate job creation
	p.outstandingJobsMtx.Lock()
	for _, recentBlockIDs := range p.outstandingJobs {
		for _, blockID := range recentBlockIDs {
			inProgressBlockIDs[blockID] = struct{}{}
		}
	}
	p.outstandingJobsMtx.Unlock()

	for _, block := range fullBlocklist {
		if _, ok := inProgressBlockIDs[block.BlockID]; !ok {
			// Include blocks that are not already in the input list from another job or recent jobs
			blocklist = append(blocklist, block)
		}
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
		p.cfg.MaxCompactionLevel,
	), len(blocklist)
}

// addToRecentJobs adds a job to the recent jobs cache
func (p *CompactionProvider) addToRecentJobs(ctx context.Context, job *work.Job) {
	_, span := tracer.Start(ctx, "addToRecentJobs")
	defer span.End()

	if job.Type != tempopb.JobType_JOB_TYPE_COMPACTION || job.JobDetail.Compaction == nil {
		return
	}

	// Don't cache jobs with empty input
	if len(job.JobDetail.Compaction.Input) == 0 {
		return
	}

	// Copy the input block IDs
	blockIDs := make([]backend.UUID, len(job.JobDetail.Compaction.Input))
	for i, blockID := range job.JobDetail.Compaction.Input {
		bid, err := backend.ParseUUID(blockID)
		if err != nil {
			level.Error(p.logger).Log("msg", "failed to parse block ID", "block_id", blockID, "err", err)
			return
		}
		blockIDs[i] = bid
	}

	p.outstandingJobsMtx.Lock()
	p.outstandingJobs[job.ID] = blockIDs
	p.outstandingJobsMtx.Unlock()
}

// TODO: above, we have a single loop which operates on a single tenant at a
// time.  We could improve this situation by breaking this up into multiple
// instance, where an instance writes a job into a channel to be merged by the
// single output channel as above.  Ideally we have some flexibiilty in how the
// instances are managed such that perhaps a large tenant continues to drain,
// while some of the smaller tenants are exhausted.  If an instance drains a
// tenant, we shut it down and start a new instance.  The number of instances
// is limited by the max concurrent tenants config using the blocking wait
// group.  I'm immagining a small loop somewhere which manages the instances
// and when one dies, a new tenant is selected and and instance is started.

// TODO: its not clear to me if the above should write directly into the output
// channel, or if we should have an intermediate channel per instance which is
// merged into the output channel.  Presumably, if the instance is writing to
// an intermediate, a new kind of object could be chosen, and the job itself
// would be created in the merger.  This would allow us to have a smaller
// object passed around between goroutines, and the job creation (which
// includes a UUID generation) would be done in a single goroutine.  This might
// be better for performance.

type tenantDrainer interface {
	run(ctx context.Context) <-chan *work.Job
}

var _ tenantDrainer = (*compactionInstance)(nil)

// compactionInstance operates on a single tenant to push compaction jobs to the scheduler.
type compactionInstance struct {
	tenant   string
	selector blockselector.CompactionBlockSelector
	cfg      CompactionConfig
	logger   log.Logger
}

func newInstance(
	tenantID string,
	selector blockselector.CompactionBlockSelector,
	cfg CompactionConfig,
	logger log.Logger,
) *compactionInstance {
	i := &compactionInstance{
		tenant:   tenantID,
		selector: selector,
		cfg:      cfg,
		logger:   log.With(logger, "tenant_id", tenantID),
	}

	return i
}

func (i *compactionInstance) run(ctx context.Context) <-chan *work.Job {
	_, span := tracer.Start(ctx, "compactionInstance.run")
	defer span.End()

	instanceJobs := make(chan *work.Job)

	go func() {
		defer close(instanceJobs)

		level.Debug(i.logger).Log("msg", "compaction instance started")

		jobCount := 0

		for {
			select {
			case <-ctx.Done():
				level.Debug(i.logger).Log("msg", "compaction instance stopping")
				span.AddEvent("context done")
				return
			default:
				job := i.createJob(ctx)
				if job == nil {
					span.AddEvent("tenant exhausted", trace.WithAttributes(
						attribute.String("tenant_id", i.tenant),
					))
					return
				}

				if jobCount >= i.cfg.MaxJobsPerTenant {
					level.Info(i.logger).Log("msg", "max jobs per tenant reached, stopping instance")
					span.AddEvent("max jobs per tenant reached", trace.WithAttributes(
						attribute.String("tenant_id", i.tenant),
						attribute.Int("job_count", jobCount),
					))
					return
				}

				select {
				case <-ctx.Done():
					level.Debug(i.logger).Log("msg", "compaction instance stopping")
					span.AddEvent("context done")
					return
				case instanceJobs <- job:
					span.AddEvent("job created", trace.WithAttributes(
						attribute.String("job_id", job.ID),
						attribute.String("tenant_id", i.tenant),
					))

					metricJobsCreated.WithLabelValues(i.tenant).Inc()

					jobCount++
				}
			}
		}
	}()

	return instanceJobs
}

func (i *compactionInstance) createJob(ctx context.Context) *work.Job {
	_, span := tracer.Start(ctx, "compactionInstance.createJob")
	defer span.End()

	span.SetAttributes(attribute.String("tenant_id", i.tenant))

	input, ok := i.getNextBlockIDs(ctx)
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
			Tenant:     i.tenant,
			Compaction: &tempopb.CompactionDetail{Input: input},
		},
	}
}

func (i *compactionInstance) getNextBlockIDs(_ context.Context) ([]string, bool) {
	ids := make([]string, 0, i.cfg.MaxInputBlocks)

	toBeCompacted, _ := i.selector.BlocksToCompact()

	if len(toBeCompacted) == 0 {
		return nil, false
	}

	for _, b := range toBeCompacted {
		ids = append(ids, b.BlockID.String())
	}

	return ids, len(ids) >= i.cfg.MinInputBlocks
}
