package tempodb

import (
	"fmt"
	"sort"
	"time"

	"github.com/grafana/tempo/tempodb/backend"
)

const (
	objectsPerHashBucket = 100000
)

// CompactionBlockSelector is an interface for different algorithms to pick suitable blocks for compaction
type CompactionBlockSelector interface {
	BlocksToCompact() ([]*backend.BlockMeta, string)
}

/*************************** Simple Block Selector **************************/

type simpleBlockSelector struct {
	cursor             int
	blocklist          []*backend.BlockMeta
	MaxCompactionRange time.Duration
}

var _ (CompactionBlockSelector) = (*simpleBlockSelector)(nil)

func (sbs *simpleBlockSelector) BlocksToCompact() ([]*backend.BlockMeta, string) {
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
	blocklist            []*backend.BlockMeta
	MaxCompactionRange   time.Duration // Size of the time window - say 6 hours
	MaxCompactionObjects int           // maximum size of compacted objects
}

var _ (CompactionBlockSelector) = (*timeWindowBlockSelector)(nil)

func newTimeWindowBlockSelector(blocklist []*backend.BlockMeta, maxCompactionRange time.Duration, maxCompactionObjects int) CompactionBlockSelector {
	twbs := &timeWindowBlockSelector{
		blocklist:            append([]*backend.BlockMeta(nil), blocklist...),
		MaxCompactionRange:   maxCompactionRange,
		MaxCompactionObjects: maxCompactionObjects,
	}

	return twbs
}

func (twbs *timeWindowBlockSelector) BlocksToCompact() ([]*backend.BlockMeta, string) {
	for len(twbs.blocklist) > 0 {
		// find everything from cursor forward that belongs to this block
		cursor := 0
		currentWindow := twbs.windowForBlock(twbs.blocklist[cursor])

		windowBlocks := make([]*backend.BlockMeta, 0)
		for cursor < len(twbs.blocklist) {
			currentBlock := twbs.blocklist[cursor]

			if currentWindow != twbs.windowForBlock(currentBlock) {
				break
			}
			cursor++

			windowBlocks = append(windowBlocks, currentBlock)
		}

		// did we find enough blocks?
		if len(windowBlocks) >= inputBlocks {
			var compactBlocks []*backend.BlockMeta

			// blocks in the currently active window
			// dangerous to use time.Now()
			activeWindow := twbs.windowForTime(time.Now())
			blockWindow := twbs.windowForBlock(windowBlocks[0])

			compact := true

			// the active window should be compacted by level
			if activeWindow <= blockWindow {
				sort.Slice(windowBlocks, func(i, j int) bool {
					return windowBlocks[i].CompactionLevel < windowBlocks[j].CompactionLevel
				})
				compactBlocks = windowBlocks[:inputBlocks]

				level := compactBlocks[0].CompactionLevel
				for _, block := range compactBlocks[1:] {
					if level != block.CompactionLevel {
						compact = false
						break
					}
				}
			} else if activeWindow-1 == blockWindow { // the most recent inactive window will be ignored to avoid race condittions
				compact = false
			} else { // all other windows will be compacted using their two smallest blocks
				sort.Slice(windowBlocks, func(i, j int) bool {
					return windowBlocks[i].TotalObjects < windowBlocks[j].TotalObjects
				})
				compactBlocks = windowBlocks[:inputBlocks]
			}

			// are they small enough
			totalObjects := 0
			for _, block := range compactBlocks {
				totalObjects += block.TotalObjects
			}
			if totalObjects > twbs.MaxCompactionObjects {
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

				return compactBlocks, fmt.Sprintf("%v-%v-%v", compactBlocks[0].TenantID, compactBlocks[0].CompactionLevel, currentWindow)
			}
		}

		// otherwise update the blocklist
		twbs.blocklist = twbs.blocklist[cursor:]
	}
	return nil, ""
}

func (twbs *timeWindowBlockSelector) windowForBlock(meta *backend.BlockMeta) int64 {
	return twbs.windowForTime(meta.EndTime)
}

func (twbs *timeWindowBlockSelector) windowForTime(t time.Time) int64 {
	return t.Unix() / int64(twbs.MaxCompactionRange/time.Second)
}
