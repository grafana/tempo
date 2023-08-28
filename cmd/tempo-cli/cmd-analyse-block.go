package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"text/tabwriter"
	"time"

	tempo_io "github.com/grafana/tempo/pkg/io"
	"github.com/parquet-go/parquet-go"

	pq "github.com/grafana/tempo/pkg/parquetquery"
	"github.com/stoewer/parquet-cli/pkg/inspect"

	"github.com/grafana/tempo/tempodb/encoding/vparquet"
	"github.com/grafana/tempo/tempodb/encoding/vparquet2"
	"github.com/grafana/tempo/tempodb/encoding/vparquet3"

	"github.com/dustin/go-humanize"
	"github.com/google/uuid"
	"github.com/grafana/tempo/tempodb/backend"
)

var (
	vparquetSpanAttrs = []string{
		vparquet.FieldSpanAttrVal,
		// TODO: Dedicated columns only support 'string' values.  We need to add support for other types
		// vparquet.FieldSpanAttrValInt,
		// vparquet.FieldSpanAttrValDouble,
		// vparquet.FieldSpanAttrValBool,
	}
	vparquetResourceAttrs = []string{
		vparquet.FieldResourceAttrVal,
		// TODO: Dedicated columns only support 'string' values.  We need to add support for other types
		// vparquet.FieldResourceAttrValInt,
		// vparquet.FieldResourceAttrValDouble,
		// vparquet.FieldResourceAttrValBool,
	}
	vparquet2SpanAttrs = []string{
		vparquet2.FieldSpanAttrVal,
	}
	vparquet2ResourceAttrs = []string{
		vparquet2.FieldResourceAttrVal,
	}
	vparquet3SpanAttrs = []string{
		vparquet3.FieldSpanAttrVal,
	}
	vparquet3ResourceAttrs = []string{
		vparquet3.FieldResourceAttrVal,
	}
)

func spanPathsForVersion(v string) (string, []string) {
	switch v {
	case vparquet.VersionString:
		return vparquet.FieldSpanAttrKey, vparquetSpanAttrs
	case vparquet2.VersionString:
		return vparquet2.FieldSpanAttrKey, vparquet2SpanAttrs
	case vparquet3.VersionString:
		return vparquet3.FieldSpanAttrKey, vparquet3SpanAttrs
	}
	return "", nil
}

func resourcePathsForVersion(v string) (string, []string) {
	switch v {
	case vparquet.VersionString:
		return vparquet.FieldResourceAttrKey, vparquetResourceAttrs
	case vparquet2.VersionString:
		return vparquet2.FieldResourceAttrKey, vparquet2ResourceAttrs
	case vparquet3.VersionString:
		return vparquet3.FieldResourceAttrKey, vparquet3ResourceAttrs
	}
	return "", nil
}

type analyseBlockCmd struct {
	backendOptions

	TenantID string `arg:"" help:"tenant-id within the bucket"`
	BlockID  string `arg:"" help:"block ID to list"`
	NumAttr  int    `help:"Number of attributes to display" default:"15"`
}

func (cmd *analyseBlockCmd) Run(ctx *globalOptions) error {
	r, _, c, err := loadBackend(&cmd.backendOptions, ctx)
	if err != nil {
		return err
	}

	blockSum, err := processBlock(r, c, cmd.TenantID, cmd.BlockID, time.Hour, 0)
	if err != nil {
		return err
	}

	if blockSum == nil {
		return errors.New("failed to process block")
	}

	return blockSum.print(cmd.NumAttr)
}

func processBlock(r backend.Reader, _ backend.Compactor, tenantID, blockID string, _ time.Duration, minCompactionLvl uint8) (*blockSummary, error) {
	id := uuid.MustParse(blockID)

	meta, err := r.BlockMeta(context.TODO(), id, tenantID)
	if err != nil && !errors.Is(err, backend.ErrDoesNotExist) {
		return nil, err
	}

	if meta == nil {
		fmt.Println("Unable to load any meta for block", blockID)
		return nil, nil
	}

	if meta.CompactionLevel < minCompactionLvl {
		return nil, nil
	}

	var reader io.ReaderAt
	switch meta.Version {
	case vparquet.VersionString:
		reader = vparquet.NewBackendReaderAt(context.Background(), r, vparquet.DataFileName, meta.BlockID, meta.TenantID)
	case vparquet2.VersionString:
		reader = vparquet2.NewBackendReaderAt(context.Background(), r, vparquet2.DataFileName, meta.BlockID, meta.TenantID)
	case vparquet3.VersionString:
		reader = vparquet3.NewBackendReaderAt(context.Background(), r, vparquet3.DataFileName, meta.BlockID, meta.TenantID)
	default:
		fmt.Println("Unsupported block version:", meta.Version)
		return nil, nil
	}

	br := tempo_io.NewBufferedReaderAt(reader, int64(meta.Size), 2*1024*1024, 64) // 128 MB memory buffering

	pf, err := parquet.OpenFile(br, int64(meta.Size), parquet.SkipBloomFilters(true), parquet.SkipPageIndex(true))
	if err != nil {
		return nil, err
	}

	fmt.Println("Scanning block contents.  Press CRTL+C to quit ...")

	// Aggregate span attributes
	spanKey, spanVals := spanPathsForVersion(meta.Version)
	spanAttrsSummary, err := aggregateAttributes(pf, spanKey, spanVals)
	if err != nil {
		return nil, err
	}

	// Aggregate resource attributes
	resourceKey, resourceVals := resourcePathsForVersion(meta.Version)
	resourceAttrsSummary, err := aggregateAttributes(pf, resourceKey, resourceVals)
	if err != nil {
		return nil, err
	}

	return &blockSummary{
		spanSummary:     spanAttrsSummary,
		resourceSummary: resourceAttrsSummary,
	}, nil
}

type blockSummary struct {
	spanSummary, resourceSummary genericAttrSummary
}

func (s *blockSummary) print(maxAttr int) error {
	if err := printSummary("span", maxAttr, s.spanSummary); err != nil {
		return err
	}
	return printSummary("resource", maxAttr, s.resourceSummary)
}

type genericAttrSummary struct {
	totalBytes uint64
	attributes map[string]uint64 // key: attribute name, value: total bytes
}

type attribute struct {
	name  string
	bytes uint64
}

func aggregateAttributes(pf *parquet.File, keyPath string, valuePaths []string) (genericAttrSummary, error) {
	keyIdx, _ := pq.GetColumnIndexByPath(pf, keyPath)
	valueIdxs := make([]int, 0, len(valuePaths))
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
		return genericAttrSummary{}, err
	}

	attrMap := make(map[string]uint64)
	totalBytes := uint64(0)

	for {
		row, err := rowStats.NextRow()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return genericAttrSummary{}, err
		}

		cells := row.Cells()

		name := cells[0].(string)
		bytes := uint64(cells[1].(int))
		attrMap[name] = bytes
		totalBytes += bytes
	}

	return genericAttrSummary{
		totalBytes: totalBytes,
		attributes: attrMap,
	}, nil
}

func printSummary(scope string, max int, summary genericAttrSummary) error {
	// TODO: Support more output formats
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	if max > len(summary.attributes) {
		max = len(summary.attributes)
	}

	fmt.Printf("Top %d %s attributes by size\n", max, scope)
	attrList := topN(max, summary.attributes)
	for _, a := range attrList {
		percentage := float64(a.bytes) / float64(summary.totalBytes) * 100
		_, err := fmt.Fprintf(w, "name: %s\t size: %s\t (%s%%)\n", a.name, humanize.Bytes(a.bytes), strconv.FormatFloat(percentage, 'f', 2, 64))
		if err != nil {
			return err
		}
	}

	return w.Flush()
}

func topN(n int, attrs map[string]uint64) []attribute {
	top := make([]attribute, 0, len(attrs))
	for name, bytes := range attrs {
		top = append(top, attribute{name, bytes})
	}
	sort.Slice(top, func(i, j int) bool {
		return top[i].bytes > top[j].bytes
	})
	if len(top) > n {
		top = top[:n]
	}
	return top
}
