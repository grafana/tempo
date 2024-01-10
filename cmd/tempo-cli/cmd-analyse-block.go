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

func dedicatedColPathForVersion(i int, scope backend.DedicatedColumnScope, v string) string {
	switch v {
	case vparquet3.VersionString:
		return vparquet3.DedicatedResourceColumnPaths[scope][backend.DedicatedColumnTypeString][i]
	}
	return ""
}

type analyseBlockCmd struct {
	backendOptions

	TenantID        string `arg:"" help:"tenant-id within the bucket"`
	BlockID         string `arg:"" help:"block ID to list"`
	NumAttr         int    `help:"Number of attributes to display" default:"15"`
	GenerateJsonnet bool   `help:"Generate overrides Jsonnet for dedicated columns"`
}

func (cmd *analyseBlockCmd) Run(ctx *globalOptions) error {
	r, _, c, err := loadBackend(&cmd.backendOptions, ctx)
	if err != nil {
		return err
	}

	blockSum, err := processBlock(r, c, cmd.TenantID, cmd.BlockID, time.Hour, 0)
	if err != nil {
		if errors.Is(err, backend.ErrDoesNotExist) {
			return fmt.Errorf("unable to analyze block: block has no block.meta because it was compacted")
		}
		return err
	}

	if blockSum == nil {
		return errors.New("failed to process block")
	}

	return blockSum.print(cmd.NumAttr, cmd.GenerateJsonnet)
}

func processBlock(r backend.Reader, _ backend.Compactor, tenantID, blockID string, _ time.Duration, minCompactionLvl uint8) (*blockSummary, error) {
	id := uuid.MustParse(blockID)

	meta, err := r.BlockMeta(context.TODO(), id, tenantID)
	if err != nil {
		return nil, err
	}
	if meta.CompactionLevel < minCompactionLvl {
		return nil, nil
	}

	var reader io.ReaderAt
	switch meta.Version {
	case vparquet.VersionString:
		reader = vparquet.NewBackendReaderAt(context.Background(), r, vparquet.DataFileName, meta)
	case vparquet2.VersionString:
		reader = vparquet2.NewBackendReaderAt(context.Background(), r, vparquet2.DataFileName, meta)
	case vparquet3.VersionString:
		reader = vparquet3.NewBackendReaderAt(context.Background(), r, vparquet3.DataFileName, meta)
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

func (s *blockSummary) print(maxAttr int, generateJsonnet bool) error {
	if err := printSummary("span", maxAttr, s.spanSummary); err != nil {
		return err
	}

	if err := printSummary("resource", maxAttr, s.resourceSummary); err != nil {
		return err
	}

	if generateJsonnet {
		printDedicatedColumnOverridesJsonnet(s.spanSummary, s.resourceSummary)
	}

	return nil
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

		attrMap["dedicated: "+dedColumn.Name] = sz
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
