package provider

import (
	"container/heap"
	"context"
	"flag"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/google/uuid"
	"github.com/grafana/tempo/modules/backendscheduler/work"
	"github.com/grafana/tempo/modules/backendscheduler/work/tenantselector"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/modules/storage"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/tempodb"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/blockselector"
)

type CompactionConfig struct {
	PollInterval     time.Duration           `yaml:"poll_interval"`
	BufferSize       int                     `yaml:"buffer_size"`
	MeasureInterval  time.Duration           `yaml:"measure_interval"`
	Compactor        tempodb.CompactorConfig `yaml:"compaction"`
	MaxJobsPerTenant int                     `yaml:"max_jobs_per_tenant"`
}

func (cfg *CompactionConfig) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	f.DurationVar(&cfg.MeasureInterval, prefix+"backend-scheduler.compaction-provider.measure-interval", time.Minute, "Interval at which to metric tenant blocklist")
	f.IntVar(&cfg.MaxJobsPerTenant, prefix+"backend-scheduler.max-jobs-per-tenant", 1000, "Maximum number of jobs to run per tenant before moving on to the next tenant")
	f.DurationVar(&cfg.PollInterval, prefix+"backend-scheduler.compaction-provider.poll-interval", 100*time.Millisecond, "Interval at which to poll for compaction jobs")
	f.IntVar(&cfg.BufferSize, prefix+"backend-scheduler.compaction-provider.buffer-size", 10, "Buffer size for compaction jobs")

	cfg.Compactor = tempodb.CompactorConfig{}
	cfg.Compactor.RegisterFlagsAndApplyDefaults(util.PrefixConfig(prefix, "compaction"), f)
}

type CompactionProvider struct {
	cfg    CompactionConfig
	logger log.Logger

	// Dependencies needed for compaction job selection
	store     storage.Store
	overrides overrides.Interface

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
) *CompactionProvider {
	return &CompactionProvider{
		cfg:         cfg,
		logger:      logger,
		store:       store,
		overrides:   overrides,
		curPriority: tenantselector.NewPriorityQueue(),
	}
}

func (p *CompactionProvider) Start(ctx context.Context) <-chan *work.Job {
	jobs := make(chan *work.Job, p.cfg.BufferSize)

	go func() {
		defer close(jobs)

		pollTicker := time.NewTicker(p.cfg.PollInterval)
		defer pollTicker.Stop()

		measureTicker := time.NewTicker(p.cfg.MeasureInterval)
		defer measureTicker.Stop()

		level.Info(p.logger).Log("msg", "compaction provider started")

		for {
			select {
			case <-ctx.Done():
				level.Info(p.logger).Log("msg", "compaction provider stopping")
				return
			case <-measureTicker.C:
				// Measure the tenants to get their current compaction status
				p.measureTenants()
			case <-pollTicker.C:
				if job := p.nextCompactionJob(ctx); job != nil {
					level.Debug(p.logger).Log(
						"msg", "scheduling compaction job",
						"job_id", job.ID,
						"tenant", job.JobDetail.Tenant,
					)

					select {
					case jobs <- job:
					default:
						// Channel full, try again next tick
					}
				}
			}
		}
	}()

	// Measure the tenants in a separate goroutine
	go func() {
		measureTicker := time.NewTicker(p.cfg.MeasureInterval)
		defer measureTicker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-measureTicker.C:
				p.measureTenants()
			}
		}
	}()

	return jobs
}

// returns the next compaction job for the tenant
func (p *CompactionProvider) nextCompactionJob(_ context.Context) *work.Job {
	var prioritized bool

	reset := func() {
		p.curSelector = nil
		p.curTenant = nil
		p.curTenantJobCount = 0
	}

	for {
		// do we have an current tenant?
		if p.curSelector != nil {

			if p.curTenantJobCount >= p.cfg.MaxJobsPerTenant {
				reset()
				continue
			}

			toBeCompacted, _ := p.curSelector.BlocksToCompact()
			if len(toBeCompacted) == 0 {
				// we have drained this selector to the next tenant!
				reset()
				continue
			}

			input := make([]string, 0, len(toBeCompacted))
			for _, b := range toBeCompacted {
				input = append(input, b.BlockID.String())
			}

			compaction := &tempopb.CompactionDetail{
				Input: input,
			}

			p.curTenantJobCount++

			return &work.Job{
				ID:   uuid.New().String(),
				Type: tempopb.JobType_JOB_TYPE_COMPACTION,
				JobDetail: tempopb.JobDetail{
					Tenant:     p.curTenant.Value(),
					Compaction: compaction,
				},
			}
		}

		// we don't have a current tenant, get the next one

		if p.curPriority.Len() == 0 {
			if prioritized {
				// no compactions are needed and we've already prioritized once
				return nil
			}
			p.prioritizeTenants()
			prioritized = true
		}

		if p.curPriority.Len() == 0 {
			// no tenants = no jobs
			return nil
		}

		p.curTenant = heap.Pop(p.curPriority).(*tenantselector.Item)
		if p.curTenant == nil {
			return nil
		}

		// set the block selector and go to the top to find a job for this tenant
		p.curSelector = blockselector.NewTimeWindowBlockSelector(
			p.store.BlockMetas(p.curTenant.Value()),
			p.cfg.Compactor.MaxCompactionRange,
			p.cfg.Compactor.MaxCompactionObjects,
			p.cfg.Compactor.MaxBlockBytes,
			blockselector.DefaultMinInputBlocks,
			blockselector.DefaultMaxInputBlocks,
		)
	}
}

// prioritizeTenants prioritizes tenants based on the number of outstanding blocks.
func (p *CompactionProvider) prioritizeTenants() {
	tenants := []tenantselector.Tenant{}

	p.curPriority = tenantselector.NewPriorityQueue() // wipe and restart

	for _, tenantID := range p.store.Tenants() {
		if p.overrides.CompactionDisabled(tenantID) {
			continue
		}

		var (
			blocklist         = p.store.BlockMetas(tenantID)
			window            = p.overrides.MaxCompactionRange(tenantID)
			outstandingBlocks = 0
			toBeCompacted     []*backend.BlockMeta
		)

		if window == 0 {
			window = p.cfg.Compactor.MaxCompactionRange
		}

		// TODO: consider using a different blockselector for this
		blockSelector := blockselector.NewTimeWindowBlockSelector(blocklist,
			window,
			p.cfg.Compactor.MaxCompactionObjects,
			p.cfg.Compactor.MaxBlockBytes,
			blockselector.DefaultMinInputBlocks,
			blockselector.DefaultMaxInputBlocks,
		)

		// Measure the outstanding blocks
		for {
			toBeCompacted, _ = blockSelector.BlocksToCompact()
			if len(toBeCompacted) == 0 {
				break
			}

			outstandingBlocks += len(toBeCompacted)
		}

		tenants = append(tenants, tenantselector.Tenant{
			ID:                         tenantID,
			BlocklistLength:            len(blocklist),
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
		window := p.overrides.MaxCompactionRange(tenant)

		if window == 0 {
			window = p.cfg.Compactor.MaxCompactionRange
		}

		blockSelector := blockselector.NewTimeWindowBlockSelector(
			p.store.BlockMetas(tenant),
			window,
			p.cfg.Compactor.MaxCompactionObjects,
			p.cfg.Compactor.MaxBlockBytes,
			blockselector.DefaultMinInputBlocks,
			blockselector.DefaultMaxInputBlocks,
		)

		yes := func(_ string) bool {
			return true
		}

		tempodb.MeasureOutstandingBlocks(tenant, blockSelector, yes)
	}
}
