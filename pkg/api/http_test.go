package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/cmd/tempo-query/tempo"
	"github.com/grafana/tempo/pkg/tempopb"
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
				Tags:            map[string]string{},
				SpansPerSpanSet: defaultSpansPerSpanSet,
			},
		},
		{
			name:     "zero ranges",
			urlQuery: "start=0&end=0",
			expected: &tempopb.SearchRequest{
				Tags:            map[string]string{},
				SpansPerSpanSet: defaultSpansPerSpanSet,
			},
		},
		{
			name:     "limit set",
			urlQuery: "limit=10",
			expected: &tempopb.SearchRequest{
				Tags:            map[string]string{},
				Limit:           10,
				SpansPerSpanSet: defaultSpansPerSpanSet,
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
				Tags:            map[string]string{},
				MinDurationMs:   10000,
				MaxDurationMs:   20000,
				SpansPerSpanSet: defaultSpansPerSpanSet,
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
			name:     "traceql query",
			urlQuery: "q=" + url.QueryEscape(`{ .foo="bar" }`),
			expected: &tempopb.SearchRequest{
				Query:           `{ .foo="bar" }`,
				Tags:            map[string]string{},
				SpansPerSpanSet: defaultSpansPerSpanSet,
			},
		},
		{
			name:     "traceql query and tags",
			urlQuery: "q=" + url.QueryEscape(`{ .foo="bar" }`) + "&tags=" + url.QueryEscape("service.name=foo"),
			err:      "invalid request: can't specify tags and q in the same query",
		},
		{
			name:     "tags and limit",
			urlQuery: "tags=" + url.QueryEscape("limit=five") + "&limit=5",
			expected: &tempopb.SearchRequest{
				Tags: map[string]string{
					"limit": "five",
				},
				Limit:           5,
				SpansPerSpanSet: defaultSpansPerSpanSet,
			},
		},
		{
			name:     "tags query parameter with duplicate tag",
			urlQuery: "tags=" + url.QueryEscape("service.name=foo service.name=bar"),
			err:      "invalid tags: tag service.name has been set twice",
		},
		{
			name:     "top-level tags with conflicting query parameter tags",
			urlQuery: "service.name=bar&tags=" + url.QueryEscape("service.name=foo"),
			expected: &tempopb.SearchRequest{
				Tags: map[string]string{
					"service.name": "foo",
				},
				SpansPerSpanSet: defaultSpansPerSpanSet,
			},
		},
		{
			name:     "start and end both set",
			urlQuery: "tags=" + url.QueryEscape("service.name=foo") + "&start=10&end=20",
			expected: &tempopb.SearchRequest{
				Tags: map[string]string{
					"service.name": "foo",
				},
				Start:           10,
				End:             20,
				SpansPerSpanSet: defaultSpansPerSpanSet,
			},
		},
		{
			name:     "end before start",
			urlQuery: "tags=" + url.QueryEscape("service.name=foo") + "&start=20&end=10",
			err:      "http parameter start must be before end. received start=20 end=10",
		},
		{
			name:     "top-level tags",
			urlQuery: "service.name=bar",
			expected: &tempopb.SearchRequest{
				Tags: map[string]string{
					"service.name": "bar",
				},
				SpansPerSpanSet: defaultSpansPerSpanSet,
			},
		},
		{
			name:     "top-level tags with range specified are ignored",
			urlQuery: "service.name=bar&start=10&end=20",
			expected: &tempopb.SearchRequest{
				Tags:            map[string]string{},
				Start:           10,
				End:             20,
				SpansPerSpanSet: defaultSpansPerSpanSet,
			},
		},
		{
			name:     "zero spss",
			urlQuery: "spss=0",
			err:      "invalid spss: must be a positive number",
		},
		{
			name:     "negative spss",
			urlQuery: "spss=-2",
			err:      "invalid spss: must be a positive number",
		},
		{
			name:     "non-numeric spss",
			urlQuery: "spss=four",
			err:      "invalid spss: strconv.Atoi: parsing \"four\": invalid syntax",
		},
		{
			name:     "only spss",
			urlQuery: "spss=2",
			expected: &tempopb.SearchRequest{
				Tags:            map[string]string{},
				SpansPerSpanSet: 2,
			},
		},
		{
			name:     "tags with spss",
			urlQuery: "tags=" + url.QueryEscape("service.name=foo") + "&spss=7",
			expected: &tempopb.SearchRequest{
				Tags: map[string]string{
					"service.name": "foo",
				},
				SpansPerSpanSet: 7,
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
			url:           "/?start=10&end=20&startPage=0&pagesToSearch=10&blockID=b92ec614-3fd7-4299-b6db-f657e7025a9b&encoding=s2&indexPageSize=10",
			expectedError: "invalid totalRecords : strconv.ParseInt: parsing \"\": invalid syntax",
		},
		{
			url:           "/?start=10&end=20&startPage=0&pagesToSearch=10&blockID=b92ec614-3fd7-4299-b6db-f657e7025a9b&encoding=s2&indexPageSize=10&totalRecords=-1",
			expectedError: "totalRecords must be greater than 0. received -1",
		},
		{
			url:           "/?start=10&end=20&startPage=0&pagesToSearch=10&blockID=b92ec614-3fd7-4299-b6db-f657e7025a9b&encoding=s2&indexPageSize=10&totalRecords=11&dataEncoding=v1",
			expectedError: "version required",
		},
		{
			url:           "/?start=10&end=20&startPage=0&pagesToSearch=10&blockID=b92ec614-3fd7-4299-b6db-f657e7025a9b&encoding=s2&indexPageSize=10&totalRecords=11&dataEncoding=v1&version=v2&size=1000",
			expectedError: "invalid footerSize : strconv.ParseUint: parsing \"\": invalid syntax",
		},
		{
			url: "/?tags=foo%3Dbar&start=10&end=20&startPage=0&pagesToSearch=10&blockID=b92ec614-3fd7-4299-b6db-f657e7025a9b&encoding=s2&footerSize=2000&indexPageSize=10&totalRecords=11&dataEncoding=v1&version=v2&size=1000",
			expected: &tempopb.SearchBlockRequest{
				SearchReq: &tempopb.SearchRequest{
					Tags: map[string]string{
						"foo": "bar",
					},
					Start:           10,
					End:             20,
					Limit:           defaultLimit,
					SpansPerSpanSet: defaultSpansPerSpanSet,
				},
				StartPage:     0,
				PagesToSearch: 10,
				BlockID:       "b92ec614-3fd7-4299-b6db-f657e7025a9b",
				Encoding:      "s2",
				IndexPageSize: 10,
				TotalRecords:  11,
				DataEncoding:  "v1",
				Version:       "v2",
				Size_:         1000,
				FooterSize:    2000,
			},
		},
		{
			url: "/?tags=foo%3Dbar&start=10&end=20&blockID=b92ec614-3fd7-4299-b6db-f657e7025a9b&dataEncoding=&dc=%5B%7B%22type%22%3A0%2C%22name%22%3A%22net.sock.host.addr%22%2C%22scope%22%3A0%7D%5D&encoding=none&footerSize=2000&indexPageSize=0&pagesToSearch=10&size=1000&startPage=0&totalRecords=2&version=vParquet3",
			expected: &tempopb.SearchBlockRequest{
				SearchReq: &tempopb.SearchRequest{
					Tags: map[string]string{
						"foo": "bar",
					},
					Start:           10,
					End:             20,
					Limit:           defaultLimit,
					SpansPerSpanSet: defaultSpansPerSpanSet,
				},
				StartPage:     0,
				PagesToSearch: 10,
				BlockID:       "b92ec614-3fd7-4299-b6db-f657e7025a9b",
				Encoding:      "none",
				IndexPageSize: 0,
				TotalRecords:  2,
				Version:       "vParquet3",
				Size_:         1000,
				FooterSize:    2000,
				DedicatedColumns: []*tempopb.DedicatedColumn{
					{Scope: tempopb.DedicatedColumn_SPAN, Name: "net.sock.host.addr", Type: tempopb.DedicatedColumn_STRING},
				},
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
				Size_:         1000,
				FooterSize:    2000,
			},
			query: "?blockID=b92ec614-3fd7-4299-b6db-f657e7025a9b&pagesToSearch=10&size=1000&startPage=0&encoding=s2&indexPageSize=10&totalRecords=11&dataEncoding=v1&version=v2&footerSize=2000",
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
				Size_:         1000,
				FooterSize:    2000,
			},
			httpReq: httptest.NewRequest("GET", "/test/path", nil),
			query:   "/test/path?blockID=b92ec614-3fd7-4299-b6db-f657e7025a9b&pagesToSearch=10&size=1000&startPage=0&encoding=s2&indexPageSize=10&totalRecords=11&dataEncoding=v1&version=v2&footerSize=2000",
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
				Size_:         1000,
				FooterSize:    2000,
			},
			query: "?start=10&end=20&limit=50&maxDuration=40ms&minDuration=30ms&tags=foo%3Dbar&blockID=b92ec614-3fd7-4299-b6db-f657e7025a9b&pagesToSearch=10&size=1000&startPage=0&encoding=s2&indexPageSize=10&totalRecords=11&dataEncoding=v1&version=v2&footerSize=2000",
		},
		{
			req: &tempopb.SearchBlockRequest{
				StartPage:     0,
				PagesToSearch: 10,
				BlockID:       "b92ec614-3fd7-4299-b6db-f657e7025a9b",
				Encoding:      "none",
				IndexPageSize: 0,
				TotalRecords:  2,
				Version:       "vParquet3",
				Size_:         1000,
				FooterSize:    2000,
				DedicatedColumns: []*tempopb.DedicatedColumn{
					{Scope: tempopb.DedicatedColumn_RESOURCE, Name: "net.sock.host.addr", Type: tempopb.DedicatedColumn_STRING},
				},
			},
			httpReq: httptest.NewRequest("GET", "/test/path", nil),
			query:   "/test/path?blockID=b92ec614-3fd7-4299-b6db-f657e7025a9b&pagesToSearch=10&size=1000&startPage=0&encoding=none&indexPageSize=0&totalRecords=2&dataEncoding=&version=vParquet3&footerSize=2000&dc=%5B%7B%22scope%22%3A1%2C%22name%22%3A%22net.sock.host.addr%22%7D%5D",
		},
	}

	for _, tc := range tests {
		jsonBytes, err := json.Marshal(tc.req.DedicatedColumns)
		require.NoError(t, err)

		actualURL, err := BuildSearchBlockRequest(tc.httpReq, tc.req, string(jsonBytes))
		assert.NoError(t, err)
		assert.Equal(t, tc.query, actualURL.URL.String())
	}
}

func TestValidateAndSanitizeRequest(t *testing.T) {
	tests := []struct {
		httpReq       *http.Request
		queryMode     string
		startTime     int64
		endTime       int64
		blockStart    string
		blockEnd      string
		expectedError string
	}{
		{
			httpReq:    httptest.NewRequest("GET", "/api/traces/1234?blockEnd=ffffffffffffffffffffffffffffffff&blockStart=00000000000000000000000000000000&mode=blocks&start=1&end=2", nil),
			queryMode:  "blocks",
			startTime:  1,
			endTime:    2,
			blockStart: "00000000000000000000000000000000",
			blockEnd:   "ffffffffffffffffffffffffffffffff",
		},
		{
			httpReq:    httptest.NewRequest("GET", "/api/traces/1234?blockEnd=ffffffffffffffffffffffffffffffff&blockStart=00000000000000000000000000000000&mode=blocks", nil),
			queryMode:  "blocks",
			startTime:  0,
			endTime:    0,
			blockStart: "00000000000000000000000000000000",
			blockEnd:   "ffffffffffffffffffffffffffffffff",
		},
		{
			httpReq:    httptest.NewRequest("GET", "/api/traces/1234?mode=blocks", nil),
			queryMode:  "blocks",
			startTime:  0,
			endTime:    0,
			blockStart: "00000000-0000-0000-0000-000000000000",
			blockEnd:   "FFFFFFFF-FFFF-FFFF-FFFF-FFFFFFFFFFFF",
		},
		{
			httpReq:    httptest.NewRequest("GET", "/api/traces/1234?mode=blocks&blockStart=12345678000000001235000001240000&blockEnd=ffffffffffffffffffffffffffffffff", nil),
			queryMode:  "blocks",
			startTime:  0,
			endTime:    0,
			blockStart: "12345678000000001235000001240000",
			blockEnd:   "ffffffffffffffffffffffffffffffff",
		},
		{
			httpReq:       httptest.NewRequest("GET", "/api/traces/1234?mode=blocks&blockStart=12345678000000001235000001240000&blockEnd=ffffffffffffffffffffffffffffffff&start=1&end=1", nil),
			queryMode:     "blocks",
			startTime:     0,
			endTime:       0,
			blockStart:    "12345678000000001235000001240000",
			blockEnd:      "ffffffffffffffffffffffffffffffff",
			expectedError: "http parameter start must be before end. received start=1 end=1",
		},
	}

	for _, tc := range tests {
		blockStart, blockEnd, queryMode, startTime, endTime, err := ValidateAndSanitizeRequest(tc.httpReq)
		if len(tc.expectedError) != 0 {
			assert.EqualError(t, err, tc.expectedError)
			continue
		}
		assert.NoError(t, err)
		assert.Equal(t, tc.queryMode, queryMode)
		assert.Equal(t, tc.blockStart, blockStart)
		assert.Equal(t, tc.blockEnd, blockEnd)
		assert.Equal(t, tc.startTime, startTime)
		assert.Equal(t, tc.endTime, endTime)
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
				Start:           10,
				End:             20,
				MinDurationMs:   30,
				MaxDurationMs:   40,
				Limit:           50,
				SpansPerSpanSet: 60,
			},
			query: "?start=10&end=20&limit=50&maxDuration=40ms&minDuration=30ms&spss=60&tags=foo%3Dbar",
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
			query: "?start=10&end=20&limit=50&maxDuration=30ms&tags=foo%3Dbar",
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
			query: "?start=10&end=20&limit=50&minDuration=30ms&tags=foo%3Dbar",
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
			query: "?start=10&end=20&maxDuration=40ms&minDuration=30ms&tags=foo%3Dbar",
		},
		{
			req: &tempopb.SearchRequest{
				Tags:          map[string]string{},
				Start:         10,
				End:           20,
				MinDurationMs: 30,
				MaxDurationMs: 40,
			},
			query: "?start=10&end=20&maxDuration=40ms&minDuration=30ms",
		},
		{
			req: &tempopb.SearchRequest{
				Query: "{ foo = `bar` }",
				Start: 10,
				End:   20,
			},
			query: "?start=10&end=20&q=%7B+foo+%3D+%60bar%60+%7D",
		},
	}

	for _, tc := range tests {
		actualURL, err := BuildSearchRequest(tc.httpReq, tc.req)
		assert.NoError(t, err)
		assert.Equal(t, tc.query, actualURL.URL.String())
	}
}

func TestAddServerlessParams(t *testing.T) {
	actualURL := AddServerlessParams(nil, 10)
	assert.Equal(t, "?maxBytes=10", actualURL.URL.String())

	req, err := http.NewRequest("GET", "http://example.com", nil)
	require.NoError(t, err)

	actualURL = AddServerlessParams(req, 10)
	assert.Equal(t, "http://example.com?maxBytes=10", actualURL.URL.String())
}

func TestExtractServerlessParam(t *testing.T) {
	r := httptest.NewRequest("GET", "http://example.com", nil)
	maxBytes, err := ExtractServerlessParams(r)
	require.NoError(t, err)
	assert.Equal(t, 0, maxBytes)

	r = httptest.NewRequest("GET", "http://example.com?maxBytes=13", nil)
	maxBytes, err = ExtractServerlessParams(r)
	require.NoError(t, err)
	assert.Equal(t, 13, maxBytes)

	r = httptest.NewRequest("GET", "http://example.com?maxBytes=blerg", nil)
	_, err = ExtractServerlessParams(r)
	assert.Error(t, err)
}

func Test_parseTimestamp(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name    string
		value   string
		def     time.Time
		want    time.Time
		wantErr bool
	}{
		{"default", "", now, now, false},
		{"unix timestamp", "1571332130", now, time.Unix(1571332130, 0), false},
		{"unix nano timestamp", "1571334162051000000", now, time.Unix(0, 1571334162051000000), false},
		{"unix timestamp with subseconds", "1571332130.934", now, time.Unix(1571332130, 934*1e6), false},
		{"RFC3339 format", "2002-10-02T15:00:00Z", now, time.Date(2002, 10, 0o2, 15, 0, 0, 0, time.UTC), false},
		{"RFC3339nano format", "2009-11-10T23:00:00.000000001Z", now, time.Date(2009, 11, 10, 23, 0, 0, 1, time.UTC), false},
		{"invalid", "we", now, time.Time{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseTimestamp(tt.value, tt.def)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseTimestamp() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseTimestamp() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestQueryRangeRoundtrip(t *testing.T) {
	tcs := []struct {
		name string
		req  *tempopb.QueryRangeRequest
	}{
		{
			name: "empty",
			req:  &tempopb.QueryRangeRequest{},
		},
		{
			name: "not empty!",
			req: &tempopb.QueryRangeRequest{
				Query:     "{ foo = `bar` }",
				Start:     uint64(24 * time.Hour),
				End:       uint64(25 * time.Hour),
				Step:      uint64(30 * time.Second),
				QueryMode: "foo",
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			jsonBytes, err := json.Marshal(tc.req.DedicatedColumns)
			require.NoError(t, err)

			httpReq := BuildQueryRangeRequest(nil, tc.req, string(jsonBytes))
			actualReq, err := ParseQueryRangeRequest(httpReq)
			require.NoError(t, err)
			assert.Equal(t, tc.req, actualReq)
		})
	}
}

func Test_determineBounds(t *testing.T) {
	type args struct {
		now         time.Time
		startString string
		endString   string
		sinceString string
	}
	tests := []struct {
		name    string
		args    args
		start   time.Time
		end     time.Time
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "no start, end, since",
			args: args{
				now:         time.Unix(3600, 0),
				startString: "",
				endString:   "",
				sinceString: "",
			},
			start:   time.Unix(0, 0),    // Default start is one hour before 'now' if nothing is provided
			end:     time.Unix(3600, 0), // Default end is 'now' if nothing is provided
			wantErr: assert.NoError,
		},
		{
			name: "no since or no start with end in the future",
			args: args{
				now:         time.Unix(3600, 0),
				startString: "",
				endString:   "2022-12-18T00:00:00Z",
				sinceString: "",
			},
			start:   time.Unix(0, 0), // Default should be one hour before now
			end:     time.Date(2022, 12, 18, 0, 0, 0, 0, time.UTC),
			wantErr: assert.NoError,
		},
		{
			name: "no since, valid start and end",
			args: args{
				now:         time.Date(2022, 12, 18, 0, 0, 0, 0, time.UTC),
				startString: "2022-12-17T00:00:00Z",
				endString:   "2022-12-18T00:00:00Z",
				sinceString: "",
			},
			start:   time.Date(2022, 12, 17, 0, 0, 0, 0, time.UTC),
			end:     time.Date(2022, 12, 18, 0, 0, 0, 0, time.UTC),
			wantErr: assert.NoError,
		},
		{
			name: "invalid end",
			args: args{
				now:         time.Date(2022, 12, 18, 0, 0, 0, 0, time.UTC),
				startString: "2022-12-17T00:00:00Z",
				endString:   "WHAT TIME IS IT?",
				sinceString: "",
			},
			start: time.Time{},
			end:   time.Time{},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorContains(t, err, "could not parse 'end' parameter:", i...)
			},
		},
		{
			name: "invalid start",
			args: args{
				now:         time.Date(2022, 12, 18, 0, 0, 0, 0, time.UTC),
				startString: "LET'S GOOO",
				endString:   "2022-12-18T00:00:00Z",
				sinceString: "",
			},
			start: time.Time{},
			end:   time.Time{},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorContains(t, err, "could not parse 'start' parameter:", i...)
			},
		},
		{
			name: "invalid since",
			args: args{
				now:         time.Date(2022, 12, 18, 0, 0, 0, 0, time.UTC),
				startString: "2022-12-17T00:00:00Z",
				endString:   "2022-12-18T00:00:00Z",
				sinceString: "HI!",
			},
			start: time.Time{},
			end:   time.Time{},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorContains(t, err, "could not parse 'since' parameter:", i...)
			},
		},
		{
			name: "since 1h with no start or end",
			args: args{
				now:         time.Date(2022, 12, 18, 0, 0, 0, 0, time.UTC),
				startString: "",
				endString:   "",
				sinceString: "1h",
			},
			start:   time.Date(2022, 12, 17, 23, 0, 0, 0, time.UTC),
			end:     time.Date(2022, 12, 18, 0, 0, 0, 0, time.UTC),
			wantErr: assert.NoError,
		},
		{
			name: "since 1d with no start or end",
			args: args{
				now:         time.Date(2022, 12, 18, 0, 0, 0, 0, time.UTC),
				startString: "",
				endString:   "",
				sinceString: "1d",
			},
			start:   time.Date(2022, 12, 17, 0, 0, 0, 0, time.UTC),
			end:     time.Date(2022, 12, 18, 0, 0, 0, 0, time.UTC),
			wantErr: assert.NoError,
		},
		{
			name: "since 1h with no start and end time in the past",
			args: args{
				now:         time.Date(2022, 12, 18, 0, 0, 0, 0, time.UTC),
				startString: "",
				endString:   "2022-12-17T00:00:00Z",
				sinceString: "1h",
			},
			start:   time.Date(2022, 12, 16, 23, 0, 0, 0, time.UTC), // start should be calculated relative to end when end is specified
			end:     time.Date(2022, 12, 17, 0, 0, 0, 0, time.UTC),
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := determineBounds(tt.args.now, tt.args.startString, tt.args.endString, tt.args.sinceString)
			if !tt.wantErr(t, err, fmt.Sprintf("determineBounds(%v, %v, %v, %v)", tt.args.now, tt.args.startString, tt.args.endString, tt.args.sinceString)) {
				return
			}
			assert.Equalf(t, tt.start, got, "determineBounds(%v, %v, %v, %v)", tt.args.now, tt.args.startString, tt.args.endString, tt.args.sinceString)
			assert.Equalf(t, tt.end, got1, "determineBounds(%v, %v, %v, %v)", tt.args.now, tt.args.startString, tt.args.endString, tt.args.sinceString)
		})
	}
}
