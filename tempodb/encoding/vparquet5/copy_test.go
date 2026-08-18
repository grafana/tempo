package vparquet5

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	tempo_io "github.com/grafana/tempo/pkg/io"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/local"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

// TestCopyBlockPreservesNoCompactFlag verifies that CopyBlock propagates the
// no-compact flag from the source block to the destination. The block-builder
// writes a block with this flag set and then copies it to the backend; the flag
// must survive the copy so the block is not compacted or polled before the
// block-builder calls AllowCompaction.
func TestCopyBlockPreservesNoCompactFlag(t *testing.T) {
	ctx := context.Background()

	rawR, rawW, _, err := local.New(&local.Config{Path: t.TempDir()})
	require.NoError(t, err)
	r := backend.NewReader(rawR)
	w := backend.NewWriter(rawW)

	cfg := &common.BlockConfig{
		BloomFP:             0.01,
		BloomShardSizeBytes: 100 * 1024,
	}

	// Build a source block.
	fromMeta := backend.NewBlockMeta("fake", uuid.New(), VersionString)
	fromMeta.TotalObjects = 1

	s, fromMeta := newStreamingBlock(ctx, cfg, fromMeta, r, w, tempo_io.NewBufferedWriter)

	id := test.ValidTraceID(nil)
	tr, _ := traceToParquet(&backend.BlockMeta{}, id, test.MakeTrace(1, id), nil)
	require.NoError(t, s.Add(tr, 0, 0))
	_, err = s.Complete()
	require.NoError(t, err)

	// Simulate the block-builder writing the no-compact flag on the source block.
	require.NoError(t, w.WriteNoCompactFlag(ctx, uuid.UUID(fromMeta.BlockID), fromMeta.TenantID))

	// Copy the block to a new location.
	toMeta := backend.NewBlockMeta(fromMeta.TenantID, uuid.New(), VersionString)
	toMeta.TotalObjects = fromMeta.TotalObjects
	require.NoError(t, CopyBlock(ctx, fromMeta, toMeta, r, w))

	// The flag must be preserved on the destination block.
	has, err := r.HasNoCompactFlag(ctx, uuid.UUID(toMeta.BlockID), toMeta.TenantID)
	require.NoError(t, err)
	require.True(t, has, "CopyBlock must preserve the no-compact flag")
}
