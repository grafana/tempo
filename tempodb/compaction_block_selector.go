package tempodb

import (
	"fmt"
	"sort"
	"time"

	"github.com/grafana/tempo/tempodb/encoding"
)

// CompactionBlockSelector is an interface for different algorithms to pick suitable blocks for compaction
type CompactionBlockSelector interface {
	BlocksToCompact() ([]*encoding.BlockMeta, string)
}

const (
	activeWindowDuration  = 24 * time.Hour
	defaultMinInputBlocks = 2
	defaultMaxInputBlocks = 8
)

/*************************** Simple Block Selector **************************/

type simpleBlockSelector struct {
	cursor             int
	blocklist          []*encoding.BlockMeta
	MaxCompactionRange time.Duration
}

var _ (CompactionBlockSelector) = (*simpleBlockSelector)(nil)

func (sbs *simpleBlockSelector) BlocksToCompact() ([]*encoding.BlockMeta, string) {
	// should never happen
	if inputBlocks > len(sbs.blocklist) {
		return nil, ""
	}

	for sbs.cursor < len(sbs.blocklist)-inputBlocks+1 {
		cursorEnd := sbs.cursor + inputBlocks - 1
		if sbs.blocklist[cursorEnd].EndTime.Sub(sbs.blocklist[sbs.cursor].StartTime) < sbs.MaxCompactionRange {
			startPos := sbs.cursor
			sbs.cursor = startPos + inputBlocks
			hashString := sbs.blocklist[startPos].TenantID

			return sbs.blocklist[startPos : startPos+inputBlocks], hashString
		}
		sbs.cursor++
	}

	return nil, ""
}

/*************************** Time Window Block Selector **************************/

// Sharding will be based on time slot - not level. Since each compactor works on two levels.
// Levels will be needed for id-range isolation
// The timeWindowBlockSelector can be used ONLY ONCE PER TIMESLOT.
// It needs to be reinitialized with updated blocklist.

type timeWindowBlockSelector struct {
	blocklist            []*encoding.BlockMeta
	MinInputBlocks       int
	MaxInputBlocks       int
	MaxCompactionRange   time.Duration // Size of the time window - say 6 hours
	MaxCompactionObjects int           // maximum size of compacted objects
}

var _ (CompactionBlockSelector) = (*timeWindowBlockSelector)(nil)

func newTimeWindowBlockSelector(blocklist []*encoding.BlockMeta, maxCompactionRange time.Duration, maxCompactionObjects int, minInputBlocks int, maxInputBlocks int) CompactionBlockSelector {
	twbs := &timeWindowBlockSelector{
		blocklist:            append([]*encoding.BlockMeta(nil), blocklist...),
		MinInputBlocks:       minInputBlocks,
		MaxInputBlocks:       maxInputBlocks,
		MaxCompactionRange:   maxCompactionRange,
		MaxCompactionObjects: maxCompactionObjects,
	}

	activeWindow := twbs.windowForTime(time.Now().Add(-activeWindowDuration))

	// exclude blocks that fall in last window from active -> inactive cut-over
	// blocks in this window will not be compacted in order to avoid
	// ownership conflicts where two compactors process the same block
	// at the same time as it transitions from last active window to first inactive window.
	var newBlocks []*encoding.BlockMeta
	for _, b := range twbs.blocklist {
		if twbs.windowForBlock(b) != activeWindow {
			newBlocks = append(newBlocks, b)
		}
	}
	twbs.blocklist = newBlocks

	// sort by compaction window, level, and then size
	sort.Slice(twbs.blocklist, func(i, j int) bool {
		bi := twbs.blocklist[i]
		bj := twbs.blocklist[j]

		wi := twbs.windowForBlock(bi)
		wj := twbs.windowForBlock(bj)

		if activeWindow <= wi && activeWindow <= wj {
			// inside active window.  sort by:  compaction lvl -> window -> size
			//  we should always choose the smallest two blocks whos compaction lvl and windows match
			if bi.CompactionLevel != bj.CompactionLevel {
				return bi.CompactionLevel < bj.CompactionLevel
			}

			if wi != wj {
				return wi > wj
			}
		} else {
			// outside active window.  sort by: window -> compaction lvl -> size
			//  we should always choose the most recent two blocks that can be compacted
			if wi != wj {
				return wi > wj
			}

			if bi.CompactionLevel != bj.CompactionLevel {
				return bi.CompactionLevel < bj.CompactionLevel
			}
		}

		return bi.TotalObjects < bj.TotalObjects
	})

	return twbs
}

func (twbs *timeWindowBlockSelector) BlocksToCompact() ([]*encoding.BlockMeta, string) {
	for len(twbs.blocklist) > 0 {
		// find everything from cursor forward that belongs to this block
		cursor := 0
		currentWindow := twbs.windowForBlock(twbs.blocklist[cursor])

		windowBlocks := make([]*encoding.BlockMeta, 0)
		for cursor < len(twbs.blocklist) {
			currentBlock := twbs.blocklist[cursor]

			if currentWindow != twbs.windowForBlock(currentBlock) {
				break
			}
			cursor++

			windowBlocks = append(windowBlocks, currentBlock)
		}

		// did we find enough blocks?
		if len(windowBlocks) >= twbs.MinInputBlocks {
			var compactBlocks []*encoding.BlockMeta

			// blocks in the currently active window
			// dangerous to use time.Now()
			activeWindow := twbs.windowForTime(time.Now().Add(-activeWindowDuration))
			blockWindow := twbs.windowForBlock(windowBlocks[0])

			hashString := fmt.Sprintf("%v", windowBlocks[0].TenantID)
			compact := true

			// the active window should be compacted by level
			if activeWindow <= blockWindow {
				// search forward for inputBlocks in a row that have the same compaction level
				// Gather as many as possible while staying within limits
				for i := 0; i <= len(windowBlocks)-twbs.MinInputBlocks+1; i++ {
					for j := i + 1; j <= len(windowBlocks)-1 &&
						windowBlocks[i].CompactionLevel == windowBlocks[j].CompactionLevel &&
						len(compactBlocks)+1 <= twbs.MaxInputBlocks &&
						totalObjects(compactBlocks)+windowBlocks[j].TotalObjects <= twbs.MaxCompactionObjects; j++ {
						compactBlocks = windowBlocks[i : j+1]
					}
					if len(compactBlocks) > 0 {
						// Found a stripe of blocks
						break
					}
				}

				compact = false
				if len(compactBlocks) >= twbs.MinInputBlocks {
					compact = true
					hashString = fmt.Sprintf("%v-%v-%v", compactBlocks[0].TenantID, compactBlocks[0].CompactionLevel, currentWindow)
				}
			} else { // all other windows will be compacted using their two smallest blocks
				compactBlocks = windowBlocks[:twbs.MinInputBlocks]
				hashString = fmt.Sprintf("%v-%v", compactBlocks[0].TenantID, currentWindow)
			}

			if totalObjects(compactBlocks) > twbs.MaxCompactionObjects {
				compact = false
			}

			if compact {
				// remove the blocks we are returning so we don't consider them again
				//   this is horribly inefficient as it's written
				for _, blockToCompact := range compactBlocks {
					for i, block := range twbs.blocklist {
						if block == blockToCompact {
							copy(twbs.blocklist[i:], twbs.blocklist[i+1:])
							twbs.blocklist[len(twbs.blocklist)-1] = nil
							twbs.blocklist = twbs.blocklist[:len(twbs.blocklist)-1]

							break
						}
					}
				}

				return compactBlocks, hashString
			}
		}

		// otherwise update the blocklist
		twbs.blocklist = twbs.blocklist[cursor:]
	}
	return nil, ""
}

func totalObjects(blocks []*encoding.BlockMeta) int {
	totalObjects := 0
	for _, b := range blocks {
		totalObjects += b.TotalObjects
	}
	return totalObjects
}

func (twbs *timeWindowBlockSelector) windowForBlock(meta *encoding.BlockMeta) int64 {
	return twbs.windowForTime(meta.EndTime)
}

func (twbs *timeWindowBlockSelector) windowForTime(t time.Time) int64 {
	return t.Unix() / int64(twbs.MaxCompactionRange/time.Second)
}
