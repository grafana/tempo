package vparquet

import (
	"context"

	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

func (b *backendBlock) FindTraceByID(ctx context.Context, id common.ID) (*tempopb.Trace, error) {
	return nil, nil
}
