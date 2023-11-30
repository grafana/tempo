package tempodb

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	gkLog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/google/uuid"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/grafana/tempo/modules/cache/memcached"
	"github.com/grafana/tempo/modules/cache/redis"
	"github.com/grafana/tempo/pkg/cache"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/traceql"
	"github.com/grafana/tempo/pkg/util/log"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/azure"
	backend_cache "github.com/grafana/tempo/tempodb/backend/cache"
	"github.com/grafana/tempo/tempodb/backend/gcs"
	"github.com/grafana/tempo/tempodb/backend/local"
	"github.com/grafana/tempo/tempodb/backend/s3"
	"github.com/grafana/tempo/tempodb/blocklist"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/grafana/tempo/tempodb/pool"
	"github.com/grafana/tempo/tempodb/wal"
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
	CompleteBlock(ctx context.Context, block common.WALBlock) (common.BackendBlock, error)
	CompleteBlockWithBackend(ctx context.Context, block common.WALBlock, r backend.Reader, w backend.Writer) (common.BackendBlock, error)
	WAL() *wal.WAL
}

type IterateObjectCallback func(id common.ID, obj []byte) bool

type Reader interface {
	Find(ctx context.Context, tenantID string, id common.ID, blockStart string, blockEnd string, timeStart int64, timeEnd int64, opts common.SearchOptions) ([]*tempopb.Trace, []error, error)
	Search(ctx context.Context, meta *backend.BlockMeta, req *tempopb.SearchRequest, opts common.SearchOptions) (*tempopb.SearchResponse, error)
	Fetch(ctx context.Context, meta *backend.BlockMeta, req traceql.FetchSpansRequest, opts common.SearchOptions) (traceql.FetchSpansResponse, error)
	BlockMetas(tenantID string) []*backend.BlockMeta
	EnablePolling(ctx context.Context, sharder blocklist.JobSharder)

	Shutdown()
}

type Compactor interface {
	EnableCompaction(ctx context.Context, cfg *CompactorConfig, sharder CompactorSharder, overrides CompactorOverrides) error
}

type CompactorSharder interface {
	Combine(dataEncoding string, tenantID string, objs ...[]byte) ([]byte, bool, error)
	Owns(hash string) bool
	RecordDiscardedSpans(count int, tenantID string, traceID string, rootSpanName string, rootServiceName string)
}

type CompactorOverrides interface {
	BlockRetentionForTenant(tenantID string) time.Duration
	MaxBytesPerTraceForTenant(tenantID string) int
	MaxCompactionRangeForTenant(tenantID string) time.Duration
}

type WriteableBlock interface {
	BlockMeta() *backend.BlockMeta
	Write(ctx context.Context, w backend.Writer) error
}

type readerWriter struct {
	r backend.Reader
	w backend.Writer
	c backend.Compactor

	wal  *wal.WAL
	pool *pool.Pool

	logger gkLog.Logger
	cfg    *Config

	blocklistPoller *blocklist.Poller
	blocklist       *blocklist.List

	compactorCfg          *CompactorConfig
	compactorSharder      CompactorSharder
	compactorOverrides    CompactorOverrides
	compactorTenantOffset uint
}

// New creates a new tempodb
func New(cfg *Config, cacheProvider cache.Provider, logger gkLog.Logger) (Reader, Writer, Compactor, error) {
	var rawR backend.RawReader
	var rawW backend.RawWriter
	var c backend.Compactor

	err := validateConfig(cfg)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("invalid config while creating tempodb: %w", err)
	}

	switch cfg.Backend {
	case backend.Local:
		rawR, rawW, c, err = local.New(cfg.Local)
	case backend.GCS:
		rawR, rawW, c, err = gcs.New(cfg.GCS)
	case backend.S3:
		rawR, rawW, c, err = s3.New(cfg.S3)
	case backend.Azure:
		rawR, rawW, c, err = azure.New(cfg.Azure)
	default:
		err = fmt.Errorf("unknown backend %s", cfg.Backend)
	}

	if err != nil {
		return nil, nil, nil, err
	}

	// build a caching layer if we have a provider
	if cacheProvider != nil {
		legacyCache, roles, err := createLegacyCache(cfg, logger)
		if err != nil {
			return nil, nil, nil, err
		}

		// inject legacy cache into the cache provider for the roles
		for _, role := range roles {
			err = cacheProvider.AddCache(role, legacyCache)
			if err != nil {
				return nil, nil, nil, fmt.Errorf("error adding legacy cache to provider: %w", err)
			}
		}

		rawR, rawW, err = backend_cache.NewCache(&cfg.BloomCacheCfg, rawR, rawW, cacheProvider, logger)
		if err != nil {
			return nil, nil, nil, err
		}
	}

	r := backend.NewReader(rawR)
	w := backend.NewWriter(rawW)
	rw := &readerWriter{
		c:         c,
		r:         r,
		w:         w,
		cfg:       cfg,
		logger:    logger,
		pool:      pool.NewPool(cfg.Pool),
		blocklist: blocklist.New(),
	}

	rw.wal, err = wal.New(rw.cfg.WAL)
	if err != nil {
		return nil, nil, nil, err
	}

	return rw, rw, rw, nil
}

func (rw *readerWriter) WriteBlock(ctx context.Context, c WriteableBlock) error {
	return c.Write(ctx, rw.w)
}

// CompleteBlock iterates the given WAL block and flushes it to the TempoDB backend.
func (rw *readerWriter) CompleteBlock(ctx context.Context, block common.WALBlock) (common.BackendBlock, error) {
	return rw.CompleteBlockWithBackend(ctx, block, rw.r, rw.w)
}

// CompleteBlock iterates the given WAL block but flushes it to the given backend instead of the default TempoDB backend. The
// new block will have the same ID as the input block.
func (rw *readerWriter) CompleteBlockWithBackend(ctx context.Context, block common.WALBlock, r backend.Reader, w backend.Writer) (common.BackendBlock, error) {
	// The destination block format:
	vers, err := encoding.FromVersion(rw.cfg.Block.Version)
	if err != nil {
		return nil, err
	}

	// force flush anything left in the wal
	err = block.Flush()
	if err != nil {
		return nil, fmt.Errorf("error flushing wal block: %w", err)
	}

	iter, err := block.Iterator()
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	walMeta := block.BlockMeta()

	inMeta := &backend.BlockMeta{
		// From the wal block
		TenantID:         walMeta.TenantID,
		BlockID:          walMeta.BlockID,
		TotalObjects:     walMeta.TotalObjects,
		StartTime:        walMeta.StartTime,
		EndTime:          walMeta.EndTime,
		DataEncoding:     walMeta.DataEncoding,
		DedicatedColumns: walMeta.DedicatedColumns,

		// Other
		Encoding: rw.cfg.Block.Encoding,
	}

	newMeta, err := vers.CreateBlock(ctx, rw.cfg.Block, inMeta, iter, r, w)
	if err != nil {
		return nil, fmt.Errorf("error creating block: %w", err)
	}

	backendBlock, err := encoding.OpenBlock(newMeta, r)
	if err != nil {
		return nil, fmt.Errorf("error opening new block: %w", err)
	}

	return backendBlock, nil
}

func (rw *readerWriter) WAL() *wal.WAL {
	return rw.wal
}

func (rw *readerWriter) BlockMetas(tenantID string) []*backend.BlockMeta {
	return rw.blocklist.Metas(tenantID)
}

func (rw *readerWriter) Find(ctx context.Context, tenantID string, id common.ID, blockStart string, blockEnd string, timeStart int64, timeEnd int64, opts common.SearchOptions) ([]*tempopb.Trace, []error, error) {
	// tracing instrumentation
	logger := log.WithContext(ctx, log.Logger)
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

	// gather appropriate blocks
	blocklist := rw.blocklist.Metas(tenantID)
	compactedBlocklist := rw.blocklist.CompactedMetas(tenantID)
	copiedBlocklist := make([]interface{}, 0, len(blocklist))
	blocksSearched := 0
	compactedBlocksSearched := 0

	for _, b := range blocklist {
		if includeBlock(b, id, blockStartBytes, blockEndBytes, timeStart, timeEnd) {
			copiedBlocklist = append(copiedBlocklist, b)
			blocksSearched++
		}
	}
	for _, c := range compactedBlocklist {
		if includeCompactedBlock(c, id, blockStartBytes, blockEndBytes, rw.cfg.BlocklistPoll, timeStart, timeEnd) {
			copiedBlocklist = append(copiedBlocklist, &c.BlockMeta)
			compactedBlocksSearched++
		}
	}
	if len(copiedBlocklist) == 0 {
		return nil, nil, nil
	}

	if rw.cfg != nil && rw.cfg.Search != nil {
		rw.cfg.Search.ApplyToOptions(&opts)
	}

	partialTraces, funcErrs, err := rw.pool.RunJobs(ctx, copiedBlocklist, func(ctx context.Context, payload interface{}) (interface{}, error) {
		meta := payload.(*backend.BlockMeta)
		block, err := encoding.OpenBlock(meta, rw.r)
		if err != nil {
			return nil, fmt.Errorf("error opening block for reading, blockID: %s: %w", meta.BlockID.String(), err)
		}

		foundObject, err := block.FindTraceByID(ctx, id, opts)
		if err != nil {
			return nil, fmt.Errorf("error finding trace by id, blockID: %s: %w", meta.BlockID.String(), err)
		}

		level.Info(logger).Log("msg", "searching for trace in block", "findTraceID", hex.EncodeToString(id), "block", meta.BlockID, "found", foundObject != nil)
		return foundObject, nil
	})

	partialTraceObjs := make([]*tempopb.Trace, len(partialTraces))
	for i := range partialTraces {
		partialTraceObjs[i] = partialTraces[i].(*tempopb.Trace)
	}

	span.SetTag("blockErrs", len(funcErrs))
	span.SetTag("liveBlocks", len(blocklist))
	span.SetTag("liveBlocksSearched", blocksSearched)
	span.SetTag("compactedBlocks", len(compactedBlocklist))
	span.SetTag("compactedBlocksSearched", compactedBlocksSearched)

	return partialTraceObjs, funcErrs, err
}

// Search the given block.  This method takes the pre-loaded block meta instead of a block ID, which
// eliminates a read per search request.
func (rw *readerWriter) Search(ctx context.Context, meta *backend.BlockMeta, req *tempopb.SearchRequest, opts common.SearchOptions) (*tempopb.SearchResponse, error) {
	block, err := encoding.OpenBlock(meta, rw.r)
	if err != nil {
		return nil, err
	}

	rw.cfg.Search.ApplyToOptions(&opts)
	return block.Search(ctx, req, opts)
}

// it only uses rw.r which has caching enabled
func (rw *readerWriter) Fetch(ctx context.Context, meta *backend.BlockMeta, req traceql.FetchSpansRequest, opts common.SearchOptions) (traceql.FetchSpansResponse, error) {
	block, err := encoding.OpenBlock(meta, rw.r)
	if err != nil {
		return traceql.FetchSpansResponse{}, err
	}

	rw.cfg.Search.ApplyToOptions(&opts)
	return block.Fetch(ctx, req, opts)
}

func (rw *readerWriter) Shutdown() {
	// todo: stop blocklist poll
	rw.pool.Shutdown()
	rw.r.Shutdown()
}

// EnableCompaction activates the compaction/retention loops
func (rw *readerWriter) EnableCompaction(ctx context.Context, cfg *CompactorConfig, c CompactorSharder, overrides CompactorOverrides) error {
	// If compactor configuration is not as expected, no need to go any further
	err := cfg.validate()
	if err != nil {
		return err
	}

	// Set default if needed. This is mainly for tests.
	if cfg.RetentionConcurrency == 0 {
		cfg.RetentionConcurrency = DefaultRetentionConcurrency
	}

	rw.compactorCfg = cfg
	rw.compactorSharder = c
	rw.compactorOverrides = overrides

	if rw.cfg.BlocklistPoll == 0 {
		level.Info(rw.logger).Log("msg", "polling cycle unset. compaction and retention disabled")
		return nil
	}

	if cfg != nil {
		level.Info(rw.logger).Log("msg", "compaction and retention enabled.")
		go rw.compactionLoop(ctx)
		go rw.retentionLoop(ctx)
	}

	return nil
}

// EnablePolling activates the polling loop. Pass nil if this component
//
//	should never be a tenant index builder.
func (rw *readerWriter) EnablePolling(ctx context.Context, sharder blocklist.JobSharder) {
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

	level.Info(rw.logger).Log("msg", "polling enabled", "interval", rw.cfg.BlocklistPoll, "blocklist_concurrency", rw.cfg.BlocklistPollConcurrency)

	blocklistPoller := blocklist.NewPoller(&blocklist.PollerConfig{
		PollConcurrency:           rw.cfg.BlocklistPollConcurrency,
		PollFallback:              rw.cfg.BlocklistPollFallback,
		TenantIndexBuilders:       rw.cfg.BlocklistPollTenantIndexBuilders,
		StaleTenantIndex:          rw.cfg.BlocklistPollStaleTenantIndex,
		PollJitterMs:              rw.cfg.BlocklistPollJitterMs,
		TolerateConsecutiveErrors: rw.cfg.BlocklistPollTolerateConsecutiveErrors,
	}, sharder, rw.r, rw.c, rw.w, rw.logger)

	rw.blocklistPoller = blocklistPoller

	// do the first poll cycle synchronously. this will allow the caller to know
	// that when this method returns the block list is updated
	rw.pollBlocklist()

	go rw.pollingLoop(ctx)
}

func (rw *readerWriter) pollingLoop(ctx context.Context) {
	ticker := time.NewTicker(rw.cfg.BlocklistPoll)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			rw.pollBlocklist()
		}
	}
}

func (rw *readerWriter) pollBlocklist() {
	blocklist, compactedBlocklist, err := rw.blocklistPoller.Do(rw.blocklist)
	if err != nil {
		level.Error(rw.logger).Log("msg", "failed to poll blocklist", "err", err)
		return
	}

	rw.blocklist.ApplyPollResults(blocklist, compactedBlocklist)
}

// includeBlock indicates whether a given block should be included in a backend search
func includeBlock(b *backend.BlockMeta, _ common.ID, blockStart []byte, blockEnd []byte, timeStart int64, timeEnd int64) bool {
	// todo: restore this functionality once it works. min/max ids are currently not recorded
	//    https://github.com/grafana/tempo/issues/1903
	//  correctly in a block
	// if bytes.Compare(id, b.MinID) == -1 || bytes.Compare(id, b.MaxID) == 1 {
	// 	return false
	// }

	if timeStart != 0 && timeEnd != 0 {
		if b.StartTime.Unix() >= timeEnd || b.EndTime.Unix() <= timeStart {
			return false
		}
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
func includeCompactedBlock(c *backend.CompactedBlockMeta, id common.ID, blockStart []byte, blockEnd []byte, poll time.Duration, timeStart int64, timeEnd int64) bool {
	lookback := time.Now().Add(-(2 * poll))
	if c.CompactedTime.Before(lookback) {
		return false
	}
	return includeBlock(&c.BlockMeta, id, blockStart, blockEnd, timeStart, timeEnd)
}

// createLegacyCache uses the config to return a cache and a list of roles.
func createLegacyCache(cfg *Config, logger gkLog.Logger) (cache.Cache, []cache.Role, error) {
	var legacyCache cache.Cache
	// if there's any cache configured, it always handles bloom filters and the trace id index
	roles := []cache.Role{cache.RoleBloom, cache.RoleTraceIDIdx}

	switch cfg.Cache {
	case "redis":
		legacyCache = redis.NewClient(cfg.Redis, cfg.BackgroundCache, "legacy", logger)
	case "memcached":
		legacyCache = memcached.NewClient(cfg.Memcached, cfg.BackgroundCache, "legacy", logger)
	}

	if legacyCache == nil {
		if cfg.Search != nil &&
			(cfg.Search.CacheControl.ColumnIndex ||
				cfg.Search.CacheControl.Footer ||
				cfg.Search.CacheControl.OffsetIndex) {
			return nil, nil, errors.New("no legacy cache configured, but cache_control is enabled. Please use the new top level cache configuration.")
		}

		return nil, nil, nil
	}

	// accumulate additional search roles
	if cfg.Search != nil {
		if cfg.Search.CacheControl.ColumnIndex {
			roles = append(roles, cache.RoleParquetColumnIdx)
		}
		if cfg.Search.CacheControl.Footer {
			roles = append(roles, cache.RoleParquetFooter)
		}
		if cfg.Search.CacheControl.OffsetIndex {
			roles = append(roles, cache.RoleParquetOffsetIdx)
		}
	}

	// log the roles
	rolesStr := make([]string, len(roles))
	for i, role := range roles {
		rolesStr[i] = string(role)
	}
	level.Warn(logger).Log("msg", "legacy cache configured with the following roles. Please migrate to the new top level cache configuration.", "roles", rolesStr)

	return legacyCache, roles, nil
}
