package frontend

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/grafana/dskit/tenant"
	"github.com/grafana/dskit/user"
	"github.com/stretchr/testify/assert"
)

func TestMultiTenantNotSupported(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		tenant  string
		err     error
		context bool
	}{
		{
			name:    "multi-tenant queries disabled",
			cfg:     Config{MultiTenantQueriesEnabled: false},
			tenant:  "test",
			err:     nil,
			context: true,
		},
		{
			name:    "multi-tenant queries disabled with multiple tenant",
			cfg:     Config{MultiTenantQueriesEnabled: false},
			tenant:  "test|test1",
			err:     nil,
			context: true,
		},
		{
			name:    "multi-tenant queries enabled with single tenant",
			cfg:     Config{MultiTenantQueriesEnabled: true},
			tenant:  "test",
			err:     nil,
			context: true,
		},
		{
			name:    "multi-tenant queries enabled with multiple tenants",
			cfg:     Config{MultiTenantQueriesEnabled: true},
			tenant:  "test|test1",
			err:     ErrMultiTenantUnsupported,
			context: true,
		},
		{
			name:    "no org id in request context",
			cfg:     Config{MultiTenantQueriesEnabled: true},
			tenant:  "test",
			err:     user.ErrNoOrgID,
			context: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			if tc.context {
				ctx := user.InjectOrgID(context.Background(), tc.tenant)
				req = req.WithContext(ctx)
			}
			resolver := tenant.NewMultiResolver()

			err := MultiTenantNotSupported(tc.cfg, resolver, req)
			assert.Equal(t, tc.err, err)
		})
	}
}
