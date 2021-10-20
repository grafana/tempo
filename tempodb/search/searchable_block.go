package search

import (
	"context"
)

type SearchableBlock interface {
	Search(ctx context.Context, p Pipeline, sr *Results) error
}

var _ SearchableBlock = (*StreamingSearchBlock)(nil)
var _ SearchableBlock = (*BackendSearchBlock)(nil)
