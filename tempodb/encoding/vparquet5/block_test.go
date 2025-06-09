package vparquet5

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/local"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/stretchr/testify/require"
)

func TestValidateFailsOnCorruptParquetFile(t *testing.T) {
	ctx := context.Background()
	block, w := validBlock(t)
	meta := block.meta

	err := block.Validate(ctx)
	require.NoError(t, err)

	// Corrupt the file
	err = w.Write(ctx, DataFileName, uuid.UUID(meta.BlockID), meta.TenantID, []byte{0, 0, 0, 0, 0, 0, 0, 0}, nil)
	require.NoError(t, err)

	err = block.Validate(ctx)
	require.Error(t, err)
}

func TestValidateFailsOnMissingBloom(t *testing.T) {
	ctx := context.Background()
	block, w := validBlock(t)
	meta := block.meta

	err := block.Validate(ctx)
	require.NoError(t, err)

	// remove a bloom
	err = w.Delete(ctx, common.BloomName(0), backend.KeyPathForBlock(uuid.UUID(meta.BlockID), meta.TenantID))
	require.NoError(t, err)

	err = block.Validate(ctx)
	require.Error(t, err)
}

func validBlock(t *testing.T) (*backendBlock, backend.Writer) {
	t.Helper()

	ctx := context.Background()

	rawR, rawW, _, err := local.New(&local.Config{
		Path: t.TempDir(),
	})
	require.NoError(t, err)

	r := backend.NewReader(rawR)
	w := backend.NewWriter(rawW)

	iter := newTestIterator()

	iter.Add(test.MakeTrace(10, nil), 100, 401)
	iter.Add(test.MakeTrace(10, nil), 101, 402)
	iter.Add(test.MakeTrace(10, nil), 102, 403)

	cfg := &common.BlockConfig{
		BloomFP:             0.01,
		BloomShardSizeBytes: 100 * 1024,
	}

	meta := backend.NewBlockMeta("fake", uuid.New(), VersionString, backend.EncNone, "")
	meta.TotalObjects = 1
	meta.StartTime = time.Unix(300, 0)
	meta.EndTime = time.Unix(305, 0)

	outMeta, err := CreateBlock(ctx, cfg, meta, iter, r, w)
	require.NoError(t, err)

	return newBackendBlock(outMeta, r), w
}
