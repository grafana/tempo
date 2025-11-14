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
	BlobThreshold    string `help:"Convert column to blob when dictionary size reaches this value. Disable with 0" default:"4MiB"`
	GenerateJsonnet  bool   `help:"Generate overrides Jsonnet for dedicated columns"`
	GenerateCliArgs  bool   `help:"Generate textual args for passing to parquet conversion command"`
	SimpleSummary    bool   `help:"Print only single line of top attributes" default:"false"`
	PrintFullSummary bool   `help:"Print full summary of the analysed block" default:"true"`
}

func (cmd *analyseBlockCmd) Run(ctx *globalOptions) error {
	blobBytes, err := humanize.ParseBytes(cmd.BlobThreshold)
	if err != nil {
		return err
	}

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

	return blockSum.print(cmd.NumAttr, cmd.GenerateJsonnet, cmd.SimpleSummary, cmd.PrintFullSummary, cmd.GenerateCliArgs, blobBytes)
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
		numRowGroups:    len(pf.RowGroups()),
		spanSummary:     spanSummary,
		resourceSummary: resSummary,
		eventSummary:    eventSummary,
	}, nil
}

func aggregateScope(pf *parquet.File, meta *backend.BlockMeta, paths scopeAttributePath) (attributeSummary, error) {
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
		}
	}

	return res, nil
}

type blockSummary struct {
	spanSummary     attributeSummary
	resourceSummary attributeSummary
	eventSummary    attributeSummary
	numRowGroups    int
}

func (s *blockSummary) add(other blockSummary) {
	s.numRowGroups += other.numRowGroups
	s.spanSummary.add(other.spanSummary)
	s.resourceSummary.add(other.resourceSummary)
	s.eventSummary.add(other.eventSummary)
}

func (s blockSummary) print(maxAttr int, generateJsonnet, simpleSummary, printFullSummary, generateCliArgs bool, blobBytes uint64) error {
	if printFullSummary {
		if err := printSummary("span", maxAttr, s.spanSummary, false, s.numRowGroups, blobBytes); err != nil {
			return err
		}

		if err := printSummary("resource", maxAttr, s.resourceSummary, false, s.numRowGroups, blobBytes); err != nil {
			return err
		}

		if err := printSummary("event", maxAttr, s.eventSummary, false, s.numRowGroups, blobBytes); err != nil {
			return err
		}
	}

	if simpleSummary {
		if err := printSummary("span", maxAttr, s.spanSummary, true, s.numRowGroups, blobBytes); err != nil {
			return err
		}

		if err := printSummary("resource", maxAttr, s.resourceSummary, true, s.numRowGroups, blobBytes); err != nil {
			return err
		}
	}

	if generateJsonnet {
		printDedicatedColumnOverridesJsonnet(s, maxAttr, s.numRowGroups, blobBytes)
	}

	if generateCliArgs {
		printCliArgs(s, maxAttr, s.numRowGroups, blobBytes)
	}

	return nil
}

type attributeSummary struct {
	attributes      map[string]*attribute // key: attribute name
	arrayAttributes map[string]*attribute // key: attribute name
	dedicated       map[string]struct{}
}

func (a attributeSummary) add(other attributeSummary) {
	for k, v := range other.attributes {
		existing, ok := a.attributes[k]
		if !ok {
			a.attributes[k] = v
			continue
		}
		existing.totalBytes += v.totalBytes
		for k, v := range v.cardinality {
			existing.cardinality[k] += v
		}
	}
	for k, v := range other.arrayAttributes {
		existing, ok := a.arrayAttributes[k]
		if !ok {
			a.arrayAttributes[k] = v
			continue
		}
		existing.totalBytes += v.totalBytes
	}
	for k := range other.dedicated {
		a.dedicated[k] = struct{}{}
	}
}

func (a attributeSummary) totalBytes() uint64 {
	total := uint64(0)
	for _, a := range a.attributes {
		total += a.totalBytes
	}
	return total
}

type attribute struct {
	name        string
	cardinality cardinality // Only populated for non-arraystring attributes
	totalBytes  uint64
}

type cardinality map[string]uint64

func (c cardinality) add(value string) {
	// TODO - instead of storing the raw value in the map, we could hash it and record the length. The
	// requirement is to be able to estimate the cardinality and total content size at the end.
	c[value]++
}

// totalBytes is the sum of all value content length regardless of cardinality or repetitino
func (c cardinality) totalBytes() uint64 {
	total := uint64(0)
	for v, count := range c {
		total += uint64(len(v)) * count
	}
	return total
}

func (c cardinality) distinctValueCount() int {
	return len(c)
}

func (c cardinality) totalOccurrences() uint64 {
	total := uint64(0)
	for _, count := range c {
		total += count
	}
	return total
}

// dictionarySize is the estimated total size of a compressed dictionary for this attribute.
func (c cardinality) dictionarySize() uint64 {
	total := uint64(0)
	for v := range c {
		total += 4 + uint64(len(v)) // 32-bit length, plus the value itself
	}
	return total
}

// avgSizePerRowGroup is the average number of bytes used for this attribute per row group, assuming a
// compressed dictionary and page content of 1 byte per row.
func (c cardinality) avgSizePerRowGroup(numRowGroups int) uint64 {
	dict := c.dictionarySize()
	content := c.totalOccurrences()
	return uint64((float64(dict) + float64(content)) / float64(numRowGroups))
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

func aggregateAttributes(pf *parquet.File, definitionLevel int, keyPath string, valuePath string, isArrayPath string) (attributeSummary, error) {
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
		attributes      = make(map[string]*attribute, 1000)
		arrayAttributes = make(map[string]*attribute, 1000)
	)

	for res, err := attrIter.Next(); res != nil; res, err = attrIter.Next() {
		if err != nil {
			return attributeSummary{}, err
		}

		for _, e := range res.OtherEntries {
			stats, ok := e.Value.(*attrStats)
			if !ok {
				continue
			}

			if stats.isArray {
				v, ok := arrayAttributes[stats.name]
				if !ok {
					v = &attribute{
						name: stats.name,
					}
					arrayAttributes[stats.name] = v
				}
				v.totalBytes += uint64(len(stats.value))
				arrayAttributes[stats.name] = v
				continue
			}

			a, ok := attributes[stats.name]
			if !ok {
				a = &attribute{
					name:        stats.name,
					cardinality: make(cardinality),
				}
				attributes[stats.name] = a
			}

			a.totalBytes += uint64(len(stats.value))
			a.cardinality.add(stats.value)

			putStats(stats)
		}
	}

	return attributeSummary{
		attributes:      attributes,
		arrayAttributes: arrayAttributes,
	}, nil
}

func aggregateDedicatedColumns(pf *parquet.File, scope backend.DedicatedColumnScope, meta *backend.BlockMeta, paths []string) (attributeSummary, error) {
	attributes := make(map[string]*attribute)

	i := 0
	for _, dedColumn := range meta.DedicatedColumns {
		if dedColumn.Scope != scope {
			continue
		}

		c, err := aggregateSingleColumn(pf, paths[i])
		if err != nil {
			return attributeSummary{}, err
		}
		i++

		attributes[dedColumn.Name] = &attribute{
			name:        dedColumn.Name,
			totalBytes:  c.totalBytes(),
			cardinality: c,
		}
	}

	return attributeSummary{
		attributes: attributes,
	}, nil
}

func aggregateSingleColumn(pf *parquet.File, colName string) (cardinality, error) {
	var (
		iter        = makeIterFunc(context.Background(), pf)(colName, nil, "value")
		cardinality = make(cardinality)
	)

	for res, err := iter.Next(); res != nil; res, err = iter.Next() {
		if err != nil {
			return nil, err
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

		cardinality[val.String()]++
	}

	return cardinality, nil
}

func printSummary(scope string, maxAttr int, summary attributeSummary, simple bool, numRowGroups int, blobBytes uint64) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	if maxAttr > len(summary.attributes) {
		maxAttr = len(summary.attributes)
	}

	fmt.Println("")
	attrList := topN(maxAttr, summary.attributes)
	if simple {
		fmt.Printf("%s attributes: ", scope)
		for _, a := range attrList {
			fmt.Printf("\"%s.%s\" ", scope, a.name)
		}
		fmt.Println("")
		return nil
	}

	fmt.Printf("Top %d %s attributes by size\n", len(attrList), scope)
	totalBytes := summary.totalBytes()
	for _, a := range attrList {

		name := a.name
		if _, ok := summary.dedicated[a.name]; ok {
			name = a.name + " (dedicated)"
		}

		var (
			thisBytes       = a.totalBytes
			percentage      = float64(thisBytes) / float64(totalBytes) * 100
			totalOccurences = a.cardinality.totalOccurrences()
			distinct        = a.cardinality.distinctValueCount()
			avgReuse        = float64(totalOccurences) / float64(distinct)
			totalSize       = a.cardinality.avgSizePerRowGroup(numRowGroups)
		)

		blob := ""
		if blobBytes > 0 && totalSize >= blobBytes {
			blob = "(blob)"
		}

		_, err := fmt.Fprintf(w, "name: %s\t size: %s\t (%.2f%%)\tcount: %d\t distinct: %d\t avg reuse: %.2f\t avg rowgroup content (dict + body): %s %s\n",
			name,
			humanize.Bytes(thisBytes),
			percentage,
			totalOccurences,
			distinct,
			avgReuse,
			humanize.Bytes(totalSize),
			blob,
		)
		if err != nil {
			return err
		}
	}

	err := w.Flush()
	if err != nil {
		return err
	}

	arrayAttrList := topN(maxAttr, summary.arrayAttributes)
	if len(arrayAttrList) > 0 {
		fmt.Printf("Top %d %s array attributes by size\n", len(arrayAttrList), scope)
		for _, a := range arrayAttrList {
			percentage := float64(a.totalBytes) / float64(totalBytes) * 100
			_, err := fmt.Fprintf(w, "name: %s\t size: %s\t (%s%%)\n", a.name, humanize.Bytes(a.totalBytes), strconv.FormatFloat(percentage, 'f', 2, 64))
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

func printDedicatedColumnOverridesJsonnet(summary blockSummary, maxAttr int, numRowGroups int, blobBytes uint64) {
	fmt.Println("")
	fmt.Printf("parquet_dedicated_columns: [\n")

	optionsText := func(a *attribute) string {
		options := []string{}
		if blobBytes > 0 && a.cardinality.avgSizePerRowGroup(numRowGroups) > blobBytes {
			options = append(options, "'blob'")
		}
		if len(options) > 0 {
			return ", options: [" + strings.Join(options, ", ") + "]"
		}
		return ""
	}

	for _, a := range topN(maxAttr, summary.spanSummary.attributes) {
		fmt.Printf(" { scope: 'span', name: '%s', type: 'string' %s },\n", a.name, optionsText(a))
	}

	for _, a := range topN(maxAttr, summary.resourceSummary.attributes) {
		fmt.Printf(" { scope: 'resource', name: '%s', type: 'string' %s },\n", a.name, optionsText(a))
	}

	for _, a := range topN(maxAttr, summary.eventSummary.attributes) {
		fmt.Printf(" { scope: 'event', name: '%s', type: 'string' %s },\n", a.name, optionsText(a))
	}

	fmt.Printf("], \n")
	fmt.Println("")
}

func printCliArgs(s blockSummary, maxAttr int, numRowGroups int, blobBytes uint64) {
	fmt.Println("")
	fmt.Printf("quoted/spaced cli list:")

	ss := []string{}
	for _, a := range topN(maxAttr, s.spanSummary.attributes) {
		if blobBytes > 0 && a.cardinality.avgSizePerRowGroup(numRowGroups) > blobBytes {
			ss = append(ss, fmt.Sprintf("\"blob/span.%s\"", a.name))
		} else {
			ss = append(ss, fmt.Sprintf("\"span.%s\"", a.name))
		}
	}

	for _, a := range topN(maxAttr, s.resourceSummary.attributes) {
		if blobBytes > 0 && a.cardinality.avgSizePerRowGroup(numRowGroups) > blobBytes {
			ss = append(ss, fmt.Sprintf("\"blob/resource.%s\"", a.name))
		} else {
			ss = append(ss, fmt.Sprintf("\"resource.%s\"", a.name))
		}
	}

	for _, a := range topN(maxAttr, s.eventSummary.attributes) {
		if blobBytes > 0 && a.cardinality.avgSizePerRowGroup(numRowGroups) > blobBytes {
			ss = append(ss, fmt.Sprintf("\"blob/event.%s\"", a.name))
		} else {
			ss = append(ss, fmt.Sprintf("\"event.%s\"", a.name))
		}
	}

	fmt.Println(strings.Join(ss, " "))
}

func topN(n int, attrs map[string]*attribute) []*attribute {
	top := make([]*attribute, 0, len(attrs))
	for _, attr := range attrs {
		top = append(top, attr)
	}
	sort.Slice(top, func(i, j int) bool {
		return top[i].totalBytes > top[j].totalBytes
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
				stats.bytes += uint64(len(stats.value))
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
