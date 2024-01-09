package backend

import (
	"context"
	"fmt"
	"io"

	"github.com/google/uuid"
	"github.com/grafana/tempo/pkg/cache"
)

const (
	Local = "local"
	GCS   = "gcs"
	S3    = "s3"
	Azure = "azure"
)

var (
	ErrDoesNotExist  = fmt.Errorf("does not exist")
	ErrEmptyTenantID = fmt.Errorf("empty tenant id")
	ErrEmptyBlockID  = fmt.Errorf("empty block id")
	ErrBadSeedFile   = fmt.Errorf("bad seed file")

	GlobalMaxBlockID = uuid.MustParse("ffffffff-ffff-ffff-ffff-ffffffffffff")
)

type CacheInfo struct {
	Meta *BlockMeta
	Role cache.Role
}

// AppendTracker is an empty interface usable by the backend to track a long running append operation
type AppendTracker interface{}

// Writer is a collection of methods to write data to tempodb backends
type Writer interface {
	// Write is for in memory data. cacheInfo contains information to make a caching decision.
	Write(ctx context.Context, name string, blockID uuid.UUID, tenantID string, buffer []byte, cacheInfo *CacheInfo) error
	// StreamWriter is for larger data payloads streamed through an io.Reader.  It is expected this will _not_ be cached.
	StreamWriter(ctx context.Context, name string, blockID uuid.UUID, tenantID string, data io.Reader, size int64) error
	// WriteBlockMeta writes a block meta to its blocks
	WriteBlockMeta(ctx context.Context, meta *BlockMeta) error
	// Append starts or continues an Append job. Pass nil to AppendTracker to start a job.
	Append(ctx context.Context, name string, blockID uuid.UUID, tenantID string, tracker AppendTracker, buffer []byte) (AppendTracker, error)
	// CloseAppend closes any resources associated with the AppendTracker
	CloseAppend(ctx context.Context, tracker AppendTracker) error
	// WriteTenantIndex writes the two meta slices as a tenant index
	WriteTenantIndex(ctx context.Context, tenantID string, meta []*BlockMeta, compactedMeta []*CompactedBlockMeta) error
}

// Reader is a collection of methods to read data from tempodb backends
type Reader interface {
	// Read is for reading entire objects from the backend. cacheInfo contains information to make a caching decision
	Read(ctx context.Context, name string, blockID uuid.UUID, tenantID string, cacheInfo *CacheInfo) ([]byte, error)
	// StreamReader is for streaming entire objects from the backend.  It is expected this will _not_ be cached.
	StreamReader(ctx context.Context, name string, blockID uuid.UUID, tenantID string) (io.ReadCloser, int64, error)
	// ReadRange is for reading parts of large objects from the backend. cacheInfo contains information to make a caching decision
	ReadRange(ctx context.Context, name string, blockID uuid.UUID, tenantID string, offset uint64, buffer []byte, cacheInfo *CacheInfo) error
	// Tenants returns a list of all tenants in a backend
	Tenants(ctx context.Context) ([]string, error)
	// Blocks returns the blockIDs, compactedBlockIDs and an error from the backend.
	Blocks(ctx context.Context, tenantID string) (blockIDs []uuid.UUID, compactedBlockIDs []uuid.UUID, err error)
	// BlockMeta returns the blockmeta given a block and tenant id
	BlockMeta(ctx context.Context, blockID uuid.UUID, tenantID string) (*BlockMeta, error)
	// TenantIndex returns lists of all metas given a tenant
	TenantIndex(ctx context.Context, tenantID string) (*TenantIndex, error)
	// Shutdown shuts...down?
	Shutdown()
}

// Compactor is a collection of methods to interact with compacted elements of a tempodb block
type Compactor interface {
	// MarkBlockCompacted marks a block as compacted. Call this after a block has been successfully compacted to a new block
	MarkBlockCompacted(blockID uuid.UUID, tenantID string) error
	// ClearBlock removes a block from the backend
	ClearBlock(blockID uuid.UUID, tenantID string) error
	// CompactedBlockMeta returns the compacted blockmeta given a block and tenant id
	CompactedBlockMeta(blockID uuid.UUID, tenantID string) (*CompactedBlockMeta, error)
}
