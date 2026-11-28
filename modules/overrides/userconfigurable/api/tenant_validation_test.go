package api

import (
	"context"
	"errors"
	"testing"

	"github.com/grafana/tempo/v3/modules/overrides/userconfigurable/client"
	"github.com/stretchr/testify/require"
)

// Tenant-aware validation must run before any write and receive the target
// tenant, so inherited limits cannot be validated against another tenant.
func TestSetUsesTenantValidation(t *testing.T) {
	validator := &tenantValidationRecorder{}
	api := &UserConfigOverridesAPI{validator: validator}
	_, err := api.set(context.Background(), "target-tenant", &client.Limits{}, "", false)
	require.ErrorContains(t, err, "inherited rules rejected")
	require.Equal(t, "target-tenant", validator.tenant)
	require.False(t, validator.globalCalled)
}

type tenantValidationRecorder struct {
	tenant       string
	globalCalled bool
}

func (v *tenantValidationRecorder) Validate(*client.Limits) error {
	v.globalCalled = true
	return nil
}

func (v *tenantValidationRecorder) ValidateForTenant(tenant string, _ *client.Limits) error {
	v.tenant = tenant
	return errors.New("inherited rules rejected")
}
