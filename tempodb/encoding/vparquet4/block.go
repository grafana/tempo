package vparquet4

import (
	"sync"

	"github.com/grafana/tempo/tempodb/backend"
	backend_v1 "github.com/grafana/tempo/tempodb/backend/v1"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

const (
	DataFileName = "data.parquet"
)

type backendBlock struct {
	meta *backend_v1.BlockMeta
	r    backend.Reader

	openMtx sync.Mutex
}

var _ common.BackendBlock = (*backendBlock)(nil)

func newBackendBlock(meta *backend_v1.BlockMeta, r backend.Reader) *backendBlock {
	return &backendBlock{
		meta: meta,
		r:    r,
	}
}

func (b *backendBlock) BlockMeta() *backend_v1.BlockMeta {
	return b.meta
}
