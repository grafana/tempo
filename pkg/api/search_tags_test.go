package api

import (
	"net/http"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
)

// TestParseSearchTagValues tests the SearchTagValues function
func TestParseSearchTagValues(t *testing.T) {
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
