package vparquet

import (
	"context"
	"errors"

	"time"

	"github.com/go-kit/log"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

func NewCompactor() common.Compactor {
	return &Compactor{}
}

type Compactor struct {
}

func (c *Compactor) Compact(ctx context.Context, l log.Logger, r backend.Reader, writerCallback func(*backend.BlockMeta, time.Time) backend.Writer, inputs []*backend.BlockMeta, opts common.CompactionOptions) ([]*backend.BlockMeta, error) {
	return nil, errors.New("compaction not supported")
}
