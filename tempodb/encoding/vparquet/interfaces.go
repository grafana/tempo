package vparquet

import (
	"context"

	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/segmentio/parquet-go"
)

type TraceIterator interface {
	NextTrace(context.Context) (common.ID, *Trace, error)
	Close()
	// __vparquet() jpe add a hidden method to prevent external implementations?
}

type RawIterator interface {
	Next(context.Context) (common.ID, parquet.Row, error)
	Close()
}
