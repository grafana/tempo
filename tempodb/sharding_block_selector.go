package tempodb

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/grafana/tempo/pkg/blockboundary"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"golang.org/x/exp/slices"
)

type shardingBlockSelector struct {
	MinInputBlocks       int
	MaxInputBlocks       int
	MaxCompactionRange   time.Duration // Size of the time window - say 6 hours
	MaxCompactionObjects int           // maximum size of compacted objects
	MaxBlockBytes        uint64        // maximum block size, estimate

	entries    []shardingBlockEntry
	boundaries [][]byte
}

type shardingBlockEntry struct {
	meta  *backend.BlockMeta
	group string // Blocks in the same group will be compacted together. Sort order also determines group priority.
	order string // Individual block priority within the group.
	hash  string // hash string used for sharding ownership, preserves backwards compatibility
	split bool
}

type shardedCompaction struct {
	blocks               []*backend.BlockMeta
	hash                 string
	maxCompactionObjects int
	split                bool
	shard                int
	shardOf              func(common.ID) int
}

func (c *shardedCompaction) Blocks() []*backend.BlockMeta {
	return c.blocks
}

func (c *shardedCompaction) Ownership() string {
	return c.hash
}

func (c *shardedCompaction) CutBlock(currBlock *backend.BlockMeta, nextTraceID common.ID) bool {
	if c.maxCompactionObjects > 0 && currBlock.TotalObjects >= c.maxCompactionObjects {
		return true
	}

	if c.split {
		// TODO this is too slow, need to switch to bytes.Compare maybe
		if shard := c.shardOf(nextTraceID); shard > c.shard {
			c.shard = shard
			return true
		}
	}

	return false
}

var (
	_ (common.Compaction)       = (*shardedCompaction)(nil)
	_ (CompactionBlockSelector) = (*shardingBlockSelector)(nil)
)

func newShardingBlockSelector(shards int, blocklist []*backend.BlockMeta, maxCompactionRange time.Duration, maxCompactionObjects int, maxBlockBytes uint64, minInputBlocks, maxInputBlocks int) CompactionBlockSelector {
	twbs := &shardingBlockSelector{
		MinInputBlocks:       minInputBlocks,
		MaxInputBlocks:       maxInputBlocks,
		MaxCompactionRange:   maxCompactionRange,
		MaxCompactionObjects: maxCompactionObjects,
		MaxBlockBytes:        maxBlockBytes,
		boundaries:           blockboundary.CreateBlockBoundaries(shards)[1:], // Remove the starting 00... boundary
	}

	var (
		now          = time.Now()
		activeWindow = twbs.windowForTime(now.Add(-activeWindowDuration))
	)

	for _, b := range blocklist {
		w := twbs.windowForBlock(b)

		// exclude blocks that fall in last window from active -> inactive cut-over
		// blocks in this window will not be compacted in order to avoid
		// ownership conflicts where two compactors process the same block
		// at the same time as it transitions from last active window to first inactive window.
		if w == activeWindow {
			continue
		}

		// These are all of the numeric values that we can group and order by,
		var (
			shard    = twbs.shardOf(b.MinID)
			shardMin = fmt.Sprintf("%03d", shard)
			shardMax = fmt.Sprintf("%03d", twbs.shardOf(b.MaxID))
			columns  = fmt.Sprintf("%016X", b.DedicatedColumnsHash())
			level    = fmt.Sprintf("%03d", b.CompactionLevel)
			window   = fmt.Sprintf("%016X", w)
			min      = fmt.Sprintf("%X", b.MinID)
		)

		if w < activeWindow {
			// Outside active window. We no longer care about compaction level
			level = ""
		}

		entry := shardingBlockEntry{
			meta: b,

			// Within group order by lowest trace ID to try to co-locate traces even further
			// This is the same for all cases.
			order: min,
		}
		switch {
		case b.CompactionLevel > 0 && shardMin == shardMax:
			// This block is within a single shard.
			entry.group = join(
				"A",            // Highest priority is recombining sharded blocks
				shardMin,       // same shard
				level,          // same level
				window,         // same window
				b.Version,      //
				b.DataEncoding, // same block format and column config
				columns,        //
			)
			entry.hash = join(
				b.TenantID, // Work sharding by tenant
				window,     // window
				shardMin,   // and shard
			)

		case b.CompactionLevel == 0:
			// Unsharded new block that needs to be split
			entry.split = true
			entry.group = join(
				"B",            // Second priority is splitting blocks
				level,          // same level
				window,         // same window
				b.Version,      //
				b.DataEncoding, // same block format and column config
				columns,        //
			)
			entry.hash = join(
				b.TenantID, // Work sharding by tenant
				window,     // window
			)

		default:
			// Existing block without sharding, or sharded under different settings.
			// Let's go ahead and resplit/combine exsting blocks, but with the lowest priority
			entry.split = true
			entry.group = join(
				"C",
				level,          // same level
				window,         // same window
				b.Version,      //
				b.DataEncoding, // same block format and column config
				columns,        //
			)
			entry.hash = join(
				b.TenantID, // Work sharding by tenant
				window,     // window
			)
		}

		twbs.entries = append(twbs.entries, entry)
	}

	// sort by group then order
	sort.SliceStable(twbs.entries, func(i, j int) bool {
		ei := twbs.entries[i]
		ej := twbs.entries[j]

		if ei.group == ej.group {
			return ei.order < ej.order
		}
		return ei.group < ej.group
	})

	return twbs
}

func (twbs *shardingBlockSelector) shardOf(id common.ID) int {
	i, _ := slices.BinarySearchFunc(twbs.boundaries, []byte(id), bytes.Compare)
	return i
}

func (twbs *shardingBlockSelector) BlocksToCompact() common.Compaction {
	for len(twbs.entries) > 0 {
		var chosen []shardingBlockEntry

		// find everything from cursor forward that belongs to this group
		// Gather contiguous blocks while staying within limits
		i := 0
		for ; i < len(twbs.entries); i++ {
			for j := i + 1; j < len(twbs.entries); j++ {
				stripe := twbs.entries[i : j+1]
				if twbs.entries[i].group == twbs.entries[j].group &&
					len(stripe) <= twbs.MaxInputBlocks &&
					totalObjects2(stripe) <= twbs.MaxCompactionObjects &&
					totalSize2(stripe) <= twbs.MaxBlockBytes {
					chosen = stripe
				} else {
					break
				}
			}
			if len(chosen) > 0 {
				// Found a stripe of blocks
				break
			}
		}

		// Remove entries that were checked so they are not considered again.
		twbs.entries = twbs.entries[i+len(chosen):]

		// did we find enough blocks?
		if len(chosen) >= twbs.MinInputBlocks {
			res := shardedCompaction{
				hash:                 chosen[0].hash,
				maxCompactionObjects: twbs.MaxCompactionObjects,
				split:                chosen[0].split,
				shardOf:              twbs.shardOf,
			}
			for _, e := range chosen {
				res.blocks = append(res.blocks, e.meta)
			}

			return &res
		}
	}
	return &shardedCompaction{}
}

func totalObjects2(entries []shardingBlockEntry) int {
	totalObjects := 0
	for _, b := range entries {
		totalObjects += b.meta.TotalObjects
	}
	return totalObjects
}

func totalSize2(entries []shardingBlockEntry) uint64 {
	sz := uint64(0)
	for _, b := range entries {
		sz += b.meta.Size
	}
	return sz
}

func join(ss ...string) string {
	return strings.Join(ss, "-")
}

func (twbs *shardingBlockSelector) windowForBlock(meta *backend.BlockMeta) int64 {
	return twbs.windowForTime(meta.EndTime)
}

func (twbs *shardingBlockSelector) windowForTime(t time.Time) int64 {
	return t.Unix() / int64(twbs.MaxCompactionRange/time.Second)
}
