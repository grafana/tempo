package backend

import (
	"path"

	"github.com/google/uuid"
)

const (
	MetaName          = "meta.json"
	CompactedMetaName = "meta.compacted.json"
)

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
