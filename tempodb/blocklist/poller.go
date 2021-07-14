package blocklist

import (
	"context"
	"sort"
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
)

// PerTenant is a map of tenant ids to backend.BlockMetas
type PerTenant map[string][]*backend.BlockMeta

// PerTenantCompacted is a map of tenant ids to backend.CompactedBlockMetas
type PerTenantCompacted map[string][]*backend.CompactedBlockMeta

// PollerSharder is used to determine if this node should build the blocklist index or just pull it
//  jpe rename? combine with others?
type PollerSharder interface {
	BuildTenantIndex() bool
}

// Poller retrieves the blocklist
type Poller struct {
	reader    backend.Reader
	writer    backend.Writer
	compactor backend.Compactor

	pollConcurrency uint
	pollFallback    bool // jpe wire this up to a config value and document

	sharder PollerSharder
	logger  log.Logger
}

// NewPoller creates the Poller
func NewPoller(pollConcurrency uint, pollFallback bool, sharder PollerSharder, reader backend.Reader, compactor backend.Compactor, writer backend.Writer, logger log.Logger) *Poller {
	return &Poller{
		reader:    reader,
		compactor: compactor,
		writer:    writer,

		pollConcurrency: pollConcurrency, // jpe remove poll prefix
		pollFallback:    pollFallback,

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
		newBlockList, newCompactedBlockList, err := p.pollTenant(ctx, tenantID)
		if err != nil {
			return nil, nil, err
		}

		metricBlocklistLength.WithLabelValues(tenantID).Set(float64(len(newBlockList)))

		blocklist[tenantID] = newBlockList
		compactedBlocklist[tenantID] = newCompactedBlockList
	}

	return blocklist, compactedBlocklist, nil
}

func (p *Poller) pollTenant(ctx context.Context, tenantID string) ([]*backend.BlockMeta, []*backend.CompactedBlockMeta, error) {
	// pull bucket index? or poll everything?
	if !p.sharder.BuildTenantIndex() {
		// jpe gauge metric: IS POLLING BUCKET INDEX create alarm if gauge == 0

		meta, compactedMeta, err := p.reader.TenantIndex(ctx, tenantID)
		if err == nil {
			return meta, compactedMeta, nil
		}

		// we had an error, do we bail?
		if !p.pollFallback {
			return nil, nil, err
		}

		// log error and keep on keeping on
		metricBlocklistErrors.WithLabelValues(tenantID).Inc()
		level.Error(p.logger).Log("msg", "failed to pull bucket index for tenant. falling back to polling", "tenant", tenantID, "err", err)
	}

	blockIDs, err := p.reader.Blocks(ctx, tenantID)
	if err != nil {
		metricBlocklistErrors.WithLabelValues(tenantID).Inc()
		return []*backend.BlockMeta{}, []*backend.CompactedBlockMeta{}, err
	}

	bg := boundedwaitgroup.New(p.pollConcurrency)
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

	// everything is happy, write this tenant index
	err = p.writer.WriteTenantIndex(ctx, tenantID, newBlockList, newCompactedBlocklist)
	if err != nil {
		// jpe metric, new metric?
		level.Error(p.logger).Log("msg", "failed to write tenant index", "tenant", tenantID, "err", err)
	}

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
		metricBlocklistErrors.WithLabelValues(tenantID).Inc()
		return nil, nil, err
	}

	return blockMeta, compactedBlockMeta, nil
}
