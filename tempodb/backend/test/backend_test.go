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

	// Create the fixtures, but commit them.
	// metas := []*backend.BlockMeta{
	// 	backend.NewBlockMeta(tenant, uuid.New().UUID, "v1", backend.EncGZIP, "adsf"),
	// 	backend.NewBlockMeta(tenant, uuid.New().UUID, "v2", backend.EncNone, "adsf"),
	// 	backend.NewBlockMeta(tenant, uuid.New().UUID, "v3", backend.EncLZ4_4M, "adsf"),
	// }

	rr, rw, _, err := local.New(&local.Config{
		Path: "./test-data",
	})
	require.NoError(t, err)

	var (
		_ = backend.NewWriter(rw)
		r = backend.NewReader(rr)
	)

	_, err = r.TenantIndex(ctx, tenant)
	assert.NoError(t, err)

	// for _, meta := range metas {
	// 	err = w.WriteBlockMeta(ctx, meta)
	// 	require.NoError(t, err)
	// }

	metas, compactedMetas, err := rr.ListBlocks(ctx, tenant)
	require.NoError(t, err)
	require.Len(t, compactedMetas, 0)

	blockMetas := make([]*backend.BlockMeta, 0, len(metas))
	for _, u := range metas {
		meta, e := r.BlockMeta(ctx, u, tenant)
		require.NoError(t, e)
		blockMetas = append(blockMetas, meta)
	}

	// err = backend.NewWriter(rw).WriteTenantIndex(ctx, tenant, blockMetas, nil)
	// require.NoError(t, err)

	// for _, meta := range metas {
	// 	m, e := r.BlockMeta(ctx, meta.BlockID.UUID, tenant)
	// 	require.NoError(t, e)
	// 	require.Equal(t, meta, m)
	// }

	var i *backend.TenantIndex
	i, err = r.TenantIndex(ctx, tenant)
	require.NoError(t, err)
	require.Equal(t, blockMetas, i.Meta)
	require.Len(t, i.Meta, len(metas))
}
