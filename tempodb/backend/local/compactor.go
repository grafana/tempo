package local

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/google/uuid"

	"github.com/grafana/tempo/tempodb/backend"
)

func (rw *Backend) MarkBlockCompacted(blockID uuid.UUID, tenantID string) error {
	// move meta file to a new location
	metaFilename := rw.metaFileName(blockID, tenantID)
	compactedMetaFilename := rw.compactedMetaFileName(blockID, tenantID)

	return os.Rename(metaFilename, compactedMetaFilename)
}

func (rw *Backend) ClearBlock(blockID uuid.UUID, tenantID string) error {
	if len(tenantID) == 0 {
		return fmt.Errorf("empty tenant id")
	}

	if blockID == uuid.Nil {
		return fmt.Errorf("empty block id")
	}

	return os.RemoveAll(rw.rootPath(blockID, tenantID))
}

func (rw *Backend) CompactedBlockMeta(blockID uuid.UUID, tenantID string) (*backend.CompactedBlockMeta, error) {
	filename := rw.compactedMetaFileName(blockID, tenantID)

	fi, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return nil, backend.ErrMetaDoesNotExist
	}
	if err != nil {
		return nil, err
	}

	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	out := &backend.CompactedBlockMeta{}
	err = json.Unmarshal(bytes, out)
	if err != nil {
		return nil, err
	}
	out.CompactedTime = fi.ModTime()

	return out, err
}

func (rw *Backend) compactedMetaFileName(blockID uuid.UUID, tenantID string) string {
	return path.Join(rw.rootPath(blockID, tenantID), "meta.compacted.json")
}
