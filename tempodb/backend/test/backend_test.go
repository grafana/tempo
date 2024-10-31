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
	require.Equal(t, blockMetas, i.Meta)
	require.Len(t, i.Meta, len(listMetas))
}

func TestOriginalFixtures(t *testing.T) {
	var (
		tenant = "3" // A sample index
		ctx    = context.Background()
	)

	rr, rw, _, err := local.New(&local.Config{
		Path: "./test-data",
	})
	require.NoError(t, err)

	var (
		_ = backend.NewWriter(rw)
		r = backend.NewReader(rr)
	)

	expectedDedicatedColumns := backend.DedicatedColumns{
		backend.DedicatedColumn{Scope: "span", Name: "db.statement", Type: "string"},
		backend.DedicatedColumn{Scope: "span", Name: "component", Type: "string"},
		backend.DedicatedColumn{Scope: "span", Name: "http.user_agent", Type: "string"},
		backend.DedicatedColumn{Scope: "span", Name: "otel.library.name", Type: "string"},
		backend.DedicatedColumn{Scope: "span", Name: "db.connection_string", Type: "string"},
		backend.DedicatedColumn{Scope: "span", Name: "organization", Type: "string"},
		backend.DedicatedColumn{Scope: "span", Name: "peer.address", Type: "string"},
		backend.DedicatedColumn{Scope: "span", Name: "net.peer.name", Type: "string"},
		backend.DedicatedColumn{Scope: "span", Name: "blockID", Type: "string"},
		backend.DedicatedColumn{Scope: "span", Name: "db.name", Type: "string"},
		backend.DedicatedColumn{Scope: "resource", Name: "host.name", Type: "string"},
		backend.DedicatedColumn{Scope: "resource", Name: "opencensus.exporterversion", Type: "string"},
		backend.DedicatedColumn{Scope: "resource", Name: "client-uuid", Type: "string"},
		backend.DedicatedColumn{Scope: "resource", Name: "ip", Type: "string"},
		backend.DedicatedColumn{Scope: "resource", Name: "database", Type: "string"},
		backend.DedicatedColumn{Scope: "resource", Name: "os.description", Type: "string"},
		backend.DedicatedColumn{Scope: "resource", Name: "process.runtime.description", Type: "string"},
		backend.DedicatedColumn{Scope: "resource", Name: "container.id", Type: "string"},
		backend.DedicatedColumn{Scope: "resource", Name: "slug", Type: "string"},
		backend.DedicatedColumn{Scope: "resource", Name: "module.path", Type: "string"},
	}

	i, err := r.TenantIndex(ctx, tenant)
	assert.NoError(t, err)
	assert.NotNil(t, i)
	assert.NotZero(t, i.CreatedAt)

	assert.Equal(t, 22435, len(i.Meta))
	assert.Equal(t, 3264, len(i.CompactedMeta))

	nonZeroMeta(t, i.Meta)
	for _, v := range i.Meta {
		assert.Equal(t, tenant, v.TenantID)
		assert.Equal(t, "vParquet4", v.Version)
		assert.NotZero(t, v.StartTime)
		assert.NotZero(t, v.EndTime)
		assert.Equal(t, 20, len(v.DedicatedColumns))
		assert.Equal(t, expectedDedicatedColumns, v.DedicatedColumns)
	}

	nonZeroCompactedMeta(t, i.CompactedMeta)
	for _, v := range i.CompactedMeta {
		assert.Equal(t, "vParquet4", v.Version)
		assert.Equal(t, tenant, v.TenantID)
		assert.NotZero(t, v.CompactedTime)
		assert.NotZero(t, v.StartTime)
		assert.NotZero(t, v.EndTime)
		assert.Equal(t, 20, len(v.DedicatedColumns))
		assert.Equal(t, expectedDedicatedColumns, v.DedicatedColumns)
	}
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
