package s3

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/minio/minio-go/v7"

	"github.com/go-kit/log/level"
	"github.com/google/uuid"
	"github.com/grafana/tempo/tempodb/backend"
)

func (rw *readerWriter) MarkBlockCompacted(blockID uuid.UUID, tenantID string) error {
	if len(tenantID) == 0 {
		return backend.ErrEmptyTenantID
	}
	if blockID == uuid.Nil {
		return backend.ErrEmptyBlockID
	}

	metaFileName := backend.MetaFileName(blockID, tenantID, rw.cfg.Prefix)
	// copy meta.json to meta.compacted.json
	_, err := rw.core.CopyObject(
		context.TODO(),
		rw.cfg.Bucket,
		metaFileName,
		rw.cfg.Bucket,
		backend.CompactedMetaFileName(blockID, tenantID, rw.cfg.Prefix),
		nil,
		minio.CopySrcOptions{},
		minio.PutObjectOptions{},
	)
	if err != nil {
		return fmt.Errorf("error copying obj meta to compacted obj meta: %w", err)
	}

	// delete meta.json
	return rw.core.RemoveObject(context.TODO(), rw.cfg.Bucket, metaFileName, minio.RemoveObjectOptions{})
}

func (rw *readerWriter) ClearBlock(blockID uuid.UUID, tenantID string) error {
	if len(tenantID) == 0 {
		return backend.ErrEmptyTenantID
	}
	if blockID == uuid.Nil {
		return backend.ErrEmptyBlockID
	}

	path := backend.RootPath(blockID, tenantID, rw.cfg.Prefix) + "/"
	level.Debug(rw.logger).Log("msg", "deleting block", "block path", path)

	// ListObjects(bucket, prefix, marker, delimiter string, maxKeys int)
	res, err := rw.core.ListObjects(rw.cfg.Bucket, path, "", "/", 0)
	if err != nil {
		return fmt.Errorf("error listing objects in bucket %s: %w", rw.cfg.Bucket, err)
	}

	level.Debug(rw.logger).Log("msg", "listing objects", "found", len(res.Contents))
	for _, obj := range res.Contents {
		err = rw.core.RemoveObject(context.TODO(), rw.cfg.Bucket, obj.Key, minio.RemoveObjectOptions{})
		if err != nil {
			return fmt.Errorf("error deleting obj from s3: %s: %w", obj.Key, err)
		}
	}

	return nil
}

func (rw *readerWriter) CompactedBlockMeta(blockID uuid.UUID, tenantID string) (*backend.CompactedBlockMeta, error) {
	if len(tenantID) == 0 {
		return nil, backend.ErrEmptyTenantID
	}
	if blockID == uuid.Nil {
		return nil, backend.ErrEmptyBlockID
	}

	compactedMetaFileName := backend.CompactedMetaFileName(blockID, tenantID, rw.cfg.Prefix)
	bytes, info, err := rw.readAllWithObjInfo(context.TODO(), compactedMetaFileName)
	if err != nil {
		return nil, readError(err)
	}

	out := &backend.CompactedBlockMeta{}
	err = json.Unmarshal(bytes, out)
	if err != nil {
		return nil, err
	}
	out.CompactedTime = info.LastModified

	return out, nil
}
