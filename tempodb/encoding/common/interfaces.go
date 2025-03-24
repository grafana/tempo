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
	FindTraceByID(ctx context.Context, id ID, opts SearchOptions) (*tempopb.TraceByIDResponse, error)
}

type (
	TagsCallback        func(t string, scope traceql.AttributeScope)
	TagValuesCallback   func(t string) bool
	TagValuesCallbackV2 func(traceql.Static) (stop bool)
	MetricsCallback     func(bytesRead uint64) // callback for accumulating bytesRead
)

type Searcher interface {
	Search(ctx context.Context, req *tempopb.SearchRequest, opts SearchOptions) (*tempopb.SearchResponse, error)
	SearchTags(ctx context.Context, scope traceql.AttributeScope, cb TagsCallback, mcb MetricsCallback, opts SearchOptions) error
	SearchTagValues(ctx context.Context, tag string, cb TagValuesCallback, mcb MetricsCallback, opts SearchOptions) error
	SearchTagValuesV2(ctx context.Context, tag traceql.Attribute, cb TagValuesCallbackV2, mcb MetricsCallback, opts SearchOptions) error

	// TODO(suraj): use MetricsCallback in Fetch and remove the Bytes callback from FetchSpansResponse
	Fetch(context.Context, traceql.FetchSpansRequest, SearchOptions) (traceql.FetchSpansResponse, error)
	FetchTagValues(context.Context, traceql.FetchTagValuesRequest, traceql.FetchTagValuesCallback, MetricsCallback, SearchOptions) error
	FetchTagNames(context.Context, traceql.FetchTagsRequest, traceql.FetchTagsCallback, MetricsCallback, SearchOptions) error
}

type SearchOptions struct {
	ChunkSizeBytes     uint32 // Buffer size to read from backend storage.
	StartPage          int    // Controls searching only a subset of the block. Which page to begin searching at.
	TotalPages         int    // Controls searching only a subset of the block. How many pages to search.
	MaxBytes           int    // Max allowable trace size in bytes. Traces exceeding this are not searched.
	PrefetchTraceCount int    // How many traces to prefetch async.
	ReadBufferCount    int
	ReadBufferSize     int
	RF1After           time.Time // Only blocks with RF1 are selected after this timestamp. RF3 is selected otherwise.
}

// DefaultSearchOptions is used in a lot of places such as local ingester searches. It is important
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
	opts := DefaultSearchOptions()
	opts.MaxBytes = maxBytes
	return opts
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

	// DropObject can be used to drop a trace from the compaction process. Currently it only receives the ID
	// of the trace to be compacted. If the function returns true, the trace will be dropped.
	DropObject func(ID) bool

	ObjectsCombined   func(compactionLevel, objects int)
	ObjectsWritten    func(compactionLevel, objects int)
	BytesWritten      func(compactionLevel, bytes int)
	SpansDiscarded    func(traceID string, rootSpanName string, rootServiceName string, spans int)
	DisconnectedTrace func()
	RootlessTrace     func()
	DedupedSpans      func(replFactor, dedupedSpans int)
}

type Iterator interface {
	Next(ctx context.Context) (ID, *tempopb.Trace, error)
	Close()
}

type BackendBlock interface {
	Finder
	Searcher

	BlockMeta() *backend.BlockMeta
	Validate(ctx context.Context) error
}

// WALBlock represents a Write-Ahead Log (WAL) block interface that extends the BackendBlock interface.
// It provides methods to append traces, manage ingestion slack, flush data, and iterate over the block's data.
type WALBlock interface {
	BackendBlock

	// Append adds the given trace to the block. This method must be safe for concurrent use with read operations.
	// Parameters:
	// - id: The ID of the trace.
	// - b: The byte slice representing the trace data.
	// - start: The start time of the trace.
	// - end: The end time of the trace.
	// - adjustIngestionSlack: If true, adjusts the ingestion slack based on the current time (now()).
	// Returns an error if the append operation fails.
	Append(id ID, b []byte, start, end uint32, adjustIngestionSlack bool) error

	// AppendTrace adds the given trace to the block. This method must be safe for concurrent use with read operations.
	// Parameters:
	// - id: The ID of the trace.
	// - tr: The trace object.
	// - start: The start time of the trace.
	// - end: The end time of the trace.
	// - adjustIngestionSlack: If true, adjusts the ingestion slack based on the current time (now()).
	// Returns an error if the append operation fails.
	AppendTrace(id ID, tr *tempopb.Trace, start, end uint32, adjustIngestionSlack bool) error

	// IngestionSlack returns the duration of the ingestion slack.
	IngestionSlack() time.Duration

	// Flush writes any unbuffered data to disk. This method must be safe for concurrent use with read operations.
	// Returns an error if the flush operation fails.
	Flush() error

	// DataLength returns the length of the data in the block.
	DataLength() uint64

	// Iterator returns an iterator for the block's data.
	// Returns an error if the iterator creation fails.
	Iterator() (Iterator, error)

	// Clear clears the block's data.
	// Returns an error if the clear operation fails.
	Clear() error
}
