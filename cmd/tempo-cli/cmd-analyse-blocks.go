package main

import (
	"context"
	"time"
)

type analyseBlocksCmd struct {
	backendOptions

	TenantID           string `arg:"" help:"tenant-id within the bucket"`
	MinCompactionLevel int    `arg:"" help:"Min compaction level to analyse" default:"3"`
	MaxBlocks          int    `arg:"" help:"Max number of blocks to analyse" default:"10"`
	NumAttr            int    `arg:"" help:"Number of attributes to display" default:"15"`
}

func (cmd *analyseBlocksCmd) Run(ctx *globalOptions) error {
	r, _, c, err := loadBackend(&cmd.backendOptions, ctx)
	if err != nil {
		return err
	}

	blocks, err := r.Blocks(context.Background(), cmd.TenantID)
	if err != nil {
		return err
	}

	processedBlocks := 0
	topGlobalAttrs := make(map[string]uint64)
	totalBytes := uint64(0)
	for _, block := range blocks {
		if processedBlocks >= cmd.MaxBlocks {
			break
		}

		blockSum, err := processBlock(r, c, cmd.TenantID, block.String(), time.Hour, uint8(cmd.MinCompactionLevel))
		if err != nil {
			return err
		}

		if blockSum == nil {
			continue
		}

		for k, v := range blockSum.spanSummary.attributes {
			topGlobalAttrs[k] += v
		}
		totalBytes += blockSum.spanSummary.totalBytes

		processedBlocks++
	}
	// Get top N attributes from map
	return (&blockSummary{
		spanSummary: genericAttrSummary{
			totalBytes: totalBytes,
			attributes: topGlobalAttrs,
		},
	}).print(cmd.NumAttr)
}
