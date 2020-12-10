package azure

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"time"

	blob "github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/google/uuid"
	"github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"

	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/util"
	"github.com/grafana/tempo/tempodb/encoding"
)

// Dir represents the char separator used by the blob virtual directory structure
const Dir = "/"

type readerWriter struct {
	cfg          *Config
	containerURL blob.ContainerURL
}

// New creates a Azure blob container
func New(cfg *Config) (backend.Reader, backend.Writer, backend.Compactor, error) {
	ctx := context.Background()

	container, err := createContainer(ctx, cfg)
	if err != nil {
		ret, ok := err.(blob.StorageError)
		if !ok {
			return nil, nil, nil, errors.Wrap(err, "creating storage container")
		}
		if ret.ServiceCode() == "ContainerAlreadyExists" {
			container, err = getContainer(ctx, cfg)
			if err != nil {
				return nil, nil, nil, errors.Wrap(err, "getting storage container")
			}
		} else {
			return nil, nil, nil, errors.Wrap(err, "creating storage container")
		}
	}

	rw := &readerWriter{
		cfg:          cfg,
		containerURL: container,
	}

	return rw, rw, rw, nil
}

func (rw *readerWriter) Write(ctx context.Context, meta *encoding.BlockMeta, bBloom [][]byte, bIndex []byte, objectFilePath string) error {
	blockID := meta.BlockID
	tenantID := meta.TenantID

	if err := util.FileExists(objectFilePath); err != nil {
		return err
	}

	src, err := os.Open(objectFilePath)
	if err != nil {
		return err
	}
	defer src.Close()

	err = rw.writer(ctx, bufio.NewReader(src), util.ObjectFileName(blockID, tenantID))
	if err != nil {
		return err
	}
	err = rw.WriteBlockMeta(ctx, nil, meta, bBloom, bIndex)
	if err != nil {
		return err
	}
	return nil
}

func (rw *readerWriter) WriteBlockMeta(ctx context.Context, tracker backend.AppendTracker, meta *encoding.BlockMeta, bBloom [][]byte, bIndex []byte) error {

	blockID := meta.BlockID
	tenantID := meta.TenantID

	for i, b := range bBloom {
		err := rw.writeAll(ctx, util.BloomFileName(blockID, tenantID, i), b)
		if err != nil {
			return err
		}
	}

	err := rw.writeAll(ctx, util.IndexFileName(blockID, tenantID), bIndex)
	if err != nil {
		return err
	}

	bMeta, err := json.Marshal(meta)
	if err != nil {
		return err
	}

	err = rw.writeAll(ctx, util.MetaFileName(blockID, tenantID), bMeta)
	if err != nil {
		return err
	}

	return nil
}

type AppenderTracker struct {
	Name string
}

func (rw *readerWriter) AppendObject(ctx context.Context, tracker backend.AppendTracker, meta *encoding.BlockMeta, bObject []byte) (backend.AppendTracker, error) {
	var a AppenderTracker
	if tracker == nil {
		blockID := meta.BlockID
		tenantID := meta.TenantID

		a.Name = util.ObjectFileName(blockID, tenantID)

		err := rw.writeAll(ctx, util.ObjectFileName(blockID, tenantID), bObject)
		if err != nil {
			return nil, err
		}
	} else {
		a = tracker.(AppenderTracker)

		_, err := rw.append(ctx, bObject, a.Name)
		if err != nil {
			return nil, err
		}
	}

	return a, nil
}

func (rw *readerWriter) Tenants(ctx context.Context) ([]string, error) {

	marker := blob.Marker{}

	tenants := make([]string, 0)
	for {
		list, err := rw.containerURL.ListBlobsHierarchySegment(ctx, marker, Dir, blob.ListBlobsSegmentOptions{
			Details: blob.BlobListingDetails{},
		})
		if err != nil {
			return tenants, errors.Wrap(err, "iterating tenants")

		}
		marker = list.NextMarker

		for _, blob := range list.Segment.BlobPrefixes {
			tenants = append(tenants, strings.TrimSuffix(blob.Name, Dir))
		}

		// Continue iterating if we are not done.
		if !marker.NotDone() {
			break
		}
	}
	return tenants, nil
}
func (rw *readerWriter) Blocks(ctx context.Context, tenantID string) ([]uuid.UUID, error) {
	var warning error
	blocks := make([]uuid.UUID, 0)

	marker := blob.Marker{}

	for {
		list, err := rw.containerURL.ListBlobsHierarchySegment(ctx, marker, Dir, blob.ListBlobsSegmentOptions{
			Prefix:  tenantID + Dir,
			Details: blob.BlobListingDetails{},
		})
		if err != nil {
			warning = err
			continue
		}
		marker = list.NextMarker

		for _, blob := range list.Segment.BlobPrefixes {
			idString := strings.TrimSuffix(strings.TrimPrefix(blob.Name, tenantID+"/"), "/")
			blockID, err := uuid.Parse(idString)
			if err != nil {
				warning = fmt.Errorf("failed parse on blockID %s: %v", idString, err)
				continue
			}
			blocks = append(blocks, blockID)
		}
		// Continue iterating if we are not done.
		if !marker.NotDone() {
			break
		}

	}
	return blocks, warning
}

func (rw *readerWriter) BlockMeta(ctx context.Context, blockID uuid.UUID, tenantID string) (*encoding.BlockMeta, error) {

	name := util.MetaFileName(blockID, tenantID)

	bytes, err := rw.readAll(ctx, name)
	if err != nil {
		return nil, backend.ErrMetaDoesNotExist
	}

	out := &encoding.BlockMeta{}
	err = json.Unmarshal(bytes, out)
	if err != nil {
		return nil, err
	}

	return out, nil
}

func (rw *readerWriter) Bloom(ctx context.Context, blockID uuid.UUID, tenantID string, shardNum int) ([]byte, error) {
	span, derivedCtx := opentracing.StartSpanFromContext(ctx, "azure.Bloom")
	defer span.Finish()

	name := util.BloomFileName(blockID, tenantID, shardNum)
	return rw.readAll(derivedCtx, name)
}

func (rw *readerWriter) Index(ctx context.Context, blockID uuid.UUID, tenantID string) ([]byte, error) {
	span, derivedCtx := opentracing.StartSpanFromContext(ctx, "azure.Index")
	defer span.Finish()

	name := util.IndexFileName(blockID, tenantID)
	return rw.readAll(derivedCtx, name)
}

func (rw *readerWriter) Object(ctx context.Context, blockID uuid.UUID, tenantID string, start uint64, buffer []byte) error {
	span, derivedCtx := opentracing.StartSpanFromContext(ctx, "azure.Object")
	defer span.Finish()

	name := util.ObjectFileName(blockID, tenantID)
	err := rw.readRange(derivedCtx, name, int64(start), buffer)

	return err
}

func (rw *readerWriter) Shutdown() {

}

func (rw *readerWriter) writeAll(ctx context.Context, name string, b []byte) error {
	err := rw.writer(ctx, bytes.NewReader(b), name)
	if err != nil {
		return err
	}

	return nil
}

func (rw *readerWriter) append(ctx context.Context, src []byte, name string) (string, error) {
	appendBlobURL := rw.containerURL.NewAppendBlobURL(name)

	resp, err := appendBlobURL.AppendBlock(ctx, bytes.NewReader(src), blob.AppendBlobAccessConditions{}, nil)

	if err != nil {
		return "", errors.Errorf("cannot upload Azure blob, address: %s", name)
	}
	return resp.RequestID(), nil

}

func (rw *readerWriter) writer(ctx context.Context, src io.Reader, name string) error {
	blobURL := rw.containerURL.NewBlockBlobURL(name)

	if _, err := blob.UploadStreamToBlockBlob(ctx, src, blobURL,
		blob.UploadStreamToBlockBlobOptions{
			BufferSize: 3 * 1024 * 1024,
			MaxBuffers: 4,
		},
	); err != nil {
		return errors.Errorf("cannot upload Azure blob, address: %s", name)
	}
	return nil
}

func (rw *readerWriter) readRange(ctx context.Context, name string, offset int64, destBuffer []byte) error {
	blobURL := rw.containerURL.NewBlockBlobURL(name)

	var props *blob.BlobGetPropertiesResponse
	props, err := blobURL.GetProperties(ctx, blob.BlobAccessConditions{})
	if err != nil {
		return backend.ErrMetaDoesNotExist
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
			Parallelism: uint16(3),
			Progress:    nil,
			RetryReaderOptionsPerBlock: blob.RetryReaderOptions{
				MaxRetryRequests: rw.cfg.MaxRetries,
			},
		},
	); err != nil {
		return backend.ErrMetaDoesNotExist
	}

	_, err = bytes.NewReader(destBuffer).Read(destBuffer)
	if err != nil {
		return err
	}

	return nil
}

func (rw *readerWriter) readAll(ctx context.Context, name string) ([]byte, error) {
	blobURL := rw.containerURL.NewBlockBlobURL(name)

	var props *blob.BlobGetPropertiesResponse
	props, err := blobURL.GetProperties(ctx, blob.BlobAccessConditions{})
	if err != nil {
		return nil, backend.ErrMetaDoesNotExist
	}

	destBuffer := make([]byte, props.ContentLength())

	if err := blob.DownloadBlobToBuffer(context.Background(), blobURL.BlobURL, 0, props.ContentLength(),
		destBuffer, blob.DownloadFromBlobOptions{
			BlockSize:   blob.BlobDefaultDownloadBlockSize,
			Parallelism: uint16(3),
			Progress:    nil,
			RetryReaderOptionsPerBlock: blob.RetryReaderOptions{
				MaxRetryRequests: rw.cfg.MaxRetries,
			},
		},
	); err != nil {
		return nil, backend.ErrMetaDoesNotExist
	}

	return destBuffer, nil
}

func getContainerURL(ctx context.Context, conf *Config) (blob.ContainerURL, error) {
	c, err := blob.NewSharedKeyCredential(conf.StorageAccountName, conf.StorageAccountKey)
	if err != nil {
		return blob.ContainerURL{}, err
	}

	retryOptions := blob.RetryOptions{
		MaxTries: int32(conf.MaxRetries),
		Policy:   blob.RetryPolicyExponential,
	}
	if deadline, ok := ctx.Deadline(); ok {
		retryOptions.TryTimeout = time.Until(deadline)
	}

	p := blob.NewPipeline(c, blob.PipelineOptions{
		Retry:     retryOptions,
		Telemetry: blob.TelemetryOptions{Value: "Tempo"},
	})

	u, err := url.Parse(fmt.Sprintf("https://%s.%s", conf.StorageAccountName, conf.Endpoint))

	if conf.DevelopmentMode {
		u, err = url.Parse(fmt.Sprintf("http://%s:10000/%s", conf.Endpoint, conf.StorageAccountName))
	}
	if err != nil {
		return blob.ContainerURL{}, err
	}

	service := blob.NewServiceURL(*u, p)

	return service.NewContainerURL(conf.ContainerName), nil
}
func getContainer(ctx context.Context, conf *Config) (blob.ContainerURL, error) {
	c, err := getContainerURL(ctx, conf)
	if err != nil {
		return blob.ContainerURL{}, err
	}
	// Getting container properties to check if container exists
	_, err = c.GetProperties(ctx, blob.LeaseAccessConditions{})
	return c, err
}

func createContainer(ctx context.Context, conf *Config) (blob.ContainerURL, error) {
	c, err := getContainerURL(ctx, conf)
	if err != nil {
		return blob.ContainerURL{}, err
	}
	_, err = c.Create(
		ctx,
		blob.Metadata{},
		blob.PublicAccessNone)
	return c, err
}

func getBlobURL(ctx context.Context, conf *Config, blobName string) (blob.BlockBlobURL, error) {
	c, err := getContainerURL(ctx, conf)
	if err != nil {
		return blob.BlockBlobURL{}, err
	}
	return c.NewBlockBlobURL(blobName), nil
}
