package benchmark

import (
	"fmt"

	"github.com/grafana/tempo/v3/tempodb/backend"
	"github.com/grafana/tempo/v3/tempodb/blocksharding"
)

// Shard is a contiguous row-group range, in the units
// common.SearchOptions.StartPage and TotalPages use.
type Shard struct {
	Index      int `json:"index"`
	StartPage  int `json:"startPage"`
	TotalPages int `json:"totalPages"`
}

// shardsForBlock splits the block into the jobs the query frontend would issue
// for it, using the frontend's own page sizing.
//
// The bound is meta.TotalRecords and the last shard is not clamped to what the
// block actually holds, because that is what production does: a job whose page
// range runs past the end simply reads fewer row groups.
func shardsForBlock(meta *backend.BlockMeta, targetBytesPerRequest int) ([]Shard, error) {
	pages := blocksharding.PagesPerRequest(meta, targetBytesPerRequest)
	if pages <= 0 {
		return nil, fmt.Errorf("block %s: cannot size a shard from %d bytes over %d row groups", meta.BlockID, meta.Size_, meta.TotalRecords)
	}

	records := int(meta.TotalRecords)
	shards := make([]Shard, 0, (records+pages-1)/pages)
	for start := 0; start < records; start += pages {
		shards = append(shards, Shard{
			Index:      len(shards),
			StartPage:  start,
			TotalPages: pages,
		})
	}
	return shards, nil
}
