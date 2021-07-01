package search

import (
	"context"

	"github.com/grafana/tempo/pkg/tempopb"
)

type SearchBlock interface {
	Search(ctx context.Context, p Pipeline) ([]*tempopb.TraceSearchMetadata, error)
}

/*
type StreamingSearchBlock interface {
	SearchBlock

	Append(ctx context.Context, id common.ID, searchData [][]byte) error
}*/
