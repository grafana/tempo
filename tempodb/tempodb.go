package tempodb

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/atomic"

	log_util "github.com/cortexproject/cortex/pkg/util/log"
	"github.com/grafana/tempo/pkg/boundedwaitgroup"
	pkg_cache "github.com/grafana/tempo/pkg/cache"
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
	"github.com/grafana/tempo/tempodb/search"
	"github.com/grafana/tempo/tempodb/wal"
	"github.com/opentracing/opentracing-go"
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
	CompleteSearchBlockWithBackend(block *search.StreamingSearchBlock, blockID uuid.UUID, tenantID string, r backend.Reader, w backend.Writer) (*search.BackendSearchBlock, error)
	WAL() *wal.WAL
}

type IterateObjectCallback func(id common.ID, obj []byte) bool

type Reader interface {
	Find(ctx context.Context, tenantID string, id common.ID, blockStart string, blockEnd string) ([][]byte, []string, []error, error)
	IterateObjects(ctx context.Context, meta *backend.BlockMeta, startPage int, totalPages int, callback IterateObjectCallback) error
	BlockMetas(tenantID string) []*backend.BlockMeta
	EnablePolling(sharder blocklist.JobSharder)

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
	BlockMeta() *backend.BlockMeta
	Write(ctx context.Context, w backend.Writer) error
}

type readerWriter struct {
	r backend.Reader
	w backend.Writer
	c backend.Compactor

	uncachedReader backend.Reader
	uncachedWriter backend.Writer

	wal  *wal.WAL
	pool *pool.Pool

	logger log.Logger
	cfg    *Config

	blocklistPoller *blocklist.Poller
	blocklist       *blocklist.List

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
	uncachedWriter := backend.NewWriter(rawW)

	var cacheBackend pkg_cache.Cache

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

	r := backend.NewReader(rawR)
	w := backend.NewWriter(rawW)
	rw := &readerWriter{
		c:              c,
		r:              r,
		uncachedReader: uncachedReader,
		uncachedWriter: uncachedWriter,
		w:              w,
		cfg:            cfg,
		logger:         logger,
		pool:           pool.NewPool(cfg.Pool),
		blocklist:      blocklist.New(),
	}

	rw.wal, err = wal.New(rw.cfg.WAL)
	if err != nil {
		return nil, nil, nil, err
	}

	return rw, rw, rw, nil
}

func (rw *readerWriter) WriteBlock(ctx context.Context, c WriteableBlock) error {
	w := rw.getWriterForBlock(c.BlockMeta(), time.Now())
	return c.Write(ctx, w)
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

	iter, err := block.Iterator(combiner)
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

func (rw *readerWriter) CompleteSearchBlockWithBackend(block *search.StreamingSearchBlock, blockID uuid.UUID, tenantID string, r backend.Reader, w backend.Writer) (*search.BackendSearchBlock, error) {
	err := search.NewBackendSearchBlock(block, w, blockID, tenantID, rw.cfg.Block.SearchEncoding, rw.cfg.Block.SearchPageSizeBytes)
	if err != nil {
		return nil, err
	}

	b := search.OpenBackendSearchBlock(blockID, tenantID, r)
	return b, nil
}

func (rw *readerWriter) WAL() *wal.WAL {
	return rw.wal
}

func (rw *readerWriter) BlockMetas(tenantID string) []*backend.BlockMeta {
	return rw.blocklist.Metas(tenantID)
}

func (rw *readerWriter) Find(ctx context.Context, tenantID string, id common.ID, blockStart string, blockEnd string) ([][]byte, []string, []error, error) {
	// tracing instrumentation
	logger := log_util.WithContext(ctx, log_util.Logger)
	span, ctx := opentracing.StartSpanFromContext(ctx, "store.Find")
	defer span.Finish()

	blockStartUUID, err := uuid.Parse(blockStart)
	if err != nil {
		return nil, nil, nil, err
	}
	blockStartBytes, err := blockStartUUID.MarshalBinary()
	if err != nil {
		return nil, nil, nil, err
	}
	blockEndUUID, err := uuid.Parse(blockEnd)
	if err != nil {
		return nil, nil, nil, err
	}
	blockEndBytes, err := blockEndUUID.MarshalBinary()
	if err != nil {
		return nil, nil, nil, err
	}

	// gather appropriate blocks
	blocklist := rw.blocklist.Metas(tenantID)
	compactedBlocklist := rw.blocklist.CompactedMetas(tenantID)
	copiedBlocklist := make([]interface{}, 0, len(blocklist))
	blocksSearched := 0
	compactedBlocksSearched := 0

	for _, b := range blocklist {
		if includeBlock(b, id, blockStartBytes, blockEndBytes) {
			copiedBlocklist = append(copiedBlocklist, b)
			blocksSearched++
		}
	}
	for _, c := range compactedBlocklist {
		if includeCompactedBlock(c, id, blockStartBytes, blockEndBytes, rw.cfg.BlocklistPoll) {
			copiedBlocklist = append(copiedBlocklist, &c.BlockMeta)
			compactedBlocksSearched++
		}
	}
	if len(copiedBlocklist) == 0 {
		return nil, nil, nil, nil
	}

	curTime := time.Now()
	partialTraces, dataEncodings, funcErrs, err := rw.pool.RunJobs(ctx, copiedBlocklist, func(ctx context.Context, payload interface{}) ([]byte, string, error) {
		meta := payload.(*backend.BlockMeta)
		r := rw.getReaderForBlock(meta, curTime)
		block, err := encoding.NewBackendBlock(meta, r)
		if err != nil {
			return nil, "", err
		}

		foundObject, err := block.Find(ctx, id)
		if err != nil {
			return nil, "", err
		}

		level.Info(logger).Log("msg", "searching for trace in block", "findTraceID", hex.EncodeToString(id), "block", meta.BlockID, "found", foundObject != nil)
		return foundObject, meta.DataEncoding, nil
	})

	size := 0
	for _, b := range partialTraces {
		size += len(b)
	}
	span.SetTag("bytesFound", size)
	span.SetTag("blockErrs", len(funcErrs))
	span.SetTag("liveBlocks", len(blocklist))
	span.SetTag("liveBlocksSearched", blocksSearched)
	span.SetTag("compactedBlocks", len(compactedBlocklist))
	span.SetTag("compactedBlocksSearched", compactedBlocksSearched)

	return partialTraces, dataEncodings, funcErrs, err
}

// IterateObjects iterates through all objects for the provided blockID, startPage and totalPages
// calling the provided callback for each object. If the callback returns true then iteration
// is stopped and the function returns. Note that the callback needs to be threadsafe as it is called
// concurrently.
func (rw *readerWriter) IterateObjects(ctx context.Context, meta *backend.BlockMeta, startPage int, totalPages int, callback IterateObjectCallback) error {
	block, err := encoding.NewBackendBlock(meta, rw.r)
	if err != nil {
		return err
	}

	// todo: a graduated chunk size would allow for faster iteration
	iter, err := block.PartialIterator(rw.cfg.Search.ChunkSizeBytes, startPage, totalPages)
	if err != nil {
		return err
	}
	iter = encoding.NewPrefetchIterator(ctx, iter, rw.cfg.Search.PrefetchTraceCount)
	wg := boundedwaitgroup.New(5)
	done := atomic.Bool{}
	for {
		id, obj, err := iter.Next(ctx)
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("error iterating %s, %w", meta.BlockID, err)
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			isDone := callback(id, obj)
			if isDone {
				done.Store(true)
			}
		}()

		if done.Load() {
			break
		}
	}
	wg.Wait()

	return nil
}

func (rw *readerWriter) Shutdown() {
	// todo: stop blocklist poll
	rw.pool.Shutdown()
	rw.r.Shutdown()
}

// EnableCompaction activates the compaction/retention loops
func (rw *readerWriter) EnableCompaction(cfg *CompactorConfig, c CompactorSharder, overrides CompactorOverrides) {
	// Set default if needed. This is mainly for tests.
	if cfg.RetentionConcurrency == 0 {
		cfg.RetentionConcurrency = DefaultRetentionConcurrency
	}

	rw.compactorCfg = cfg
	rw.compactorSharder = c
	rw.compactorOverrides = overrides

	if rw.cfg.BlocklistPoll == 0 {
		level.Info(rw.logger).Log("msg", "polling cycle unset. compaction and retention disabled")
		return
	}

	if cfg != nil {
		level.Info(rw.logger).Log("msg", "compaction and retention enabled.")
		go rw.compactionLoop()
		go rw.retentionLoop()
	}
}

// EnablePolling activates the polling loop. Pass nil if this component
//  should never be a tenant index builder.
func (rw *readerWriter) EnablePolling(sharder blocklist.JobSharder) {
	if sharder == nil {
		sharder = blocklist.OwnsNothingSharder
	}

	if rw.cfg.BlocklistPoll == 0 {
		rw.cfg.BlocklistPoll = DefaultBlocklistPoll
	}

	if rw.cfg.BlocklistPollConcurrency == 0 {
		rw.cfg.BlocklistPollConcurrency = DefaultBlocklistPollConcurrency
	}

	if rw.cfg.BlocklistPollTenantIndexBuilders <= 0 {
		rw.cfg.BlocklistPollTenantIndexBuilders = DefaultTenantIndexBuilders
	}

	level.Info(rw.logger).Log("msg", "polling enabled", "interval", rw.cfg.BlocklistPoll, "concurrency", rw.cfg.BlocklistPollConcurrency)

	blocklistPoller := blocklist.NewPoller(&blocklist.PollerConfig{
		PollConcurrency:     rw.cfg.BlocklistPollConcurrency,
		PollFallback:        rw.cfg.BlocklistPollFallback,
		TenantIndexBuilders: rw.cfg.BlocklistPollTenantIndexBuilders,
		StaleTenantIndex:    rw.cfg.BlocklistPollStaleTenantIndex,
	}, sharder, rw.r, rw.c, rw.w, rw.logger)

	rw.blocklistPoller = blocklistPoller

	// do the first poll cycle synchronously. this will allow the caller to know
	// that when this method returns the block list is updated
	rw.pollBlocklist()

	go rw.pollingLoop()
}

func (rw *readerWriter) pollingLoop() {
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

	rw.blocklist.ApplyPollResults(blocklist, compactedBlocklist)
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

func (rw *readerWriter) getReaderForBlock(meta *backend.BlockMeta, curTime time.Time) backend.Reader {
	if rw.shouldCache(meta, curTime) {
		return rw.r
	}

	return rw.uncachedReader
}

func (rw *readerWriter) getWriterForBlock(meta *backend.BlockMeta, curTime time.Time) backend.Writer {
	if rw.shouldCache(meta, curTime) {
		return rw.w
	}

	return rw.uncachedWriter
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
