package common

import (
	"context"

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

type TagCallbackV2 func(traceql.Static) (stop bool)

type Searcher interface {
	Search(ctx context.Context, req *tempopb.SearchRequest, opts SearchOptions) (*tempopb.SearchResponse, error)
	SearchTags(ctx context.Context, scope traceql.AttributeScope, cb TagCallback, opts SearchOptions) error
	SearchTagValues(ctx context.Context, tag string, cb TagCallback, opts SearchOptions) error
	SearchTagValuesV2(ctx context.Context, tag traceql.Attribute, cb TagCallbackV2, opts SearchOptions) error

	Fetch(context.Context, traceql.FetchSpansRequest, SearchOptions) (traceql.FetchSpansResponse, error)
	FetchTagValues(context.Context, traceql.AutocompleteRequest, traceql.AutocompleteCallback, SearchOptions) error
}

type SearchOptions struct {
	ChunkSizeBytes     uint32 // Buffer size to read from backend storage.
	StartPage          int    // Controls searching only a subset of the block. Which page to begin searching at.
	TotalPages         int    // Controls searching only a subset of the block. How many pages to search.
	MaxBytes           int    // Max allowable trace size in bytes. Traces exceeding this are not searched.
	PrefetchTraceCount int    // How many traces to prefetch async.
	ReadBufferCount    int
	ReadBufferSize     int
}

// DefaultSearchOptions() is used in a lot of places such as local ingester searches. It is important
// in these cases to set a reasonable read buffer size and count to prevent constant tiny readranges
// against the local backend.
// TODO: Note that there is another method of creating "default search options" that looks like this:
// tempodb.SearchConfig{}.ApplyToOptions(&searchOpts). we should consolidate these.
func DefaultSearchOptions() SearchOptions {
	return SearchOptions{
		ReadBufferCount: 32,
		ReadBufferSize:  1024 * 1024,
		ChunkSizeBytes:  4 * 1024 * 1024,
	}
}

func DefaultSearchOptionsWithMaxBytes(maxBytes int) SearchOptions {
	return SearchOptions{
		ReadBufferCount: 32,
		ReadBufferSize:  1024 * 1024,
		ChunkSizeBytes:  4 * 1024 * 1024,
		MaxBytes:        maxBytes,
	}
}

type Compactor interface {
	Compact(ctx context.Context, l log.Logger, r backend.Reader, w backend.Writer, inputs []*backend.BlockMeta) ([]*backend.BlockMeta, error)
}

type CompactionOptions struct {
	ChunkSizeBytes     uint32
	FlushSizeBytes     uint32
	IteratorBufferSize int // How many traces to prefetch async.
	MaxBytesPerTrace   int
	OutputBlocks       uint8
	BlockConfig        BlockConfig
	Combiner           model.ObjectCombiner

	ObjectsCombined   func(compactionLevel, objects int)
	ObjectsWritten    func(compactionLevel, objects int)
	BytesWritten      func(compactionLevel, bytes int)
	SpansDiscarded    func(traceID string, rootSpanName string, rootServiceName string, spans int)
	DisconnectedTrace func()
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

	// Append the given trace to the block. Must be safe for concurrent use with read operations.
	Append(id ID, b []byte, start, end uint32) error

	AppendTrace(id ID, tr *tempopb.Trace, start, end uint32) error

	// Flush any unbuffered data to disk.  Must be safe for concurrent use with read operations.
	Flush() error

	DataLength() uint64
	Iterator() (Iterator, error)
	Clear() error
}
