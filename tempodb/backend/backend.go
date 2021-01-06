package backend

import (
	"context"
	"fmt"
	"io"

	"github.com/google/uuid"
)

var (
	ErrMetaDoesNotExist = fmt.Errorf("meta does not exist")
	ErrEmptyTenantID    = fmt.Errorf("empty tenant id")
	ErrEmptyBlockID     = fmt.Errorf("empty block id")

	MaxCacheSizeBytes = int64(1024 * 1024)
)

type AppendTracker interface{}

type Writer interface {
	Write(ctx context.Context, name string, blockID uuid.UUID, tenantID string, data io.Reader, size int64) error
	WriteBlockMeta(ctx context.Context, meta *BlockMeta) error

	Append(ctx context.Context, name string, blockID uuid.UUID, tenantID string, tracker AppendTracker, buffer []byte) (AppendTracker, error)
	CloseAppend(ctx context.Context, tracker AppendTracker) error
}

type Reader interface {
	Tenants(ctx context.Context) ([]string, error)
	Blocks(ctx context.Context, tenantID string) ([]uuid.UUID, error)
	BlockMeta(ctx context.Context, blockID uuid.UUID, tenantID string) (*BlockMeta, error)
	Bloom(ctx context.Context, blockID uuid.UUID, tenantID string, bloomShard int) ([]byte, error)
	Index(ctx context.Context, blockID uuid.UUID, tenantID string) ([]byte, error)
	Object(ctx context.Context, blockID uuid.UUID, tenantID string, offset uint64, buffer []byte) error

	Shutdown()
}

type Compactor interface {
	MarkBlockCompacted(blockID uuid.UUID, tenantID string) error
	ClearBlock(blockID uuid.UUID, tenantID string) error
	CompactedBlockMeta(blockID uuid.UUID, tenantID string) (*CompactedBlockMeta, error)
}
