package search

import (
	"context"
)

type SearchableBlock interface {
	Tags(ctx context.Context, tags map[string]struct{}) error
	TagValues(ctx context.Context, tag string, tagValues map[string]struct{}) error
	Search(ctx context.Context, p Pipeline, sr *Results) error
}

var _ SearchableBlock = (*StreamingSearchBlock)(nil)
var _ SearchableBlock = (*BackendSearchBlock)(nil)
