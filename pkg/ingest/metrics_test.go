package ingest

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kfake"
	"github.com/twmb/franz-go/pkg/kgo"
)

// newGetGroupLagCluster creates a kfake cluster with the given number of partitions
// for topicName (defined in partition_offset_client_test.go as "test").
// Returns the kgo client, kadm admin client, and partition offset client.
func newGetGroupLagCluster(t *testing.T, numPartitions int32) (*kgo.Client, *kadm.Client, *PartitionOffsetClient) {
	t.Helper()
	fake, err := kfake.NewCluster(kfake.NumBrokers(1), kfake.SeedTopics(numPartitions, topicName))
	require.NoError(t, err)
	t.Cleanup(fake.Close)

	client, err := kgo.NewClient(
		kgo.SeedBrokers(fake.ListenAddrs()...),
		kgo.DisableClientMetrics(),
		kgo.RecordPartitioner(kgo.ManualPartitioner()),
	)
	require.NoError(t, err)
	t.Cleanup(client.Close)

	return client, kadm.NewClient(client), NewPartitionOffsetClient(client, topicName)
}

// commitOffset commits a single offset for topicName/partition to the given group via
// the kadm admin API (standalone commit, no group membership required).
func commitOffset(ctx context.Context, t *testing.T, adm *kadm.Client, group string, partition int32, offset int64) {
	t.Helper()
	var os kadm.Offsets
	os.Add(kadm.Offset{Topic: topicName, Partition: partition, At: offset})
	_, err := adm.CommitOffsets(ctx, group, os)
	require.NoError(t, err)
}

func TestGetGroupLag(t *testing.T) {
	ctx := context.Background()

	t.Run("empty assigned partitions returns empty lag", func(t *testing.T) {
		t.Parallel()
		_, adm, partClient := newGetGroupLagCluster(t, 2)

		lag, err := getGroupLag(ctx, adm, partClient, "test-group", topicName, nil)
		require.NoError(t, err)
		assert.Empty(t, lag)
	})

	t.Run("no commit falls back to start offset", func(t *testing.T) {
		t.Parallel()
		client, adm, partClient := newGetGroupLagCluster(t, 2)

		// Produce 5 records to partition 0; end=5, start=0.
		for range 5 {
			produceRecord(ctx, t, client, 0, []byte("x"))
		}

		lag, err := getGroupLag(ctx, adm, partClient, "test-group", topicName, []int32{0})
		require.NoError(t, err)

		l, ok := lag.Lookup(topicName, 0)
		require.True(t, ok)
		assert.Equal(t, int64(5), l.Lag)
	})

	t.Run("committed offset produces positive lag", func(t *testing.T) {
		t.Parallel()
		client, adm, partClient := newGetGroupLagCluster(t, 2)

		for range 5 {
			produceRecord(ctx, t, client, 0, []byte("x"))
		}
		// Consume through offset 2 (next to read = 3); lag = 5-3 = 2.
		commitOffset(ctx, t, adm, "test-group", 0, 3)

		lag, err := getGroupLag(ctx, adm, partClient, "test-group", topicName, []int32{0})
		require.NoError(t, err)

		l, ok := lag.Lookup(topicName, 0)
		require.True(t, ok)
		assert.Equal(t, int64(2), l.Lag)
	})

	t.Run("commit ahead of end is clamped to zero", func(t *testing.T) {
		t.Parallel()
		client, adm, partClient := newGetGroupLagCluster(t, 2)

		for range 3 {
			produceRecord(ctx, t, client, 0, []byte("x"))
		}
		// Commit beyond end (e.g. after log truncation); raw lag = 3-10 = -7.
		commitOffset(ctx, t, adm, "test-group", 0, 10)

		lag, err := getGroupLag(ctx, adm, partClient, "test-group", topicName, []int32{0})
		require.NoError(t, err)

		l, ok := lag.Lookup(topicName, 0)
		require.True(t, ok)
		assert.Equal(t, int64(0), l.Lag) // clamped from -7 to 0
	})

	t.Run("multiple partitions reported independently", func(t *testing.T) {
		t.Parallel()
		client, adm, partClient := newGetGroupLagCluster(t, 2)

		// partition 0: 4 records, committed at 2 → lag 2
		for range 4 {
			produceRecord(ctx, t, client, 0, []byte("x"))
		}
		commitOffset(ctx, t, adm, "test-group", 0, 2)

		// partition 1: 3 records, no commit → lag 3 (end-start = 3-0)
		for range 3 {
			produceRecord(ctx, t, client, 1, []byte("x"))
		}

		lag, err := getGroupLag(ctx, adm, partClient, "test-group", topicName, []int32{0, 1})
		require.NoError(t, err)

		l0, ok := lag.Lookup(topicName, 0)
		require.True(t, ok)
		assert.Equal(t, int64(2), l0.Lag)

		l1, ok := lag.Lookup(topicName, 1)
		require.True(t, ok)
		assert.Equal(t, int64(3), l1.Lag)
	})
}
