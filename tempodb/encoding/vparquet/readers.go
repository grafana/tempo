package vparquet

import (
	"context"
	"encoding/binary"
	"io"
	"os"

	"github.com/google/uuid"
	"go.uber.org/atomic"

	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

// This stack of readers is used to bridge the gap between the backend.Reader and the parquet.File.
//  each fulfills a different role.
// backend.Reader <- BackendReaderAt <- io.BufferedReaderAt <- parquetOptimizedReaderAt <- cachedReaderAt <- parquet.File
//                                \                                                         /
//                                  <------------------------------------------------------

// BackendReaderAt is used to track backend requests and present a io.ReaderAt interface backed
// by a backend.Reader
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
	// this is where Reader is account for data read...
	// parquet-go will call ReadAt when reading data from a parquet file...
	// replicate BackendReaderAt, and rename is WalReaderAt
	// or maybe we can just use BackendReaderAt for WAL??
	b.TotalBytesRead.Add(uint64(len(p)))
	err := b.r.ReadRange(b.ctx, b.name, b.blockID, b.tenantID, uint64(off), p, false)
	if err != nil {
		return 0, err
	}
	return len(p), err
}

func (b *BackendReaderAt) ReadAtWithCache(p []byte, off int64) (int, error) {
	err := b.r.ReadRange(b.ctx, b.name, b.blockID, b.tenantID, uint64(off), p, true)
	if err != nil {
		return 0, err
	}
	return len(p), err
}

// parquetOptimizedReaderAt is used to cheat a few parquet calls. By default when opening a
// file parquet always requests the magic number and then the footer length. We can save
// both of these calls from going to the backend.
type parquetOptimizedReaderAt struct {
	r          io.ReaderAt
	readerSize int64
	footerSize uint32
}

var _ io.ReaderAt = (*parquetOptimizedReaderAt)(nil)

func newParquetOptimizedReaderAt(r io.ReaderAt, size int64, footerSize uint32) *parquetOptimizedReaderAt {
	return &parquetOptimizedReaderAt{r, size, footerSize}
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

	return r.r.ReadAt(p, off)
}

// cachedReaderAt is used to route specific reads to the caching layer. this must be passed directly into
// the parquet.File so thet Set*Section() methods get called.
type cachedReaderAt struct {
	r             io.ReaderAt
	br            *BackendReaderAt
	cacheControl  common.CacheControl
	cachedObjects map[int64]int64 // storing offsets and length of objects we want to cache
}

var _ io.ReaderAt = (*cachedReaderAt)(nil)

func newCachedReaderAt(br io.ReaderAt, rr *BackendReaderAt, cc common.CacheControl) *cachedReaderAt {
	return &cachedReaderAt{br, rr, cc, map[int64]int64{}}
}

// called by parquet-go in OpenFile() to set offset and length of footer section
func (r *cachedReaderAt) SetFooterSection(offset, length int64) {
	if r.cacheControl.Footer {
		r.cachedObjects[offset] = length
	}
}

// called by parquet-go in OpenFile() to set offset and length of column indexes
func (r *cachedReaderAt) SetColumnIndexSection(offset, length int64) {
	if r.cacheControl.ColumnIndex {
		r.cachedObjects[offset] = length
	}
}

// called by parquet-go in OpenFile() to set offset and length of offset index section
func (r *cachedReaderAt) SetOffsetIndexSection(offset, length int64) {
	if r.cacheControl.OffsetIndex {
		r.cachedObjects[offset] = length
	}
}

func (r *cachedReaderAt) ReadAt(p []byte, off int64) (int, error) {
	// check if the offset and length is stored as a special object
	if r.cachedObjects[off] == int64(len(p)) {
		return r.br.ReadAtWithCache(p, off)
	}

	return r.r.ReadAt(p, off)
}

// WalReaderAt is used compute to total amount of data read when searching walBlock
type WalReaderAt struct {
	f *os.File

	TotalBytesRead atomic.Uint64
}

var _ io.ReaderAt = (*WalReaderAt)(nil)

func NewWalReaderAt(f *os.File) *WalReaderAt {
	return &WalReaderAt{f, atomic.Uint64{}}
}

func (b *WalReaderAt) ReadAt(p []byte, off int64) (int, error) {
	// this is where Reader is account for data read...
	// parquet-go will call ReadAt when reading data from a parquet file...
	// replicate BackendReaderAt, and rename is WalReaderAt
	// or maybe we can just use BackendReaderAt for WAL??
	b.TotalBytesRead.Add(uint64(len(p)))
	return b.f.ReadAt(p, off)
}
