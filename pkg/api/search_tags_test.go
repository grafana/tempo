package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
)

// TestParseSearchTagValues tests the SearchTagValues function
func TestParseSearchTagValuesRequest(t *testing.T) {
	tcs := []struct {
		tagName        string
		enforceTraceQL bool
		expectError    bool
	}{
		{
			expectError: true,
		},
		{
			tagName: "test",
		},
		{
			tagName:        "test",
			enforceTraceQL: true,
			expectError:    true,
		},
		{
			tagName:        "span.test",
			enforceTraceQL: true,
		},
	}

	for _, tc := range tcs {
		r := mux.SetURLVars(&http.Request{}, map[string]string{muxVarTagName: tc.tagName})

		req, err := parseSearchTagValuesRequest(r, tc.enforceTraceQL)
		if tc.expectError {
			require.Error(t, err)
			continue
		}
		require.Equal(t, tc.tagName, req.TagName)
	}
}

// TestParseSearchTags tests the SearchTagValues function
func TestParseSearchTagsRequest(t *testing.T) {
	tcs := []struct {
		url         string
		scope       string
		expectError bool
	}{
		{
			url: "/",
		},
		{
			url:   "/?scope=span",
			scope: "span",
		},
		{
			url:   "/?scope=intrinsic",
			scope: "intrinsic",
		},
		{
			url: "/?scope=",
		},
		{
			url:         "/?scope=blerg",
			expectError: true,
		},
	}

	for _, tc := range tcs {
		r := httptest.NewRequest("GET", tc.url, nil)
		req, err := ParseSearchTagsRequest(r)
		if tc.expectError {
			require.Error(t, err)
			continue
		}
		require.Equal(t, tc.scope, req.Scope)
	}
}
