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

/*************************** Time Window Block Selector **************************/

type timeWindowBlockSelector struct {
	cursor             int
	slotStartTime      time.Time
	blocklist          []*backend.BlockMeta
	MaxCompactionRange time.Duration // Size of the time window - say 6 hours
}

var _ (CompactionBlockSelector) = (*timeWindowBlockSelector)(nil)

func newTimeWindowBlockSelector(blocklist []*backend.BlockMeta, maxCompactionRange time.Duration) CompactionBlockSelector {
	twbs := &timeWindowBlockSelector{
		blocklist:          blocklist,
		MaxCompactionRange: maxCompactionRange,
	}

	y, m, d := blocklist[0].StartTime.Date()
	twbs.slotStartTime = time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	return twbs
}

func (twbs *timeWindowBlockSelector) BlocksToCompact() []*backend.BlockMeta {
	// Pick the same time window as the start block
	startTime := twbs.blocklist[twbs.cursor].StartTime
	for twbs.slotStartTime.Add(twbs.MaxCompactionRange).Sub(startTime).Seconds() < 0 {
		twbs.slotStartTime.Add(twbs.MaxCompactionRange)
	}

	for twbs.cursor < len(twbs.blocklist)-inputBlocks+1 {
		// Pick blocks in slotStartTime <> slotEndTime
		slotEndTime := twbs.slotStartTime.Add(twbs.MaxCompactionRange)
		cursorEnd := twbs.cursor + inputBlocks - 1
		if twbs.blocklist[cursorEnd].StartTime.Sub(slotEndTime).Seconds() <= 0 {
			startPos := twbs.cursor
			twbs.cursor = startPos + inputBlocks
			return twbs.blocklist[startPos : startPos+inputBlocks]
		}
		twbs.cursor++
		twbs.slotStartTime.Add(twbs.MaxCompactionRange)
	}

	return nil
}
