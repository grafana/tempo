package main

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/grafana/tempo/tempodb/backend"
)

type analyseBlocksCmd struct {
	backendOptions

	Jsonnet            bool   `help:"output jsonnet nexessary for overrides"`
	TenantID           string `arg:"" help:"tenant-id within the bucket"`
	MinCompactionLevel int    `help:"Min compaction level to analyse" default:"3"`
	MaxBlocks          int    `help:"Max number of blocks to analyse" default:"10"`
	NumAttr            int    `help:"Number of attributes to display" default:"15"`
	MaxStartTime       string `help:"Oldest start time for a block to be processed. RFC3339 format '2006-01-02T15:04:05Z07:00'" default:""`
	MinStartTime       string `help:"Newest start time for a block to be processed. RFC3339 format '2006-01-02T15:04:05Z07:00'" default:""`
}

func (cmd *analyseBlocksCmd) Run(ctx *globalOptions) error {
	r, _, _, err := loadBackend(&cmd.backendOptions, ctx)
	if err != nil {
		return err
	}

	// TODO: Parallelize this
	blocks, _, err := r.Blocks(context.Background(), cmd.TenantID)
	if err != nil {
		return err
	}

	processedBlocks := map[uuid.UUID]struct{}{}
	topSpanAttrs, topResourceAttrs := make(map[string]uint64), make(map[string]uint64)
	totalSpanBytes, totalResourceBytes := uint64(0), uint64(0)

	var maxStartTime, minStartTime time.Time
	if cmd.MaxStartTime != "" {
		maxStartTime, err = time.Parse(time.RFC3339, cmd.MaxStartTime)
		if err != nil {
			return err
		}
	}
	if cmd.MinStartTime != "" {
		minStartTime, err = time.Parse(time.RFC3339, cmd.MinStartTime)
		if err != nil {
			return err
		}
	}

	for i := 0; i < len(blocks) && len(processedBlocks) < cmd.MaxBlocks; i++ {
		block := blocks[i]
		if _, ok := processedBlocks[block]; ok {
			continue
		}

		blockSum, err := processBlock(r, cmd.TenantID, block.String(), maxStartTime, minStartTime, uint8(cmd.MinCompactionLevel))
		if err != nil {
			if !errors.Is(err, backend.ErrDoesNotExist) {
				return err
			}

			// the block was already compacted and blocks might be outdated: refreshing blocks
			blocks, _, err = r.Blocks(context.Background(), cmd.TenantID)
			if err != nil {
				return err
			}
			i = -1

			continue
		}

		if blockSum == nil {
			continue
		}

		for k, v := range blockSum.spanSummary.attributes {
			topSpanAttrs[k] += v
		}
		totalSpanBytes += blockSum.spanSummary.totalBytes

		for k, v := range blockSum.resourceSummary.attributes {
			topResourceAttrs[k] += v
		}
		totalResourceBytes += blockSum.resourceSummary.totalBytes

		processedBlocks[block] = struct{}{}
	}

	// Get top N attributes from map
	return (&blockSummary{
		spanSummary: genericAttrSummary{
			totalBytes: totalSpanBytes,
			attributes: topSpanAttrs,
		},
		resourceSummary: genericAttrSummary{
			totalBytes: totalResourceBytes,
			attributes: topResourceAttrs,
		},
	}).print(cmd.NumAttr, cmd.Jsonnet)
}
