package main

import (
	"context"
	"errors"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/google/uuid"

	"github.com/grafana/tempo/tempodb/backend"
)

type analyseBlocksCmd struct {
	backendOptions

	Jsonnet             bool    `help:"output Jsonnet necessary for overrides"`
	Cli                 bool    `help:"Generate textual args for passing to parquet conversion command"`
	SimpleSummary       bool    `help:"Print only single line of top attributes" default:"false"`
	PrintFullSummary    bool    `help:"Print full summary of the analysed block" default:"true"`
	TenantID            string  `arg:"" help:"tenant-id within the bucket"`
	MinCompactionLevel  int     `help:"Min compaction level to analyse" default:"3"`
	MaxBlocks           int     `help:"Max number of blocks to analyse" default:"10"`
	NumAttr             int     `help:"Number of attributes to display" default:"20"`
	NumIntAttr          int     `help:"Number of integer attributes to display. If set to 0 then it will use the other parameter." default:"5"`
	StrPercentThreshold float64 `help:"Threshold for string attributes put in dedicated columns. Default 3% = 0.03." default:"0.03"`
	IntPercentThreshold float64 `help:"Threshold for integer attributes put in dedicated columns. Default 5% = 0.05" default:"0.05"`
	IncludeWellKnown    bool    `help:"Include well-known attributes in the analysis. These are attributes with fixed columns in some versions of parquet, like http.url." default:"false"`
	BlobThreshold       string  `help:"Convert column to blob when dictionary size reaches this value" default:"4MiB"`
	MaxStartTime        string  `help:"Oldest start time for a block to be processed. RFC3339 format '2006-01-02T15:04:05Z07:00'" default:""`
	MinStartTime        string  `help:"Newest start time for a block to be processed. RFC3339 format '2006-01-02T15:04:05Z07:00'" default:""`
}

func (cmd *analyseBlocksCmd) Run(ctx *globalOptions) error {
	blobBytes, err := humanize.ParseBytes(cmd.BlobThreshold)
	if err != nil {
		return err
	}

	r, _, _, err := loadBackend(&cmd.backendOptions, ctx)
	if err != nil {
		return err
	}

	// TODO: Parallelize this
	blocks, _, err := r.Blocks(context.Background(), cmd.TenantID)
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

	if cmd.StrPercentThreshold <= 0 || cmd.StrPercentThreshold >= 1 {
		return errors.New("str percent threshold must be between 0 and 1")
	}

	if cmd.IntPercentThreshold <= 0 || cmd.IntPercentThreshold >= 1 {
		return errors.New("int percent threshold must be between 0 and 1")
	}

	settings := heuristicSettings{
		NumStringAttr:       cmd.NumAttr,
		NumIntAttr:          cmd.NumIntAttr,
		BlobThresholdBytes:  blobBytes,
		StrThresholdPercent: cmd.StrPercentThreshold,
		IntThresholdPercent: cmd.IntPercentThreshold,
	}

	printSettings := printSettings{
		Simple:  cmd.SimpleSummary,
		Full:    cmd.PrintFullSummary,
		Jsonnet: cmd.Jsonnet,
		CliArgs: cmd.Cli,
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

	return totalSummary.print(settings, printSettings)
}
