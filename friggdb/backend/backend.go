package backend

import (
	"context"

	"github.com/google/uuid"
	"github.com/grafana/frigg/friggdb/encoding"
)

type Writer interface {
	Write(ctx context.Context, blockID uuid.UUID, tenantID string, meta *encoding.BlockMeta, bBloom []byte, bIndex []byte, objectFilePath string) error
}

type Reader interface {
	Tenants() ([]string, error)
	Blocks(tenantID string) ([]uuid.UUID, error)
	BlockMeta(blockID uuid.UUID, tenantID string) (*encoding.BlockMeta, error)
	Bloom(blockID uuid.UUID, tenantID string) ([]byte, error)
	Index(blockID uuid.UUID, tenantID string) ([]byte, error)
	Object(blockID uuid.UUID, tenantID string, start uint64, length uint32) ([]byte, error)

	Shutdown()
}

type Compactor interface {
	MarkBlockCompacted(blockID uuid.UUID, tenantID string) error
	ClearBlock(blockID uuid.UUID, tenantID string) error
	CompactedBlockMeta(blockID uuid.UUID, tenantID string) (*encoding.CompactedBlockMeta, error)
}
