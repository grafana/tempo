package provider

import (
	"container/heap"
	"context"
	"flag"
	"sync"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
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
	"go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("modules/backendscheduler/provider/compaction")

type jobTracker interface {
	addToRecentJobs(ctx context.Context, job *work.Job) (added bool)
}

var _ jobTracker = (*CompactionProvider)(nil)

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
	f.IntVar(&cfg.MaxConcurrentTenants, prefix+"backend-scheduler.compaction-provider.max-concurrent-tenants", 1, "Maximum number of tenants to compact concurrently.")

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
	priorityMtx        sync.Mutex
	lastPrioritizeTime time.Time

	// Recent jobs cache for duplicate block ID prevention.
	outstandingJobs    map[string][]backend.UUID
	outstandingJobsMtx sync.Mutex

	// Keep track of active tenants
	activeTenants    map[string]struct{}
	activeTenantsMtx sync.Mutex
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
		activeTenants:   make(map[string]struct{}, cfg.MaxConcurrentTenants),
	}
}

func (p *CompactionProvider) Start(ctx context.Context) <-chan *work.Job {
	var (
		// jobs is the main output channel for jobs created by this provider
		jobs = make(chan *work.Job, 1)
		// instanceJobs is the channel used by instances to send jobs to the provider
		instanceJobs = make(chan *work.Job)
		// tenantCh is the channel used to send tenants to the instance manager
		tenantCh = make(chan *tenantselector.Item)
	)

	// Main job creation loop.  Jobs received from the instances are sent to the jobs channel.
	go func() {
		defer func() {
			level.Info(p.logger).Log("msg", "compaction provider stopping")
			close(jobs)
		}()

		level.Info(p.logger).Log("msg", "compaction provider started")

		level.Info(p.logger).Log("msg", "compaction provider waiting for poll notification")
		<-p.store.PollNotification(ctx)

		var (
			job *work.Job
			ok  bool
		)

		for {
			select {
			case <-ctx.Done():
				return
			case job, ok = <-instanceJobs:
				if !ok {
					return
				}
			}

			if job == nil {
				metricTenantEmptyJob.Inc()
				continue
			}

			select {
			case <-ctx.Done():
				level.Info(p.logger).Log("msg", "compaction provider stopping")
				return
			case jobs <- job:
				level.Info(p.logger).Log("msg", "compaction job created", "job_id", job.ID, "tenant_id", job.Tenant())
			}
		}
	}()

	// Manage the instances which operate on a tenant.  Limits the number of
	// concurrent tenants, while ensuring that each tenant only has one instance.
	go func() {
		var (
			wg       = boundedwaitgroup.New(uint(p.cfg.MaxConcurrentTenants))
			tenant   *tenantselector.Item
			ok       bool
			tenantID string
		)

		for {
			select {
			case <-ctx.Done():
				level.Info(p.logger).Log("msg", "compaction provider instance manager stopping")
				wg.Wait()
				return
			case tenant, ok = <-tenantCh:
				if !ok {
					wg.Wait()
					return
				}

				if tenant == nil {
					continue
				}

				tenantID = tenant.Value()

				p.activeTenantsMtx.Lock()
				if _, ok := p.activeTenants[tenantID]; ok {
					// Tenant is already being processed, skip it.
					p.activeTenantsMtx.Unlock()
					continue
				}

				p.activeTenants[tenantID] = struct{}{}
				p.activeTenantsMtx.Unlock()

				// Merge instance jobs into instanceJobs channel.
				wg.Add(1)
				go func(tenantID string) {
					defer func() {
						p.activeTenantsMtx.Lock()
						level.Info(p.logger).Log("msg", "compaction instance for tenant stopped", "tenant_id", tenantID)
						delete(p.activeTenants, tenantID)
						p.activeTenantsMtx.Unlock()

						wg.Done()
					}()

					level.Info(p.logger).Log("msg", "starting compaction instance for tenant", "tenant_id", tenantID)

					var (
						selector, _ = p.newBlockSelector(tenantID)
						i           = newCompactionInstance(tenantID, selector, p.cfg, p.logger, p)
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
								level.Info(p.logger).Log("msg", "compaction provider instance job sent to main job channel", "job_id", job.ID, "tenant_id", tenantID)
								metricJobsCreated.WithLabelValues(tenantID).Inc()
							}
						}
					}
				}(tenantID)
			}
		}
	}()

	// Tenant channel writer.  Prioritizes tenants and sends them to the instance manager.
	go func() {
		defer close(tenantCh)

		var tenant *tenantselector.Item

		level.Info(p.logger).Log("msg", "tenant priority loop started")

		for {
			select {
			case <-ctx.Done():
				return
			case <-p.store.PollNotification(ctx):
				p.prioritizeTenants(ctx)

			default:
				tenant = p.getNextTenant(ctx)
				if tenant == nil {
					continue
				}

				select {
				case <-ctx.Done():
					return
				case tenantCh <- tenant:
					metricTenantReset.WithLabelValues(tenant.Value()).Inc()
					level.Info(p.logger).Log("msg", "sent tenant to instance manager")
				}
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

// getNextTenant returns the next tenant to process from the priority queue. If
// the priority queue is empty, it will prioritize tenants and wait for a poll
// notification if no tenants have work available.
func (p *CompactionProvider) getNextTenant(ctx context.Context) *tenantselector.Item {
	_, span := tracer.Start(ctx, "getNextTenant")
	defer span.End()

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			// If we have not prioritized any tenants yet, do so now.
			if p.priority.Len() == 0 {
				level.Info(p.logger).Log("msg", "priority queue empty, prioritizing tenants")
				p.prioritizeTenants(ctx)

				if p.priority.Len() == 0 {
					level.Info(p.logger).Log("msg", "no tenants with work available, waiting for poll notification")
					<-p.store.PollNotification(ctx)

					continue
				}
			}

			tenant := heap.Pop(p.priority).(*tenantselector.Item)
			if tenant != nil {
				return tenant
			}

			// If we have an empty tenant, then we have read all tenants from the
			// current priority queue.  However, its possible that we got here
			// because we did not fully drain all tenants due to per-tenant job
			// limits.  Repopulate the priority queue and try again.
			p.prioritizeTenants(ctx)
			tenant = heap.Pop(p.priority).(*tenantselector.Item)
			if tenant != nil {
				return tenant
			}

		}
	}
}

// prioritizeTenants prioritizes tenants based on the number of outstanding blocks.
func (p *CompactionProvider) prioritizeTenants(ctx context.Context) {
	if ctx.Err() != nil {
		return
	}

	p.priorityMtx.Lock()
	defer func() {
		p.lastPrioritizeTime = time.Now()
		p.priorityMtx.Unlock()
	}()

	level.Info(p.logger).Log("msg", "prioritizing tenants for compaction")

	// If we have been called too recently, wait until the min cycle interval has passed.
	if elapsed := time.Since(p.lastPrioritizeTime); elapsed < p.cfg.MinCycleInterval {
		level.Debug(p.logger).Log("msg", "rate limiting tenant prioritization; waiting")
		time.Sleep(p.cfg.MinCycleInterval - elapsed)

		// Continue
	}

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
func (p *CompactionProvider) addToRecentJobs(ctx context.Context, job *work.Job) bool {
	_, span := tracer.Start(ctx, "addToRecentJobs")
	defer span.End()

	if job.Type != tempopb.JobType_JOB_TYPE_COMPACTION || job.JobDetail.Compaction == nil {
		return false
	}

	// Don't cache jobs with empty input
	if len(job.JobDetail.Compaction.Input) == 0 {
		return false
	}

	// Copy the input block IDs
	blockIDs := make([]backend.UUID, len(job.JobDetail.Compaction.Input))
	for i, blockID := range job.JobDetail.Compaction.Input {
		bid, err := backend.ParseUUID(blockID)
		if err != nil {
			level.Error(p.logger).Log("msg", "failed to parse block ID", "block_id", blockID, "err", err)
			return false
		}
		blockIDs[i] = bid
	}

	p.outstandingJobsMtx.Lock()
	p.outstandingJobs[job.ID] = blockIDs
	p.outstandingJobsMtx.Unlock()
	return true
}
