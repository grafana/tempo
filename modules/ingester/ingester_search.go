package ingester

import (
	"context"

	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/weaveworks/common/user"
)

const searchDir = "search"

func (i *Ingester) Search(ctx context.Context, req *tempopb.SearchRequest) (*tempopb.SearchResponse, error) {
	instanceID, err := user.ExtractOrgID(ctx)
	if err != nil {
		return nil, err
	}
	inst, ok := i.getInstanceByID(instanceID)
	if !ok || inst == nil {
		return &tempopb.SearchResponse{}, nil
	}

	res, err := inst.Search(ctx, req)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (i *Ingester) SearchTags(ctx context.Context, req *tempopb.SearchTagsRequest) (*tempopb.SearchTagsResponse, error) {
	instanceID, err := user.ExtractOrgID(ctx)
	if err != nil {
		return nil, err
	}
	inst, ok := i.getInstanceByID(instanceID)
	if !ok || inst == nil {
		return &tempopb.SearchTagsResponse{}, nil
	}

	res, err := inst.SearchTags(ctx)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (i *Ingester) SearchTagValues(ctx context.Context, req *tempopb.SearchTagValuesRequest) (*tempopb.SearchTagValuesResponse, error) {
	instanceID, err := user.ExtractOrgID(ctx)
	if err != nil {
		return nil, err
	}
	inst, ok := i.getInstanceByID(instanceID)
	if !ok || inst == nil {
		return &tempopb.SearchTagValuesResponse{}, nil
	}

	res, err := inst.SearchTagValues(ctx, req.TagName)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// todo(search): consolidate. this only exists so that the ingester continues to implement the tempopb.QuerierServer interface.
func (i *Ingester) BackendSearch(ctx context.Context, req *tempopb.BackendSearchRequest) (*tempopb.SearchResponse, error) {
	return nil, nil
}
