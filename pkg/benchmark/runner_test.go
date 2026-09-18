package benchmark

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/tempodb/backend"
)

func blockPath(bucket string, meta *backend.BlockMeta) string {
	return filepath.Join(bucket, meta.TenantID, meta.BlockID.String())
}

func TestRun(t *testing.T) {
	ctx := context.Background()
	meta, r, bucket := testBlock(t, 300)

	profile, err := ProfileBlock(ctx, meta, r, ProfileOptions{NumTraceIDs: 25})
	require.NoError(t, err)

	result, err := Run(ctx, blockPath(bucket, meta), profile, RunOptions{})
	require.NoError(t, err)

	require.Equal(t, ResultSchemaVersion, result.SchemaVersion)
	require.Equal(t, profile.Block.BlockID.String(), result.Profile.BlockID)
	require.Equal(t, 25, result.Profile.PresentIDs)
	require.Positive(t, result.DurationNs)
	require.Len(t, result.Cases, 3)

	byID := map[string]CaseResult{}
	for _, c := range result.Cases {
		require.Empty(t, c.Error, "case %s", c.ID)
		require.Positive(t, c.Executions, "case %s", c.ID)
		require.Positive(t, c.WallNs.Max, "case %s", c.ID)
		require.LessOrEqual(t, c.WallNs.Min, c.WallNs.P50)
		require.LessOrEqual(t, c.WallNs.P50, c.WallNs.P99)
		require.LessOrEqual(t, c.WallNs.P99, c.WallNs.Max)
		byID[c.ID] = c
	}

	// Every present ID must be found, and no absent one.
	present := byID["traceid/present"]
	require.Equal(t, 25, present.Executions)
	require.Equal(t, int64(25), present.Matched)

	absent := byID["traceid/absent"]
	require.Equal(t, 25, absent.Executions)
	require.Zero(t, absent.Matched, "an absent ID was found, so the profile is wrong")

	search := byID["search/nopredicate"]
	require.Positive(t, search.Matched)
	require.Positive(t, search.InspectedBytes)

	// The block is read through a counting reader, so I/O must be observed.
	require.Positive(t, present.Backend.Reads)
	require.Positive(t, present.Backend.Bytes)
}

func TestRunRepeatMultipliesExecutions(t *testing.T) {
	ctx := context.Background()
	meta, r, bucket := testBlock(t, 200)

	profile, err := ProfileBlock(ctx, meta, r, ProfileOptions{NumTraceIDs: 10})
	require.NoError(t, err)

	result, err := Run(ctx, blockPath(bucket, meta), profile, RunOptions{Repeat: 3, Warmup: 1})
	require.NoError(t, err)

	for _, c := range result.Cases {
		if c.API != apiTraceByID {
			continue
		}
		require.Equal(t, 30, c.Executions, "case %s", c.ID)
	}
}

// A profile from another block would make every lookup a miss, which would read
// as a fast run rather than a broken one.
func TestRunRejectsForeignProfile(t *testing.T) {
	ctx := context.Background()
	meta, r, bucket := testBlock(t, 100)

	profile, err := ProfileBlock(ctx, meta, r, ProfileOptions{NumTraceIDs: 5})
	require.NoError(t, err)

	other := *profile.Block
	other.BlockID = backend.UUID(uuid.New())
	foreign := *profile
	foreign.Block = &other

	_, err = Run(ctx, blockPath(bucket, meta), &foreign, RunOptions{})
	require.ErrorContains(t, err, "profile is for block")
}

func TestRunRejectsAllMode(t *testing.T) {
	ctx := context.Background()
	meta, r, bucket := testBlock(t, 100)

	profile, err := ProfileBlock(ctx, meta, r, ProfileOptions{NumTraceIDs: TraceIDsAll})
	require.NoError(t, err)

	_, err = Run(ctx, blockPath(bucket, meta), profile, RunOptions{})
	require.ErrorContains(t, err, "embeds no IDs")
}

func TestResultRoundTrip(t *testing.T) {
	ctx := context.Background()
	meta, r, bucket := testBlock(t, 100)

	profile, err := ProfileBlock(ctx, meta, r, ProfileOptions{NumTraceIDs: 5})
	require.NoError(t, err)

	want, err := Run(ctx, blockPath(bucket, meta), profile, RunOptions{})
	require.NoError(t, err)

	var buf bytes.Buffer
	require.NoError(t, want.Write(&buf))

	got, err := LoadResult(&buf)
	require.NoError(t, err)
	require.Equal(t, want.Profile, got.Profile)
	require.Equal(t, want.Options, got.Options)
	require.Len(t, got.Cases, len(want.Cases))
}

func TestThin(t *testing.T) {
	samples := make([]int64, 100)
	for i := range samples {
		samples[i] = int64(i)
	}

	require.Equal(t, samples, thin(samples, 100))
	require.Equal(t, samples, thin(samples, 1000))

	got := thin(samples, 10)
	require.Len(t, got, 10)
	require.Equal(t, int64(0), got[0])
	require.Equal(t, int64(90), got[9], "should stride across the whole set")
}

func TestSummarize(t *testing.T) {
	require.Equal(t, Summary{}, summarize(nil))

	s := summarize([]int64{5, 1, 4, 2, 3})
	require.Equal(t, 5, s.Count)
	require.Equal(t, int64(1), s.Min)
	require.Equal(t, int64(3), s.P50)
	require.Equal(t, int64(5), s.Max)
	require.InDelta(t, 3.0, s.Mean, 0.001)

	// Percentiles are nearest rank, so they are values that were measured.
	one := summarize([]int64{42})
	require.Equal(t, int64(42), one.Min)
	require.Equal(t, int64(42), one.P99)
	require.Zero(t, one.StdDev)
}

func TestShardsForBlock(t *testing.T) {
	meta := func(size uint64, rowGroups uint32) *backend.BlockMeta {
		m := backend.NewBlockMeta("t", uuid.New(), "vParquet5")
		m.Size_, m.TotalRecords = size, rowGroups
		return m
	}

	for _, tc := range []struct {
		name        string
		meta        *backend.BlockMeta
		targetBytes int
		want        []Shard
	}{
		{"block smaller than the target is one shard", meta(50, 4), 100, []Shard{{0, 0, 4}}},
		{"one row group per shard when target-sized", meta(400, 4), 100, []Shard{{0, 0, 1}, {1, 1, 1}, {2, 2, 1}, {3, 3, 1}}},
		{"small row groups grouped up to the target", meta(400, 8), 100, []Shard{{0, 0, 2}, {1, 2, 2}, {2, 4, 2}, {3, 6, 2}}},
		{"last shard is short when it does not divide", meta(500, 5), 200, []Shard{{0, 0, 2}, {1, 2, 2}, {2, 4, 1}}},
		{"row groups larger than the target get one each", meta(1000, 2), 100, []Shard{{0, 0, 1}, {1, 1, 1}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := shardsForBlock(tc.meta, int(tc.meta.TotalRecords), tc.targetBytes)
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}

	unsized := backend.NewBlockMeta("t", uuid.New(), "vParquet5")
	unsized.TotalRecords = 4
	_, err := shardsForBlock(unsized, 4, DefaultTargetBytesPerRequest)
	require.ErrorContains(t, err, "cannot size a shard")
}
