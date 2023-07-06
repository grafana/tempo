package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/stoewer/parquet-cli/pkg/output"

	"github.com/segmentio/parquet-go"

	pq "github.com/grafana/tempo/pkg/parquetquery"
	"github.com/stoewer/parquet-cli/pkg/inspect"

	"github.com/grafana/tempo/tempodb/encoding/vparquet2"

	"github.com/dustin/go-humanize"
	"github.com/google/uuid"
	tempodb_backend "github.com/grafana/tempo/tempodb/backend"
)

var (
	spanAttrValPaths = []string{
		vparquet2.FieldSpanAttrVal,
		vparquet2.FieldSpanAttrValInt,
		vparquet2.FieldSpanAttrValDouble,
		vparquet2.FieldSpanAttrValBool,
	}
	resourceAttrValPaths = []string{
		vparquet2.FieldResourceAttrVal,
		vparquet2.FieldResourceAttrValInt,
		vparquet2.FieldResourceAttrValDouble,
		vparquet2.FieldResourceAttrValBool,
	}
)

type analyseBlockCmd struct {
	backendOptions

	BlockID  string `arg:"" help:"block ID to list"`
	TenantID string `arg:"" help:"tenant-id within the bucket"`
}

func (cmd *analyseBlockCmd) Run(ctx *globalOptions) error {
	r, _, c, err := loadBackend(&cmd.backendOptions, ctx)
	if err != nil {
		return err
	}

	return processBlock(r, c, cmd.TenantID, time.Hour, cmd.BlockID)
}

func processBlock(r tempodb_backend.Reader, c tempodb_backend.Compactor, tenantID string, windowRange time.Duration, blockID string) error {
	id := uuid.MustParse(blockID)

	meta, err := r.BlockMeta(context.TODO(), id, tenantID)
	if err != nil && err != tempodb_backend.ErrDoesNotExist {
		return err
	}

	compactedMeta, err := c.CompactedBlockMeta(id, tenantID)
	if err != nil && err != tempodb_backend.ErrDoesNotExist {
		return err
	}

	if meta == nil && compactedMeta == nil {
		fmt.Println("Unable to load any meta for block", blockID)
		return nil
	}

	unifiedMeta := getMeta(meta, compactedMeta, windowRange)

	fmt.Println("ID            : ", unifiedMeta.BlockID)
	fmt.Println("Version       : ", unifiedMeta.Version)
	fmt.Println("Total Objects : ", unifiedMeta.TotalObjects)
	fmt.Println("Data Size     : ", humanize.Bytes(unifiedMeta.Size))
	fmt.Println("Encoding      : ", unifiedMeta.Encoding)
	fmt.Println("Level         : ", unifiedMeta.CompactionLevel)
	fmt.Println("Window        : ", unifiedMeta.window)
	fmt.Println("Start         : ", unifiedMeta.StartTime)
	fmt.Println("End           : ", unifiedMeta.EndTime)
	fmt.Println("Duration      : ", fmt.Sprint(unifiedMeta.EndTime.Sub(unifiedMeta.StartTime).Round(time.Second)))
	fmt.Println("Age           : ", fmt.Sprint(time.Since(unifiedMeta.EndTime).Round(time.Second)))

	if unifiedMeta.Version != vparquet2.VersionString {
		return fmt.Errorf("cannot scan block contents. unsupported block version: %s", unifiedMeta.Version)
	}

	fmt.Println("Scanning block contents.  Press CRTL+C to quit ...")

	block := vparquet2.NewBackendBlock(&unifiedMeta.BlockMeta, r)

	pf, _, err := block.Open(context.Background())
	if err != nil {
		return err
	}

	// Aggregate span attributes
	fmt.Println("Aggregating span attributes stats")
	if err := aggregateAttributes(pf, vparquet2.FieldSpanAttrKey, spanAttrValPaths); err != nil {
		return err
	}

	// Aggregate resource attributes
	fmt.Println("Aggregating resource attributes stats")
	return aggregateAttributes(pf, vparquet2.FieldResourceAttrKey, resourceAttrValPaths)
}

func aggregateAttributes(pf *parquet.File, keyPath string, valuePaths []string) error {
	keyIdx, _ := pq.GetColumnIndexByPath(pf, keyPath)
	var valueIdxs []int
	for _, v := range valuePaths {
		idx, _ := pq.GetColumnIndexByPath(pf, v)
		valueIdxs = append(valueIdxs, idx)
	}

	opts := inspect.AggregateOptions{
		GroupByColumn: keyIdx,
		Columns:       valueIdxs,
	}
	rowStats, err := inspect.NewAggregateCalculator(pf, opts)
	if err != nil {
		return err
	}

	// TODO: Support more output formats
	return output.PrintTable(os.Stdout, output.FormatTab, rowStats)
}
