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
	Search(ctx context.Context, req *tempopb.SearchRequest, opts ...SearchOption) (*tempopb.SearchResponse, error)
}

type SearchOptions struct {
	chunkSizeBytes     uint32
	startPage          int
	totalPages         int
	maxBytes           int
	prefetchTraceCount int
}

type SearchOption func(*SearchOptions)

func WithPages(startPage, totalPages int) SearchOption {
	return func(opt *SearchOptions) {
		opt.startPage = startPage
		opt.totalPages = totalPages
	}
}

func WithChunkSize(chunkSizeBytes uint32) SearchOption {
	return func(opt *SearchOptions) {
		opt.chunkSizeBytes = chunkSizeBytes
	}
}

func WithMaxBytes(maxBytes int) SearchOption {
	return func(opt *SearchOptions) {
		opt.maxBytes = maxBytes
	}
}

func WithPrefetchTraceCount(count int) SearchOption {
	return func(opt *SearchOptions) {
		opt.prefetchTraceCount = count
	}
}
