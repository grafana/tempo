package cos

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-kit/log/level"
	"github.com/google/uuid"
	"github.com/tencentyun/cos-go-sdk-v5"

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
	compactedMetaFileName := backend.CompactedMetaFileName(blockID, tenantID, rw.cfg.Prefix)
	ctx := context.TODO()

	src, _, err := rw.readAll(ctx, metaFileName)
	if err != nil {
		return readError(err)
	}

	_, err = rw.client.Object.Put(ctx, compactedMetaFileName, bytes.NewReader(src), nil)
	if err != nil {
		return fmt.Errorf("error copying meta to compacted meta: %w", err)
	}

	return rw.Delete(ctx, metaFileName, []string{}, nil)
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

	ctx := context.TODO()

	opt := &cos.BucketGetOptions{
		Prefix:  path,
		MaxKeys: 1000,
	}
	result, _, err := rw.client.Bucket.Get(ctx, opt)
	if err != nil {
		return fmt.Errorf("error listing objects in bucket %s: %w", rw.cfg.Bucket, err)
	}

	level.Debug(rw.logger).Log("msg", "listing objects", "found", len(result.Contents))
	for _, obj := range result.Contents {
		_, err = rw.client.Object.Delete(ctx, obj.Key)
		if err != nil {
			return fmt.Errorf("error deleting obj from cos: %s: %w", obj.Key, err)
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
	bytesData, info, err := rw.readAllWithObjInfo(context.TODO(), compactedMetaFileName)
	if err != nil {
		return nil, readError(err)
	}

	out := &backend.CompactedBlockMeta{}
	err = json.Unmarshal(bytesData, out)
	if err != nil {
		return nil, err
	}

	out.CompactedTime, _ = time.Parse(time.RFC1123, info.LastModified)

	return out, nil
}
