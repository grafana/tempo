package cache

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

// fakeScanner drives scanAndDelete through scripted SCAN batches so we can
// reproduce the real Redis semantics that miniredis does not: multiple
// iterations, empty batches paired with a non-zero cursor, and termination only
// when the cursor returns to 0.
type fakeScanner struct {
	batches [][]string
	call    int
	deleted []string
	delSize []int // number of keys per Del call, in order
	scanErr error
	delErr  error
}

func (f *fakeScanner) Scan(_ context.Context, _ uint64, _ string, _ int64) *redis.ScanCmd {
	if f.scanErr != nil {
		return redis.NewScanCmdResult(nil, 0, f.scanErr)
	}

	var keys []string
	if f.call < len(f.batches) {
		keys = f.batches[f.call]
	}
	f.call++

	// The cursor stays non-zero while scripted batches remain, so the loop must
	// keep going even across empty batches; only the final batch reports 0.
	var cursor uint64
	if f.call < len(f.batches) {
		cursor = uint64(f.call)
	}
	return redis.NewScanCmdResult(keys, cursor, nil)
}

func (f *fakeScanner) Del(_ context.Context, keys ...string) *redis.IntCmd {
	if f.delErr != nil {
		return redis.NewIntResult(0, f.delErr)
	}
	f.deleted = append(f.deleted, keys...)
	f.delSize = append(f.delSize, len(keys))
	return redis.NewIntResult(int64(len(keys)), nil)
}

func TestScanAndDelete_DrivesCursorThroughEmptyBatches(t *testing.T) {
	f := &fakeScanner{batches: [][]string{
		{"a", "b"},
		{}, // empty batch with a non-zero cursor: iteration must continue
		{"c"},
		{},    // and again
		{"d"}, // final batch reports cursor 0
	}}

	deleted, err := scanAndDelete(context.Background(), f, "p*")
	require.NoError(t, err)
	require.Equal(t, 4, deleted)
	require.Equal(t, []string{"a", "b", "c", "d"}, f.deleted)
	require.Equal(t, len(f.batches), f.call, "must SCAN through every batch, including the empty ones")
}

func TestScanAndDelete_EmptyKeyspace(t *testing.T) {
	// A single empty batch with cursor 0 (nothing matched the prefix).
	f := &fakeScanner{batches: [][]string{{}}}

	deleted, err := scanAndDelete(context.Background(), f, "p*")
	require.NoError(t, err)
	require.Equal(t, 0, deleted)
	require.Empty(t, f.deleted)
	require.Empty(t, f.delSize, "no DEL should be issued when nothing matched")
}

func TestScanAndDelete_CapsDeleteBatchSize(t *testing.T) {
	big := make([]string, redisDeleteBatch+redisDeleteBatch/2) // 1.5x the batch cap
	for i := range big {
		big[i] = fmt.Sprintf("k%d", i)
	}
	f := &fakeScanner{batches: [][]string{big, {"tail"}}}

	deleted, err := scanAndDelete(context.Background(), f, "p*")
	require.NoError(t, err)
	require.Equal(t, len(big)+1, deleted)

	want := append(append([]string{}, big...), "tail")
	require.Equal(t, want, f.deleted)
	for _, n := range f.delSize {
		require.LessOrEqual(t, n, redisDeleteBatch, "no DEL command may exceed the batch cap")
	}
}

func TestScanAndDelete_PropagatesScanError(t *testing.T) {
	sentinel := errors.New("scan boom")
	f := &fakeScanner{scanErr: sentinel}

	deleted, err := scanAndDelete(context.Background(), f, "p*")
	require.ErrorIs(t, err, sentinel)
	require.Zero(t, deleted)
}

func TestScanAndDelete_PropagatesDeleteError(t *testing.T) {
	sentinel := errors.New("del boom")
	f := &fakeScanner{batches: [][]string{{"a"}}, delErr: sentinel}

	_, err := scanAndDelete(context.Background(), f, "p*")
	require.ErrorIs(t, err, sentinel)
}

func TestScanAndDelete_RespectsContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	f := &fakeScanner{batches: [][]string{{"a"}}}
	deleted, err := scanAndDelete(ctx, f, "p*")
	require.ErrorIs(t, err, context.Canceled)
	require.Zero(t, deleted)
	require.Zero(t, f.call, "a cancelled context must stop before the first SCAN")
}

func TestGlobPrefixReplacer(t *testing.T) {
	cases := map[string]string{
		"tenant:block:": "tenant:block:",
		`a*b:block:`:    `a\*b:block:`,
		`a?b:block:`:    `a\?b:block:`,
		`a[b]:block:`:   `a\[b\]:block:`,
		`a\b:block:`:    `a\\b:block:`,
		`*?[]\`:         `\*\?\[\]\\`,
	}
	for in, want := range cases {
		require.Equal(t, want, globPrefixReplacer.Replace(in), "input %q", in)
	}
}

// TestRedisClient_RemoveByPrefix exercises the real SCAN MATCH + DEL path against
// miniredis (single node): only keys under the requested prefix are removed, and
// offset-keyed range entries are covered like whole-object ones.
func TestRedisClient_RemoveByPrefix(t *testing.T) {
	client, err := mockRedisClientSingle()
	require.NoError(t, err)
	defer func() { _ = client.Close() }()

	ctx := context.Background()

	target := []string{
		"tenantA:block1:bloom-0",
		"tenantA:block1:index",
		"tenantA:block1:data.parquet:0:100",
		"tenantA:block1:data.parquet:100:250",
	}
	survivors := []string{
		"tenantA:block2:data.parquet:0:100", // different block, same tenant
		"tenantB:block1:data.parquet:0:100", // different tenant, same block id
	}

	all := append(append([]string{}, target...), survivors...)
	vals := make([][]byte, len(all))
	for i := range all {
		vals[i] = []byte("v")
	}
	require.NoError(t, client.MSet(ctx, all, vals))

	deleted, err := client.RemoveByPrefix(ctx, "tenantA:block1:")
	require.NoError(t, err)
	require.Equal(t, len(target), deleted)

	got, err := client.MGet(ctx, target)
	require.NoError(t, err)
	for i, key := range target {
		require.Nil(t, got[i], "key %q must be evicted", key)
	}

	got, err = client.MGet(ctx, survivors)
	require.NoError(t, err)
	for i, key := range survivors {
		require.NotNil(t, got[i], "unrelated key %q must survive", key)
	}
}
