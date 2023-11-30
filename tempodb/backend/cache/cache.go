package cache

import (
	"bytes"
	"context"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/google/uuid"
	"github.com/grafana/tempo/pkg/cache"

	tempo_io "github.com/grafana/tempo/pkg/io"
	"github.com/grafana/tempo/tempodb/backend"
)

type BloomConfig struct {
	CacheMinCompactionLevel uint8         `yaml:"cache_min_compaction_level"`
	CacheMaxBlockAge        time.Duration `yaml:"cache_max_block_age"`
}

type readerWriter struct {
	cfgBloom *BloomConfig

	nextReader backend.RawReader
	nextWriter backend.RawWriter

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
	}

	level.Info(logger).Log("msg", "caches available to storage backend",
		"footer", rw.footerCache != nil,
		"bloom", rw.bloomCache != nil,
		"offset_idx", rw.offsetIdxCache != nil,
		"column_idx", rw.columnIdxCache != nil,
		"trace_id_idx", rw.traceIDIdxCache != nil,
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

// Read implements backend.RawReader
func (r *readerWriter) Read(ctx context.Context, name string, keypath backend.KeyPath, cacheInfo *backend.CacheInfo) (io.ReadCloser, int64, error) {
	var k string
	cache := r.cacheFor(cacheInfo)
	if cache != nil {
		k = key(keypath, name)
		found, vals, _ := cache.Fetch(ctx, []string{k})
		if len(found) > 0 {
			return io.NopCloser(bytes.NewReader(vals[0])), int64(len(vals[0])), nil
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
		cache.Store(ctx, []string{k}, [][]byte{b})
	}

	return io.NopCloser(bytes.NewReader(b)), size, err
}

// ReadRange implements backend.RawReader
func (r *readerWriter) ReadRange(ctx context.Context, name string, keypath backend.KeyPath, offset uint64, buffer []byte, cacheInfo *backend.CacheInfo) error {
	var k string
	cache := r.cacheFor(cacheInfo)
	if cache != nil {
		// cache key is tenantID:blockID:offset:length - file name is not needed in key
		keyGen := keypath
		keyGen = append(keyGen, strconv.Itoa(int(offset)), strconv.Itoa(len(buffer)))
		k = strings.Join(keyGen, ":")
		found, vals, _ := cache.Fetch(ctx, []string{k})
		if len(found) > 0 {
			copy(buffer, vals[0])
		}
	}

	// previous implemenation always passed false forward for "shouldCache" so we are matching that behavior by passing nil for cacheInfo
	// todo: reevaluate. should we pass the cacheInfo forward?
	err := r.nextReader.ReadRange(ctx, name, keypath, offset, buffer, nil)
	if err == nil && cache != nil {
		cache.Store(ctx, []string{k}, [][]byte{buffer})
	}

	return err
}

// Shutdown implements backend.RawReader
func (r *readerWriter) Shutdown() {
	r.nextReader.Shutdown()

	stopCache := func(c cache.Cache) {
		if c != nil {
			c.Stop()
		}
	}

	stopCache(r.footerCache)
	stopCache(r.bloomCache)
	stopCache(r.offsetIdxCache)
	stopCache(r.columnIdxCache)
	stopCache(r.traceIDIdxCache)
}

// Write implements backend.Writer
func (r *readerWriter) Write(ctx context.Context, name string, keypath backend.KeyPath, data io.Reader, size int64, cacheInfo *backend.CacheInfo) error {
	b, err := tempo_io.ReadAllWithEstimate(data, size)
	if err != nil {
		return err
	}

	if cache := r.cacheFor(cacheInfo); cache != nil {
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
	if cacheInfo != nil {
		panic("delete is not supported for cache.Cache backend")
	}
	return r.nextWriter.Delete(ctx, name, keypath, nil)
}

func key(keypath backend.KeyPath, name string) string {
	return strings.Join(keypath, ":") + ":" + name
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
		if r.cfgBloom.CacheMinCompactionLevel > 0 && cacheInfo.Meta.CompactionLevel > r.cfgBloom.CacheMinCompactionLevel {
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
