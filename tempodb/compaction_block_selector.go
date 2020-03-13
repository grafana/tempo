package tempodb

import (
	"time"

	"github.com/grafana/tempo/tempodb/backend"
)

// CompactionBlockSelector is an interface for different algorithms to pick suitable blocks for compaction
type CompactionBlockSelector interface {
	ResetCursor()
	BlocksToCompactInSameLevel(blocklist []*backend.BlockMeta) int
}

/*************************** Simple Block Selector **************************/

type simpleBlockSelector struct {
	cursor             int
	MaxCompactionRange time.Duration
}

var _ (CompactionBlockSelector) = (*simpleBlockSelector)(nil)

func newSimpleBlockSelector(maxCompactionRange time.Duration) CompactionBlockSelector {
	return &simpleBlockSelector{
		MaxCompactionRange: maxCompactionRange,
	}
}

func (sbs *simpleBlockSelector) ResetCursor() {
	sbs.cursor = 0
}

// todo: switch to iterator pattern?
func (sbs *simpleBlockSelector) BlocksToCompactInSameLevel(blocklist []*backend.BlockMeta) int {
	// should never happen
	if inputBlocks > len(blocklist) {
		return -1
	}

	for sbs.cursor < len(blocklist)-inputBlocks+1 {
		if blocklist[sbs.cursor+inputBlocks-1].EndTime.Sub(blocklist[sbs.cursor].StartTime) < sbs.MaxCompactionRange {
			retMe := sbs.cursor
			sbs.cursor++
			return retMe
		}
		sbs.cursor++
	}

	// Could not find blocks suitable for compaction, break
	return -1
}
