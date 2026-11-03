package search

import (
	"testing"

	"github.com/grafana/tempo/v3/pkg/tempopb"
	"github.com/stretchr/testify/require"
)

func TestNaiveTagValuesV2Combine(t *testing.T) {
	tests := []struct {
		name     string
		rNew     *tempopb.SearchTagValuesV2Response
		rInto    *tempopb.SearchTagValuesV2Response
		expected *tempopb.SearchTagValuesV2Response
	}{
		{
			name: "combine unique tag values",
			rNew: &tempopb.SearchTagValuesV2Response{
				TagValues: []*tempopb.TagValue{
					{Type: "string", Value: "value1"},
					{Type: "string", Value: "value2"},
				},
				Metrics: &tempopb.MetadataMetrics{
					InspectedBytes: 100,
					CompletedJobs:  1,
				},
			},
			rInto: &tempopb.SearchTagValuesV2Response{
				TagValues: []*tempopb.TagValue{
					{Type: "string", Value: "value3"},
				},
				Metrics: &tempopb.MetadataMetrics{
					InspectedBytes: 50,
					CompletedJobs:  1,
				},
			},
			expected: &tempopb.SearchTagValuesV2Response{
				TagValues: []*tempopb.TagValue{
					{Type: "string", Value: "value3"},
					{Type: "string", Value: "value1"},
					{Type: "string", Value: "value2"},
				},
				Metrics: &tempopb.MetadataMetrics{
					InspectedBytes: 150,
					CompletedJobs:  2,
				},
			},
		},
		{
			name: "skip duplicate tag values",
			rNew: &tempopb.SearchTagValuesV2Response{
				TagValues: []*tempopb.TagValue{
					{Type: "string", Value: "value1"},
					{Type: "string", Value: "value2"},
				},
				Metrics: &tempopb.MetadataMetrics{
					InspectedBytes: 100,
					CompletedJobs:  1,
				},
			},
			rInto: &tempopb.SearchTagValuesV2Response{
				TagValues: []*tempopb.TagValue{
					{Type: "string", Value: "value1"},
				},
				Metrics: &tempopb.MetadataMetrics{
					InspectedBytes: 50,
					CompletedJobs:  1,
				},
			},
			expected: &tempopb.SearchTagValuesV2Response{
				TagValues: []*tempopb.TagValue{
					{Type: "string", Value: "value1"},
					{Type: "string", Value: "value2"},
				},
				Metrics: &tempopb.MetadataMetrics{
					InspectedBytes: 150,
					CompletedJobs:  2,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			naiveTagValuesV2Combine(tt.rNew, tt.rInto)
			require.Equal(t, tt.expected, tt.rInto)
		})
	}
}

func TestNaiveTagsV2Combine(t *testing.T) {
	tests := []struct {
		name     string
		rNew     *tempopb.SearchTagsV2Response
		rInto    *tempopb.SearchTagsV2Response
		expected int // expected number of unique tags across all scopes
	}{
		{
			name: "combine unique tags from different scopes",
			rNew: &tempopb.SearchTagsV2Response{
				Scopes: []*tempopb.SearchTagsV2Scope{
					{
						Name: "resource",
						Tags: []string{"tag1", "tag2"},
					},
				},
				Metrics: &tempopb.MetadataMetrics{
					InspectedBytes: 100,
					CompletedJobs:  1,
				},
			},
			rInto: &tempopb.SearchTagsV2Response{
				Scopes: []*tempopb.SearchTagsV2Scope{
					{
						Name: "span",
						Tags: []string{"tag3"},
					},
				},
				Metrics: &tempopb.MetadataMetrics{
					InspectedBytes: 50,
					CompletedJobs:  1,
				},
			},
			expected: 3, // tag1, tag2, tag3
		},
		{
			name: "deduplicate tags within same scope",
			rNew: &tempopb.SearchTagsV2Response{
				Scopes: []*tempopb.SearchTagsV2Scope{
					{
						Name: "resource",
						Tags: []string{"tag1", "tag2"},
					},
				},
				Metrics: &tempopb.MetadataMetrics{
					InspectedBytes: 100,
					CompletedJobs:  1,
				},
			},
			rInto: &tempopb.SearchTagsV2Response{
				Scopes: []*tempopb.SearchTagsV2Scope{
					{
						Name: "resource",
						Tags: []string{"tag1", "tag3"},
					},
				},
				Metrics: &tempopb.MetadataMetrics{
					InspectedBytes: 50,
					CompletedJobs:  1,
				},
			},
			expected: 3, // tag1, tag2, tag3
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			naiveTagsV2Combine(tt.rNew, tt.rInto)

			// Count total unique tags across all scopes
			totalTags := 0
			for _, scope := range tt.rInto.Scopes {
				totalTags += len(scope.Tags)
			}
			require.Equal(t, tt.expected, totalTags)

			// Verify metrics were combined
			expectedBytes := tt.rNew.Metrics.InspectedBytes + 50 // original rInto value
			expectedJobs := tt.rNew.Metrics.CompletedJobs + 1    // original rInto value
			require.Equal(t, expectedBytes, tt.rInto.Metrics.InspectedBytes)
			require.Equal(t, expectedJobs, tt.rInto.Metrics.CompletedJobs)
		})
	}
}
