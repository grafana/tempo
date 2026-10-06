package frontend

import (
	"testing"
	"time"

	"github.com/grafana/tempo/v3/pkg/tempopb"
	"github.com/grafana/tempo/v3/tempodb/backend"
	"github.com/stretchr/testify/require"
)

func TestCacheKeyForJob(t *testing.T) {
	tcs := []struct {
		name          string
		tenant        string
		queryHash     uint64
		req           *tempopb.SearchRequest
		meta          *backend.BlockMeta
		searchPage    int
		pagesToSearch int

		expected string
	}{
		{
			name:      "valid!",
			tenant:    "foo",
			queryHash: 42,
			req: &tempopb.SearchRequest{
				Start: 10,
				End:   20,
			},
			meta: &backend.BlockMeta{
				BlockID:   backend.MustParse("00000000-0000-0000-0000-000000000123"),
				StartTime: time.Unix(15, 0),
				EndTime:   time.Unix(16, 0),
			},
			searchPage:    1,
			pagesToSearch: 2,
			expected:      "sj:foo:42:00000000-0000-0000-0000-000000000123:1:2",
		},
		{
			name:      "no query hash means no query cache",
			queryHash: 0,
			req: &tempopb.SearchRequest{
				Start: 10,
				End:   20,
			},
			meta: &backend.BlockMeta{
				BlockID:   backend.MustParse("00000000-0000-0000-0000-000000000123"),
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
				BlockID:   backend.MustParse("00000000-0000-0000-0000-000000000123"),
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
				BlockID:   backend.MustParse("00000000-0000-0000-0000-000000000123"),
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
				BlockID:   backend.MustParse("00000000-0000-0000-0000-000000000123"),
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
				BlockID:   backend.MustParse("00000000-0000-0000-0000-000000000123"),
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
				BlockID:   backend.MustParse("00000000-0000-0000-0000-000000000123"),
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
			startTime := time.Unix(int64(tc.req.Start), 0)
			endTime := time.Unix(int64(tc.req.End), 0)

			actual := searchJobCacheKey(tc.tenant, tc.queryHash, startTime, endTime, tc.meta, tc.searchPage, tc.pagesToSearch)
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
		BlockID:   backend.MustParse("00000000-0000-0000-0000-000000000123"),
		StartTime: time.Unix(15, 0),
		EndTime:   time.Unix(16, 0),
	}

	startTime := time.Unix(int64(req.Start), 0)
	endTime := time.Unix(int64(req.End), 0)

	for i := 0; i < b.N; i++ {
		s := searchJobCacheKey("foo", 10, startTime, endTime, meta, 1, 2)
		if len(s) == 0 {
			b.Fatalf("expected non-empty string")
		}
	}
}

func TestTraceByIDJobCacheKey(t *testing.T) {
	traceID := []byte{0x01, 0x02, 0x03}
	base := traceByIDJobCacheKey("foo", "v2", traceID, 1000, "AQoAGAA")
	require.NotEmpty(t, base)
	require.Equal(t, base, traceByIDJobCacheKey("foo", "v2", traceID, 1000, "AQoAGAA"))

	tcs := []struct {
		name     string
		tenant   string
		api      string
		traceID  []byte
		maxBytes int
		blocks   string
	}{
		{name: "tenant", tenant: "bar", api: "v2", traceID: traceID, maxBytes: 1000, blocks: "AQoAGAA"},
		{name: "api version", tenant: "foo", api: "v1", traceID: traceID, maxBytes: 1000, blocks: "AQoAGAA"},
		{name: "trace id", tenant: "foo", api: "v2", traceID: []byte{0x01, 0x02, 0x04}, maxBytes: 1000, blocks: "AQoAGAA"},
		{name: "max bytes per trace", tenant: "foo", api: "v2", traceID: traceID, maxBytes: 2000, blocks: "AQoAGAA"},
		{name: "blocks", tenant: "foo", api: "v2", traceID: traceID, maxBytes: 1000, blocks: "AQoAGAE"},
	}
	for _, tc := range tcs {
		t.Run(tc.name+" changes the key", func(t *testing.T) {
			require.NotEqual(t, base, traceByIDJobCacheKey(tc.tenant, tc.api, tc.traceID, tc.maxBytes, tc.blocks))
		})
	}

	t.Run("no blocks has no key", func(t *testing.T) {
		require.Empty(t, traceByIDJobCacheKey("foo", "v2", traceID, 1000, ""))
	})
}
