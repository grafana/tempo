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

// Sharding will be based on time slot - not level. Since each compactor works on two levels.
// Levels will be needed for id-range isolation
// The timeWindowBlockSelector can be used ONLY ONCE PER TIMESLOT.
// It needs to be reinitialized with updated blocklist.

type timeWindowBlockSelector struct {
	cursor             int
	slotStartTime      time.Time // this will eventually be passed to the selector via ring selection
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
	levelStartTime := twbs.blocklist[0].StartTime
	i := twbs.slotStartTime
	for i.Add(twbs.MaxCompactionRange).Before(levelStartTime) {
		twbs.slotStartTime.Add(twbs.MaxCompactionRange)
	}
	slotEndTime := twbs.slotStartTime.Add(twbs.MaxCompactionRange)

	for twbs.cursor < len(twbs.blocklist)-inputBlocks+1 {
		// Pick blocks in slotStartTime <> slotEndTime
		cursorBlock := twbs.blocklist[twbs.cursor]
		if cursorBlock.StartTime.After(twbs.slotStartTime) && cursorBlock.StartTime.Before(slotEndTime) {
			// Pick inputBlocks and promote to next level
			cursorEnd := twbs.cursor + inputBlocks - 1
			if cursorEnd < len(twbs.blocklist) && twbs.blocklist[cursorEnd].StartTime.Before(slotEndTime) {
				startPos := twbs.cursor
				twbs.cursor = startPos + inputBlocks
				return twbs.blocklist[startPos : startPos+inputBlocks]
			}
		}
		twbs.cursor++
		if twbs.blocklist[twbs.cursor].StartTime.After(slotEndTime) {
			twbs.slotStartTime.Add(twbs.MaxCompactionRange)
			slotEndTime = twbs.slotStartTime.Add(twbs.MaxCompactionRange)
		}
	}
	return nil
}
