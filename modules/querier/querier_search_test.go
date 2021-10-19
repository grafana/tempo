package querier

import (
	"fmt"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/stretchr/testify/assert"
)

func TestQuerierParseSearchRequest(t *testing.T) {
	q := Querier{
		cfg: Config{
			SearchDefaultResultLimit: 20,
			SearchMaxResultLimit:     100,
		},
	}

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
				Limit: q.cfg.SearchDefaultResultLimit,
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
				Limit: q.cfg.SearchMaxResultLimit,
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
				Limit:         q.cfg.SearchDefaultResultLimit,
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
					"service.name": "foo",
					"limit":        "five",
					"query":        "1+1=2",
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
			err:      "invalid tags: tag service.name has been set twice",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "http://tempo/api/search?"+tt.urlQuery, nil)
			fmt.Println("RequestURI:", r.RequestURI)

			searchRequest, err := q.parseSearchRequest(r)

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
		{"service.name=foo =error", strMap{"service.name": "foo"}},
		{"service.name=\"foo bar\"", strMap{"service.name": "foo bar"}},
		{"service.name=\"foo\\bar\"", strMap{"service.name": "foo\bar"}},
		{"service.name=\"foo \\\"bar\\\"\"", strMap{"service.name": "foo \"bar\""}},
	}

	for _, tt := range tests {
		t.Run(tt.tags, func(t *testing.T) {
			r := httptest.NewRequest("GET", "http://tempo/api/search?tags="+url.QueryEscape(tt.tags), nil)
			fmt.Println("RequestURI:", r.RequestURI)

			searchRequest, err := (&Querier{}).parseSearchRequest(r)

			assert.NoError(t, err)
			assert.Equal(t, tt.expected, searchRequest.Tags)
		})
	}
}
