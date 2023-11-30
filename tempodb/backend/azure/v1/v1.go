package v1

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"

	blob "github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/go-kit/log/level"
	"github.com/google/uuid"
	"github.com/opentracing/opentracing-go"

	"github.com/grafana/tempo/pkg/util/log"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/azure/config"
)

const (
	// dir represents the char separator used by the blob virtual directory structure
	dir = "/"
	// max parallelism on uploads
	maxParallelism = 3
)

type V1 struct {
	cfg                *config.Config
	containerURL       blob.ContainerURL
	hedgedContainerURL blob.ContainerURL
}

var (
	_ backend.RawReader             = (*V1)(nil)
	_ backend.RawWriter             = (*V1)(nil)
	_ backend.Compactor             = (*V1)(nil)
	_ backend.VersionedReaderWriter = (*V1)(nil)
)

type appendTracker struct {
	Name string
}

func New(cfg *config.Config, confirm bool) (*V1, error) {
	ctx := context.Background()

	container, err := GetContainer(ctx, cfg, false)
	if err != nil {
		return nil, fmt.Errorf("getting storage container: %w", err)
	}

	hedgedContainer, err := GetContainer(ctx, cfg, true)
	if err != nil {
		return nil, fmt.Errorf("getting hedged storage container: %w", err)
	}

	if confirm {
		// Getting container properties to check if container exists
		_, err = container.GetProperties(ctx, blob.LeaseAccessConditions{})
		if err != nil {
			return nil, fmt.Errorf("failed to GetProperties: %w", err)
		}
	}

	rw := &V1{
		cfg:                cfg,
		containerURL:       container,
		hedgedContainerURL: hedgedContainer,
	}

	return rw, nil
}

// Write implements backend.Writer
func (rw *V1) Write(ctx context.Context, name string, keypath backend.KeyPath, data io.Reader, _ int64, _ *backend.CacheInfo) error {
	keypath = backend.KeyPathWithPrefix(keypath, rw.cfg.Prefix)

	span, derivedCtx := opentracing.StartSpanFromContext(ctx, "azure.Write")
	defer span.Finish()

	return rw.writer(derivedCtx, bufio.NewReader(data), backend.ObjectFileName(keypath, name))
}

// Append implements backend.Writer
func (rw *V1) Append(ctx context.Context, name string, keypath backend.KeyPath, tracker backend.AppendTracker, buffer []byte) (backend.AppendTracker, error) {
	keypath = backend.KeyPathWithPrefix(keypath, rw.cfg.Prefix)
	var a appendTracker
	if tracker == nil {
		a.Name = backend.ObjectFileName(keypath, name)

		err := rw.writeAll(ctx, a.Name, buffer)
		if err != nil {
			return nil, err
		}
	} else {
		a = tracker.(appendTracker)

		err := rw.append(ctx, buffer, a.Name)
		if err != nil {
			return nil, err
		}
	}

	return a, nil
}

// CloseAppend implements backend.Writer
func (rw *V1) CloseAppend(context.Context, backend.AppendTracker) error {
	return nil
}

func (rw *V1) Delete(ctx context.Context, name string, keypath backend.KeyPath, _ *backend.CacheInfo) error {
	blobURL, err := GetBlobURL(ctx, rw.cfg, backend.ObjectFileName(keypath, name))
	if err != nil {
		return fmt.Errorf("cannot get Azure blob URL, name: %s: %w", backend.ObjectFileName(keypath, name), err)
	}

	if _, err = blobURL.Delete(ctx, blob.DeleteSnapshotsOptionInclude, blob.BlobAccessConditions{}); err != nil {
		return readError(err)
	}
	return nil
}

// List implements backend.Reader
func (rw *V1) List(ctx context.Context, keypath backend.KeyPath) ([]string, error) {
	keypath = backend.KeyPathWithPrefix(keypath, rw.cfg.Prefix)

	marker := blob.Marker{}
	prefix := path.Join(keypath...)

	if len(prefix) > 0 {
		prefix = prefix + dir
	}

	objects := make([]string, 0)
	for {
		list, err := rw.containerURL.ListBlobsHierarchySegment(ctx, marker, dir, blob.ListBlobsSegmentOptions{
			Prefix:  prefix,
			Details: blob.BlobListingDetails{},
		})
		if err != nil {
			return objects, fmt.Errorf("iterating tenants: %w", err)
		}
		marker = list.NextMarker

		for _, blob := range list.Segment.BlobPrefixes {
			objects = append(objects, strings.TrimPrefix(strings.TrimSuffix(blob.Name, dir), prefix))
		}

		// Continue iterating if we are not done.
		if !marker.NotDone() {
			break
		}
	}
	return objects, nil
}

// ListBlocks implements backend.Reader
func (rw *V1) ListBlocks(ctx context.Context, tenant string) ([]uuid.UUID, []uuid.UUID, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "V1.ListBlocks")
	defer span.Finish()

	var (
		blockIDs          = make([]uuid.UUID, 0, 1000)
		compactedBlockIDs = make([]uuid.UUID, 0, 1000)
		keypath           = backend.KeyPathWithPrefix(backend.KeyPath{tenant}, rw.cfg.Prefix)
		marker            = blob.Marker{}
		parts             []string
		id                uuid.UUID
	)

	prefix := path.Join(keypath...)
	if len(prefix) > 0 {
		prefix += dir
	}

	for {
		res, err := rw.containerURL.ListBlobsFlatSegment(ctx, marker, blob.ListBlobsSegmentOptions{
			Prefix:  prefix,
			Details: blob.BlobListingDetails{},
		})
		if err != nil {
			return nil, nil, fmt.Errorf("iterating objects: %w", err)
		}
		marker = res.NextMarker

		for _, blob := range res.Segment.BlobItems {
			obj := strings.TrimPrefix(strings.TrimSuffix(blob.Name, dir), prefix)
			parts = strings.Split(obj, "/")

			// ie: <blockID>/meta.json
			if len(parts) != 2 {
				continue
			}

			switch parts[1] {
			case backend.MetaName, backend.CompactedMetaName:
			default:
				continue
			}

			id, err = uuid.Parse(parts[0])
			if err != nil {
				return nil, nil, err
			}

			switch parts[1] {
			case backend.MetaName:
				blockIDs = append(blockIDs, id)
			case backend.CompactedMetaName:
				compactedBlockIDs = append(compactedBlockIDs, id)
			}

		}

		// Continue iterating if we are not done.
		if !marker.NotDone() {
			break
		}
	}
	return blockIDs, compactedBlockIDs, nil
}

// Read implements backend.Reader
func (rw *V1) Read(ctx context.Context, name string, keypath backend.KeyPath, _ *backend.CacheInfo) (io.ReadCloser, int64, error) {
	keypath = backend.KeyPathWithPrefix(keypath, rw.cfg.Prefix)

	span, derivedCtx := opentracing.StartSpanFromContext(ctx, "azure.Read")
	defer span.Finish()

	object := backend.ObjectFileName(keypath, name)
	b, _, err := rw.readAll(derivedCtx, object)
	if err != nil {
		return nil, 0, readError(err)
	}

	return io.NopCloser(bytes.NewReader(b)), int64(len(b)), nil
}

// ReadRange implements backend.Reader
func (rw *V1) ReadRange(ctx context.Context, name string, keypath backend.KeyPath, offset uint64, buffer []byte, _ *backend.CacheInfo) error {
	keypath = backend.KeyPathWithPrefix(keypath, rw.cfg.Prefix)

	span, derivedCtx := opentracing.StartSpanFromContext(ctx, "azure.ReadRange", opentracing.Tags{
		"len":    len(buffer),
		"offset": offset,
	})
	defer span.Finish()

	object := backend.ObjectFileName(keypath, name)
	err := rw.readRange(derivedCtx, object, int64(offset), buffer)
	if err != nil {
		return readError(err)
	}

	return nil
}

// Shutdown implements backend.Reader
func (rw *V1) Shutdown() {
}

func (rw *V1) WriteVersioned(ctx context.Context, name string, keypath backend.KeyPath, data io.Reader, version backend.Version) (backend.Version, error) {
	// TODO use conditional if-match API
	_, currentVersion, err := rw.ReadVersioned(ctx, name, keypath)
	if err != nil && !errors.Is(err, backend.ErrDoesNotExist) {
		return "", err
	}

	level.Info(log.Logger).Log("msg", "WriteVersioned - fetching data", "currentVersion", currentVersion, "err", err, "version", version)

	// object does not exist - supplied version must be "0"
	if errors.Is(err, backend.ErrDoesNotExist) && version != backend.VersionNew {
		return "", backend.ErrVersionDoesNotMatch
	}
	if !errors.Is(err, backend.ErrDoesNotExist) && version != currentVersion {
		return "", backend.ErrVersionDoesNotMatch
	}

	err = rw.Write(ctx, name, keypath, data, -1, nil)
	if err != nil {
		return "", err
	}

	_, currentVersion, err = rw.ReadVersioned(ctx, name, keypath)
	return currentVersion, err
}

func (rw *V1) DeleteVersioned(ctx context.Context, name string, keypath backend.KeyPath, version backend.Version) error {
	// TODO use conditional if-match API
	_, currentVersion, err := rw.ReadVersioned(ctx, name, keypath)
	if err != nil && errors.Is(err, backend.ErrDoesNotExist) {
		return err
	}
	if !errors.Is(err, backend.ErrDoesNotExist) && currentVersion != version {
		return backend.ErrVersionDoesNotMatch
	}

	return rw.Delete(ctx, name, keypath, nil)
}

func (rw *V1) ReadVersioned(ctx context.Context, name string, keypath backend.KeyPath) (io.ReadCloser, backend.Version, error) {
	keypath = backend.KeyPathWithPrefix(keypath, rw.cfg.Prefix)

	span, derivedCtx := opentracing.StartSpanFromContext(ctx, "azure.ReadVersioned")
	defer span.Finish()

	object := backend.ObjectFileName(keypath, name)
	b, etag, err := rw.readAll(derivedCtx, object)
	if err != nil {
		return nil, "", readError(err)
	}

	return io.NopCloser(bytes.NewReader(b)), backend.Version(etag), nil
}

func (rw *V1) writeAll(ctx context.Context, name string, b []byte) error {
	err := rw.writer(ctx, bytes.NewReader(b), name)
	if err != nil {
		return err
	}

	return nil
}

func (rw *V1) append(ctx context.Context, src []byte, name string) error {
	appendBlobURL := rw.containerURL.NewBlockBlobURL(name)

	// These helper functions convert a binary block ID to a base-64 string and vice versa
	// NOTE: The blockID must be <= 64 bytes and ALL blockIDs for the block must be the same length
	blockIDBinaryToBase64 := func(blockID []byte) string { return base64.StdEncoding.EncodeToString(blockID) }

	blockIDIntToBase64 := func(blockID int) string {
		binaryBlockID := (&[64]byte{})[:]
		binary.LittleEndian.PutUint32(binaryBlockID, uint32(blockID))
		return blockIDBinaryToBase64(binaryBlockID)
	}

	l, err := appendBlobURL.GetBlockList(ctx, blob.BlockListAll, blob.LeaseAccessConditions{})
	if err != nil {
		return err
	}

	// generate the next block id
	id := blockIDIntToBase64(len(l.CommittedBlocks) + 1)

	_, err = appendBlobURL.StageBlock(ctx, id, bytes.NewReader(src), blob.LeaseAccessConditions{}, nil, blob.ClientProvidedKeyOptions{})
	if err != nil {
		return err
	}

	base64BlockIDs := make([]string, len(l.CommittedBlocks)+1)
	for i := 0; i < len(l.CommittedBlocks); i++ {
		base64BlockIDs[i] = l.CommittedBlocks[i].Name
	}

	base64BlockIDs[len(l.CommittedBlocks)] = id

	// After all the blocks are uploaded, atomically commit them to the blob.
	_, err = appendBlobURL.CommitBlockList(ctx, base64BlockIDs, blob.BlobHTTPHeaders{}, blob.Metadata{}, blob.BlobAccessConditions{}, blob.DefaultAccessTier, blob.BlobTagsMap{}, blob.ClientProvidedKeyOptions{}, blob.ImmutabilityPolicyOptions{})
	if err != nil {
		return err
	}
	return nil
}

func (rw *V1) writer(ctx context.Context, src io.Reader, name string) error {
	blobURL := rw.containerURL.NewBlockBlobURL(name)

	if _, err := blob.UploadStreamToBlockBlob(ctx, src, blobURL,
		blob.UploadStreamToBlockBlobOptions{
			BufferSize: rw.cfg.BufferSize,
			MaxBuffers: rw.cfg.MaxBuffers,
		},
	); err != nil {
		return fmt.Errorf("cannot upload blob, name: %s: %w", name, err)
	}
	return nil
}

func (rw *V1) readRange(ctx context.Context, name string, offset int64, destBuffer []byte) error {
	blobURL := rw.hedgedContainerURL.NewBlockBlobURL(name)

	var props *blob.BlobGetPropertiesResponse
	props, err := blobURL.GetProperties(ctx, blob.BlobAccessConditions{}, blob.ClientProvidedKeyOptions{})
	if err != nil {
		return err
	}

	length := int64(len(destBuffer))
	var size int64

	if length > 0 && length <= props.ContentLength()-offset {
		size = length
	} else {
		size = props.ContentLength() - offset
	}

	if err := blob.DownloadBlobToBuffer(context.Background(), blobURL.BlobURL, offset, size,
		destBuffer, blob.DownloadFromBlobOptions{
			BlockSize:   blob.BlobDefaultDownloadBlockSize,
			Parallelism: maxParallelism,
			Progress:    nil,
			RetryReaderOptionsPerBlock: blob.RetryReaderOptions{
				MaxRetryRequests: maxRetries,
			},
		},
	); err != nil {
		return err
	}

	_, err = bytes.NewReader(destBuffer).Read(destBuffer)
	if err != nil {
		return err
	}

	return nil
}

func (rw *V1) readAll(ctx context.Context, name string) ([]byte, blob.ETag, error) {
	blobURL := rw.hedgedContainerURL.NewBlockBlobURL(name)

	var props *blob.BlobGetPropertiesResponse
	props, err := blobURL.GetProperties(ctx, blob.BlobAccessConditions{}, blob.ClientProvidedKeyOptions{})
	if err != nil {
		return nil, "", err
	}

	destBuffer := make([]byte, props.ContentLength())

	if err := blob.DownloadBlobToBuffer(context.Background(), blobURL.BlobURL, 0, props.ContentLength(),
		destBuffer, blob.DownloadFromBlobOptions{
			BlockSize:   blob.BlobDefaultDownloadBlockSize,
			Parallelism: uint16(maxParallelism),
			Progress:    nil,
			RetryReaderOptionsPerBlock: blob.RetryReaderOptions{
				MaxRetryRequests: maxRetries,
			},
		},
	); err != nil {
		return nil, "", err
	}

	return destBuffer, props.ETag(), nil
}

func readError(err error) error {
	var storageError blob.StorageError
	errors.As(err, &storageError)

	if storageError == nil {
		return fmt.Errorf("reading storage container: %w", err)
	}
	if storageError.ServiceCode() == blob.ServiceCodeBlobNotFound {
		return backend.ErrDoesNotExist
	}

	if err != nil {
		return fmt.Errorf("reading Azure blob container: %w", storageError)
	}
	return nil
}
