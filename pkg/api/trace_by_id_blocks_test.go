package api

import (
	"encoding/base64"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protowire"

	"github.com/grafana/tempo/v3/tempodb/backend"
)

func TestTraceByIDBlocksRoundTrip(t *testing.T) {
	start := time.Unix(1700000000, 0).UTC()
	metas := []*backend.BlockMeta{
		{
			Version:         "vParquet5",
			BlockID:         backend.MustParse("00000000-0000-0000-0000-000000000001"),
			TenantID:        "tenant",
			StartTime:       start,
			EndTime:         start.Add(time.Hour),
			Size_:           1024,
			CompactionLevel: 2,
			BloomShardCount: 3,
			FooterSize:      100,
			DedicatedColumns: backend.DedicatedColumns{
				{Scope: backend.DedicatedColumnScopeSpan, Name: "http.method", Type: backend.DedicatedColumnTypeString},
			},
		},
		{
			Version:  "vParquet4",
			BlockID:  backend.MustParse("ffffffff-0000-0000-0000-000000000002"),
			TenantID: "tenant",
		},
		{
			Version:  "vParquet5",
			BlockID:  backend.MustParse("ffffffff-0000-0000-0000-000000000003"),
			TenantID: "tenant",
			DedicatedColumns: backend.DedicatedColumns{
				{Scope: backend.DedicatedColumnScopeResource, Name: "k8s.namespace", Type: backend.DedicatedColumnTypeString},
			},
		},
		{
			Version:  "vParquet5",
			BlockID:  backend.MustParse("ffffffff-0000-0000-0000-000000000004"),
			TenantID: "tenant",
			DedicatedColumns: backend.DedicatedColumns{
				{Scope: backend.DedicatedColumnScopeSpan, Name: "http.method", Type: backend.DedicatedColumnTypeString},
			},
		},
	}

	encoded, err := EncodeTraceByIDBlocks(metas)
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "/querier?"+BlocksKey+"="+encoded, nil)
	actual, err := ParseTraceByIDBlocks(req)
	require.NoError(t, err)
	require.Equal(t, metas, actual)

	// blocks 0 and 3 share dedicated columns, so only 2 of the 3 sets are written
	b, err := base64.RawURLEncoding.DecodeString(encoded)
	require.NoError(t, err)
	b = b[1:]
	columnSets := 0
	for len(b) > 0 {
		num, typ, n := protowire.ConsumeTag(b)
		require.GreaterOrEqual(t, n, 0)
		b = b[n:]
		n = protowire.ConsumeFieldValue(num, typ, b)
		require.GreaterOrEqual(t, n, 0)
		b = b[n:]
		if num == blocksColumnsField {
			columnSets++
		}
	}
	require.Equal(t, 2, columnSets)
}

func TestParseTraceByIDBlocks(t *testing.T) {
	empty, err := EncodeTraceByIDBlocks(nil)
	require.NoError(t, err)

	tests := []struct {
		name     string
		query    string
		expected []*backend.BlockMeta
		wantErr  bool
	}{
		{name: "absent", query: "", expected: nil},
		{name: "empty list", query: BlocksKey + "=" + empty, expected: []*backend.BlockMeta{}},
		{name: "not base64", query: BlocksKey + "=" + url.QueryEscape("!!"), wantErr: true},
		{name: "unsupported version", query: BlocksKey + "=Ag", wantErr: true},
		// version 1, dedicated columns index 1 with no block and no columns
		{name: "dangling columns index", query: BlocksKey + "=ARgB", wantErr: true},
		// version 1, empty block, then dedicated columns index 1 with no columns written
		{name: "columns index past written columns", query: BlocksKey + "=AQoAGAE", wantErr: true},
		// version 1, unknown field 4 as varint
		{name: "unknown field", query: BlocksKey + "=ASAB", wantErr: true},
		// version 1, block field claiming 5 bytes with 1 present
		{name: "truncated block", query: BlocksKey + "=AQoFAA", wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/querier?"+tc.query, nil)
			metas, err := ParseTraceByIDBlocks(req)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.expected, metas)
		})
	}
}
