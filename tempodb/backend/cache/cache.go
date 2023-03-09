package cache

import (
	"bytes"
	"context"
	"io"
	"strconv"
	"strings"

	"github.com/grafana/tempo/pkg/cache"

	tempo_io "github.com/grafana/tempo/pkg/io"
	"github.com/grafana/tempo/tempodb/backend"
)

type readerWriter struct {
	nextReader backend.RawReader
	nextWriter backend.RawWriter
	cache      cache.Cache
}

func NewCache(nextReader backend.RawReader, nextWriter backend.RawWriter, cache cache.Cache) (backend.RawReader, backend.RawWriter, error) {
	rw := &readerWriter{
		cache:      cache,
		nextReader: nextReader,
		nextWriter: nextWriter,
	}

	return rw, rw, nil
}

// List implements backend.RawReader
func (r *readerWriter) List(ctx context.Context, keypath backend.KeyPath) ([]string, error) {
	return r.nextReader.List(ctx, keypath)
}

// Read implements backend.RawReader
func (r *readerWriter) Read(ctx context.Context, name string, keypath backend.KeyPath, shouldCache bool) (io.ReadCloser, int64, error) {
	var k string
	if shouldCache {
		k = key(keypath, name)
		found, vals, _ := r.cache.Fetch(ctx, []string{k})
		if len(found) > 0 {
			return io.NopCloser(bytes.NewReader(vals[0])), int64(len(vals[0])), nil
		}
	}

	object, size, err := r.nextReader.Read(ctx, name, keypath, false)
	if err != nil {
		return nil, 0, err
	}
	defer object.Close()

	b, err := tempo_io.ReadAllWithEstimate(object, size)
	if err == nil && shouldCache {
		r.cache.Store(ctx, []string{k}, [][]byte{b})
	}

	return io.NopCloser(bytes.NewReader(b)), size, err
}

// ReadRange implements backend.RawReader
func (r *readerWriter) ReadRange(ctx context.Context, name string, keypath backend.KeyPath, offset uint64, buffer []byte, shouldCache bool) error {
	var k string
	if shouldCache {
		// cache key is tenantID:blockID:offset:length - file name is not needed in key
		keyGen := keypath
		keyGen = append(keyGen, strconv.Itoa(int(offset)), strconv.Itoa(len(buffer)))
		k = strings.Join(keyGen, ":")
		found, vals, _ := r.cache.Fetch(ctx, []string{k})
		if len(found) > 0 {
			copy(buffer, vals[0])
		}
	}

	err := r.nextReader.ReadRange(ctx, name, keypath, offset, buffer, false)
	if err == nil && shouldCache {
		r.cache.Store(ctx, []string{k}, [][]byte{buffer})
	}

	return err
}

// Shutdown implements backend.RawReader
func (r *readerWriter) Shutdown() {
	r.nextReader.Shutdown()
	r.cache.Stop()
}

// Write implements backend.Writer
func (r *readerWriter) Write(ctx context.Context, name string, keypath backend.KeyPath, data io.Reader, size int64, shouldCache bool) error {
	b, err := tempo_io.ReadAllWithEstimate(data, size)
	if err != nil {
		return err
	}

	if shouldCache {
		r.cache.Store(ctx, []string{key(keypath, name)}, [][]byte{b})
	}
	return r.nextWriter.Write(ctx, name, keypath, bytes.NewReader(b), int64(len(b)), false)
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
