package client

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/v2/tempodb/backend"
	"github.com/grafana/tempo/v2/tempodb/backend/local"
)

func TestUserConfigOverridesClient(t *testing.T) {
	// Other backends are tested through the integration tests

	ctx := context.Background()
	tenant := "foo"
	dir := t.TempDir()

	cfg := &Config{
		Backend: backend.Local,
		Local: &local.Config{
			Path: dir,
		},
	}

	client, err := New(cfg)
	require.NoError(t, err)

	// List
	list, err := client.List(ctx)
	assert.NoError(t, err)
	assert.Empty(t, list)

	// Set
	limits := &Limits{
		Forwarders: &[]string{"my-forwarder"},
	}
	_, err = client.Set(ctx, tenant, limits, "")
	assert.NoError(t, err)

	// Get
	retrievedLimits, _, err := client.Get(ctx, tenant)
	assert.NoError(t, err)
	assert.Equal(t, limits, retrievedLimits)

	// List
	list, err = client.List(ctx)
	assert.NoError(t, err)
	assert.Equal(t, []string{tenant}, list)

	// Delete
	assert.NoError(t, client.Delete(ctx, tenant, ""))

	// Get - does not exist
	_, _, err = client.Get(ctx, tenant)
	assert.ErrorIs(t, err, backend.ErrDoesNotExist)
}
