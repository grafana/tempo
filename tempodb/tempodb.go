package tempodb

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	willf_bloom "github.com/willf/bloom"
	"go.uber.org/atomic"

	"github.com/cortexproject/cortex/pkg/util"
	"github.com/opentracing/opentracing-go"
	ot_log "github.com/opentracing/opentracing-go/log"

	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/diskcache"
	"github.com/grafana/tempo/tempodb/backend/gcs"
	"github.com/grafana/tempo/tempodb/backend/local"
	"github.com/grafana/tempo/tempodb/backend/memcached"
	"github.com/grafana/tempo/tempodb/backend/s3"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/encoding/bloom"
	"github.com/grafana/tempo/tempodb/pool"
	"github.com/grafana/tempo/tempodb/wal"
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
		Buckets:   prometheus.ExponentialBuckets(.25, 2, 6),
	})
	metricBlocklistLength = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "tempodb",
		Name:      "blocklist_length",
		Help:      "Total number of blocks per tenant.",
	}, []string{"tenant"})
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
	Find(ctx context.Context, tenantID string, id encoding.ID) ([]byte, FindMetrics, error)
	Shutdown()
}

type Compactor interface {
	EnableCompaction(cfg *CompactorConfig, sharder CompactorSharder)
}

type CompactorSharder interface {
	Combine(objA []byte, objB []byte) []byte
	Owns(hash string) bool
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
	blockLists    map[string][]*encoding.BlockMeta
	blockListsMtx sync.Mutex

	compactorCfg        *CompactorConfig
	compactedBlockLists map[string][]*encoding.CompactedBlockMeta
	compactorSharder    CompactorSharder
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
	case "s3":
		r, w, c, err = s3.New(cfg.S3)
	default:
		err = fmt.Errorf("unknown backend %s", cfg.Backend)
	}

	if err != nil {
		return nil, nil, nil, err
	}

	if cfg.Diskcache != nil {
		r, err = diskcache.New(r, cfg.Diskcache, logger)

		if err != nil {
			return nil, nil, nil, err
		}
	}

	if cfg.Memcached != nil {
		r, w, err = memcached.New(r, w, cfg.Memcached, logger)

		if err != nil {
			return nil, nil, nil, err
		}
	}

	rw := &readerWriter{
		c:                   c,
		compactedBlockLists: make(map[string][]*encoding.CompactedBlockMeta),
		r:                   r,
		w:                   w,
		cfg:                 cfg,
		logger:              logger,
		pool:                pool.NewPool(cfg.Pool),
		blockLists:          make(map[string][]*encoding.BlockMeta),
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
	indexBytes, err := encoding.MarshalRecords(records)
	if err != nil {
		return err
	}

	bloomBuffers, err := c.BloomFilter().WriteTo()
	if err != nil {
		return err
	}

	meta := c.BlockMeta()
	err = rw.w.Write(ctx, meta, bloomBuffers, indexBytes, c.ObjectFilePath())
	if err != nil {
		return err
	}

	err = c.Flushed()
	if err != nil {
		return err
	}

	return nil
}

func (rw *readerWriter) WriteBlockMeta(ctx context.Context, tracker backend.AppendTracker, c wal.WriteableBlock) error {
	records := c.Records()
	indexBytes, err := encoding.MarshalRecords(records)
	if err != nil {
		return err
	}

	bloomBuffers, err := c.BloomFilter().WriteTo()
	if err != nil {
		return err
	}

	meta := c.BlockMeta()
	err = rw.w.WriteBlockMeta(ctx, tracker, meta, bloomBuffers, indexBytes)
	if err != nil {
		return err
	}

	return nil
}

func (rw *readerWriter) WAL() *wal.WAL {
	return rw.wal
}

func (rw *readerWriter) Find(ctx context.Context, tenantID string, id encoding.ID) ([]byte, FindMetrics, error) {
	metrics := FindMetrics{
		BloomFilterReads:     atomic.NewInt32(0),
		BloomFilterBytesRead: atomic.NewInt32(0),
		IndexReads:           atomic.NewInt32(0),
		IndexBytesRead:       atomic.NewInt32(0),
		BlockReads:           atomic.NewInt32(0),
		BlockBytesRead:       atomic.NewInt32(0),
	}

	// tracing instrumentation
	logger := util.WithContext(ctx, util.Logger)
	span, derivedCtx := opentracing.StartSpanFromContext(ctx, "store.Find")
	defer span.Finish()

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
		return nil, metrics, nil
	}

	foundBytes, err := rw.pool.RunJobs(derivedCtx, copiedBlocklist, func(ctx context.Context, payload interface{}) ([]byte, error) {
		meta := payload.(*encoding.BlockMeta)

		shardKey := bloom.ShardKeyForTraceID(id)
		level.Debug(logger).Log("msg", "fetching bloom", "shardKey", shardKey)
		bloomBytes, err := rw.r.Bloom(ctx, meta.BlockID, tenantID, shardKey)
		if err != nil {
			return nil, fmt.Errorf("error retrieving bloom %v", err)
		}

		filter := &willf_bloom.BloomFilter{}
		_, err = filter.ReadFrom(bytes.NewReader(bloomBytes))
		if err != nil {
			return nil, fmt.Errorf("error parsing bloom %v", err)
		}

		metrics.BloomFilterReads.Inc()
		metrics.BloomFilterBytesRead.Add(int32(len(bloomBytes)))
		if !filter.Test(id) {
			return nil, nil
		}

		indexBytes, err := rw.r.Index(ctx, meta.BlockID, tenantID)
		metrics.IndexReads.Inc()
		metrics.IndexBytesRead.Add(int32(len(indexBytes)))
		if err != nil {
			return nil, fmt.Errorf("error reading index %v", err)
		}

		record, err := encoding.FindRecord(id, indexBytes) // todo: replace with backend.Finder
		if err != nil {
			return nil, fmt.Errorf("error finding record %v", err)
		}

		if record == nil {
			return nil, nil
		}

		objectBytes := make([]byte, record.Length)
		err = rw.r.Object(ctx, meta.BlockID, tenantID, record.Start, objectBytes)
		metrics.BlockReads.Inc()
		metrics.BlockBytesRead.Add(int32(len(objectBytes)))
		if err != nil {
			return nil, fmt.Errorf("error reading object %v", err)
		}

		iter := encoding.NewIterator(bytes.NewReader(objectBytes))
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
		level.Info(logger).Log("msg", "searching for trace in block", "traceID", hex.EncodeToString(id), "block", meta.BlockID, "found", foundObject != nil)
		span.LogFields(ot_log.String("msg", "searching for trace in block"), ot_log.String("traceID", hex.EncodeToString(id)), ot_log.String("block", meta.BlockID.String()), ot_log.Bool("found", foundObject != nil))
		if foundObject != nil {
			span.SetTag("object bytes", len(foundObject))
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

func (rw *readerWriter) EnableCompaction(cfg *CompactorConfig, c CompactorSharder) {
	rw.compactorCfg = cfg
	rw.compactorSharder = c

	if rw.cfg.BlocklistPoll == 0 {
		level.Info(rw.logger).Log("msg", "maintenance cycle unset.  compaction and retention disabled.")
		return
	}

	if cfg != nil {
		level.Info(rw.logger).Log("msg", "compaction and retention enabled.")
		go rw.compactionLoop()
		go rw.retentionLoop()
	}
}

func (rw *readerWriter) maintenanceLoop() {
	if rw.cfg.BlocklistPoll == 0 {
		level.Info(rw.logger).Log("msg", "maintenance cycle unset.  blocklist polling disabled.")
		return
	}

	rw.pollBlocklist()

	ticker := time.NewTicker(rw.cfg.BlocklistPoll)
	for range ticker.C {
		rw.pollBlocklist()
	}
}

func (rw *readerWriter) pollBlocklist() {
	start := time.Now()
	defer func() { metricBlocklistPollDuration.Observe(time.Since(start).Seconds()) }()

	ctx := context.Background()
	tenants, err := rw.r.Tenants(ctx)
	if err != nil {
		metricBlocklistErrors.WithLabelValues("").Inc()
		level.Error(rw.logger).Log("msg", "error retrieving tenants while polling blocklist", "err", err)
	}

	for _, tenantID := range tenants {
		blockIDs, err := rw.r.Blocks(ctx, tenantID)
		if err != nil {
			metricBlocklistErrors.WithLabelValues(tenantID).Inc()
			level.Error(rw.logger).Log("msg", "error polling blocklist", "tenantID", tenantID, "err", err)
		}
		if len(blockIDs) == 0 {
			rw.blockListsMtx.Lock()
			delete(rw.blockLists, tenantID)
			delete(rw.compactedBlockLists, tenantID)
			rw.blockListsMtx.Unlock()
			level.Info(rw.logger).Log("msg", "deleted in-memory blocklists", "tenantID", tenantID)
		}

		interfaceSlice := make([]interface{}, 0, len(blockIDs))
		for _, id := range blockIDs {
			interfaceSlice = append(interfaceSlice, id)
		}

		listMutex := sync.Mutex{}
		blocklist := make([]*encoding.BlockMeta, 0, len(blockIDs))
		compactedBlocklist := make([]*encoding.CompactedBlockMeta, 0, len(blockIDs))
		_, err = rw.pool.RunJobs(ctx, interfaceSlice, func(ctx context.Context, payload interface{}) ([]byte, error) {
			blockID := payload.(uuid.UUID)

			var compactedBlockMeta *encoding.CompactedBlockMeta
			blockMeta, err := rw.r.BlockMeta(ctx, blockID, tenantID)
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

		metricBlocklistLength.WithLabelValues(tenantID).Set(float64(len(blocklist)))

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

// todo: pass a context/chan in to cancel this cleanly
//  once a maintenance cycle cleanup any blocks
func (rw *readerWriter) retentionLoop() {
	ticker := time.NewTicker(rw.cfg.BlocklistPoll)
	for range ticker.C {
		rw.doRetention()
	}
}

func (rw *readerWriter) doRetention() {
	tenants := rw.blocklistTenants()

	// todo: continued abuse of runJobs.  need a runAllJobs() method or something
	_, err := rw.pool.RunJobs(context.TODO(), tenants, func(_ context.Context, payload interface{}) ([]byte, error) {
		start := time.Now()
		defer func() { metricRetentionDuration.Observe(time.Since(start).Seconds()) }()

		tenantID := payload.(string)

		// iterate through block list.  make compacted anything that is past retention.
		cutoff := time.Now().Add(-rw.compactorCfg.BlockRetention)
		blocklist := rw.blocklist(tenantID)
		for _, b := range blocklist {
			if b.EndTime.Before(cutoff) {
				level.Info(rw.logger).Log("msg", "marking block for deletion", "blockID", b.BlockID, "tenantID", tenantID)
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
				level.Info(rw.logger).Log("msg", "deleting block", "blockID", b.BlockID, "tenantID", tenantID)
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

func (rw *readerWriter) blocklist(tenantID string) []*encoding.BlockMeta {
	rw.blockListsMtx.Lock()
	defer rw.blockListsMtx.Unlock()

	if tenantID == "" {
		return nil
	}

	copiedBlocklist := make([]*encoding.BlockMeta, 0, len(rw.blockLists[tenantID]))
	copiedBlocklist = append(copiedBlocklist, rw.blockLists[tenantID]...)
	return copiedBlocklist
}

// todo:  make separate compacted list mutex?
func (rw *readerWriter) compactedBlocklist(tenantID string) []*encoding.CompactedBlockMeta {
	rw.blockListsMtx.Lock()
	defer rw.blockListsMtx.Unlock()

	if tenantID == "" {
		return nil
	}

	copiedBlocklist := make([]*encoding.CompactedBlockMeta, 0, len(rw.compactedBlockLists[tenantID]))
	copiedBlocklist = append(copiedBlocklist, rw.compactedBlockLists[tenantID]...)

	return copiedBlocklist
}
