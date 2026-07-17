package backend

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protowire"
)

// compactedBlockMetaFixtureJSON is the gogo-era flat JSON shape that
// (gogoproto.embed) produced for CompactedBlockMeta. Stored
// meta.compacted.json files and tenant index "compacted" entries were
// written in this shape; field values match the BlockMeta golden fixture in
// block_meta_test.go's TestBlockMetaJSONProtoRoundTrip.
const compactedBlockMetaFixtureJSON = `{
    "format": "vParquet3",
    "blockID": "00000000-0000-0000-0000-000000000000",
    "tenantID": "single-tenant",
    "startTime": "2021-01-01T00:00:00Z",
    "endTime": "2021-01-02T00:00:00Z",
    "totalObjects": 10,
    "size": 12345,
    "compactionLevel": 1,
    "indexPageSize": 250000,
    "totalRecords": 124356,
    "bloomShards": 244,
    "footerSize": 15775,
    "dedicatedColumns": [
        {"s": "resource", "n": "namespace"},
        {"s": "resource", "n": "net.host.port", "t": "int"},
        {"n": "http.method"},
        {"n": "namespace"},
        {"n": "http.response.body.size", "t": "int"},
        {"n": "http.request.header.accept", "o": ["array"]}
    ],
    "compactedTime": "2021-01-03T00:00:00Z"
}`

// compactedBlockMetaFixture is the Go-struct equivalent of
// compactedBlockMetaFixtureJSON.
func compactedBlockMetaFixture(t *testing.T) CompactedBlockMeta {
	t.Helper()
	return CompactedBlockMeta{
		BlockMeta: BlockMeta{
			Version:         "vParquet3",
			BlockID:         MustParse("00000000-0000-0000-0000-000000000000"),
			TenantID:        "single-tenant",
			StartTime:       mustParseRFC3339(t, "2021-01-01T00:00:00Z"),
			EndTime:         mustParseRFC3339(t, "2021-01-02T00:00:00Z"),
			TotalObjects:    10,
			Size_:           12345,
			CompactionLevel: 1,
			IndexPageSize:   250000,
			TotalRecords:    124356,
			BloomShardCount: 244,
			FooterSize:      15775,
			DedicatedColumns: DedicatedColumns{
				{Scope: "resource", Name: "namespace", Type: "string"},
				{Scope: "resource", Name: "net.host.port", Type: "int"},
				{Scope: "span", Name: "http.method", Type: "string"},
				{Scope: "span", Name: "namespace", Type: "string"},
				{Scope: "span", Name: "http.response.body.size", Type: "int"},
				{Scope: "span", Name: "http.request.header.accept", Type: "string", Options: DedicatedColumnOptions{DedicatedColumnOptionArray}},
			},
		},
		CompactedTime: mustParseRFC3339(t, "2021-01-03T00:00:00Z"),
	}
}

func mustParseRFC3339(t *testing.T, s string) time.Time {
	t.Helper()
	tm, err := time.Parse(time.RFC3339, s)
	require.NoError(t, err)
	return tm
}

// TestCompactedBlockMetaJSON_GogoShape pins the flat JSON shape gogo's
// (gogoproto.embed) produced for CompactedBlockMeta. wiresmith has no embed
// option, so the shape is reproduced by hand: MarshalJSON in
// wiresmith_custom.go, UnmarshalJSON in block_meta.go.
func TestCompactedBlockMetaJSON_GogoShape(t *testing.T) {
	want := compactedBlockMetaFixture(t)

	var got CompactedBlockMeta
	require.NoError(t, json.Unmarshal([]byte(compactedBlockMetaFixtureJSON), &got))
	assert.Equal(t, want, got)

	marshaled, err := json.Marshal(&got)
	require.NoError(t, err)
	// JSONEq, not byte equality: key order isn't part of the contract, no
	// consumer content-hashes these files.
	assert.JSONEq(t, compactedBlockMetaFixtureJSON, string(marshaled))
}

// TestCompactedBlockMetaJSON_EmptyShape pins MarshalJSON's comma handling
// (the `len(bm) > 2` check in wiresmith_custom.go) for the degenerate
// all-zero CompactedBlockMeta. BlockMeta's JSON tags always emit its scalar
// fields even when zero-valued (no omitempty), so `bm` is never literally
// "{}" today; the check is a guard against a future all-omitempty BlockMeta
// producing one, and this test pins that the comma is placed correctly
// either way.
func TestCompactedBlockMetaJSON_EmptyShape(t *testing.T) {
	var want CompactedBlockMeta

	marshaled, err := json.Marshal(&want)
	require.NoError(t, err)
	require.False(t, bytes.Contains(marshaled, []byte("{,")), "stray leading comma: %s", marshaled)

	var got CompactedBlockMeta
	require.NoError(t, json.Unmarshal(marshaled, &got))
	// cmp.Equal (not assert.Equal): the zero CompactedTime/StartTime/EndTime
	// round-trip through an RFC3339 string and back, which leaves the
	// decoded time.Time with a non-nil UTC Location pointer even though it
	// represents the same zero instant as the untouched struct literal;
	// cmp defers to time.Time's own Equal method and sidesteps that. See
	// TestIndexMarshalUnmarshal for the same pattern.
	assert.True(t, cmp.Equal(want, got))
}

// topLevelFieldNumbers returns the set of protobuf field numbers present at
// the top level of a marshaled message, used to pin which fields
// wiresmith's no-presence stdtime/customtype codecs omit for zero/empty
// values.
func topLevelFieldNumbers(t *testing.T, data []byte) map[protowire.Number]bool {
	t.Helper()
	seen := map[protowire.Number]bool{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		require.Positive(t, n, "malformed tag in %x", data)
		data = data[n:]
		vn := protowire.ConsumeFieldValue(num, typ, data)
		require.GreaterOrEqual(t, vn, 0, "malformed field value in %x", data)
		seen[num] = true
		data = data[vn:]
	}
	return seen
}

// embeddedMessage returns the payload bytes of the first length-delimited
// top-level field with the given number (e.g. CompactedBlockMeta's embedded
// BlockMeta at field 1), so its fields can be inspected in turn.
func embeddedMessage(t *testing.T, data []byte, field protowire.Number) []byte {
	t.Helper()
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		require.Positive(t, n, "malformed tag in %x", data)
		data = data[n:]
		if num == field && typ == protowire.BytesType {
			val, vn := protowire.ConsumeBytes(data)
			require.GreaterOrEqual(t, vn, 0, "malformed bytes field in %x", data)
			return val
		}
		vn := protowire.ConsumeFieldValue(num, typ, data)
		require.GreaterOrEqual(t, vn, 0, "malformed field value in %x", data)
		data = data[vn:]
	}
	t.Fatalf("field %d not found in %x", field, data)
	return nil
}

// TestCompactedBlockMetaProto_WiresmithDivergence pins the two known wire
// divergences between gogo's and wiresmith's generated codecs for
// CompactedBlockMeta/BlockMeta: gogo's stdtime and customtype+nullable=false
// fields (start_time, end_time, compacted_time, dedicated_columns) were
// always written regardless of value (see
// TestCompactedBlockMetaProto_FrozenGogoBytes for gogo's actual output);
// wiresmith generates them with proto3 implicit presence, so a zero
// time.Time or empty DedicatedColumns is omitted from the wire. Both sides
// tolerate either encoding on read (a message decoder simply leaves an
// omitted field at its zero value), so this is a byte-shape pin, not a
// compatibility requirement.
func TestCompactedBlockMetaProto_WiresmithDivergence(t *testing.T) {
	t.Run("fully populated", func(t *testing.T) {
		want := compactedBlockMetaFixture(t)
		data, err := want.Marshal()
		require.NoError(t, err)

		fields := topLevelFieldNumbers(t, data)
		assert.True(t, fields[2], "compacted_time (field 2) should be on the wire")
		bmFields := topLevelFieldNumbers(t, embeddedMessage(t, data, 1))
		assert.True(t, bmFields[6], "start_time (field 6) should be on the wire")
		assert.True(t, bmFields[7], "end_time (field 7) should be on the wire")
		assert.True(t, bmFields[17], "dedicated_columns (field 17) should be on the wire")

		var got CompactedBlockMeta
		require.NoError(t, got.Unmarshal(data))
		assert.Equal(t, want, got)
	})

	t.Run("zero times", func(t *testing.T) {
		want := compactedBlockMetaFixture(t)
		want.BlockMeta.StartTime = time.Time{}
		want.BlockMeta.EndTime = time.Time{}
		want.CompactedTime = time.Time{}
		data, err := want.Marshal()
		require.NoError(t, err)

		fields := topLevelFieldNumbers(t, data)
		assert.False(t, fields[2], "compacted_time (field 2) must be omitted for a zero time")
		bmFields := topLevelFieldNumbers(t, embeddedMessage(t, data, 1))
		assert.False(t, bmFields[6], "start_time (field 6) must be omitted for a zero time")
		assert.False(t, bmFields[7], "end_time (field 7) must be omitted for a zero time")

		var got CompactedBlockMeta
		require.NoError(t, got.Unmarshal(data))
		assert.Equal(t, want, got)
	})

	t.Run("empty dedicated columns", func(t *testing.T) {
		want := compactedBlockMetaFixture(t)
		want.BlockMeta.DedicatedColumns = nil
		data, err := want.Marshal()
		require.NoError(t, err)

		bmFields := topLevelFieldNumbers(t, embeddedMessage(t, data, 1))
		assert.False(t, bmFields[17], "dedicated_columns (field 17) must be omitted when empty")

		var got CompactedBlockMeta
		require.NoError(t, got.Unmarshal(data))
		assert.Equal(t, want, got)
	})
}

// Frozen gogo-era wire bytes, generated on upstream/main at commit
// 6bf6ee34f44501a96e8277377365bec8f91f2965 (pre-migration,
// protoc-gen-gogofaster) by constructing the fixture values below with
// gogo's generated types and calling their .Marshal(). Reconstruction
// recipe:
//
//	bm := BlockMeta{
//		Version: "vParquet3", BlockID: MustParse("00000000-0000-0000-0000-000000000000"),
//		TenantID: "single-tenant",
//		StartTime: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
//		EndTime:   time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
//		TotalObjects: 10, Size_: 12345, CompactionLevel: 1, IndexPageSize: 250000,
//		TotalRecords: 124356, BloomShardCount: 244, FooterSize: 15775,
//		DedicatedColumns: DedicatedColumns{
//			{Scope: "resource", Name: "namespace", Type: "string"},
//			{Scope: "resource", Name: "net.host.port", Type: "int"},
//			{Scope: "span", Name: "http.method", Type: "string"},
//			{Scope: "span", Name: "namespace", Type: "string"},
//			{Scope: "span", Name: "http.response.body.size", Type: "int"},
//			{Scope: "span", Name: "http.request.header.accept", Type: "string", Options: DedicatedColumnOptions{DedicatedColumnOptionArray}},
//		},
//	}
//	cbm := CompactedBlockMeta{BlockMeta: bm, CompactedTime: time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC)}
//	// frozenGogoCompactedBlockMetaHex = hex.EncodeToString(mustMarshal(&cbm))
//	idx := TenantIndex{
//		CreatedAt:     time.Date(2021, 1, 4, 0, 0, 0, 0, time.UTC),
//		Meta:          []*BlockMeta{&bm},
//		CompactedMeta: []*CompactedBlockMeta{&cbm},
//	}
//	// frozenGogoTenantIndexHex = hex.EncodeToString(mustMarshal(&idx))
const (
	frozenGogoCompactedBlockMetaHex = `0aa8020a097650617271756574331210000000000000000000000000000000002a0d73696e676c652d74656e616e7432060880ccb9ff053a060880efbeff05400a48b96050016090a10f68c4cb0778f40180019f7b8a01d2015b7b2273223a227265736f75726365222c226e223a226e616d657370616365227d2c7b2273223a227265736f75726365222c226e223a226e65742e686f73742e706f7274222c2274223a22696e74227d2c7b226e223a22687474702e6d6574686f64227d2c7b226e223a226e616d657370616365227d2c7b226e223a22687474702e726573706f6e73652e626f64792e73697a65222c2274223a22696e74227d2c7b226e223a22687474702e726571756573742e6865616465722e616363657074222c226f223a5b226172726179225d7d5d1206088092c4ff05`

	frozenGogoTenantIndexHex = `0a060880b5c9ff0512a8020a097650617271756574331210000000000000000000000000000000002a0d73696e676c652d74656e616e7432060880ccb9ff053a060880efbeff05400a48b96050016090a10f68c4cb0778f40180019f7b8a01d2015b7b2273223a227265736f75726365222c226e223a226e616d657370616365227d2c7b2273223a227265736f75726365222c226e223a226e65742e686f73742e706f7274222c2274223a22696e74227d2c7b226e223a22687474702e6d6574686f64227d2c7b226e223a226e616d657370616365227d2c7b226e223a22687474702e726573706f6e73652e626f64792e73697a65222c2274223a22696e74227d2c7b226e223a22687474702e726571756573742e6865616465722e616363657074222c226f223a5b226172726179225d7d5d1ab3020aa8020a097650617271756574331210000000000000000000000000000000002a0d73696e676c652d74656e616e7432060880ccb9ff053a060880efbeff05400a48b96050016090a10f68c4cb0778f40180019f7b8a01d2015b7b2273223a227265736f75726365222c226e223a226e616d657370616365227d2c7b2273223a227265736f75726365222c226e223a226e65742e686f73742e706f7274222c2274223a22696e74227d2c7b226e223a22687474702e6d6574686f64227d2c7b226e223a226e616d657370616365227d2c7b226e223a22687474702e726573706f6e73652e626f64792e73697a65222c2274223a22696e74227d2c7b226e223a22687474702e726571756573742e6865616465722e616363657074222c226f223a5b226172726179225d7d5d1206088092c4ff05`
)

// TestCompactedBlockMetaProto_FrozenGogoBytes decodes wire bytes actually
// produced by the pre-migration gogo compiler, proving that blocks and
// tenant indexes already written to production buckets stay readable after
// the wiresmith migration.
func TestCompactedBlockMetaProto_FrozenGogoBytes(t *testing.T) {
	want := compactedBlockMetaFixture(t)

	frozenCompactedBlockMeta, err := hex.DecodeString(frozenGogoCompactedBlockMetaHex)
	require.NoError(t, err)
	var gotCompacted CompactedBlockMeta
	require.NoError(t, gotCompacted.Unmarshal(frozenCompactedBlockMeta))
	assert.Equal(t, want, gotCompacted)

	wantIndex := TenantIndex{
		CreatedAt:     mustParseRFC3339(t, "2021-01-04T00:00:00Z"),
		Meta:          []*BlockMeta{&want.BlockMeta},
		CompactedMeta: []*CompactedBlockMeta{&want},
	}
	frozenTenantIndex, err := hex.DecodeString(frozenGogoTenantIndexHex)
	require.NoError(t, err)
	var gotIndex TenantIndex
	require.NoError(t, gotIndex.Unmarshal(frozenTenantIndex))
	assert.Equal(t, wantIndex, gotIndex)
}

// TestTenantIndexDegenerateRoundTrip mirrors TestIndexMarshalUnmarshal but
// pins the degenerate values exercising the wiresmith/gogo wire divergences
// documented in TestCompactedBlockMetaProto_WiresmithDivergence: a zero
// CreatedAt/CompactedTime and empty DedicatedColumns, round-tripped through
// both TenantIndex encodings.
func TestTenantIndexDegenerateRoundTrip(t *testing.T) {
	idx := &TenantIndex{
		CompactedMeta: []*CompactedBlockMeta{
			{
				BlockMeta: BlockMeta{
					Version:  "vParquet3",
					TenantID: "single-tenant",
				},
			},
		},
	}

	t.Run("json", func(t *testing.T) {
		buff, err := idx.marshal()
		require.NoError(t, err)

		got := &TenantIndex{}
		require.NoError(t, got.unmarshalJSONGz(buff))
		// cmp.Equal: see TestCompactedBlockMetaJSON_EmptyShape.
		assert.True(t, cmp.Equal(idx, got))
	})

	t.Run("proto", func(t *testing.T) {
		buff, err := idx.marshalPb()
		require.NoError(t, err)

		got := &TenantIndex{}
		require.NoError(t, got.unmarshalPb(buff))
		assert.True(t, cmp.Equal(idx, got))
	})
}
