package querier

import (
	"net/http/httptest"
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
			name: "Empty url query",
			expected: &tempopb.SearchRequest{
				Tags:  map[string]string{},
				Limit: q.cfg.SearchDefaultResultLimit,
			},
		},
		{
			name:     "With limit set",
			urlQuery: "limit=10",
			expected: &tempopb.SearchRequest{
				Tags:  map[string]string{},
				Limit: 10,
			},
		},
		{
			name:     "With limit exceeding max",
			urlQuery: "limit=120",
			expected: &tempopb.SearchRequest{
				Tags:  map[string]string{},
				Limit: q.cfg.SearchMaxResultLimit,
			},
		},
		{
			name:     "With zero limit",
			urlQuery: "limit=0",
			err:      "invalid limit: must be a positive number",
		},
		{
			name:     "With negative limit",
			urlQuery: "limit=-5",
			err:      "invalid limit: must be a positive number",
		},
		{
			name:     "With non-numeric limit",
			urlQuery: "limit=five",
			err:      "invalid limit: strconv.Atoi: parsing \"five\": invalid syntax",
		},
		{
			name:     "With minDuration and maxDuration",
			urlQuery: "minDuration=10s&maxDuration=20s",
			expected: &tempopb.SearchRequest{
				Tags:          map[string]string{},
				MinDurationMs: 10000,
				MaxDurationMs: 20000,
				Limit:         q.cfg.SearchDefaultResultLimit,
			},
		},
		{
			name:     "With minDuration greater than maxDuration",
			urlQuery: "minDuration=20s&maxDuration=5s",
			err:      "invalid maxDuration: must be greater than minDuration",
		},
		{
			name:     "With invalid minDuration",
			urlQuery: "minDuration=10seconds",
			err:      "invalid minDuration: time: unknown unit \"seconds\" in duration \"10seconds\"",
		},
		{
			name:     "With invalid maxDuration",
			urlQuery: "maxDuration=1msec",
			err:      "invalid maxDuration: time: unknown unit \"msec\" in duration \"1msec\"",
		},
		{
			name:     "With tags and limit",
			urlQuery: "service.name=foo.bar&limit=5&query=1%2B1%3D2",
			expected: &tempopb.SearchRequest{
				Tags: map[string]string{
					"service.name": "foo.bar",
					"query":        "1+1=2",
				},
				Limit: 5,
			},
		},
		{
			name:     "Tags query parameter",
			urlQuery: "tags=service.name%3Dfoo%20http.url%3Dsearch",
			expected: &tempopb.SearchRequest{
				Tags: map[string]string{
					"service.name": "foo",
					"http.url":     "search",
				},
				Limit: q.cfg.SearchDefaultResultLimit,
			},
		},
		{
			name:     "Tags query parameter with duplicate tag",
			urlQuery: "tags=service.name%3Dfoo%20service.name%3Dbar",
			err:      "invalid tags: tag service.name has been set twice",
		},
		{
			name:     "Tags query parameter with top-level tags",
			urlQuery: "service.id=5&tags=service.name%3Dfoo%20http.url%3Dsearch&test=bar",
			expected: &tempopb.SearchRequest{
				Tags: map[string]string{
					"service.id":   "5",
					"service.name": "foo",
					"http.url":     "search",
					"test":         "bar",
				},
				Limit: q.cfg.SearchDefaultResultLimit,
			},
		},
		{
			name:     "Tags query parameter with top-level tags with duplicate tag",
			urlQuery: "service.name=bar&tags=service.name%3Dfoo%20http.url%3Dsearch&test=bar",
			err:      "invalid tags: tag service.name has been set twice",
		},
		{
			name:     "Tags query parameter with space in value",
			urlQuery: "tags=service.name%3D%22my%20service%22%20http.url%3Dsearch",
			expected: &tempopb.SearchRequest{
				Tags: map[string]string{
					"service.name": "my service",
					"http.url":     "search",
				},
				Limit: q.cfg.SearchDefaultResultLimit,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "http://tempo/api/search?"+tt.urlQuery, nil)

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

func TestParseEncodedTags(t *testing.T) {
	tests := []struct {
		name        string
		encodedTags string
		tags        map[string]string
	}{
		{
			name:        "Tags",
			encodedTags: "service.name=foo http.url=api/search",
			tags: map[string]string{
				"service.name": "foo",
				"http.url":     "api/search",
			},
		},
		{
			name:        "Tag value with space",
			encodedTags: "service.name=\"foo bar\" http.url=api/search",
			tags: map[string]string{
				"service.name": "foo bar",
				"http.url":     "api/search",
			},
		},
		{
			name:        "Tag without value",
			encodedTags: "service.name=\"foo bar\" http.url=api/search error",
			tags: map[string]string{
				"service.name": "foo bar",
				"http.url":     "api/search",
				"error":        "",
			},
		},
		{
			name:        "Tag without name",
			encodedTags: "service.name=\"foo bar\" http.url=api/search =error",
			tags: map[string]string{
				"service.name": "foo bar",
				"http.url":     "api/search",
			},
		},
		{
			name:        "Funky characters",
			encodedTags: "service%name=\"foo=bar\" http&url=\"foo\\\"bar\\\"bzz\"",
			tags: map[string]string{
				"service%name": "foo=bar",
				"http&url":     "foo\"bar\"bzz",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tags := make(map[string]string)

			_ = parseEncodedTags(tt.encodedTags, tags)

			assert.Equal(t, tt.tags, tags)
		})
	}
}
