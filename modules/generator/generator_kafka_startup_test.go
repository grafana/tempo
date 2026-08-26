package generator

import (
	"context"
	"flag"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/kgo"

	"github.com/grafana/tempo/pkg/ingest"
	"github.com/grafana/tempo/pkg/ingest/testkafka"
)

// TestGenerator_adjustStartupOffsets_skipsStaleBacklog verifies the
// AdjustFetchOffsetsFn hook seeks a partition forward past backlog older than
// the startup replay horizon to the first in-horizon record, when the committed
// offset is behind that horizon.
func TestGenerator_adjustStartupOffsets_skipsStaleBacklog(t *testing.T) {
	ctx := context.Background()
	const topic = "test-topic"
	_, addr := testkafka.CreateCluster(t, 1, topic)

	client, err := kgo.NewClient(
		kgo.SeedBrokers(addr),
		kgo.DefaultProduceTopic(topic),
		kgo.RecordPartitioner(kgo.ManualPartitioner()),
	)
	require.NoError(t, err)
	t.Cleanup(client.Close)

	now := time.Now()
	produce := func(ts time.Time) {
		rec := &kgo.Record{Topic: topic, Partition: 0, Timestamp: ts, Value: []byte("x")}
		require.NoError(t, client.ProduceSync(ctx, rec).FirstErr())
	}
	produce(now.Add(-time.Hour))        // offset 0, stale
	produce(now.Add(-30 * time.Minute)) // offset 1, stale
	produce(now)                        // offset 2, within horizon

	g := minimalGeneratorForKafkaTest()
	g.cfg.SkipStaleBacklogOnStartup = true
	g.cfg.MetricsIngestionSlack = 5 * time.Minute
	g.partitionClient = ingest.NewPartitionOffsetClient(client, topic)

	// Committed at offset 0 (behind the horizon) -> should seek forward to 2.
	offsets := map[string]map[int32]kgo.Offset{topic: {0: kgo.NewOffset().At(0)}}
	got, err := g.adjustStartupOffsets(ctx, offsets)
	require.NoError(t, err)
	assert.Equal(t, int64(2), got[topic][0].EpochOffset().Offset)
}

// TestGenerator_adjustStartupOffsets_failOpenOnFetchError verifies the hook
// fails open: if the offset lookup errors (here, a canceled context) it returns
// the original committed offsets unchanged rather than surfacing an error and
// blocking startup. Skipping stale backlog is a best-effort optimization.
func TestGenerator_adjustStartupOffsets_failOpenOnFetchError(t *testing.T) {
	const topic = "test-topic"
	_, addr := testkafka.CreateCluster(t, 1, topic)
	client, err := kgo.NewClient(kgo.SeedBrokers(addr))
	require.NoError(t, err)
	t.Cleanup(client.Close)

	g := minimalGeneratorForKafkaTest()
	g.cfg.SkipStaleBacklogOnStartup = true
	g.cfg.MetricsIngestionSlack = 5 * time.Minute
	g.partitionClient = ingest.NewPartitionOffsetClient(client, topic)

	canceled, cancel := context.WithCancel(context.Background())
	cancel() // force the horizon lookup to fail

	in := map[string]map[int32]kgo.Offset{topic: {0: kgo.NewOffset().At(7)}}
	got, err := g.adjustStartupOffsets(canceled, in)
	require.NoError(t, err)
	assert.Equal(t, int64(7), got[topic][0].EpochOffset().Offset, "committed offset should be preserved on fetch error")
}

// TestStartupSeekOffsets verifies the per-partition decision is aggregated into
// a seek map: only partitions whose committed offset is behind the horizon (or
// absent) are included, and partitions without a known horizon offset are left
// untouched.
func TestStartupSeekOffsets(t *testing.T) {
	partitions := []int32{0, 1, 2, 3, 4}
	committedOffsets := map[int32]int64{
		0: 100,             // ahead of horizon -> no seek
		1: 50,              // behind horizon -> seek forward
		2: kafkaOffsetNone, // no commit -> seek to horizon
		// 3: absent from committed map -> treated as no commit -> seek
		4: 10, // behind horizon, but no horizon offset known -> skip
	}
	horizonOffsets := map[int32]int64{
		0: 80,
		1: 120,
		2: 30,
		3: 200,
		// 4: absent -> cannot seek
	}

	got := startupSeekOffsets(partitions, committedOffsets, horizonOffsets)
	assert.Equal(t, map[int32]int64{1: 120, 2: 30, 3: 200}, got)
}

// TestConfig_skipStaleBacklogOnStartupDefault verifies the feature is off by default.
func TestConfig_skipStaleBacklogOnStartupDefault(t *testing.T) {
	cfg := &Config{}
	f := flag.NewFlagSet("test", flag.PanicOnError)
	cfg.RegisterFlagsAndApplyDefaults("", f)

	assert.False(t, cfg.SkipStaleBacklogOnStartup, "skip should default off")
}

// TestStartupSeekOffset verifies the offset the generator resumes from when
// skip_stale_backlog_on_startup is enabled: it never rewinds behind the group's
// committed offset, but it skips forward past stale backlog to the horizon
// offset (the first record at/after now-horizon) when the commit is behind that
// horizon or absent.
func TestStartupSeekOffset(t *testing.T) {
	for _, tc := range []struct {
		name          string
		committed     int64
		horizonOffset int64
		wantOffset    int64
		wantSeek      bool
	}{
		{"committed ahead of horizon keeps position, no seek", 100, 50, 100, false},
		{"committed behind horizon skips forward", 50, 100, 100, true},
		{"no committed offset seeks to horizon", kafkaOffsetNone, 100, 100, true},
		{"committed equal to horizon does not seek", 100, 100, 100, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			offset, seek := startupSeekOffset(tc.committed, tc.horizonOffset)
			assert.Equal(t, tc.wantOffset, offset, "offset")
			assert.Equal(t, tc.wantSeek, seek, "seek")
		})
	}
}
