package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/dustin/go-humanize"
	"github.com/google/uuid"

	"github.com/grafana/tempo/tempodb/backend"
)

type suggestColumnsCmd struct {
	backendOptions
	outputOptions

	TenantID string `arg:"" help:"tenant-id within the bucket"`

	BlockID             string  `help:"specific block ID to analyse"`
	MinCompactionLevel  int     `help:"Min compaction level to analyse" default:"3"`
	MaxBlocks           int     `help:"Max number of blocks to analyse" default:"10"`
	NumAttr             int     `help:"Number of attributes to display" default:"20"`
	NumIntAttr          int     `help:"Number of integer attributes to display. If set to 0 then it will use the NumAttr." default:"5"`
	StrPercentThreshold float64 `help:"Threshold for string attributes put in dedicated columns. Default 3% = 0.03." default:"0.03"`
	IntPercentThreshold float64 `help:"Threshold for integer attributes put in dedicated columns. Default 5% = 0.05" default:"0.05"`
	IncludeWellKnown    bool    `help:"Include well-known attributes in the analysis. These are attributes with fixed columns in some versions of parquet, like http.url." default:"true"`
	BlobThreshold       string  `help:"Convert column to blob when dictionary size reaches this value" default:"4MiB"`
	MaxStartTime        string  `help:"Oldest start time for a block to be processed. RFC3339 format '2006-01-02T15:04:05Z07:00'" default:""`
	MinStartTime        string  `help:"Newest start time for a block to be processed. RFC3339 format '2006-01-02T15:04:05Z07:00'" default:""`
}

func (cmd *suggestColumnsCmd) Run(ctx *globalOptions) error {
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

		return printDedicatedColumns(blockSum.ToDedicatedColumns(settings), cmd.Format, cmd.Out)
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

	return printDedicatedColumns(totalSummary.ToDedicatedColumns(settings), cmd.Format, cmd.Out)
}

func printDedicatedColumns(dedicatedCols backend.DedicatedColumns, format string, outPath string) error {
	// json prints out JSON format but using backend.DedicatedColumn struct which the keys are s, n, t, o
	// jsonnet prints out JSONNET format but what user would put in Jsonnet overrides file to generate yaml
	switch format {
	case "jsonnet":
		printDedicatedColumnSuggestionsJsonnet(dedicatedCols, outPath)
	case "yaml":
		printDedicatedColumnSuggestionsYaml(dedicatedCols, outPath)
	default:
		return errors.New("unknown format: " + format)
	}
	return nil
}

type DedicatedColumnYAMLReadableKeys struct {
	// The Scope of the attribute
	Scope string `yaml:"scope" json:"scope,omitempty"`
	// The Name of the attribute stored in the dedicated column
	Name string `yaml:"name" json:"name"`
	// The Type of attribute value
	Type string `yaml:"type" json:"type,omitempty"`
	// The Options applied to the dedicated attribute column
	Options []string `yaml:"options" json:"options,omitempty"`
}

func backendDedicatedColumnsToYAMLReadableKeys(dedCol backend.DedicatedColumns) []DedicatedColumnYAMLReadableKeys {
	var printDedCols []DedicatedColumnYAMLReadableKeys

	for _, c := range dedCol {
		dedColPrint := DedicatedColumnYAMLReadableKeys{
			Scope:   string(c.Scope),
			Name:    c.Name,
			Type:    string(c.Type),
			Options: []string{},
		}
		for _, o := range c.Options {
			dedColPrint.Options = append(dedColPrint.Options, string(o))
		}
		printDedCols = append(printDedCols, dedColPrint)
	}
	return printDedCols
}

func printDedicatedColumnSuggestionsJsonnet(dedCol backend.DedicatedColumns, outPath string) {
	printDedCols := backendDedicatedColumnsToYAMLReadableKeys(dedCol)

	outBytes, err := json.MarshalIndent(printDedCols, "", "  ")
	if err != nil {
		fmt.Println("error marshaling dedicated column suggestions to json:", err)
		return
	}
	outBytes = append(outBytes, byte('\n'))

	if outPath != "" {
		err = os.WriteFile(outPath, outBytes, 0o600)
		if err != nil {
			fmt.Println("error writing dedicated column suggestions to file:", err)
			return
		}
		return
	}

	fmt.Println(string(outBytes))
}

func printDedicatedColumnSuggestionsYaml(dedCol backend.DedicatedColumns, outPath string) {
	outBytes, err := yaml.Marshal(dedCol)
	if err != nil {
		fmt.Println("error marshaling dedicated column suggestions to yaml:", err)
		return
	}

	if outPath != "" {
		err = os.WriteFile(outPath, outBytes, 0o600)
		if err != nil {
			fmt.Println("error writing dedicated column suggestions to file:", err)
			return
		}
		return
	}

	fmt.Println(string(outBytes))
}
