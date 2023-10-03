package ingester

import (
	"context"
	"errors"

	"github.com/grafana/dskit/user"
	"github.com/grafana/tempo/pkg/tempopb"
)

func (i *Ingester) SearchRecent(ctx context.Context, req *tempopb.SearchRequest) (*tempopb.SearchResponse, error) {
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

	res, err := inst.SearchTags(ctx, req.Scope)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (i *Ingester) SearchTagsV2(ctx context.Context, req *tempopb.SearchTagsRequest) (*tempopb.SearchTagsV2Response, error) {
	instanceID, err := user.ExtractOrgID(ctx)
	if err != nil {
		return nil, err
	}
	inst, ok := i.getInstanceByID(instanceID)
	if !ok || inst == nil {
		return &tempopb.SearchTagsV2Response{}, nil
	}

	res, err := inst.SearchTagsV2(ctx, req.Scope)
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

func (i *Ingester) SearchTagValuesV2(ctx context.Context, req *tempopb.SearchTagValuesRequest) (*tempopb.SearchTagValuesV2Response, error) {
	instanceID, err := user.ExtractOrgID(ctx)
	if err != nil {
		return nil, err
	}
	inst, ok := i.getInstanceByID(instanceID)
	if !ok || inst == nil {
		return &tempopb.SearchTagValuesV2Response{}, nil
	}

	res, err := inst.SearchTagValuesV2(ctx, req)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// SearchBlock only exists here to fulfill the protobuf interface. The ingester will never support
// backend search
func (i *Ingester) SearchBlock(context.Context, *tempopb.SearchBlockRequest) (*tempopb.SearchResponse, error) {
	return nil, errors.New("not implemented")
}
