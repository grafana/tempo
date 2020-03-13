package tempodb

import (
	"time"

	"github.com/grafana/tempo/tempodb/backend"
)

// CompactionBlockSelector is an interface for different algorithms to pick suitable blocks for compaction
type CompactionBlockSelector interface {
	BlocksToCompact() []*backend.BlockMeta
}

/*************************** Simple Block Selector **************************/

type simpleBlockSelector struct {
	cursor             int
	blocklist          []*backend.BlockMeta
	MaxCompactionRange time.Duration
}

var _ (CompactionBlockSelector) = (*simpleBlockSelector)(nil)

func newSimpleBlockSelector(blocklist []*backend.BlockMeta, maxCompactionRange time.Duration) CompactionBlockSelector {
	return &simpleBlockSelector{
		blocklist:          blocklist,
		MaxCompactionRange: maxCompactionRange,
	}
}

func (sbs *simpleBlockSelector) BlocksToCompact() []*backend.BlockMeta {
	// should never happen
	if inputBlocks > len(sbs.blocklist) {
		return nil
	}

	for sbs.cursor < len(sbs.blocklist)-inputBlocks+1 {
		cursorEnd := sbs.cursor + inputBlocks - 1
		if sbs.blocklist[cursorEnd].EndTime.Sub(sbs.blocklist[sbs.cursor].StartTime) < sbs.MaxCompactionRange {
			startPos := sbs.cursor
			sbs.cursor = startPos + inputBlocks
			return sbs.blocklist[startPos : startPos+inputBlocks]
		}
		sbs.cursor++
	}

	// Could not find blocks suitable for compaction, break
	return nil
}
