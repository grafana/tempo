package tempodb

import (
	"time"

	"github.com/grafana/tempo/tempodb/backend"
)

// CompactionBlockSelector is an interface for different algorithms to pick suitable blocks for compaction
type CompactionBlockSelector interface {
	ResetCursor()
	BlocksToCompactInSameLevel(blocklist []*backend.BlockMeta) int
	// BlocksToCompactAcrossLevels(block *backend.BlockMeta, blocklist []*backend.BlockMeta) []*backend.BlockMeta
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
	return
}

// todo: switch to iterator pattern?
func (sbs *simpleBlockSelector) BlocksToCompactInSameLevel(blocklist []*backend.BlockMeta) int {
	// should never happen
	if inputBlocks > len(blocklist) {
		return -1
	}

	for cursor := 0; cursor < len(blocklist)-inputBlocks+1; cursor++ {
		if blocklist[cursor+inputBlocks-1].EndTime.Sub(blocklist[cursor].StartTime) < sbs.MaxCompactionRange {
			return cursor
		}
	}

	// Could not find blocks suitable for compaction, break
	return -1

}
