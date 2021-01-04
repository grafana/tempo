package gcs

import (
	"context"
	"encoding/json"
	"fmt"

	"cloud.google.com/go/storage"
	"github.com/google/uuid"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/util"
	"google.golang.org/api/iterator"
)

func (rw *readerWriter) MarkBlockCompacted(blockID uuid.UUID, tenantID string) error {
	// move meta file to a new location
	metaFilename := util.MetaFileName(blockID, tenantID)
	compactedMetaFilename := util.CompactedMetaFileName(blockID, tenantID)

	src := rw.bucket.Object(metaFilename)
	dst := rw.bucket.Object(compactedMetaFilename)

	ctx := context.TODO()
	_, err := dst.CopierFrom(src).Run(ctx)
	if err != nil {
		return err
	}

	return src.Delete(ctx)
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
		Prefix:   util.RootPath(blockID, tenantID),
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

func (rw *readerWriter) CompactedBlockMeta(blockID uuid.UUID, tenantID string) (*backend.CompactedBlockMeta, error) {
	name := util.CompactedMetaFileName(blockID, tenantID)

	bytes, modTime, err := rw.readAllWithModTime(context.Background(), name)
	if err == storage.ErrObjectNotExist {
		return nil, backend.ErrMetaDoesNotExist
	}
	if err != nil {
		return nil, err
	}

	out := &backend.CompactedBlockMeta{}
	err = json.Unmarshal(bytes, out)
	if err != nil {
		return nil, err
	}
	out.CompactedTime = modTime

	return out, err
}
