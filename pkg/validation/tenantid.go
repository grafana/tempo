package validation

import (
	"context"

	"github.com/grafana/dskit/tenant"
	"github.com/grafana/dskit/user"
)

func ExtractValidTenantID(ctx context.Context) (string, error) {
	tenantID, err := tenant.TenantID(ctx)
	if err != nil {
		return "", err
	}
	if tenantID == "" {
		return "", user.ErrNoOrgID
	}
	return tenantID, nil
}
