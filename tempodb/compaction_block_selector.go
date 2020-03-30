package tempodb

import (
	"container/heap"
	"fmt"
	"time"

	"github.com/grafana/tempo/tempodb/backend"
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
	cursor               int
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
	var blocksToCompact BlockMetaHeap

	for twbs.cursor < len(twbs.blocklist) {
		blocksToCompact = BlockMetaHeap(make([]*backend.BlockMeta, 0))
		heap.Init(&blocksToCompact)

		// find everything from cursor forward that belongs to this block
		cursorEnd := twbs.cursor
		currentWindow := twbs.windowForBlock(twbs.blocklist[twbs.cursor])

		for cursorEnd < len(twbs.blocklist) {
			currentBlock := twbs.blocklist[cursorEnd]

			if currentWindow != twbs.windowForBlock(currentBlock) {
				break
			}
			cursorEnd++

			heap.Push(&blocksToCompact, currentBlock)
		}

		// did we find enough blocks?
		if len(blocksToCompact) >= inputBlocks {

			// pop all but the ones we want
			for len(blocksToCompact) > inputBlocks {
				heap.Pop(&blocksToCompact)
			}

			// are they small enough
			totalObjects := 0
			for _, blocksToCompact := range blocksToCompact {
				totalObjects += blocksToCompact.TotalObjects
			}

			if totalObjects < twbs.MaxCompactionObjects {
				// remove the blocks we are returning so we don't consider them again
				//   this is horribly inefficient as it's written
				for _, blockToCompact := range blocksToCompact {
					for i, block := range twbs.blocklist {
						if block == blockToCompact {
							copy(twbs.blocklist[i:], twbs.blocklist[i+1:])
							twbs.blocklist[len(twbs.blocklist)-1] = nil
							twbs.blocklist = twbs.blocklist[:len(twbs.blocklist)-1]

							break
						}
					}
				}

				return blocksToCompact, fmt.Sprintf("%v-%v-%v", blocksToCompact[0].TenantID, totalObjects/100000, currentWindow)
			}
		}

		// otherwise update the cursor and attempt the next window
		twbs.cursor = cursorEnd
	}
	return nil, ""
}

func (twbs *timeWindowBlockSelector) windowForBlock(meta *backend.BlockMeta) int64 {
	return meta.StartTime.Unix() / int64(twbs.MaxCompactionRange/time.Second)
}

type BlockMetaHeap []*backend.BlockMeta

func (h BlockMetaHeap) Len() int {
	return len(h)
}

func (h BlockMetaHeap) Less(i, j int) bool {
	return h[i].TotalObjects > h[j].TotalObjects
}

func (h BlockMetaHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

func (h *BlockMetaHeap) Push(x interface{}) {
	item := x.(*backend.BlockMeta)
	*h = append(*h, item)
}

func (h *BlockMetaHeap) Pop() interface{} {
	old := *h
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	*h = old[0 : n-1]
	return item
}
