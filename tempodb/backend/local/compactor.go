package local

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

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
		return errors.New("empty tenant id")
	}

	if blockID == uuid.Nil {
		return errors.New("empty block id")
	}

	p := rw.rootPath(backend.KeyPathForBlock(blockID, tenantID))
	err := os.RemoveAll(p)
	if err != nil {
		return fmt.Errorf("failed to remove keypath for block %s: %w", p, err)
	}

	return nil
}

// TombstoneBlock renames meta.json → meta.deleted.json. Hides the block from
// BlockMeta lookups while data files remain until ClearBlock. Idempotent:
// a missing meta.json returns nil.
func (rw *Backend) TombstoneBlock(blockID uuid.UUID, tenantID string) error {
	if len(tenantID) == 0 {
		return errors.New("empty tenant id")
	}
	if blockID == uuid.Nil {
		return errors.New("empty block id")
	}
	keypath := rw.rootPath(backend.KeyPathForBlock(blockID, tenantID))
	from := filepath.Join(keypath, backend.MetaName)
	to := filepath.Join(keypath, backend.DeletedMetaName)
	if err := os.Rename(from, to); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to tombstone block %s: %w", blockID, err)
	}
	return nil
}

// ClearTombstonedBlocks removes block dirs containing meta.deleted.json.
// Use at startup to clean up after a crash between TombstoneBlock and
// ClearBlock. Returns the number of dirs removed.
func (rw *Backend) ClearTombstonedBlocks() (int, error) {
	root := rw.cfg.Path
	cleared := 0
	tenants, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("read tenants: %w", err)
	}
	for _, tenant := range tenants {
		if !tenant.IsDir() {
			continue
		}
		tenantPath := filepath.Join(root, tenant.Name())
		blocks, err := os.ReadDir(tenantPath)
		if err != nil {
			return cleared, fmt.Errorf("read tenant %s: %w", tenant.Name(), err)
		}
		for _, b := range blocks {
			if !b.IsDir() {
				continue
			}
			marker := filepath.Join(tenantPath, b.Name(), backend.DeletedMetaName)
			if _, err := os.Stat(marker); err != nil {
				if os.IsNotExist(err) {
					continue
				}
				return cleared, fmt.Errorf("stat marker %s: %w", marker, err)
			}
			blockDir := filepath.Join(tenantPath, b.Name())
			if err := os.RemoveAll(blockDir); err != nil {
				return cleared, fmt.Errorf("remove tombstoned block %s: %w", blockDir, err)
			}
			cleared++
		}
	}
	return cleared, nil
}

func (rw *Backend) CompactedBlockMeta(blockID uuid.UUID, tenantID string) (*backend.CompactedBlockMeta, error) {
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

func (rw *Backend) compactedMetaFileName(blockID uuid.UUID, tenantID string) string {
	return filepath.Join(rw.rootPath(backend.KeyPathForBlock(blockID, tenantID)), backend.CompactedMetaName)
}
