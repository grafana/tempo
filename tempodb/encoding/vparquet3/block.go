package vparquet3

import (
	"context"
	"sync"

	"github.com/grafana/tempo/pkg/traceql"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"go.opentelemetry.io/otel"
)

const (
	DataFileName = "data.parquet"
)

var tracer = otel.Tracer("tempodb/encoding/vparquet3")

type backendBlock struct {
	meta *backend.BlockMeta
	r    backend.Reader

	openMtx sync.Mutex
}

var _ common.BackendBlock = (*backendBlock)(nil)

func newBackendBlock(meta *backend.BlockMeta, r backend.Reader) *backendBlock {
	return &backendBlock{
		meta: meta,
		r:    r,
	}
}

func (b *backendBlock) BlockMeta() *backend.BlockMeta {
	return b.meta
}

func (b *backendBlock) Validate(context.Context) error {
	return util.ErrUnsupported
}

func (b *backendBlock) FetcherFor(opts common.SearchOptions) traceql.Fetcher {
	return &blockFetcher{b: b, opts: opts}
}

type blockFetcher struct {
	b    *backendBlock
	opts common.SearchOptions
}

var _ traceql.Fetcher = (*blockFetcher)(nil)

func (b *blockFetcher) SpansetFetcher() traceql.SpansetFetcher {
	return b
}

func (b *blockFetcher) SpanFetcher() traceql.SpanFetcher {
	return nil
}

func (b *blockFetcher) Fetch(ctx context.Context, req traceql.FetchSpansRequest) (traceql.FetchSpansResponse, error) {
	return b.b.Fetch(ctx, req, b.opts)
}
