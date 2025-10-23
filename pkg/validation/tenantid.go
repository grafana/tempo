package validation

import (
	"github.com/grafana/dskit/tenant"
	"github.com/grafana/dskit/user"
)

func TempoValidTenantID(tenantID string) error {
	if tenantID == "" {
		return user.ErrNoOrgID
	}
	return tenant.ValidTenantID(tenantID)
}
