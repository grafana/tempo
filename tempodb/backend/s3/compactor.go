package s3

import (
	"context"
	"encoding/json"

	"github.com/minio/minio-go/v7"

	"github.com/go-kit/kit/log/level"
	"github.com/google/uuid"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/util"
	"github.com/pkg/errors"
)

func (rw *readerWriter) MarkBlockCompacted(blockID uuid.UUID, tenantID string) error {
	if len(tenantID) == 0 {
		return backend.ErrEmptyTenantID
	}
	if blockID == uuid.Nil {
		return backend.ErrEmptyBlockID
	}

	metaFileName := util.MetaFileName(blockID, tenantID)
	// copy meta.json to meta.compacted.json
	_, err := rw.core.CopyObject(
		context.TODO(),
		rw.cfg.Bucket,
		metaFileName,
		rw.cfg.Bucket,
		util.CompactedMetaFileName(blockID, tenantID),
		nil,
		minio.CopySrcOptions{},
		minio.PutObjectOptions{},
	)
	if err != nil {
		return errors.Wrap(err, "error copying obj meta to compacted obj meta")
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

	path := util.RootPath(blockID, tenantID) + "/"
	level.Debug(rw.logger).Log("msg", "deleting block", "block path", path)

	// ListObjects(bucket, prefix, marker, delimiter string, maxKeys int)
	res, err := rw.core.ListObjects(rw.cfg.Bucket, path, "", "/", 0)
	if err != nil {
		return errors.Wrapf(err, "error listing objects in bucket %s", rw.cfg.Bucket)
	}

	level.Debug(rw.logger).Log("msg", "listing objects", "found", len(res.Contents))
	for _, obj := range res.Contents {
		err = rw.core.RemoveObject(context.TODO(), rw.cfg.Bucket, obj.Key, minio.RemoveObjectOptions{})
		if err != nil {
			return errors.Wrapf(err, "error deleting obj from s3: %s", obj.Key)
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

	compactedMetaFileName := util.CompactedMetaFileName(blockID, tenantID)
	bytes, info, err := rw.readAllWithObjInfo(context.TODO(), compactedMetaFileName)
	if err != nil && err == backend.ErrMetaDoesNotExist {
		return nil, backend.ErrMetaDoesNotExist
	} else if err != nil {
		return nil, errors.Wrapf(err, "error fetching compacted meta file %s", compactedMetaFileName)
	}

	out := &backend.CompactedBlockMeta{}
	err = json.Unmarshal(bytes, out)
	if err != nil {
		return nil, err
	}
	out.CompactedTime = info.LastModified

	return out, err
}
