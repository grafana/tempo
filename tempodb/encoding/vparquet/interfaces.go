package vparquet

import (
	"context"

	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/segmentio/parquet-go"
)

type Iterator interface {
	Next(context.Context) (*Trace, error)
	Close()
}

type RawIterator interface {
	Next(context.Context) (common.ID, parquet.Row, error)
	Close()
}
