package blocklist

import (
	"context"
	"sort"
	"strconv"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/google/uuid"
	"github.com/grafana/tempo/pkg/boundedwaitgroup"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/atomic"
)

var (
	metricBlocklistErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempodb",
		Name:      "blocklist_poll_errors_total",
		Help:      "Total number of times an error occurred while polling the blocklist.",
	}, []string{"tenant"})
	metricBlocklistPollDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace: "tempodb",
		Name:      "blocklist_poll_duration_seconds",
		Help:      "Records the amount of time to poll and update the blocklist.",
		Buckets:   prometheus.LinearBuckets(0, 60, 5),
	})
	metricBlocklistLength = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "tempodb",
		Name:      "blocklist_length",
		Help:      "Total number of blocks per tenant.",
	}, []string{"tenant"})
	metricTenantIndexErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempodb",
		Name:      "blocklist_tenant_index_errors_total",
		Help:      "Total number of times an error occurred while retrieving or building the tenant index.",
	}, []string{"tenant"})
	metricTenantIndexBuilder = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "tempodb",
		Name:      "blocklist_tenant_index_builder",
		Help:      "A value of 1 indicates this instance of tempodb is building the tenant index.",
	})
	metricTenantIndexAgeSeconds = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "tempodb",
		Name:      "blocklist_tenant_index_age_seconds",
		Help:      "Age in seconds of the last pulled tenant index.",
	}, []string{"tenant"})
)

// Config is used to configure the poller
type PollerConfig struct {
	PollConcurrency     uint
	PollFallback        bool
	TenantIndexBuilders int
}

// JobSharder is used to determine if a particular job is owned by this process
type JobSharder interface {
	// Owns is used to ask if a job, identified by a string, is owned by this process
	Owns(string) bool
}

const jobPrefix = "build-tenant-index-"

// Poller retrieves the blocklist
type Poller struct {
	reader    backend.Reader
	writer    backend.Writer
	compactor backend.Compactor

	cfg *PollerConfig

	sharder JobSharder
	logger  log.Logger
}

// NewPoller creates the Poller
func NewPoller(cfg *PollerConfig, sharder JobSharder, reader backend.Reader, compactor backend.Compactor, writer backend.Writer, logger log.Logger) *Poller {
	return &Poller{
		reader:    reader,
		compactor: compactor,
		writer:    writer,

		cfg:     cfg,
		sharder: sharder,
		logger:  logger,
	}
}

// Do does the doing of getting a blocklist
func (p *Poller) Do() (PerTenant, PerTenantCompacted, error) {
	start := time.Now()
	defer func() { metricBlocklistPollDuration.Observe(time.Since(start).Seconds()) }()

	ctx := context.Background()
	tenants, err := p.reader.Tenants(ctx)
	if err != nil {
		metricBlocklistErrors.WithLabelValues("").Inc()
		return nil, nil, err
	}

	blocklist := PerTenant{}
	compactedBlocklist := PerTenantCompacted{}

	for _, tenantID := range tenants {
		newBlockList, newCompactedBlockList, err := p.pollTenantAndCreateIndex(ctx, tenantID)
		if err != nil {
			return nil, nil, err
		}

		metricBlocklistLength.WithLabelValues(tenantID).Set(float64(len(newBlockList)))

		blocklist[tenantID] = newBlockList
		compactedBlocklist[tenantID] = newCompactedBlockList
	}

	return blocklist, compactedBlocklist, nil
}

func (p *Poller) pollTenantAndCreateIndex(ctx context.Context, tenantID string) ([]*backend.BlockMeta, []*backend.CompactedBlockMeta, error) {
	// are we a tenant index builder?
	if !p.buildTenantIndex() {
		metricTenantIndexBuilder.Set(0)

		i, err := p.reader.TenantIndex(ctx, tenantID)
		if err == nil {
			// success! return the retrieved index
			metricTenantIndexAgeSeconds.WithLabelValues(tenantID).Set(float64(time.Since(i.CreatedAt) / time.Second))
			level.Info(p.logger).Log("msg", "successfully pulled tenant index", "tenant", tenantID, "createdAt", i.CreatedAt, "metas", len(i.Meta), "compactedMetas", len(i.CompactedMeta))
			return i.Meta, i.CompactedMeta, nil
		}

		metricTenantIndexErrors.WithLabelValues(tenantID).Inc()

		// there was an error, return the error if we're not supposed to fallback to polling
		if !p.cfg.PollFallback {
			return nil, nil, err
		}

		// polling fallback is true, log the error and continue in this method to completely poll the backend
		level.Error(p.logger).Log("msg", "failed to pull bucket index for tenant. falling back to polling", "tenant", tenantID, "err", err)
	}

	// if we're here then we have been configured to be a tenant index builder OR there was a failure to pull
	// the tenant index and we are configured to fall back to polling
	metricTenantIndexBuilder.Set(1)
	blocklist, compactedBlocklist, err := p.pollTenantBlocks(ctx, tenantID)
	if err != nil {
		return nil, nil, err
	}

	// everything is happy, write this tenant index
	level.Info(p.logger).Log("msg", "writing tenant index", "tenant", tenantID, "metas", len(blocklist), "compactedMetas", len(compactedBlocklist))
	err = p.writer.WriteTenantIndex(ctx, tenantID, blocklist, compactedBlocklist)
	if err != nil {
		metricTenantIndexErrors.WithLabelValues(tenantID).Inc()
		level.Error(p.logger).Log("msg", "failed to write tenant index", "tenant", tenantID, "err", err)
	}

	return blocklist, compactedBlocklist, nil
}

func (p *Poller) pollTenantBlocks(ctx context.Context, tenantID string) ([]*backend.BlockMeta, []*backend.CompactedBlockMeta, error) {
	blockIDs, err := p.reader.Blocks(ctx, tenantID)
	if err != nil {
		metricBlocklistErrors.WithLabelValues(tenantID).Inc()
		return []*backend.BlockMeta{}, []*backend.CompactedBlockMeta{}, err
	}

	bg := boundedwaitgroup.New(p.cfg.PollConcurrency)
	chMeta := make(chan *backend.BlockMeta, len(blockIDs))
	chCompactedMeta := make(chan *backend.CompactedBlockMeta, len(blockIDs))
	anyError := atomic.Error{}

	for _, blockID := range blockIDs {
		bg.Add(1)
		go func(uuid uuid.UUID) {
			defer bg.Done()
			m, cm, err := p.pollBlock(ctx, tenantID, uuid)
			if m != nil {
				chMeta <- m
			} else if cm != nil {
				chCompactedMeta <- cm
			} else if err != nil {
				anyError.Store(err)
			}
		}(blockID)
	}

	bg.Wait()
	close(chMeta)
	close(chCompactedMeta)

	if err = anyError.Load(); err != nil {
		metricTenantIndexErrors.WithLabelValues(tenantID).Inc()
		return nil, nil, err
	}

	newBlockList := make([]*backend.BlockMeta, 0, len(blockIDs))
	for m := range chMeta {
		newBlockList = append(newBlockList, m)
	}
	sort.Slice(newBlockList, func(i, j int) bool {
		return newBlockList[i].StartTime.Before(newBlockList[j].StartTime)
	})

	newCompactedBlocklist := make([]*backend.CompactedBlockMeta, 0, len(blockIDs))
	for cm := range chCompactedMeta {
		newCompactedBlocklist = append(newCompactedBlocklist, cm)
	}
	sort.Slice(newCompactedBlocklist, func(i, j int) bool {
		return newCompactedBlocklist[i].StartTime.Before(newCompactedBlocklist[j].StartTime)
	})

	return newBlockList, newCompactedBlocklist, nil
}

func (p *Poller) pollBlock(ctx context.Context, tenantID string, blockID uuid.UUID) (*backend.BlockMeta, *backend.CompactedBlockMeta, error) {
	var compactedBlockMeta *backend.CompactedBlockMeta
	blockMeta, err := p.reader.BlockMeta(ctx, blockID, tenantID)
	// if the normal meta doesn't exist maybe it's compacted.
	if err == backend.ErrDoesNotExist {
		blockMeta = nil
		compactedBlockMeta, err = p.compactor.CompactedBlockMeta(blockID, tenantID)
	}

	// blocks in intermediate states may not have a compacted or normal block meta.
	//   this is not necessarily an error, just bail out
	if err == backend.ErrDoesNotExist {
		return nil, nil, nil
	}

	if err != nil {
		return nil, nil, err
	}

	return blockMeta, compactedBlockMeta, nil
}

func (p *Poller) buildTenantIndex() bool {
	for i := 0; i < p.cfg.TenantIndexBuilders; i++ {
		job := jobPrefix + strconv.Itoa(i)
		if p.sharder.Owns(job) {
			return true
		}
	}

	return false
}
