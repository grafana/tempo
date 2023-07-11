package userconfigurableapi

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/tempodb/backend/local"
)

func TestUserConfigOverridesClient(t *testing.T) {
	// Other backends are tested through the integration tests

	ctx := context.Background()
	tenant := "foo"
	dir := t.TempDir()

	cfg := &UserConfigurableOverridesClientConfig{
		Backend: "local",
		Local: &local.Config{
			Path: dir,
		},
	}

	client, err := NewUserConfigOverridesClient(cfg)
	require.NoError(t, err)

	// List
	list, err := client.List(ctx)
	assert.NoError(t, err)
	assert.Empty(t, list)

	// Set
	limits := &UserConfigurableLimits{
		Version:    "v1",
		Forwarders: &[]string{"my-forwarder"},
	}
	assert.NoError(t, client.Set(ctx, tenant, limits))

	// Get
	retrievedLimits, err := client.Get(ctx, tenant)
	assert.NoError(t, err)
	assert.Equal(t, limits, retrievedLimits)

	// List
	list, err = client.List(ctx)
	assert.NoError(t, err)
	assert.Equal(t, []string{tenant}, list)

	// Delete
	assert.NoError(t, client.Delete(ctx, tenant))

	// Get - does not exist
	retrievedLimits, err = client.Get(ctx, tenant)
	assert.NoError(t, err)
	assert.Nil(t, retrievedLimits)

	// List - should be empty
	list, err = client.List(ctx)
	assert.NoError(t, err)
	assert.Empty(t, list)
}
