package api

import (
	"errors"
	"fmt"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/google/uuid"
	"github.com/grafana/tempo/cmd/tempo-query/tempo"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/stretchr/testify/assert"
)

// For licensing reasons these strings exist in two packages. This test exists to make sure they don't
// drift.
func TestEquality(t *testing.T) {
	assert.Equal(t, HeaderAccept, tempo.AcceptHeaderKey)
	assert.Equal(t, HeaderAcceptProtobuf, tempo.ProtobufTypeHeaderValue)
}

func TestQuerierParseSearchRequest(t *testing.T) {
	defaultLimit := uint32(20)
	maxLimit := uint32(100)

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
			name:     "limit exceeding max",
			urlQuery: "limit=120",
			expected: &tempopb.SearchRequest{
				Tags:  map[string]string{},
				Limit: 100,
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "http://tempo/api/search?"+tt.urlQuery, nil)
			fmt.Println("RequestURI:", r.RequestURI)

			searchRequest, err := ParseSearchRequest(r, defaultLimit, maxLimit)

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

			searchRequest, err := ParseSearchRequest(r, 0, 0)

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

			_, err := ParseSearchRequest(r, 0, 0)

			assert.EqualError(t, err, tt.err)
		})
	}
}

func TestParseBackendSearch(t *testing.T) {
	tests := []struct {
		start         int64
		end           int64
		limit         int
		expectedLimit int
		expectedError error
	}{
		{
			expectedError: errors.New("please provide positive values for http parameters start and end"),
		},
		{
			start:         10,
			expectedError: errors.New("please provide positive values for http parameters start and end"),
		},
		{
			end:           10,
			expectedError: errors.New("please provide positive values for http parameters start and end"),
		},
		{
			start:         15,
			end:           10,
			expectedError: errors.New("http parameter start must be before end. received start=15 end=10"),
		},
		{
			start:         10,
			end:           100000,
			expectedError: errors.New("range specified by start and end exceeds 1800 seconds. received start=10 end=100000"),
		},
		{
			start:         10,
			end:           20,
			expectedLimit: 20,
		},
		{
			start:         10,
			end:           20,
			limit:         30,
			expectedLimit: 30,
		},
	}

	for _, tc := range tests {
		url := "/blerg?"
		if tc.start != 0 {
			url += fmt.Sprintf("&start=%d", tc.start)
		}
		if tc.end != 0 {
			url += fmt.Sprintf("&end=%d", tc.end)
		}
		if tc.limit != 0 {
			url += fmt.Sprintf("&limit=%d", tc.limit)
		}
		r := httptest.NewRequest("GET", url, nil)

		actualStart, actualEnd, actualLimit, actualError := ParseBackendSearch(r)

		if tc.expectedError != nil {
			assert.Equal(t, tc.expectedError, actualError)
			continue
		}
		assert.NoError(t, actualError)
		assert.Equal(t, tc.start, actualStart)
		assert.Equal(t, tc.end, actualEnd)
		assert.Equal(t, tc.expectedLimit, actualLimit)
	}
}

func TestParseBackendSearchQuerier(t *testing.T) {
	tests := []struct {
		url                string
		expectedError      string
		expectedStartPage  uint32
		expectedTotalPages uint32
		expectedBlockID    uuid.UUID
	}{
		{
			url:           "/",
			expectedError: "blockID required",
		},
		{
			url:           "/?blockID=asdf",
			expectedError: "blockID: invalid UUID length: 4",
		},
		{
			url:             "/?blockID=b92ec614-3fd7-4299-b6db-f657e7025a9b",
			expectedBlockID: uuid.MustParse("b92ec614-3fd7-4299-b6db-f657e7025a9b"),
		},
		{
			url:           "/?blockID=b92ec614-3fd7-4299-b6db-f657e7025a9b&startPage=-1",
			expectedError: "startPage must be non-negative. received: -1",
		},
		{
			url:           "/?blockID=b92ec614-3fd7-4299-b6db-f657e7025a9b&startPage=0&totalPages=0",
			expectedError: "totalPages must be greater than 0. received: 0",
		},
		{
			url:                "/?blockID=b92ec614-3fd7-4299-b6db-f657e7025a9b&startPage=4&totalPages=3",
			expectedStartPage:  4,
			expectedTotalPages: 3,
			expectedBlockID:    uuid.MustParse("b92ec614-3fd7-4299-b6db-f657e7025a9b"),
		},
	}

	for _, tc := range tests {
		r := httptest.NewRequest("GET", tc.url, nil)
		actualStartPage, actualTotalPages, actualBlockID, actualErr := ParseBackendSearchQuerier(r)

		if len(tc.expectedError) != 0 {
			assert.EqualError(t, actualErr, tc.expectedError)
			continue
		}
		assert.Equal(t, tc.expectedStartPage, actualStartPage)
		assert.Equal(t, tc.expectedTotalPages, actualTotalPages)
		assert.Equal(t, tc.expectedBlockID, actualBlockID)
	}
}

// todo(search): improve this test. it is currently thin b/c i expect it to change
func TestParseBackendSearchServerless(t *testing.T) {
	tests := []struct {
		url                   string
		expectedError         string
		expectedEncoding      backend.Encoding
		expectedDataEncoding  string
		expectedIndexPageSize uint32
		expectedTotalRecords  uint32
		expectedTenant        string
		expectedVersion       string
	}{
		{
			url:                   "/?encoding=none&dataEncoding=v2&indexPageSize=2&totalRecords=3&tenant=yay&version=v3",
			expectedEncoding:      backend.EncNone,
			expectedDataEncoding:  "v2",
			expectedIndexPageSize: 2,
			expectedTotalRecords:  3,
			expectedTenant:        "yay",
			expectedVersion:       "v3",
		},
	}

	for _, tc := range tests {
		r := httptest.NewRequest("GET", tc.url, nil)
		encoding, dataEncoding, indexPageSize, totalRecords, tenant, version, err := ParseBackendSearchServerless(r)

		if len(tc.expectedError) != 0 {
			assert.EqualError(t, err, tc.expectedError)
			continue
		}
		assert.Equal(t, tc.expectedEncoding, encoding)
		assert.Equal(t, tc.expectedDataEncoding, dataEncoding)
		assert.Equal(t, tc.expectedIndexPageSize, indexPageSize)
		assert.Equal(t, tc.expectedTotalRecords, totalRecords)
		assert.Equal(t, tc.expectedTenant, tenant)
		assert.Equal(t, tc.expectedVersion, version)
	}
}
