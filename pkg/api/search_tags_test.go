package api

import (
	"fmt"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
)

// TestParseSearchTagValues tests the SearchTagValues function
func TestParseSearchTagValuesRequest(t *testing.T) {
	tcs := []struct {
		tagName, query  string
		enforceTraceQL  bool
		expectError     bool
		expectedTagName string
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
		{
			tagName:        "span.test",
			query:          "{}",
			enforceTraceQL: true,
		},
		{
			tagName:        "span.test",
			query:          `{"foo":"bar"}`,
			enforceTraceQL: true,
		},
		{
			tagName:         "span.encoded%2FtagName",
			expectedTagName: "span.encoded/tagName",
		},
		{
			tagName:         "span.encoded%2DtagName",
			expectedTagName: "span.encoded-tagName",
		},
	}

	for _, tc := range tcs {
		testUrl := fmt.Sprintf("http://tempo/api/v2/search/tag/%s/values", tc.tagName)
		if tc.query != "" {
			testUrl = fmt.Sprintf("%s?q=%s", testUrl, tc.query)
		}

		httpReq := httptest.NewRequest("GET", testUrl, nil)
		escapedTagName, err := url.PathUnescape(tc.tagName)
		require.NoError(t, err)
		r := mux.SetURLVars(httpReq, map[string]string{MuxVarTagName: escapedTagName})

		req, err := parseSearchTagValuesRequest(r, tc.enforceTraceQL)
		if tc.expectError {
			require.Error(t, err)
			continue
		}

		expectedTagName := tc.expectedTagName
		if expectedTagName == "" {
			expectedTagName = tc.tagName
		}
		require.Equal(t, expectedTagName, req.TagName)
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
