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
			err:      "limit must be a positive number",
		},
		{
			name:     "With negative limit",
			urlQuery: "limit=-5",
			err:      "limit must be a positive number",
		},
		{
			name:     "With non-numeric limit",
			urlQuery: "limit=five",
			err:      "strconv.Atoi: parsing \"five\": invalid syntax",
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
		// TODO should we fail this query?
		{
			name:     "With minDuration greater than maxDuration",
			urlQuery: "minDuration=20s&maxDuration=5s",
			expected: &tempopb.SearchRequest{
				Tags:          map[string]string{},
				MinDurationMs: 20000,
				MaxDurationMs: 5000,
				Limit:         q.cfg.SearchDefaultResultLimit,
			},
		},
		{
			name:     "With invalid minDuration",
			urlQuery: "minDuration=10seconds",
			err:      "time: unknown unit \"seconds\" in duration \"10seconds\"",
		},
		{
			name:     "With invalid maxDuration",
			urlQuery: "maxDuration=1msec",
			err:      "time: unknown unit \"msec\" in duration \"1msec\"",
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
