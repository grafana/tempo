package tempodb

import (
	"bytes"
	"context"
	"fmt"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/willf/bloom"
	"go.uber.org/atomic"

	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/cache"
	"github.com/grafana/tempo/tempodb/backend/gcs"
	"github.com/grafana/tempo/tempodb/backend/local"
	"github.com/grafana/tempo/tempodb/pool"
	"github.com/grafana/tempo/tempodb/wal"
)

var (
	metricMaintenanceTotal = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "tempodb",
		Name:      "maintenance_total",
		Help:      "Total number of times the maintenance cycle has occurred.",
	})
	metricBlocklistErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempodb",
		Name:      "blocklist_poll_errors_total",
		Help:      "Total number of times an error occurred while polling the blocklist.",
	}, []string{"tenant"})
	metricBlocklistPollDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace: "tempodb",
		Name:      "blocklist_poll_duration_seconds",
		Help:      "Records the amount of time to poll and update the blocklist.",
		Buckets:   prometheus.ExponentialBuckets(.25, 2, 6),
	})
	metricBlocklistLength = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "tempodb",
		Name:      "blocklist_length",
		Help:      "Total number of blocks per tenant.",
	}, []string{"tenant", "level"})
	metricRetentionDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace: "tempodb",
		Name:      "retention_duration_seconds",
		Help:      "Records the amount of time to perform retention tasks.",
		Buckets:   prometheus.ExponentialBuckets(.25, 2, 6),
	})
	metricRetentionErrors = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "tempodb",
		Name:      "retention_errors_total",
		Help:      "Total number of times an error occurred while performing retention tasks.",
	})
	metricMarkedForDeletion = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "tempodb",
		Name:      "retention_marked_for_deletion_total",
		Help:      "Total number of blocks marked for deletion.",
	})
	metricDeleted = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "tempodb",
		Name:      "retention_deleted_total",
		Help:      "Total number of blocks deleted.",
	})
)

type Writer interface {
	WriteBlock(ctx context.Context, block wal.WriteableBlock) error
	WAL() *wal.WAL
}

type Reader interface {
	Find(tenantID string, id backend.ID) ([]byte, FindMetrics, error)
	Shutdown()
}

type Compactor interface {
	EnableCompaction(cfg *CompactorConfig)
}

type FindMetrics struct {
	BloomFilterReads     *atomic.Int32
	BloomFilterBytesRead *atomic.Int32
	IndexReads           *atomic.Int32
	IndexBytesRead       *atomic.Int32
	BlockReads           *atomic.Int32
	BlockBytesRead       *atomic.Int32
}

type readerWriter struct {
	r backend.Reader
	w backend.Writer
	c backend.Compactor

	wal  *wal.WAL
	pool *pool.Pool

	logger        log.Logger
	cfg           *Config
	compactorCfg  *CompactorConfig
	blockLists    map[string][]*backend.BlockMeta
	blockListsMtx sync.Mutex

	jobStopper          *pool.Stopper
	compactedBlockLists map[string][]*backend.CompactedBlockMeta
}

func New(cfg *Config, logger log.Logger) (Reader, Writer, Compactor, error) {
	var err error
	var r backend.Reader
	var w backend.Writer
	var c backend.Compactor

	switch cfg.Backend {
	case "local":
		r, w, c, err = local.New(cfg.Local)
	case "gcs":
		r, w, c, err = gcs.New(cfg.GCS)
	default:
		err = fmt.Errorf("unknown local %s", cfg.Backend)
	}

	if err != nil {
		return nil, nil, nil, err
	}

	if cfg.Cache != nil {
		r, err = cache.New(r, cfg.Cache, logger)

		if err != nil {
			return nil, nil, nil, err
		}
	}

	rw := &readerWriter{
		c:                   c,
		compactedBlockLists: make(map[string][]*backend.CompactedBlockMeta),
		r:                   r,
		w:                   w,
		cfg:                 cfg,
		logger:              logger,
		pool:                pool.NewPool(cfg.Pool),
		blockLists:          make(map[string][]*backend.BlockMeta),
	}

	rw.wal, err = wal.New(rw.cfg.WAL)
	if err != nil {
		return nil, nil, nil, err
	}

	go rw.maintenanceLoop()

	return rw, rw, rw, nil
}

func (rw *readerWriter) WriteBlock(ctx context.Context, c wal.WriteableBlock) error {
	records := c.Records()
	indexBytes, err := backend.MarshalRecords(records)
	if err != nil {
		return err
	}

	bloomBuffer := &bytes.Buffer{}
	_, err = c.BloomFilter().WriteTo(bloomBuffer)
	if err != nil {
		return err
	}

	meta := c.BlockMeta()
	err = rw.w.Write(ctx, meta, bloomBuffer.Bytes(), indexBytes, c.ObjectFilePath())
	if err != nil {
		return err
	}

	c.BlockWroteSuccessfully(time.Now())

	return nil
}

func (rw *readerWriter) WriteBlockMeta(ctx context.Context, tracker backend.AppendTracker, c wal.WriteableBlock) error {
	records := c.Records()
	indexBytes, err := backend.MarshalRecords(records)
	if err != nil {
		return err
	}

	bloomBuffer := &bytes.Buffer{}
	_, err = c.BloomFilter().WriteTo(bloomBuffer)
	if err != nil {
		return err
	}

	meta := c.BlockMeta()
	err = rw.w.WriteBlockMeta(ctx, tracker, meta, bloomBuffer.Bytes(), indexBytes)
	if err != nil {
		return err
	}

	c.BlockWroteSuccessfully(time.Now())

	return nil
}

func (rw *readerWriter) WAL() *wal.WAL {
	return rw.wal
}

func (rw *readerWriter) Find(tenantID string, id backend.ID) ([]byte, FindMetrics, error) {
	metrics := FindMetrics{
		BloomFilterReads:     atomic.NewInt32(0),
		BloomFilterBytesRead: atomic.NewInt32(0),
		IndexReads:           atomic.NewInt32(0),
		IndexBytesRead:       atomic.NewInt32(0),
		BlockReads:           atomic.NewInt32(0),
		BlockBytesRead:       atomic.NewInt32(0),
	}

	rw.blockListsMtx.Lock()
	blocklist, found := rw.blockLists[tenantID]
	copiedBlocklist := make([]interface{}, 0, len(blocklist))
	for _, b := range blocklist {
		// if in range copy
		if bytes.Compare(id, b.MinID) != -1 && bytes.Compare(id, b.MaxID) != 1 {
			copiedBlocklist = append(copiedBlocklist, b)
		}
	}
	rw.blockListsMtx.Unlock()

	if !found {
		return nil, metrics, fmt.Errorf("tenantID %s not found", tenantID)
	}

	foundBytes, err := rw.pool.RunJobs(copiedBlocklist, func(payload interface{}) ([]byte, error) {
		meta := payload.(*backend.BlockMeta)

		bloomBytes, err := rw.r.Bloom(meta.BlockID, tenantID)
		if err != nil {
			return nil, fmt.Errorf("error retrieving bloom %v", err)
		}

		filter := &bloom.BloomFilter{}
		_, err = filter.ReadFrom(bytes.NewReader(bloomBytes))
		if err != nil {
			return nil, fmt.Errorf("error parsing bloom %v", err)
		}

		metrics.BloomFilterReads.Inc()
		metrics.BloomFilterBytesRead.Add(int32(len(bloomBytes)))
		if !filter.Test(id) {
			return nil, nil
		}

		indexBytes, err := rw.r.Index(meta.BlockID, tenantID)
		metrics.IndexReads.Inc()
		metrics.IndexBytesRead.Add(int32(len(indexBytes)))
		if err != nil {
			return nil, fmt.Errorf("error reading index %v", err)
		}

		record, err := backend.FindRecord(id, indexBytes) // todo: replace with backend.Finder
		if err != nil {
			return nil, fmt.Errorf("error finding record %v", err)
		}

		if record == nil {
			return nil, nil
		}

		objectBytes := make([]byte, record.Length)
		err = rw.r.Object(meta.BlockID, tenantID, record.Start, objectBytes)
		metrics.BlockReads.Inc()
		metrics.BlockBytesRead.Add(int32(len(objectBytes)))
		if err != nil {
			return nil, fmt.Errorf("error reading object %v", err)
		}

		iter := backend.NewIterator(bytes.NewReader(objectBytes))
		var foundObject []byte
		for {
			iterID, iterObject, err := iter.Next()
			if iterID == nil {
				break
			}
			if err != nil {
				return nil, err
			}
			if bytes.Equal(iterID, id) {
				foundObject = iterObject
				break
			}
		}
		return foundObject, nil
	})

	return foundBytes, metrics, err
}

func (rw *readerWriter) Shutdown() {
	// todo: stop blocklist poll
	rw.pool.Shutdown()
	rw.r.Shutdown()
}

func (rw *readerWriter) EnableCompaction(cfg *CompactorConfig) {
	if cfg != nil {
		level.Info(rw.logger).Log("msg", "compaction enabled.")
	}
	rw.compactorCfg = cfg
}

func (rw *readerWriter) maintenanceLoop() {
	if rw.cfg.MaintenanceCycle == 0 {
		level.Info(rw.logger).Log("msg", "blocklist Refresh Rate unset.  tempodb querying, compaction and retention effectively disabled.")
		return
	}

	rw.doMaintenance()

	ticker := time.NewTicker(rw.cfg.MaintenanceCycle)
	for range ticker.C {
		rw.doMaintenance()
	}
}

func (rw *readerWriter) doMaintenance() {
	metricMaintenanceTotal.Inc()

	rw.pollBlocklist()

	if rw.compactorCfg != nil {
		rw.doCompaction()
		rw.doRetention()
	}
}

func (rw *readerWriter) pollBlocklist() {
	start := time.Now()
	defer func() { metricBlocklistPollDuration.Observe(time.Since(start).Seconds()) }()

	tenants, err := rw.r.Tenants()
	if err != nil {
		metricBlocklistErrors.WithLabelValues("").Inc()
		level.Error(rw.logger).Log("msg", "error retrieving tenants while polling blocklist", "err", err)
	}

	for _, tenantID := range tenants {
		blockIDs, err := rw.r.Blocks(tenantID)
		if err != nil {
			metricBlocklistErrors.WithLabelValues(tenantID).Inc()
			level.Error(rw.logger).Log("msg", "error polling blocklist", "tenantID", tenantID, "err", err)
		}

		interfaceSlice := make([]interface{}, 0, len(blockIDs))
		for _, id := range blockIDs {
			interfaceSlice = append(interfaceSlice, id)
		}

		listMutex := sync.Mutex{}
		blocklist := make([]*backend.BlockMeta, 0, len(blockIDs))
		compactedBlocklist := make([]*backend.CompactedBlockMeta, 0, len(blockIDs))
		_, err = rw.pool.RunJobs(interfaceSlice, func(payload interface{}) ([]byte, error) {
			blockID := payload.(uuid.UUID)

			var compactedBlockMeta *backend.CompactedBlockMeta
			blockMeta, err := rw.r.BlockMeta(blockID, tenantID)
			// if the normal meta doesn't exist maybe it's compacted.
			if err == backend.ErrMetaDoesNotExist {
				blockMeta = nil
				compactedBlockMeta, err = rw.c.CompactedBlockMeta(blockID, tenantID)
			}

			if err != nil {
				metricBlocklistErrors.WithLabelValues(tenantID).Inc()
				level.Error(rw.logger).Log("msg", "failed to retrieve block meta", "tenantID", tenantID, "blockID", blockID, "err", err)
				return nil, nil
			}

			// todo:  make this not terrible. this mutex is dumb we should be returning results with a channel. shoehorning this into the worker pool is silly.
			//        make the worker pool more generic? and reusable in this case
			listMutex.Lock()
			if blockMeta != nil {
				blocklist = append(blocklist, blockMeta)

			} else if compactedBlockMeta != nil {
				compactedBlocklist = append(compactedBlocklist, compactedBlockMeta)
			}
			listMutex.Unlock()

			return nil, nil
		})

		if err != nil {
			metricBlocklistErrors.WithLabelValues(tenantID).Inc()
			level.Error(rw.logger).Log("msg", "run blocklist jobs", "tenantID", tenantID, "err", err)
			continue
		}

		// Get compacted block metrics from compactedBlocklist (for level>0)
		metricsPerLevel := make([]int, maxNumLevels)
		for _, block := range compactedBlocklist {
			if block.CompactionLevel >= maxNumLevels {
				continue
			}
			metricsPerLevel[block.CompactionLevel]++
		}

		metricBlocklistLength.WithLabelValues(tenantID, "0").Set(float64(len(blocklist)))
		for i := 1; i < maxNumLevels; i++ {
			metricBlocklistLength.WithLabelValues(tenantID, strconv.Itoa(i)).Set(float64(metricsPerLevel[i]))
		}

		sort.Slice(blocklist, func(i, j int) bool {
			return blocklist[i].StartTime.Before(blocklist[j].StartTime)
		})
		sort.Slice(compactedBlocklist, func(i, j int) bool {
			return compactedBlocklist[i].StartTime.Before(compactedBlocklist[j].StartTime)
		})

		rw.blockListsMtx.Lock()
		rw.blockLists[tenantID] = blocklist
		rw.compactedBlockLists[tenantID] = compactedBlocklist
		rw.blockListsMtx.Unlock()
	}
}

func (rw *readerWriter) doRetention() {
	tenants := rw.blocklistTenants()

	// todo: continued abuse of runJobs.  need a runAllJobs() method or something
	_, err := rw.pool.RunJobs(tenants, func(payload interface{}) ([]byte, error) {
		start := time.Now()
		defer func() { metricRetentionDuration.Observe(time.Since(start).Seconds()) }()

		tenantID := payload.(string)

		// iterate through block list.  make compacted anything that is past retention.
		cutoff := time.Now().Add(-rw.compactorCfg.BlockRetention)
		blocklist := rw.blocklist(tenantID)
		for _, b := range blocklist {
			if b.EndTime.Before(cutoff) {
				err := rw.c.MarkBlockCompacted(b.BlockID, tenantID)
				if err != nil {
					level.Error(rw.logger).Log("msg", "failed to mark block compacted during retention", "blockID", b.BlockID, "tenantID", tenantID, "err", err)
					metricRetentionErrors.Inc()
				} else {
					metricMarkedForDeletion.Inc()
				}
			}
		}

		// iterate through compacted list looking for blocks ready to be cleared
		cutoff = time.Now().Add(-rw.compactorCfg.CompactedBlockRetention)
		compactedBlocklist := rw.compactedBlocklist(tenantID)
		for _, b := range compactedBlocklist {
			if b.CompactedTime.Before(cutoff) {
				err := rw.c.ClearBlock(b.BlockID, tenantID)
				if err != nil {
					level.Error(rw.logger).Log("msg", "failed to clear compacted block during retention", "blockID", b.BlockID, "tenantID", tenantID, "err", err)
					metricRetentionErrors.Inc()
				} else {
					metricDeleted.Inc()
				}
			}
		}

		return nil, nil
	})

	if err != nil {
		level.Error(rw.logger).Log("msg", "failure to start retention.  retention disabled until the next maintenance cycle", "err", err)
		metricRetentionErrors.Inc()
	}
}

func (rw *readerWriter) blocklistTenants() []interface{} {
	rw.blockListsMtx.Lock()
	defer rw.blockListsMtx.Unlock()

	tenants := make([]interface{}, 0, len(rw.blockLists))
	for tenant := range rw.blockLists {
		tenants = append(tenants, tenant)
	}

	return tenants
}

func (rw *readerWriter) blocklist(tenantID string) []*backend.BlockMeta {
	rw.blockListsMtx.Lock()
	defer rw.blockListsMtx.Unlock()

	if tenantID == "" {
		return nil
	}

	copiedBlocklist := make([]*backend.BlockMeta, 0, len(rw.blockLists[tenantID]))
	copiedBlocklist = append(copiedBlocklist, rw.blockLists[tenantID]...)
	return copiedBlocklist
}

// todo:  make separate compacted list mutex?
func (rw *readerWriter) compactedBlocklist(tenantID string) []*backend.CompactedBlockMeta {
	rw.blockListsMtx.Lock()
	defer rw.blockListsMtx.Unlock()

	if tenantID == "" {
		return nil
	}

	copiedBlocklist := make([]*backend.CompactedBlockMeta, 0, len(rw.compactedBlockLists[tenantID]))
	copiedBlocklist = append(copiedBlocklist, rw.compactedBlockLists[tenantID]...)

	return copiedBlocklist
}
