package benchmark

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/v3/pkg/benchmark/metrics"
	"github.com/grafana/tempo/v3/tempodb/backend"
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
	require.Positive(t, result.DurationNs)
	// A sharded case runs one execution per shard per pass, so the shard count
	// is what makes an execution count legible next to Repeat.
	require.Positive(t, result.Shards)
	require.Equal(t, result.Shards, byAPI(t, result, apiSearch).Executions)
	// 2 trace-by-ID + 1 search + 2 metrics + 6 tag name scopes.
	require.Len(t, result.Cases, 11)

	byID := map[string]CaseResult{}
	for _, c := range result.Cases {
		require.Empty(t, c.Error, "case %s", c.ID)
		require.Positive(t, c.Executions, "case %s", c.ID)

		// Every measurement has the same shape, whatever it came from, so one
		// loop checks them all.
		for key, m := range c.Metrics {
			require.Contains(t, []metrics.Kind{metrics.Counter, metrics.Gauge}, m.Kind, "%s/%s", c.ID, key)
			require.Positive(t, m.Summary.Count, "%s/%s has no summary", c.ID, key)
			require.LessOrEqual(t, m.Summary.Min, m.Summary.P25, "%s/%s", c.ID, key)
			require.LessOrEqual(t, m.Summary.P25, m.Summary.P50, "%s/%s", c.ID, key)
			require.LessOrEqual(t, m.Summary.P50, m.Summary.P75, "%s/%s", c.ID, key)
			require.LessOrEqual(t, m.Summary.P75, m.Summary.P99, "%s/%s", c.ID, key)
			require.LessOrEqual(t, m.Summary.P99, m.Summary.Max, "%s/%s", c.ID, key)
		}

		wall := c.Metrics[metrics.KeyWallNs]
		require.Positive(t, wall.Summary.Max, "case %s", c.ID)
		require.Equal(t, c.Executions, wall.Summary.Count, "case %s", c.ID)

		byID[c.ID] = c
	}

	// Every present ID must be found, and no absent one.
	present := byID["traceid/present"]
	require.Equal(t, 25, present.Executions)
	require.Equal(t, int64(25), present.Matched)

	absent := byID["traceid/absent"]
	require.Equal(t, 25, absent.Executions)
	require.Zero(t, absent.Matched, "an absent ID was found, so the profile is wrong")
	// A bloom miss returns no response, so nothing is reported rather than a
	// row of zeroes. The backend counter is what covers this case.
	require.NotContains(t, absent.Metrics, metrics.PrefixResponse+"inspectedBytes")
	require.Positive(t, absent.Metrics[metrics.KeyBackendReads].Total)

	search := byID["search/nopredicate"]
	require.Positive(t, search.Matched)
	// Keyed by Tempo's own name under its source prefix.
	require.Positive(t, search.Metrics[metrics.PrefixResponse+"inspectedBytes"].Total)

	// The block is read through a counting reader, so I/O must be observed,
	// and per execution rather than only as a case total.
	require.Positive(t, present.Metrics[metrics.KeyBackendReads].Total)
	require.Positive(t, present.Metrics[metrics.KeyBackendBytes].Total)
	require.Positive(t, present.Metrics[metrics.KeyBackendTimeNs].Summary.Count)

	// CPU and allocations are measured per execution too.
	require.Positive(t, present.Metrics[metrics.KeyAllocBytes].Total)
	require.Positive(t, present.Metrics[metrics.KeyCPUNs].Summary.Count)

	// Whatever Tempo emitted to the default registry is picked up without the
	// runner naming any metric, and with a distribution like everything else.
	var processKeys int
	for key, m := range search.Metrics {
		if strings.HasPrefix(key, metrics.PrefixProcess) {
			processKeys++
			require.Equal(t, search.Executions, m.Summary.Count, key)
		}
	}
	require.Positive(t, processKeys)

	// The metrics path reports through a different type, so check it lands
	// under the same keys as search rather than the evaluator's field names.
	rate := byID["metrics/rate"]
	require.Positive(t, rate.Matched, "a metrics query returned no series")
	require.Positive(t, rate.Metrics[metrics.PrefixResponse+"inspectedBytes"].Total)
	require.Positive(t, rate.Metrics[metrics.PrefixResponse+"inspectedSpans"].Total)

	// Tag names run once, not per shard: SearchTags walks the whole file, so a
	// shard per row group would multiply the match count by the shard count.
	for id, c := range byID {
		if c.API == apiMetadata {
			require.Equal(t, 1, c.Executions, "%s must not shard", id)
		}
	}

	// Tag names report bytes through a callback, keyed the way responses are.
	unscoped := byID["metadata/tagnames/none"]
	require.Positive(t, unscoped.Matched, "no tag names were found")
	require.Positive(t, unscoped.Metrics[metrics.PrefixResponse+"inspectedBytes"].Total)

	// A scope with no attributes still costs a read, which is worth measuring.
	require.Positive(t, byID["metadata/tagnames/instrumentation"].Metrics[metrics.PrefixResponse+"inspectedBytes"].Total)
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
	require.Equal(t, want.Options, got.Options)
	require.Len(t, got.Cases, len(want.Cases))
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
		{"last shard overruns when it does not divide, as production's does", meta(500, 5), 200, []Shard{{0, 0, 2}, {1, 2, 2}, {2, 4, 2}}},
		{"row groups larger than the target get one each", meta(1000, 2), 100, []Shard{{0, 0, 1}, {1, 1, 1}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := shardsForBlock(tc.meta, tc.targetBytes)
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}

	unsized := backend.NewBlockMeta("t", uuid.New(), "vParquet5")
	unsized.TotalRecords = 4
	_, err := shardsForBlock(unsized, DefaultTargetBytesPerRequest)
	require.ErrorContains(t, err, "cannot size a shard")
}

// byAPI returns the one case for an API, failing if there is not exactly one.
func byAPI(t *testing.T, result *Result, api string) CaseResult {
	t.Helper()
	var found []CaseResult
	for _, c := range result.Cases {
		if c.API == api {
			found = append(found, c)
		}
	}
	require.Len(t, found, 1, "expected one %s case", api)
	return found[0]
}
