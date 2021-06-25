package backend

import (
	"context"
	"fmt"
	"io"

	"github.com/google/uuid"
)

var (
	ErrMetaDoesNotExist = fmt.Errorf("meta does not exist") // jpe rename to just ErrDoesNotExist
	ErrEmptyTenantID    = fmt.Errorf("empty tenant id")     // jpe do we need these?
	ErrEmptyBlockID     = fmt.Errorf("empty block id")
)

// AppendTracker is an empty interface usable by the backend to track a long running append operation
type AppendTracker interface{}

// KeyPath is an ordered set of strings that govern where data is read/written from the backend
type KeyPath []string

// Writer is a collection of methods to write data to tempodb backends
type Writer interface {
	// Write is for in memory data.  It is expected that this data will be cached.
	Write(ctx context.Context, name string, keypath KeyPath, buffer []byte) error
	// WriteReader is for larger data payloads streamed through an io.Reader.  It is expected this will _not_ be cached.
	WriteReader(ctx context.Context, name string, keypath KeyPath, data io.Reader, size int64) error
	// Append starts or continues an append job. Pass nil to AppendTracker to start a job.
	Append(ctx context.Context, name string, keypath KeyPath, tracker AppendTracker, buffer []byte) (AppendTracker, error)
	// Closes any resources associated with the AppendTracker
	CloseAppend(ctx context.Context, tracker AppendTracker) error
}

// Reader is a collection of methods to read data from tempodb backends
type Reader interface {
	// Reader is for reading entire objects from the backend.  It is expected that there will be an attempt to retrieve this from cache
	Read(ctx context.Context, name string, keypath KeyPath) ([]byte, error)
	// ReadReader is for streaming entire objects from the backend.  It is expected this will _not_ be cached.
	ReadReader(ctx context.Context, name string, keypath KeyPath) (io.ReadCloser, int64, error)
	// ReadRange is for reading parts of large objects from the backend.  It is expected this will _not_ be cached.
	ReadRange(ctx context.Context, name string, keypath KeyPath, offset uint64, buffer []byte) error
	// List returns all objects one level beneath the provided keypath
	List(ctx context.Context, keypath KeyPath) ([]string, error)
	// Shutdown must be called when the Reader is finished and cleans up any associated resources.
	Shutdown()
}

// jpe - add context
// Compactor is a collection of methods to interact with compacted elements of a tempodb block
type Compactor interface {
	// Marks a block compacted by renaming meta.json => meta.compacted.json
	MarkBlockCompacted(blockID uuid.UUID, tenantID string) error
	// Removes a block
	ClearBlock(blockID uuid.UUID, tenantID string) error
	// CompactedBlockMeta reuturns the compacted block meta for a block if available
	CompactedBlockMeta(blockID uuid.UUID, tenantID string) (*CompactedBlockMeta, error)
}
