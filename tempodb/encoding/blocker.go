package encoding

import (
	"context"

	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

type Finder interface {
	FindTraceByID(ctx context.Context, id common.ID) (*tempopb.Trace, error)
}

type Searcher interface {
	Search(ctx context.Context, req *tempopb.SearchRequest, opts SearchOptions) (*tempopb.SearchResponse, error)
}

type SearchOptions struct {
	ChunkSizeBytes     uint32 // Buffer size to read from backend storage.
	StartPage          int    // Controls searching only a subset of the block. Which page to begin searching at.
	TotalPages         int    // Controls searching only a subset of the block. How many pages to search.
	MaxBytes           int    // Max allowable trace size in bytes. Traces exceeding this are not searched.
	PrefetchTraceCount int    // How many traces to prefetch async.
}

func DefaultSearchOptions() SearchOptions {
	return SearchOptions{
		ChunkSizeBytes: 1_000_000,
	}
}
