package search

import (
	"context"
)

type tagCallback func(t string)

type SearchableBlock interface {
	Tags(ctx context.Context, cb tagCallback) error
	TagValues(ctx context.Context, tagName string, cb tagCallback) error
	Search(ctx context.Context, p Pipeline, sr *Results) error
}

var _ SearchableBlock = (*StreamingSearchBlock)(nil)
var _ SearchableBlock = (*BackendSearchBlock)(nil)
