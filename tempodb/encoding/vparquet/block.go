package vparquet

import (
	"context"
	"sync"

	"github.com/grafana/tempo/pkg/traceql"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"go.opentelemetry.io/otel"
)

const (
	DataFileName = "data.parquet"
)

var tracer = otel.Tracer("tempodb/encoding/vparquet")

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

func (b *backendBlock) FetchTagValues(context.Context, traceql.AutocompleteRequest, traceql.AutocompleteCallback, common.SearchOptions) error {
	// TODO: Add support?
	return common.ErrUnsupported
}
