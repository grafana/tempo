package test

import (
	"context"
	"testing"

	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/local"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFixtures(t *testing.T) {
	var (
		tenant = "single-tenant"
		ctx    = context.Background()
	)

	rr, rw, rc, err := local.New(&local.Config{
		Path: "./test-data",
	})
	require.NoError(t, err)

	var (
		_ = backend.NewWriter(rw)
		r = backend.NewReader(rr)
	)

	_, err = r.TenantIndex(ctx, tenant)
	assert.NoError(t, err)

	// To regenerate the fixtures, uncomment the write path.
	// metas := []*backend.BlockMeta{
	// 	backend.NewBlockMeta(tenant, uuid.New(), "v1", backend.EncGZIP, "adsf"),
	// 	backend.NewBlockMeta(tenant, uuid.New(), "v2", backend.EncNone, "adsf"),
	// 	backend.NewBlockMeta(tenant, uuid.New(), "v3", backend.EncLZ4_4M, "adsf"),
	// 	backend.NewBlockMeta(tenant, uuid.New(), "v4", backend.EncLZ4_1M, "adsf"),
	// }
	//
	// for _, meta := range metas {
	// 	err = w.WriteBlockMeta(ctx, meta)
	// 	require.NoError(t, err)
	// }
	//
	// err = rc.MarkBlockCompacted((uuid.UUID)(metas[0].BlockID), tenant)
	// assert.NoError(t, err)

	listMetas, listCompactedMetas, err := rr.ListBlocks(ctx, tenant)
	require.NoError(t, err)
	require.Len(t, listCompactedMetas, 1)

	for _, v := range listMetas {
		t.Logf("listMetas: %v", v)
	}

	blockMetas := make([]*backend.BlockMeta, 0, len(listMetas))
	for _, u := range listMetas {
		m, e := r.BlockMeta(ctx, u, tenant)
		require.NoError(t, e)
		blockMetas = append(blockMetas, m)
		assert.Equal(t, tenant, m.TenantID)
	}

	compactedBlockMetas := make([]*backend.CompactedBlockMeta, 0, len(listCompactedMetas))
	for _, u := range listCompactedMetas {
		m, e := rc.CompactedBlockMeta(u, tenant)
		assert.NoError(t, e)
		compactedBlockMetas = append(compactedBlockMetas, m)
		assert.Equal(t, tenant, m.TenantID)
	}

	nonZeroMeta(t, blockMetas)
	nonZeroCompactedMeta(t, compactedBlockMetas)

	// err = backend.NewWriter(rw).WriteTenantIndex(ctx, tenant, blockMetas, compactedBlockMetas)
	// require.NoError(t, err)

	// for _, meta := range metas {
	// 	m, e := r.BlockMeta(ctx, meta.BlockID.UUID, tenant)
	// 	require.NoError(t, e)
	// 	require.Equal(t, meta, m)
	// }

	var i *backend.TenantIndex
	i, err = r.TenantIndex(ctx, tenant)
	require.NoError(t, err)
	require.Equal(t, blockMetas, i.Metas)
	require.Len(t, i.Metas, len(listMetas))
}

func nonZeroMeta(t *testing.T, m []*backend.BlockMeta) {
	for _, v := range m {
		assert.NotZero(t, v.BlockID, "blockid is zero, id: %v", v.BlockID)
	}
}

func nonZeroCompactedMeta(t *testing.T, m []*backend.CompactedBlockMeta) {
	for _, v := range m {
		assert.NotZero(t, v.BlockID, "blockid is zero, id: %v", v.BlockID)
	}
}
