package local

import (
	"encoding/json"
	"errors"
	"fmt"
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

// TombstoneBlock atomically renames the block's meta.json to meta.deleted.json.
// After this call the block is invisible to BlockMeta lookups (which return
// ErrDoesNotExist) but data files remain on disk until ClearBlock is called.
// Crash-safe: a process that dies between TombstoneBlock and ClearBlock will
// see the tombstoned dir on restart and ClearBlock can finish the cleanup.
func (rw *Backend) TombstoneBlock(blockID uuid.UUID, tenantID string) error {
	if len(tenantID) == 0 {
		return errors.New("empty tenant id")
	}
	if blockID == uuid.Nil {
		return errors.New("empty block id")
	}
	keypath := rw.rootPath(backend.KeyPathForBlock(blockID, tenantID))
	from := path.Join(keypath, backend.MetaName)
	to := path.Join(keypath, backend.DeletedMetaName)
	if err := os.Rename(from, to); err != nil {
		return fmt.Errorf("failed to tombstone block %s: %w", blockID, err)
	}
	return nil
}

// ClearTombstonedBlocks walks the local backend root and removes any block
// directory containing meta.deleted.json. Use at startup to reclaim blocks
// that were tombstoned but not Cleared by a prior process (e.g. crash between
// TombstoneBlock and ClearBlock). Returns the number of dirs removed.
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
		tenantPath := path.Join(root, tenant.Name())
		blocks, err := os.ReadDir(tenantPath)
		if err != nil {
			continue
		}
		for _, b := range blocks {
			if !b.IsDir() {
				continue
			}
			marker := path.Join(tenantPath, b.Name(), backend.DeletedMetaName)
			if _, err := os.Stat(marker); err != nil {
				continue
			}
			blockDir := path.Join(tenantPath, b.Name())
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
	return path.Join(rw.rootPath(backend.KeyPathForBlock(blockID, tenantID)), backend.CompactedMetaName)
}
