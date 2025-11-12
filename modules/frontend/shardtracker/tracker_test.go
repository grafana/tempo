package shardtracker

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCompletionTracker(t *testing.T) {
	const addShards = -1

	tcs := []struct {
		name   string
		add    []int // -1 means send shards
		shards []Shard
		exp    []uint32 // expected completedThroughSeconds after each operation
	}{
		// shards only
		{
			name: "shards only",
			add:  []int{addShards},
			shards: []Shard{
				{
					TotalJobs:               1,
					CompletedThroughSeconds: 100,
				},
			},
			exp: []uint32{0},
		},
		// indexes only
		{
			name: "indexes only",
			add:  []int{1, 0, 1, 3, 2, 0, 1, 1},
			shards: []Shard{
				{
					TotalJobs:               1,
					CompletedThroughSeconds: 100,
				},
			},
			exp: []uint32{0, 0, 0, 0, 0, 0, 0, 0},
		},
		// first shard complete, shards first
		{
			name: "first shard complete, shards first",
			add:  []int{addShards, 0},
			shards: []Shard{
				{
					TotalJobs:               1,
					CompletedThroughSeconds: 100,
				},
			},
			exp: []uint32{0, 1}, // 1 = TimestampAlways when all shards complete
		},
		// first shard complete, index first
		{
			name: "first shard complete, index first",
			add:  []int{0, addShards},
			shards: []Shard{
				{
					TotalJobs:               1,
					CompletedThroughSeconds: 100,
				},
			},
			exp: []uint32{0, 1}, // 1 = TimestampAlways when all shards complete
		},
		// shards received at various times
		{
			name: "shards received at various times",
			add:  []int{addShards, 0, 0, 1, 1},
			shards: []Shard{
				{
					TotalJobs:               2,
					CompletedThroughSeconds: 100,
				},
				{
					TotalJobs:               2,
					CompletedThroughSeconds: 200,
				},
			},
			exp: []uint32{0, 0, 100, 100, 1}, // 1 = TimestampAlways when all shards complete
		},
		{
			name: "shards received at various times",
			add:  []int{0, addShards, 0, 1, 1},
			shards: []Shard{
				{
					TotalJobs:               2,
					CompletedThroughSeconds: 100,
				},
				{
					TotalJobs:               2,
					CompletedThroughSeconds: 200,
				},
			},
			exp: []uint32{0, 0, 100, 100, 1}, // 1 = TimestampAlways when all shards complete
		},
		{
			name: "shards received at various times",
			add:  []int{0, 0, 1, addShards, 1},
			shards: []Shard{
				{
					TotalJobs:               2,
					CompletedThroughSeconds: 100,
				},
				{
					TotalJobs:               2,
					CompletedThroughSeconds: 200,
				},
			},
			exp: []uint32{0, 0, 0, 100, 1}, // 1 = TimestampAlways when all shards complete
		},
		{
			name: "shards received at various times",
			add:  []int{0, 0, 1, 1, addShards},
			shards: []Shard{
				{
					TotalJobs:               2,
					CompletedThroughSeconds: 100,
				},
				{
					TotalJobs:               2,
					CompletedThroughSeconds: 200,
				},
			},
			exp: []uint32{0, 0, 0, 0, 1}, // 1 = TimestampAlways when all shards complete
		},
		// bad data received
		{
			name: "bad data received last",
			add:  []int{addShards, 0, 0, 2},
			shards: []Shard{
				{
					TotalJobs:               2,
					CompletedThroughSeconds: 100,
				},
				{
					TotalJobs:               2,
					CompletedThroughSeconds: 200,
				},
			},
			exp: []uint32{0, 0, 100, 100},
		},
		{
			name: "bad data immediately after shards",
			add:  []int{addShards, 2, 0, 0},
			shards: []Shard{
				{
					TotalJobs:               2,
					CompletedThroughSeconds: 100,
				},
				{
					TotalJobs:               2,
					CompletedThroughSeconds: 200,
				},
			},
			exp: []uint32{0, 0, 0, 100},
		},
		{
			name: "bad data immediately before shards",
			add:  []int{0, 0, 2, addShards},
			shards: []Shard{
				{
					TotalJobs:               2,
					CompletedThroughSeconds: 100,
				},
				{
					TotalJobs:               2,
					CompletedThroughSeconds: 200,
				},
			},
			exp: []uint32{0, 0, 0, 100},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			tracker := &CompletionTracker{}

			require.Len(t, tc.exp, len(tc.add), "expected values must match number of operations")

			for i, sc := range tc.add {
				var ct uint32
				if sc == -1 {
					ct = tracker.AddShards(tc.shards)
				} else {
					ct = tracker.AddShardIdx(sc)
				}
				require.Equal(t, int(tc.exp[i]), int(ct), "operation %d", i)
			}
		})
	}
}
