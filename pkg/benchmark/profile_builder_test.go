package benchmark

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

func TestProfileBlock(t *testing.T) {
	ctx := context.Background()
	meta, r, _ := testBlock(t, 300)

	p, err := ProfileBlock(ctx, meta, r, ProfileOptions{NumTraceIDs: 50})
	require.NoError(t, err)
	require.NoError(t, p.Validate())

	footer, err := rowGroupCount(ctx, meta, r)
	require.NoError(t, err)

	require.Equal(t, ProfileSchemaVersion, p.SchemaVersion)
	require.Equal(t, meta, p.Block)
	require.Equal(t, footer, p.RowGroups)

	require.Equal(t, TraceIDModeSample, p.TraceIDs.Mode)
	require.Len(t, p.TraceIDs.Present, 50)
	require.Len(t, p.TraceIDs.Absent, 50)
}

func TestProfileBlockPresentIDsAreFound(t *testing.T) {
	ctx := context.Background()
	meta, r, _ := testBlock(t, 300)

	p, err := ProfileBlock(ctx, meta, r, ProfileOptions{NumTraceIDs: 20})
	require.NoError(t, err)

	blk, err := encoding.OpenBlock(meta, r)
	require.NoError(t, err)

	for _, hexID := range p.TraceIDs.Present {
		id, err := util.HexStringToTraceID(hexID)
		require.NoError(t, err)

		resp, err := blk.FindTraceByID(ctx, id, common.DefaultSearchOptions())
		require.NoError(t, err)
		require.NotNil(t, resp, "profiled present ID %s was not found", hexID)
		require.NotNil(t, resp.Trace)
	}
}

func TestProfileBlockAbsentIDsAreNotFound(t *testing.T) {
	ctx := context.Background()
	meta, r, _ := testBlock(t, 300)

	p, err := ProfileBlock(ctx, meta, r, ProfileOptions{NumTraceIDs: 5})
	require.NoError(t, err)

	blk, err := encoding.OpenBlock(meta, r)
	require.NoError(t, err)

	for _, hexID := range p.TraceIDs.Absent {
		id, err := util.HexStringToTraceID(hexID)
		require.NoError(t, err)

		resp, err := blk.FindTraceByID(ctx, id, common.DefaultSearchOptions())
		require.NoError(t, err)
		if resp != nil {
			require.Nil(t, resp.Trace, "profiled absent ID %s was found", hexID)
		}
	}
}

// The sample must be stable across calls, otherwise two variants of an
// experiment would look up different IDs and their latencies would not compare.
func TestProfileBlockIsDeterministic(t *testing.T) {
	ctx := context.Background()
	meta, r, _ := testBlock(t, 300)

	first, err := ProfileBlock(ctx, meta, r, ProfileOptions{NumTraceIDs: 40})
	require.NoError(t, err)
	second, err := ProfileBlock(ctx, meta, r, ProfileOptions{NumTraceIDs: 40})
	require.NoError(t, err)

	require.Equal(t, first.TraceIDs, second.TraceIDs)
}

// The sample must spread over each row group's rows, not sit at its head, or a
// The sample must spread over each row group's rows, not sit at its head, or a
// trace-by-ID benchmark reads only the opening pages of each group.
func TestSampleTraceIDsSpreadsOverAllRows(t *testing.T) {
	ctx := context.Background()
	meta, r, _ := testBlock(t, 300)

	blk, err := encoding.OpenBlock(meta, r)
	require.NoError(t, err)

	pf, err := openParquetFile(ctx, meta, r)
	require.NoError(t, err)
	require.Greater(t, len(pf.RowGroups()), 1)

	// Where each ID sits within its row group, in sorted order.
	type place struct{ group, offset, groupLen int }
	placeOf := make(map[string]place)
	for rg, group := range pf.RowGroups() {
		ids, err := listTraceIDs(ctx, blk, group, rg)
		require.NoError(t, err)
		require.Equal(t, int(group.NumRows()), len(ids), "the ID list must be complete")

		slices.SortFunc(ids, bytes.Compare)
		for i, id := range ids {
			placeOf[string(id)] = place{group: rg, offset: i, groupLen: len(ids)}
		}
	}

	const num = 20
	present, _, err := sampleTraceIDs(ctx, blk, pf, num)
	require.NoError(t, err)
	require.Len(t, present, num)

	groups := make(map[int]struct{})
	deepest := 0.0
	for _, hexID := range present {
		id, err := util.HexStringToTraceID(hexID)
		require.NoError(t, err)

		pl, ok := placeOf[string(id)]
		require.True(t, ok, "sampled ID is not in the block")
		groups[pl.group] = struct{}{}
		deepest = max(deepest, float64(pl.offset)/float64(pl.groupLen))
	}

	require.Greater(t, len(groups), 1, "sample should span row groups")
	require.Greater(t, deepest, 0.5, "at least one sampled ID should come from the back half of its row group")
}

// A complete per-row-group ID list is what makes the absent IDs provably
// absent, so the scan must return every row.
func TestListTraceIDsReturnsEveryRow(t *testing.T) {
	ctx := context.Background()
	meta, r, _ := testBlock(t, 300)

	blk, err := encoding.OpenBlock(meta, r)
	require.NoError(t, err)
	pf, err := openParquetFile(ctx, meta, r)
	require.NoError(t, err)

	total := 0
	for rg, group := range pf.RowGroups() {
		ids, err := listTraceIDs(ctx, blk, group, rg)
		require.NoError(t, err)
		require.Equal(t, int(group.NumRows()), len(ids), "row group %d", rg)
		total += len(ids)
	}
	require.Equal(t, int(meta.TotalObjects), total)
}

func TestProfileBlockTraceIDsAll(t *testing.T) {
	ctx := context.Background()
	meta, r, _ := testBlock(t, 100)

	p, err := ProfileBlock(ctx, meta, r, ProfileOptions{NumTraceIDs: TraceIDsAll})
	require.NoError(t, err)

	require.Equal(t, TraceIDModeAll, p.TraceIDs.Mode)
	require.Empty(t, p.TraceIDs.Present)
	require.Empty(t, p.TraceIDs.Absent)
	require.NoError(t, p.Validate())
}

// A block whose bloom filters are not present locally can still be profiled,
// so asking for no trace IDs must not read them.
func TestProfileBlockNoTraceIDs(t *testing.T) {
	ctx := context.Background()
	meta, r, _ := testBlock(t, 50)

	p, err := ProfileBlock(ctx, meta, r, ProfileOptions{NumTraceIDs: 0})
	require.NoError(t, err)

	require.Equal(t, TraceIDModeSample, p.TraceIDs.Mode)
	require.Empty(t, p.TraceIDs.Present)
	require.Empty(t, p.TraceIDs.Absent)
	require.NoError(t, p.Validate())
}

// The footer is authoritative: meta.TotalRecords must not be reported, since it
// can over-count and a shard past the real end of the file reads nothing.
func TestProfileBlockIgnoresStaleRowGroupCount(t *testing.T) {
	ctx := context.Background()
	meta, r, _ := testBlock(t, 300)

	truth, err := rowGroupCount(ctx, meta, r)
	require.NoError(t, err)

	meta.TotalRecords = uint32(truth) + 7

	p, err := ProfileBlock(ctx, meta, r, ProfileOptions{NumTraceIDs: 0})
	require.NoError(t, err)
	require.Equal(t, truth, p.RowGroups)
	require.NoError(t, p.Validate())
}

func TestLoadLocalBlock(t *testing.T) {
	want, _, bucket := testBlock(t, 20)

	path := filepath.Join(bucket, want.TenantID, want.BlockID.String())
	got, r, err := LoadLocalBlock(context.Background(), path)
	require.NoError(t, err)
	require.NotNil(t, r)

	// meta.json is JSON, so times lose their monotonic reading; compare fields.
	require.Equal(t, want.BlockID, got.BlockID)
	require.Equal(t, want.TenantID, got.TenantID)
	require.Equal(t, want.Version, got.Version)
	require.Equal(t, want.TotalObjects, got.TotalObjects)
	require.Equal(t, want.TotalRecords, got.TotalRecords)
	require.Equal(t, want.Size_, got.Size_)
	require.True(t, want.StartTime.Equal(got.StartTime))
	require.True(t, want.EndTime.Equal(got.EndTime))

	// A trailing separator must address the same block.
	got, _, err = LoadLocalBlock(context.Background(), path+string(filepath.Separator))
	require.NoError(t, err)
	require.Equal(t, want.BlockID, got.BlockID)
}

func TestLoadLocalBlockRejectsBadPaths(t *testing.T) {
	ctx := context.Background()

	for _, tc := range []struct {
		name    string
		path    string
		wantErr string
	}{
		{"block dir is not a UUID", filepath.Join(t.TempDir(), "tenant", "not-a-uuid"), "not a block directory"},
		{"no tenant above the block", "/00000000-0000-0000-0000-000000000000", "no tenant directory"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := LoadLocalBlock(ctx, tc.path)
			require.ErrorContains(t, err, tc.wantErr)
		})
	}
}

// Each absent ID must sit strictly between two present IDs: that is what
// Each absent ID must be the midpoint between a present ID and the ID that
// follows it in the block, which is what makes it absent by construction.
func TestAbsentTraceIDsLieNextToPresentIDs(t *testing.T) {
	ctx := context.Background()
	meta, r, _ := testBlock(t, 300)

	blk, err := encoding.OpenBlock(meta, r)
	require.NoError(t, err)
	pf, err := openParquetFile(ctx, meta, r)
	require.NoError(t, err)

	var all [][]byte
	for rg, group := range pf.RowGroups() {
		ids, err := listTraceIDs(ctx, blk, group, rg)
		require.NoError(t, err)
		all = append(all, ids...)
	}
	slices.SortFunc(all, bytes.Compare)

	present, absent, err := sampleTraceIDs(ctx, blk, pf, 30)
	require.NoError(t, err)
	require.Len(t, absent, len(present))

	for i, hexPresent := range present {
		p, err := util.HexStringToTraceID(hexPresent)
		require.NoError(t, err)
		a, err := util.HexStringToTraceID(absent[i])
		require.NoError(t, err)

		at, found := slices.BinarySearchFunc(all, p, bytes.Compare)
		require.True(t, found, "present ID is not in the block")
		require.Less(t, at+1, len(all), "sample should not include the last ID")

		require.Equal(t, midpointTraceID(all[at], all[at+1]), a)
		require.Negative(t, bytes.Compare(p, a), "absent ID should sort after its present ID")
		require.Negative(t, bytes.Compare(a, all[at+1]), "absent ID should sort before the next present ID")
	}
}

// IDs must land across the block's shards rather than on a few of them.
func TestAbsentTraceIDsCoverBloomShards(t *testing.T) {
	ctx := context.Background()
	meta, r, _ := testBlock(t, 300)

	p, err := ProfileBlock(ctx, meta, r, ProfileOptions{NumTraceIDs: 200})
	require.NoError(t, err)

	const shardCount = 28
	seen := make(map[int]struct{}, shardCount)
	for _, hexID := range p.TraceIDs.Absent {
		id, err := util.HexStringToTraceID(hexID)
		require.NoError(t, err)
		seen[common.ShardKeyForTraceID(id, shardCount)] = struct{}{}
	}
	require.Equal(t, shardCount, len(seen))
}

func TestMidpointTraceID(t *testing.T) {
	for _, tc := range []struct{ name, a, b, want string }{
		{"halfway", "00000000000000000000000000000000", "00000000000000000000000000000010", "00000000000000000000000000000008"},
		{"odd gap rounds down", "00000000000000000000000000000000", "00000000000000000000000000000003", "00000000000000000000000000000001"},
		{"adjacent yields the lower", "00000000000000000000000000000004", "00000000000000000000000000000005", "00000000000000000000000000000004"},
		{"equal yields itself", "1111111111111111111111111111111f", "1111111111111111111111111111111f", "1111111111111111111111111111111f"},
		{"carry across the whole width", "00000000000000000000000000000000", "ffffffffffffffffffffffffffffffff", "7fffffffffffffffffffffffffffffff"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, err := util.HexStringToTraceID(tc.a)
			require.NoError(t, err)
			b, err := util.HexStringToTraceID(tc.b)
			require.NoError(t, err)

			got := midpointTraceID(a, b)
			require.Len(t, got, 16)
			require.Equal(t, tc.want, util.PadTraceIDString(util.TraceIDToHexString(got)))
		})
	}
}

// Absent IDs are derived from adjacency rather than looked up, so a block whose
// bloom filters are missing can still be profiled in full.
func TestProfileBlockWithoutBloomFilters(t *testing.T) {
	ctx := context.Background()
	meta, r, bucket := testBlock(t, 300)

	blockDir := filepath.Join(bucket, meta.TenantID, meta.BlockID.String())
	blooms, err := filepath.Glob(filepath.Join(blockDir, "bloom-*"))
	require.NoError(t, err)
	require.NotEmpty(t, blooms, "fixture should have bloom filters to remove")
	for _, f := range blooms {
		require.NoError(t, os.Remove(f))
	}

	p, err := ProfileBlock(ctx, meta, r, ProfileOptions{NumTraceIDs: 20})
	require.NoError(t, err)
	require.Len(t, p.TraceIDs.Present, 20)
	require.Len(t, p.TraceIDs.Absent, 20)
	require.NoError(t, p.Validate())
}
