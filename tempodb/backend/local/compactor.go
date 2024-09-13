package local

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"

	"github.com/go-kit/log/level"
	"github.com/google/uuid"
	"github.com/grafana/tempo/pkg/util/log"
	"github.com/grafana/tempo/tempodb/backend"
)

func (rw *Backend) MarkBlockCompacted(blockID uuid.UUID, tenantID string) error {
	metaFilenamePb := rw.metaFileNamePb(blockID, tenantID)
	compactedMetaFilenamePb := rw.compactedMetaFileNamePb(blockID, tenantID)

	err := os.Rename(metaFilenamePb, compactedMetaFilenamePb)
	if err != nil {
		level.Error(log.Logger).Log("msg", "error copying obj meta.pb to compacted.pb, is this block from previous Tempo version?", "err", err)
	}

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

func (rw *Backend) CompactedBlockMeta(blockID uuid.UUID, tenantID string) (*backend.CompactedBlockMeta, error) {
	outPb, err := rw.compactedBlockMetaPb(blockID, tenantID)
	if err == nil {
		return outPb, nil
	}

	// TODO: record a note about fallback

	filename := rw.compactedMetaFileName(blockID, tenantID)

	fi, err := os.Stat(filename)
	if err != nil {
		return nil, readError(err)
	}

	bytes, err := os.ReadFile(filename)
	if err != nil {
		return nil, readError(err)
	}

	out := &backend.CompactedBlockMeta{}
	err = json.Unmarshal(bytes, out)
	if err != nil {
		return nil, err
	}
	out.CompactedTime = fi.ModTime()

	return out, nil
}

func (rw *Backend) compactedBlockMetaPb(blockID uuid.UUID, tenantID string) (*backend.CompactedBlockMeta, error) {
	filename := rw.compactedMetaFileNamePb(blockID, tenantID)
	fi, err := os.Stat(filename)
	if err != nil {
		return nil, readError(err)
	}

	bytes, err := os.ReadFile(filename)
	if err != nil {
		return nil, readError(err)
	}

	out := &backend.CompactedBlockMeta{}
	err = out.Unmarshal(bytes)
	if err != nil {
		return nil, err
	}
	out.CompactedTime = fi.ModTime()

	return out, nil
}

func (rw *Backend) compactedMetaFileName(blockID uuid.UUID, tenantID string) string {
	return path.Join(rw.rootPath(backend.KeyPathForBlock(blockID, tenantID)), backend.CompactedMetaName)
}

func (rw *Backend) compactedMetaFileNamePb(blockID uuid.UUID, tenantID string) string {
	return path.Join(rw.rootPath(backend.KeyPathForBlock(blockID, tenantID)), backend.CompactedMetaNamePb)
}
