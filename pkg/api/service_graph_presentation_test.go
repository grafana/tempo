package api

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestServiceGraphPresentationRequest checks validation limits and malformed
// rules, then verifies defaults are copied and backend request sanitization
// preserves unrelated parameters without modifying the original request.
func TestServiceGraphPresentationRequest(t *testing.T) {
	for _, tc := range []struct {
		name, query string
		wantError   bool
	}{
		{"duplicate flag", "include_service_graph_presentation=true&include_service_graph_presentation=false", true},
		{"invalid utf8", "bypass_service=%FF", true},
		{"rule count limit", strings.Repeat("bypass_service=a&", 129), true},
		{"text size limit", "bypass_service=" + strings.Repeat("a", 65537), true},
		{"valid regex", "bypass_service_regex=%5Egateway%24", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/api/traces/id?"+tc.query, nil)
			_, err := ParseServiceGraphPresentationRequest(r, nil, nil)
			if tc.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
	t.Run("copies defaults", func(t *testing.T) {
		names, patterns := []string{"gateway"}, []string{"^proxy"}
		p, err := ParseServiceGraphPresentationRequest(httptest.NewRequest("GET", "/?include_service_graph_presentation=true", nil), names, patterns)
		require.NoError(t, err)
		p.BypassServices[0] = "changed"
		p.BypassServiceRegex[0] = "changed"
		require.Equal(t, []string{"gateway"}, names)
		require.Equal(t, []string{"^proxy"}, patterns)
	})
	t.Run("strips only presentation parameters without mutating request", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/?start=10&bypass_service=gateway&bypass_service_regex=proxy&include_service_graph_presentation=true", nil)
		original := r.URL.RawQuery
		next := WithoutServiceGraphPresentationParams(r)
		require.NotSame(t, r, next)
		require.Equal(t, "start=10", next.URL.RawQuery)
		require.Equal(t, original, r.URL.RawQuery)
		require.Same(t, next, WithoutServiceGraphPresentationParams(next))
	})
}
