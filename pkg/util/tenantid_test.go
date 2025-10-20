package util

import (
	"context"
	"strings"
	"testing"

	"github.com/grafana/dskit/tenant"
	"github.com/grafana/dskit/user"
	"github.com/stretchr/testify/require"
)

func TestExtractValidOrgID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		orgID   string
		want    string
		wantErr bool
	}{
		{
			name:  "valid single tenant",
			orgID: "tenant-123",
			want:  "tenant-123",
		},
		{
			name:    "invalid single tenant",
			orgID:   "tenant#123",
			wantErr: true,
		},
		{
			name:    "invalid single tenant unsupported character slash",
			orgID:   "tenant/123",
			wantErr: true,
		},
		{
			name:    "invalid single tenant unsafe path segment",
			orgID:   "..",
			wantErr: true,
		},
		{
			name:    "invalid single tenant too long",
			orgID:   strings.Repeat("a", tenant.MaxTenantIDLength+1),
			wantErr: true,
		},
		{
			name:  "valid multi tenant",
			orgID: "tenantA|tenantB",
			want:  "tenantA|tenantB",
		},
		{
			name:    "invalid multi tenant",
			orgID:   "tenantA|tenant#B",
			wantErr: true,
		},
		{
			name:    "invalid multi tenant with unsafe path segment",
			orgID:   "tenantA|..",
			wantErr: true,
		},
		{
			name:    "invalid multi tenant with unsupported character slash",
			orgID:   "tenantA|tenant/B",
			wantErr: true,
		},
		{
			name:    "invalid multi tenant too long segment",
			orgID:   "tenantA|" + strings.Repeat("b", tenant.MaxTenantIDLength+1),
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := user.InjectOrgID(context.Background(), tc.orgID)

			got, err := ExtractValidOrgID(ctx)
			if tc.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}
