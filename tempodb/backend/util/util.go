package util

import (
	"os"
	"path"

	"github.com/google/uuid"
)

func MetaFileName(blockID uuid.UUID, tenantID string) string {
	return path.Join(RootPath(blockID, tenantID), "meta.json")
}

func ObjectFileName(blockID uuid.UUID, tenantID string, name string) string {
	return path.Join(RootPath(blockID, tenantID), name)
}

func CompactedMetaFileName(blockID uuid.UUID, tenantID string) string {
	return path.Join(RootPath(blockID, tenantID), "meta.compacted.json")
}

// nolint:interfacer
func RootPath(blockID uuid.UUID, tenantID string) string {
	return path.Join(tenantID, blockID.String())
}

func FileExists(filename string) error {
	_, err := os.Stat(filename)
	return err
}
