package util

import (
	"os"
	"path"

	"github.com/google/uuid"
)

func MetaFileName(blockID uuid.UUID, tenantID string) string {
	return path.Join(rootPath(blockID, tenantID), "meta.json")
}

func BloomFileName(blockID uuid.UUID, tenantID string) string {
	return path.Join(rootPath(blockID, tenantID), "bloom")
}

func IndexFileName(blockID uuid.UUID, tenantID string) string {
	return path.Join(rootPath(blockID, tenantID), "index")
}

func ObjectFileName(blockID uuid.UUID, tenantID string) string {
	return path.Join(rootPath(blockID, tenantID), "data")
}

func CompactedMetaFileName(blockID uuid.UUID, tenantID string) string {
	return path.Join(rootPath(blockID, tenantID), "meta.compacted.json")
}

func rootPath(blockID uuid.UUID, tenantID string) string {
	return path.Join(tenantID, blockID.String())
}

func FileExists(filename string) error {
	_, err := os.Stat(filename)
	return err
}
