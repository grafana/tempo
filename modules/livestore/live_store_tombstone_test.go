package livestore

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/grafana/dskit/services"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLiveStoreClearsTombstonedBlocksOnStartup verifies the crash-recovery
// claim from PR #7132: a process that crashed between Tombstone() and the
// quarantine reclaim leaves dirs containing meta.deleted.json, and on the
// next startup reloadBlocks must sweep them via wal.ClearTombstonedBlocks
// and local.Backend.ClearTombstonedBlocks before the rescan.
//
// We don't go through the LiveStore API to produce the tombstones — that
// would require driving a full block lifecycle. Instead we plant the
// on-disk shape that a crash leaves behind and assert startup reclaims it.
func TestLiveStoreClearsTombstonedBlocksOnStartup(t *testing.T) {
	tmpDir := t.TempDir()

	// --- WAL layer ---
	// wal.WAL.Filepath = tmpDir; wal block dirs are direct children. A
	// tombstoned dir has meta.deleted.json (and meta.json gone). A "bare"
	// dir with no marker must remain untouched after the sweep.
	tombWAL := filepath.Join(tmpDir, uuid.NewString()+"+fake+v5")
	bareWAL := filepath.Join(tmpDir, uuid.NewString()+"+fake+v5")
	require.NoError(t, os.MkdirAll(tombWAL, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(tombWAL, backend.DeletedMetaName), []byte(`{}`), 0o600))
	require.NoError(t, os.MkdirAll(bareWAL, 0o700))

	// --- Complete-block layer ---
	// local.Backend roots at <wal>/blocks/<tenant>/<blockID>/. A tombstoned
	// complete block has meta.deleted.json under its block dir.
	const tenant = "fake"
	blocksRoot := filepath.Join(tmpDir, "blocks")
	tombCBID := uuid.New()
	tombCBDir := filepath.Join(blocksRoot, tenant, tombCBID.String())
	require.NoError(t, os.MkdirAll(tombCBDir, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(tombCBDir, backend.DeletedMetaName), []byte(`{}`), 0o600))

	// Sanity: the tombstones are present before startup.
	_, err := os.Stat(filepath.Join(tombWAL, backend.DeletedMetaName))
	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(tombCBDir, backend.DeletedMetaName))
	require.NoError(t, err)

	// Boot the LiveStore — reloadBlocks runs as part of starting() and
	// should clear both tombstones before scanning.
	liveStore, err := defaultLiveStore(t, tmpDir)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = services.StopAndAwaitTerminated(t.Context(), liveStore)
	})

	// Tombstoned WAL dir is gone.
	_, err = os.Stat(tombWAL)
	assert.True(t, os.IsNotExist(err), "tombstoned WAL dir must be reclaimed on startup")

	// Bare WAL dir without a marker is untouched (sweep must not over-reach).
	_, err = os.Stat(bareWAL)
	assert.NoError(t, err, "WAL dir without meta.deleted.json must not be removed")

	// Tombstoned complete-block dir is gone.
	_, err = os.Stat(tombCBDir)
	assert.True(t, os.IsNotExist(err), "tombstoned complete-block dir must be reclaimed on startup")

	// No instance was created — only tombstoned blocks were on disk.
	_, ok := liveStore.instances[tenant]
	assert.False(t, ok, "no instance should be created from tombstoned-only state")
}
