package vparquet4

import (
	"context"

	"github.com/grafana/tempo/v2/tempodb/encoding/common"
	"github.com/parquet-go/parquet-go"
)

type TraceIterator interface {
	NextTrace(context.Context) (common.ID, *Trace, error)
	Close()
}

type RawIterator interface {
	Next(context.Context) (common.ID, parquet.Row, error)
	Close()
}
