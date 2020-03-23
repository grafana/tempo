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

	return nil
}

/*************************** Time Window Block Selector **************************/

// Sharding will be based on time slot - not level. Since each compactor works on two levels.
// Levels will be needed for id-range isolation
// The timeWindowBlockSelector can be used ONLY ONCE PER TIMESLOT.
// It needs to be reinitialized with updated blocklist.

type timeWindowBlockSelector struct {
	cursor             int
	blocklist          []*backend.BlockMeta
	MaxCompactionRange time.Duration // Size of the time window - say 6 hours
}

var _ (CompactionBlockSelector) = (*timeWindowBlockSelector)(nil)

func newTimeWindowBlockSelector(blocklist []*backend.BlockMeta, maxCompactionRange time.Duration) CompactionBlockSelector {
	twbs := &timeWindowBlockSelector{
		blocklist:          blocklist,
		MaxCompactionRange: maxCompactionRange,
	}

	return twbs
}

func (twbs *timeWindowBlockSelector) BlocksToCompact() []*backend.BlockMeta {

	for twbs.cursor < len(twbs.blocklist)-inputBlocks+1 {
		// Pick blocks in slotStartTime <> slotEndTime
		cursorBlock := twbs.blocklist[twbs.cursor]
		currentWindow := twbs.windowForBlock(cursorBlock)
		cursorEnd := twbs.cursor + inputBlocks - 1

		if cursorEnd < len(twbs.blocklist) && currentWindow == twbs.windowForBlock(twbs.blocklist[cursorEnd]) {
			startPos := twbs.cursor
			twbs.cursor = startPos + inputBlocks
			return twbs.blocklist[startPos : startPos+inputBlocks]
		}
		twbs.cursor++
	}
	return nil
}

func (twbs *timeWindowBlockSelector) windowForBlock(meta *backend.BlockMeta) int64 {
	return meta.StartTime.Unix() / int64(twbs.MaxCompactionRange/time.Second)
}
