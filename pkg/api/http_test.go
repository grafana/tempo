package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

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
				Limit:           defaultLimit,
				SpansPerSpanSet: defaultSpansPerSpanSet,
			},
		},
		{
			name:     "zero ranges",
			urlQuery: "start=0&end=0",
			expected: &tempopb.SearchRequest{
				Tags:            map[string]string{},
				Limit:           defaultLimit,
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
				Limit:           defaultLimit,
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
				Limit:           defaultLimit,
				SpansPerSpanSet: defaultSpansPerSpanSet,
			},
		},
		{
			name:     "invalid traceql query",
			urlQuery: "q=" + url.QueryEscape(`{ .foo="bar" `),
			err:      "invalid TraceQL query: parse error at line 1, col 14: syntax error: unexpected $end",
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
				Limit:           defaultLimit,
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
				Limit:           defaultLimit,
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
				Limit:           defaultLimit,
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
				Limit:           defaultLimit,
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
				Limit:           defaultLimit,
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
				Limit:           defaultLimit,
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
			query: "?blockID=b92ec614-3fd7-4299-b6db-f657e7025a9b&dataEncoding=v1&encoding=s2&footerSize=2000&indexPageSize=10&pagesToSearch=10&size=1000&startPage=0&totalRecords=11&version=v2",
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
			query:   "/test/path?blockID=b92ec614-3fd7-4299-b6db-f657e7025a9b&dataEncoding=v1&encoding=s2&footerSize=2000&indexPageSize=10&pagesToSearch=10&size=1000&startPage=0&totalRecords=11&version=v2",
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
			query: "?blockID=b92ec614-3fd7-4299-b6db-f657e7025a9b&dataEncoding=v1&encoding=s2&end=20&footerSize=2000&indexPageSize=10&limit=50&maxDuration=40ms&minDuration=30ms&pagesToSearch=10&size=1000&start=10&startPage=0&tags=foo%3Dbar&totalRecords=11&version=v2",
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
			query:   "/test/path?blockID=b92ec614-3fd7-4299-b6db-f657e7025a9b&dataEncoding=&dc=%5B%7B%22scope%22%3A1%2C%22name%22%3A%22net.sock.host.addr%22%7D%5D&encoding=none&footerSize=2000&indexPageSize=0&pagesToSearch=10&size=1000&startPage=0&totalRecords=2&version=vParquet3",
		},
	}

	for _, tc := range tests {
		actualURL, err := BuildSearchBlockRequest(tc.httpReq, tc.req)
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
			query: "?end=20&limit=50&maxDuration=40ms&minDuration=30ms&spss=60&start=10&tags=foo%3Dbar",
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
		{
			req: &tempopb.SearchRequest{
				Query: "{ foo = `bar` }",
				Start: 10,
				End:   20,
			},
			query: "?end=20&q=%7B+foo+%3D+%60bar%60+%7D&start=10",
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
