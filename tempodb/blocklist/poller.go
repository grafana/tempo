package blocklist

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/google/uuid"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/atomic"

	"github.com/grafana/tempo/pkg/boundedwaitgroup"
	"github.com/grafana/tempo/tempodb/backend"
)

const (
	blockStatusLiveLabel      = "live"
	blockStatusCompactedLabel = "compacted"
)

var (
	metricBackendObjects = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "tempodb",
		Name:      "backend_objects_total",
		Help:      "Total number of objects (traces) in the backend",
	}, []string{"tenant", "status"})
	metricBackendBytes = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "tempodb",
		Name:      "backend_bytes_total",
		Help:      "Total number of bytes in the backend.",
	}, []string{"tenant", "status"})
	metricBlocklistErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempodb",
		Name:      "blocklist_poll_errors_total",
		Help:      "Total number of times an error occurred while polling the blocklist.",
	}, []string{"tenant"})
	metricBlocklistPollDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace: "tempodb",
		Name:      "blocklist_poll_duration_seconds",
		Help:      "Records the amount of time to poll and update the blocklist.",
		Buckets:   prometheus.LinearBuckets(0, 60, 10),
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
	metricTenantIndexBuilder = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "tempodb",
		Name:      "blocklist_tenant_index_builder",
		Help:      "A value of 1 indicates this instance of tempodb is building the tenant index.",
	}, []string{"tenant"})
	metricTenantIndexAgeSeconds = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "tempodb",
		Name:      "blocklist_tenant_index_age_seconds",
		Help:      "Age in seconds of the last pulled tenant index.",
	}, []string{"tenant"})
)

// Config is used to configure the poller
type PollerConfig struct {
	PollConcurrency           uint
	PollTenantConcurrency     uint
	PollFallback              bool
	TenantIndexBuilders       int
	StaleTenantIndex          time.Duration
	PollJitterMs              int
	TolerateConsecutiveErrors int
}

// JobSharder is used to determine if a particular job is owned by this process
type JobSharder interface {
	// Owns is used to ask if a job, identified by a string, is owned by this process
	Owns(string) bool
}

// OwnsNothingSharder owns nothing. You do not want this developer on your team.
var OwnsNothingSharder = ownsNothingSharder{}

type ownsNothingSharder struct{}

func (ownsNothingSharder) Owns(_ string) bool {
	return false
}

// OwnsEverythingSharder owns everything. You do not want this developer on your team.
var OwnsEverythingSharder = ownsEverythingSharder{}

type ownsEverythingSharder struct{}

func (ownsEverythingSharder) Owns(_ string) bool {
	return true
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
func (p *Poller) Do(previous *List) (PerTenant, PerTenantCompacted, error) {
	start := time.Now()
	defer func() {
		diff := time.Since(start).Seconds()
		metricBlocklistPollDuration.Observe(diff)
		level.Info(p.logger).Log("msg", "blocklist poll complete", "seconds", diff)
	}()

	ctx := context.Background()
	tenants, err := p.reader.Tenants(ctx)
	if err != nil {
		metricBlocklistErrors.WithLabelValues("").Inc()
		return nil, nil, err
	}

	m := &sync.Mutex{}
	blocklist := PerTenant{}
	compactedBlocklist := PerTenantCompacted{}
	errs := []error{}

	bg := boundedwaitgroup.New(p.cfg.PollTenantConcurrency)
	for _, tenantID := range tenants {
		m.Lock()
		if len(errs) > p.cfg.TolerateConsecutiveErrors {
			m.Unlock()
			break
		}
		m.Unlock()

		bg.Add(1)
		go func(tenantID string) {
			defer bg.Done()

			level.Info(p.logger).Log("msg", "polling tenant", "tenant", tenantID)

			newBlockList, newCompactedBlockList, err := p.pollTenantAndCreateIndex(ctx, tenantID, previous)
			if err != nil {
				level.Error(p.logger).Log("msg", "failed to poll tenant and create index", "err", err)
				m.Lock()
				errs = append(errs, err)
				m.Unlock()
			}

			m.Lock()
			if len(errs) > p.cfg.TolerateConsecutiveErrors {
				level.Error(p.logger).Log("msg", "exiting polling loop early because too many errors", "errCount", len(errs))
				m.Unlock()
				return
			}
			m.Unlock()

			if len(newBlockList) > 0 || len(newCompactedBlockList) > 0 {
				m.Lock()
				blocklist[tenantID] = newBlockList
				compactedBlocklist[tenantID] = newCompactedBlockList
				m.Unlock()

				metricBlocklistLength.WithLabelValues(tenantID).Set(float64(len(newBlockList)))

				backendMetaMetrics := sumTotalBackendMetaMetrics(newBlockList, newCompactedBlockList)
				metricBackendObjects.WithLabelValues(tenantID, blockStatusLiveLabel).Set(float64(backendMetaMetrics.blockMetaTotalObjects))
				metricBackendObjects.WithLabelValues(tenantID, blockStatusCompactedLabel).Set(float64(backendMetaMetrics.compactedBlockMetaTotalObjects))
				metricBackendBytes.WithLabelValues(tenantID, blockStatusLiveLabel).Set(float64(backendMetaMetrics.blockMetaTotalBytes))
				metricBackendBytes.WithLabelValues(tenantID, blockStatusCompactedLabel).Set(float64(backendMetaMetrics.compactedBlockMetaTotalBytes))
				return
			}
			metricBlocklistLength.DeleteLabelValues(tenantID)
			metricBackendObjects.DeleteLabelValues(tenantID)
			metricBackendObjects.DeleteLabelValues(tenantID)
			metricBackendBytes.DeleteLabelValues(tenantID)
		}(tenantID)
	}

	bg.Wait()

	if len(errs) > p.cfg.TolerateConsecutiveErrors {
		return nil, nil, errors.Join(errs...)
	}

	return blocklist, compactedBlocklist, nil
}

func (p *Poller) pollTenantAndCreateIndex(
	ctx context.Context,
	tenantID string,
	previous *List,
) ([]*backend.BlockMeta, []*backend.CompactedBlockMeta, error) {
	span, derivedCtx := opentracing.StartSpanFromContext(ctx, "poll tenant index")
	defer span.Finish()

	// are we a tenant index builder?
	if !p.tenantIndexBuilder(tenantID) {
		metricTenantIndexBuilder.WithLabelValues(tenantID).Set(0)

		i, err := p.reader.TenantIndex(derivedCtx, tenantID)
		err = p.tenantIndexPollError(i, err)
		if err == nil {
			// success! return the retrieved index
			metricTenantIndexAgeSeconds.WithLabelValues(tenantID).Set(float64(time.Since(i.CreatedAt) / time.Second))
			level.Info(p.logger).Log("msg", "successfully pulled tenant index", "tenant", tenantID, "createdAt", i.CreatedAt, "metas", len(i.Meta), "compactedMetas", len(i.CompactedMeta))
			return i.Meta, i.CompactedMeta, nil
		}

		metricTenantIndexErrors.WithLabelValues(tenantID).Inc()

		// there was an error, return the error if we're not supposed to fallback to polling
		if !p.cfg.PollFallback {
			return nil, nil, fmt.Errorf("failed to pull tenant index and no fallback configured: %w", err)
		}

		// polling fallback is true, log the error and continue in this method to completely poll the backend
		level.Error(p.logger).Log("msg", "failed to pull bucket index for tenant. falling back to polling", "tenant", tenantID, "err", err)
	}

	// if we're here then we have been configured to be a tenant index builder OR
	// there was a failure to pull the tenant index and we are configured to fall
	// back to polling.  If quick polling fails, fall back to long poll.
	metricTenantIndexBuilder.WithLabelValues(tenantID).Set(1)
	blocklist, compactedBlocklist, err := p.pollTenantBlocks(derivedCtx, tenantID, previous)
	if err != nil {
		return nil, nil, err
	}

	// everything is happy, write this tenant index
	level.Info(p.logger).Log("msg", "writing tenant index", "tenant", tenantID, "metas", len(blocklist), "compactedMetas", len(compactedBlocklist))
	err = p.writer.WriteTenantIndex(derivedCtx, tenantID, blocklist, compactedBlocklist)
	if err != nil {
		metricTenantIndexErrors.WithLabelValues(tenantID).Inc()
		level.Error(p.logger).Log("msg", "failed to write tenant index", "tenant", tenantID, "err", err)
	}
	metricTenantIndexAgeSeconds.WithLabelValues(tenantID).Set(0)

	return blocklist, compactedBlocklist, nil
}

func (p *Poller) pollTenantBlocks(
	ctx context.Context,
	tenantID string,
	previous *List,
) ([]*backend.BlockMeta, []*backend.CompactedBlockMeta, error) {
	currentBlockIDs, currentCompactedBlockIDs, err := p.reader.Blocks(ctx, tenantID)
	if err != nil {
		return nil, nil, err
	}

	metas := previous.Metas(tenantID)
	compactedMetas := previous.CompactedMetas(tenantID)

	// mm, cm := metaMap(previous.Metas(tenantID), previous.CompactedMetas(tenantID))

	mm := make(map[uuid.UUID]*backend.BlockMeta, len(metas))
	cm := make(map[uuid.UUID]*backend.CompactedBlockMeta, len(compactedMetas))

	for _, i := range metas {
		mm[i.BlockID] = i
	}

	for _, i := range compactedMetas {
		cm[i.BlockID] = i
	}

	chMeta := make(chan *backend.BlockMeta, len(currentBlockIDs))
	chCompactedMeta := make(chan *backend.CompactedBlockMeta, len(currentCompactedBlockIDs))
	anyError := atomic.Error{}

	newBlockIDs := []uuid.UUID{}
	for _, blockID := range currentBlockIDs {
		// if we already have this block id in our previous list, use the existing data.
		if v, ok := mm[blockID]; ok {
			chMeta <- v
			continue
		}
		newBlockIDs = append(newBlockIDs, blockID)
	}

	for _, blockID := range currentCompactedBlockIDs {
		// if we already have this block id in our previous list, use the existing data.
		if v, ok := cm[blockID]; ok {
			chCompactedMeta <- v
			continue
		}

		// TODO: Review this hack to avoid polling for compacted blocks that we
		// know about.  But we need to poll this to ensure that we have the correct
		// time. Probably.
		// if v, ok := mm[blockID]; ok {
		// 	chCompactedMeta <- &backend.CompactedBlockMeta{
		// 		BlockMeta: *v,
		// 		// CompactedTime: time.Now(),
		// 	}
		// 	continue
		// }
		newBlockIDs = append(newBlockIDs, blockID)
	}

	bg := boundedwaitgroup.New(p.cfg.PollConcurrency)
	for _, blockID := range newBlockIDs {
		bg.Add(1)
		go func(id uuid.UUID) {
			defer bg.Done()

			if p.cfg.PollJitterMs > 0 {
				time.Sleep(time.Duration(rand.Intn(p.cfg.PollJitterMs)) * time.Millisecond)
			}

			m, cm, pollBlockErr := p.pollBlock(ctx, tenantID, id)
			if m != nil {
				chMeta <- m
			} else if cm != nil {
				chCompactedMeta <- cm
			} else if pollBlockErr != nil {
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

	return p.flushMetaChannels(chMeta, chCompactedMeta)
}

func (p *Poller) flushMetaChannels(
	chMeta chan *backend.BlockMeta,
	chCompactedMeta chan *backend.CompactedBlockMeta,
) ([]*backend.BlockMeta, []*backend.CompactedBlockMeta, error) {
	newBlockList := make([]*backend.BlockMeta, 0, len(chMeta))
	for m := range chMeta {
		newBlockList = append(newBlockList, m)
	}
	sort.Slice(newBlockList, func(i, j int) bool {
		return newBlockList[i].StartTime.Before(newBlockList[j].StartTime)
	})

	newCompactedBlocklist := make([]*backend.CompactedBlockMeta, 0, len(chCompactedMeta))
	for cm := range chCompactedMeta {
		newCompactedBlocklist = append(newCompactedBlocklist, cm)
	}
	sort.Slice(newCompactedBlocklist, func(i, j int) bool {
		return newCompactedBlocklist[i].StartTime.Before(newCompactedBlocklist[j].StartTime)
	})

	return newBlockList, newCompactedBlocklist, nil
}

func (p *Poller) pollBlock(
	ctx context.Context,
	tenantID string,
	blockID uuid.UUID,
) (*backend.BlockMeta, *backend.CompactedBlockMeta, error) {
	var compactedBlockMeta *backend.CompactedBlockMeta
	blockMeta, err := p.reader.BlockMeta(ctx, blockID, tenantID)
	// if the normal meta doesn't exist maybe it's compacted.
	if errors.Is(err, backend.ErrDoesNotExist) {
		blockMeta = nil
		compactedBlockMeta, err = p.compactor.CompactedBlockMeta(blockID, tenantID)
	}

	// blocks in intermediate states may not have a compacted or normal block meta.
	//   this is not necessarily an error, just bail out
	if errors.Is(err, backend.ErrDoesNotExist) {
		return nil, nil, nil
	}

	if err != nil {
		return nil, nil, err
	}

	return blockMeta, compactedBlockMeta, nil
}

// tenantIndexBuilder returns true if this poller owns this tenant
func (p *Poller) tenantIndexBuilder(tenant string) bool {
	for i := 0; i < p.cfg.TenantIndexBuilders; i++ {
		job := jobPrefix + strconv.Itoa(i) + "-" + tenant
		if p.sharder.Owns(job) {
			return true
		}
	}

	return false
}

func (p *Poller) tenantIndexPollError(idx *backend.TenantIndex, err error) error {
	if err != nil {
		return err
	}

	if p.cfg.StaleTenantIndex != 0 && time.Since(idx.CreatedAt) > p.cfg.StaleTenantIndex {
		return fmt.Errorf("tenant index created at %s is stale", idx.CreatedAt)
	}

	return nil
}

type backendMetaMetrics struct {
	blockMetaTotalObjects          int
	compactedBlockMetaTotalObjects int
	blockMetaTotalBytes            uint64
	compactedBlockMetaTotalBytes   uint64
}

func sumTotalBackendMetaMetrics(
	blockMeta []*backend.BlockMeta,
	compactedBlockMeta []*backend.CompactedBlockMeta,
) backendMetaMetrics {
	var sumTotalObjectsBM int
	var sumTotalObjectsCBM int
	var sumTotalBytesBM uint64
	var sumTotalBytesCBM uint64

	for _, bm := range blockMeta {
		sumTotalObjectsBM += bm.TotalObjects
		sumTotalBytesBM += bm.Size
	}

	for _, cbm := range compactedBlockMeta {
		sumTotalObjectsCBM += cbm.TotalObjects
		sumTotalBytesCBM += cbm.Size
	}

	return backendMetaMetrics{
		blockMetaTotalObjects:          sumTotalObjectsBM,
		compactedBlockMetaTotalObjects: sumTotalObjectsCBM,
		blockMetaTotalBytes:            sumTotalBytesBM,
		compactedBlockMetaTotalBytes:   sumTotalBytesCBM,
	}
}
