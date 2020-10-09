package backend

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/grafana/tempo/tempodb/encoding"
)

var (
	ErrMetaDoesNotExist = fmt.Errorf("meta does not exist")
	ErrEmptyTenantID    = fmt.Errorf("empty tenant id")
	ErrEmptyBlockID     = fmt.Errorf("empty block id")
)

type AppendTracker interface{}

type Writer interface {
	Write(ctx context.Context, meta *encoding.BlockMeta, bBloom []byte, bIndex []byte, objectFilePath string) error

	WriteBlockMeta(ctx context.Context, tracker AppendTracker, meta *encoding.BlockMeta, bBloom []byte, bIndex []byte) error
	AppendObject(ctx context.Context, tracker AppendTracker, meta *encoding.BlockMeta, bObject []byte) (AppendTracker, error)
}

type Reader interface {
	Tenants(ctx context.Context) ([]string, error)
	Blocks(ctx context.Context, tenantID string) ([]uuid.UUID, error)
	BlockMeta(ctx context.Context, blockID uuid.UUID, tenantID string) (*encoding.BlockMeta, error)
	Bloom(ctx context.Context, blockID uuid.UUID, tenantID string) ([]byte, error)
	Index(ctx context.Context, blockID uuid.UUID, tenantID string) ([]byte, error)
	Object(ctx context.Context, blockID uuid.UUID, tenantID string, offset uint64, buffer []byte) error

	Shutdown()
}

type Compactor interface {
	MarkBlockCompacted(blockID uuid.UUID, tenantID string) error
	ClearBlock(blockID uuid.UUID, tenantID string) error
	CompactedBlockMeta(blockID uuid.UUID, tenantID string) (*encoding.CompactedBlockMeta, error)
}
