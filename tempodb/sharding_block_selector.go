package tempodb

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/grafana/tempo/pkg/util/traceidboundary"
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
	meta      *backend.BlockMeta
	group     string // Blocks in the same group will be compacted together. Sort order also determines group priority.
	order     string // Individual block priority within the group.
	hash      string // hash string used for sharding ownership
	split     bool
	minBlocks int
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
		// If the shard of the next trace is higher, then it's time to cut the block.
		if shard := c.shardOf(nextTraceID); shard > c.shard {
			c.shard = shard
			return true
		}
	}

	return false
}

var (
	_ (common.CompactionRound)  = (*shardedCompaction)(nil)
	_ (CompactionBlockSelector) = (*shardingBlockSelector)(nil)
)

func newShardingBlockSelector(shards int, blocklist []*backend.BlockMeta, maxCompactionRange time.Duration, maxCompactionObjects int, maxBlockBytes uint64, minInputBlocks, maxInputBlocks int) CompactionBlockSelector {
	s := &shardingBlockSelector{
		MinInputBlocks:       minInputBlocks,
		MaxInputBlocks:       maxInputBlocks,
		MaxCompactionRange:   maxCompactionRange,
		MaxCompactionObjects: maxCompactionObjects,
		MaxBlockBytes:        maxBlockBytes,
		boundaries:           traceidboundary.All(uint32(shards)),
	}

	var (
		now        = time.Now()
		currWindow = s.windowForTime(now)
	)

	for _, b := range blocklist {

		var (
			w     = s.windowForBlock(b)
			shard = s.shardOf(b.MinID)

			// These are all of the numeric values that we can group and order by,
			shardMin = fmt.Sprintf("%03d", shard)
			shardMax = fmt.Sprintf("%03d", s.shardOf(b.MaxID))
			columns  = fmt.Sprintf("%016X", b.DedicatedColumnsHash())
			level    = fmt.Sprintf("%03d", b.CompactionLevel)
			window   = fmt.Sprintf("%v", w)
			age      = fmt.Sprintf("%016X", currWindow-w)
			min      = fmt.Sprintf("%X", b.MinID)
		)

		entry := shardingBlockEntry{
			meta:      b,
			minBlocks: s.MinInputBlocks,
			// Within group order by lowest compaction level first,
			// then lowest trace ID to try to co-locate traces even further.
			// This is the same for all cases.
			order: join(level, min),
		}

		switch {
		case b.CompactionLevel > 0 && shardMin == shardMax:
			// This is a previously split block that contains only a single shard.
			// Combine/dedupe with other blocks in the same window and shard.
			// This is prioritized over splitting new blocks to keep the block count down.
			entry.group = join(
				"A",            // Highest priority
				age,            // prioritize newer windows (more recent data)
				shardMin,       // same shard
				b.Version,      //
				b.DataEncoding, // same block format and column config
				columns,        //
			)
			entry.hash = join(
				b.TenantID, // Work sharding by tenant, window
				window,     //
				shardMin,   // and shard
			)

		case b.CompactionLevel == 0:
			// Unsharded new block that needs to be split. This step generates work
			// for the above step. MinBlocks=1 allows stray single blocks to still be split.
			entry.split = true
			entry.minBlocks = 1
			entry.group = join(
				"B",            // Second priority
				age,            //
				b.Version,      //
				b.DataEncoding, //
				columns,        //
			)
			entry.hash = join(
				b.TenantID, // Work sharding by tenant and window
				window,     //
			)

		default:
			// Existing block without sharding, or sharded under different settings.
			// Retroactively upgrade them but only if there is spare compactor
			// capacity, by assigning them the lowest priority.
			entry.split = true
			entry.minBlocks = 1
			entry.group = join(
				"C",            // lowest priority
				age,            //
				b.Version,      //
				b.DataEncoding, //
				columns,        //
			)
			entry.hash = join(
				b.TenantID, // Work sharding by tenant and window
				window,     //
			)
		}

		s.entries = append(s.entries, entry)
	}

	// sort by group then order
	sort.SliceStable(s.entries, func(i, j int) bool {
		ei := s.entries[i]
		ej := s.entries[j]

		if ei.group == ej.group {
			return ei.order < ej.order
		}
		return ei.group < ej.group
	})

	return s
}

func (s *shardingBlockSelector) shardOf(id common.ID) int {
	i, _ := slices.BinarySearchFunc(s.boundaries, []byte(id), bytes.Compare)
	return i
}

func (s *shardingBlockSelector) BlocksToCompact() common.CompactionRound {
	for len(s.entries) > 0 {
		var chosen []shardingBlockEntry

		// find everything from cursor forward that belongs to this group
		// Gather contiguous blocks while staying within limits
		i := 0
		for ; i < len(s.entries); i++ {
			chosen = s.entries[i:1]
			for j := i + 1; j < len(s.entries); j++ {
				stripe := s.entries[i : j+1]
				if s.entries[i].group == s.entries[j].group &&
					len(stripe) <= s.MaxInputBlocks &&
					totalObjects2(stripe) <= s.MaxCompactionObjects &&
					totalSize2(stripe) <= s.MaxBlockBytes {
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
		s.entries = s.entries[i+len(chosen):]

		// did we find enough blocks?
		if len(chosen) > 0 && len(chosen) >= chosen[0].minBlocks {
			res := shardedCompaction{
				maxCompactionObjects: s.MaxCompactionObjects,
				hash:                 chosen[0].hash,
				split:                chosen[0].split,
				shardOf:              s.shardOf,
			}
			for _, e := range chosen {
				res.blocks = append(res.blocks, e.meta)
			}
			return &res
		}
	}
	return &shardedCompaction{}
}

func (s *shardingBlockSelector) windowForBlock(meta *backend.BlockMeta) int64 {
	return s.windowForTime(meta.EndTime)
}

func (s *shardingBlockSelector) windowForTime(t time.Time) int64 {
	return t.Unix() / int64(s.MaxCompactionRange/time.Second)
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

// join is a minimum syntax wrapper for strings.Join
func join(ss ...string) string {
	return strings.Join(ss, "-")
}
