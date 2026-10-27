package api

import (
	"fmt"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/v3/pkg/tempopb"
	"github.com/grafana/tempo/v3/tempodb/backend"
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
		testURL := fmt.Sprintf("http://tempo/api/v2/search/tag/%s/values", tc.tagName)
		if tc.query != "" {
			testURL = fmt.Sprintf("%s?q=%s", testURL, tc.query)
		}

		httpReq := httptest.NewRequest("GET", testURL, nil)
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

// TestSearchTagValuesRequestRoundTripPreservesLimits ensures the per-request
// MaxTagValues (limit) and StaleValueThreshold survive the frontend->querier
// serialization round trip. If they don't, queriers/live-store scan with no
// count limit and no stale-value early-exit (tempo-squad#1355).
func TestSearchTagValuesRequestRoundTripPreservesLimits(t *testing.T) {
	original := &tempopb.SearchTagValuesRequest{
		TagName:             "span.foo",
		Query:               "{ span.bar = `baz` }",
		Start:               100,
		End:                 200,
		MaxTagValues:        50,
		StaleValueThreshold: 25,
	}

	httpReq, err := BuildSearchTagValuesRequest(httptest.NewRequest("GET", "http://tempo/", nil), original)
	require.NoError(t, err)

	// the querier-side parser reads the tag name from mux vars, mirroring routing
	httpReq = mux.SetURLVars(httpReq, map[string]string{MuxVarTagName: original.TagName})

	parsed, err := parseSearchTagValuesRequest(httpReq, false)
	require.NoError(t, err)

	require.Equal(t, original.MaxTagValues, parsed.MaxTagValues, "MaxTagValues (limit) must survive the round trip")
	require.Equal(t, original.StaleValueThreshold, parsed.StaleValueThreshold, "StaleValueThreshold must survive the round trip")
}

// TestSearchTagValuesBlockRequestRoundTripPreservesLimits locks that the block
// request variant (frontend sharder -> querier block search) also carries
// MaxTagValues / StaleValueThreshold through its SearchReq (tempo-squad#1355).
func TestSearchTagValuesBlockRequestRoundTripPreservesLimits(t *testing.T) {
	original := &tempopb.SearchTagValuesBlockRequest{
		SearchReq: &tempopb.SearchTagValuesRequest{
			TagName:             "span.foo",
			Query:               "{ span.bar = `baz` }",
			Start:               100,
			End:                 200,
			MaxTagValues:        50,
			StaleValueThreshold: 25,
		},
		BlockID:       uuid.New().String(),
		StartPage:     2,
		PagesToSearch: 10,
		IndexPageSize: 100,
		TotalRecords:  1000,
		Version:       "vParquet4",
		Size_:         12345,
		FooterSize:    456,
	}

	httpReq, err := BuildSearchTagValuesBlockRequest(httptest.NewRequest("GET", "http://tempo/", nil), original, "")
	require.NoError(t, err)
	httpReq = mux.SetURLVars(httpReq, map[string]string{MuxVarTagName: original.SearchReq.TagName})

	parsed, err := ParseSearchTagValuesBlockRequest(httpReq)
	require.NoError(t, err)

	require.Equal(t, original.SearchReq.MaxTagValues, parsed.SearchReq.MaxTagValues)
	require.Equal(t, original.SearchReq.StaleValueThreshold, parsed.SearchReq.StaleValueThreshold)
}

// TestSearchTagsBlockRequestsRoundTripDedicatedColumns locks that dedicated columns reach the querier, which needs them to find those attributes.
func TestSearchTagsBlockRequestsRoundTripDedicatedColumns(t *testing.T) {
	cols := backend.DefaultDedicatedColumns()
	expected, err := cols.ToTempopb()
	require.NoError(t, err)
	dcJSON, err := NewDedicatedColumnsToJSON().JSONForDedicatedColumns(cols)
	require.NoError(t, err)

	tagsReq, err := BuildSearchTagsBlockRequest(httptest.NewRequest("GET", "http://tempo/", nil), &tempopb.SearchTagsBlockRequest{
		SearchReq:     &tempopb.SearchTagsRequest{Scope: "span", Start: 100, End: 200},
		BlockID:       uuid.New().String(),
		PagesToSearch: 1,
		TotalRecords:  1,
		Version:       "vParquet5",
	}, dcJSON)
	require.NoError(t, err)
	parsedTags, err := ParseSearchTagsBlockRequest(tagsReq)
	require.NoError(t, err)
	require.Equal(t, expected, parsedTags.DedicatedColumns)

	valuesReq, err := BuildSearchTagValuesBlockRequest(httptest.NewRequest("GET", "http://tempo/", nil), &tempopb.SearchTagValuesBlockRequest{
		SearchReq:     &tempopb.SearchTagValuesRequest{TagName: "span.http.request.method", Start: 100, End: 200},
		BlockID:       uuid.New().String(),
		PagesToSearch: 1,
		TotalRecords:  1,
		Version:       "vParquet5",
	}, dcJSON)
	require.NoError(t, err)
	parsedValues, err := ParseSearchTagValuesBlockRequestV2(mux.SetURLVars(valuesReq, map[string]string{MuxVarTagName: "span.http.request.method"}))
	require.NoError(t, err)
	require.Equal(t, expected, parsedValues.DedicatedColumns)
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
