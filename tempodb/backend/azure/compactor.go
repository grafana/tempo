package azure

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"
	"github.com/google/uuid"

	"github.com/grafana/tempo/pkg/boundedwaitgroup"
	"github.com/grafana/tempo/tempodb/backend"
)

type BlobAttributes struct {
	// Size is the blob size in bytes.
	Size int64 `json:"size"`

	// LastModified is the timestamp the blob was last modified.
	LastModified time.Time `json:"last_modified"`
}

func (rw *Azure) MarkBlockCompacted(blockID uuid.UUID, tenantID string) error {
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

	// metaFilename is already prefixed so use deleteRaw - rw.Delete would re-apply it.
	return rw.deleteRaw(ctx, metaFilename)
}

// blobDeleteConcurrency is how many of a block's blobs are deleted at once. A
// block is a meta, a data file, an index and one bloom blob per shard, so
// deleting them one at a time costs ~30 sequential round trips per block.
const blobDeleteConcurrency = uint(16)

func (rw *Azure) ClearBlock(blockID uuid.UUID, tenantID string) error {
	if len(tenantID) == 0 {
		return fmt.Errorf("empty tenant id")
	}

	if blockID == uuid.Nil {
		return fmt.Errorf("empty block id")
	}

	ctx := context.TODO()

	var (
		warning error
		names   []string
		prefix  = backend.RootPath(blockID, tenantID, rw.cfg.Prefix)
		pager   = rw.containerClient.NewListBlobsHierarchyPager("", &container.ListBlobsHierarchyOptions{
			Include: container.ListBlobsInclude{},
			Prefix:  &prefix,
		})
	)

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			warning = err
			continue
		}

		for _, b := range page.Segment.BlobItems {
			if b.Name == nil {
				return fmt.Errorf("unexpected empty blob name when listing %s", prefix)
			}
			names = append(names, *b.Name)
		}
	}

	// The deletes are independent and latency bound, so issue them concurrently
	// rather than one at a time.
	var (
		bg  = boundedwaitgroup.New(blobDeleteConcurrency)
		mtx sync.Mutex
	)

	for _, name := range names {
		bg.Add(1)
		go func(name string) {
			defer bg.Done()

			// b.Name from the listing is already prefixed so use deleteRaw - rw.Delete would re-apply it.
			if err := rw.deleteRaw(ctx, name); err != nil {
				mtx.Lock()
				warning = err
				mtx.Unlock()
			}
		}(name)
	}

	bg.Wait()

	return warning
}

func (rw *Azure) CompactedBlockMeta(blockID uuid.UUID, tenantID string) (*backend.CompactedBlockMeta, error) {
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

func (rw *Azure) readAllWithModTime(ctx context.Context, name string) ([]byte, time.Time, error) {
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
func (rw *Azure) getAttributes(ctx context.Context, name string) (BlobAttributes, error) {
	blobClient, err := getBlobClient(ctx, rw.cfg, name)
	if err != nil {
		return BlobAttributes{}, fmt.Errorf("cannot get Azure blob client, name: %s: %w", name, err)
	}

	props, err := blobClient.GetProperties(ctx, &blob.GetPropertiesOptions{})
	if err != nil {
		return BlobAttributes{}, err
	}

	if props.ContentLength == nil {
		return BlobAttributes{}, fmt.Errorf("expected content length but got none for blob %s: %w", name, err)
	}

	if props.LastModified == nil {
		return BlobAttributes{}, fmt.Errorf("expected last modified but got none for blob %s: %w", name, err)
	}

	return BlobAttributes{
		Size:         *props.ContentLength,
		LastModified: *props.LastModified,
	}, nil
}
