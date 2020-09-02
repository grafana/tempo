package s3

import (
	"context"
	"encoding/json"

	"github.com/grafana/tempo/tempodb/backend"
	"github.com/pkg/errors"
	"github.com/grafana/tempo/tempodb/backend/util"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/google/uuid"
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
	_, err := rw.core.CopyObjectWithContext(
		context.TODO(),
		rw.cfg.Bucket,
		metaFileName,
		rw.cfg.Bucket,
		util.CompactedMetaFileName(blockID,tenantID),
		nil,
		)
	if err != nil {
		return errors.Wrap(err, "error copying obj meta to compacted obj meta")
	}

	// delete meta.json
	return rw.core.RemoveObject(rw.cfg.Bucket, metaFileName)
}

func (rw *readerWriter) ClearBlock(blockID uuid.UUID, tenantID string) error {
	if len(tenantID) == 0 {
		return backend.ErrEmptyTenantID
	}
	if blockID == uuid.Nil {
		return backend.ErrEmptyBlockID
	}

	return rw.core.RemoveObject(rw.cfg.Bucket, util.MetaFileName(blockID, tenantID))
}

func (rw *readerWriter) CompactedBlockMeta(blockID uuid.UUID, tenantID string) (*encoding.CompactedBlockMeta, error) {
	if len(tenantID) == 0 {
		return nil, backend.ErrEmptyTenantID
	}
	if blockID == uuid.Nil {
		return nil, backend.ErrEmptyBlockID
	}

	compactedMetaFileName := util.CompactedMetaFileName(blockID, tenantID)
	bytes, info, err := rw.readAllWithObjInfo(context.TODO(), compactedMetaFileName)
	if err != nil {
		return nil, errors.Wrapf(err, "error fetching compacted meta file %s", compactedMetaFileName)
	}

	out := &encoding.CompactedBlockMeta{}
	err = json.Unmarshal(bytes, out)
	if err != nil {
		return nil, err
	}
	out.CompactedTime = info.LastModified

	return out, err
}

