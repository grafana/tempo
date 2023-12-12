package local

import (
	"bytes"
	"context"
	crand "crypto/rand"
	"fmt"
	"math/rand"
	"os"
	"testing"

	"github.com/grafana/tempo/pkg/io"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/tempodb/backend"
)

const objectName = "test"

func TestReadWrite(t *testing.T) {
	fakeTracesFile, err := os.CreateTemp("/tmp", "")
	defer os.Remove(fakeTracesFile.Name())
	assert.NoError(t, err, "unexpected error creating temp file")

	r, w, _, err := New(&Config{
		Path: t.TempDir(),
	})
	assert.NoError(t, err, "unexpected error creating local backend")

	blockID := uuid.New()
	tenantIDs := []string{"fake"}

	for i := 0; i < 10; i++ {
		tenantIDs = append(tenantIDs, fmt.Sprintf("%d", rand.Int()))
	}

	fakeMeta := &backend.BlockMeta{
		BlockID: blockID,
	}

	fakeObject := make([]byte, 20)

	_, err = crand.Read(fakeObject)
	assert.NoError(t, err, "unexpected error creating fakeObject")

	ctx := context.Background()
	for _, id := range tenantIDs {
		fakeMeta.TenantID = id
		err = w.Write(ctx, objectName, backend.KeyPathForBlock(fakeMeta.BlockID, id), bytes.NewReader(fakeObject), int64(len(fakeObject)), nil)
		assert.NoError(t, err, "unexpected error writing")

		err = w.Write(ctx, backend.MetaName, backend.KeyPathForBlock(fakeMeta.BlockID, id), bytes.NewReader(fakeObject), int64(len(fakeObject)), nil)
		assert.NoError(t, err, "unexpected error meta.json")
		err = w.Write(ctx, backend.CompactedMetaName, backend.KeyPathForBlock(fakeMeta.BlockID, id), bytes.NewReader(fakeObject), int64(len(fakeObject)), nil)
		assert.NoError(t, err, "unexpected error meta.compacted.json")
	}

	actualObject, size, err := r.Read(ctx, objectName, backend.KeyPathForBlock(blockID, tenantIDs[0]), nil)
	assert.NoError(t, err, "unexpected error reading")
	actualObjectBytes, err := io.ReadAllWithEstimate(actualObject, size)
	assert.NoError(t, err, "unexpected error reading")
	assert.Equal(t, fakeObject, actualObjectBytes)

	actualReadRange := make([]byte, 5)
	err = r.ReadRange(ctx, objectName, backend.KeyPathForBlock(blockID, tenantIDs[0]), 5, actualReadRange, nil)
	assert.NoError(t, err, "unexpected error range")
	assert.Equal(t, fakeObject[5:10], actualReadRange)

	list, err := r.List(ctx, backend.KeyPath{tenantIDs[0]})
	assert.NoError(t, err, "unexpected error listing")
	assert.Len(t, list, 1)
	assert.Equal(t, blockID.String(), list[0])

	m, cm, err := r.ListBlocks(ctx, tenantIDs[0])
	assert.NoError(t, err, "unexpected error listing blocks")
	assert.Len(t, m, 1)
	assert.Len(t, cm, 1)
}

func TestShutdownLeavesTenantsWithBlocks(t *testing.T) {
	r, w, _, err := New(&Config{
		Path: t.TempDir(),
	})
	require.NoError(t, err)

	ctx := context.Background()
	blockID := uuid.New()
	contents := bytes.NewReader([]byte("test"))
	tenant := "fake"

	// write a "block"
	err = w.Write(ctx, "test", backend.KeyPathForBlock(blockID, tenant), contents, contents.Size(), nil)
	require.NoError(t, err)

	tenantExists(t, tenant, r)
	blockExists(t, blockID, tenant, r)

	// shutdown the backend
	r.Shutdown()

	tenantExists(t, tenant, r)
	blockExists(t, blockID, tenant, r)
}

func TestShutdownRemovesTenantsWithoutBlocks(t *testing.T) {
	r, w, c, err := New(&Config{
		Path: t.TempDir(),
	})
	require.NoError(t, err)

	ctx := context.Background()
	blockID := uuid.New()
	contents := bytes.NewReader([]byte("test"))
	tenant := "tenant"

	// write a "block"
	err = w.Write(ctx, "test", backend.KeyPathForBlock(blockID, tenant), contents, contents.Size(), nil)
	require.NoError(t, err)

	tenantExists(t, tenant, r)
	blockExists(t, blockID, tenant, r)

	// clear the block
	err = c.ClearBlock(blockID, tenant)
	require.NoError(t, err)

	tenantExists(t, tenant, r)

	// block should not exist
	blocks, err := r.List(ctx, backend.KeyPath{tenant})
	require.NoError(t, err)
	require.Len(t, blocks, 0)

	// shutdown the backend
	r.Shutdown()

	// tenant should not exist
	tenants, err := r.List(ctx, backend.KeyPath{})
	require.NoError(t, err)
	require.Len(t, tenants, 0)
}

func tenantExists(t *testing.T, tenant string, r backend.RawReader) {
	tenants, err := r.List(context.Background(), backend.KeyPath{})
	require.NoError(t, err)
	require.Len(t, tenants, 1)
	require.Equal(t, tenant, tenants[0])
}

func blockExists(t *testing.T, blockID uuid.UUID, tenant string, r backend.RawReader) {
	blocks, err := r.List(context.Background(), backend.KeyPath{tenant})
	require.NoError(t, err)
	require.Len(t, blocks, 1)
	require.Equal(t, blockID.String(), blocks[0])
}
