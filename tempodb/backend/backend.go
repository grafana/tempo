package backend

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

var ErrMetaDoesNotExist = fmt.Errorf("meta does not exist")

type AppendTracker interface{}

type Writer interface {
	Write(ctx context.Context, meta *BlockMeta, bBloom []byte, bIndex []byte, objectFilePath string) error

	WriteBlockMeta(ctx context.Context, tracker AppendTracker, meta *BlockMeta, bBloom []byte, bIndex []byte) error
	AppendObject(ctx context.Context, tracker AppendTracker, meta *BlockMeta, bObject []byte) (AppendTracker, error)
}

type Reader interface {
	Tenants() ([]string, error)
	Blocks(tenantID string) ([]uuid.UUID, error)
	BlockMeta(blockID uuid.UUID, tenantID string) (*BlockMeta, error)
	Bloom(blockID uuid.UUID, tenantID string) ([]byte, error)
	Index(blockID uuid.UUID, tenantID string) ([]byte, error)
	Object(blockID uuid.UUID, tenantID string, start uint64, buffer []byte) error

	Shutdown()
}

type Compactor interface {
	MarkBlockCompacted(blockID uuid.UUID, tenantID string) error
	ClearBlock(blockID uuid.UUID, tenantID string) error
	CompactedBlockMeta(blockID uuid.UUID, tenantID string) (*CompactedBlockMeta, error)
}
