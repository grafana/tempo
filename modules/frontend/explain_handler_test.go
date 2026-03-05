package frontend

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-kit/log"
	"github.com/grafana/tempo/pkg/api"
	"github.com/stretchr/testify/require"
)

func TestExplainHandler_MissingQuery(t *testing.T) {
	handler := newExplainHTTPHandler(log.NewNopLogger())
	req := httptest.NewRequest(http.MethodGet, "/api/explain", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestExplainHandler_InvalidQuery(t *testing.T) {
	handler := newExplainHTTPHandler(log.NewNopLogger())
	req := httptest.NewRequest(http.MethodGet, "/api/explain?q=!!!invalid!!!", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestExplainHandler_SearchQuery(t *testing.T) {
	handler := newExplainHTTPHandler(log.NewNopLogger())
	req := httptest.NewRequest(http.MethodGet, "/api/explain?q=%7Bstatus%3Derror%7D", nil) // {status=error}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var resp api.ExplainResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	require.NotNil(t, resp.Plan)
	require.NotEmpty(t, resp.Plan.Name)
}

func TestExplainHandler_MetricsQuery(t *testing.T) {
	handler := newExplainHTTPHandler(log.NewNopLogger())
	// {status=error} | rate()
	req := httptest.NewRequest(http.MethodGet, "/api/explain?q=%7Bstatus%3Derror%7D+%7C+rate%28%29", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp api.ExplainResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	require.NotNil(t, resp.Plan)
	require.Equal(t, "RateNode", resp.Plan.Name)
}
