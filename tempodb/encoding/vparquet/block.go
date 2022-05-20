package vparquet

import (
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

const (
	DataFileName = "data.parquet"
)

type backendBlock struct {
	meta *backend.BlockMeta
	r    backend.Reader
}

var _ common.BackendBlock = (*backendBlock)(nil)

func NewBackendBlock(meta *backend.BlockMeta, r backend.Reader) (*backendBlock, error) {
	return &backendBlock{meta, r}, nil
}

func (b *backendBlock) BlockMeta() *backend.BlockMeta {
	return b.meta
}
