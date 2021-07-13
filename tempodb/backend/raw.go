package backend

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path"

	"github.com/google/uuid"
)

const (
	MetaName          = "meta.json"
	CompactedMetaName = "meta.compacted.json"
	BlockIndexName    = "blockindex.json.gz"
)

// KeyPath is an ordered set of strings that govern where data is read/written from the backend
type KeyPath []string

// RawWriter is a collection of methods to write data to tempodb backends
type RawWriter interface {
	// Write is for in memory data.  It is expected that this data will be cached.
	Write(ctx context.Context, name string, keypath KeyPath, buffer []byte) error
	// StreamWriter is for larger data payloads streamed through an io.Reader.  It is expected this will _not_ be cached.
	StreamWriter(ctx context.Context, name string, keypath KeyPath, data io.Reader, size int64) error
	// Append starts or continues an Append job. Pass nil to AppendTracker to start a job.
	Append(ctx context.Context, name string, keypath KeyPath, tracker AppendTracker, buffer []byte) (AppendTracker, error)
	// Closes any resources associated with the AppendTracker
	CloseAppend(ctx context.Context, tracker AppendTracker) error
}

// RawReader is a collection of methods to read data from tempodb backends
type RawReader interface {
	// Read is for reading entire objects from the backend.  It is expected that there will be an attempt to retrieve this from cache
	Read(ctx context.Context, name string, keypath KeyPath) ([]byte, error)
	// StreamReader is for streaming entire objects from the backend.  It is expected this will _not_ be cached.
	StreamReader(ctx context.Context, name string, keypath KeyPath) (io.ReadCloser, int64, error)
	// ReadRange is for reading parts of large objects from the backend.  It is expected this will _not_ be cached.
	ReadRange(ctx context.Context, name string, keypath KeyPath, offset uint64, buffer []byte) error
	// List returns all objects one level beneath the provided keypath
	List(ctx context.Context, keypath KeyPath) ([]string, error)
	// Shutdown must be called when the Reader is finished and cleans up any associated resources.
	Shutdown()
}

type writer struct {
	w RawWriter
}

// NewWriter returns an object that implements Writer and bridges to a RawWriter
func NewWriter(w RawWriter) Writer {
	return &writer{
		w: w,
	}
}

func (w *writer) Write(ctx context.Context, name string, blockID uuid.UUID, tenantID string, buffer []byte) error {
	return w.w.Write(ctx, name, KeyPathForBlock(blockID, tenantID), buffer)
}

func (w *writer) StreamWriter(ctx context.Context, name string, blockID uuid.UUID, tenantID string, data io.Reader, size int64) error {
	return w.w.StreamWriter(ctx, name, KeyPathForBlock(blockID, tenantID), data, size)
}

func (w *writer) WriteBlockMeta(ctx context.Context, meta *BlockMeta) error {
	blockID := meta.BlockID
	tenantID := meta.TenantID

	bMeta, err := json.Marshal(meta)
	if err != nil {
		return err
	}

	return w.w.Write(ctx, MetaName, KeyPathForBlock(blockID, tenantID), bMeta)
}

func (w *writer) Append(ctx context.Context, name string, blockID uuid.UUID, tenantID string, tracker AppendTracker, buffer []byte) (AppendTracker, error) {
	return w.w.Append(ctx, name, KeyPathForBlock(blockID, tenantID), tracker, buffer)
}

func (w *writer) CloseAppend(ctx context.Context, tracker AppendTracker) error {
	return w.w.CloseAppend(ctx, tracker)
}

type reader struct {
	r RawReader
}

// NewReader returns an object that implements Reader and bridges to a RawReader
func NewReader(r RawReader) Reader {
	return &reader{
		r: r,
	}
}

func (r *reader) Read(ctx context.Context, name string, blockID uuid.UUID, tenantID string) ([]byte, error) {
	return r.r.Read(ctx, name, KeyPathForBlock(blockID, tenantID))
}

func (r *reader) StreamReader(ctx context.Context, name string, blockID uuid.UUID, tenantID string) (io.ReadCloser, int64, error) {
	return r.r.StreamReader(ctx, name, KeyPathForBlock(blockID, tenantID))
}

func (r *reader) ReadRange(ctx context.Context, name string, blockID uuid.UUID, tenantID string, offset uint64, buffer []byte) error {
	return r.r.ReadRange(ctx, name, KeyPathForBlock(blockID, tenantID), offset, buffer)
}

func (r *reader) Tenants(ctx context.Context) ([]string, error) {
	return r.r.List(ctx, nil)
}

func (r *reader) Blocks(ctx context.Context, tenantID string) ([]uuid.UUID, error) {
	objects, err := r.r.List(ctx, KeyPath{tenantID})
	if err != nil {
		return nil, err
	}

	// translate everything to UUIDs, if we see a bucket index we can skip that
	blockIDs := make([]uuid.UUID, 0, len(objects))
	for _, id := range objects {
		if id == BlockIndexName {
			continue
		}
		uuid, err := uuid.Parse(id)
		if err != nil {
			return nil, fmt.Errorf("failed to parse %s: %w", id, err)
		}
		blockIDs = append(blockIDs, uuid)
	}

	return blockIDs, nil
}

func (r *reader) BlockMeta(ctx context.Context, blockID uuid.UUID, tenantID string) (*BlockMeta, error) {
	bytes, err := r.r.Read(ctx, MetaName, KeyPathForBlock(blockID, tenantID))
	if err != nil {
		return nil, err
	}

	out := &BlockMeta{}
	err = json.Unmarshal(bytes, out)
	if err != nil {
		return nil, err
	}

	return out, nil
}

func (r *reader) Shutdown() {
	r.r.Shutdown()
}

// KeyPathForBlock returns a correctly ordered keypath given a block id and tenantid
// nolint:interfacer
func KeyPathForBlock(blockID uuid.UUID, tenantID string) KeyPath {
	return []string{tenantID, blockID.String()}
}

// ObjectFileName returns a unique identifier for an object in object storage given its name and keypath
func ObjectFileName(keypath KeyPath, name string) string {
	return path.Join(path.Join(keypath...), name)
}

// MetaFileName returns the object name for the block meta given a block id and tenantid
func MetaFileName(blockID uuid.UUID, tenantID string) string {
	return path.Join(RootPath(blockID, tenantID), MetaName)
}

// CompactedMetaFileName returns the object name for the compacted block meta given a block id and tenantid
func CompactedMetaFileName(blockID uuid.UUID, tenantID string) string {
	return path.Join(RootPath(blockID, tenantID), CompactedMetaName)
}

// RootPath returns the root path for a block given a block id and tenantid
// nolint:interfacer
func RootPath(blockID uuid.UUID, tenantID string) string {
	return path.Join(tenantID, blockID.String())
}
