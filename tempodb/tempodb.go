package tempodb

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	cortex_cache "github.com/cortexproject/cortex/pkg/chunk/cache"
	log_util "github.com/cortexproject/cortex/pkg/util/log"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/azure"
	"github.com/grafana/tempo/tempodb/backend/cache"
	"github.com/grafana/tempo/tempodb/backend/cache/memcached"
	"github.com/grafana/tempo/tempodb/backend/cache/redis"
	"github.com/grafana/tempo/tempodb/backend/gcs"
	"github.com/grafana/tempo/tempodb/backend/local"
	"github.com/grafana/tempo/tempodb/backend/s3"
	"github.com/grafana/tempo/tempodb/blocklist"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/grafana/tempo/tempodb/pool"
	"github.com/grafana/tempo/tempodb/wal"
	"github.com/opentracing/opentracing-go"
	ot_log "github.com/opentracing/opentracing-go/log"
)

const (
	// BlockIDMin is the minimum possible value for a block id as a string
	BlockIDMin = "00000000-0000-0000-0000-000000000000"
	// BlockIDMax is the maximum possible value for a block id as a string
	BlockIDMax = "FFFFFFFF-FFFF-FFFF-FFFF-FFFFFFFFFFFF"
)

var (
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
	CompleteBlock(block *wal.AppendBlock, combiner common.ObjectCombiner) (*encoding.BackendBlock, error)
	CompleteBlockWithBackend(ctx context.Context, block *wal.AppendBlock, combiner common.ObjectCombiner, r backend.Reader, w backend.Writer) (*encoding.BackendBlock, error)
	WAL() *wal.WAL
}

type Reader interface {
	Find(ctx context.Context, tenantID string, id common.ID, blockStart string, blockEnd string) ([][]byte, []string, error)
	Shutdown()
}

type Compactor interface {
	EnableCompaction(cfg *CompactorConfig, sharder CompactorSharder, overrides CompactorOverrides)
}

type CompactorSharder interface {
	common.ObjectCombiner
	Owns(hash string) bool
}

type CompactorOverrides interface {
	BlockRetentionForTenant(tenantID string) time.Duration
}

type WriteableBlock interface {
	Write(ctx context.Context, w backend.Writer) error
}

type readerWriter struct {
	r backend.Reader
	w backend.Writer
	c backend.Compactor

	uncachedReader backend.Reader

	wal  *wal.WAL
	pool *pool.Pool

	logger log.Logger
	cfg    *Config

	blockListsMtx       sync.Mutex
	blockLists          blocklist.PerTenant
	compactedBlockLists blocklist.PerTenantCompacted
	blocklistPoller     *blocklist.Poller

	compactorCfg          *CompactorConfig
	compactorSharder      CompactorSharder
	compactorOverrides    CompactorOverrides
	compactorTenantOffset uint
}

// New creates a new tempodb
func New(cfg *Config, logger log.Logger) (Reader, Writer, Compactor, error) {
	var rawR backend.RawReader
	var rawW backend.RawWriter
	var c backend.Compactor

	err := validateConfig(cfg)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("invalid config while creating tempodb: %w", err)
	}

	switch cfg.Backend {
	case "local":
		rawR, rawW, c, err = local.New(cfg.Local)
	case "gcs":
		rawR, rawW, c, err = gcs.New(cfg.GCS)
	case "s3":
		rawR, rawW, c, err = s3.New(cfg.S3)
	case "azure":
		rawR, rawW, c, err = azure.New(cfg.Azure)
	default:
		err = fmt.Errorf("unknown backend %s", cfg.Backend)
	}

	if err != nil {
		return nil, nil, nil, err
	}

	uncachedReader := backend.NewReader(rawR)

	var cacheBackend cortex_cache.Cache

	switch cfg.Cache {
	case "redis":
		cacheBackend = redis.NewClient(cfg.Redis, cfg.BackgroundCache, logger)
	case "memcached":
		cacheBackend = memcached.NewClient(cfg.Memcached, cfg.BackgroundCache, logger)
	}

	if cacheBackend != nil {
		rawR, rawW, err = cache.NewCache(rawR, rawW, cacheBackend)
		if err != nil {
			return nil, nil, nil, err
		}
	}

	if cfg.BlocklistPollConcurrency == 0 {
		cfg.BlocklistPollConcurrency = DefaultBlocklistPollConcurrency
	}

	r := backend.NewReader(rawR)
	w := backend.NewWriter(rawW)
	rw := &readerWriter{
		c:                   c,
		r:                   r,
		uncachedReader:      uncachedReader,
		w:                   w,
		cfg:                 cfg,
		logger:              logger,
		pool:                pool.NewPool(cfg.Pool),
		blockLists:          make(blocklist.PerTenant),
		compactedBlockLists: make(blocklist.PerTenantCompacted),
		blocklistPoller:     blocklist.NewPoller(cfg.BlocklistPollConcurrency, r, c),
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

// CompleteBlock iterates the given WAL block and flushes it to the TempoDB backend.
func (rw *readerWriter) CompleteBlock(block *wal.AppendBlock, combiner common.ObjectCombiner) (*encoding.BackendBlock, error) {
	return rw.CompleteBlockWithBackend(context.TODO(), block, combiner, rw.r, rw.w)
}

// CompleteBlock iterates the given WAL block but flushes it to the given backend instead of the default TempoDB backend. The
// new block will have the same ID as the input block.
func (rw *readerWriter) CompleteBlockWithBackend(ctx context.Context, block *wal.AppendBlock, combiner common.ObjectCombiner, r backend.Reader, w backend.Writer) (*encoding.BackendBlock, error) {
	meta := block.Meta()
	blockID := meta.BlockID
	tenantID := meta.TenantID

	// Default and nil check is primarily to make testing easier.
	flushSize := DefaultFlushSizeBytes
	if rw.compactorCfg != nil && rw.compactorCfg.FlushSizeBytes > 0 {
		flushSize = rw.compactorCfg.FlushSizeBytes
	}

	iter, err := block.GetIterator(combiner)
	if err != nil {
		return nil, errors.Wrap(err, "error getting completing block iterator")
	}
	defer iter.Close()

	newBlock, err := encoding.NewStreamingBlock(rw.cfg.Block, blockID, tenantID, []*backend.BlockMeta{meta}, meta.TotalObjects)
	if err != nil {
		return nil, errors.Wrap(err, "error creating compactor block")
	}

	var tracker backend.AppendTracker
	for {
		id, data, err := iter.Next(ctx)
		if err != nil && err != io.EOF {
			return nil, errors.Wrap(err, "error iterating")
		}

		if id == nil {
			break
		}

		err = newBlock.AddObject(id, data)
		if err != nil {
			return nil, errors.Wrap(err, "error adding object to compactor block")
		}

		if newBlock.CurrentBufferLength() > int(flushSize) {
			tracker, _, err = newBlock.FlushBuffer(ctx, tracker, w)
			if err != nil {
				return nil, errors.Wrap(err, "error flushing compactor block")
			}
		}
	}

	_, err = newBlock.Complete(ctx, tracker, w)
	if err != nil {
		return nil, errors.Wrap(err, "error completing compactor block")
	}

	backendBlock, err := encoding.NewBackendBlock(newBlock.BlockMeta(), r)
	if err != nil {
		return nil, errors.Wrap(err, "error creating creating backend block")
	}

	return backendBlock, nil
}

func (rw *readerWriter) WAL() *wal.WAL {
	return rw.wal
}

func (rw *readerWriter) Find(ctx context.Context, tenantID string, id common.ID, blockStart string, blockEnd string) ([][]byte, []string, error) {
	// tracing instrumentation
	logger := log_util.WithContext(ctx, log_util.Logger)
	span, ctx := opentracing.StartSpanFromContext(ctx, "store.Find")
	defer span.Finish()

	blockStartUUID, err := uuid.Parse(blockStart)
	if err != nil {
		return nil, nil, err
	}
	blockStartBytes, err := blockStartUUID.MarshalBinary()
	if err != nil {
		return nil, nil, err
	}
	blockEndUUID, err := uuid.Parse(blockEnd)
	if err != nil {
		return nil, nil, err
	}
	blockEndBytes, err := blockEndUUID.MarshalBinary()
	if err != nil {
		return nil, nil, err
	}

	rw.blockListsMtx.Lock()
	blocklist, found := rw.blockLists[tenantID]
	copiedBlocklist := make([]interface{}, 0, len(blocklist))

	for _, b := range blocklist {
		if includeBlock(b, id, blockStartBytes, blockEndBytes) {
			copiedBlocklist = append(copiedBlocklist, b)
		}
	}

	compactedBlocklist := rw.compactedBlockLists[tenantID]
	for _, c := range compactedBlocklist {
		if includeCompactedBlock(c, id, blockStartBytes, blockEndBytes, rw.cfg.BlocklistPoll) {
			copiedBlocklist = append(copiedBlocklist, &c.BlockMeta)
		}
	}
	rw.blockListsMtx.Unlock()

	// deliberately placed outside the blocklist mtx unlock
	if !found {
		return nil, nil, nil
	}

	curTime := time.Now()
	partialTraces, dataEncodings, err := rw.pool.RunJobs(ctx, copiedBlocklist, func(ctx context.Context, payload interface{}) ([]byte, string, error) {
		meta := payload.(*backend.BlockMeta)
		var block *encoding.BackendBlock
		var err error
		if rw.shouldCache(meta, curTime) {
			block, err = encoding.NewBackendBlock(meta, rw.r)
		} else {
			block, err = encoding.NewBackendBlock(meta, rw.uncachedReader)
		}
		if err != nil {
			return nil, "", err
		}

		foundObject, err := block.Find(ctx, id)
		if err != nil {
			return nil, "", err
		}

		level.Info(logger).Log("msg", "searching for trace in block", "findTraceID", hex.EncodeToString(id), "block", meta.BlockID, "found", foundObject != nil)
		span.LogFields(
			ot_log.String("msg", "searching for trace in block"),
			ot_log.String("blockID", meta.BlockID.String()),
			ot_log.Bool("found", foundObject != nil),
			ot_log.Int("bytes", len(foundObject)))

		return foundObject, meta.DataEncoding, nil
	})

	return partialTraces, dataEncodings, err
}

func (rw *readerWriter) Shutdown() {
	// todo: stop blocklist poll
	rw.pool.Shutdown()
	rw.r.Shutdown()
}

func (rw *readerWriter) EnableCompaction(cfg *CompactorConfig, c CompactorSharder, overrides CompactorOverrides) {
	// Set default if needed. This is mainly for tests.
	if cfg.RetentionConcurrency == 0 {
		cfg.RetentionConcurrency = DefaultRetentionConcurrency
	}

	rw.compactorCfg = cfg
	rw.compactorSharder = c
	rw.compactorOverrides = overrides

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
	blocklist, compactedBlocklist, err := rw.blocklistPoller.Do()

	if err != nil {
		level.Error(rw.logger).Log("msg", "failed to poll blocklist. using previously polled lists", "err", err)
		return
	}

	rw.blockListsMtx.Lock()
	defer rw.blockListsMtx.Unlock()

	rw.blockLists = blocklist
	rw.compactedBlockLists = compactedBlocklist
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

func (rw *readerWriter) shouldCache(meta *backend.BlockMeta, curTime time.Time) bool {
	// compaction level is _atleast_ CacheMinCompactionLevel
	if rw.cfg.CacheMinCompactionLevel > 0 && meta.CompactionLevel < rw.cfg.CacheMinCompactionLevel {
		return false
	}

	// block is not older than CacheMaxBlockAge
	if rw.cfg.CacheMaxBlockAge > 0 && curTime.Sub(meta.StartTime) > rw.cfg.CacheMaxBlockAge {
		return false
	}

	return true
}

// includeBlock indicates whether a given block should be included in a backend search
func includeBlock(b *backend.BlockMeta, id common.ID, blockStart []byte, blockEnd []byte) bool {
	if bytes.Compare(id, b.MinID) == -1 || bytes.Compare(id, b.MaxID) == 1 {
		return false
	}

	blockIDBytes, _ := b.BlockID.MarshalBinary()
	// check block is in shard boundaries
	// blockStartBytes <= blockIDBytes <= blockEndBytes
	if bytes.Compare(blockIDBytes, blockStart) == -1 || bytes.Compare(blockIDBytes, blockEnd) == 1 {
		return false
	}

	return true
}

// if block is compacted within lookback period, and is within shard ranges, include it in search
func includeCompactedBlock(c *backend.CompactedBlockMeta, id common.ID, blockStart []byte, blockEnd []byte, poll time.Duration) bool {
	lookback := time.Now().Add(-(2 * poll))
	if c.CompactedTime.Before(lookback) {
		return false
	}
	return includeBlock(&c.BlockMeta, id, blockStart, blockEnd)
}
