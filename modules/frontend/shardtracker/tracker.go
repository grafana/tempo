package shardtracker

import "math"

const (
	TimestampNever   = uint32(math.MaxUint32)
	TimestampAlways  = uint32(1)
	TimestampUnknown = uint32(0)
)

// Shard represents a single shard with its job count and completion timestamp.
// CompletedThroughSeconds indicates the time boundary (in Unix seconds) up to which
// all results in this shard are guaranteed to be complete.
type Shard struct {
	TotalJobs               uint32
	CompletedThroughSeconds uint32
}

// JobMetadata contains shard information for tracking job progress and completion.
// This is typically embedded in a pipeline-specific metadata response struct.
type JobMetadata struct {
	TotalBlocks int
	TotalJobs   int
	TotalBytes  uint64
	Shards      []Shard
}

// CompletionTracker tracks which shards have been completed so that results
// can be released progressively as shards finish. This allows streaming results
// to users as soon as they're guaranteed to be complete, rather than waiting
// for all jobs to finish.
type CompletionTracker struct {
	shards         []Shard
	foundResponses []int

	completedThroughSeconds uint32
	curShard                int
}

// AddShards initializes or updates the shard information in the tracker.
// This should be called when job metadata arrives from the sharder.
// Returns the current completedThroughSeconds value.
func (c *CompletionTracker) AddShards(shards []Shard) uint32 {
	if len(shards) == 0 {
		return c.completedThroughSeconds
	}

	c.shards = shards

	// grow foundResponses to match while keeping the existing values
	if len(c.shards) > len(c.foundResponses) {
		temp := make([]int, len(c.shards))
		copy(temp, c.foundResponses)
		c.foundResponses = temp
	}

	c.incrementCurShardIfComplete()

	return c.completedThroughSeconds
}

// AddShardIdx records that a response has been received for the given shard index.
// This should be called each time a job completes for a particular shard.
// Returns the current completedThroughSeconds value.
func (c *CompletionTracker) AddShardIdx(shardIdx int) uint32 {
	// we haven't received shards yet
	if len(c.shards) == 0 {
		// if shardIdx doesn't fit in foundResponses then alloc a new slice and copy foundResponses forward
		if shardIdx >= len(c.foundResponses) {
			temp := make([]int, shardIdx+1)
			copy(temp, c.foundResponses)
			c.foundResponses = temp
		}

		// and record this idx for when we get shards
		c.foundResponses[shardIdx]++

		return 0
	}

	//
	if shardIdx >= len(c.foundResponses) {
		return c.completedThroughSeconds
	}

	c.foundResponses[shardIdx]++
	c.incrementCurShardIfComplete()

	return c.completedThroughSeconds
}

// CompletedThroughSeconds returns the current completion timestamp.
// All results up to this timestamp are guaranteed to be complete and can be released.
func (c *CompletionTracker) CompletedThroughSeconds() uint32 {
	return c.completedThroughSeconds
}

// incrementCurShardIfComplete tests to see if the current shard is complete and increments it if so.
// it does this repeatedly until it finds a shard that is not complete.
func (c *CompletionTracker) incrementCurShardIfComplete() {
	for {
		if c.curShard >= len(c.shards) {
			c.completedThroughSeconds = 1
			break
		}

		if c.foundResponses[c.curShard] == int(c.shards[c.curShard].TotalJobs) {
			c.completedThroughSeconds = c.shards[c.curShard].CompletedThroughSeconds
			c.curShard++
		} else {
			break
		}
	}
}
