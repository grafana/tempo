package vparquet2

import (
	"sync"

	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

const (
	DataFileName = "data.parquet"
)

type BackendBlock struct {
	meta *backend.BlockMeta
	r    backend.Reader

	openMtx sync.Mutex
}

var _ common.BackendBlock = (*BackendBlock)(nil)

func NewBackendBlock(meta *backend.BlockMeta, r backend.Reader) *BackendBlock {
	return &BackendBlock{
		meta: meta,
		r:    r,
	}
}

func (b *BackendBlock) BlockMeta() *backend.BlockMeta {
	return b.meta
}
