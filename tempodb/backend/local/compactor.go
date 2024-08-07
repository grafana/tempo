package local

import (
	"errors"
	"fmt"
	"os"
	"path"

	"github.com/gogo/protobuf/proto"
	"github.com/google/uuid"
	"github.com/grafana/tempo/tempodb/backend"
	backend_v1 "github.com/grafana/tempo/tempodb/backend/v1"
)

func (rw *Backend) MarkBlockCompacted(blockID uuid.UUID, tenantID string) error {
	// move meta file to a new location
	metaFilename := rw.metaFileName(blockID, tenantID)
	compactedMetaFilename := rw.compactedMetaFileName(blockID, tenantID)

	return os.Rename(metaFilename, compactedMetaFilename)
}

func (rw *Backend) ClearBlock(blockID uuid.UUID, tenantID string) error {
	if len(tenantID) == 0 {
		return errors.New("empty tenant id")
	}

	if blockID == uuid.Nil {
		return errors.New("empty block id")
	}

	path := rw.rootPath(backend.KeyPathForBlock(blockID, tenantID))
	err := os.RemoveAll(path)
	if err != nil {
		return fmt.Errorf("failed to remove keypath for block %s: %w", path, err)
	}

	return nil
}

func (rw *Backend) CompactedBlockMeta(blockID uuid.UUID, tenantID string) (*backend_v1.CompactedBlockMeta, error) {
	// TODO: consider fetching the json version here as well.

	filename := rw.compactedMetaFileName(blockID, tenantID)

	fi, err := os.Stat(filename)
	if err != nil {
		return nil, readError(err)
	}

	bytes, err := os.ReadFile(filename)
	if err != nil {
		return nil, readError(err)
	}

	out := &backend_v1.CompactedBlockMeta{}
	err = proto.Unmarshal(bytes, out)
	if err != nil {
		return nil, err
	}
	out.CompactedTime = fi.ModTime().Unix()

	return out, nil
}

func (rw *Backend) compactedMetaFileName(blockID uuid.UUID, tenantID string) string {
	return path.Join(rw.rootPath(backend.KeyPathForBlock(blockID, tenantID)), backend.CompactedMetaName)
}
