package blockselector

import (
	"fmt"
	"sort"
	"time"

	"github.com/grafana/tempo/tempodb/backend"
)

// CompactionBlockSelector is an interface for different algorithms to pick suitable blocks for compaction
type CompactionBlockSelector interface {
	BlocksToCompact() ([]*backend.BlockMeta, string)
}

const (
	activeWindowDuration  = 24 * time.Hour
	DefaultMinInputBlocks = 2
	DefaultMaxInputBlocks = 4
)

/*************************** Time Window Block Selector **************************/

// Sharding will be based on time slot - not level. Since each compactor works on two levels.
// Levels will be needed for id-range isolation
// The timeWindowBlockSelector can be used ONLY ONCE PER TIMESLOT.
// It needs to be reinitialized with updated blocklist.

type timeWindowBlockSelector struct {
	MinInputBlocks       int
	MaxInputBlocks       int
	MaxCompactionRange   time.Duration // Size of the time window - say 6 hours
	MaxCompactionObjects int           // maximum size of compacted objects
	MaxBlockBytes        uint64        // maximum block size, estimate

	entries []timeWindowBlockEntry
}

type timeWindowBlockEntry struct {
	meta  *backend.BlockMeta
	group string // Blocks in the same group will be compacted together. Sort order also determines group priority.
	order string // Individual block priority within the group.
	hash  string // hash string used for sharding ownership, preserves backwards compatibility
}

var _ (CompactionBlockSelector) = (*timeWindowBlockSelector)(nil)

func NewTimeWindowBlockSelector(blocklist []*backend.BlockMeta, maxCompactionRange time.Duration, maxCompactionObjects int, maxBlockBytes uint64, minInputBlocks, maxInputBlocks int) CompactionBlockSelector {
	twbs := &timeWindowBlockSelector{
		MinInputBlocks:       minInputBlocks,
		MaxInputBlocks:       maxInputBlocks,
		MaxCompactionRange:   maxCompactionRange,
		MaxCompactionObjects: maxCompactionObjects,
		MaxBlockBytes:        maxBlockBytes,
	}

	now := time.Now()
	currWindow := twbs.windowForTime(now)
	activeWindow := twbs.windowForTime(now.Add(-activeWindowDuration))

	for _, b := range blocklist {
		w := twbs.windowForBlock(b)

		// exclude blocks that fall in last window from active -> inactive cut-over
		// blocks in this window will not be compacted in order to avoid
		// ownership conflicts where two compactors process the same block
		// at the same time as it transitions from last active window to first inactive window.
		if w == activeWindow {
			continue
		}

		entry := timeWindowBlockEntry{
			meta: b,
		}

		age := currWindow - w
		if activeWindow <= w {
			// inside active window.
			// Group by compaction level and window.
			// Choose lowest compaction level and most recent windows first.
			entry.group = fmt.Sprintf("A-%v-%016X-%v", b.CompactionLevel, age, b.ReplicationFactor)

			// Within group choose smallest blocks first.
			// update after parquet: we want to make sure blocks of the same version end up together
			// update afert vParquet3: we want to make sure blocks of the same dedicated columns end up together
			entry.order = fmt.Sprintf("%016X-%v-%016X", entry.meta.TotalObjects, entry.meta.Version, entry.meta.DedicatedColumnsHash())

			entry.hash = fmt.Sprintf("%v-%v-%v-%v", b.TenantID, b.CompactionLevel, w, b.ReplicationFactor)
		} else {
			// outside active window.
			// Group by window only.  Choose most recent windows first.
			entry.group = fmt.Sprintf("B-%016X-%v", age, b.ReplicationFactor)

			// Within group chose lowest compaction lvl and smallest blocks first.
			// update after parquet: we want to make sure blocks of the same version end up together
			// update afert vParquet3: we want to make sure blocks of the same dedicated columns end up together
			entry.order = fmt.Sprintf("%v-%016X-%v-%016X", b.CompactionLevel, entry.meta.TotalObjects, entry.meta.Version, entry.meta.DedicatedColumnsHash())

			entry.hash = fmt.Sprintf("%v-%v-%v", b.TenantID, w, b.ReplicationFactor)
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

func (twbs *timeWindowBlockSelector) BlocksToCompact() ([]*backend.BlockMeta, string) {
	for len(twbs.entries) > 0 {
		var chosen []timeWindowBlockEntry

		// find everything from cursor forward that belongs to this group
		// Gather contiguous blocks while staying within limits
		i := 0
		for ; i < len(twbs.entries); i++ {
			for j := i + 1; j < len(twbs.entries); j++ {
				stripe := twbs.entries[i : j+1]

				if twbs.entries[i].group == twbs.entries[j].group &&
					twbs.entries[i].meta.DataEncoding == twbs.entries[j].meta.DataEncoding &&
					twbs.entries[i].meta.Version == twbs.entries[j].meta.Version && // update after parquet: only compact blocks of the same version
					twbs.entries[i].meta.DedicatedColumnsHash() == twbs.entries[j].meta.DedicatedColumnsHash() && // update after vParquet3: only compact blocks of the same dedicated columns
					len(stripe) <= twbs.MaxInputBlocks &&
					totalObjects(stripe) <= twbs.MaxCompactionObjects &&
					totalSize(stripe) <= twbs.MaxBlockBytes {
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

			compactBlocks := make([]*backend.BlockMeta, 0)
			for _, e := range chosen {
				compactBlocks = append(compactBlocks, e.meta)
			}

			return compactBlocks, chosen[0].hash
		}
	}
	return nil, ""
}

func totalObjects(entries []timeWindowBlockEntry) int {
	totalObjects := 0
	for _, b := range entries {
		totalObjects += int(b.meta.TotalObjects)
	}
	return totalObjects
}

func totalSize(entries []timeWindowBlockEntry) uint64 {
	sz := uint64(0)
	for _, b := range entries {
		sz += b.meta.Size_
	}
	return sz
}

func (twbs *timeWindowBlockSelector) windowForBlock(meta *backend.BlockMeta) int64 {
	return twbs.windowForTime(meta.EndTime)
}

func (twbs *timeWindowBlockSelector) windowForTime(t time.Time) int64 {
	return t.Unix() / int64(twbs.MaxCompactionRange/time.Second)
}
