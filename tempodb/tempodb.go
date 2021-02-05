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

	"github.com/cortexproject/cortex/pkg/util"
	"github.com/opentracing/opentracing-go"
	ot_log "github.com/opentracing/opentracing-go/log"

	"github.com/grafana/tempo/pkg/boundedwaitgroup"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/azure"
	"github.com/grafana/tempo/tempodb/backend/cache"
	"github.com/grafana/tempo/tempodb/backend/cache/memcached"
	"github.com/grafana/tempo/tempodb/backend/cache/redis"
	"github.com/grafana/tempo/tempodb/backend/gcs"
	"github.com/grafana/tempo/tempodb/backend/local"
	"github.com/grafana/tempo/tempodb/backend/s3"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/grafana/tempo/tempodb/pool"
	"github.com/grafana/tempo/tempodb/wal"
)

const (
	BlockIDMin = "00000000-0000-0000-0000-000000000000"
	BlockIDMax = "FFFFFFFF-FFFF-FFFF-FFFF-FFFFFFFFFFFF"
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
	WriteBlock(ctx context.Context, block WriteableBlock) error
	CompleteBlock(block *wal.AppendBlock, combiner common.ObjectCombiner) (*encoding.CompleteBlock, error)
	WAL() *wal.WAL
}

type Reader interface {
	Find(ctx context.Context, tenantID string, id common.ID, blockStart string, blockEnd string) ([][]byte, error)
	Shutdown()
}

type Compactor interface {
	EnableCompaction(cfg *CompactorConfig, sharder CompactorSharder)
}

type CompactorSharder interface {
	Combine(objA []byte, objB []byte) []byte
	Owns(hash string) bool
}

type WriteableBlock interface {
	Write(ctx context.Context, w backend.Writer) error
}

type readerWriter struct {
	r backend.Reader
	w backend.Writer
	c backend.Compactor

	wal  *wal.WAL
	pool *pool.Pool

	logger        log.Logger
	cfg           *Config
	blockLists    map[string][]*backend.BlockMeta
	blockListsMtx sync.Mutex

	compactorCfg          *CompactorConfig
	compactedBlockLists   map[string][]*backend.CompactedBlockMeta
	compactorSharder      CompactorSharder
	compactorTenantOffset uint
}

// New creates a new tempodb
func New(cfg *Config, logger log.Logger) (Reader, Writer, Compactor, error) {
	var r backend.Reader
	var w backend.Writer
	var c backend.Compactor

	err := validateConfig(cfg)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("invalid config while creating tempodb: %w", err)
	}

	switch cfg.Backend {
	case "local":
		r, w, c, err = local.New(cfg.Local)
	case "gcs":
		r, w, c, err = gcs.New(cfg.GCS)
	case "s3":
		r, w, c, err = s3.New(cfg.S3)
	case "azure":
		r, w, c, err = azure.New(cfg.Azure)
	default:
		err = fmt.Errorf("unknown backend %s", cfg.Backend)
	}

	if err != nil {
		return nil, nil, nil, err
	}

	var cacheBackend cache.Client

	switch cfg.Cache {
	case "redis":
		cacheBackend = redis.NewClient(cfg.Redis, logger)
	case "memcached":
		cacheBackend = memcached.NewClient(cfg.Memcached, logger)
	}

	if cacheBackend != nil {
		r, w, err = cache.NewCache(r, w, cacheBackend)
		if err != nil {
			return nil, nil, nil, err
		}
	}

	if cfg.BlocklistPollConcurrency == 0 {
		cfg.BlocklistPollConcurrency = DefaultBlocklistPollConcurrency
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

func (rw *readerWriter) WriteBlock(ctx context.Context, c WriteableBlock) error {
	return c.Write(ctx, rw.w)
}

func (rw *readerWriter) CompleteBlock(block *wal.AppendBlock, combiner common.ObjectCombiner) (*encoding.CompleteBlock, error) {
	return block.Complete(rw.cfg.Block, rw.wal, combiner)
}

func (rw *readerWriter) WAL() *wal.WAL {
	return rw.wal
}

func (rw *readerWriter) Find(ctx context.Context, tenantID string, id common.ID, blockStart string, blockEnd string) ([][]byte, error) {
	// tracing instrumentation
	logger := util.WithContext(ctx, util.Logger)
	span, ctx := opentracing.StartSpanFromContext(ctx, "store.Find")
	defer span.Finish()

	blockStartUUID, err := uuid.Parse(blockStart)
	if err != nil {
		return nil, err
	}
	blockStartBytes, err := blockStartUUID.MarshalBinary()
	if err != nil {
		return nil, err
	}
	blockEndUUID, err := uuid.Parse(blockEnd)
	if err != nil {
		return nil, err
	}
	blockEndBytes, err := blockEndUUID.MarshalBinary()
	if err != nil {
		return nil, err
	}

	rw.blockListsMtx.Lock()
	blocklist, found := rw.blockLists[tenantID]
	copiedBlocklist := make([]interface{}, 0, len(blocklist))

	for _, b := range blocklist {
		// if in range copy
		if bytes.Compare(id, b.MinID) != -1 && bytes.Compare(id, b.MaxID) != 1 {
			blockIDBytes, _ := b.BlockID.MarshalBinary()
			// check block is in shard boundaries
			// blockStartBytes <= blockIDBytes <= blockEndBytes
			if (bytes.Compare(blockIDBytes, blockStartBytes) >= 0) && (bytes.Compare(blockIDBytes, blockEndBytes) <= 0) {
				copiedBlocklist = append(copiedBlocklist, b)
			}
		}
	}
	rw.blockListsMtx.Unlock()

	// deliberately placed outside the blocklist mtx unlock
	if !found {
		return nil, nil
	}

	partialTraces, err := rw.pool.RunJobs(ctx, copiedBlocklist, func(ctx context.Context, payload interface{}) ([]byte, error) {
		meta := payload.(*backend.BlockMeta)
		block, err := encoding.NewBackendBlock(meta, rw.r)
		if err != nil {
			return nil, err
		}

		foundObject, err := block.Find(ctx, id)
		if err != nil {
			return nil, err
		}

		level.Info(logger).Log("msg", "searching for trace in block", "findTraceID", hex.EncodeToString(id), "block", meta.BlockID, "found", foundObject != nil)
		span.LogFields(
			ot_log.String("msg", "searching for trace in block"),
			ot_log.String("blockID", meta.BlockID.String()),
			ot_log.Bool("found", foundObject != nil),
			ot_log.Int("bytes", len(foundObject)))

		return foundObject, nil
	})

	return partialTraces, err
}

func (rw *readerWriter) Shutdown() {
	// todo: stop blocklist poll
	rw.pool.Shutdown()
	rw.r.Shutdown()
}

func (rw *readerWriter) EnableCompaction(cfg *CompactorConfig, c CompactorSharder) {
	// Set default if needed. This is mainly for tests.
	if cfg.RetentionConcurrency == 0 {
		cfg.RetentionConcurrency = DefaultRetentionConcurrency
	}

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

	rw.cleanMissingTenants(tenants)

	for _, tenantID := range tenants {

		newBlockList, newCompactedBlockList := rw.pollTenant(ctx, tenantID)

		metricBlocklistLength.WithLabelValues(tenantID).Set(float64(len(newBlockList)))

		rw.blockListsMtx.Lock()
		rw.blockLists[tenantID] = newBlockList
		rw.compactedBlockLists[tenantID] = newCompactedBlockList
		rw.blockListsMtx.Unlock()
	}
}

func (rw *readerWriter) pollTenant(ctx context.Context, tenantID string) ([]*backend.BlockMeta, []*backend.CompactedBlockMeta) {
	blockIDs, err := rw.r.Blocks(ctx, tenantID)
	if err != nil {
		metricBlocklistErrors.WithLabelValues(tenantID).Inc()
		level.Error(rw.logger).Log("msg", "error polling blocklist", "tenantID", tenantID, "err", err)
		return []*backend.BlockMeta{}, []*backend.CompactedBlockMeta{}
	}

	bg := boundedwaitgroup.New(rw.cfg.BlocklistPollConcurrency)
	chMeta := make(chan *backend.BlockMeta, len(blockIDs))
	chCompactedMeta := make(chan *backend.CompactedBlockMeta, len(blockIDs))

	for _, blockID := range blockIDs {
		bg.Add(1)
		go func(b uuid.UUID) {
			defer bg.Done()
			m, cm := rw.pollBlock(ctx, tenantID, b)
			if m != nil {
				chMeta <- m
			} else if cm != nil {
				chCompactedMeta <- cm
			}
		}(blockID)
	}

	bg.Wait()
	close(chMeta)
	close(chCompactedMeta)

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

	return newBlockList, newCompactedBlocklist
}

func (rw *readerWriter) pollBlock(ctx context.Context, tenantID string, blockID uuid.UUID) (*backend.BlockMeta, *backend.CompactedBlockMeta) {
	var compactedBlockMeta *backend.CompactedBlockMeta
	blockMeta, err := rw.r.BlockMeta(ctx, blockID, tenantID)
	// if the normal meta doesn't exist maybe it's compacted.
	if err == backend.ErrMetaDoesNotExist {
		blockMeta = nil
		compactedBlockMeta, err = rw.c.CompactedBlockMeta(blockID, tenantID)
	}

	// blocks in intermediate states may not have a compacted or normal block meta.
	//   this is not necessarily an error, just bail out
	if err == backend.ErrMetaDoesNotExist {
		return nil, nil
	}

	if err != nil {
		metricBlocklistErrors.WithLabelValues(tenantID).Inc()
		level.Error(rw.logger).Log("msg", "failed to retrieve block meta", "tenantID", tenantID, "blockID", blockID, "err", err)
		return nil, nil
	}

	return blockMeta, compactedBlockMeta
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

func (rw *readerWriter) cleanMissingTenants(tenants []string) {
	tenantSet := make(map[string]struct{})
	for _, tenantID := range tenants {
		tenantSet[tenantID] = struct{}{}
	}

	for tenantID := range rw.blockLists {
		if _, present := tenantSet[tenantID]; !present {
			rw.blockListsMtx.Lock()
			delete(rw.blockLists, tenantID)
			rw.blockListsMtx.Unlock()
			level.Info(rw.logger).Log("msg", "deleted in-memory blocklists", "tenantID", tenantID)
		}
	}

	for tenantID := range rw.compactedBlockLists {
		if _, present := tenantSet[tenantID]; !present {
			rw.blockListsMtx.Lock()
			delete(rw.compactedBlockLists, tenantID)
			rw.blockListsMtx.Unlock()
			level.Info(rw.logger).Log("msg", "deleted in-memory compacted blocklists", "tenantID", tenantID)
		}
	}
}

// updateBlocklist Add and remove regular or compacted blocks from the in-memory blocklist.
// Changes are temporary and will be overwritten on the next poll.
func (rw *readerWriter) updateBlocklist(tenantID string, add []*backend.BlockMeta, remove []*backend.BlockMeta, compactedAdd []*backend.CompactedBlockMeta) {
	if tenantID == "" {
		return
	}

	rw.blockListsMtx.Lock()
	defer rw.blockListsMtx.Unlock()

	// ******** Regular blocks ********
	blocklist := rw.blockLists[tenantID]

	matchedRemovals := make(map[uuid.UUID]struct{})
	for _, b := range blocklist {
		for _, rem := range remove {
			if b.BlockID == rem.BlockID {
				matchedRemovals[rem.BlockID] = struct{}{}
			}
		}
	}

	newblocklist := make([]*backend.BlockMeta, 0, len(blocklist)-len(matchedRemovals)+len(add))
	for _, b := range blocklist {
		if _, ok := matchedRemovals[b.BlockID]; !ok {
			newblocklist = append(newblocklist, b)
		}
	}
	newblocklist = append(newblocklist, add...)
	rw.blockLists[tenantID] = newblocklist

	// ******** Compacted blocks ********
	rw.compactedBlockLists[tenantID] = append(rw.compactedBlockLists[tenantID], compactedAdd...)
}
