package main

import (
	"cmp"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/fnv"
	"io"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"text/tabwriter"
	"unsafe"

	"github.com/parquet-go/parquet-go"

	pq "github.com/grafana/tempo/pkg/parquetquery"
	"github.com/grafana/tempo/tempodb/backend"
	vp4 "github.com/grafana/tempo/tempodb/encoding/vparquet4"
)

// attrIndexCmd represents a command to generate attribute indices from a parquet block.
//
// This command is highly experimental and meant to facilitate experimentation with different
// kinds of indexes.
type attrIndexCmd struct {
	In            string   `arg:"" help:"The input parquet block to read from."`
	AddIntrinsics bool     `help:"Add some intrinsic attributes to the index like name, kind, status, etc."`
	IndexTypes    []string `enum:"rows,codes" help:"The type of index to generate (rows | codes | rows,codes)" default:"rows,codes"`
	dedicatedRes  []string `kong:"-"`
	dedicatedSpan []string `kong:"-"`
}

func (cmd *attrIndexCmd) Run(_ *globalOptions) error {
	cmd.In = getPathToBlockDir(cmd.In)
	fmt.Printf("Analyzing parquet block from %s\n", cmd.In)

	meta, err := readBlockMeta(cmd.In)
	if err != nil {
		return err
	}
	if meta.Version != vp4.VersionString {
		return fmt.Errorf("unsupported parquet version %s", meta.Version)
	}

	cmd.readDedicatedAttributes(meta)

	stats, err := cmd.collectAttributeStats()
	if err != nil {
		return err
	}
	stats.printStats()

	if len(cmd.IndexTypes) == 0 || len(cmd.IndexTypes) == 2 {
		fmt.Println("Generating combined index with inverted index and key/value codes")

		index := generateCombinedIndex(stats)
		err = writeAttributeIndex(cmd.In, index)
	} else if len(cmd.IndexTypes) == 1 {
		if cmd.IndexTypes[0] == "rows" {
			fmt.Println("Generating inverted index with rows")

			index := generateRowsIndex(stats)
			err = writeAttributeIndex(cmd.In, index)
		} else if cmd.IndexTypes[0] == "codes" {
			fmt.Println("Generating index with key/value codes")

			index := generateCodesIndex(stats)
			err = writeAttributeIndex(cmd.In, index)
		}
	}
	if err != nil {
		return err
	}

	fmt.Printf("\nSuccessfully generated attribute index in %s/index.parquet\n", cmd.In)
	return nil
}

func (cmd *attrIndexCmd) readDedicatedAttributes(meta *backend.BlockMeta) {
	for _, ded := range meta.DedicatedColumns {
		switch ded.Scope {
		case backend.DedicatedColumnScopeResource:
			cmd.dedicatedRes = append(cmd.dedicatedRes, ded.Name)
		case backend.DedicatedColumnScopeSpan:
			cmd.dedicatedSpan = append(cmd.dedicatedSpan, ded.Name)
		}
	}
}

func (cmd *attrIndexCmd) collectAttributeStats() (*fileStats, error) {
	stats := fileStats{
		Attributes: make(map[string]attributeInfo, 200),
	}

	in, pf, err := openParquetFile(cmd.In)
	if err != nil {
		return nil, err
	}
	defer in.Close()

	reader := parquet.NewGenericReader[vp4.Trace](pf)
	defer reader.Close()

	var (
		traceBuffer = make([]vp4.Trace, 1024)
		readCount   int
	)

	for {
		readCount, err = reader.Read(traceBuffer)
		if err != nil {
			if !errors.Is(err, io.EOF) {
				return nil, err
			}
			break
		}
		runtime.GC() // after reading the new traces to the buffer GC can free the old ones

		if readCount > 0 {
			cmd.collectAttributeStatsForTraces(&stats, traceBuffer[:readCount])
		}
	}
	if readCount > 0 {
		cmd.collectAttributeStatsForTraces(&stats, traceBuffer[:readCount])
	}

	return &stats, nil
}

func (cmd *attrIndexCmd) collectAttributeStatsForTraces(stats *fileStats, traces []vp4.Trace) {
	row := pq.EmptyRowNumber()
	stats.Traces += len(traces)
	for _, tr := range traces {
		stats.Resources += len(tr.ResourceSpans)
		row.Next(0, 0, 3)

		for _, rs := range tr.ResourceSpans {
			row.Next(1, 1, 3)

			res := rs.Resource
			stats.addAttributes(row, scopeResource, res.Attrs)
			stats.addDedicatedAttributes(row, scopeResource, cmd.dedicatedRes, &res.DedicatedAttributes)

			stats.addAttribute(row, scopeResource, "service.name", res.ServiceName)
			stats.addAttribute(row, scopeResource, "cluster", res.Cluster)
			stats.addAttribute(row, scopeResource, "namespace", res.Namespace)
			stats.addAttribute(row, scopeResource, "pod", res.Pod)
			stats.addAttribute(row, scopeResource, "container", res.Container)
			stats.addAttribute(row, scopeResource, "k8s.cluster.name", res.K8sClusterName)
			stats.addAttribute(row, scopeResource, "k8s.namespace.name", res.K8sNamespaceName)
			stats.addAttribute(row, scopeResource, "k8s.pod.name", res.K8sPodName)
			stats.addAttribute(row, scopeResource, "k8s.container.name", res.K8sContainerName)
			for _, ss := range rs.ScopeSpans {
				row.Next(2, 2, 3)

				scope := ss.Scope
				stats.Spans += len(ss.Spans)

				stats.addAttributes(row, scopeScope, scope.Attrs)
				if cmd.AddIntrinsics {
					// adding scope to distinguish from span.name
					stats.addAttribute(row, scopeScope, "scope.name", scope.Name)
					stats.addAttribute(row, scopeScope, "version", scope.Version)
				}
				for _, sp := range ss.Spans {
					row.Next(3, 3, 3)

					stats.Events += len(sp.Events)
					stats.Links += len(sp.Links)

					stats.addAttributes(row, scopeSpan, sp.Attrs)
					stats.addDedicatedAttributes(row, scopeSpan, cmd.dedicatedSpan, &sp.DedicatedAttributes)
					stats.addAttribute(row, scopeSpan, "http.method", sp.HttpMethod)
					stats.addAttribute(row, scopeSpan, "http.url", sp.HttpUrl)
					stats.addAttribute(row, scopeSpan, "http.status_code", sp.HttpStatusCode)
					if cmd.AddIntrinsics {
						stats.addAttribute(row, scopeSpan, "name", sp.Name)
						stats.addAttribute(row, scopeSpan, "kind", sp.Kind)
						stats.addAttribute(row, scopeSpan, "status.code", sp.StatusCode)
						stats.addAttribute(row, scopeSpan, "status.message", sp.StatusMessage)
					}
					for _, ev := range sp.Events {
						stats.addAttributes(row, scopeEvent, ev.Attrs)
						if cmd.AddIntrinsics {
							// adding scope to distinguish from span.name
							stats.addAttribute(row, scopeEvent, "event.name", ev.Name)
						}
					}
					for _, ln := range sp.Links {
						stats.addAttributes(row, scopeLink, ln.Attrs)
					}
				}
			}
		}
	}
}

func (fs *fileStats) printStats() {
	fmt.Println("File stats:")

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', tabwriter.DiscardEmptyColumns)
	tmpl := "%s\t%d\n"
	_, _ = fmt.Fprintf(w, tmpl, "Traces", fs.Traces)
	_, _ = fmt.Fprintf(w, tmpl, "Resources", fs.Resources)
	_, _ = fmt.Fprintf(w, tmpl, "Spans", fs.Spans)
	_, _ = fmt.Fprintf(w, tmpl, "Events", fs.Events)
	_, _ = fmt.Fprintf(w, tmpl, "Links", fs.Links)
	_, _ = fmt.Fprintf(w, tmpl, "Arrays", fs.Arrays)
	_ = w.Flush()

	// sort attributes by scope and count
	attrs := make([]attributeInfo, 0, len(fs.Attributes))
	for _, attr := range fs.Attributes {
		attrs = append(attrs, attr)
	}
	sort.Slice(attrs, func(i, j int) bool {
		if n := cmp.Compare(attrs[i].ScopeMask, attrs[j].ScopeMask); n != 0 {
			return n < 0
		}
		return attrs[i].Count > attrs[j].Count
	})

	fmt.Println("\nAttribute stats:")
	w = tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', tabwriter.DiscardEmptyColumns)
	_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", "Name", "Scopes", "Count", "Cardinality")
	tmpl = "%s\t%s\t%d\t%d\n"
	for _, attr := range attrs {
		_, _ = fmt.Fprintf(w, tmpl, attr.Key, attr.ScopeMask.String(), attr.Count, len(attr.ValuesString)+len(attr.ValuesInt)+len(attr.ValuesFloat)+len(attr.ValuesBool))
	}
	_ = w.Flush()

	fmt.Printf("\n\n")
}

func generateCombinedIndex(stats *fileStats) []indexedAttrCombined {
	var (
		index   = make([]indexedAttrCombined, 0, len(stats.Attributes))
		keyCode int64
	)

	for _, attr := range stats.Attributes {
		keyCode++

		a := indexedAttrCombined{
			Key:       attr.Key,
			KeyCode:   keyCode,
			ScopeMask: attr.ScopeMask,
		}

		if len(attr.ValuesString) > 0 {
			a.ValuesString = make([]indexedValCombined[string], 0, len(attr.ValuesString))

			for _, v := range attr.ValuesString {
				a.ValuesString = append(a.ValuesString, indexedValCombined[string]{
					Value:      v.Value,
					RowNumbers: v.RowNumbers,
				})
			}

			sort.Slice(a.ValuesString, func(i, j int) bool {
				return cmpSlice(a.ValuesString[i].Value, a.ValuesString[j].Value) < 0
			})

			var valueCode int64
			for i := range a.ValuesString {
				valueCode++
				a.ValuesString[i].ValueCode = valueCode
			}
		}

		if len(attr.ValuesInt) > 0 {
			a.ValuesInt = make([]indexedValCombined[int64], 0, len(attr.ValuesInt))

			for _, v := range attr.ValuesInt {
				a.ValuesInt = append(a.ValuesInt, indexedValCombined[int64]{
					Value:      v.Value,
					RowNumbers: v.RowNumbers,
				})
			}

			sort.Slice(a.ValuesInt, func(i, j int) bool {
				return cmpSlice(a.ValuesInt[i].Value, a.ValuesInt[j].Value) < 0
			})

			var valueCode int64
			for i := range a.ValuesInt {
				valueCode++
				a.ValuesInt[i].ValueCode = valueCode
			}
		}

		if len(attr.ValuesFloat) > 0 {
			a.ValuesFloat = make([]indexedValCombined[float64], 0, len(attr.ValuesFloat))

			for _, v := range attr.ValuesFloat {
				a.ValuesFloat = append(a.ValuesFloat, indexedValCombined[float64]{
					Value:      v.Value,
					RowNumbers: v.RowNumbers,
				})
			}

			sort.Slice(a.ValuesFloat, func(i, j int) bool {
				return cmpSlice(a.ValuesFloat[i].Value, a.ValuesFloat[j].Value) < 0
			})

			var valueCode int64
			for i := range a.ValuesFloat {
				valueCode++
				a.ValuesFloat[i].ValueCode = valueCode
			}
		}

		if len(attr.ValuesBool) > 0 {
			a.ValuesBool = make([]indexedValCombined[bool], 0, len(attr.ValuesBool))

			for _, v := range attr.ValuesBool {
				a.ValuesBool = append(a.ValuesBool, indexedValCombined[bool]{
					Value:      v.Value,
					RowNumbers: v.RowNumbers,
				})
			}

			sort.Slice(a.ValuesBool, func(i, j int) bool {
				return cmpSliceBool(a.ValuesBool[i].Value, a.ValuesBool[j].Value) < 0
			})

			var valueCode int64
			for i := range a.ValuesBool {
				valueCode++
				a.ValuesBool[i].ValueCode = valueCode
			}
		}

		index = append(index, a)
	}

	return index
}

func generateRowsIndex(stats *fileStats) []indexedAttrRows {
	var (
		index   = make([]indexedAttrRows, 0, len(stats.Attributes))
		keyCode int64
	)

	for _, attr := range stats.Attributes {
		keyCode++

		a := indexedAttrRows{
			Key:       attr.Key,
			ScopeMask: attr.ScopeMask,
		}

		if len(attr.ValuesString) > 0 {
			a.ValuesString = make([]indexedValRows[string], 0, len(attr.ValuesString))

			for _, v := range attr.ValuesString {
				a.ValuesString = append(a.ValuesString, indexedValRows[string](v))
			}

			sort.Slice(a.ValuesString, func(i, j int) bool {
				return cmpSlice(a.ValuesString[i].Value, a.ValuesString[j].Value) < 0
			})
		}

		if len(attr.ValuesInt) > 0 {
			a.ValuesInt = make([]indexedValRows[int64], 0, len(attr.ValuesInt))

			for _, v := range attr.ValuesInt {
				a.ValuesInt = append(a.ValuesInt, indexedValRows[int64](v))
			}

			sort.Slice(a.ValuesInt, func(i, j int) bool {
				return cmpSlice(a.ValuesInt[i].Value, a.ValuesInt[j].Value) < 0
			})
		}

		if len(attr.ValuesFloat) > 0 {
			a.ValuesFloat = make([]indexedValRows[float64], 0, len(attr.ValuesFloat))

			for _, v := range attr.ValuesFloat {
				a.ValuesFloat = append(a.ValuesFloat, indexedValRows[float64](v))
			}

			sort.Slice(a.ValuesFloat, func(i, j int) bool {
				return cmpSlice(a.ValuesFloat[i].Value, a.ValuesFloat[j].Value) < 0
			})
		}

		if len(attr.ValuesBool) > 0 {
			a.ValuesBool = make([]indexedValRows[bool], 0, len(attr.ValuesBool))

			for _, v := range attr.ValuesBool {
				a.ValuesBool = append(a.ValuesBool, indexedValRows[bool](v))
			}

			sort.Slice(a.ValuesBool, func(i, j int) bool {
				return cmpSliceBool(a.ValuesBool[i].Value, a.ValuesBool[j].Value) < 0
			})
		}

		index = append(index, a)
	}

	return index
}

func generateCodesIndex(stats *fileStats) []indexedAttrCodes {
	var (
		index   = make([]indexedAttrCodes, 0, len(stats.Attributes))
		keyCode int64
	)

	for _, attr := range stats.Attributes {
		keyCode++

		a := indexedAttrCodes{
			Key:       attr.Key,
			KeyCode:   keyCode,
			ScopeMask: attr.ScopeMask,
		}

		if len(attr.ValuesString) > 0 {
			a.ValuesString = make([]indexedValCodes[string], 0, len(attr.ValuesString))

			for _, v := range attr.ValuesString {
				a.ValuesString = append(a.ValuesString, indexedValCodes[string]{
					Value: v.Value,
				})
			}

			sort.Slice(a.ValuesString, func(i, j int) bool {
				return cmpSlice(a.ValuesString[i].Value, a.ValuesString[j].Value) < 0
			})

			var valueCode int64
			for i := range a.ValuesString {
				valueCode++
				a.ValuesString[i].ValueCode = valueCode
			}
		}

		if len(attr.ValuesInt) > 0 {
			a.ValuesInt = make([]indexedValCodes[int64], 0, len(attr.ValuesInt))

			for _, v := range attr.ValuesInt {
				a.ValuesInt = append(a.ValuesInt, indexedValCodes[int64]{
					Value: v.Value,
				})
			}

			sort.Slice(a.ValuesInt, func(i, j int) bool {
				return cmpSlice(a.ValuesInt[i].Value, a.ValuesInt[j].Value) < 0
			})

			var valueCode int64
			for i := range a.ValuesInt {
				valueCode++
				a.ValuesInt[i].ValueCode = valueCode
			}
		}

		if len(attr.ValuesFloat) > 0 {
			a.ValuesFloat = make([]indexedValCodes[float64], 0, len(attr.ValuesFloat))

			for _, v := range attr.ValuesFloat {
				a.ValuesFloat = append(a.ValuesFloat, indexedValCodes[float64]{
					Value: v.Value,
				})
			}

			sort.Slice(a.ValuesFloat, func(i, j int) bool {
				return cmpSlice(a.ValuesFloat[i].Value, a.ValuesFloat[j].Value) < 0
			})

			var valueCode int64
			for i := range a.ValuesFloat {
				valueCode++
				a.ValuesFloat[i].ValueCode = valueCode
			}
		}

		if len(attr.ValuesBool) > 0 {
			a.ValuesBool = make([]indexedValCodes[bool], 0, len(attr.ValuesBool))

			for _, v := range attr.ValuesBool {
				a.ValuesBool = append(a.ValuesBool, indexedValCodes[bool]{
					Value: v.Value,
				})
			}

			sort.Slice(a.ValuesBool, func(i, j int) bool {
				return cmpSliceBool(a.ValuesBool[i].Value, a.ValuesBool[j].Value) < 0
			})

			var valueCode int64
			for i := range a.ValuesBool {
				valueCode++
				a.ValuesBool[i].ValueCode = valueCode
			}
		}

		index = append(index, a)
	}

	return index
}

type indexedAttrCombined struct {
	Key          string                        `parquet:",snappy"`
	KeyCode      int64                         `parquet:",snappy,delta"`
	ScopeMask    scopeMask                     `parquet:",snappy,delta"`
	ValuesString []indexedValCombined[string]  `parquet:",list"`
	ValuesInt    []indexedValCombined[int64]   `parquet:",list"`
	ValuesFloat  []indexedValCombined[float64] `parquet:",list"`
	ValuesBool   []indexedValCombined[bool]    `parquet:",list"`
}

type indexedValCombined[T comparable] struct {
	Value      []T   `parquet:",snappy"`
	ValueCode  int64 `parquet:",snappy,delta"`
	RowNumbers []rowNumberCols
}

type indexedAttrRows struct {
	Key          string                    `parquet:",snappy"`
	ScopeMask    scopeMask                 `parquet:",snappy,delta"`
	ValuesString []indexedValRows[string]  `parquet:",list"`
	ValuesInt    []indexedValRows[int64]   `parquet:",list"`
	ValuesFloat  []indexedValRows[float64] `parquet:",list"`
	ValuesBool   []indexedValRows[bool]    `parquet:",list"`
}

type indexedValRows[T comparable] struct {
	Value      []T `parquet:",snappy"`
	RowNumbers []rowNumberCols
}

type indexedAttrCodes struct {
	Key          string                     `parquet:",snappy"`
	KeyCode      int64                      `parquet:",snappy,delta"`
	ScopeMask    scopeMask                  `parquet:",snappy,delta"`
	ValuesString []indexedValCodes[string]  `parquet:",list"`
	ValuesInt    []indexedValCodes[int64]   `parquet:",list"`
	ValuesFloat  []indexedValCodes[float64] `parquet:",list"`
	ValuesBool   []indexedValCodes[bool]    `parquet:",list"`
}

type indexedValCodes[T comparable] struct {
	Value     []T   `parquet:",snappy"`
	ValueCode int64 `parquet:",snappy,delta"`
}

type rowNumberCols struct {
	Lvl01 int64 `parquet:",snappy,delta"`
	Lvl02 int64 `parquet:",snappy,delta"`
	Lvl03 int64 `parquet:",snappy,delta"`
	Lvl04 int64 `parquet:",snappy,delta"`
}

func writeAttributeIndex[T any](in string, index []T) error {
	stat, err := os.Stat(filepath.Join(in, "data.parquet"))
	if err != nil {
		return err
	}

	out, err := os.OpenFile(filepath.Join(in, "index.parquet"), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, stat.Mode())
	if err != nil {
		return err
	}
	defer out.Close()

	writer := parquet.NewGenericWriter[T](out)
	defer writer.Close()

	writeCount := 0
	for writeCount < len(index) {
		n, err := writer.Write(index[writeCount:])
		if err != nil {
			return err
		}
		writeCount += n
	}

	err = writer.Flush()
	if err != nil {
		return err
	}

	return nil
}

type fileStats struct {
	Traces     int
	Resources  int
	Spans      int
	Events     int
	Links      int
	Arrays     int
	Attributes map[string]attributeInfo
}

type attributeInfo struct {
	Key          string
	ScopeMask    scopeMask
	Count        int64
	ValuesString map[uint64]valueInfo[string]
	ValuesInt    map[uint64]valueInfo[int64]
	ValuesFloat  map[uint64]valueInfo[float64]
	ValuesBool   map[uint64]valueInfo[bool]
}

type valueInfo[T comparable] struct {
	Value      []T
	RowNumbers []rowNumberCols
}

func (fs *fileStats) addAttributes(row pq.RowNumber, scope scopeMask, attrs []vp4.Attribute) {
	for _, attr := range attrs {
		if attr.IsArray {
			fs.addAttribute(row, scope, attr.Key, attr.Value)
			fs.Arrays++
		} else if len(attr.Value) > 0 {
			fs.addAttribute(row, scope, attr.Key, attr.Value[0])
		} else if len(attr.ValueInt) > 0 {
			fs.addAttribute(row, scope, attr.Key, attr.ValueInt[0])
		} else if len(attr.ValueDouble) > 0 {
			fs.addAttribute(row, scope, attr.Key, attr.ValueDouble[0])
		} else if len(attr.ValueBool) > 0 {
			fs.addAttribute(row, scope, attr.Key, attr.ValueBool[0])
		}
	}
}

func (fs *fileStats) addDedicatedAttributes(row pq.RowNumber, scope scopeMask, columns []string, attrs *vp4.DedicatedAttributes) {
	if attrs == nil {
		return
	}

	if attrs.String01 != nil && len(columns) > 0 {
		fs.addAttribute(row, scope, columns[0], attrs.String01)
	}
	if attrs.String02 != nil && len(columns) > 1 {
		fs.addAttribute(row, scope, columns[1], attrs.String02)
	}
	if attrs.String03 != nil && len(columns) > 2 {
		fs.addAttribute(row, scope, columns[2], attrs.String03)
	}
	if attrs.String04 != nil && len(columns) > 3 {
		fs.addAttribute(row, scope, columns[3], attrs.String04)
	}
	if attrs.String05 != nil && len(columns) > 4 {
		fs.addAttribute(row, scope, columns[4], attrs.String05)
	}
	if attrs.String06 != nil && len(columns) > 5 {
		fs.addAttribute(row, scope, columns[5], attrs.String06)
	}
	if attrs.String07 != nil && len(columns) > 6 {
		fs.addAttribute(row, scope, columns[6], attrs.String07)
	}
	if attrs.String08 != nil && len(columns) > 7 {
		fs.addAttribute(row, scope, columns[7], attrs.String08)
	}
	if attrs.String09 != nil && len(columns) > 8 {
		fs.addAttribute(row, scope, columns[8], attrs.String09)
	}
	if attrs.String10 != nil && len(columns) > 9 {
		fs.addAttribute(row, scope, columns[9], attrs.String10)
	}
}

func (fs *fileStats) addAttribute(row pq.RowNumber, scope scopeMask, key string, value any) {
	attr, ok := fs.Attributes[key]
	if !ok {
		attr = attributeInfo{
			Key:       key,
			ScopeMask: scope,
		}
	}

	attr.Count++
	attr.ScopeMask.Add(scope)

	switch value := value.(type) {
	case string, *string, []string:
		s := toSlice[string](value)
		if len(s) == 0 {
			return
		}
		if attr.ValuesString == nil {
			attr.ValuesString = make(map[uint64]valueInfo[string], 10)
		}

		sum := fnvStrings(s)
		info, ok := attr.ValuesString[sum]
		if !ok {
			info = valueInfo[string]{
				Value:      s,
				RowNumbers: make([]rowNumberCols, 0, 1),
			}
		}

		info.RowNumbers = append(info.RowNumbers, toRowNumberCols(row))
		attr.ValuesString[sum] = info
	case int64, *int64, []int64:
		s := toSlice[int64](value)
		if len(s) == 0 {
			return
		}
		if attr.ValuesInt == nil {
			attr.ValuesInt = make(map[uint64]valueInfo[int64], 10)
		}

		sum := fnvInts(s)
		v, ok := attr.ValuesInt[sum]
		if !ok {
			v = valueInfo[int64]{
				Value:      s,
				RowNumbers: make([]rowNumberCols, 0, 1),
			}
		}
		v.RowNumbers = append(v.RowNumbers, toRowNumberCols(row))
		attr.ValuesInt[sum] = v
	case float64, *float64, []float64:
		s := toSlice[float64](value)
		if len(s) == 0 {
			return
		}
		if attr.ValuesFloat == nil {
			attr.ValuesFloat = make(map[uint64]valueInfo[float64])
		}

		sum := fnvFloats(s)
		v, ok := attr.ValuesFloat[sum]
		if !ok {
			v = valueInfo[float64]{
				Value:      s,
				RowNumbers: make([]rowNumberCols, 0, 1),
			}
		}
		v.RowNumbers = append(v.RowNumbers, toRowNumberCols(row))
		attr.ValuesFloat[sum] = v
	case bool, *bool, []bool:
		s := toSlice[bool](value)
		if len(s) == 0 {
			return
		}
		if attr.ValuesBool == nil {
			attr.ValuesBool = make(map[uint64]valueInfo[bool])
		}

		sum := fnvBools(s)
		v, ok := attr.ValuesBool[sum]
		if !ok {
			v = valueInfo[bool]{
				Value:      s,
				RowNumbers: make([]rowNumberCols, 0, 1),
			}
		}
		v.RowNumbers = append(v.RowNumbers, toRowNumberCols(row))
		attr.ValuesBool[sum] = v
	}

	fs.Attributes[key] = attr
}

type scopeMask int64

const (
	scopeResource = 1 << iota
	scopeScope
	scopeSpan
	scopeEvent
	scopeLink
)

var scopeMap = [5]struct {
	mask scopeMask
	name string
}{
	{scopeResource, "resource"},
	{scopeScope, "scope"},
	{scopeSpan, "span"},
	{scopeEvent, "event"},
	{scopeLink, "link"},
}

func (s *scopeMask) Add(o scopeMask) {
	*s |= o
}

func (s *scopeMask) Has(o scopeMask) bool {
	return *s&o != 0
}

func (s *scopeMask) Scopes() []string {
	scopes := make([]string, 0, 5)
	for _, scope := range scopeMap {
		if s.Has(scope.mask) {
			scopes = append(scopes, scope.name)
		}
	}
	return scopes
}

func (s *scopeMask) String() string {
	return strings.Join(s.Scopes(), " ")
}

func toRowNumberCols(row pq.RowNumber) rowNumberCols {
	return rowNumberCols{
		Lvl01: int64(row[0]),
		Lvl02: int64(row[1]),
		Lvl03: int64(row[2]),
		Lvl04: int64(row[3]),
	}
}

func toSlice[T any](val any) []T {
	switch v := val.(type) {
	case []T:
		return v
	case T:
		return []T{v}
	case *T:
		if v == nil {
			return nil
		}
		return []T{*v}
	default:
		panic(fmt.Sprintf("unexpected type %T", v))
	}
}

func fnvStrings(values []string) uint64 {
	h := fnv.New64a()
	for _, v := range values {
		_, _ = h.Write(unsafe.Slice(unsafe.StringData(v), len(v)))
	}
	return h.Sum64()
}

func fnvInts(values []int64) uint64 {
	h := fnv.New64a()
	var buf [8]byte
	for _, v := range values {
		binary.LittleEndian.PutUint64(buf[:], uint64(v))
		_, _ = h.Write(buf[:])
	}
	return h.Sum64()
}

func fnvFloats(values []float64) uint64 {
	h := fnv.New64a()
	var buf [8]byte
	for _, v := range values {
		binary.LittleEndian.PutUint64(buf[:], math.Float64bits(v))
		_, _ = h.Write(buf[:])
	}
	return h.Sum64()
}

func fnvBools(values []bool) uint64 {
	h := fnv.New64a()
	var buf [1]byte
	for _, v := range values {
		if v {
			buf[0] = 1
		} else {
			buf[0] = 0
		}
		_, _ = h.Write(buf[:])
	}
	return h.Sum64()
}

func cmpSlice[T cmp.Ordered](a, b []T) int {
	for i := range min(len(a), len(b)) {
		if n := cmp.Compare(a[i], b[i]); n != 0 {
			return n
		}
	}
	return cmp.Compare(len(a), len(b))
}

func cmpSliceBool(a, b []bool) int {
	for i := range min(len(a), len(b)) {
		if !a[i] && b[i] {
			return -1
		}
		if a[i] && !b[i] {
			return 1
		}
	}
	return cmp.Compare(len(a), len(b))
}
