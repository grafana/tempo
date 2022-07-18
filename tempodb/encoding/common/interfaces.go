package common

import (
	"context"
	"time"

	"github.com/go-kit/log"
	"github.com/grafana/tempo/pkg/model"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/tempodb/backend"
)

type Finder interface {
	FindTraceByID(ctx context.Context, id ID) (*tempopb.Trace, error)
}

type Searcher interface {
	Search(ctx context.Context, req *tempopb.SearchRequest, opts SearchOptions) (*tempopb.SearchResponse, error)
}

type CacheControl struct {
	Footer      bool
	ColumnIndex bool
	OffsetIndex bool
}

type SearchOptions struct {
	ChunkSizeBytes     uint32 // Buffer size to read from backend storage.
	StartPage          int    // Controls searching only a subset of the block. Which page to begin searching at.
	TotalPages         int    // Controls searching only a subset of the block. How many pages to search.
	MaxBytes           int    // Max allowable trace size in bytes. Traces exceeding this are not searched.
	PrefetchTraceCount int    // How many traces to prefetch async.
	ReadBufferCount    int
	ReadBufferSize     int
	CacheControl       CacheControl
}

type Compactor interface {
	Compact(ctx context.Context, l log.Logger, r backend.Reader, writerCallback func(*backend.BlockMeta, time.Time) backend.Writer, inputs []*backend.BlockMeta) ([]*backend.BlockMeta, error)
}

type CompactionOptions struct {
	ChunkSizeBytes     uint32
	FlushSizeBytes     uint32
	IteratorBufferSize int // How many traces to prefetch async.
	OutputBlocks       uint8
	BlockConfig        BlockConfig
	Combiner           model.ObjectCombiner

	ObjectsWritten func(compactionLevel, objects int)
	BytesWritten   func(compactionLevel, bytes int)
}

func DefaultCompactionOptions() CompactionOptions {
	return CompactionOptions{
		ChunkSizeBytes:     1_000_000,
		FlushSizeBytes:     30 * 1024 * 1024, // 30 MiB
		IteratorBufferSize: 1000,
		OutputBlocks:       1,
	}
}

type Iterator interface {
	// Next returns the next trace and optionally the start and stop times
	// for the trace that may have been adjusted.
	Next(ctx context.Context) (ID, []byte, error)
	Close()
}

type BackendBlock interface {
	Finder
	Searcher

	BlockMeta() *backend.BlockMeta
}
