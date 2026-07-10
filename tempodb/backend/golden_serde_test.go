package backend

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The fixtures under testdata/golden/ were serialized by the pre-embed-removal code
// (with gogoproto.embed on CompactedBlockMeta.block_meta). They pin the on-disk formats
// so that removing the embed option cannot silently change them: the current code must
// both decode these historic bytes and re-encode to byte-identical output.
//
// goldenFixture builds the value those fixtures were generated from. It must stay in sync
// with the generator; regenerate the files (not this function) if the schema changes in a
// backward-compatible way.
func goldenFixture() (CompactedBlockMeta, TenantIndex) {
	bm := BlockMeta{
		Version:           "vParquet4",
		BlockID:           MustParse("11111111-2222-3333-4444-555555555555"),
		TenantID:          "test-tenant",
		StartTime:         time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
		EndTime:           time.Date(2021, 1, 2, 3, 4, 5, 0, time.UTC),
		TotalObjects:      42,
		Size_:             123456789,
		CompactionLevel:   3,
		IndexPageSize:     250000,
		TotalRecords:      98765,
		BloomShardCount:   16,
		FooterSize:        4096,
		ReplicationFactor: 3,
		DedicatedColumns: DedicatedColumns{
			{Scope: "resource", Name: "namespace", Type: "string"},
			{Scope: "span", Name: "http.status_code", Type: "int"},
			{Scope: "span", Name: "http.url", Type: "string", Options: DedicatedColumnOptions{DedicatedColumnOptionArray}},
		},
	}
	cbm := CompactedBlockMeta{
		BlockMeta:     bm,
		CompactedTime: time.Date(2021, 1, 3, 12, 0, 0, 0, time.UTC),
	}
	ti := TenantIndex{
		CreatedAt:     time.Date(2021, 1, 3, 12, 30, 0, 0, time.UTC),
		Meta:          []*BlockMeta{&bm},
		CompactedMeta: []*CompactedBlockMeta{&cbm},
	}
	return cbm, ti
}

func TestCompactedBlockMetaGoldenSerde(t *testing.T) {
	cbm, _ := goldenFixture()

	t.Run("json", func(t *testing.T) {
		want, err := os.ReadFile("testdata/golden/compacted_block_meta.json")
		require.NoError(t, err)

		// Encode: current code must reproduce the historic flattened layout byte-for-byte.
		// TrimSpace tolerates a trailing newline the fixture file may pick up from tooling;
		// json.Marshal never emits one.
		got, err := json.Marshal(&cbm)
		require.NoError(t, err)
		assert.Equal(t, strings.TrimSpace(string(want)), string(got))

		// Decode: current code must read the historic bytes back with no loss.
		var decoded CompactedBlockMeta
		require.NoError(t, json.Unmarshal(want, &decoded))
		assert.Equal(t, cbm, decoded)
	})

	t.Run("proto", func(t *testing.T) {
		want, err := os.ReadFile("testdata/golden/compacted_block_meta.pb")
		require.NoError(t, err)

		got, err := cbm.Marshal()
		require.NoError(t, err)
		assert.Equal(t, want, got)

		var decoded CompactedBlockMeta
		require.NoError(t, decoded.Unmarshal(want))
		assert.Equal(t, cbm, decoded)
	})
}

func TestTenantIndexGoldenSerde(t *testing.T) {
	_, ti := goldenFixture()

	t.Run("json", func(t *testing.T) {
		want, err := os.ReadFile("testdata/golden/tenant_index.json")
		require.NoError(t, err)

		got, err := json.Marshal(&ti)
		require.NoError(t, err)
		assert.Equal(t, strings.TrimSpace(string(want)), string(got))

		decoded := &TenantIndex{}
		require.NoError(t, json.Unmarshal(want, decoded))
		assert.Equal(t, &ti, decoded)
	})

	t.Run("proto", func(t *testing.T) {
		want, err := os.ReadFile("testdata/golden/tenant_index.pb")
		require.NoError(t, err)

		got, err := ti.Marshal()
		require.NoError(t, err)
		assert.Equal(t, want, got)

		decoded := &TenantIndex{}
		require.NoError(t, decoded.Unmarshal(want))
		assert.Equal(t, &ti, decoded)
	})
}
