package v1

import (
	"context"

	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

// used by the compactor, cli, and tempodb.Find

type BackendBlock struct {
	meta *backend.BlockMeta
}

// NewBackendBlock returns a block used for finding traces in the backend
func NewBackendBlock(meta *backend.BlockMeta) *BackendBlock {
	return &BackendBlock{
		meta: meta,
	}
}

// Find searches a block for the ID and returns an object if found.
func (b *BackendBlock) Find(ctx context.Context, r backend.Reader, id common.ID) ([]byte, error) {
	return nil, nil
}

// Iterator searches a block for the ID and returns an object if found.
func (b *BackendBlock) Iterator(chunkSizeBytes uint32, r backend.Reader) (common.Iterator, error) {
	// definitely will need a new iterator.  based on

	return nil, nil
}
