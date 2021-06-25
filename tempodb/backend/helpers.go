package backend

import (
	"context"
	"encoding/json"
	"path"

	"github.com/google/uuid"
)

const (
	// jpe NameMeta? NameCompactedMeta?
	MetaName          = "meta.json"
	CompactedMetaName = "meta.compacted.json"
	BlockIndexName    = "blockindex.json.gz"
)

// jpe test/comment/organize better
func WriteBlockMeta(ctx context.Context, w Writer, meta *BlockMeta) error {
	blockID := meta.BlockID
	tenantID := meta.TenantID

	bMeta, err := json.Marshal(meta)
	if err != nil {
		return err
	}

	return w.Write(ctx, MetaName, KeyPathForBlock(blockID, tenantID), bMeta)
}

func Tenants(ctx context.Context, r Reader) ([]string, error) {
	return r.List(ctx, nil)
}

func Blocks(ctx context.Context, r Reader, tenantID string) ([]uuid.UUID, error) {
	objects, err := r.List(ctx, KeyPath{tenantID})
	if err != nil {
		return nil, err
	}

	// translate everything to UUIDs, if we see a bucket index we can skip that
	blockIDs := make([]uuid.UUID, 0, len(objects))
	for _, id := range objects {
		if id == BlockIndexName {
			continue
		}
		uuid, err := uuid.Parse(id)
		if err != nil {
			return nil, err
		}
		blockIDs = append(blockIDs, uuid)
	}

	return blockIDs, nil
}

func ReadBlockMeta(ctx context.Context, r Reader, blockID uuid.UUID, tenantID string) (*BlockMeta, error) {
	bytes, err := r.Read(ctx, MetaName, KeyPathForBlock(blockID, tenantID))
	if err != nil {
		return nil, err
	}

	out := &BlockMeta{}
	err = json.Unmarshal(bytes, out)
	if err != nil {
		return nil, err
	}

	return out, nil
}

func ReadCompactedBlockMeta(ctx context.Context, r Reader, blockID uuid.UUID, tenantID string) (*CompactedBlockMeta, error) {
	bytes, err := r.Read(ctx, CompactedMetaName, KeyPathForBlock(blockID, tenantID))
	if err != nil {
		return nil, err
	}

	out := &CompactedBlockMeta{}
	err = json.Unmarshal(bytes, out)
	if err != nil {
		return nil, err
	}

	return out, nil
}

// jpe review these, do we need them all?
func ObjectFileName(keypath KeyPath, name string) string {
	return path.Join(path.Join(keypath...), name)
}

func MetaFileName(blockID uuid.UUID, tenantID string) string {
	return path.Join(RootPath(blockID, tenantID), MetaName)
}

func CompactedMetaFileName(blockID uuid.UUID, tenantID string) string {
	return path.Join(RootPath(blockID, tenantID), CompactedMetaName)
}

// nolint:interfacer
func RootPath(blockID uuid.UUID, tenantID string) string {
	return path.Join(tenantID, blockID.String())
}

func KeyPathForBlock(blockID uuid.UUID, tenantID string) KeyPath {
	return []string{tenantID, blockID.String()}
}
