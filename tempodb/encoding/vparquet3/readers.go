package vparquet3

import (
	"context"
	"encoding/binary"
	"io"

	"go.uber.org/atomic"

	"github.com/grafana/tempo/pkg/cache"
	"github.com/grafana/tempo/tempodb/backend"
)

type cacheReaderAt interface {
	ReadAtWithCache([]byte, int64, cache.Role) (int, error)
}

// BackendReaderAt is used to track backend requests and present a io.ReaderAt interface backed
// by a backend.Reader
type BackendReaderAt struct {
	ctx  context.Context
	r    backend.Reader
	name string
	meta *backend.BlockMeta

	bytesRead atomic.Uint64
}

var _ cacheReaderAt = (*BackendReaderAt)(nil)

func NewBackendReaderAt(ctx context.Context, r backend.Reader, name string, meta *backend.BlockMeta) *BackendReaderAt {
	return &BackendReaderAt{ctx, r, name, meta, atomic.Uint64{}}
}

func (b *BackendReaderAt) ReadAt(p []byte, off int64) (int, error) {
	return b.ReadAtWithCache(p, off, cache.RoleNone)
}

func (b *BackendReaderAt) ReadAtWithCache(p []byte, off int64, role cache.Role) (int, error) {
	err := b.r.ReadRange(b.ctx, b.name, b.meta.BlockID, b.meta.TenantID, uint64(off), p, &backend.CacheInfo{
		Role: role,
		Meta: b.meta,
	})
	if err != nil {
		return 0, err
	}
	b.bytesRead.Add(uint64(len(p)))
	return len(p), nil
}

func (b *BackendReaderAt) BytesRead() uint64 {
	return b.bytesRead.Load()
}

type cachedObjectRecord struct {
	length int64
	role   cache.Role
}

// cachedReaderAt is used to route specific reads to the caching layer. this must be passed directly into
// the parquet.File so thet Set*Section() methods get called.
type cachedReaderAt struct {
	r             cacheReaderAt
	cachedObjects map[int64]cachedObjectRecord // storing offsets and length of objects we want to cache

	readerSize int64
	footerSize uint32

	maxPageSize int
}

var (
	_ cacheReaderAt = (*cachedReaderAt)(nil)
	_ io.ReaderAt   = (*cachedReaderAt)(nil)
)

func newCachedReaderAt(r cacheReaderAt, maxPageSize int, size int64, footerSize uint32) *cachedReaderAt {
	return &cachedReaderAt{r, map[int64]cachedObjectRecord{}, size, footerSize, maxPageSize}
}

// called by parquet-go in OpenFile() to set offset and length of footer section
func (r *cachedReaderAt) SetFooterSection(offset, length int64) {
	r.cachedObjects[offset] = cachedObjectRecord{length, cache.RoleParquetFooter}
}

// called by parquet-go in OpenFile() to set offset and length of column indexes
func (r *cachedReaderAt) SetColumnIndexSection(offset, length int64) {
	r.cachedObjects[offset] = cachedObjectRecord{length, cache.RoleParquetColumnIdx}
}

// called by parquet-go in OpenFile() to set offset and length of offset index section
func (r *cachedReaderAt) SetOffsetIndexSection(offset, length int64) {
	r.cachedObjects[offset] = cachedObjectRecord{length, cache.RoleParquetOffsetIdx}
}

func (r *cachedReaderAt) ReadAt(p []byte, off int64) (int, error) {
	if len(p) == 4 && off == 0 {
		// Magic header
		return copy(p, []byte("PAR1")), nil
	}

	if len(p) == 8 && off == r.readerSize-8 && r.footerSize > 0 /* not present in previous block metas */ {
		// Magic footer
		binary.LittleEndian.PutUint32(p, r.footerSize)
		copy(p[4:8], []byte("PAR1"))
		return 8, nil
	}

	// check if the offset and length is stored as a special object
	rec, ok := r.cachedObjects[off]
	if ok && rec.length == int64(len(p)) {
		return r.r.ReadAtWithCache(p, off, rec.role)
	}

	if len(p) <= r.maxPageSize {
		return r.r.ReadAtWithCache(p, off, cache.RoleParquetPage)
	}

	return r.r.ReadAtWithCache(p, off, cache.RoleNone)
}

func (r *cachedReaderAt) ReadAtWithCache(p []byte, off int64, role cache.Role) (int, error) {
	return r.r.ReadAtWithCache(p, off, role)
}

// walReaderAt is wrapper over io.ReaderAt, and is used to measure the total bytes read when searching walBlock.
type walReaderAt struct {
	ctx       context.Context
	r         io.ReaderAt
	bytesRead atomic.Uint64
}

var _ io.ReaderAt = (*walReaderAt)(nil)

func newWalReaderAt(ctx context.Context, r io.ReaderAt) *walReaderAt {
	return &walReaderAt{ctx, r, atomic.Uint64{}}
}

func (wr *walReaderAt) ReadAt(p []byte, off int64) (int, error) {
	if err := wr.ctx.Err(); err != nil {
		return 0, err
	}

	// parquet-go will call ReadAt when reading data from a parquet file.
	n, err := wr.r.ReadAt(p, off)
	// ReadAt can read less than len(p) bytes in some cases
	wr.bytesRead.Add(uint64(n))

	if err != nil {
		return 0, err
	}

	return n, nil
}

func (wr *walReaderAt) BytesRead() uint64 {
	return wr.bytesRead.Load()
}
