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
	"github.com/grafana/tempo/pkg/parquetquery"
	"github.com/grafana/tempo/pkg/traceql"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/vparquet3"
	"github.com/grafana/tempo/tempodb/encoding/vparquet4"
	"github.com/grafana/tempo/tempodb/encoding/vparquet5"
)

type attributePaths struct {
	span  scopeAttributePath
	res   scopeAttributePath
	event scopeAttributePath
}

type scopeAttributePath struct {
	defLevel              int
	keyPath               string
	valPath               string
	intPath               string
	isArrayPath           string
	dedicatedColScope     backend.DedicatedColumnScope
	dedicatedColsPaths    []string
	dedicatedColsPathsInt []string
	wellKnownPathsString  map[string]string // key: attribute name, value: path
	wellKnownPathsInt     map[string]string // key: attribute name, value: path
	rowCountPath          string
}

func pathsForVersion(v string) attributePaths {
	switch v {
	case vparquet3.VersionString:
		return attributePaths{
			span: scopeAttributePath{
				defLevel:              vparquet3.DefinitionLevelResourceSpansILSSpanAttrs,
				keyPath:               vparquet3.FieldSpanAttrKey,
				valPath:               vparquet3.FieldSpanAttrVal,
				intPath:               vparquet3.FieldSpanAttrValInt,
				dedicatedColScope:     backend.DedicatedColumnScopeSpan,
				dedicatedColsPaths:    vparquet3.DedicatedResourceColumnPaths[backend.DedicatedColumnScopeSpan][backend.DedicatedColumnTypeString],
				dedicatedColsPathsInt: vparquet3.DedicatedResourceColumnPaths[backend.DedicatedColumnScopeSpan][backend.DedicatedColumnTypeInt],
			},
			res: scopeAttributePath{
				defLevel:              vparquet3.DefinitionLevelResourceAttrs,
				keyPath:               vparquet3.FieldResourceAttrKey,
				valPath:               vparquet3.FieldResourceAttrVal,
				intPath:               vparquet3.FieldResourceAttrValInt,
				dedicatedColScope:     backend.DedicatedColumnScopeResource,
				dedicatedColsPaths:    vparquet3.DedicatedResourceColumnPaths[backend.DedicatedColumnScopeResource][backend.DedicatedColumnTypeString],
				dedicatedColsPathsInt: vparquet3.DedicatedResourceColumnPaths[backend.DedicatedColumnScopeResource][backend.DedicatedColumnTypeInt],
			},
		}
	case vparquet4.VersionString:
		return attributePaths{
			span: scopeAttributePath{
				defLevel:              vparquet4.DefinitionLevelResourceSpansILSSpanAttrs,
				keyPath:               vparquet4.FieldSpanAttrKey,
				valPath:               vparquet4.FieldSpanAttrVal,
				isArrayPath:           vparquet4.FieldSpanAttrIsArray,
				intPath:               vparquet4.FieldSpanAttrValInt,
				dedicatedColScope:     backend.DedicatedColumnScopeSpan,
				dedicatedColsPaths:    vparquet4.DedicatedResourceColumnPaths[backend.DedicatedColumnScopeSpan][backend.DedicatedColumnTypeString],
				dedicatedColsPathsInt: vparquet4.DedicatedResourceColumnPaths[backend.DedicatedColumnScopeSpan][backend.DedicatedColumnTypeInt],
				wellKnownPathsString: map[string]string{
					"http.method": vparquet4.WellKnownColumnLookups["http.method"].ColumnPath,
					"http.url":    vparquet4.WellKnownColumnLookups["http.url"].ColumnPath,
				},
				wellKnownPathsInt: map[string]string{
					"http.status_code": vparquet4.WellKnownColumnLookups["http.status_code"].ColumnPath,
				},
				rowCountPath: vparquet4.ColumnPathSpanName,
			},
			res: scopeAttributePath{
				defLevel:              vparquet4.DefinitionLevelResourceAttrs,
				keyPath:               vparquet4.FieldResourceAttrKey,
				valPath:               vparquet4.FieldResourceAttrVal,
				isArrayPath:           vparquet4.FieldResourceAttrIsArray,
				intPath:               vparquet4.FieldResourceAttrValInt,
				dedicatedColScope:     backend.DedicatedColumnScopeResource,
				dedicatedColsPaths:    vparquet4.DedicatedResourceColumnPaths[backend.DedicatedColumnScopeResource][backend.DedicatedColumnTypeString],
				dedicatedColsPathsInt: vparquet4.DedicatedResourceColumnPaths[backend.DedicatedColumnScopeResource][backend.DedicatedColumnTypeInt],
				wellKnownPathsString: map[string]string{
					// Service name is a fixed column in every parquet version
					// So we don't need to measure it.
					// "service.name":       vparquet4.WellKnownColumnLookups["service.name"].ColumnPath,
					"cluster":            vparquet4.WellKnownColumnLookups["cluster"].ColumnPath,
					"namespace":          vparquet4.WellKnownColumnLookups["namespace"].ColumnPath,
					"pod":                vparquet4.WellKnownColumnLookups["pod"].ColumnPath,
					"container":          vparquet4.WellKnownColumnLookups["container"].ColumnPath,
					"k8s.cluster.name":   vparquet4.WellKnownColumnLookups["k8s.cluster.name"].ColumnPath,
					"k8s.namespace.name": vparquet4.WellKnownColumnLookups["k8s.namespace.name"].ColumnPath,
					"k8s.pod.name":       vparquet4.WellKnownColumnLookups["k8s.pod.name"].ColumnPath,
					"k8s.container.name": vparquet4.WellKnownColumnLookups["k8s.container.name"].ColumnPath,
				},
				rowCountPath: vparquet4.ColumnPathResourceServiceName,
			},
			event: scopeAttributePath{
				defLevel:     vparquet4.DefinitionLevelResourceSpansILSSpanEventAttrs,
				keyPath:      vparquet4.FieldEventAttrKey,
				valPath:      vparquet4.FieldEventAttrVal,
				intPath:      vparquet4.FieldEventAttrValInt,
				isArrayPath:  vparquet4.FieldEventAttrIsArray,
				rowCountPath: vparquet4.ColumnPathEventName,
			},
		}
	case vparquet5.VersionString:
		return attributePaths{
			span: scopeAttributePath{
				defLevel:              vparquet5.DefinitionLevelResourceSpansILSSpanAttrs,
				keyPath:               vparquet5.FieldSpanAttrKey,
				valPath:               vparquet5.FieldSpanAttrVal,
				intPath:               vparquet5.FieldSpanAttrValInt,
				isArrayPath:           vparquet5.FieldSpanAttrIsArray,
				dedicatedColScope:     backend.DedicatedColumnScopeSpan,
				dedicatedColsPaths:    vparquet5.DedicatedResourceColumnPaths[backend.DedicatedColumnScopeSpan][backend.DedicatedColumnTypeString],
				dedicatedColsPathsInt: vparquet5.DedicatedResourceColumnPaths[backend.DedicatedColumnScopeSpan][backend.DedicatedColumnTypeInt],
				rowCountPath:          vparquet5.ColumnPathSpanName,
			},
			res: scopeAttributePath{
				defLevel:              vparquet5.DefinitionLevelResourceAttrs,
				keyPath:               vparquet5.FieldResourceAttrKey,
				valPath:               vparquet5.FieldResourceAttrVal,
				intPath:               vparquet5.FieldResourceAttrValInt,
				isArrayPath:           vparquet5.FieldResourceAttrIsArray,
				dedicatedColScope:     backend.DedicatedColumnScopeResource,
				dedicatedColsPaths:    vparquet5.DedicatedResourceColumnPaths[backend.DedicatedColumnScopeResource][backend.DedicatedColumnTypeString],
				dedicatedColsPathsInt: vparquet5.DedicatedResourceColumnPaths[backend.DedicatedColumnScopeResource][backend.DedicatedColumnTypeInt],
				rowCountPath:          vparquet5.ColumnPathResourceServiceName,
			},
			event: scopeAttributePath{
				defLevel:              vparquet5.DefinitionLevelResourceSpansILSSpanEventAttrs,
				keyPath:               vparquet5.FieldEventAttrKey,
				valPath:               vparquet5.FieldEventAttrVal,
				intPath:               vparquet5.FieldEventAttrValInt,
				isArrayPath:           vparquet5.FieldEventAttrIsArray,
				dedicatedColScope:     backend.DedicatedColumnScopeEvent,
				dedicatedColsPaths:    vparquet5.DedicatedResourceColumnPaths[backend.DedicatedColumnScopeEvent][backend.DedicatedColumnTypeString],
				dedicatedColsPathsInt: vparquet5.DedicatedResourceColumnPaths[backend.DedicatedColumnScopeEvent][backend.DedicatedColumnTypeInt],
				rowCountPath:          vparquet5.ColumnPathEventName,
			},
		}
	default:
		panic("unsupported version")
	}
}

type heuristicSettings struct {
	NumStringAttr       int     // Target number of dedicated string attributes
	NumIntAttr          int     // Target number of dedicated integer attributes
	BlobThresholdBytes  uint64  // Attribute row group content above this size are denoted as blobs
	IntThresholdPercent float64 // Integers surpassing this percent of rows are recommended to be dedicated
	StrThresholdPercent float64 // Strings surpassing this percent of rows are recommended to be dedicated
}

type printSettings struct {
	Simple  bool
	Full    bool
	Jsonnet bool
	CliArgs bool
}

type analyseBlockCmd struct {
	backendOptions

	TenantID            string  `arg:"" help:"tenant-id within the bucket"`
	BlockID             string  `arg:"" help:"block ID to list"`
	NumAttr             int     `help:"Number of attributes to display" default:"20"`
	NumIntAttr          int     `help:"Number of integer attributes to display. If set to 0 then it will use the other parameter." default:"5"`
	BlobThreshold       string  `help:"Convert column to blob when dictionary size reaches this value. Disable with 0" default:"4MiB"`
	IntPercentThreshold float64 `help:"Threshold for integer attributes put in dedicated columns. Default 5% = 0.05" default:"0.05"`
	StrPercentThreshold float64 `help:"Threshold for string attributes put in dedicated columns. Default 3% = 0.03." default:"0.03"`
	IncludeWellKnown    bool    `help:"Include well-known attributes in the analysis. These are attributes with fixed columns in some versions of parquet, like http.url. This should be enabled when generating dedicated columns targeting vParquet5 or higher which can make use of them." default:"false"`
	GenerateJsonnet     bool    `help:"Generate overrides Jsonnet for dedicated columns"`
	GenerateCliArgs     bool    `help:"Generate textual args for passing to parquet conversion command"`
	SimpleSummary       bool    `help:"Print only single line of top attributes" default:"false"`
	PrintFullSummary    bool    `help:"Print full summary of the analysed block" default:"true"`
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

	blockSum, err := processBlock(r, cmd.TenantID, cmd.BlockID, cmd.IncludeWellKnown, time.Time{}, time.Time{}, 0)
	if err != nil {
		if errors.Is(err, backend.ErrDoesNotExist) {
			return fmt.Errorf("unable to analyze block: block has no block.meta because it was compacted")
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

	if cmd.StrPercentThreshold <= 0 || cmd.StrPercentThreshold >= 1 {
		return errors.New("str percent threshold must be between 0 and 1")
	}

	settings := heuristicSettings{
		NumStringAttr:       cmd.NumAttr,
		NumIntAttr:          cmd.NumIntAttr,
		BlobThresholdBytes:  blobBytes,
		IntThresholdPercent: cmd.IntPercentThreshold,
		StrThresholdPercent: cmd.StrPercentThreshold,
	}

	printSettings := printSettings{
		Simple:  cmd.SimpleSummary,
		Full:    cmd.PrintFullSummary,
		Jsonnet: cmd.GenerateJsonnet,
		CliArgs: cmd.GenerateCliArgs,
	}

	return blockSum.print(settings, printSettings)
}

func processBlock(r backend.Reader, tenantID, blockID string, includeWellKnown bool, maxStartTime, minStartTime time.Time, minCompactionLvl uint32) (*blockSummary, error) {
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
	case vparquet3.VersionString:
		reader = vparquet3.NewBackendReaderAt(context.Background(), r, vparquet3.DataFileName, meta)
	case vparquet4.VersionString:
		reader = vparquet4.NewBackendReaderAt(context.Background(), r, vparquet4.DataFileName, meta)
	case vparquet5.VersionString:
		reader = vparquet5.NewBackendReaderAt(context.Background(), r, vparquet5.DataFileName, meta)
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

	spanSummary, err := aggregateScope(pf, meta, paths.span, includeWellKnown)
	if err != nil {
		return nil, err
	}

	resSummary, err := aggregateScope(pf, meta, paths.res, includeWellKnown)
	if err != nil {
		return nil, err
	}

	eventSummary, err := aggregateScope(pf, meta, paths.event, includeWellKnown)
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

func aggregateScope(pf *parquet.File, meta *backend.BlockMeta, paths scopeAttributePath, includeWellKnown bool) (attributeSummary, error) {
	var strings []backend.DedicatedColumn
	var ints []backend.DedicatedColumn
	for _, c := range meta.DedicatedColumns {
		if c.Scope == paths.dedicatedColScope {
			switch c.Type {
			case backend.DedicatedColumnTypeString:
				strings = append(strings, c)
			case backend.DedicatedColumnTypeInt:
				ints = append(ints, c)
			}
		}
	}

	res, err := aggregateGenericAttributes(pf, paths.defLevel, paths.keyPath, paths.valPath, paths.intPath, paths.isArrayPath)
	if err != nil {
		return res, err
	}

	res.dedicated = make(map[string]struct{})

	for i, c := range strings {
		cardinality, err := aggregateStringColumn(pf, paths.dedicatedColsPaths[i])
		if err != nil {
			return res, err
		}
		res.attributes[c.Name] = &stringAttributeSummary{
			name:        c.Name,
			totalBytes:  cardinality.totalBytes(),
			cardinality: cardinality,
		}
		res.dedicated[c.Name] = struct{}{}
	}

	if includeWellKnown {
		for wellKnownAttr, path := range paths.wellKnownPathsString {
			cardinality, err := aggregateStringColumn(pf, path)
			if err != nil {
				return res, err
			}
			res.attributes[wellKnownAttr] = &stringAttributeSummary{
				name:        wellKnownAttr,
				totalBytes:  cardinality.totalBytes(),
				cardinality: cardinality,
			}
			res.dedicated[wellKnownAttr] = struct{}{} // Well-known columns are also dedicated columns.
		}
	}

	for i, c := range ints {
		path := paths.dedicatedColsPathsInt[i]
		count, err := aggregateIntegerColumn(pf, path)
		if err != nil {
			return res, err
		}
		res.integerAttributes[path] = &integerAttributeSummary{
			name:  c.Name,
			count: count,
		}
		res.dedicated[path] = struct{}{}
	}

	if includeWellKnown {
		for wellKnownAttr, path := range paths.wellKnownPathsInt {
			count, err := aggregateIntegerColumn(pf, path)
			if err != nil {
				return res, err
			}
			res.integerAttributes[wellKnownAttr] = &integerAttributeSummary{
				name:  wellKnownAttr,
				count: count,
			}
			res.dedicated[wellKnownAttr] = struct{}{} // Well-known columns are also dedicated columns.
		}
	}

	if paths.rowCountPath != "" {
		count, err := rowCount(pf, paths.rowCountPath)
		if err != nil {
			return res, err
		}
		res.rowCount = count
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

func (s blockSummary) print(settings heuristicSettings, printSettings printSettings) error {
	if printSettings.Full {
		if err := printFullSummary("span", settings, s.spanSummary, s.numRowGroups); err != nil {
			return err
		}

		if err := printFullSummary("resource", settings, s.resourceSummary, s.numRowGroups); err != nil {
			return err
		}

		if err := printFullSummary("event", settings, s.eventSummary, s.numRowGroups); err != nil {
			return err
		}
	}

	if printSettings.Simple {
		printSimpleSummary("span", settings.NumStringAttr, s.spanSummary)
		printSimpleSummary("resource", settings.NumStringAttr, s.resourceSummary)
		printSimpleSummary("event", settings.NumStringAttr, s.eventSummary)
	}

	if printSettings.Jsonnet {
		printDedicatedColumnOverridesJsonnet(s, settings, s.numRowGroups)
	}

	if printSettings.CliArgs {
		printCliArgs(s, settings, s.numRowGroups)
	}

	return nil
}

func (s blockSummary) ToDedicatedColumns(settings heuristicSettings) []backend.DedicatedColumn {
	var dedicatedCols []backend.DedicatedColumn

	doStringSummary := func(summary attributeSummary, scope backend.DedicatedColumnScope) {
		if summary.rowCount == 0 {
			return
		}
		for _, attr := range topN(settings.NumStringAttr, summary.attributes) {
			percentOfRows := float64(attr.cardinality.totalOccurrences()) / float64(summary.rowCount)
			if percentOfRows < settings.StrThresholdPercent {
				continue
			}

			options := backend.DedicatedColumnOptions{}
			totalSize := attr.cardinality.avgSizePerRowGroup(s.numRowGroups)
			if settings.BlobThresholdBytes > 0 && totalSize >= settings.BlobThresholdBytes {
				options = append(options, backend.DedicatedColumnOptionBlob)
			}
			dedicatedCols = append(dedicatedCols, backend.DedicatedColumn{
				Name:    attr.name,
				Scope:   scope,
				Type:    backend.DedicatedColumnTypeString,
				Options: options,
			})
		}
	}

	doIntSummary := func(summary attributeSummary, scope backend.DedicatedColumnScope) {
		if summary.rowCount == 0 {
			return
		}
		for _, attr := range topNInt(settings.NumIntAttr, summary.integerAttributes) {
			percentOfRows := float64(attr.count) / float64(summary.rowCount)
			if percentOfRows < settings.IntThresholdPercent {
				continue
			}
			dedicatedCols = append(dedicatedCols, backend.DedicatedColumn{
				Name:    attr.name,
				Scope:   scope,
				Type:    backend.DedicatedColumnTypeInt,
				Options: backend.DedicatedColumnOptions{},
			})
		}
	}

	doStringSummary(s.spanSummary, backend.DedicatedColumnScopeSpan)
	doIntSummary(s.spanSummary, backend.DedicatedColumnScopeSpan)
	doStringSummary(s.resourceSummary, backend.DedicatedColumnScopeResource)
	doIntSummary(s.resourceSummary, backend.DedicatedColumnScopeResource)
	doStringSummary(s.eventSummary, backend.DedicatedColumnScopeEvent)
	doIntSummary(s.eventSummary, backend.DedicatedColumnScopeEvent)

	return dedicatedCols
}

type attributeSummary struct {
	attributes             map[string]*stringAttributeSummary  // key: attribute name
	arrayAttributes        map[string]*stringAttributeSummary  // key: attribute name
	integerAttributes      map[string]*integerAttributeSummary // key: attribute name
	integerArrayAttributes map[string]*integerAttributeSummary // key: attribute name
	dedicated              map[string]struct{}
	rowCount               uint64
}

func (a *attributeSummary) add(other attributeSummary) {
	a.rowCount += other.rowCount

	if a.dedicated == nil {
		a.dedicated = make(map[string]struct{}, len(other.dedicated))
	}
	for k := range other.dedicated {
		a.dedicated[k] = struct{}{}
	}

	mergeStringSummary := func(m *map[string]*stringAttributeSummary, other map[string]*stringAttributeSummary) {
		if *m == nil {
			*m = make(map[string]*stringAttributeSummary, len(other))
		}
		for k, v := range other {
			existing, ok := (*m)[k]
			if !ok {
				(*m)[k] = v
				continue
			}
			existing.totalBytes += v.totalBytes
			for k, v := range v.cardinality {
				existing.cardinality[k] += v
			}
		}
	}

	mergeIntegerSummary := func(m *map[string]*integerAttributeSummary, other map[string]*integerAttributeSummary) {
		if *m == nil {
			*m = make(map[string]*integerAttributeSummary, len(other))
		}
		for k, v := range other {
			existing, ok := (*m)[k]
			if !ok {
				(*m)[k] = v
				continue
			}
			existing.count += v.count
		}
	}

	mergeStringSummary(&a.attributes, other.attributes)
	mergeStringSummary(&a.arrayAttributes, other.arrayAttributes)
	mergeIntegerSummary(&a.integerAttributes, other.integerAttributes)
	mergeIntegerSummary(&a.integerArrayAttributes, other.integerArrayAttributes)
}

func (a attributeSummary) totalBytes() uint64 {
	total := uint64(0)
	for _, a := range a.attributes {
		total += a.totalBytes
	}
	return total
}

func (a attributeSummary) totalStringCount() uint64 {
	total := uint64(0)
	for _, a := range a.attributes {
		total += a.cardinality.totalOccurrences()
	}
	return total
}

func (a attributeSummary) totalIntegerCount() uint64 {
	total := uint64(0)
	for _, a := range a.integerAttributes {
		total += a.count
	}
	return total
}

type stringAttributeSummary struct {
	name        string
	cardinality cardinality // Only populated for non-arraystring attributes
	totalBytes  uint64
}

type integerAttributeSummary struct {
	name  string
	count uint64
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

type makeIterFn func(columnName string, predicate parquetquery.Predicate, selectAs string) parquetquery.Iterator

func makeIterFunc(ctx context.Context, pf *parquet.File) makeIterFn {
	return func(name string, predicate parquetquery.Predicate, selectAs string) parquetquery.Iterator {
		index, _, maxDef := parquetquery.GetColumnIndexByPath(pf, name)
		if index == -1 {
			panic("column not found in parquet file:" + name)
		}

		opts := []parquetquery.SyncIteratorOpt{
			parquetquery.SyncIteratorOptColumnName(name),
			parquetquery.SyncIteratorOptPredicate(predicate),
			parquetquery.SyncIteratorOptSelectAs(selectAs),
			parquetquery.SyncIteratorOptMaxDefinitionLevel(maxDef),
		}

		return parquetquery.NewSyncIterator(ctx, pf.RowGroups(), index, opts...)
	}
}

func aggregateGenericAttributes(pf *parquet.File, definitionLevel int, keyPath string, valuePath string, intPath string, isArrayPath string) (attributeSummary, error) {
	makeIter := makeIterFunc(context.Background(), pf)

	required := []parquetquery.Iterator{
		makeIter(keyPath, parquetquery.NewSkipNilsPredicate(), "key"),
	}
	optional := []parquetquery.Iterator{
		makeIter(valuePath, parquetquery.NewSkipNilsPredicate(), "value"),
		makeIter(intPath, parquetquery.NewSkipNilsPredicate(), "int"),
	}
	if isArrayPath != "" {
		optional = append(optional, makeIter(isArrayPath, parquetquery.NewSkipNilsPredicate(), "isArray"))
	}

	attrIter, err := parquetquery.NewLeftJoinIterator(definitionLevel, required, optional, &attrStatsCollector{})
	if err != nil {
		return attributeSummary{}, err
	}
	defer attrIter.Close()

	var (
		attributes             = make(map[string]*stringAttributeSummary, 1000)
		stringArrayAttributes  = make(map[string]*stringAttributeSummary, 1000)
		integerAttributes      = make(map[string]*integerAttributeSummary, 1000)
		integerArrayAttributes = make(map[string]*integerAttributeSummary, 1000)
	)

	getString := func(name string, from map[string]*stringAttributeSummary) *stringAttributeSummary {
		v, ok := from[name]
		if !ok {
			v = &stringAttributeSummary{
				name:        name,
				cardinality: make(cardinality),
			}
			from[name] = v
		}
		return v
	}

	getInt := func(name string, from map[string]*integerAttributeSummary) *integerAttributeSummary {
		v, ok := from[name]
		if !ok {
			v = &integerAttributeSummary{
				name: name,
			}
			from[name] = v
		}
		return v
	}

	for res, err := attrIter.Next(); res != nil; res, err = attrIter.Next() {
		if err != nil {
			return attributeSummary{}, err
		}

		for _, e := range res.OtherEntries {
			stats, ok := e.Value.(*attrStats)
			if !ok {
				continue
			}

			switch stats.typ {
			case attrTypeString:
				if stats.isArray {
					v := getString(stats.name, stringArrayAttributes)
					v.totalBytes += uint64(len(stats.value))
					v.cardinality.add(stats.value)
				} else {
					v := getString(stats.name, attributes)
					v.totalBytes += uint64(len(stats.value))
					v.cardinality.add(stats.value)
				}
			case attrTypeInt:
				if stats.isArray {
					v := getInt(stats.name, integerArrayAttributes)
					v.count++
				} else {
					v := getInt(stats.name, integerAttributes)
					v.count++
				}
			case attrTypeNull:
			}

			putStats(stats)
		}
	}

	return attributeSummary{
		attributes:             attributes,
		arrayAttributes:        stringArrayAttributes,
		integerAttributes:      integerAttributes,
		integerArrayAttributes: integerArrayAttributes,
	}, nil
}

func aggregateIntegerColumn(pf *parquet.File, colName string) (uint64, error) {
	var (
		iter  = makeIterFunc(context.Background(), pf)(colName, parquetquery.NewSkipNilsPredicate(), "")
		count uint64
	)

	for res, err := iter.Next(); res != nil; res, err = iter.Next() {
		if err != nil {
			return 0, err
		}
		count++
	}

	return count, nil
}

func aggregateStringColumn(pf *parquet.File, colName string) (cardinality, error) {
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

func rowCount(pf *parquet.File, colName string) (count uint64, err error) {
	index, _, _ := parquetquery.GetColumnIndexByPath(pf, colName)
	if index == -1 {
		return 0, errors.New("column not found in parquet file:" + colName)
	}

	for _, rg := range pf.RowGroups() {
		count += uint64(rg.ColumnChunks()[index].NumValues())
	}

	return count, nil
}

func printSimpleSummary(scope string, maxAttr int, summary attributeSummary) {
	if maxAttr > len(summary.attributes) {
		maxAttr = len(summary.attributes)
	}

	fmt.Println("")
	attrList := topN(maxAttr, summary.attributes)
	fmt.Printf("%s attributes: ", scope)
	for _, a := range attrList {
		fmt.Printf("\"%s\", ", a.name)
	}
	fmt.Println("")
}

func printFullSummary(scope string, settings heuristicSettings, summary attributeSummary, numRowGroups int) error {
	var (
		w                 = tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		err               error
		totalBytes        = summary.totalBytes()
		totalIntegerCount = summary.totalIntegerCount()
	)

	fmt.Println("--------------------------------")
	fmt.Printf("- %s Summary -\n", scope)
	fmt.Println("--------------------------------")

	fmt.Printf("Total rows: %d\n", summary.rowCount)
	fmt.Printf("Total string values: %d\n", summary.totalStringCount())

	fmt.Println("")
	attrList := topN(settings.NumStringAttr, summary.attributes)
	if len(attrList) > 0 {
		fmt.Printf("Top %d attributes by size\n", len(attrList))

		for _, a := range attrList {
			var (
				name               = a.name
				thisBytes          = a.totalBytes
				percentage         = float64(thisBytes) / float64(totalBytes) * 100
				totalOccurences    = a.cardinality.totalOccurrences()
				distinct           = a.cardinality.distinctValueCount()
				avgReuse           = float64(totalOccurences) / float64(distinct)
				totalSize          = a.cardinality.avgSizePerRowGroup(numRowGroups)
				blobText           = ""
				percentOfRowsText  = ""
				shouldDedicateText = ""
			)

			if _, ok := summary.dedicated[a.name]; ok {
				name = a.name + " (dedicated)"
			}

			if settings.BlobThresholdBytes > 0 && totalSize >= settings.BlobThresholdBytes {
				blobText = "(blob)"
			}

			if summary.rowCount > 0 {
				percentOfRows := float64(totalOccurences) / float64(summary.rowCount)
				percentOfRowsText = fmt.Sprintf("(%.2f%% of rows)", percentOfRows*100)
				if percentOfRows >= settings.StrThresholdPercent {
					shouldDedicateText = "✅ Recommended dedicated column"
				}
			}

			_, err := fmt.Fprintf(w, "name: %s\t size: %s\t (%.2f%%)\tcount: %d\t%s\t distinct: %d\t avg reuse: %.2f\t avg rowgroup content (dict + body): %s %s\t%s\n",
				name,
				humanize.Bytes(thisBytes),
				percentage,
				totalOccurences,
				percentOfRowsText,
				distinct,
				avgReuse,
				humanize.Bytes(totalSize),
				blobText,
				shouldDedicateText,
			)
			if err != nil {
				return err
			}
		}

		err := w.Flush()
		if err != nil {
			return err
		}
	}

	arrayAttrList := topN(settings.NumStringAttr, summary.arrayAttributes)
	if len(arrayAttrList) > 0 {
		fmt.Println("")
		fmt.Printf("Top %d array attributes by size\n", len(arrayAttrList))
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

	integerAttrList := topNInt(settings.NumIntAttr, summary.integerAttributes)
	if len(integerAttrList) > 0 {
		fmt.Println("")
		fmt.Println("Total integer attribute values:", totalIntegerCount)
		fmt.Printf("Top %d integer attributes by count\n", len(integerAttrList))
		for _, a := range integerAttrList {
			var (
				name                = a.name
				percentOfValuesText = "n/a"
				percentOfRowsText   = "n/a"
				shouldDedicateText  = ""
			)

			if _, ok := summary.dedicated[a.name]; ok {
				name = a.name + " (dedicated)"
			}
			if totalIntegerCount > 0 {
				percentOfValuesText = fmt.Sprintf("(%.2f%% of values)", float64(a.count)/float64(totalIntegerCount)*100)
			}
			if summary.rowCount > 0 {
				percentOfRows := float64(a.count) / float64(summary.rowCount)
				percentOfRowsText = fmt.Sprintf("(%.2f%% of rows)", percentOfRows*100)
				if percentOfRows >= settings.IntThresholdPercent {
					shouldDedicateText = "✅ Recommended dedicated column"
				}
			}
			fmt.Fprintf(w, "name: %s\t count: %d\t %s\t%s\t%s\n", name, a.count, percentOfValuesText, percentOfRowsText, shouldDedicateText)
		}
		err = w.Flush()
		if err != nil {
			return err
		}
	}

	return nil
}

func printDedicatedColumnOverridesJsonnet(summary blockSummary, settings heuristicSettings, numRowGroups int) {
	fmt.Println("")
	fmt.Printf("parquet_dedicated_columns: [\n")

	optionsText := func(a *stringAttributeSummary) string {
		options := []string{}
		if settings.BlobThresholdBytes > 0 && a.cardinality.avgSizePerRowGroup(numRowGroups) > settings.BlobThresholdBytes {
			options = append(options, "'blob'")
		}
		if len(options) > 0 {
			return ", options: [" + strings.Join(options, ", ") + "]"
		}
		return ""
	}

	for _, a := range topN(settings.NumStringAttr, summary.spanSummary.attributes) {
		fmt.Printf(" { scope: 'span', name: '%s', type: 'string' %s },\n", a.name, optionsText(a))
	}

	for _, a := range topN(settings.NumStringAttr, summary.resourceSummary.attributes) {
		fmt.Printf(" { scope: 'resource', name: '%s', type: 'string' %s },\n", a.name, optionsText(a))
	}

	for _, a := range topN(settings.NumStringAttr, summary.eventSummary.attributes) {
		fmt.Printf(" { scope: 'event', name: '%s', type: 'string' %s },\n", a.name, optionsText(a))
	}

	fmt.Printf("], \n")
	fmt.Println("")
}

func printCliArgs(s blockSummary, settings heuristicSettings, numRowGroups int) {
	fmt.Println("")
	fmt.Printf("quoted/spaced cli list:")

	escapeString := func(s string) string {
		return strings.ReplaceAll(s, "\"", "\\\"")
	}

	doStringSummary := func(summary attributeSummary, scope traceql.AttributeScope) {
		for _, a := range topN(settings.NumStringAttr, summary.attributes) {
			if float64(a.cardinality.totalOccurrences())/float64(summary.rowCount) < settings.StrThresholdPercent {
				// Did not meet threshold
				continue
			}
			attrStr := traceql.NewScopedAttribute(scope, false, a.name).String()
			if settings.BlobThresholdBytes > 0 && a.cardinality.avgSizePerRowGroup(numRowGroups) > settings.BlobThresholdBytes {
				attrStr = "blob/" + attrStr
			}
			fmt.Printf("\"%s\" ", escapeString(attrStr))
		}
	}

	doIntSummary := func(summary attributeSummary, scope traceql.AttributeScope) {
		for _, a := range topNInt(settings.NumIntAttr, summary.integerAttributes) {
			if float64(a.count)/float64(summary.rowCount) < settings.IntThresholdPercent {
				continue
			}
			attrStr := traceql.NewScopedAttribute(scope, false, a.name).String()
			fmt.Printf("\"%s\" ", escapeString(attrStr))
		}
	}

	doStringSummary(s.spanSummary, traceql.AttributeScopeSpan)
	doStringSummary(s.resourceSummary, traceql.AttributeScopeResource)
	doStringSummary(s.eventSummary, traceql.AttributeScopeEvent)

	doIntSummary(s.spanSummary, traceql.AttributeScopeSpan)
	doIntSummary(s.resourceSummary, traceql.AttributeScopeResource)
	doIntSummary(s.eventSummary, traceql.AttributeScopeEvent)
}

func topN(n int, attrs map[string]*stringAttributeSummary) []*stringAttributeSummary {
	top := make([]*stringAttributeSummary, 0, len(attrs))
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

func topNInt(n int, attrs map[string]*integerAttributeSummary) []*integerAttributeSummary {
	top := make([]*integerAttributeSummary, 0, len(attrs))
	for _, attr := range attrs {
		top = append(top, attr)
	}
	sort.Slice(top, func(i, j int) bool {
		return top[i].count > top[j].count
	})
	if len(top) > n {
		top = top[:n]
	}
	return top
}

var _ parquetquery.GroupPredicate = (*attrStatsCollector)(nil)

type attrType int

const (
	attrTypeNull attrType = iota
	attrTypeString
	attrTypeInt
)

type attrStats struct {
	name     string
	value    string
	bytes    uint64
	intValue int64
	isArray  bool
	typ      attrType
}

var statsPool = sync.Pool{
	New: func() interface{} {
		return &attrStats{}
	},
}

func putStats(s *attrStats) {
	*s = attrStats{}
	statsPool.Put(s)
}

func getStats() *attrStats {
	return statsPool.Get().(*attrStats)
}

type attrStatsCollector struct{}

func (a attrStatsCollector) String() string {
	return "attrStatsCollector{}"
}

func (a attrStatsCollector) KeepGroup(res *parquetquery.IteratorResult) bool {
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
			stats.typ = attrTypeString
			stats.value = e.Value.String()
			stats.bytes += uint64(len(stats.value))
		case "int":
			stats.typ = attrTypeInt
			stats.intValue = e.Value.Int64()
		case "isArray":
			stats.isArray = e.Value.Boolean()
		}
	}

	res.Reset()
	if stats.typ == attrTypeNull {
		putStats(stats)
		return false
	}

	res.AppendOtherValue("stats", stats)
	return true
}
