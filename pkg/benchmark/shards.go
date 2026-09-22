package benchmark

import (
	"fmt"

	"github.com/grafana/tempo/v3/tempodb/backend"
)

// Shard is a contiguous row-group range, in the units
// common.SearchOptions.StartPage and TotalPages use.
type Shard struct {
	Index      int `json:"index"`
	StartPage  int `json:"startPage"`
	TotalPages int `json:"totalPages"`
}

// shardsForBlock splits the block the way the query frontend splits a search:
// the unit is row groups, but a byte target decides how many go in a shard.
// Mirrors modules/frontend.pagesPerRequest.
func shardsForBlock(meta *backend.BlockMeta, rowGroups, targetBytesPerRequest int) ([]Shard, error) {
	pages := pagesPerShard(meta, rowGroups, targetBytesPerRequest)
	if pages <= 0 {
		return nil, fmt.Errorf("block %s: cannot size a shard from %d bytes over %d row groups", meta.BlockID, meta.Size_, rowGroups)
	}

	shards := make([]Shard, 0, (rowGroups+pages-1)/pages)
	for start := 0; start < rowGroups; start += pages {
		shards = append(shards, Shard{
			Index:      len(shards),
			StartPage:  start,
			TotalPages: min(pages, rowGroups-start),
		})
	}
	return shards, nil
}

func pagesPerShard(meta *backend.BlockMeta, rowGroups, targetBytesPerRequest int) int {
	if meta.Size_ == 0 || rowGroups == 0 {
		return 0
	}
	// A block smaller than the target is one job.
	if meta.Size_ < uint64(targetBytesPerRequest) {
		return rowGroups
	}

	bytesPerPage := meta.Size_ / uint64(rowGroups)
	if bytesPerPage == 0 {
		return 0
	}
	return max(targetBytesPerRequest/int(bytesPerPage), 1)
}
