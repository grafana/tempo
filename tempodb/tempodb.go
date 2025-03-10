package tempodb

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/grafana/tempo/pkg/collector"
	"go.opentelemetry.io/otel/attribute"

	gkLog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/grafana/dskit/user"
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
		Namespace:                       "tempodb",
		Name:                            "retention_duration_seconds",
		Help:                            "Records the amount of time to perform retention tasks.",
		Buckets:                         prometheus.ExponentialBuckets(.25, 2, 6),
		NativeHistogramBucketFactor:     1.1,
		NativeHistogramMaxBucketNumber:  100,
		NativeHistogramMinResetDuration: 1 * time.Hour,
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
	Find(ctx context.Context, tenantID string, id common.ID, blockStart string, blockEnd string, timeStart int64, timeEnd int64, opts common.SearchOptions) ([]*tempopb.TraceByIDResponse, []error, error)
	Search(ctx context.Context, meta *backend.BlockMeta, req *tempopb.SearchRequest, opts common.SearchOptions) (*tempopb.SearchResponse, error)
	SearchTags(ctx context.Context, meta *backend.BlockMeta, req *tempopb.SearchTagsBlockRequest, opts common.SearchOptions) (*tempopb.SearchTagsV2Response, error)
	SearchTagValues(ctx context.Context, meta *backend.BlockMeta, req *tempopb.SearchTagValuesBlockRequest, opts common.SearchOptions) (*tempopb.SearchTagValuesResponse, error)
	SearchTagValuesV2(ctx context.Context, meta *backend.BlockMeta, req *tempopb.SearchTagValuesRequest, opts common.SearchOptions) (*tempopb.SearchTagValuesV2Response, error)

	// TODO(suraj): use common.MetricsCallback in Fetch and remove the Bytes callback from traceql.FetchSpansResponse
	Fetch(ctx context.Context, meta *backend.BlockMeta, req traceql.FetchSpansRequest, opts common.SearchOptions) (traceql.FetchSpansResponse, error)
	FetchTagValues(ctx context.Context, meta *backend.BlockMeta, req traceql.FetchTagValuesRequest, cb traceql.FetchTagValuesCallback, mcb common.MetricsCallback, opts common.SearchOptions) error
	FetchTagNames(ctx context.Context, meta *backend.BlockMeta, req traceql.FetchTagsRequest, cb traceql.FetchTagsCallback, mcb common.MetricsCallback, opts common.SearchOptions) error

	BlockMetas(tenantID string) []*backend.BlockMeta
	EnablePolling(ctx context.Context, sharder blocklist.JobSharder)
	Tenants() []string

	Shutdown()
}

type Compactor interface {
	EnableCompaction(ctx context.Context, cfg *CompactorConfig, sharder CompactorSharder, overrides CompactorOverrides) error
	CompactWithConfig(ctx context.Context, metas []*backend.BlockMeta, tenant string, cfg *CompactorConfig, sharder CompactorSharder, overrides CompactorOverrides) error
}

type CompactorSharder interface {
	Combine(dataEncoding string, tenantID string, objs ...[]byte) ([]byte, bool, error)
	Owns(hash string) bool
	RecordDiscardedSpans(count int, tenantID string, traceID string, rootSpanName string, rootServiceName string)
}

type CompactorOverrides interface {
	BlockRetentionForTenant(tenantID string) time.Duration
	CompactionDisabledForTenant(tenantID string) bool
	MaxBytesPerTraceForTenant(tenantID string) int
	MaxCompactionRangeForTenant(tenantID string) time.Duration
}

type WriteableBlock interface {
	BlockMeta() *backend.BlockMeta
	Write(ctx context.Context, w backend.Writer) error
}

var _ Reader = (*readerWriter)(nil)

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

// CompleteBlockWithBackend iterates the given WAL block but flushes it to the given backend instead of the default TempoDB backend. The
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

func (rw *readerWriter) Tenants() []string {
	return rw.blocklist.Tenants()
}

func (rw *readerWriter) Find(ctx context.Context, tenantID string, id common.ID, blockStart string, blockEnd string, timeStart int64, timeEnd int64, opts common.SearchOptions) ([]*tempopb.TraceByIDResponse, []error, error) {
	// tracing instrumentation
	logger := log.WithContext(ctx, log.Logger)
	ctx, span := tracer.Start(ctx, "store.Find")
	defer span.End()

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
		if includeBlock(b, id, blockStartBytes, blockEndBytes, timeStart, timeEnd, opts.BlockReplicationFactor) {
			copiedBlocklist = append(copiedBlocklist, b)
			blocksSearched++
		}
	}
	for _, c := range compactedBlocklist {
		if includeCompactedBlock(c, id, blockStartBytes, blockEndBytes, rw.cfg.BlocklistPoll, timeStart, timeEnd, opts.BlockReplicationFactor) {
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

		level.Debug(logger).Log("msg", "searching for trace in block", "findTraceID", hex.EncodeToString(id), "block", meta.BlockID, "found", foundObject != nil)
		return foundObject, nil
	})

	partialTraceObjs := make([]*tempopb.TraceByIDResponse, 0)
	for i := range partialTraces {
		if trace, ok := partialTraces[i].(*tempopb.TraceByIDResponse); ok {
			if trace == nil {
				continue
			}
			partialTraceObjs = append(partialTraceObjs, trace)
		}
	}

	span.SetAttributes(attribute.Int("blockErrs", len(funcErrs)))
	span.SetAttributes(attribute.Int("liveBlocks", len(blocklist)))
	span.SetAttributes(attribute.Int("liveBlocksSearched", blocksSearched))
	span.SetAttributes(attribute.Int("compactedBlocks", len(compactedBlocklist)))
	span.SetAttributes(attribute.Int("compactedBlocksSearched", compactedBlocksSearched))

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

func (rw *readerWriter) SearchTags(ctx context.Context, meta *backend.BlockMeta, req *tempopb.SearchTagsBlockRequest, opts common.SearchOptions) (*tempopb.SearchTagsV2Response, error) {
	scope := req.SearchReq.Scope
	attributeScope := traceql.AttributeScopeFromString(scope)

	if attributeScope == traceql.AttributeScopeUnknown {
		return nil, fmt.Errorf("unknown scope: %s", scope)
	}

	block, err := encoding.OpenBlock(meta, rw.r)
	if err != nil {
		return nil, err
	}

	distinctValues := collector.NewScopedDistinctString(0, req.SearchReq.MaxTagsPerScope, req.SearchReq.StaleValuesThreshold)
	mc := collector.NewMetricsCollector()

	rw.cfg.Search.ApplyToOptions(&opts)
	err = block.SearchTags(ctx, attributeScope, func(s string, scope traceql.AttributeScope) {
		distinctValues.Collect(scope.String(), s)
	}, mc.Add, opts)
	if err != nil {
		return nil, err
	}

	orgID, _ := user.ExtractOrgID(ctx)
	if distinctValues.Exceeded() {
		level.Warn(log.Logger).Log("msg", "Search tags exceeded limit, reduce cardinality or size of tags", "orgID", orgID, "stopReason", distinctValues.StopReason())
	}

	// build response
	collected := distinctValues.Strings()
	resp := &tempopb.SearchTagsV2Response{
		Scopes:  make([]*tempopb.SearchTagsV2Scope, 0, len(collected)),
		Metrics: &tempopb.MetadataMetrics{InspectedBytes: mc.TotalValue()},
	}
	for scope, vals := range collected {
		resp.Scopes = append(resp.Scopes, &tempopb.SearchTagsV2Scope{
			Name: scope,
			Tags: vals,
		})
	}

	return resp, nil
}

func (rw *readerWriter) SearchTagValues(ctx context.Context, meta *backend.BlockMeta, req *tempopb.SearchTagValuesBlockRequest, opts common.SearchOptions) (response *tempopb.SearchTagValuesResponse, err error) {
	block, err := encoding.OpenBlock(meta, rw.r)
	if err != nil {
		return &tempopb.SearchTagValuesResponse{}, err
	}

	dv := collector.NewDistinctString(0, req.SearchReq.MaxTagValues, req.SearchReq.StaleValueThreshold)
	mc := collector.NewMetricsCollector()
	rw.cfg.Search.ApplyToOptions(&opts)
	err = block.SearchTagValues(ctx, req.SearchReq.TagName, dv.Collect, mc.Add, opts)

	orgID, _ := user.ExtractOrgID(ctx)
	if dv.Exceeded() {
		level.Warn(log.Logger).Log("msg", "Search tags exceeded limit, reduce cardinality or size of tags", "orgID", orgID, "stopReason", dv.StopReason())
	}

	return &tempopb.SearchTagValuesResponse{
		TagValues: dv.Strings(),
		Metrics:   &tempopb.MetadataMetrics{InspectedBytes: mc.TotalValue()},
	}, err
}

func (rw *readerWriter) SearchTagValuesV2(ctx context.Context, meta *backend.BlockMeta, req *tempopb.SearchTagValuesRequest, opts common.SearchOptions) (*tempopb.SearchTagValuesV2Response, error) {
	block, err := encoding.OpenBlock(meta, rw.r)
	if err != nil {
		return nil, err
	}

	tag, err := traceql.ParseIdentifier(req.TagName)
	if err != nil {
		return nil, err
	}

	dv := collector.NewDistinctValue(0, req.MaxTagValues, req.StaleValueThreshold, func(v tempopb.TagValue) int { return len(v.Type) + len(v.Value) })
	mc := collector.NewMetricsCollector()
	rw.cfg.Search.ApplyToOptions(&opts)
	err = block.SearchTagValuesV2(ctx, tag, traceql.MakeCollectTagValueFunc(dv.Collect), mc.Add, opts)
	if err != nil {
		return nil, err
	}

	orgID, _ := user.ExtractOrgID(ctx)
	if dv.Exceeded() {
		level.Warn(log.Logger).Log("msg", "Search tags exceeded limit, reduce cardinality or size of tags", "orgID", orgID, "stopReason", dv.StopReason())
	}

	resp := &tempopb.SearchTagValuesV2Response{
		Metrics: &tempopb.MetadataMetrics{InspectedBytes: mc.TotalValue()},
	}
	for _, v := range dv.Values() {
		v2 := v
		resp.TagValues = append(resp.TagValues, &v2)
	}

	return resp, nil
}

// Fetch only uses rw.r which has caching enabled
func (rw *readerWriter) Fetch(ctx context.Context, meta *backend.BlockMeta, req traceql.FetchSpansRequest, opts common.SearchOptions) (traceql.FetchSpansResponse, error) {
	block, err := encoding.OpenBlock(meta, rw.r)
	if err != nil {
		return traceql.FetchSpansResponse{}, err
	}

	rw.cfg.Search.ApplyToOptions(&opts)
	return block.Fetch(ctx, req, opts)
}

func (rw *readerWriter) FetchTagValues(ctx context.Context, meta *backend.BlockMeta, req traceql.FetchTagValuesRequest, cb traceql.FetchTagValuesCallback, mcb common.MetricsCallback, opts common.SearchOptions) error {
	block, err := encoding.OpenBlock(meta, rw.r)
	if err != nil {
		return err
	}

	rw.cfg.Search.ApplyToOptions(&opts)
	return block.FetchTagValues(ctx, req, cb, mcb, opts)
}

func (rw *readerWriter) FetchTagNames(ctx context.Context, meta *backend.BlockMeta, req traceql.FetchTagsRequest, cb traceql.FetchTagsCallback, mcb common.MetricsCallback, opts common.SearchOptions) error {
	block, err := encoding.OpenBlock(meta, rw.r)
	if err != nil {
		return err
	}

	rw.cfg.Search.ApplyToOptions(&opts)
	return block.FetchTagNames(ctx, req, cb, mcb, opts)
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

	if rw.cfg.BlocklistPollTenantConcurrency == 0 {
		rw.cfg.BlocklistPollTenantConcurrency = DefaultBlocklistPollTenantConcurrency
	}

	if rw.cfg.BlocklistPollTenantIndexBuilders <= 0 {
		rw.cfg.BlocklistPollTenantIndexBuilders = DefaultTenantIndexBuilders
	}

	if rw.cfg.EmptyTenantDeletionAge <= 0 {
		rw.cfg.EmptyTenantDeletionAge = DefaultEmptyTenantDeletionAge
	}

	level.Info(rw.logger).Log("msg", "polling enabled", "interval", rw.cfg.BlocklistPoll, "blocklist_concurrency", rw.cfg.BlocklistPollConcurrency)

	blocklistPoller := blocklist.NewPoller(&blocklist.PollerConfig{
		PollConcurrency:            rw.cfg.BlocklistPollConcurrency,
		PollFallback:               rw.cfg.BlocklistPollFallback,
		TenantIndexBuilders:        rw.cfg.BlocklistPollTenantIndexBuilders,
		StaleTenantIndex:           rw.cfg.BlocklistPollStaleTenantIndex,
		PollJitterMs:               rw.cfg.BlocklistPollJitterMs,
		TolerateConsecutiveErrors:  rw.cfg.BlocklistPollTolerateConsecutiveErrors,
		TolerateTenantFailures:     rw.cfg.BlocklistPollTolerateTenantFailures,
		TenantPollConcurrency:      rw.cfg.BlocklistPollTenantConcurrency,
		EmptyTenantDeletionAge:     rw.cfg.EmptyTenantDeletionAge,
		EmptyTenantDeletionEnabled: rw.cfg.EmptyTenantDeletionEnabled,
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
func includeBlock(b *backend.BlockMeta, _ common.ID, blockStart, blockEnd []byte, timeStart, timeEnd int64, replicationFactor int) bool {
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

	blockIDBytes, _ := b.BlockID.Marshal()
	// check block is in shard boundaries
	// blockStartBytes <= blockIDBytes <= blockEndBytes
	if bytes.Compare(blockIDBytes, blockStart) == -1 || bytes.Compare(blockIDBytes, blockEnd) == 1 {
		return false
	}

	return b.ReplicationFactor == uint32(replicationFactor)
}

// if block is compacted within lookback period, and is within shard ranges, include it in search
func includeCompactedBlock(c *backend.CompactedBlockMeta, id common.ID, blockStart, blockEnd []byte, poll time.Duration, timeStart, timeEnd int64, replicationFactor int) bool {
	lookback := time.Now().Add(-(2 * poll))
	if c.CompactedTime.Before(lookback) {
		return false
	}
	return includeBlock(&c.BlockMeta, id, blockStart, blockEnd, timeStart, timeEnd, replicationFactor)
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
