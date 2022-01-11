package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/grafana/tempo/cmd/tempo-query/tempo"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/stretchr/testify/assert"
)

// For licensing reasons these strings exist in two packages. This test exists to make sure they don't
// drift.
func TestEquality(t *testing.T) {
	assert.Equal(t, HeaderAccept, tempo.AcceptHeaderKey)
	assert.Equal(t, HeaderAcceptProtobuf, tempo.ProtobufTypeHeaderValue)
}

func TestQuerierParseSearchRequest(t *testing.T) {
	tests := []struct {
		name     string
		urlQuery string
		err      string
		expected *tempopb.SearchRequest
	}{
		{
			name: "empty query",
			expected: &tempopb.SearchRequest{
				Tags:  map[string]string{},
				Limit: defaultLimit,
			},
		},
		{
			name:     "limit set",
			urlQuery: "limit=10",
			expected: &tempopb.SearchRequest{
				Tags:  map[string]string{},
				Limit: 10,
			},
		},
		{
			name:     "zero limit",
			urlQuery: "limit=0",
			err:      "invalid limit: must be a positive number",
		},
		{
			name:     "negative limit",
			urlQuery: "limit=-5",
			err:      "invalid limit: must be a positive number",
		},
		{
			name:     "non-numeric limit",
			urlQuery: "limit=five",
			err:      "invalid limit: strconv.Atoi: parsing \"five\": invalid syntax",
		},
		{
			name:     "minDuration and maxDuration",
			urlQuery: "minDuration=10s&maxDuration=20s",
			expected: &tempopb.SearchRequest{
				Tags:          map[string]string{},
				MinDurationMs: 10000,
				MaxDurationMs: 20000,
				Limit:         defaultLimit,
			},
		},
		{
			name:     "minDuration greater than maxDuration",
			urlQuery: "minDuration=20s&maxDuration=5s",
			err:      "invalid maxDuration: must be greater than minDuration",
		},
		{
			name:     "invalid minDuration",
			urlQuery: "minDuration=10seconds",
			err:      "invalid minDuration: time: unknown unit \"seconds\" in duration \"10seconds\"",
		},
		{
			name:     "invalid maxDuration",
			urlQuery: "maxDuration=1msec",
			err:      "invalid maxDuration: time: unknown unit \"msec\" in duration \"1msec\"",
		},
		{
			name:     "tags and limit",
			urlQuery: "service.name=foo&tags=limit%3Dfive&limit=5&query=1%2B1%3D2",
			expected: &tempopb.SearchRequest{
				Tags: map[string]string{
					"limit": "five",
				},
				Limit: 5,
			},
		},
		{
			name:     "tags query parameter with duplicate tag",
			urlQuery: "tags=service.name%3Dfoo%20service.name%3Dbar",
			err:      "invalid tags: tag service.name has been set twice",
		},
		{
			name:     "top-level tags with conflicting query parameter tags",
			urlQuery: "service.name=bar&tags=service.name%3Dfoo",
			expected: &tempopb.SearchRequest{
				Tags: map[string]string{
					"service.name": "foo",
				},
				Limit: defaultLimit,
			},
		},
		{
			name:     "start and end both set",
			urlQuery: "tags=service.name%3Dfoo&start=10&end=20",
			expected: &tempopb.SearchRequest{
				Tags: map[string]string{
					"service.name": "foo",
				},
				Start: 10,
				End:   20,
				Limit: defaultLimit,
			},
		},
		{
			name:     "end before start",
			urlQuery: "tags=service.name%3Dfoo&start=20&end=10",
			err:      "http parameter start must be before end. received start=20 end=10",
		},
		{
			name:     "top-level tags",
			urlQuery: "service.name=bar",
			expected: &tempopb.SearchRequest{
				Tags: map[string]string{
					"service.name": "bar",
				},
				Limit: defaultLimit,
			},
		},
		{
			name:     "top-level tags with range specified are ignored",
			urlQuery: "service.name=bar&start=10&end=20",
			expected: &tempopb.SearchRequest{
				Tags:  map[string]string{},
				Start: 10,
				End:   20,
				Limit: defaultLimit,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "http://tempo/api/search?"+tt.urlQuery, nil)
			fmt.Println("RequestURI:", r.RequestURI)

			searchRequest, err := ParseSearchRequest(r)

			if tt.err != "" {
				assert.EqualError(t, err, tt.err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, searchRequest)
			}
		})
	}
}

func TestQuerierParseSearchRequestTags(t *testing.T) {
	type strMap map[string]string

	tests := []struct {
		tags     string
		expected map[string]string
	}{
		{"service.name=foo http.url=api/search", strMap{"service.name": "foo", "http.url": "api/search"}},
		{"service%n@me=foo", strMap{"service%n@me": "foo"}},
		{"service.name=foo error", strMap{"service.name": "foo", "error": ""}},
		{"service.name=\"foo bar\"", strMap{"service.name": "foo bar"}},
		{"service.name=\"foo=bar\"", strMap{"service.name": "foo=bar"}},
		{"service.name=\"foo\\bar\"", strMap{"service.name": "foo\bar"}},
		{"service.name=\"foo \\\"bar\\\"\"", strMap{"service.name": "foo \"bar\""}},
	}

	for _, tt := range tests {
		t.Run(tt.tags, func(t *testing.T) {
			r := httptest.NewRequest("GET", "http://tempo/api/search?tags="+url.QueryEscape(tt.tags), nil)
			fmt.Println("RequestURI:", r.RequestURI)

			searchRequest, err := ParseSearchRequest(r)

			assert.NoError(t, err)
			assert.Equal(t, tt.expected, searchRequest.Tags)
		})
	}
}

func TestQuerierParseSearchRequestTagsError(t *testing.T) {
	tests := []struct {
		tags string
		err  string
	}{
		{"service.name=foo =error", "invalid tags: unexpected '=' at pos 18"},
		{"service.name=foo=bar", "invalid tags: unexpected '=' at pos 17"},
		{"service.name=\"foo bar", "invalid tags: unterminated quoted value at pos 22"},
		{"\"service name\"=foo", "invalid tags: unexpected '\"' at pos 1"},
	}

	for _, tt := range tests {
		t.Run(tt.tags, func(t *testing.T) {
			r := httptest.NewRequest("GET", "http://tempo/api/search?tags="+url.QueryEscape(tt.tags), nil)
			fmt.Println("RequestURI:", r.RequestURI)

			_, err := ParseSearchRequest(r)

			assert.EqualError(t, err, tt.err)
		})
	}
}

func TestParseSearchBlockRequest(t *testing.T) {
	tests := []struct {
		url           string
		expected      *tempopb.SearchBlockRequest
		expectedError string
	}{
		{
			url:           "/",
			expectedError: "start and end required",
		},
		{
			url:           "/?start=10&end=20",
			expectedError: "invalid startPage: strconv.ParseInt: parsing \"\": invalid syntax",
		},
		{
			url:           "/?start=10&end=20&startPage=0",
			expectedError: "invalid pagesToSearch : strconv.ParseInt: parsing \"\": invalid syntax",
		},
		{
			url:           "/?start=10&end=20&startPage=0&pagesToSearch=10",
			expectedError: "invalid blockID: invalid UUID length: 0",
		},
		{
			url:           "/?start=10&end=20&startPage=0&pagesToSearch=10&blockID=adsf",
			expectedError: "invalid blockID: invalid UUID length: 4",
		},
		{
			url:           "/?start=10&end=20&startPage=0&pagesToSearch=10&blockID=b92ec614-3fd7-4299-b6db-f657e7025a9b",
			expectedError: "invalid encoding: , supported: none, gzip, lz4-64k, lz4-256k, lz4-1M, lz4, snappy, zstd, s2",
		},
		{
			url:           "/?start=10&end=20&startPage=0&pagesToSearch=10&blockID=b92ec614-3fd7-4299-b6db-f657e7025a9b&encoding=blerg",
			expectedError: "invalid encoding: blerg, supported: none, gzip, lz4-64k, lz4-256k, lz4-1M, lz4, snappy, zstd, s2",
		},
		{
			url:           "/?start=10&end=20&startPage=0&pagesToSearch=10&blockID=b92ec614-3fd7-4299-b6db-f657e7025a9b&encoding=s2",
			expectedError: "invalid indexPageSize : strconv.ParseInt: parsing \"\": invalid syntax",
		},
		{
			url:           "/?start=10&end=20&startPage=0&pagesToSearch=10&blockID=b92ec614-3fd7-4299-b6db-f657e7025a9b&encoding=s2&indexPageSize=0",
			expectedError: "indexPageSize must be greater than 0. received 0",
		},
		{
			url:           "/?start=10&end=20&startPage=0&pagesToSearch=10&blockID=b92ec614-3fd7-4299-b6db-f657e7025a9b&encoding=s2&indexPageSize=10",
			expectedError: "invalid totalRecords : strconv.ParseInt: parsing \"\": invalid syntax",
		},
		{
			url:           "/?start=10&end=20&startPage=0&pagesToSearch=10&blockID=b92ec614-3fd7-4299-b6db-f657e7025a9b&encoding=s2&indexPageSize=10&totalRecords=-1",
			expectedError: "totalRecords must be greater than 0. received -1",
		},
		{
			url:           "/?start=10&end=20&startPage=0&pagesToSearch=10&blockID=b92ec614-3fd7-4299-b6db-f657e7025a9b&encoding=s2&indexPageSize=10&totalRecords=11",
			expectedError: "dataEncoding required",
		},
		{
			url:           "/?start=10&end=20&startPage=0&pagesToSearch=10&blockID=b92ec614-3fd7-4299-b6db-f657e7025a9b&encoding=s2&indexPageSize=10&totalRecords=11&dataEncoding=v1",
			expectedError: "version required",
		},
		{
			url: "/?tags=foo%3Dbar&start=10&end=20&startPage=0&pagesToSearch=10&blockID=b92ec614-3fd7-4299-b6db-f657e7025a9b&encoding=s2&indexPageSize=10&totalRecords=11&dataEncoding=v1&version=v2",
			expected: &tempopb.SearchBlockRequest{
				SearchReq: &tempopb.SearchRequest{
					Tags: map[string]string{
						"foo": "bar",
					},
					Start: 10,
					End:   20,
					Limit: defaultLimit,
				},
				StartPage:     0,
				PagesToSearch: 10,
				BlockID:       "b92ec614-3fd7-4299-b6db-f657e7025a9b",
				Encoding:      "s2",
				IndexPageSize: 10,
				TotalRecords:  11,
				DataEncoding:  "v1",
				Version:       "v2",
			},
		},
	}

	for _, tc := range tests {
		r := httptest.NewRequest("GET", tc.url, nil)
		actualReq, actualErr := ParseSearchBlockRequest(r)

		if len(tc.expectedError) != 0 {
			assert.EqualError(t, actualErr, tc.expectedError)
			assert.Nil(t, actualReq)
			continue
		}
		assert.Equal(t, tc.expected, actualReq)
	}
}

func TestBuildSearchBlockRequest(t *testing.T) {
	tests := []struct {
		req     *tempopb.SearchBlockRequest
		httpReq *http.Request
		query   string
	}{
		{
			req: &tempopb.SearchBlockRequest{
				StartPage:     0,
				PagesToSearch: 10,
				BlockID:       "b92ec614-3fd7-4299-b6db-f657e7025a9b",
				Encoding:      "s2",
				IndexPageSize: 10,
				TotalRecords:  11,
				DataEncoding:  "v1",
				Version:       "v2",
			},
			query: "?blockID=b92ec614-3fd7-4299-b6db-f657e7025a9b&dataEncoding=v1&encoding=s2&indexPageSize=10&pagesToSearch=10&startPage=0&totalRecords=11&version=v2",
		},
		{
			req: &tempopb.SearchBlockRequest{
				StartPage:     0,
				PagesToSearch: 10,
				BlockID:       "b92ec614-3fd7-4299-b6db-f657e7025a9b",
				Encoding:      "s2",
				IndexPageSize: 10,
				TotalRecords:  11,
				DataEncoding:  "v1",
				Version:       "v2",
			},
			httpReq: httptest.NewRequest("GET", "/test/path", nil),
			query:   "/test/path?blockID=b92ec614-3fd7-4299-b6db-f657e7025a9b&dataEncoding=v1&encoding=s2&indexPageSize=10&pagesToSearch=10&startPage=0&totalRecords=11&version=v2",
		},
		{
			req: &tempopb.SearchBlockRequest{
				SearchReq: &tempopb.SearchRequest{
					Tags: map[string]string{
						"foo": "bar",
					},
					Start:         10,
					End:           20,
					MinDurationMs: 30,
					MaxDurationMs: 40,
					Limit:         50,
				},
				StartPage:     0,
				PagesToSearch: 10,
				BlockID:       "b92ec614-3fd7-4299-b6db-f657e7025a9b",
				Encoding:      "s2",
				IndexPageSize: 10,
				TotalRecords:  11,
				DataEncoding:  "v1",
				Version:       "v2",
			},
			query: "?blockID=b92ec614-3fd7-4299-b6db-f657e7025a9b&dataEncoding=v1&encoding=s2&end=20&indexPageSize=10&limit=50&maxDuration=40ms&minDuration=30ms&pagesToSearch=10&start=10&startPage=0&tags=foo%3Dbar&totalRecords=11&version=v2",
		},
	}

	for _, tc := range tests {
		actualURL, err := BuildSearchBlockRequest(tc.httpReq, tc.req)
		assert.NoError(t, err)
		assert.Equal(t, tc.query, actualURL.URL.String())
	}
}

func TestBuildSearchRequest(t *testing.T) {
	tests := []struct {
		req     *tempopb.SearchRequest
		httpReq *http.Request
		query   string
	}{
		{
			req: &tempopb.SearchRequest{
				Tags: map[string]string{
					"foo": "bar",
				},
				Start:         10,
				End:           20,
				MinDurationMs: 30,
				MaxDurationMs: 40,
				Limit:         50,
			},
			query: "?end=20&limit=50&maxDuration=40ms&minDuration=30ms&start=10&tags=foo%3Dbar",
		},
		{
			req: &tempopb.SearchRequest{
				Tags: map[string]string{
					"foo": "bar",
				},
				Start:         10,
				End:           20,
				MaxDurationMs: 30,
				Limit:         50,
			},
			query: "?end=20&limit=50&maxDuration=30ms&start=10&tags=foo%3Dbar",
		},
		{
			req: &tempopb.SearchRequest{
				Tags: map[string]string{
					"foo": "bar",
				},
				Start:         10,
				End:           20,
				MinDurationMs: 30,
				Limit:         50,
			},
			query: "?end=20&limit=50&minDuration=30ms&start=10&tags=foo%3Dbar",
		},
		{
			req: &tempopb.SearchRequest{
				Tags: map[string]string{
					"foo": "bar",
				},
				Start:         10,
				End:           20,
				MinDurationMs: 30,
				MaxDurationMs: 40,
			},
			query: "?end=20&maxDuration=40ms&minDuration=30ms&start=10&tags=foo%3Dbar",
		},
		{
			req: &tempopb.SearchRequest{
				Tags:          map[string]string{},
				Start:         10,
				End:           20,
				MinDurationMs: 30,
				MaxDurationMs: 40,
			},
			query: "?end=20&maxDuration=40ms&minDuration=30ms&start=10",
		},
	}

	for _, tc := range tests {
		actualURL, err := BuildSearchRequest(tc.httpReq, tc.req)
		assert.NoError(t, err)
		assert.Equal(t, tc.query, actualURL.URL.String())
	}
}
