package main

import (
	"cmp"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/parquet-go/parquet-go"

	pq "github.com/grafana/tempo/pkg/parquetquery"
	"github.com/grafana/tempo/tempodb/backend"
	vp4 "github.com/grafana/tempo/tempodb/encoding/vparquet4"
)

type attrIndexCmd struct {
	In            string   `arg:"" help:"The input parquet block to read from."`
	dedicatedRes  []string `kong:"-"`
	dedicatedSpan []string `kong:"-"`
}

func (cmd *attrIndexCmd) Run(ctx *globalOptions) error {
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

	cmd.printAttrStats(stats)

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
		Attributes: make(map[string]*attributeInfo, 100),
	}

	in, pf, err := openParquetFile(cmd.In)
	if err != nil {
		return nil, err
	}
	defer in.Close()

	reader := parquet.NewGenericReader[vp4.Trace](pf)
	traceBuffer := make([]vp4.Trace, 10240)

	var readCount int
	for {
		readCount, err = reader.Read(traceBuffer)
		if err != nil {
			if !errors.Is(err, io.EOF) {
				return nil, err
			}
			break
		}
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
		// TODO row.Next()
		stats.Resources += len(tr.ResourceSpans)
		for _, rs := range tr.ResourceSpans {
			// TODO row.Next()
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
				// TODO row.Next()
				scope := ss.Scope
				stats.Spans += len(ss.Spans)
				stats.addAttributes(row, scopeScope, scope.Attrs)
				// TODO maybe add scope.name and scope.version
				for _, sp := range ss.Spans {
					// TODO row.Next()
					stats.Events += len(sp.Events)
					stats.Links += len(sp.Links)
					stats.addAttributes(row, scopeSpan, sp.Attrs)
					stats.addDedicatedAttributes(row, scopeSpan, cmd.dedicatedSpan, &sp.DedicatedAttributes)

					stats.addAttribute(row, scopeSpan, "http.method", sp.HttpMethod)
					stats.addAttribute(row, scopeSpan, "http.url", sp.HttpUrl)
					stats.addAttribute(row, scopeSpan, "http.status_code", sp.HttpStatusCode)
					// TODO maybe add span.kind, span.name, span.status.code, span.status
					for _, ev := range sp.Events {
						stats.addAttributes(row, scopeEvent, ev.Attrs)
						// TODO maybe add event.name
					}
					for _, ln := range sp.Links {
						stats.addAttributes(row, scopeLink, ln.Attrs)
					}
				}
			}
		}
	}
}

func (cmd *attrIndexCmd) printAttrStats(stats *fileStats) {
	fmt.Println("File stats:")

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', tabwriter.DiscardEmptyColumns)
	tmpl := "%s\t%d\n"
	_, _ = fmt.Fprintf(w, tmpl, "Traces", stats.Traces)
	_, _ = fmt.Fprintf(w, tmpl, "Resources", stats.Resources)
	_, _ = fmt.Fprintf(w, tmpl, "Spans", stats.Spans)
	_, _ = fmt.Fprintf(w, tmpl, "Events", stats.Events)
	_, _ = fmt.Fprintf(w, tmpl, "Links", stats.Links)
	_ = w.Flush()

	// sort attributes by scope and count
	attrs := make([]*attributeInfo, 0, len(stats.Attributes))
	for _, attr := range stats.Attributes {
		attrs = append(attrs, attr)
	}
	sort.Slice(attrs, func(i, j int) bool {
		if n := cmp.Compare(attrs[i].ScopeMask, attrs[j].ScopeMask); n != 0 {
			return n < 0
		}
		return attrs[i].Count > attrs[j].Count
	})

	fmt.Println("\nAttributes:")
	w = tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', tabwriter.DiscardEmptyColumns)
	_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", "Name", "Scopes", "Count", "Cardinality")
	tmpl = "%s\t%s\t%d\t%d\n"
	for _, attr := range attrs {
		_, _ = fmt.Fprintf(w, tmpl, attr.Key, attr.ScopeMask.String(), attr.Count, len(attr.ValuesString)+len(attr.ValuesInt64)+len(attr.ValuesFloat64)+len(attr.ValuesBool))
	}
	_ = w.Flush()
}

func (cmd *attrIndexCmd) writeAttributeIndex(stats *fileStats) error {
	return nil
}

type indexedAttribute struct {
	Key           string
	KeyCode       int64
	ScopeMask     scopeMask
	ValuesString  []indexedValued[string]
	ValuesInt64   []indexedValued[int64]
	ValuesFloat64 []indexedValued[float64]
	ValuesBool    []indexedValued[bool]
}

type indexedValued[T comparable] struct {
	Value      T
	ValueCode  int64
	RowNumbers []pq.RowNumber
}
type attributeInfo struct {
	Key           string
	ScopeMask     scopeMask
	Count         int64
	ValuesString  map[string]*valueInfo[string]
	ValuesInt64   map[int64]*valueInfo[int64]
	ValuesFloat64 map[float64]*valueInfo[float64]
	ValuesBool    map[bool]*valueInfo[bool]
}

type valueInfo[T comparable] struct {
	Value      T
	RowNumbers []pq.RowNumber
}

type fileStats struct {
	Traces     int
	Resources  int
	Spans      int
	Events     int
	Links      int
	Attributes map[string]*attributeInfo
}

func (fs *fileStats) addAttributes(row pq.RowNumber, scope scopeMask, attrs []vp4.Attribute) {
	for _, attr := range attrs {
		if attr.IsArray {
			fs.addAttribute(row, scope, attr.Key, attr.Value)
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
	value = dereferenceAny(value)
	if value == nil {
		return
	}

	attr, ok := fs.Attributes[key]
	if !ok {
		attr = &attributeInfo{
			Key:           key,
			ScopeMask:     scope,
			ValuesString:  make(map[string]*valueInfo[string]),
			ValuesInt64:   make(map[int64]*valueInfo[int64]),
			ValuesFloat64: make(map[float64]*valueInfo[float64]),
			ValuesBool:    make(map[bool]*valueInfo[bool]),
		}
		fs.Attributes[key] = attr
	}

	attr.Count++
	attr.ScopeMask.Add(scope)

	switch value := value.(type) {
	case string:
		v, ok := attr.ValuesString[value]
		if !ok {
			v = &valueInfo[string]{
				Value:      value,
				RowNumbers: make([]pq.RowNumber, 0, 1000),
			}
			attr.ValuesString[value] = v
		}

		v.RowNumbers = append(v.RowNumbers, row)
	case int64:
		v, ok := attr.ValuesInt64[value]
		if !ok {
			v = &valueInfo[int64]{
				Value:      value,
				RowNumbers: make([]pq.RowNumber, 0, 1000),
			}
			attr.ValuesInt64[value] = v
		}
		v.RowNumbers = append(v.RowNumbers, row)
	case float64:
		v, ok := attr.ValuesFloat64[value]
		if !ok {
			v = &valueInfo[float64]{
				Value:      value,
				RowNumbers: make([]pq.RowNumber, 0, 1000),
			}
			attr.ValuesFloat64[value] = v
		}
		v.RowNumbers = append(v.RowNumbers, row)
	case bool:
		v, ok := attr.ValuesBool[value]
		if !ok {
			v = &valueInfo[bool]{
				Value:      value,
				RowNumbers: make([]pq.RowNumber, 0, 1000),
			}
			attr.ValuesBool[value] = v
		}
		v.RowNumbers = append(v.RowNumbers, row)
	}
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

func dereferenceAny(val any) any {
	switch v := val.(type) {
	case string, int64, float64, bool:
		return v
	case *string:
		if v == nil {
			return nil
		}
		return *v
	case *int64:
		if v == nil {
			return nil
		}
		return *v
	case *float64:
		if v == nil {
			return nil
		}
		return *v
	case *bool:
		if v == nil {
			return nil
		}
		return *v
	default:
		if value := reflect.ValueOf(val); value.Kind() == reflect.Ptr {
			if value.IsNil() {
				return nil
			}
			return value.Elem().Interface()
		}
		return v
	}
}
