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

	"github.com/dustin/go-humanize"
	"github.com/google/uuid"
	"github.com/parquet-go/parquet-go"
	"github.com/stoewer/parquet-cli/pkg/inspect"

	tempo_io "github.com/grafana/tempo/pkg/io"
	pq "github.com/grafana/tempo/pkg/parquetquery"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/vparquet2"
	"github.com/grafana/tempo/tempodb/encoding/vparquet3"
	"github.com/grafana/tempo/tempodb/encoding/vparquet4"
)

var (
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
	vparquet4SpanAttrs = []string{
		vparquet4.FieldSpanAttrVal,
	}
	vparquet4ResourceAttrs = []string{
		vparquet4.FieldResourceAttrVal,
	}
)

func spanPathsForVersion(v string) (string, []string) {
	switch v {
	case vparquet2.VersionString:
		return vparquet2.FieldSpanAttrKey, vparquet2SpanAttrs
	case vparquet3.VersionString:
		return vparquet3.FieldSpanAttrKey, vparquet3SpanAttrs
	case vparquet4.VersionString:
		return vparquet4.FieldSpanAttrKey, vparquet4SpanAttrs
	}
	return "", nil
}

func resourcePathsForVersion(v string) (string, []string) {
	switch v {
	case vparquet2.VersionString:
		return vparquet2.FieldResourceAttrKey, vparquet2ResourceAttrs
	case vparquet3.VersionString:
		return vparquet3.FieldResourceAttrKey, vparquet3ResourceAttrs
	case vparquet4.VersionString:
		return vparquet4.FieldResourceAttrKey, vparquet4ResourceAttrs
	}
	return "", nil
}

func dedicatedColPathForVersion(i int, scope backend.DedicatedColumnScope, v string) string {
	switch v {
	case vparquet3.VersionString:
		return vparquet3.DedicatedResourceColumnPaths[scope][backend.DedicatedColumnTypeString][i]
	case vparquet4.VersionString:
		return vparquet4.DedicatedResourceColumnPaths[scope][backend.DedicatedColumnTypeString][i]
	}
	return ""
}

type analyseBlockCmd struct {
	backendOptions

	TenantID         string `arg:"" help:"tenant-id within the bucket"`
	BlockID          string `arg:"" help:"block ID to list"`
	NumAttr          int    `help:"Number of attributes to display" default:"15"`
	GenerateJsonnet  bool   `help:"Generate overrides Jsonnet for dedicated columns"`
	SimpleSummary    bool   `help:"Print only single line of top attributes" default:"false"`
	PrintFullSummary bool   `help:"Print full summary of the analysed block" default:"true"`
}

func (cmd *analyseBlockCmd) Run(ctx *globalOptions) error {
	r, _, _, err := loadBackend(&cmd.backendOptions, ctx)
	if err != nil {
		return err
	}

	blockSum, err := processBlock(r, cmd.TenantID, cmd.BlockID, time.Time{}, time.Time{}, 0)
	if err != nil {
		if errors.Is(err, backend.ErrDoesNotExist) {
			return fmt.Errorf("unable to analyze block: block has no block.meta because it was compacted")
		}
		return err
	}

	if blockSum == nil {
		return errors.New("failed to process block")
	}

	return blockSum.print(cmd.NumAttr, cmd.GenerateJsonnet, cmd.SimpleSummary, cmd.PrintFullSummary)
}

func processBlock(r backend.Reader, tenantID, blockID string, maxStartTime, minStartTime time.Time, minCompactionLvl uint8) (*blockSummary, error) {
	id := uuid.MustParse(blockID)

	meta, err := r.BlockMeta(context.TODO(), id, tenantID)
	if err != nil {
		return nil, err
	}
	if meta.CompactionLevel < minCompactionLvl {
		return nil, nil
	}
	if !maxStartTime.IsZero() && meta.StartTime.After(maxStartTime) {
		// Block is newer than maxStartTime
		return nil, nil
	}
	if !minStartTime.IsZero() && meta.StartTime.Before(minStartTime) {
		// Block is older than minStartTime
		return nil, nil
	}

	var reader io.ReaderAt
	switch meta.Version {
	case vparquet2.VersionString:
		reader = vparquet2.NewBackendReaderAt(context.Background(), r, vparquet2.DataFileName, meta)
	case vparquet3.VersionString:
		reader = vparquet3.NewBackendReaderAt(context.Background(), r, vparquet3.DataFileName, meta)
	case vparquet4.VersionString:
		reader = vparquet4.NewBackendReaderAt(context.Background(), r, vparquet4.DataFileName, meta)
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

	// add up dedicated span attribute columns
	spanDedicatedSummary, err := aggregateDedicatedColumns(pf, backend.DedicatedColumnScopeSpan, meta)
	if err != nil {
		return nil, err
	}
	// merge dedicated with span attributes
	for k, v := range spanDedicatedSummary.attributes {
		spanAttrsSummary.attributes[k] = v
		spanAttrsSummary.dedicated[k] = struct{}{}
	}
	spanAttrsSummary.totalBytes += spanDedicatedSummary.totalBytes

	// Aggregate resource attributes
	resourceKey, resourceVals := resourcePathsForVersion(meta.Version)
	resourceAttrsSummary, err := aggregateAttributes(pf, resourceKey, resourceVals)
	if err != nil {
		return nil, err
	}

	// add up dedicated resource attribute columns
	resourceDedicatedSummary, err := aggregateDedicatedColumns(pf, backend.DedicatedColumnScopeResource, meta)
	if err != nil {
		return nil, err
	}
	// merge dedicated with span attributes
	for k, v := range resourceDedicatedSummary.attributes {
		resourceAttrsSummary.attributes[k] = v
		resourceAttrsSummary.dedicated[k] = struct{}{}
	}
	resourceAttrsSummary.totalBytes += spanDedicatedSummary.totalBytes

	return &blockSummary{
		spanSummary:     spanAttrsSummary,
		resourceSummary: resourceAttrsSummary,
	}, nil
}

type blockSummary struct {
	spanSummary, resourceSummary genericAttrSummary
}

func (s *blockSummary) print(maxAttr int, generateJsonnet, simpleSummary, printFullSummary bool) error {
	if printFullSummary {
		if err := printSummary("span", maxAttr, s.spanSummary, false); err != nil {
			return err
		}

		if err := printSummary("resource", maxAttr, s.resourceSummary, false); err != nil {
			return err
		}
	}

	if simpleSummary {
		if err := printSummary("span", maxAttr, s.spanSummary, true); err != nil {
			return err
		}

		if err := printSummary("resource", maxAttr, s.resourceSummary, true); err != nil {
			return err
		}
	}

	if generateJsonnet {
		printDedicatedColumnOverridesJsonnet(s.spanSummary, s.resourceSummary)
	}

	return nil
}

type genericAttrSummary struct {
	totalBytes uint64
	attributes map[string]uint64 // key: attribute name, value: total bytes
	dedicated  map[string]struct{}
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
		dedicated:  make(map[string]struct{}),
	}, nil
}

func aggregateDedicatedColumns(pf *parquet.File, scope backend.DedicatedColumnScope, meta *backend.BlockMeta) (genericAttrSummary, error) {
	attrMap := make(map[string]uint64)
	totalBytes := uint64(0)

	i := 0
	for _, dedColumn := range meta.DedicatedColumns {
		if dedColumn.Scope != scope {
			continue
		}

		path := dedicatedColPathForVersion(i, scope, meta.Version)
		sz, err := aggregateColumn(pf, path)
		if err != nil {
			return genericAttrSummary{}, err
		}
		i++

		attrMap[dedColumn.Name] = sz
		totalBytes += sz
	}

	return genericAttrSummary{
		totalBytes: totalBytes,
		attributes: attrMap,
	}, nil
}

func aggregateColumn(pf *parquet.File, colName string) (uint64, error) {
	idx, _ := pq.GetColumnIndexByPath(pf, colName)
	calc, err := inspect.NewRowStatCalculator(pf, inspect.RowStatOptions{
		Columns: []int{idx},
	})
	if err != nil {
		return 0, err
	}

	totalBytes := uint64(0)
	for {
		row, err := calc.NextRow()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return 0, err
		}

		cells := row.Cells()

		bytes := uint64(cells[1].(int))
		totalBytes += bytes
	}

	return totalBytes, nil
}

func printSummary(scope string, max int, summary genericAttrSummary, simple bool) error {
	// TODO: Support more output formats
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	if max > len(summary.attributes) {
		max = len(summary.attributes)
	}

	fmt.Println("")
	attrList := topN(max, summary.attributes)
	if simple {
		fmt.Printf("%s attributes: ", scope)
		for _, a := range attrList {
			fmt.Printf("\"%s\", ", a.name)
		}
		fmt.Println("")
	} else {
		fmt.Printf("Top %d %s attributes by size\n", max, scope)
		for _, a := range attrList {

			name := a.name
			if _, ok := summary.dedicated[a.name]; ok {
				name = a.name + " (dedicated)"
			}

			percentage := float64(a.bytes) / float64(summary.totalBytes) * 100
			_, err := fmt.Fprintf(w, "name: %s\t size: %s\t (%s%%)\n", name, humanize.Bytes(a.bytes), strconv.FormatFloat(percentage, 'f', 2, 64))
			if err != nil {
				return err
			}
		}
	}

	return w.Flush()
}

func printDedicatedColumnOverridesJsonnet(spanSummary, resourceSummary genericAttrSummary) {
	fmt.Println("")
	fmt.Printf("parquet_dedicated_columns: [\n")

	// span attributes first
	spanAttrList := topN(10, spanSummary.attributes)
	for _, a := range spanAttrList {
		fmt.Printf(" { scope: 'span', name: '%s', type: 'string' },\n", a.name)
	}

	// span attributes first
	resourceAttrList := topN(10, resourceSummary.attributes)
	for _, a := range resourceAttrList {
		fmt.Printf(" { scope: 'resource', name: '%s', type: 'string' },\n", a.name)
	}
	fmt.Printf("], \n")
	fmt.Println("")
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
