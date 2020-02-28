package util

import (
	"path"

	"github.com/google/uuid"
)

func CompactedMetaFileName(blockID uuid.UUID, tenantID string) string {
	return path.Join(RootPath("", tenantID, blockID), "meta.compacted.json")
}

func RootPath(basepath string, tenantID string, blockID uuid.UUID) string {
	return path.Join(basepath, tenantID, blockID.String())
}
