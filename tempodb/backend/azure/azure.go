package azure

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"path"
	"strings"

	blob "github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/go-kit/log/level"
	"github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"

	"github.com/grafana/tempo/pkg/util/log"
	"github.com/grafana/tempo/tempodb/backend"
)

const (
	// dir represents the char separator used by the blob virtual directory structure
	dir = "/"
	// max parallelism on uploads
	maxParallelism = 3
)

type readerWriter struct {
	cfg                *Config
	containerURL       blob.ContainerURL
	hedgedContainerURL blob.ContainerURL
}

var (
	_ backend.RawReader             = (*readerWriter)(nil)
	_ backend.RawWriter             = (*readerWriter)(nil)
	_ backend.Compactor             = (*readerWriter)(nil)
	_ backend.VersionedReaderWriter = (*readerWriter)(nil)
)

type appendTracker struct {
	Name string
}

// NewNoConfirm gets the Azure blob container without testing it
func NewNoConfirm(cfg *Config) (backend.RawReader, backend.RawWriter, backend.Compactor, error) {
	rw, err := internalNew(cfg, false)
	return rw, rw, rw, err
}

// New gets the Azure blob container
func New(cfg *Config) (backend.RawReader, backend.RawWriter, backend.Compactor, error) {
	rw, err := internalNew(cfg, true)
	return rw, rw, rw, err
}

// NewVersionedReaderWriter creates a client to perform versioned requests. Note that write requests are
// best-effort for now. We need to update the SDK to make use of the precondition headers.
// https://github.com/grafana/tempo/issues/2705
func NewVersionedReaderWriter(cfg *Config) (backend.VersionedReaderWriter, error) {
	return internalNew(cfg, true)
}

func internalNew(cfg *Config, confirm bool) (*readerWriter, error) {
	ctx := context.Background()

	container, err := GetContainer(ctx, cfg, false)
	if err != nil {
		return nil, errors.Wrap(err, "getting storage container")
	}

	hedgedContainer, err := GetContainer(ctx, cfg, true)
	if err != nil {
		return nil, errors.Wrap(err, "getting hedged storage container")
	}

	if confirm {
		// Getting container properties to check if container exists
		_, err = container.GetProperties(ctx, blob.LeaseAccessConditions{})
		if err != nil {
			return nil, fmt.Errorf("failed to GetProperties: %w", err)
		}
	}

	rw := &readerWriter{
		cfg:                cfg,
		containerURL:       container,
		hedgedContainerURL: hedgedContainer,
	}

	return rw, nil
}

// Write implements backend.Writer
func (rw *readerWriter) Write(ctx context.Context, name string, keypath backend.KeyPath, data io.Reader, _ int64, _ bool) error {
	keypath = backend.KeyPathWithPrefix(keypath, rw.cfg.Prefix)

	span, derivedCtx := opentracing.StartSpanFromContext(ctx, "azure.Write")
	defer span.Finish()

	return rw.writer(derivedCtx, bufio.NewReader(data), backend.ObjectFileName(keypath, name))
}

// Append implements backend.Writer
func (rw *readerWriter) Append(ctx context.Context, name string, keypath backend.KeyPath, tracker backend.AppendTracker, buffer []byte) (backend.AppendTracker, error) {
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
func (rw *readerWriter) CloseAppend(context.Context, backend.AppendTracker) error {
	return nil
}

func (rw *readerWriter) Delete(ctx context.Context, name string, keypath backend.KeyPath, _ bool) error {
	blobURL, err := GetBlobURL(ctx, rw.cfg, backend.ObjectFileName(keypath, name))
	if err != nil {
		return errors.Wrapf(err, "cannot get Azure blob URL, name: %s", backend.ObjectFileName(keypath, name))
	}

	if _, err = blobURL.Delete(ctx, blob.DeleteSnapshotsOptionInclude, blob.BlobAccessConditions{}); err != nil {
		return readError(err)
	}
	return nil
}

// List implements backend.Reader
func (rw *readerWriter) List(ctx context.Context, keypath backend.KeyPath) ([]string, error) {
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
			return objects, errors.Wrap(err, "iterating tenants")
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

// Read implements backend.Reader
func (rw *readerWriter) Read(ctx context.Context, name string, keypath backend.KeyPath, _ bool) (io.ReadCloser, int64, error) {
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
func (rw *readerWriter) ReadRange(ctx context.Context, name string, keypath backend.KeyPath, offset uint64, buffer []byte, _ bool) error {
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
func (rw *readerWriter) Shutdown() {
}

func (rw *readerWriter) WriteVersioned(ctx context.Context, name string, keypath backend.KeyPath, data io.Reader, version backend.Version) (backend.Version, error) {
	// TODO use conditional if-match API
	_, currentVersion, err := rw.ReadVersioned(ctx, name, keypath)
	if err != nil && err != backend.ErrDoesNotExist {
		return "", err
	}

	level.Info(log.Logger).Log("msg", "WriteVersioned - fetching data", "currentVersion", currentVersion, "err", err, "version", version)

	// object does not exist - supplied version must be "0"
	if err == backend.ErrDoesNotExist && version != backend.VersionNew {
		return "", backend.ErrVersionDoesNotMatch
	}
	if err != backend.ErrDoesNotExist && version != currentVersion {
		return "", backend.ErrVersionDoesNotMatch
	}

	err = rw.Write(ctx, name, keypath, data, -1, false)
	if err != nil {
		return "", err
	}

	_, currentVersion, err = rw.ReadVersioned(ctx, name, keypath)
	return currentVersion, err
}

func (rw *readerWriter) DeleteVersioned(ctx context.Context, name string, keypath backend.KeyPath, version backend.Version) error {
	// TODO use conditional if-match API
	_, currentVersion, err := rw.ReadVersioned(ctx, name, keypath)
	if err != nil && err != backend.ErrDoesNotExist {
		return err
	}
	if err != backend.ErrDoesNotExist && currentVersion != version {
		return backend.ErrVersionDoesNotMatch
	}

	return rw.Delete(ctx, name, keypath, false)
}

func (rw *readerWriter) ReadVersioned(ctx context.Context, name string, keypath backend.KeyPath) (io.ReadCloser, backend.Version, error) {
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

func (rw *readerWriter) writeAll(ctx context.Context, name string, b []byte) error {
	err := rw.writer(ctx, bytes.NewReader(b), name)
	if err != nil {
		return err
	}

	return nil
}

func (rw *readerWriter) append(ctx context.Context, src []byte, name string) error {
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

func (rw *readerWriter) writer(ctx context.Context, src io.Reader, name string) error {
	blobURL := rw.containerURL.NewBlockBlobURL(name)

	if _, err := blob.UploadStreamToBlockBlob(ctx, src, blobURL,
		blob.UploadStreamToBlockBlobOptions{
			BufferSize: rw.cfg.BufferSize,
			MaxBuffers: rw.cfg.MaxBuffers,
		},
	); err != nil {
		return errors.Wrapf(err, "cannot upload blob, name: %s", name)
	}
	return nil
}

func (rw *readerWriter) readRange(ctx context.Context, name string, offset int64, destBuffer []byte) error {
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

func (rw *readerWriter) readAll(ctx context.Context, name string) ([]byte, blob.ETag, error) {
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
		return errors.Wrap(err, "reading storage container")
	}
	if storageError.ServiceCode() == blob.ServiceCodeBlobNotFound {
		return backend.ErrDoesNotExist
	}
	return errors.Wrap(storageError, "reading Azure blob container")
}
