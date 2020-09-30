package util

import (
	"os"
	"path"
	"strconv"

	"github.com/google/uuid"
)

func MetaFileName(blockID uuid.UUID, tenantID string) string {
	return path.Join(RootPath(blockID, tenantID), "meta.json")
}

func BloomFileName(blockID uuid.UUID, tenantID string, bloomShard uint64) string {
	return path.Join(RootPath(blockID, tenantID), "bloom"+"-"+strconv.Itoa(int(bloomShard)))
}

func IndexFileName(blockID uuid.UUID, tenantID string) string {
	return path.Join(RootPath(blockID, tenantID), "index")
}

func ObjectFileName(blockID uuid.UUID, tenantID string) string {
	return path.Join(RootPath(blockID, tenantID), "data")
}

func CompactedMetaFileName(blockID uuid.UUID, tenantID string) string {
	return path.Join(RootPath(blockID, tenantID), "meta.compacted.json")
}

func BlockFileName(blockID uuid.UUID, tenantID string) string {
	return RootPath(blockID, tenantID) + "/"
}

// nolint:interfacer
func RootPath(blockID uuid.UUID, tenantID string) string {
	return path.Join(tenantID, blockID.String())
}

func FileExists(filename string) error {
	_, err := os.Stat(filename)
	return err
}
