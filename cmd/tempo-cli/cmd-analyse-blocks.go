package main

import (
	"context"
	"time"
)

type analyseBlocksCmd struct {
	backendOptions

	TenantID           string `arg:"" help:"tenant-id within the bucket"`
	MinCompactionLevel int    `help:"Min compaction level to analyse" default:"3"`
	MaxBlocks          int    `help:"Max number of blocks to analyse" default:"10"`
	NumAttr            int    `help:"Number of attributes to display" default:"15"`
}

func (cmd *analyseBlocksCmd) Run(ctx *globalOptions) error {
	r, _, c, err := loadBackend(&cmd.backendOptions, ctx)
	if err != nil {
		return err
	}

	// TODO: Parallelize this
	blocks, _, err := r.Blocks(context.Background(), cmd.TenantID)
	if err != nil {
		return err
	}

	processedBlocks := 0
	topSpanAttrs, topResourceAttrs := make(map[string]uint64), make(map[string]uint64)
	totalSpanBytes, totalResourceBytes := uint64(0), uint64(0)
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
			topSpanAttrs[k] += v
		}
		totalSpanBytes += blockSum.spanSummary.totalBytes

		for k, v := range blockSum.resourceSummary.attributes {
			topResourceAttrs[k] += v
		}
		totalResourceBytes += blockSum.resourceSummary.totalBytes

		processedBlocks++
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
	}).print(cmd.NumAttr)
}
