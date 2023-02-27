package search

import (
	"context"
)

type tagCallback func(t string)

// tagValueCallback is a callback for tag values.  If it returns true, the search will stop.
type tagValueCallback func(v string) bool

type SearchableBlock interface {
	Tags(ctx context.Context, cb tagCallback) error
	TagValues(ctx context.Context, tagName string, cb tagValueCallback) error
	Search(ctx context.Context, p Pipeline, sr *Results) error
}

var _ SearchableBlock = (*StreamingSearchBlock)(nil)
var _ SearchableBlock = (*BackendSearchBlock)(nil)
