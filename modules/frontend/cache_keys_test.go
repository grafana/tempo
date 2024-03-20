package frontend

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/stretchr/testify/require"
)

func TestCacheKeyForJob(t *testing.T) {
	tcs := []struct {
		name          string
		queryHash     uint64
		req           *tempopb.SearchRequest
		meta          *backend.BlockMeta
		searchPage    int
		pagesToSearch int

		expected string
	}{
		{
			name:      "valid!",
			queryHash: 42,
			req: &tempopb.SearchRequest{
				Start: 10,
				End:   20,
			},
			meta: &backend.BlockMeta{
				BlockID:   uuid.MustParse("00000000-0000-0000-0000-000000000123"),
				StartTime: time.Unix(15, 0),
				EndTime:   time.Unix(16, 0),
			},
			searchPage:    1,
			pagesToSearch: 2,
			expected:      "sj:42:00000000-0000-0000-0000-000000000123:1:2",
		},
		{
			name:      "no query hash means no query cache",
			queryHash: 0,
			req: &tempopb.SearchRequest{
				Start: 10,
				End:   20,
			},
			meta: &backend.BlockMeta{
				BlockID:   uuid.MustParse("00000000-0000-0000-0000-000000000123"),
				StartTime: time.Unix(15, 0),
				EndTime:   time.Unix(16, 0),
			},
			searchPage:    1,
			pagesToSearch: 2,
			expected:      "",
		},
		{
			name:      "meta before start time",
			queryHash: 42,
			req: &tempopb.SearchRequest{
				Start: 10,
				End:   20,
			},
			meta: &backend.BlockMeta{
				BlockID:   uuid.MustParse("00000000-0000-0000-0000-000000000123"),
				StartTime: time.Unix(5, 0),
				EndTime:   time.Unix(6, 0),
			},
			searchPage:    1,
			pagesToSearch: 2,
			expected:      "",
		},
		{
			name:      "meta overlaps search start",
			queryHash: 42,
			req: &tempopb.SearchRequest{
				Start: 10,
				End:   20,
			},
			meta: &backend.BlockMeta{
				BlockID:   uuid.MustParse("00000000-0000-0000-0000-000000000123"),
				StartTime: time.Unix(5, 0),
				EndTime:   time.Unix(15, 0),
			},
			searchPage:    1,
			pagesToSearch: 2,
			expected:      "",
		},
		{
			name:      "meta overlaps search end",
			queryHash: 42,
			req: &tempopb.SearchRequest{
				Start: 10,
				End:   20,
			},
			meta: &backend.BlockMeta{
				BlockID:   uuid.MustParse("00000000-0000-0000-0000-000000000123"),
				StartTime: time.Unix(15, 0),
				EndTime:   time.Unix(25, 0),
			},
			searchPage:    1,
			pagesToSearch: 2,
			expected:      "",
		},
		{
			name:      "meta after search range",
			queryHash: 42,
			req: &tempopb.SearchRequest{
				Start: 10,
				End:   20,
			},
			meta: &backend.BlockMeta{
				BlockID:   uuid.MustParse("00000000-0000-0000-0000-000000000123"),
				StartTime: time.Unix(25, 0),
				EndTime:   time.Unix(30, 0),
			},
			searchPage:    1,
			pagesToSearch: 2,
			expected:      "",
		},
		{
			name:      "meta encapsulates search range",
			queryHash: 42,
			req: &tempopb.SearchRequest{
				Start: 10,
				End:   20,
			},
			meta: &backend.BlockMeta{
				BlockID:   uuid.MustParse("00000000-0000-0000-0000-000000000123"),
				StartTime: time.Unix(5, 0),
				EndTime:   time.Unix(30, 0),
			},
			searchPage:    1,
			pagesToSearch: 2,
			expected:      "",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			actual := searchJobCacheKey(tc.queryHash, int64(tc.req.Start), int64(tc.req.End), tc.meta, tc.searchPage, tc.pagesToSearch)
			require.Equal(t, tc.expected, actual)
		})
	}
}

func BenchmarkCacheKeyForJob(b *testing.B) {
	req := &tempopb.SearchRequest{
		Start: 10,
		End:   20,
	}
	meta := &backend.BlockMeta{
		BlockID:   uuid.MustParse("00000000-0000-0000-0000-000000000123"),
		StartTime: time.Unix(15, 0),
		EndTime:   time.Unix(16, 0),
	}

	for i := 0; i < b.N; i++ {
		s := searchJobCacheKey(10, int64(req.Start), int64(req.End), meta, 1, 2)
		if len(s) == 0 {
			b.Fatalf("expected non-empty string")
		}
	}
}
