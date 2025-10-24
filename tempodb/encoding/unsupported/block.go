package unsupported

import (
	"context"

	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/traceql"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

type Block struct {
	meta *backend.BlockMeta
}

var _ common.BackendBlock = &Block{}

func (b Block) FindTraceByID(context.Context, common.ID, common.SearchOptions) (*tempopb.TraceByIDResponse, error) {
	return nil, util.ErrUnsupported
}

func (b Block) BlockMeta() *backend.BlockMeta {
	return b.meta
}

func (b Block) Search(context.Context, *tempopb.SearchRequest, common.SearchOptions) (*tempopb.SearchResponse, error) {
	return nil, nil
}

func (b Block) SearchTags(context.Context, traceql.AttributeScope, common.TagsCallback, common.MetricsCallback, common.SearchOptions) error {
	return nil
}

func (b Block) SearchTagValues(context.Context, string, common.TagValuesCallback, common.MetricsCallback, common.SearchOptions) error {
	return nil
}

func (b Block) SearchTagValuesV2(context.Context, traceql.Attribute, common.TagValuesCallbackV2, common.MetricsCallback, common.SearchOptions) error {
	return nil
}

func (b Block) Fetch(context.Context, traceql.FetchSpansRequest, common.SearchOptions) (traceql.FetchSpansResponse, error) {
	return traceql.FetchSpansResponse{}, util.ErrUnsupported
}

func (b Block) FetchTagValues(context.Context, traceql.FetchTagValuesRequest, traceql.FetchTagValuesCallback, common.MetricsCallback, common.SearchOptions) error {
	return nil
}

func (b Block) FetchTagNames(context.Context, traceql.FetchTagsRequest, traceql.FetchTagsCallback, common.MetricsCallback, common.SearchOptions) error {
	return nil
}

func (b Block) Validate(context.Context) error {
	return nil
}
