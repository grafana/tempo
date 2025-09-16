package vparquet2

import (
	"context"
	"sync"

	"github.com/grafana/tempo/pkg/traceql"
	"github.com/grafana/tempo/pkg/util"
	"go.opentelemetry.io/otel"

	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

const (
	DataFileName = "data.parquet"
)

var tracer = otel.Tracer("tempodb/encoding/vparquet2")

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

func (b *backendBlock) FetchTagValues(context.Context, traceql.FetchTagValuesRequest, traceql.FetchTagValuesCallback, common.MetricsCallback, common.SearchOptions) error {
	return util.ErrUnsupported
}

func (b *backendBlock) FetchTagNames(context.Context, traceql.FetchTagsRequest, traceql.FetchTagsCallback, common.MetricsCallback, common.SearchOptions) error {
	return util.ErrUnsupported
}

func (b *backendBlock) Validate(context.Context) error {
	return util.ErrUnsupported
}
