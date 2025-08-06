package unsupported

import (
	"context"

	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/traceql"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

type Block struct{}

var _ common.BackendBlock = &Block{}

func (b Block) FindTraceByID(ctx context.Context, id common.ID, opts common.SearchOptions) (*tempopb.TraceByIDResponse, error) {
	return nil, common.ErrUnsupported
}

func (b Block) BlockMeta() *backend.BlockMeta {
	return nil
}

func (b Block) Search(ctx context.Context, req *tempopb.SearchRequest, opts common.SearchOptions) (*tempopb.SearchResponse, error) {
	return nil, nil
}

func (b Block) SearchTags(ctx context.Context, scope traceql.AttributeScope, cb common.TagsCallback, mcb common.MetricsCallback, opts common.SearchOptions) error {
	return nil
}

func (b Block) SearchTagValues(ctx context.Context, tag string, cb common.TagValuesCallback, mcb common.MetricsCallback, opts common.SearchOptions) error {
	return nil
}

func (b Block) SearchTagValuesV2(ctx context.Context, tag traceql.Attribute, cb common.TagValuesCallbackV2, mcb common.MetricsCallback, opts common.SearchOptions) error {
	return nil
}

func (b Block) Fetch(ctx context.Context, req traceql.FetchSpansRequest, opts common.SearchOptions) (traceql.FetchSpansResponse, error) {
	return traceql.FetchSpansResponse{}, nil
}

func (b Block) FetchTagValues(context.Context, traceql.FetchTagValuesRequest, traceql.FetchTagValuesCallback, common.MetricsCallback, common.SearchOptions) error {
	return nil
}

func (b Block) FetchTagNames(context.Context, traceql.FetchTagsRequest, traceql.FetchTagsCallback, common.MetricsCallback, common.SearchOptions) error {
	return nil
}

func (b Block) Validate(ctx context.Context) error {
	return nil
}
