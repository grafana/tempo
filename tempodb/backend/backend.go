package backend

import (
	"context"
	"fmt"
	"io"

	"github.com/google/uuid"
)

var (
	ErrDoesNotExist  = fmt.Errorf("does not exist")
	ErrEmptyTenantID = fmt.Errorf("empty tenant id")
	ErrEmptyBlockID  = fmt.Errorf("empty block id")
)

// AppendTracker is an empty interface usable by the backend to track a long running append operation
type AppendTracker interface{}

// jpe comment all of this

// Writer is a collection of methods to write data to tempodb backends
type Writer interface {
	// Write is for in memory data.  It is expected that this data will be cached.
	Write(ctx context.Context, name string, blockID uuid.UUID, tenantID string, buffer []byte) error
	// StreamWriter is for larger data payloads streamed through an io.Reader.  It is expected this will _not_ be cached.
	StreamWriter(ctx context.Context, name string, blockID uuid.UUID, tenantID string, data io.Reader, size int64) error
	// WriteBlockMeta writes a block meta to its blocks
	WriteBlockMeta(ctx context.Context, meta *BlockMeta) error

	Append(ctx context.Context, name string, blockID uuid.UUID, tenantID string, tracker AppendTracker, buffer []byte) (AppendTracker, error)
	CloseAppend(ctx context.Context, tracker AppendTracker) error
}

// Reader is a collection of methods to read data from tempodb backends
type Reader interface {
	// Reader is for reading entire objects from the backend.  It is expected that there will be an attempt to retrieve this from cache
	Read(ctx context.Context, name string, blockID uuid.UUID, tenantID string) ([]byte, error)
	// StreamReader is for streaming entire objects from the backend.  It is expected this will _not_ be cached.
	StreamReader(ctx context.Context, name string, blockID uuid.UUID, tenantID string) (io.ReadCloser, int64, error)
	// ReadRange is for reading parts of large objects from the backend.  It is expected this will _not_ be cached.
	ReadRange(ctx context.Context, name string, blockID uuid.UUID, tenantID string, offset uint64, buffer []byte) error

	Tenants(ctx context.Context) ([]string, error)
	Blocks(ctx context.Context, tenantID string) ([]uuid.UUID, error)
	BlockMeta(ctx context.Context, blockID uuid.UUID, tenantID string) (*BlockMeta, error)

	Shutdown()
}

// Compactor is a collection of methods to interact with compacted elements of a tempodb block
type Compactor interface {
	MarkBlockCompacted(blockID uuid.UUID, tenantID string) error
	ClearBlock(blockID uuid.UUID, tenantID string) error
	CompactedBlockMeta(blockID uuid.UUID, tenantID string) (*CompactedBlockMeta, error)
}
