package gcs

import (
	"context"
	"fmt"
	"path"

	"cloud.google.com/go/storage"
	"github.com/google/uuid"
	"google.golang.org/api/iterator"
)

func (rw *readerWriter) MarkBlockCompacted(blockID uuid.UUID, tenantID string) error {
	// move meta file to a new location
	metaFilename := rw.metaFileName(blockID, tenantID)
	compactedMetaFilename := rw.compactedMetaFileName(blockID, tenantID)

	src := rw.bucket.Object(metaFilename)
	dst := rw.bucket.Object(compactedMetaFilename)

	ctx := context.TODO()
	_, err := dst.CopierFrom(src).Run(ctx)

	return err
}

func (rw *readerWriter) ClearBlock(blockID uuid.UUID, tenantID string) error {
	if len(tenantID) == 0 {
		return fmt.Errorf("empty tenant id")
	}

	if blockID == uuid.Nil {
		return fmt.Errorf("empty block id")
	}

	ctx := context.TODO()
	iter := rw.bucket.Objects(ctx, &storage.Query{
		Prefix:   rw.rootPath(blockID, tenantID),
		Versions: false,
	})

	for {
		attrs, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return err
		}

		o := rw.bucket.Object(attrs.Name)
		err = o.Delete(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

func (rw *readerWriter) CompactedBlockMeta(blockID uuid.UUID, tenantID string) ([]byte, error) {
	name := rw.compactedMetaFileName(blockID, tenantID)
	return rw.readAll(context.Background(), name)
}

func (rw *readerWriter) compactedMetaFileName(blockID uuid.UUID, tenantID string) string {
	return path.Join(rw.rootPath(blockID, tenantID), "meta.compacted.json")
}
