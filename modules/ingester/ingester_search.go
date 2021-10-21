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

	tags, err := inst.GetSearchTags(ctx)
	if err != nil {
		return nil, err
	}

	return &tempopb.SearchTagsResponse{
		TagNames: tags,
	}, nil
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

	vals, err := inst.GetSearchTagValues(ctx, req.TagName)
	if err != nil {
		return nil, err
	}

	return &tempopb.SearchTagValuesResponse{
		TagValues: vals,
	}, nil
}
