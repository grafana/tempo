package backend

import (
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func richBlockMeta(i int, dc DedicatedColumns) *BlockMeta {
	m := NewBlockMetaWithDedicatedColumns("tenant", uuid.New(), "vParquet5", dc)
	m.StartTime = time.Unix(1700000000+int64(i), 0)
	m.EndTime = m.StartTime.Add(time.Duration(i+1) * time.Minute)
	m.TotalObjects = int64(1000 + i)
	m.Size_ = uint64(1<<30 + i)
	m.CompactionLevel = uint32(i % 4)
	m.TotalRecords = uint32(i)
	m.FooterSize = 4096
	m.ReplicationFactor = uint32(i % 2)
	m.IndexPageSize = 250
	m.BloomShardCount = 3
	if i%3 == 0 {
		m.Options = map[string]string{"feature": "x", "other": "y"}
	}
	return m
}

func TestTenantIndexV2RoundTrip(t *testing.T) {
	dcA := DedicatedColumns{
		{Scope: DedicatedColumnScopeSpan, Name: "a", Type: DedicatedColumnTypeString},
		{Scope: DedicatedColumnScopeResource, Name: "b", Type: DedicatedColumnTypeInt, Options: DedicatedColumnOptions{DedicatedColumnOptionArray, DedicatedColumnOptionBlob}},
		{Scope: DedicatedColumnScopeEvent, Name: "c", Type: DedicatedColumnTypeString, Options: DedicatedColumnOptions{DedicatedColumnOptionBlob}},
	}
	dcB := DedicatedColumns{{Scope: DedicatedColumnScopeSpan, Name: "z", Type: DedicatedColumnTypeString}}

	tests := []struct {
		name string
		idx  *TenantIndex
	}{
		{"empty", &TenantIndex{CreatedAt: time.Unix(1, 0)}},
		{"live only", &TenantIndex{
			CreatedAt: time.Unix(1700000000, 123),
			Meta:      []*BlockMeta{richBlockMeta(0, dcA), richBlockMeta(1, dcB), richBlockMeta(2, dcA), richBlockMeta(3, nil)},
		}},
		{"zero times", &TenantIndex{
			CreatedAt: time.Unix(1, 0),
			Meta:      []*BlockMeta{NewBlockMeta("t", uuid.New(), "v2")},
		}},
		{"all sections", &TenantIndex{
			CreatedAt: time.Unix(1700000000, 0),
			Meta:      []*BlockMeta{richBlockMeta(0, dcA), richBlockMeta(1, dcA)},
			CompactedMeta: []*CompactedBlockMeta{
				{BlockMeta: *richBlockMeta(2, dcB), CompactedTime: time.Unix(1700001000, 5)},
				{BlockMeta: *richBlockMeta(3, dcA), CompactedTime: time.Unix(1700002000, 0)},
			},
			NoCompact: []UUID{UUID(uuid.New()), UUID(uuid.New())},
		}},
	}

	for _, tt := range tests {
		for _, zstd := range []bool{false, true} {
			name := tt.name
			if zstd {
				name += "/zstd"
			}
			t.Run(name, func(t *testing.T) {
				var (
					buf []byte
					err error
					got = &TenantIndex{}
				)
				if zstd {
					buf, err = tt.idx.marshalV2Zstd()
					require.NoError(t, err)
					require.NoError(t, got.unmarshalV2Zstd(buf))
				} else {
					buf, err = tt.idx.marshalV2()
					require.NoError(t, err)
					require.NoError(t, got.unmarshalV2(buf))
				}

				require.True(t, tt.idx.CreatedAt.Equal(got.CreatedAt))
				require.Len(t, got.Meta, len(tt.idx.Meta))
				for i, want := range tt.idx.Meta {
					requireBlockMetaEqual(t, want, got.Meta[i])
				}
				require.Len(t, got.CompactedMeta, len(tt.idx.CompactedMeta))
				for i, want := range tt.idx.CompactedMeta {
					requireBlockMetaEqual(t, &want.BlockMeta, &got.CompactedMeta[i].BlockMeta)
					require.True(t, want.CompactedTime.Equal(got.CompactedMeta[i].CompactedTime))
				}
				require.Equal(t, tt.idx.NoCompact, got.NoCompact)
			})
		}
	}
}

func requireBlockMetaEqual(t *testing.T, want, got *BlockMeta) {
	t.Helper()
	require.True(t, want.StartTime.Equal(got.StartTime), "start time")
	require.True(t, want.EndTime.Equal(got.EndTime), "end time")
	w, g := *want, *got
	w.StartTime, w.EndTime, g.StartTime, g.EndTime = time.Time{}, time.Time{}, time.Time{}, time.Time{}
	if len(w.DedicatedColumns) == 0 { // nil and empty are equivalent
		w.DedicatedColumns, g.DedicatedColumns = nil, nil
	}
	require.Equal(t, w, g)
}

// requireAllFieldsSet fails if any field reachable from v is its zero value, so that adding a
// field to BlockMeta (or the index) without populating it in the round trip test below is caught.
func requireAllFieldsSet(t *testing.T, v reflect.Value, path string) {
	t.Helper()
	switch v.Kind() {
	case reflect.Pointer:
		require.False(t, v.IsNil(), path)
		requireAllFieldsSet(t, v.Elem(), path)
	case reflect.Struct:
		if v.Type() == reflect.TypeOf(time.Time{}) {
			require.False(t, v.Interface().(time.Time).IsZero(), path)
			return
		}
		for i := range v.NumField() {
			requireAllFieldsSet(t, v.Field(i), path+"."+v.Type().Field(i).Name)
		}
	case reflect.Slice:
		require.NotZero(t, v.Len(), path)
		if v.Type().Elem().Kind() == reflect.Struct || v.Type().Elem().Kind() == reflect.Pointer {
			for i := range v.Len() {
				requireAllFieldsSet(t, v.Index(i), path)
			}
		}
	default:
		require.False(t, v.IsZero(), path)
	}
}

// TestTenantIndexV2FullRoundTrip populates every field of the v1 index, converts it to V2 and
// back, and requires the result to equal the original.
func TestTenantIndexV2FullRoundTrip(t *testing.T) {
	dc := DedicatedColumns{
		{Scope: DedicatedColumnScopeSpan, Name: "a", Type: DedicatedColumnTypeInt, Options: DedicatedColumnOptions{DedicatedColumnOptionArray}},
		{Scope: DedicatedColumnScopeResource, Name: "b", Type: DedicatedColumnTypeString, Options: DedicatedColumnOptions{DedicatedColumnOptionBlob}},
	}
	full := func(i int) BlockMeta {
		return BlockMeta{
			Version:           "vParquet5",
			BlockID:           UUID(uuid.New()),
			TenantID:          "tenant",
			StartTime:         time.Unix(1700000000+int64(i), 11).UTC(),
			EndTime:           time.Unix(1700003600+int64(i), 22).UTC(),
			TotalObjects:      int64(100 + i),
			Size_:             uint64(1<<30 + i),
			CompactionLevel:   uint32(2 + i),
			IndexPageSize:     250,
			TotalRecords:      uint32(7 + i),
			BloomShardCount:   3,
			FooterSize:        4096,
			DedicatedColumns:  dc,
			ReplicationFactor: 3,
			Options:           map[string]string{"feature": "x", "other": "y"},
		}
	}

	live := []*BlockMeta{{}, {}}
	*live[0], *live[1] = full(0), full(1)
	orig := &TenantIndex{
		CreatedAt: time.Unix(1700010000, 55).UTC(),
		Meta:      live,
		CompactedMeta: []*CompactedBlockMeta{
			{BlockMeta: full(2), CompactedTime: time.Unix(1700007200, 33).UTC()},
			{BlockMeta: full(3), CompactedTime: time.Unix(1700007300, 44).UTC()},
		},
		NoCompact: []UUID{live[0].BlockID},
	}
	requireAllFieldsSet(t, reflect.ValueOf(orig), "TenantIndex")

	// v1 -> v2 -> v1
	v1, err := orig.marshalPb()
	require.NoError(t, err)
	fromV1 := &TenantIndex{}
	require.NoError(t, fromV1.unmarshalPb(v1))

	v2, err := fromV1.marshalV2Zstd()
	require.NoError(t, err)
	got := &TenantIndex{}
	require.NoError(t, got.unmarshalV2Zstd(v2))

	require.Equal(t, orig, got)

	// and the result must still encode as a valid v1 index with the same content. Bytes are not
	// compared because proto marshals the options map in random order.
	again, err := got.marshalPb()
	require.NoError(t, err)
	final := &TenantIndex{}
	require.NoError(t, final.unmarshalPb(again))
	require.Equal(t, orig, final)
}

func TestTenantIndexV2DedupesSchemas(t *testing.T) {
	dc := DedicatedColumns{{Scope: DedicatedColumnScopeSpan, Name: "a", Type: DedicatedColumnTypeString}}
	idx := &TenantIndex{CreatedAt: time.Unix(1, 0)}
	for i := range 10 {
		idx.Meta = append(idx.Meta, richBlockMeta(i, dc))
	}

	buf, err := idx.marshalV2()
	require.NoError(t, err)
	got := &TenantIndex{}
	require.NoError(t, got.unmarshalV2(buf))

	// replication factor alternates and every third block has options, so blocks 0 and 6 share
	// a schema (even, options) as do 2 and 8 (even, no options); blocks within a schema share
	// one dedicated columns backing array and one options map
	require.Same(t, &got.Meta[2].DedicatedColumns[0], &got.Meta[8].DedicatedColumns[0])
	// identical layouts are also interned across schemas by DedicatedColumns.Unmarshal
	require.Same(t, &got.Meta[2].DedicatedColumns[0], &got.Meta[1].DedicatedColumns[0])
	require.Equal(t, map[string]string{"feature": "x", "other": "y"}, got.Meta[6].Options)
	require.Equal(t, reflect.ValueOf(got.Meta[0].Options).Pointer(), reflect.ValueOf(got.Meta[6].Options).Pointer())
	require.Nil(t, got.Meta[2].Options)
}

func TestTenantIndexV2Invalid(t *testing.T) {
	idx := &TenantIndex{
		CreatedAt: time.Unix(1, 0),
		Meta:      []*BlockMeta{richBlockMeta(0, DedicatedColumns{{Scope: DedicatedColumnScopeSpan, Name: "a", Type: DedicatedColumnTypeString}})},
		NoCompact: []UUID{UUID(uuid.New())},
	}
	buf, err := idx.marshalV2()
	require.NoError(t, err)

	t.Run("garbage", func(t *testing.T) {
		require.Error(t, (&TenantIndex{}).unmarshalV2([]byte("nope, not a proto")))
		require.Error(t, (&TenantIndex{}).unmarshalV2Zstd([]byte("nope")))
	})
	t.Run("truncated does not panic", func(_ *testing.T) {
		for n := range len(buf) {
			_ = (&TenantIndex{}).unmarshalV2(buf[:n]) // a cut on a field boundary is valid proto
		}
	})
	t.Run("block references a missing schema", func(t *testing.T) {
		bad := &TenantIndexV2{Blocks: []BlockEntry{{Schema: 3}}}
		raw, err := bad.Marshal()
		require.NoError(t, err)
		require.Error(t, (&TenantIndex{}).unmarshalV2(raw))
	})
}

func FuzzTenantIndexV2Unmarshal(f *testing.F) {
	idx := makeTestTenantIndex(5)
	buf, err := idx.marshalV2()
	require.NoError(f, err)
	f.Add(buf)
	f.Fuzz(func(_ *testing.T, data []byte) {
		_ = (&TenantIndex{}).unmarshalV2(data) // must not panic
	})
}
