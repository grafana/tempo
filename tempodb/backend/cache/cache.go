package cache

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/grafana/tempo/pkg/cache"

	tempo_io "github.com/grafana/tempo/pkg/io"
	tempo_log "github.com/grafana/tempo/pkg/util/log"
	"github.com/grafana/tempo/tempodb/backend"
)

// cacheStoreSizeBytes records the byte size of every item written to a tempodb backend cache, labelled by role.
var cacheStoreSizeBytes = promauto.NewHistogramVec(prometheus.HistogramOpts{
	Namespace: "tempodb",
	Name:      "cache_store_size_bytes",
	Help:      "Distribution of item sizes written to tempodb backend caches, by role.",
	Buckets:   prometheus.ExponentialBuckets(512, 2, 15), // 512 B, 1 KiB, ..., 8 MiB
}, []string{"role"})

// pageCacheErrorBytes counts cache write-path bytes that did not make it
// into the cache because the backend read failed with a non-nil non-EOF
// error. Hit / miss / store totals are already observable from memcached's
// own metrics, so we only emit the loss signal here.
var pageCacheErrorBytes = promauto.NewCounterVec(prometheus.CounterOpts{
	Namespace: "tempodb",
	Name:      "cache_store_error_bytes_total",
	Help:      "Parquet cache bytes lost to backend read errors, by error class and cache role.",
}, []string{"class", "role"})

func recordCacheError(ci *backend.CacheInfo, err error, name string, offset uint64, n int, logger *tempo_log.RateLimitedLogger) {
	if ci == nil {
		return
	}
	class := errorClass(err)
	pageCacheErrorBytes.WithLabelValues(class, string(ci.Role)).Add(float64(n))
	if logger != nil {
		logger.Log(
			"msg", "cache write-path read error",
			"class", class,
			"role", string(ci.Role),
			"err", err,
			"name", name,
			"offset", offset,
			"len", n,
		)
	}
}

// errorClass maps a backend read error to a stable, low-cardinality label.
// io.EOF is intentionally absent because it is not an error in this context
// (io.ReaderAt returns valid data alongside io.EOF at end-of-object).
func errorClass(err error) string {
	switch {
	case errors.Is(err, io.ErrUnexpectedEOF):
		return "unexpected_eof"
	case errors.Is(err, context.Canceled):
		return "canceled"
	case errors.Is(err, context.DeadlineExceeded):
		return "deadline_exceeded"
	default:
		return "other"
	}
}

// errorLogsPerSecond caps cache write-path error log volume across the pod.
const errorLogsPerSecond = 10

type BloomConfig struct {
	CacheMinCompactionLevel uint8         `yaml:"cache_min_compaction_level"`
	CacheMaxBlockAge        time.Duration `yaml:"cache_max_block_age"`
}

type readerWriter struct {
	cfgBloom *BloomConfig

	nextReader backend.RawReader
	nextWriter backend.RawWriter

	errLogger *tempo_log.RateLimitedLogger

	footerCache     cache.Cache
	bloomCache      cache.Cache
	columnIdxCache  cache.Cache
	offsetIdxCache  cache.Cache
	traceIDIdxCache cache.Cache
	pageCache       cache.Cache
}

func NewCache(cfgBloom *BloomConfig, nextReader backend.RawReader, nextWriter backend.RawWriter, cacheProvider cache.Provider, logger log.Logger) (backend.RawReader, backend.RawWriter, error) {
	rw := &readerWriter{
		cfgBloom: cfgBloom,

		footerCache:     cacheProvider.CacheFor(cache.RoleParquetFooter),
		bloomCache:      cacheProvider.CacheFor(cache.RoleBloom),
		offsetIdxCache:  cacheProvider.CacheFor(cache.RoleParquetOffsetIdx),
		columnIdxCache:  cacheProvider.CacheFor(cache.RoleParquetColumnIdx),
		traceIDIdxCache: cacheProvider.CacheFor(cache.RoleTraceIDIdx),
		pageCache:       cacheProvider.CacheFor(cache.RoleParquetPage),

		nextReader: nextReader,
		nextWriter: nextWriter,

		errLogger: tempo_log.NewRateLimitedLogger(errorLogsPerSecond, level.Warn(logger)),
	}

	level.Info(logger).Log("msg", "caches available to storage backend",
		cache.RoleParquetFooter, rw.footerCache != nil,
		cache.RoleBloom, rw.bloomCache != nil,
		cache.RoleParquetOffsetIdx, rw.offsetIdxCache != nil,
		cache.RoleParquetColumnIdx, rw.columnIdxCache != nil,
		cache.RoleTraceIDIdx, rw.traceIDIdxCache != nil,
		cache.RoleParquetPage, rw.pageCache != nil,
	)

	return rw, rw, nil
}

// List implements backend.RawReader
func (r *readerWriter) List(ctx context.Context, keypath backend.KeyPath) ([]string, error) {
	return r.nextReader.List(ctx, keypath)
}

func (r *readerWriter) ListBlocks(ctx context.Context, tenant string) (blockIDs []uuid.UUID, compactedBlockIDs []uuid.UUID, err error) {
	return r.nextReader.ListBlocks(ctx, tenant)
}

// Find implements backend.Reader
func (r *readerWriter) Find(ctx context.Context, keypath backend.KeyPath, f backend.FindFunc) (err error) {
	return r.nextReader.Find(ctx, keypath, f)
}

// Read implements backend.RawReader
func (r *readerWriter) Read(ctx context.Context, name string, keypath backend.KeyPath, cacheInfo *backend.CacheInfo) (io.ReadCloser, int64, error) {
	var k string
	cache := r.cacheFor(cacheInfo)
	if cache != nil {
		k = key(keypath, name)
		b, found := cache.FetchKey(ctx, k)
		if found {
			return io.NopCloser(bytes.NewReader(b)), int64(len(b)), nil
		}
	}

	// previous implemenation always passed false forward for "shouldCache" so we are matching that behavior by passing nil for cacheInfo
	// todo: reevaluate. should we pass the cacheInfo forward?
	object, size, err := r.nextReader.Read(ctx, name, keypath, nil)
	if err != nil {
		return nil, 0, err
	}
	defer object.Close()

	b, err := tempo_io.ReadAllWithEstimate(object, size)
	if err == nil && cache != nil {
		store(ctx, cache, cacheInfo.Role, k, b)
	}

	return io.NopCloser(bytes.NewReader(b)), size, err
}

// ReadRange implements backend.RawReader
func (r *readerWriter) ReadRange(ctx context.Context, name string, keypath backend.KeyPath, offset uint64, buffer []byte, cacheInfo *backend.CacheInfo) error {
	var k string
	cache := r.cacheFor(cacheInfo)
	if cache != nil {
		k = strings.Join(append(keypath, name, strconv.Itoa(int(offset)), strconv.Itoa(len(buffer))), ":")
		b, found := cache.FetchKey(ctx, k)
		if found {
			copy(buffer, b)
			cache.Release(b)
			return nil
		}
	}

	// previous implemenation always passed false forward for "shouldCache" so we are matching that behavior by passing nil for cacheInfo
	// todo: reevaluate. should we pass the cacheInfo forward?
	err := r.nextReader.ReadRange(ctx, name, keypath, offset, buffer, nil)

	// io.EOF alongside valid data is a successful read per io.ReaderAt;
	// store the bytes instead of dropping them, but still surface io.EOF
	// to the caller so downstream code that distinguishes EOF keeps working.
	if err != nil && !errors.Is(err, io.EOF) {
		if cache != nil {
			recordCacheError(cacheInfo, err, name, offset, len(buffer), r.errLogger)
		}
		return err
	}
	if cache != nil {
		store(ctx, cache, cacheInfo.Role, k, buffer)
	}

	return err
}

// Shutdown implements backend.RawReader
func (r *readerWriter) Shutdown() {
	r.nextReader.Shutdown()
}

// Write implements backend.Writer
func (r *readerWriter) Write(ctx context.Context, name string, keypath backend.KeyPath, data io.Reader, size int64, cacheInfo *backend.CacheInfo) error {
	b, err := tempo_io.ReadAllWithEstimate(data, size)
	if err != nil {
		return err
	}

	if cache := r.cacheFor(cacheInfo); cache != nil {
		cacheStoreSizeBytes.WithLabelValues(string(cacheInfo.Role)).Observe(float64(len(b)))
		cache.Store(ctx, []string{key(keypath, name)}, [][]byte{b})
	}

	// previous implemenation always passed false forward for "shouldCache" so we are matching that behavior by passing nil for cacheInfo
	// todo: reevaluate. should we pass the cacheInfo forward?
	return r.nextWriter.Write(ctx, name, keypath, bytes.NewReader(b), int64(len(b)), nil)
}

// Append implements backend.Writer
func (r *readerWriter) Append(ctx context.Context, name string, keypath backend.KeyPath, tracker backend.AppendTracker, buffer []byte) (backend.AppendTracker, error) {
	return r.nextWriter.Append(ctx, name, keypath, tracker, buffer)
}

// CloseAppend implements backend.Writer
func (r *readerWriter) CloseAppend(ctx context.Context, tracker backend.AppendTracker) error {
	return r.nextWriter.CloseAppend(ctx, tracker)
}

func (r *readerWriter) Delete(ctx context.Context, name string, keypath backend.KeyPath, cacheInfo *backend.CacheInfo) error {
	if c := r.cacheFor(cacheInfo); c != nil {
		c.Remove(ctx, []string{key(keypath, name)})
	}
	return r.nextWriter.Delete(ctx, name, keypath, nil)
}

func key(keypath backend.KeyPath, name string) string {
	return strings.Join(keypath, ":") + ":" + name
}

// BlockKeyPrefix returns the cache key prefix shared by bloom-filter shards and the
// trace-id-index for a block. Whole-object cache keys are BlockKeyPrefix + name.
// Note: ReadRange keys also embed ":offset:length" suffixes and cannot be derived from this prefix.
func BlockKeyPrefix(blockID uuid.UUID, tenantID string) string {
	return key(backend.KeyPathForBlock(blockID, tenantID), "")
}

// cacheFor evaluates the cacheInfo and returns the appropriate cache.
func (r *readerWriter) cacheFor(cacheInfo *backend.CacheInfo) cache.Cache {
	if cacheInfo == nil {
		return nil
	}

	switch cacheInfo.Role {
	case cache.RoleParquetFooter:
		return r.footerCache
	case cache.RoleParquetColumnIdx:
		return r.columnIdxCache
	case cache.RoleParquetOffsetIdx:
		return r.offsetIdxCache
	case cache.RoleParquetPage:
		return r.pageCache
	case cache.RoleTraceIDIdx:
		return r.traceIDIdxCache
	case cache.RoleBloom:
		// if there is no bloom cfg then there are no restrictions on bloom filter caching
		if r.cfgBloom == nil {
			return r.bloomCache
		}

		if cacheInfo.Meta == nil {
			return nil
		}

		// compaction level is _atleast_ CacheMinCompactionLevel
		if r.cfgBloom.CacheMinCompactionLevel > 0 && cacheInfo.Meta.CompactionLevel > uint32(r.cfgBloom.CacheMinCompactionLevel) {
			return nil
		}

		curTime := time.Now()
		// block is not older than CacheMaxBlockAge
		if r.cfgBloom.CacheMaxBlockAge > 0 && curTime.Sub(cacheInfo.Meta.StartTime) > r.cfgBloom.CacheMaxBlockAge {
			return nil
		}

		return r.bloomCache
	}

	return nil
}

func store(ctx context.Context, cache cache.Cache, role cache.Role, key string, val []byte) {
	cacheStoreSizeBytes.WithLabelValues(string(role)).Observe(float64(len(val)))

	write := val
	if needsCopy(role) {
		write = make([]byte, len(val))
		copy(write, val)
	}

	cache.Store(ctx, []string{key}, [][]byte{write})
}

// needsCopy returns true if the role should be copied into a new buffer before being written to the cache
// todo: should this be signalled through cacheinfo instead?
func needsCopy(role cache.Role) bool {
	return role == cache.RoleParquetPage // parquet pages are reused by the library. if we don't copy them then the buffer may be reused before written to cache
}
