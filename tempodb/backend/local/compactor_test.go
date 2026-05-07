package local

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/tempodb/backend"
)

func TestTombstoneBlock_renamesMeta(t *testing.T) {
	dir := t.TempDir()
	_, w, c, err := New(&Config{Path: dir})
	require.NoError(t, err)

	ctx := context.Background()
	blockID := uuid.UUID(backend.NewUUID())
	tenant := "fake"

	// Seed a meta.json so TombstoneBlock has something to rename.
	require.NoError(t, w.Write(ctx, backend.MetaName,
		backend.KeyPathForBlock(blockID, tenant), bytes.NewReader([]byte(`{}`)), 2, nil))

	require.NoError(t, c.(*Backend).TombstoneBlock(blockID, tenant))

	keypath := filepath.Join(dir, backend.KeyPathForBlock(blockID, tenant)[0], backend.KeyPathForBlock(blockID, tenant)[1])
	_, err = os.Stat(filepath.Join(keypath, backend.MetaName))
	assert.True(t, os.IsNotExist(err), "meta.json should be gone")
	_, err = os.Stat(filepath.Join(keypath, backend.DeletedMetaName))
	assert.NoError(t, err, "meta.deleted.json should be present")
}

func TestClearTombstonedBlocks_reclaimsTombstonedDirs(t *testing.T) {
	dir := t.TempDir()
	_, w, c, err := New(&Config{Path: dir})
	require.NoError(t, err)
	be := c.(*Backend)

	ctx := context.Background()
	tenant := "fake"

	// Three blocks: one tombstoned, one alive, one already-cleared (no marker).
	tombstonedID := uuid.UUID(backend.NewUUID())
	aliveID := uuid.UUID(backend.NewUUID())
	emptyID := uuid.UUID(backend.NewUUID())

	for _, id := range []uuid.UUID{tombstonedID, aliveID} {
		require.NoError(t, w.Write(ctx, backend.MetaName,
			backend.KeyPathForBlock(id, tenant), bytes.NewReader([]byte(`{}`)), 2, nil))
	}
	// emptyID directory exists but has no meta files.
	require.NoError(t, os.MkdirAll(filepath.Join(dir, tenant, emptyID.String()), 0o755))

	// Tombstone the tombstonedID.
	require.NoError(t, be.TombstoneBlock(tombstonedID, tenant))

	cleared, err := be.ClearTombstonedBlocks()
	require.NoError(t, err)
	assert.Equal(t, 1, cleared, "exactly one tombstoned block should have been cleared")

	// tombstonedID dir is gone.
	_, err = os.Stat(filepath.Join(dir, tenant, tombstonedID.String()))
	assert.True(t, os.IsNotExist(err))

	// aliveID dir still has meta.json.
	_, err = os.Stat(filepath.Join(dir, tenant, aliveID.String(), backend.MetaName))
	assert.NoError(t, err)

	// emptyID dir untouched (no marker, not removed).
	_, err = os.Stat(filepath.Join(dir, tenant, emptyID.String()))
	assert.NoError(t, err)
}

func TestClearTombstonedBlocks_emptyRoot(t *testing.T) {
	_, _, c, err := New(&Config{Path: t.TempDir()})
	require.NoError(t, err)
	cleared, err := c.(*Backend).ClearTombstonedBlocks()
	require.NoError(t, err)
	assert.Equal(t, 0, cleared)
}
