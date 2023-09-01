package azure

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"
	"github.com/google/uuid"
	"github.com/pkg/errors"

	"github.com/grafana/tempo/tempodb/backend"
)

type BlobAttributes struct {
	// Size is the blob size in bytes.
	Size int64 `json:"size"`

	// LastModified is the timestamp the blob was last modified.
	LastModified time.Time `json:"last_modified"`
}

func (rw *readerWriter) MarkBlockCompacted(blockID uuid.UUID, tenantID string) error {
	if len(tenantID) == 0 {
		return backend.ErrEmptyTenantID
	}
	if blockID == uuid.Nil {
		return backend.ErrEmptyBlockID
	}

	// move meta file to a new location
	metaFilename := backend.MetaFileName(blockID, tenantID, rw.cfg.Prefix)
	compactedMetaFilename := backend.CompactedMetaFileName(blockID, tenantID, rw.cfg.Prefix)
	ctx := context.TODO()

	src, _, err := rw.readAll(ctx, metaFilename)
	if err != nil {
		return err
	}

	err = rw.writeAll(ctx, compactedMetaFilename, src)
	if err != nil {
		return err
	}

	// delete the old file
	return rw.delete(ctx, metaFilename)
}

func (rw *readerWriter) ClearBlock(blockID uuid.UUID, tenantID string) error {
	var warning error
	if len(tenantID) == 0 {
		return fmt.Errorf("empty tenant id")
	}

	if blockID == uuid.Nil {
		return fmt.Errorf("empty block id")
	}

	ctx := context.TODO()

	prefix := backend.RootPath(blockID, tenantID, rw.cfg.Prefix)
	pager := rw.containerClient.NewListBlobsHierarchyPager("", &container.ListBlobsHierarchyOptions{
		Include: container.ListBlobsInclude{},
		Prefix:  &prefix,
	})

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			warning = err
			continue
		}

		for _, b := range page.Segment.BlobItems {
			err = rw.delete(ctx, *b.Name)
			if err != nil {
				warning = err
				continue
			}
		}
	}

	return warning
}

func (rw *readerWriter) CompactedBlockMeta(blockID uuid.UUID, tenantID string) (*backend.CompactedBlockMeta, error) {
	if len(tenantID) == 0 {
		return nil, backend.ErrEmptyTenantID
	}
	if blockID == uuid.Nil {
		return nil, backend.ErrEmptyBlockID
	}
	name := backend.CompactedMetaFileName(blockID, tenantID, rw.cfg.Prefix)

	bytes, modTime, err := rw.readAllWithModTime(context.Background(), name)
	if err != nil {
		return nil, readError(err)
	}

	out := &backend.CompactedBlockMeta{}
	err = json.Unmarshal(bytes, out)
	if err != nil {
		return nil, err
	}
	out.CompactedTime = modTime

	return out, nil
}

func (rw *readerWriter) readAllWithModTime(ctx context.Context, name string) ([]byte, time.Time, error) {
	bytes, _, err := rw.readAll(ctx, name)
	if err != nil {
		return nil, time.Time{}, err
	}

	att, err := rw.getAttributes(ctx, name)
	if err != nil {
		return nil, time.Time{}, err
	}
	return bytes, att.LastModified, nil
}

// getAttributes returns information about the specified blob using its name.
func (rw *readerWriter) getAttributes(ctx context.Context, name string) (BlobAttributes, error) {
	blobClient, err := getBlobClient(ctx, rw.cfg, name)
	if err != nil {
		return BlobAttributes{}, errors.Wrapf(err, "cannot get Azure blob client, name: %s", name)
	}

	props, err := blobClient.GetProperties(ctx, &blob.GetPropertiesOptions{})
	if err != nil {
		return BlobAttributes{}, err
	}

	return BlobAttributes{
		Size:         *props.ContentLength,
		LastModified: *props.LastModified,
	}, nil
}

// Delete removes the blob with the given name.
func (rw *readerWriter) delete(ctx context.Context, name string) error {
	blobClient, err := getBlobClient(ctx, rw.cfg, name)
	if err != nil {
		return errors.Wrapf(err, "cannot get Azure blob client, name: %s", name)
	}

	deleteSnapshots := azblob.DeleteSnapshotsOptionTypeInclude
	if _, err := blobClient.Delete(ctx, &blob.DeleteOptions{DeleteSnapshots: &deleteSnapshots}); err != nil {
		return errors.Wrapf(err, "error deleting blob, name: %s", name)
	}
	return nil
}
