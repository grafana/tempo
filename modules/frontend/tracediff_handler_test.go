package frontend

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-kit/log"
	"github.com/grafana/dskit/user"
	"github.com/stretchr/testify/require"
)

func TestTraceDiffHandlerSkeleton(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		body       string
		statusCode int
	}{
		{
			name:       "valid post returns not implemented",
			method:     http.MethodPost,
			body:       `{"base":{"traceId":"abc123"},"compare":{"traceId":"def456"}}`,
			statusCode: http.StatusNotImplemented,
		},
		{
			name:       "invalid post returns bad request",
			method:     http.MethodPost,
			body:       `{"base":{"traceId":"abc123"}}`,
			statusCode: http.StatusBadRequest,
		},
		{
			name:       "get returns method not allowed",
			method:     http.MethodGet,
			statusCode: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := newHandler(nil, newTraceDiffHandler(log.NewNopLogger()), log.NewNopLogger())
			req := httptest.NewRequest(tt.method, "/api/v2/traces/diff", strings.NewReader(tt.body))
			req = req.WithContext(user.InjectOrgID(req.Context(), "test-tenant"))
			resp := httptest.NewRecorder()

			handler.ServeHTTP(resp, req)

			require.Equal(t, tt.statusCode, resp.Code)
		})
	}
}
