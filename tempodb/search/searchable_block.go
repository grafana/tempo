package search

import (
	"context"
)

type SearchableBlock interface {
	Search(ctx context.Context, p Pipeline, sr *Results) error
}
