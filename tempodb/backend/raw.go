package backend

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path"

	"github.com/google/uuid"

	tempo_io "github.com/grafana/tempo/pkg/io"
)

const (
	MetaName          = "meta.json"
	CompactedMetaName = "meta.compacted.json"
	TenantIndexName   = "index.json.gz"
	// File name for the cluster seed file.
	ClusterSeedFileName = "tempo_cluster_seed.json"
)

// KeyPath is an ordered set of strings that govern where data is read/written from the backend
type KeyPath []string

// RawWriter is a collection of methods to write data to tempodb backends
type RawWriter interface {
	// Write is for in memory data. shouldCache specifies whether or not caching should be attempted.
	Write(ctx context.Context, name string, keypath KeyPath, data io.Reader, size int64, shouldCache bool) error
	// Append starts or continues an Append job. Pass nil to AppendTracker to start a job.
	Append(ctx context.Context, name string, keypath KeyPath, tracker AppendTracker, buffer []byte) (AppendTracker, error)
	// CloseAppend closes any resources associated with the AppendTracker.
	CloseAppend(ctx context.Context, tracker AppendTracker) error
	// Delete deletes a file.
	Delete(ctx context.Context, name string, keypath KeyPath, shouldCache bool) error
}

// RawReader is a collection of methods to read data from tempodb backends
type RawReader interface {
	// List returns all objects one level beneath the provided keypath
	List(ctx context.Context, keypath KeyPath) ([]string, error)
	// Read is for streaming entire objects from the backend.  There will be an attempt to retrieve this from cache if shouldCache is true.
	Read(ctx context.Context, name string, keyPath KeyPath, shouldCache bool) (io.ReadCloser, int64, error)
	// ReadRange is for reading parts of large objects from the backend.
	// There will be an attempt to retrieve this from cache if shouldCache is true. Cache key will be tenantID:blockID:offset:bufferLength
	ReadRange(ctx context.Context, name string, keypath KeyPath, offset uint64, buffer []byte, shouldCache bool) error
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

func (w *writer) Write(ctx context.Context, name string, blockID uuid.UUID, tenantID string, buffer []byte, shouldCache bool) error {
	return w.w.Write(ctx, name, KeyPathForBlock(blockID, tenantID), bytes.NewReader(buffer), int64(len(buffer)), shouldCache)
}

func (w *writer) StreamWriter(ctx context.Context, name string, blockID uuid.UUID, tenantID string, data io.Reader, size int64) error {
	return w.w.Write(ctx, name, KeyPathForBlock(blockID, tenantID), data, size, false)
}

func (w *writer) WriteBlockMeta(ctx context.Context, meta *BlockMeta) error {
	blockID := meta.BlockID
	tenantID := meta.TenantID

	bMeta, err := json.Marshal(meta)
	if err != nil {
		return err
	}

	return w.w.Write(ctx, MetaName, KeyPathForBlock(blockID, tenantID), bytes.NewReader(bMeta), int64(len(bMeta)), false)
}

func (w *writer) Append(ctx context.Context, name string, blockID uuid.UUID, tenantID string, tracker AppendTracker, buffer []byte) (AppendTracker, error) {
	return w.w.Append(ctx, name, KeyPathForBlock(blockID, tenantID), tracker, buffer)
}

func (w *writer) CloseAppend(ctx context.Context, tracker AppendTracker) error {
	return w.w.CloseAppend(ctx, tracker)
}

func (w *writer) WriteTenantIndex(ctx context.Context, tenantID string, meta []*BlockMeta, compactedMeta []*CompactedBlockMeta) error {
	// If meta and compactedMeta are empty, call delete the tenant index.
	if len(meta) == 0 && len(compactedMeta) == 0 {
		// Skip returning an error when the object is already deleted.
		err := w.w.Delete(ctx, TenantIndexName, []string{tenantID}, false)
		if err != nil && !errors.Is(err, ErrDoesNotExist) {
			return err
		}
		return nil
	}

	b := newTenantIndex(meta, compactedMeta)

	indexBytes, err := b.marshal()
	if err != nil {
		return err
	}

	err = w.w.Write(ctx, TenantIndexName, KeyPath([]string{tenantID}), bytes.NewReader(indexBytes), int64(len(indexBytes)), false)
	if err != nil {
		return err
	}

	return nil
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

func (r *reader) Read(ctx context.Context, name string, blockID uuid.UUID, tenantID string, shouldCache bool) ([]byte, error) {
	objReader, size, err := r.r.Read(ctx, name, KeyPathForBlock(blockID, tenantID), shouldCache)
	if err != nil {
		return nil, err
	}
	defer objReader.Close()
	return tempo_io.ReadAllWithEstimate(objReader, size)
}

func (r *reader) StreamReader(ctx context.Context, name string, blockID uuid.UUID, tenantID string) (io.ReadCloser, int64, error) {
	return r.r.Read(ctx, name, KeyPathForBlock(blockID, tenantID), false)
}

func (r *reader) ReadRange(ctx context.Context, name string, blockID uuid.UUID, tenantID string, offset uint64, buffer []byte, shouldCache bool) error {
	return r.r.ReadRange(ctx, name, KeyPathForBlock(blockID, tenantID), offset, buffer, shouldCache)
}

func (r *reader) Tenants(ctx context.Context) ([]string, error) {
	list, err := r.r.List(ctx, nil)

	// this filter is added to fix a GCS usage stats issue that would result in ""
	var filteredList []string
	for _, tenant := range list {
		if tenant != "" && tenant != ClusterSeedFileName {
			filteredList = append(filteredList, tenant)
		}
	}

	return filteredList, err
}

func (r *reader) Blocks(ctx context.Context, tenantID string) ([]uuid.UUID, error) {
	objects, err := r.r.List(ctx, KeyPath{tenantID})
	if err != nil {
		return nil, err
	}

	// translate everything to UUIDs, if we see a bucket index we can skip that
	blockIDs := make([]uuid.UUID, 0, len(objects))
	for _, id := range objects {
		// TODO: this line exists due to behavior differences in backends: https://github.com/grafana/tempo/issues/880
		// revisit once #880 is resolved.
		if id == TenantIndexName || id == "" {
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
	reader, size, err := r.r.Read(ctx, MetaName, KeyPathForBlock(blockID, tenantID), false)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	bytes, err := tempo_io.ReadAllWithEstimate(reader, size)
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

func (r *reader) TenantIndex(ctx context.Context, tenantID string) (*TenantIndex, error) {
	reader, size, err := r.r.Read(ctx, TenantIndexName, KeyPath([]string{tenantID}), false)
	if err != nil {
		return nil, err
	}

	defer reader.Close()

	bytes, err := tempo_io.ReadAllWithEstimate(reader, size)
	if err != nil {
		return nil, err
	}

	i := &TenantIndex{}
	err = i.unmarshal(bytes)
	if err != nil {
		return nil, err
	}

	return i, nil
}

func (r *reader) Shutdown() {
	r.r.Shutdown()
}

// KeyPathForBlock returns a correctly ordered keypath given a block id and tenantid
func KeyPathForBlock(blockID uuid.UUID, tenantID string) KeyPath {
	return []string{tenantID, blockID.String()}
}

// ObjectFileName returns a unique identifier for an object in object storage given its name and keypath
func ObjectFileName(keypath KeyPath, name string) string {
	return path.Join(path.Join(keypath...), name)
}

// KeyPathWithPrefix returns a keypath with a prefix
func KeyPathWithPrefix(keypath KeyPath, prefix string) KeyPath {
	if len(prefix) == 0 {
		return keypath
	}

	return append([]string{prefix}, keypath...)
}

// MetaFileName returns the object name for the block meta given a block id and tenantid
func MetaFileName(blockID uuid.UUID, tenantID, prefix string) string {
	return path.Join(prefix, tenantID, blockID.String(), MetaName)
}

// CompactedMetaFileName returns the object name for the compacted block meta given a block id and tenantid
func CompactedMetaFileName(blockID uuid.UUID, tenantID, prefix string) string {
	return path.Join(prefix, tenantID, blockID.String(), CompactedMetaName)
}

// RootPath returns the root path for a block given a block id and tenantid
func RootPath(blockID uuid.UUID, tenantID, prefix string) string {
	return path.Join(prefix, tenantID, blockID.String())
}
