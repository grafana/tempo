package common

import (
	"context"
	"time"

	"github.com/go-kit/log"

	"github.com/grafana/tempo/pkg/model"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/traceql"
	"github.com/grafana/tempo/tempodb/backend"
)

type Finder interface {
	FindTraceByID(ctx context.Context, id ID, opts SearchOptions) (*tempopb.Trace, error)
}

type TagCallback func(t string)

type Searcher interface {
	Search(ctx context.Context, req *tempopb.SearchRequest, opts SearchOptions) (*tempopb.SearchResponse, error)
	SearchTags(ctx context.Context, cb TagCallback, opts SearchOptions) error
	SearchTagValues(ctx context.Context, tag string, cb TagCallback, opts SearchOptions) error
	Fetch(context.Context, traceql.FetchSpansRequest, SearchOptions) (traceql.FetchSpansResponse, error)
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
	MaxBytesPerTrace   int
	OutputBlocks       uint8
	BlockConfig        BlockConfig
	Combiner           model.ObjectCombiner

	ObjectsCombined func(compactionLevel, objects int)
	ObjectsWritten  func(compactionLevel, objects int)
	BytesWritten    func(compactionLevel, bytes int)
	SpansDiscarded  func(spans int)
}

type Iterator interface {
	Next(ctx context.Context) (ID, *tempopb.Trace, error)
	Close()
}

type BackendBlock interface {
	Finder
	Searcher

	BlockMeta() *backend.BlockMeta
}

type WALBlock interface {
	BackendBlock

	Append(id ID, b []byte, start, end uint32) error
	DataLength() uint64
	Length() int
	Iterator() (Iterator, error)
	Clear() error
}
