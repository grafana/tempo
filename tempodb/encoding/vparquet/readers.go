package vparquet

import (
	"context"
	"encoding/binary"
	"io"
	"sync/atomic"

	"github.com/google/uuid"

	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

type BackendReaderAt struct {
	ctx      context.Context
	r        backend.Reader
	name     string
	blockID  uuid.UUID
	tenantID string

	TotalBytesRead atomic.Uint64
}

var _ io.ReaderAt = (*BackendReaderAt)(nil)

func NewBackendReaderAt(ctx context.Context, r backend.Reader, name string, blockID uuid.UUID, tenantID string) *BackendReaderAt {
	return &BackendReaderAt{ctx, r, name, blockID, tenantID, atomic.Uint64{}}
}

func (b *BackendReaderAt) ReadAt(p []byte, off int64) (int, error) {
	b.TotalBytesRead.Add(uint64(len(p)))
	err := b.r.ReadRange(b.ctx, b.name, b.blockID, b.tenantID, uint64(off), p, false)
	return len(p), err
}

func (b *BackendReaderAt) ReadAtWithCache(p []byte, off int64) (int, error) {
	err := b.r.ReadRange(b.ctx, b.name, b.blockID, b.tenantID, uint64(off), p, true)
	return len(p), err
}

type parquetOptimizedReaderAt struct {
	r          io.ReaderAt
	br         *BackendReaderAt
	readerSize int64
	footerSize uint32

	cacheControl  common.CacheControl
	cachedObjects map[int64]int64 // storing offsets and length of objects we want to cache
}

var _ io.ReaderAt = (*parquetOptimizedReaderAt)(nil)

func newParquetOptimizedReaderAt(br io.ReaderAt, rr *BackendReaderAt, size int64, footerSize uint32, cc common.CacheControl) *parquetOptimizedReaderAt {
	return &parquetOptimizedReaderAt{br, rr, size, footerSize, cc, map[int64]int64{}}
}

// called by parquet-go in OpenFile() to set offset and length of footer section
func (r *parquetOptimizedReaderAt) SetFooterSection(offset, length int64) {
	if r.cacheControl.Footer {
		r.cachedObjects[offset] = length
	}
}

// called by parquet-go in OpenFile() to set offset and length of column indexes
func (r *parquetOptimizedReaderAt) SetColumnIndexSection(offset, length int64) {
	if r.cacheControl.ColumnIndex {
		r.cachedObjects[offset] = length
	}
}

// called by parquet-go in OpenFile() to set offset and length of offset index section
func (r *parquetOptimizedReaderAt) SetOffsetIndexSection(offset, length int64) {
	if r.cacheControl.OffsetIndex {
		r.cachedObjects[offset] = length
	}
}

func (r *parquetOptimizedReaderAt) ReadAt(p []byte, off int64) (int, error) {
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
	if r.cachedObjects[off] == int64(len(p)) {
		return r.br.ReadAtWithCache(p, off)
	}

	return r.r.ReadAt(p, off)
}
