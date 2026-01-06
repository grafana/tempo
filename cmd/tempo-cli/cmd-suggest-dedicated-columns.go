package main

import (
	"context"
	"errors"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/google/uuid"

	"github.com/grafana/tempo/tempodb/backend"
)

type suggestDedicatedColumnsCmd struct {
	backendOptions

	TenantID string `arg:"" help:"tenant-id within the bucket"`

	Format              string  `help:"Output format (json/jsonnet/yaml)" enum:"json,jsonnet,yaml" default:"jsonnet"`
	BlockID             string  `help:"specific block ID to analyse"`
	MinCompactionLevel  int     `help:"Min compaction level to analyse" default:"3"`
	MaxBlocks           int     `help:"Max number of blocks to analyse" default:"10"`
	NumAttr             int     `help:"Number of attributes to display" default:"15"`
	NumIntAttr          int     `help:"Number of integer attributes to display. If set to 0 then it will use the NumAttr." default:"0"`
	IntPercentThreshold float64 `help:"Threshold for integer attributes put in dedicated columns. Default 5% = 0.05" default:"0.05"`
	IncludeWellKnown    bool    `help:"Include well-known attributes in the analysis. These are attributes with fixed columns in some versions of parquet, like http.url." default:"true"`
	BlobThreshold       string  `help:"Convert column to blob when dictionary size reaches this value" default:"4MiB"`
	MaxStartTime        string  `help:"Oldest start time for a block to be processed. RFC3339 format '2006-01-02T15:04:05Z07:00'" default:""`
	MinStartTime        string  `help:"Newest start time for a block to be processed. RFC3339 format '2006-01-02T15:04:05Z07:00'" default:""`
	Out                 string  `help:"(internal) file to write output to, instead of stdout" default:""`
}

func (cmd *suggestDedicatedColumnsCmd) Run(ctx *globalOptions) error {
	blobBytes, err := humanize.ParseBytes(cmd.BlobThreshold)
	if err != nil {
		return err
	}

	r, _, _, err := loadBackend(&cmd.backendOptions, ctx)
	if err != nil {
		return err
	}

	var (
		processedBlocks            = map[uuid.UUID]struct{}{}
		totalSummary               blockSummary
		maxStartTime, minStartTime time.Time
	)

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

	if cmd.NumIntAttr == 0 {
		cmd.NumIntAttr = cmd.NumAttr
	}

	if cmd.IntPercentThreshold <= 0 || cmd.IntPercentThreshold >= 1 {
		return errors.New("int percent threshold must be between 0 and 1")
	}

	settings := heuristicSettings{
		NumStringAttr:       cmd.NumAttr,
		NumIntAttr:          cmd.NumIntAttr,
		BlobThresholdBytes:  blobBytes,
		IntThresholdPercent: cmd.IntPercentThreshold,
	}

	// if just one block
	if cmd.BlockID != "" {
		blockSum, err := processBlock(r, cmd.TenantID, cmd.BlockID, cmd.IncludeWellKnown, time.Time{}, time.Time{}, 0)
		if err != nil {
			if errors.Is(err, backend.ErrDoesNotExist) {
				return errors.New("unable to analyze block: block has no block.meta because it was compacted")
			}
			return err
		}

		if blockSum == nil {
			return errors.New("failed to process block")
		}

		if cmd.NumIntAttr == 0 {
			cmd.NumIntAttr = cmd.NumAttr
		}

		if cmd.IntPercentThreshold <= 0 || cmd.IntPercentThreshold >= 1 {
			return errors.New("int percent threshold must be between 0 and 1")
		}

		return blockSum.printDedicatedColumns(settings, cmd.Format, cmd.Out)
	}

	// TODO: Parallelize this
	blocks, _, err := r.Blocks(context.Background(), cmd.TenantID)
	if err != nil {
		return err
	}

	for i := 0; i < len(blocks) && len(processedBlocks) < cmd.MaxBlocks; i++ {
		block := blocks[i]
		if _, ok := processedBlocks[block]; ok {
			continue
		}

		blockSum, err := processBlock(r, cmd.TenantID, block.String(), cmd.IncludeWellKnown, maxStartTime, minStartTime, uint32(cmd.MinCompactionLevel))
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

		totalSummary.add(*blockSum)

		processedBlocks[block] = struct{}{}
	}

	return totalSummary.printDedicatedColumns(settings, cmd.Format, cmd.Out)
}
