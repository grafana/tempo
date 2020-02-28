package gcs

import (
	"context"
	"fmt"

	"cloud.google.com/go/storage"
	"github.com/google/uuid"
	"github.com/grafana/frigg/friggdb/backend"
	"github.com/grafana/frigg/pkg/util"
	"google.golang.org/api/iterator"
)

type compactor struct {
	rw *readerWriter
}

func NewCompactor(cfg *Config) (backend.Compactor, error) {
	rw, err := newReaderWriter(cfg)
	if err != nil {
		return nil, err
	}

	return &compactor{
		rw: rw,
	}, nil
}

func (c *compactor) MarkBlockCompacted(blockID uuid.UUID, tenantID string) error {
	// move meta file to a new location
	metaFilename := c.rw.metaFileName(blockID, tenantID)
	compactedMetaFilename := util.CompactedMetaFileName(blockID, tenantID)

	src := c.rw.bucket.Object(metaFilename)
	dst := c.rw.bucket.Object(compactedMetaFilename)

	ctx := context.TODO()
	_, err := dst.CopierFrom(src).Run(ctx)
	if err != nil {
		return err
	}

	return src.Delete(ctx)
}

func (c *compactor) ClearBlock(blockID uuid.UUID, tenantID string) error {
	if len(tenantID) == 0 {
		return fmt.Errorf("empty tenant id")
	}

	if blockID == uuid.Nil {
		return fmt.Errorf("empty block id")
	}

	ctx := context.TODO()
	iter := c.rw.bucket.Objects(ctx, &storage.Query{
		Prefix:   util.RootPath("", tenantID, blockID),
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

		o := c.rw.bucket.Object(attrs.Name)
		err = o.Delete(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}
