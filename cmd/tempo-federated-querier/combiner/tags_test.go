package combiner

import (
	"net/http"
	"testing"

	"github.com/go-kit/log"
	"github.com/grafana/tempo/pkg/tempopb"
)

func TestCombineTagsResults(t *testing.T) {
	logger := log.NewNopLogger()
	c := New(10*1024*1024, logger)

	tests := []struct {
		name          string
		results       []QueryResult
		wantTagsCount int
	}{
		{
			name:          "empty results",
			results:       []QueryResult{},
			wantTagsCount: 0,
		},
		{
			name: "single instance with tags",
			results: []QueryResult{
				{
					Instance: "inst1",
					Response: &http.Response{StatusCode: http.StatusOK},
					Body:     []byte(`{"tagNames":["service.name","http.method","http.status_code"]}`),
				},
			},
			wantTagsCount: 3,
		},
		{
			name: "multiple instances with duplicate tags",
			results: []QueryResult{
				{
					Instance: "inst1",
					Response: &http.Response{StatusCode: http.StatusOK},
					Body:     []byte(`{"tagNames":["service.name","http.method"]}`),
				},
				{
					Instance: "inst2",
					Response: &http.Response{StatusCode: http.StatusOK},
					Body:     []byte(`{"tagNames":["service.name","http.status_code"]}`),
				},
			},
			wantTagsCount: 3, // Deduplicated: service.name, http.method, http.status_code
		},
		{
			name: "skip 404 responses",
			results: []QueryResult{
				{
					Instance: "inst1",
					Response: &http.Response{StatusCode: http.StatusOK},
					Body:     []byte(`{"tagNames":["tag1"]}`),
				},
				{
					Instance: "inst2",
					Response: &http.Response{StatusCode: http.StatusNotFound},
				},
			},
			wantTagsCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := c.CombineTagsResults(tt.results)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(resp.TagNames) != tt.wantTagsCount {
				t.Errorf("tags count = %d, want %d", len(resp.TagNames), tt.wantTagsCount)
			}
		})
	}
}

func TestCombineTagsV2Results(t *testing.T) {
	logger := log.NewNopLogger()
	c := New(10*1024*1024, logger)

	tests := []struct {
		name            string
		results         []QueryResult
		wantScopesCount int
	}{
		{
			name: "single instance with scopes",
			results: []QueryResult{
				{
					Instance: "inst1",
					Response: &http.Response{StatusCode: http.StatusOK},
					Body:     []byte(`{"scopes":[{"name":"resource","tags":["service.name"]},{"name":"span","tags":["http.method"]}]}`),
				},
			},
			wantScopesCount: 2,
		},
		{
			name: "multiple instances merge scopes",
			results: []QueryResult{
				{
					Instance: "inst1",
					Response: &http.Response{StatusCode: http.StatusOK},
					Body:     []byte(`{"scopes":[{"name":"resource","tags":["service.name"]}]}`),
				},
				{
					Instance: "inst2",
					Response: &http.Response{StatusCode: http.StatusOK},
					Body:     []byte(`{"scopes":[{"name":"resource","tags":["service.namespace"]},{"name":"span","tags":["http.method"]}]}`),
				},
			},
			wantScopesCount: 2, // resource and span scopes merged
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := c.CombineTagsV2Results(tt.results)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(resp.Scopes) != tt.wantScopesCount {
				t.Errorf("scopes count = %d, want %d", len(resp.Scopes), tt.wantScopesCount)
			}
		})
	}
}

func TestCombineTagValuesResults(t *testing.T) {
	logger := log.NewNopLogger()
	c := New(10*1024*1024, logger)

	tests := []struct {
		name            string
		results         []QueryResult
		wantValuesCount int
	}{
		{
			name: "single instance with values",
			results: []QueryResult{
				{
					Instance: "inst1",
					Response: &http.Response{StatusCode: http.StatusOK},
					Body:     []byte(`{"tagValues":["value1","value2","value3"]}`),
				},
			},
			wantValuesCount: 3,
		},
		{
			name: "multiple instances with duplicate values",
			results: []QueryResult{
				{
					Instance: "inst1",
					Response: &http.Response{StatusCode: http.StatusOK},
					Body:     []byte(`{"tagValues":["value1","value2"]}`),
				},
				{
					Instance: "inst2",
					Response: &http.Response{StatusCode: http.StatusOK},
					Body:     []byte(`{"tagValues":["value2","value3"]}`),
				},
			},
			wantValuesCount: 3, // Deduplicated
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := c.CombineTagValuesResults(tt.results)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(resp.TagValues) != tt.wantValuesCount {
				t.Errorf("values count = %d, want %d", len(resp.TagValues), tt.wantValuesCount)
			}
		})
	}
}

func TestCombineTagValuesV2Results(t *testing.T) {
	logger := log.NewNopLogger()
	c := New(10*1024*1024, logger)

	tests := []struct {
		name            string
		results         []QueryResult
		wantValuesCount int
	}{
		{
			name: "single instance with typed values",
			results: []QueryResult{
				{
					Instance: "inst1",
					Response: &http.Response{StatusCode: http.StatusOK},
					Body:     []byte(`{"tagValues":[{"type":"string","value":"val1"},{"type":"int","value":"123"}]}`),
				},
			},
			wantValuesCount: 2,
		},
		{
			name: "multiple instances deduplicate by value",
			results: []QueryResult{
				{
					Instance: "inst1",
					Response: &http.Response{StatusCode: http.StatusOK},
					Body:     []byte(`{"tagValues":[{"type":"string","value":"val1"}]}`),
				},
				{
					Instance: "inst2",
					Response: &http.Response{StatusCode: http.StatusOK},
					Body:     []byte(`{"tagValues":[{"type":"string","value":"val1"},{"type":"string","value":"val2"}]}`),
				},
			},
			wantValuesCount: 2, // val1 deduplicated
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := c.CombineTagValuesV2Results(tt.results)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(resp.TagValues) != tt.wantValuesCount {
				t.Errorf("values count = %d, want %d", len(resp.TagValues), tt.wantValuesCount)
			}
		})
	}
}

func TestCombineMetadataMetrics(t *testing.T) {
	tests := []struct {
		name     string
		existing *tempopb.MetadataMetrics
		incoming *tempopb.MetadataMetrics
		want     *tempopb.MetadataMetrics
	}{
		{
			name:     "nil existing",
			existing: nil,
			incoming: &tempopb.MetadataMetrics{TotalBlocks: 10},
			want:     &tempopb.MetadataMetrics{TotalBlocks: 10},
		},
		{
			name:     "nil incoming",
			existing: &tempopb.MetadataMetrics{TotalBlocks: 10},
			incoming: nil,
			want:     &tempopb.MetadataMetrics{TotalBlocks: 10},
		},
		{
			name:     "both values sum",
			existing: &tempopb.MetadataMetrics{TotalBlocks: 10, TotalJobs: 5},
			incoming: &tempopb.MetadataMetrics{TotalBlocks: 20, TotalJobs: 10},
			want:     &tempopb.MetadataMetrics{TotalBlocks: 30, TotalJobs: 15},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := combineMetadataMetrics(tt.existing, tt.incoming)

			if result == nil && tt.want != nil {
				t.Fatal("expected non-nil result")
			}
			if result != nil && tt.want == nil {
				t.Fatal("expected nil result")
			}
			if result != nil {
				if result.TotalBlocks != tt.want.TotalBlocks {
					t.Errorf("TotalBlocks = %d, want %d", result.TotalBlocks, tt.want.TotalBlocks)
				}
				if result.TotalJobs != tt.want.TotalJobs {
					t.Errorf("TotalJobs = %d, want %d", result.TotalJobs, tt.want.TotalJobs)
				}
			}
		})
	}
}
