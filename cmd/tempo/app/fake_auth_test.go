package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/grafana/dskit/user"
	"github.com/grafana/tempo/pkg/util"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

func TestNoAuthTenantID(t *testing.T) {
	require.Equal(t, util.FakeTenantID, (&Config{}).NoAuthTenantID())
	require.Equal(t, "custom-tenant", (&Config{NoAuthTenant: "custom-tenant"}).NoAuthTenantID())
	require.Equal(t, util.FakeTenantID, NewDefaultConfig().NoAuthTenantID())
}

func TestFakeHTTPAuthMiddlewareUsesConfiguredTenant(t *testing.T) {
	const tenantID = "custom-tenant"

	handler := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		orgID, err := user.ExtractOrgID(r.Context())
		require.NoError(t, err)
		require.Equal(t, tenantID, orgID)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	resp := httptest.NewRecorder()

	fakeHTTPAuthMiddleware(tenantID).Wrap(handler).ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
}

func TestFakeGRPCAuthUnaryMiddlewareUsesConfiguredTenant(t *testing.T) {
	const tenantID = "custom-tenant"

	_, err := fakeGRPCAuthUnaryMiddleware(tenantID)(
		context.Background(),
		nil,
		&grpc.UnaryServerInfo{},
		func(ctx context.Context, _ interface{}) (interface{}, error) {
			orgID, err := user.ExtractOrgID(ctx)
			require.NoError(t, err)
			require.Equal(t, tenantID, orgID)
			return nil, nil
		},
	)

	require.NoError(t, err)
}

func TestSetupAuthMiddlewareConfiguresGeneratorNoAuthTenant(t *testing.T) {
	cfg := NewDefaultConfig()
	cfg.NoAuthTenant = "custom-tenant"

	app := &App{cfg: *cfg}
	app.setupAuthMiddleware()

	require.Equal(t, "custom-tenant", app.cfg.Generator.Storage.NoAuthTenant)
}
