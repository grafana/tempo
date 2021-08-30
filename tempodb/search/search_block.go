package search

import (
	"context"
)

// nolint:golint
type SearchBlock interface {
	Search(ctx context.Context, p Pipeline, sr *SearchResults) error
}
