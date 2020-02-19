package local

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/google/uuid"
)

func (rw *readerWriter) MarkBlockCompacted(blockID uuid.UUID, tenantID string) error {
	// move meta file to a new location
	metaFilename := rw.metaFileName(blockID, tenantID)
	compactedMetaFilename := rw.compactedMetaFileName(blockID, tenantID)

	return os.Rename(metaFilename, compactedMetaFilename)
}

func (rw *readerWriter) ClearBlock(blockID uuid.UUID, tenantID string) error {
	if len(tenantID) == 0 {
		return fmt.Errorf("empty tenant id")
	}

	if blockID == uuid.Nil {
		return fmt.Errorf("empty block id")
	}

	return os.RemoveAll(rw.rootPath(blockID, tenantID))
}

func (rw *readerWriter) CompactedBlockMeta(blockID uuid.UUID, tenantID string) ([]byte, error) {
	filename := rw.compactedMetaFileName(blockID, tenantID)
	return ioutil.ReadFile(filename)
}

func (rw *readerWriter) compactedMetaFileName(blockID uuid.UUID, tenantID string) string {
	return path.Join(rw.rootPath(blockID, tenantID), "meta.compacted.json")
}
