package backend

import (
	"path"

	"github.com/google/uuid"
)

func ObjectFileName(keypath KeyPath, name string) string {
	return path.Join(path.Join(keypath...), name)
}

func MetaFileName(blockID uuid.UUID, tenantID string) string {
	return path.Join(RootPath(blockID, tenantID), "meta.json")
}

func CompactedMetaFileName(blockID uuid.UUID, tenantID string) string {
	return path.Join(RootPath(blockID, tenantID), "meta.compacted.json")
}

// nolint:interfacer
func RootPath(blockID uuid.UUID, tenantID string) string {
	return path.Join(tenantID, blockID.String())
}

func KeyPathForBlock(blockID uuid.UUID, tenantID string) KeyPath {
	return []string{tenantID, blockID.String()}
}
