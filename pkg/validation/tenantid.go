package validation

import (
	"context"
	"strings"

	"github.com/grafana/dskit/tenant"
	"github.com/grafana/dskit/user"
)

// tenantIDsSeparator separates the tenants of a multi-tenant org ID ("tenant-a|tenant-b").
const tenantIDsSeparator = "|"

// ExtractValidTenantID returns the non-empty tenant ID from the context, with any tenant
// metadata ("tenant:key=value") stripped.
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

// ExtractTenantWithMetadata returns the non-empty tenant ID and the tenant metadata from the context.
// The org ID may carry metadata after the tenant ID as "tenant:key=value[:key2=value2]", with keys
// sorted and unique. Malformed metadata is an error.
func ExtractTenantWithMetadata(ctx context.Context) (string, tenant.Metadata, error) {
	tenantID, md, err := tenant.ExtractWithMetadata(ctx)
	if err != nil {
		return "", tenant.Metadata{}, err
	}
	if tenantID == "" {
		return "", tenant.Metadata{}, user.ErrNoOrgID
	}
	return tenantID, md, nil
}

// ExtractValidTenantIDs is like ExtractValidTenantID but also accepts the multi-tenant org ID of
// cross-tenant queries ("tenant-a|tenant-b"), stripping the metadata from every tenant.
func ExtractValidTenantIDs(ctx context.Context) (string, error) {
	orgID, err := user.ExtractOrgID(ctx)
	if err != nil {
		return "", err
	}

	tenantIDs := strings.Split(orgID, tenantIDsSeparator)
	for i, tenantID := range tenantIDs {
		tenantID = tenant.TrimMetadata(tenantID)
		if tenantID == "" {
			return "", user.ErrNoOrgID
		}
		if err := tenant.ValidTenantID(tenantID); err != nil {
			return "", err
		}
		tenantIDs[i] = tenantID
	}
	return strings.Join(tenantIDs, tenantIDsSeparator), nil
}
