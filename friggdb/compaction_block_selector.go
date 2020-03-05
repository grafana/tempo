package friggdb

import (
	"time"

	"github.com/grafana/frigg/friggdb/backend"
)

// CompactionBlockSelector is an interface for different algorithms to pick suitable blocks for compaction
type CompactionBlockSelector interface {
	BlocksToCompact(blocklist []*backend.BlockMeta) []*backend.BlockMeta
	IsRunning(tenantID string) bool // Added for now, but rethink this
}

/*************************** Simple Block Selector **************************/

type simpleBlockSelector struct {
	// no need for a lock around cursor.
	// Even if multiple maintenance cycles execute simultaneously,
	// simpleBlockSelector will not run in parallel yet because of rw.blockListsMtx.Lock().
	cursor             map[string]int // per-tenant because each of these is parallel
	MaxCompactionRange time.Duration
}

var _ (CompactionBlockSelector) = (*simpleBlockSelector)(nil)

func newSimpleBlockSelector(maxCompactionRange time.Duration) CompactionBlockSelector {
	return &simpleBlockSelector{
		cursor:             make(map[string]int),
		MaxCompactionRange: maxCompactionRange,
	}
}

// todo: switch to iterator pattern?
func (sbs *simpleBlockSelector) BlocksToCompact(blocklist []*backend.BlockMeta) []*backend.BlockMeta {
	if blocklist == nil {
		return nil
	}

	// loop through blocks starting at cursor for the given tenant, blocks are sorted by start date so candidates for compaction should be near each other
	//   - consider candidateBlocks at a time.
	//   - find the blocks with the fewest records that are within the compaction range
	myTenant := blocklist[0].TenantID
	if inputBlocks > len(blocklist) {
		// Not enough blocks to compact, break
		sbs.cursor[myTenant] = 0
		return nil
	}

	if _, ok := sbs.cursor[myTenant]; !ok {
		sbs.cursor[myTenant] = 0
	}

	cursorEnd := sbs.cursor[myTenant] + inputBlocks - 1
	for {
		if cursorEnd >= len(blocklist) {
			break
		}

		blockStart := blocklist[sbs.cursor[myTenant]]
		blockEnd := blocklist[cursorEnd]

		if blockEnd.EndTime.Sub(blockStart.StartTime) < sbs.MaxCompactionRange {
			sbs.cursor[myTenant] = cursorEnd + 1
			return blocklist[sbs.cursor[myTenant] : cursorEnd+1]
		}

		sbs.cursor[myTenant]++
		cursorEnd = sbs.cursor[myTenant] + inputBlocks - 1
	}

	// Could not find blocks suitable for compaction, break
	sbs.cursor[myTenant] = 0
	return nil
}

func (sbs *simpleBlockSelector) IsRunning(tenantID string) bool {
	return sbs.cursor[tenantID] != 0
}

/*************************** Distributed Block Selector **************************/

type distributedBlockSelector struct {
	// will eventually hold ring state
}
