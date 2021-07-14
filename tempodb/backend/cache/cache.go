package cache

import (
	"context"
	"io"
	"strings"

	cortex_cache "github.com/cortexproject/cortex/pkg/chunk/cache"
	"github.com/grafana/tempo/tempodb/backend"
)

type readerWriter struct {
	nextReader backend.RawReader
	nextWriter backend.RawWriter
	cache      cortex_cache.Cache
}

func NewCache(nextReader backend.RawReader, nextWriter backend.RawWriter, cache cortex_cache.Cache) (backend.RawReader, backend.RawWriter, error) {
	rw := &readerWriter{
		cache:      cache,
		nextReader: nextReader,
		nextWriter: nextWriter,
	}

	return rw, rw, nil
}

// List implements backend.Reader
func (r *readerWriter) List(ctx context.Context, keypath backend.KeyPath) ([]string, error) {
	return r.nextReader.List(ctx, keypath)
}

// Read implements backend.Reader
func (r *readerWriter) Read(ctx context.Context, name string, keypath backend.KeyPath) ([]byte, error) {
	var k string
	if shouldCache(name) {
		k = key(keypath, name)
		found, vals, _ := r.cache.Fetch(ctx, []string{k})
		if len(found) > 0 {
			return vals[0], nil
		}
	}

	val, err := r.nextReader.Read(ctx, name, keypath)
	if err == nil && shouldCache(name) {
		r.cache.Store(ctx, []string{k}, [][]byte{val})
	}

	return val, err
}

func (r *readerWriter) StreamReader(ctx context.Context, name string, keypath backend.KeyPath) (io.ReadCloser, int64, error) {
	panic("StreamReader is not yet supported for cache")
}

// ReadRange implements backend.Reader
func (r *readerWriter) ReadRange(ctx context.Context, name string, keypath backend.KeyPath, offset uint64, buffer []byte) error {
	return r.nextReader.ReadRange(ctx, name, keypath, offset, buffer)
}

// Shutdown implements backend.Reader
func (r *readerWriter) Shutdown() {
	r.nextReader.Shutdown()
	r.cache.Stop()
}

// Write implements backend.Writer
func (r *readerWriter) Write(ctx context.Context, name string, keypath backend.KeyPath, buffer []byte) error {
	if shouldCache(name) {
		r.cache.Store(ctx, []string{key(keypath, name)}, [][]byte{buffer})
	}

	return r.nextWriter.Write(ctx, name, keypath, buffer)
}

// Write implements backend.Writer
func (r *readerWriter) StreamWriter(ctx context.Context, name string, keypath backend.KeyPath, data io.Reader, size int64) error {
	return r.nextWriter.StreamWriter(ctx, name, keypath, data, size)
}

// Append implements backend.Writer
func (r *readerWriter) Append(ctx context.Context, name string, keypath backend.KeyPath, tracker backend.AppendTracker, buffer []byte) (backend.AppendTracker, error) {
	return r.nextWriter.Append(ctx, name, keypath, tracker, buffer)
}

// CloseAppend implements backend.Writer
func (r *readerWriter) CloseAppend(ctx context.Context, tracker backend.AppendTracker) error {
	return r.nextWriter.CloseAppend(ctx, tracker)
}

func key(keypath backend.KeyPath, name string) string {
	return strings.Join(keypath, ":") + ":" + name
}

func shouldCache(name string) bool {
	return name != backend.MetaName && name != backend.CompactedMetaName && name != backend.TenantIndexName
}
