package search

import (
	"context"
)

type SearchBlock interface {
	Search(ctx context.Context, p Pipeline, sr *SearchResults) error
}
