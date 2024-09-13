package gcs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"cloud.google.com/go/storage"
	"github.com/go-kit/log/level"
	"github.com/google/uuid"
	"github.com/googleapis/gax-go/v2"
	"google.golang.org/api/iterator"

	"github.com/grafana/tempo/tempodb/backend"
)

func (rw *readerWriter) MarkBlockCompacted(blockID uuid.UUID, tenantID string) error {
	// move meta files to a new location

	metaFilenamePb := backend.MetaFileNamePb(blockID, tenantID, rw.cfg.Prefix)
	compactedMetaFilenamePb := backend.CompactedMetaFileNamePb(blockID, tenantID, rw.cfg.Prefix)

	srcPb := rw.bucket.Object(metaFilenamePb)
	dstPb := rw.bucket.Object(compactedMetaFilenamePb).Retryer(
		storage.WithBackoff(gax.Backoff{}),
		storage.WithPolicy(storage.RetryAlways),
	)

	ctx := context.TODO()
	_, err := dstPb.CopierFrom(srcPb).Run(ctx)
	if err != nil {
		level.Error(rw.logger).Log("msg", "error copying obj meta.pb to compacted.pb, is this block from previous Tempo version?", "err", err)
	} else {
		err = srcPb.Delete(ctx)
		if err != nil {
			return err
		}
	}

	metaFilename := backend.MetaFileName(blockID, tenantID, rw.cfg.Prefix)
	compactedMetaFilename := backend.CompactedMetaFileName(blockID, tenantID, rw.cfg.Prefix)

	src := rw.bucket.Object(metaFilename)
	dst := rw.bucket.Object(compactedMetaFilename).Retryer(
		storage.WithBackoff(gax.Backoff{}),
		storage.WithPolicy(storage.RetryAlways),
	)

	_, err = dst.CopierFrom(src).Run(ctx)
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
		Prefix:   backend.RootPath(blockID, tenantID, rw.cfg.Prefix),
		Versions: false,
	})

	for {
		attrs, err := iter.Next()
		if errors.Is(err, iterator.Done) {
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
	outPb, err := rw.compactedBlockMetaPb(blockID, tenantID)
	if err == nil {
		return outPb, nil
	}

	// TODO: record a note about fallback

	name := backend.CompactedMetaFileName(blockID, tenantID, rw.cfg.Prefix)

	bytes, attrs, err := rw.readAll(context.Background(), name)
	if err != nil {
		return nil, readError(err)
	}

	out := &backend.CompactedBlockMeta{}
	err = json.Unmarshal(bytes, out)
	if err != nil {
		return nil, err
	}
	out.CompactedTime = attrs.LastModified

	return out, nil
}

func (rw *readerWriter) compactedBlockMetaPb(blockID uuid.UUID, tenantID string) (*backend.CompactedBlockMeta, error) {
	name := backend.CompactedMetaFileNamePb(blockID, tenantID, rw.cfg.Prefix)

	bytes, attrs, err := rw.readAll(context.Background(), name)
	if err != nil {
		return nil, readError(err)
	}

	out := &backend.CompactedBlockMeta{}
	err = out.Unmarshal(bytes)
	if err != nil {
		return nil, err
	}
	out.CompactedTime = attrs.LastModified

	return out, nil
}
