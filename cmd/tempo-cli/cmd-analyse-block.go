package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"text/tabwriter"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/google/uuid"
	"github.com/parquet-go/parquet-go"

	tempo_io "github.com/grafana/tempo/pkg/io"
	pq "github.com/grafana/tempo/pkg/parquetquery"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/vparquet2"
	"github.com/grafana/tempo/tempodb/encoding/vparquet3"
	"github.com/grafana/tempo/tempodb/encoding/vparquet4"
)

type attributePaths struct {
	span  scopeAttributePath
	res   scopeAttributePath
	event scopeAttributePath
}

type scopeAttributePath struct {
	defLevel           int
	keyPath            string
	valPath            string
	isArrayPath        string
	dedicatedColScope  backend.DedicatedColumnScope
	dedicatedColsPaths []string
}

func pathsForVersion(v string) attributePaths {
	switch v {
	case vparquet2.VersionString:
		return attributePaths{
			span: scopeAttributePath{
				defLevel: vparquet2.DefinitionLevelResourceSpansILSSpanAttrs,
				keyPath:  vparquet2.FieldSpanAttrKey,
				valPath:  vparquet2.FieldSpanAttrVal,
			},
			res: scopeAttributePath{
				defLevel: vparquet2.DefinitionLevelResourceAttrs,
				keyPath:  vparquet2.FieldResourceAttrKey,
				valPath:  vparquet2.FieldResourceAttrVal,
			},
		}
	case vparquet3.VersionString:
		return attributePaths{
			span: scopeAttributePath{
				defLevel:           vparquet3.DefinitionLevelResourceSpansILSSpanAttrs,
				keyPath:            vparquet3.FieldSpanAttrKey,
				valPath:            vparquet3.FieldSpanAttrVal,
				dedicatedColScope:  backend.DedicatedColumnScopeSpan,
				dedicatedColsPaths: vparquet3.DedicatedResourceColumnPaths[backend.DedicatedColumnScopeSpan][backend.DedicatedColumnTypeString],
			},
			res: scopeAttributePath{
				defLevel:           vparquet3.DefinitionLevelResourceAttrs,
				keyPath:            vparquet3.FieldResourceAttrKey,
				valPath:            vparquet3.FieldResourceAttrVal,
				dedicatedColScope:  backend.DedicatedColumnScopeResource,
				dedicatedColsPaths: vparquet3.DedicatedResourceColumnPaths[backend.DedicatedColumnScopeResource][backend.DedicatedColumnTypeString],
			},
		}
	case vparquet4.VersionString:
		return attributePaths{
			span: scopeAttributePath{
				defLevel:           vparquet4.DefinitionLevelResourceSpansILSSpanAttrs,
				keyPath:            vparquet4.FieldSpanAttrKey,
				valPath:            vparquet4.FieldSpanAttrVal,
				isArrayPath:        vparquet4.FieldSpanAttrIsArray,
				dedicatedColScope:  backend.DedicatedColumnScopeSpan,
				dedicatedColsPaths: vparquet4.DedicatedResourceColumnPaths[backend.DedicatedColumnScopeSpan][backend.DedicatedColumnTypeString],
			},
			res: scopeAttributePath{
				defLevel:           vparquet4.DefinitionLevelResourceAttrs,
				keyPath:            vparquet4.FieldResourceAttrKey,
				valPath:            vparquet4.FieldResourceAttrVal,
				isArrayPath:        vparquet4.FieldResourceAttrIsArray,
				dedicatedColScope:  backend.DedicatedColumnScopeResource,
				dedicatedColsPaths: vparquet4.DedicatedResourceColumnPaths[backend.DedicatedColumnScopeResource][backend.DedicatedColumnTypeString],
			},
			event: scopeAttributePath{
				defLevel:    vparquet4.DefinitionLevelResourceSpansILSSpanEventAttrs,
				keyPath:     vparquet4.FieldEventAttrKey,
				valPath:     vparquet4.FieldEventAttrVal,
				isArrayPath: vparquet4.FieldEventAttrIsArray,
			},
		}
	default:
		panic("unsupported version")
	}
}

type analyseBlockCmd struct {
	backendOptions

	TenantID         string `arg:"" help:"tenant-id within the bucket"`
	BlockID          string `arg:"" help:"block ID to list"`
	NumAttr          int    `help:"Number of attributes to display" default:"15"`
	GenerateJsonnet  bool   `help:"Generate overrides Jsonnet for dedicated columns"`
	GenerateCliArgs  bool   `help:"Generate textual args for passing to parquet conversion command"`
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

	return blockSum.print(cmd.NumAttr, cmd.GenerateJsonnet, cmd.SimpleSummary, cmd.PrintFullSummary, cmd.GenerateCliArgs)
}

func processBlock(r backend.Reader, tenantID, blockID string, maxStartTime, minStartTime time.Time, minCompactionLvl uint32) (*blockSummary, error) {
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

	br := tempo_io.NewBufferedReaderAt(reader, int64(meta.Size_), 2*1024*1024, 64) // 128 MB memory buffering

	pf, err := parquet.OpenFile(br, int64(meta.Size_), parquet.SkipBloomFilters(true), parquet.SkipPageIndex(true))
	if err != nil {
		return nil, err
	}

	fmt.Println("Scanning block contents.  Press CRTL+C to quit ...")

	paths := pathsForVersion(meta.Version)

	spanSummary, err := aggregateScope(pf, meta, paths.span)
	if err != nil {
		return nil, err
	}

	resSummary, err := aggregateScope(pf, meta, paths.res)
	if err != nil {
		return nil, err
	}

	eventSummary, err := aggregateScope(pf, meta, paths.event)
	if err != nil {
		return nil, err
	}

	return &blockSummary{
		spanSummary:     spanSummary,
		resourceSummary: resSummary,
		eventSummary:    eventSummary,
	}, nil
}

func aggregateScope(pf *parquet.File, meta *backend.BlockMeta, paths scopeAttributePath) (genericAttrSummary, error) {
	res, err := aggregateAttributes(pf, paths.defLevel, paths.keyPath, paths.valPath, paths.isArrayPath)
	if err != nil {
		return res, err
	}

	if len(paths.dedicatedColsPaths) > 0 {
		dedicatedData, err := aggregateDedicatedColumns(pf, paths.dedicatedColScope, meta, paths.dedicatedColsPaths)
		if err != nil {
			return res, err
		}
		// merge dedicated with span attributes
		res.dedicated = make(map[string]struct{}, len(dedicatedData.attributes))
		for k, v := range dedicatedData.attributes {
			res.attributes[k] = v
			res.dedicated[k] = struct{}{}
			res.cardinality[k] = dedicatedData.cardinality[k]
		}
		res.totalBytes += dedicatedData.totalBytes
	}

	return res, nil
}

type blockSummary struct {
	spanSummary, resourceSummary, eventSummary genericAttrSummary
}

func (s *blockSummary) print(maxAttr int, generateJsonnet, simpleSummary, printFullSummary, generateCliArgs bool) error {
	if printFullSummary {
		if err := printSummary("span", maxAttr, s.spanSummary, false); err != nil {
			return err
		}

		if err := printSummary("resource", maxAttr, s.resourceSummary, false); err != nil {
			return err
		}

		if err := printSummary("event", maxAttr, s.eventSummary, false); err != nil {
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

	if generateCliArgs {
		printCliArgs(s, maxAttr)
	}

	return nil
}

type genericAttrSummary struct {
	totalBytes      uint64
	attributes      map[string]uint64 // key: attribute name, value: total bytes
	arrayAttributes map[string]uint64 // key: attribute name, value: total bytes
	dedicated       map[string]struct{}

	cardinality map[string]map[string]uint64
}

type attribute struct {
	name  string
	bytes uint64
}

type makeIterFn func(columnName string, predicate pq.Predicate, selectAs string) pq.Iterator

func makeIterFunc(ctx context.Context, pf *parquet.File) makeIterFn {
	return func(name string, predicate pq.Predicate, selectAs string) pq.Iterator {
		index, _, maxDef := pq.GetColumnIndexByPath(pf, name)
		if index == -1 {
			panic("column not found in parquet file:" + name)
		}

		opts := []pq.SyncIteratorOpt{
			pq.SyncIteratorOptColumnName(name),
			pq.SyncIteratorOptPredicate(predicate),
			pq.SyncIteratorOptSelectAs(selectAs),
			pq.SyncIteratorOptMaxDefinitionLevel(maxDef),
		}

		return pq.NewSyncIterator(ctx, pf.RowGroups(), index, opts...)
	}
}

func aggregateAttributes(pf *parquet.File, definitionLevel int, keyPath string, valuePath string, isArrayPath string) (genericAttrSummary, error) {
	makeIter := makeIterFunc(context.Background(), pf)

	iters := []pq.Iterator{
		makeIter(keyPath, nil, "key"),
		makeIter(valuePath, nil, "value"),
	}
	if isArrayPath != "" {
		iters = append(iters, makeIter(isArrayPath, nil, "isArray"))
	}

	attrIter := pq.NewJoinIterator(definitionLevel, iters, &attrStatsCollector{})
	defer attrIter.Close()

	var (
		totalBytes      uint64
		attributes      = make(map[string]uint64, 1000)
		cardinality     = make(map[string]map[string]uint64, 1000)
		arrayAttributes = make(map[string]uint64, 1000)
	)

	for res, err := attrIter.Next(); res != nil; res, err = attrIter.Next() {
		if err != nil {
			return genericAttrSummary{}, err
		}

		for _, e := range res.OtherEntries {
			stats, ok := e.Value.(*attrStats)
			if !ok {
				continue
			}

			if stats.isArray {
				arrayAttributes[stats.name] += stats.bytes
				continue
			}

			attributes[stats.name] += stats.bytes
			totalBytes += stats.bytes

			if cardinality[stats.name] == nil {
				cardinality[stats.name] = make(map[string]uint64, 1000)
			}
			cardinality[stats.name][stats.value]++

			putStats(stats)
		}
	}

	return genericAttrSummary{
		totalBytes:      totalBytes,
		attributes:      attributes,
		arrayAttributes: arrayAttributes,
		cardinality:     cardinality,
	}, nil
}

func aggregateDedicatedColumns(pf *parquet.File, scope backend.DedicatedColumnScope, meta *backend.BlockMeta, paths []string) (genericAttrSummary, error) {
	attrMap := make(map[string]uint64)
	cardinality := make(map[string]map[string]uint64)
	totalBytes := uint64(0)

	i := 0
	for _, dedColumn := range meta.DedicatedColumns {
		if dedColumn.Scope != scope {
			continue
		}

		sz, c, err := aggregateSingleColumn(pf, paths[i])
		if err != nil {
			return genericAttrSummary{}, err
		}
		i++

		attrMap[dedColumn.Name] = sz
		totalBytes += sz
		cardinality[dedColumn.Name] = c
	}

	return genericAttrSummary{
		totalBytes:  totalBytes,
		attributes:  attrMap,
		cardinality: cardinality,
	}, nil
}

func aggregateSingleColumn(pf *parquet.File, colName string) (uint64, map[string]uint64, error) {
	var (
		iter        = makeIterFunc(context.Background(), pf)(colName, nil, "value")
		cardinality = make(map[string]uint64)
		totalBytes  uint64
	)

	for res, err := iter.Next(); res != nil; res, err = iter.Next() {
		if err != nil {
			return 0, nil, err
		}

		var val parquet.Value
		for _, e := range res.Entries {
			if e.Key == "value" {
				val = e.Value
			}
		}

		if val.IsNull() {
			continue
		}

		totalBytes += val.Uint64() // for strings Uint64() returns the length of the string

		cardinality[val.String()]++
	}

	return totalBytes, cardinality, nil
}

func printSummary(scope string, max int, summary genericAttrSummary, simple bool) error {
	// TODO: Support more output formats
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	if max > len(summary.attributes) {
		max = len(summary.attributes)
	}

	fmt.Println("")
	attrList := topN(max, summary.attributes)
	arrayAttrList := topN(max, summary.arrayAttributes)
	if simple {
		fmt.Printf("%s attributes: ", scope)
		for _, a := range attrList {
			fmt.Printf("\"%s.%s\" ", scope, a.name)
		}
		fmt.Println("")
		return nil
	}

	fmt.Printf("Top %d %s attributes by size\n", len(attrList), scope)
	for _, a := range attrList {

		name := a.name
		if _, ok := summary.dedicated[a.name]; ok {
			name = a.name + " (dedicated)"
		}

		percentage := float64(a.bytes) / float64(summary.totalBytes) * 100
		maxLength := uint64(0)
		totalOccurances := uint64(0)
		for v, count := range summary.cardinality[a.name] {
			totalOccurances += count
			if len(v) > int(maxLength) {
				maxLength = uint64(len(v))
			}
		}
		avgOccurance := float64(totalOccurances) / float64(len(summary.cardinality[a.name]))

		_, err := fmt.Fprintf(w, "name: %s\t size: %s\t (%.2f%%)\t max length: %s\t count: %d\t distinct: %d\t avg reuse: %.2f\n",
			name,
			humanize.Bytes(a.bytes),
			percentage,
			humanize.Bytes(maxLength),
			totalOccurances,
			len(summary.cardinality[a.name]),
			avgOccurance,
		)
		if err != nil {
			return err
		}
	}

	err := w.Flush()
	if err != nil {
		return err
	}

	if len(arrayAttrList) > 0 {
		fmt.Printf("Top %d %s array attributes by size\n", len(arrayAttrList), scope)
		for _, a := range arrayAttrList {
			percentage := float64(a.bytes) / float64(summary.totalBytes) * 100
			_, err := fmt.Fprintf(w, "name: %s\t size: %s\t (%s%%)\n", a.name, humanize.Bytes(a.bytes), strconv.FormatFloat(percentage, 'f', 2, 64))
			if err != nil {
				return err
			}
		}

		err = w.Flush()
		if err != nil {
			return err
		}
	}

	return nil
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

func printCliArgs(s *blockSummary, maxAttr int) {
	fmt.Println("")
	fmt.Printf("quoted/spaced cli list:")

	ss := []string{}

	span := topN(maxAttr, s.spanSummary.attributes)
	for _, a := range span {
		ss = append(ss, fmt.Sprintf("\"span.%s\"", a.name))
	}

	resource := topN(maxAttr, s.resourceSummary.attributes)
	for _, a := range resource {
		ss = append(ss, fmt.Sprintf("\"resource.%s\"", a.name))
	}

	fmt.Println(strings.Join(ss, " "))
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

var _ pq.GroupPredicate = (*attrStatsCollector)(nil)

type attrStats struct {
	name    string
	value   string
	bytes   uint64
	isArray bool
	isNull  bool
}

var statsPool = sync.Pool{
	New: func() interface{} {
		return &attrStats{}
	},
}

func putStats(s *attrStats) {
	s.name = ""
	s.value = ""
	s.bytes = 0
	s.isArray = false
	s.isNull = false
	statsPool.Put(s)
}

func getStats() *attrStats {
	return statsPool.Get().(*attrStats)
}

type attrStatsCollector struct{}

func (a attrStatsCollector) String() string {
	return "attrStatsCollector{}"
}

func (a attrStatsCollector) KeepGroup(res *pq.IteratorResult) bool {
	var stats *attrStats

	for _, e := range res.OtherEntries {
		if s, ok := e.Value.(*attrStats); ok {
			stats = s
			break
		}
	}

	if stats == nil {
		stats = getStats()
	}

	for _, e := range res.Entries {
		switch e.Key {
		case "key":
			stats.name = e.Value.String()
		case "value":
			if e.Value.IsNull() {
				stats.isNull = true
			} else {
				stats.value = e.Value.String()
				stats.bytes += e.Value.Uint64() // for strings Uint64() returns the length of the string
			}
		case "isArray":
			if !stats.isArray {
				stats.isArray = e.Value.Boolean()
			}
		}
	}

	res.Reset()
	if stats.isNull {
		putStats(stats)
		return false
	}

	res.AppendOtherValue("stats", stats)
	return true
}
