package cache

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
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

	deleted, err := scanAndDelete(context.Background(), f, f, "p*")
	require.NoError(t, err)
	require.Equal(t, 4, deleted)
	require.Equal(t, []string{"a", "b", "c", "d"}, f.deleted)
	require.Equal(t, len(f.batches), f.call, "must SCAN through every batch, including the empty ones")
}

func TestScanAndDelete_EmptyKeyspace(t *testing.T) {
	// A single empty batch with cursor 0 (nothing matched the prefix).
	f := &fakeScanner{batches: [][]string{{}}}

	deleted, err := scanAndDelete(context.Background(), f, f, "p*")
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

	deleted, err := scanAndDelete(context.Background(), f, f, "p*")
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

	deleted, err := scanAndDelete(context.Background(), f, f, "p*")
	require.ErrorIs(t, err, sentinel)
	require.Zero(t, deleted)
}

func TestScanAndDelete_PropagatesDeleteError(t *testing.T) {
	sentinel := errors.New("del boom")
	f := &fakeScanner{batches: [][]string{{"a"}}, delErr: sentinel}

	_, err := scanAndDelete(context.Background(), f, f, "p*")
	require.ErrorIs(t, err, sentinel)
}

func TestScanAndDelete_RespectsContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	f := &fakeScanner{batches: [][]string{{"a"}}}
	deleted, err := scanAndDelete(ctx, f, f, "p*")
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

// testSlot stands in for Redis's CRC16-based slot assignment. The exact value
// does not matter — only that distinct keys can land in distinct slots, so a
// multi-key DEL can span slots the way it does on a real cluster (and the way
// miniredis, which ignores slots, never reproduces).
func testSlot(key string) int {
	h := fnv.New32a()
	_, _ = h.Write([]byte(key))
	return int(h.Sum32() % 16384)
}

// crossSlotNode models a single Redis Cluster master: a DEL whose keys span more
// than one hash slot is rejected with CROSSSLOT, exactly as a real node rejects
// it. A per-master node client (redis.ClusterClient.ForEachMaster) behaves this
// way — which is why deleting a scanned batch through it is a bug.
type crossSlotNode struct {
	store map[string]struct{}
	dels  [][]string
}

func newCrossSlotNode(keys []string) *crossSlotNode {
	n := &crossSlotNode{store: make(map[string]struct{}, len(keys))}
	for _, k := range keys {
		n.store[k] = struct{}{}
	}
	return n
}

func (n *crossSlotNode) Del(_ context.Context, keys ...string) *redis.IntCmd {
	n.dels = append(n.dels, append([]string(nil), keys...))

	slot := -1
	for _, k := range keys {
		if s := testSlot(k); slot == -1 {
			slot = s
		} else if s != slot {
			return redis.NewIntResult(0, errors.New("CROSSSLOT Keys in request don't hash to the same slot"))
		}
	}

	var deleted int64
	for _, k := range keys {
		if _, ok := n.store[k]; ok {
			delete(n.store, k)
			deleted++
		}
	}
	return redis.NewIntResult(deleted, nil)
}

// slotSplittingDeleter models redis.ClusterClient.Del: it groups keys by slot and
// issues one single-slot DEL per group, so a cross-slot batch never reaches the
// node as one command. This is the deleter the cluster path now passes.
type slotSplittingDeleter struct{ node *crossSlotNode }

func (d slotSplittingDeleter) Del(ctx context.Context, keys ...string) *redis.IntCmd {
	bySlot := map[int][]string{}
	for _, k := range keys {
		s := testSlot(k)
		bySlot[s] = append(bySlot[s], k)
	}
	var total int64
	for _, group := range bySlot {
		cnt, err := d.node.Del(ctx, group...).Result()
		total += cnt
		if err != nil {
			return redis.NewIntResult(total, err)
		}
	}
	return redis.NewIntResult(total, nil)
}

// TestScanAndDelete_ClusterDeletionMustSplitBySlot pins the reason the cluster
// path deletes through the cluster client rather than the per-master node client.
// A batch scanned from one master spans many slots, so a single multi-key DEL to
// the node fails with CROSSSLOT (the original bug); routing the same batch through
// a slot-splitting deleter removes every key.
func TestScanAndDelete_ClusterDeletionMustSplitBySlot(t *testing.T) {
	ctx := context.Background()

	// Keys chosen so they hash to more than one slot under testSlot.
	keys := []string{"tenant:block:footer", "tenant:block:col", "tenant:block:page:0:100", "tenant:block:page:100:250"}
	slots := map[int]struct{}{}
	for _, k := range keys {
		slots[testSlot(k)] = struct{}{}
	}
	require.Greater(t, len(slots), 1, "test keys must span >1 slot to exercise CROSSSLOT")

	// Deleting a cross-slot batch straight to the node fails — this is what the
	// old code did by handing ForEachMaster's node client to the delete loop.
	node := newCrossSlotNode(keys)
	_, err := scanAndDelete(ctx, &fakeScanner{batches: [][]string{keys}}, node, "tenant:block:*")
	require.ErrorContains(t, err, "CROSSSLOT")

	// Deleting through the slot-splitting deleter (what redis.ClusterClient does)
	// removes every key and never sends a cross-slot DEL to the node.
	node = newCrossSlotNode(keys)
	deleted, err := scanAndDelete(ctx, &fakeScanner{batches: [][]string{keys}}, slotSplittingDeleter{node: node}, "tenant:block:*")
	require.NoError(t, err)
	require.Equal(t, len(keys), deleted)
	require.Empty(t, node.store, "all keys must be evicted")
	for _, del := range node.dels {
		gotSlots := map[int]struct{}{}
		for _, k := range del {
			gotSlots[testSlot(k)] = struct{}{}
		}
		require.LessOrEqual(t, len(gotSlots), 1, "no DEL reaching a node may span slots")
	}
}

// TestRedisClient_RemoveByPrefix_Cluster exercises prefix eviction against a real
// Redis Cluster, the default topology. A block's keys spread across every
// master/slot must all be removed without the CROSSSLOT error a per-node multi-key
// DEL would hit — which miniredis cannot model, so this is skipped unless
// TEMPO_REDIS_CLUSTER_ADDRS points at a running cluster. Bring one up with:
//
//	docker run --rm -d --name c -p 7000-7005:7000-7005 -e IP=0.0.0.0 grokzen/redis-cluster:7.2.4
//	TEMPO_REDIS_CLUSTER_ADDRS=127.0.0.1:7000,127.0.0.1:7001,127.0.0.1:7002 \
//	  go test ./pkg/cache -run TestRedisClient_RemoveByPrefix_Cluster -v
func TestRedisClient_RemoveByPrefix_Cluster(t *testing.T) {
	addrs := os.Getenv("TEMPO_REDIS_CLUSTER_ADDRS")
	if addrs == "" {
		t.Skip("set TEMPO_REDIS_CLUSTER_ADDRS to a running Redis Cluster to run this test")
	}

	cfg := &RedisConfig{
		Endpoint:   addrs,
		Timeout:    5 * time.Second,
		Expiration: time.Minute,
		// SingleNode defaults to false: exercise the cluster path.
	}
	client, err := NewRedisClient(cfg, "test-cluster", prometheus.NewRegistry())
	require.NoError(t, err)
	defer func() { _ = client.Close() }()

	ctx := context.Background()

	// A fresh block ID per run keeps the test isolated on a shared cluster.
	blockID := uuid.New()
	prefix := fmt.Sprintf("tenantA:%s:", blockID)
	// Enough offset-keyed range entries that they spread across masters and slots.
	var target []string
	for i := 0; i < 200; i++ {
		target = append(target, fmt.Sprintf("%sdata.parquet:%d:%d", prefix, i*100, i*100+100))
	}
	target = append(target, prefix+"bloom-0", prefix+"index")

	survivors := []string{
		fmt.Sprintf("tenantA:%s:data.parquet:0:100", uuid.New()), // different block, same tenant
		fmt.Sprintf("tenantB:%s:data.parquet:0:100", blockID),    // different tenant, same block id
	}

	all := append(append([]string{}, target...), survivors...)
	vals := make([][]byte, len(all))
	for i := range all {
		vals[i] = []byte("v")
	}
	require.NoError(t, client.MSet(ctx, all, vals))

	deleted, err := client.RemoveByPrefix(ctx, prefix)
	require.NoError(t, err, "cross-slot prefix eviction must not fail with CROSSSLOT")
	require.Equal(t, len(target), deleted, "every entry under the prefix must be deleted exactly once")

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
