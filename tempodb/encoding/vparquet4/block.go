package vparquet3

import (
	"sync"

	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

const (
	DataFileName = "data.parquet"
)

type backendBlock struct {
	meta *backend.BlockMeta
	r    backend.Reader

	openMtx sync.Mutex
}

var _ common.BackendBlock = (*backendBlock)(nil)

func newBackendBlock(meta *backend.BlockMeta, r backend.Reader) *backendBlock {
	return &backendBlock{
		meta: meta,
		r:    r,
	}
}

func (b *backendBlock) BlockMeta() *backend.BlockMeta {
	return b.meta
}
