package validation

import (
	"context"
	"testing"

	"github.com/grafana/dskit/user"
	"github.com/stretchr/testify/require"
)

func TestExtractTenantWithMetadata(t *testing.T) {
	tcs := []struct {
		name         string
		orgID        string
		noOrgID      bool
		wantTenant   string
		wantMetadata string
		wantErr      bool
	}{
		{name: "plain tenant", orgID: "test", wantTenant: "test", wantMetadata: ""},
		{name: "tenant with metadata", orgID: "test:source=foo", wantTenant: "test", wantMetadata: ":source=foo"},
		{name: "tenant with sorted metadata pairs", orgID: "test:env=prod:source=foo", wantTenant: "test", wantMetadata: ":env=prod:source=foo"},
		{name: "metadata without key-value separator", orgID: "test:garbage", wantErr: true},
		{name: "unsorted metadata keys", orgID: "test:source=foo:env=prod", wantErr: true},
		{name: "metadata with unsupported character", orgID: "test:source=a/b", wantErr: true},
		{name: "empty tenant", orgID: "", wantErr: true},
		{name: "empty tenant with metadata", orgID: ":source=foo", wantErr: true},
		{name: "multiple tenants", orgID: "a|b", wantErr: true},
		{name: "no org ID", noOrgID: true, wantErr: true},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			if !tc.noOrgID {
				ctx = user.InjectOrgID(ctx, tc.orgID)
			}

			tenantID, md, err := ExtractTenantWithMetadata(ctx)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.wantTenant, tenantID)
			require.Equal(t, tc.wantMetadata, md.Encode())
			require.Equal(t, tc.orgID, md.WithTenant(tenantID))

			// the trimming helper must agree on the tenant
			trimmed, err := ExtractValidTenantID(ctx)
			require.NoError(t, err)
			require.Equal(t, tc.wantTenant, trimmed)
		})
	}
}

func TestExtractValidTenantIDs(t *testing.T) {
	tcs := []struct {
		name    string
		orgID   string
		noOrgID bool
		want    string
		wantErr bool
	}{
		{name: "plain tenant", orgID: "test", want: "test"},
		{name: "tenant with metadata", orgID: "test:source=foo", want: "test"},
		{name: "multiple tenants", orgID: "b|a", want: "b|a"},
		{name: "multiple tenants with metadata", orgID: "b:source=foo|a:env=prod", want: "b|a"},
		{name: "invalid tenant", orgID: "a|b/c", wantErr: true},
		{name: "empty tenant", orgID: "", wantErr: true},
		{name: "empty tenant among tenants", orgID: "a||b", wantErr: true},
		{name: "no org ID", noOrgID: true, wantErr: true},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			if !tc.noOrgID {
				ctx = user.InjectOrgID(ctx, tc.orgID)
			}

			got, err := ExtractValidTenantIDs(ctx)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}
