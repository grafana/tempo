package frontend

import (
	"io"
	"net/http"
	"testing"

	"github.com/go-kit/log"
	"github.com/grafana/dskit/user"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractTenant(t *testing.T) {
	logger := log.NewNopLogger()

	t.Run("success case - tenant extracted from context", func(t *testing.T) {
		// Create request with tenant in context
		req, err := http.NewRequest("GET", "/api/traces/123", nil)
		require.NoError(t, err)

		// Inject tenant ID into context
		ctx := user.InjectOrgID(req.Context(), "test-tenant")
		req = req.WithContext(ctx)

		// Call extractTenant
		tenant, errResp := extractTenant(req, logger)

		// Verify success
		assert.Equal(t, "test-tenant", tenant)
		assert.Nil(t, errResp)
	})

	t.Run("success case - single-tenant mode", func(t *testing.T) {
		// Create request with single-tenant ID
		req, err := http.NewRequest("GET", "/api/search", nil)
		require.NoError(t, err)

		// Inject single-tenant ID (simulating fake auth middleware)
		ctx := user.InjectOrgID(req.Context(), "single-tenant")
		req = req.WithContext(ctx)

		// Call extractTenant
		tenant, errResp := extractTenant(req, logger)

		// Verify success
		assert.Equal(t, "single-tenant", tenant)
		assert.Nil(t, errResp)
	})

	t.Run("error case - no tenant in context", func(t *testing.T) {
		// Create request without tenant in context
		req, err := http.NewRequest("GET", "/api/traces/123", nil)
		require.NoError(t, err)

		// Call extractTenant
		tenant, errResp := extractTenant(req, logger)

		// Verify error response
		assert.Empty(t, tenant)
		require.NotNil(t, errResp)

		// Check HTTP response details
		assert.Equal(t, http.StatusBadRequest, errResp.StatusCode)
		assert.Equal(t, "Bad Request", errResp.Status)

		// Read response body
		bodyBytes, err := io.ReadAll(errResp.Body)
		require.NoError(t, err)
		bodyStr := string(bodyBytes)

		// Should contain error message about missing org ID
		assert.Contains(t, bodyStr, "no org id")
	})
}
