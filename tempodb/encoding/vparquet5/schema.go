package vparquet5

import (
	"bytes"

	"github.com/golang/protobuf/jsonpb" //nolint:all //deprecated
	"github.com/parquet-go/parquet-go"

	"github.com/grafana/tempo/pkg/tempopb"
	v1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
	v1_resource "github.com/grafana/tempo/pkg/tempopb/resource/v1"
	v1_trace "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

// Label names for conversion b/n Proto <> Parquet
const (
	LabelRootSpanName    = "root.name"
	LabelRootServiceName = "root.service.name"

	LabelServiceName = "service.name"
	LabelCluster     = "cluster"
	LabelNamespace   = "namespace"
	LabelPod         = "pod"
	LabelContainer   = "container"

	LabelK8sClusterName   = "k8s.cluster.name"
	LabelK8sNamespaceName = "k8s.namespace.name"
	LabelK8sPodName       = "k8s.pod.name"
	LabelK8sContainerName = "k8s.container.name"

	LabelName                   = "name"
	LabelHTTPMethod             = "http.method"
	LabelHTTPUrl                = "http.url"
	LabelHTTPStatusCode         = "http.status_code"
	LabelStatusCode             = "status.code"
	LabelStatus                 = "status"
	LabelKind                   = "kind"
	LabelTraceQLRootServiceName = "rootServiceName"
	LabelTraceQLRootName        = "rootName"
	LabelTraceID                = "trace:id"
	LabelSpanID                 = "span:id"
)

// These definition levels match the schema below
const (
	DefinitionLevelTrace                          = 0
	DefinitionLevelServiceStats                   = 1
	DefinitionLevelResourceSpans                  = 1
	DefinitionLevelResourceAttrs                  = 2
	DefinitionLevelInstrumentationScope           = 2
	DefinitionLevelInstrumentationScopeAttrs      = 3
	DefinitionLevelResourceSpansILSSpan           = 3
	DefinitionLevelResourceSpansILSSpanAttrs      = 4
	DefinitionLevelResourceSpansILSSpanEvent      = 4
	DefinitionLevelResourceSpansILSSpanLink       = 4
	DefinitionLevelResourceSpansILSSpanEventAttrs = 5
	DefinitionLevelResourceSpansILSSpanLinkAttrs  = 5

	FieldResourceAttrKey       = "rs.list.element.Resource.Attrs.list.element.Key"
	FieldResourceAttrIsArray   = "rs.list.element.Resource.Attrs.list.element.IsArray"
	FieldResourceAttrVal       = "rs.list.element.Resource.Attrs.list.element.Value.list.element"
	FieldResourceAttrValInt    = "rs.list.element.Resource.Attrs.list.element.ValueInt.list.element"
	FieldResourceAttrValDouble = "rs.list.element.Resource.Attrs.list.element.ValueDouble.list.element"
	FieldResourceAttrValBool   = "rs.list.element.Resource.Attrs.list.element.ValueBool.list.element"

	FieldSpanAttrKey       = "rs.list.element.ss.list.element.Spans.list.element.Attrs.list.element.Key"
	FieldSpanAttrIsArray   = "rs.list.element.ss.list.element.Spans.list.element.Attrs.list.element.IsArray"
	FieldSpanAttrVal       = "rs.list.element.ss.list.element.Spans.list.element.Attrs.list.element.Value.list.element"
	FieldSpanAttrValInt    = "rs.list.element.ss.list.element.Spans.list.element.Attrs.list.element.ValueInt.list.element"
	FieldSpanAttrValDouble = "rs.list.element.ss.list.element.Spans.list.element.Attrs.list.element.ValueDouble.list.element"
	FieldSpanAttrValBool   = "rs.list.element.ss.list.element.Spans.list.element.Attrs.list.element.ValueBool.list.element"
)

const (
	// These are used to round span start times into  smaller integers that are high compressable.
	// Start value of 0 means the unix epoch. End time doesn't matter it just needs to be sufficiently high
	// to allow the math to work without overflow.
	roundingStart = uint64(0)
	roundingEnd   = uint64(0xF000000000000000)
)

var (
	jsonMarshaler = new(jsonpb.Marshaler)

	// todo: remove this when support for tag based search is removed. we only
	// need the below mappings for tag name search
	labelMappings = map[string]string{
		LabelRootSpanName:    "RootSpanName",
		LabelRootServiceName: "RootServiceName",
		LabelServiceName:     "rs.list.element.Resource.ServiceName",
		LabelName:            "rs.list.element.ss.list.element.Spans.list.element.Name",
		LabelStatusCode:      "rs.list.element.ss.list.element.Spans.list.element.StatusCode",
	}
	// the two below are used in tag name search. they only include
	// custom attributes that are mapped to parquet "special" columns
	// TODO there is just one column left in this mapping, can this be replaced?
	traceqlResourceLabelMappings = map[string]string{
		LabelServiceName: "rs.list.element.Resource.ServiceName",
	}

	parquetSchema = parquet.SchemaOf(&Trace{})
)

type Attribute struct {
	Key string `parquet:",snappy,dict"`

	IsArray          bool      `parquet:",snappy"`
	Value            []string  `parquet:",snappy,dict,list"`
	ValueInt         []int64   `parquet:",snappy,list"`
	ValueDouble      []float64 `parquet:",snappy,list"`
	ValueBool        []bool    `parquet:",snappy,list"`
	ValueUnsupported *string   `parquet:",snappy,optional"`
}

// DedicatedAttributes add spare columns to the schema that can be assigned to attributes at runtime.
type DedicatedAttributes struct {
	String01 []string `parquet:",snappy,optional,dict"`
	String02 []string `parquet:",snappy,optional,dict"`
	String03 []string `parquet:",snappy,optional,dict"`
	String04 []string `parquet:",snappy,optional,dict"`
	String05 []string `parquet:",snappy,optional,dict"`
	String06 []string `parquet:",snappy,optional,dict"`
	String07 []string `parquet:",snappy,optional,dict"`
	String08 []string `parquet:",snappy,optional,dict"`
	String09 []string `parquet:",snappy,optional,dict"`
	String10 []string `parquet:",snappy,optional,dict"`
	Int01    []int64  `parquet:",snappy,optional"`
	Int02    []int64  `parquet:",snappy,optional"`
	Int03    []int64  `parquet:",snappy,optional"`
	Int04    []int64  `parquet:",snappy,optional"`
	Int05    []int64  `parquet:",snappy,optional"`
}

func (da *DedicatedAttributes) Reset() {
	da.String01 = da.String01[:0]
	da.String02 = da.String02[:0]
	da.String03 = da.String03[:0]
	da.String04 = da.String04[:0]
	da.String05 = da.String05[:0]
	da.String06 = da.String06[:0]
	da.String07 = da.String07[:0]
	da.String08 = da.String08[:0]
	da.String09 = da.String09[:0]
	da.String10 = da.String10[:0]
	da.Int01 = da.Int01[:0]
	da.Int02 = da.Int02[:0]
	da.Int03 = da.Int03[:0]
	da.Int04 = da.Int04[:0]
	da.Int05 = da.Int05[:0]
}

type Event struct {
	TimeSinceStartNano     uint64      `parquet:",delta"`
	Name                   string      `parquet:",snappy,dict"`
	Attrs                  []Attribute `parquet:",list"`
	DroppedAttributesCount int32       `parquet:",snappy,delta"`
}

type Link struct {
	TraceID                []byte      `parquet:","`
	SpanID                 []byte      `parquet:","`
	TraceState             string      `parquet:",snappy"`
	Attrs                  []Attribute `parquet:",list"`
	DroppedAttributesCount int32       `parquet:",snappy,delta"`
}

// nolint:revive
// Ignore field naming warnings
type Span struct {
	// SpanID is []byte to save space. It doesn't need to be user-friendly
	// like trace ID, and []byte is half the size of string.
	SpanID                 []byte      `parquet:","`
	ParentSpanID           []byte      `parquet:","`
	ParentID               int32       `parquet:",delta"` // can be zero for non-root spans, use IsRoot to check for root spans
	NestedSetLeft          int32       `parquet:",delta"` // doubles as numeric ID and is used to fill ParentID of child spans
	NestedSetRight         int32       `parquet:",delta"`
	Name                   string      `parquet:",snappy,dict"`
	Kind                   int         `parquet:",delta"`
	TraceState             string      `parquet:",snappy"`
	StartTimeUnixNano      uint64      `parquet:",delta"`
	DurationNano           uint64      `parquet:",delta"`
	StatusCode             int         `parquet:",delta"`
	StatusMessage          string      `parquet:",snappy"`
	Attrs                  []Attribute `parquet:",list"`
	DroppedAttributesCount int32       `parquet:",snappy,delta"`
	Events                 []Event     `parquet:",list"`
	DroppedEventsCount     int32       `parquet:",snappy"`
	Links                  []Link      `parquet:",list"`
	DroppedLinksCount      int32       `parquet:",snappy"`

	// Dynamically assignable dedicated attribute columns
	DedicatedAttributes DedicatedAttributes `parquet:""`

	// Precomputed/Optimized values for metrics
	// These are stored as intervals from the unix epoch.
	// Interval 1 at 15s means "00:00:30".  These values can be represented in
	// fewer bits and compress better than rounding the full 64-bit nanos timestamp
	// to the same granularity. They can also be stored safely in uint32.
	StartTimeRounded15   uint32 `parquet:",delta"`
	StartTimeRounded60   uint32 `parquet:",delta"`
	StartTimeRounded300  uint32 `parquet:",delta"`
	StartTimeRounded3600 uint32 `parquet:",delta"`
}

func (s *Span) IsRoot() bool {
	return len(s.ParentSpanID) == 0
}

type InstrumentationScope struct {
	Name                   string      `parquet:",snappy,dict"`
	Version                string      `parquet:",snappy,dict"`
	Attrs                  []Attribute `parquet:",list"`
	DroppedAttributesCount int32       `parquet:",snappy,delta"`
}

type ScopeSpans struct {
	Scope     InstrumentationScope `parquet:""`
	Spans     []Span               `parquet:",list"`
	SpanCount int32                `parquet:",delta"`
}

type Resource struct {
	ServiceName string `parquet:",snappy,dict"`

	Attrs                  []Attribute `parquet:",list"`
	DroppedAttributesCount int32       `parquet:",snappy,delta"`

	// Dynamically assignable dedicated attribute columns
	DedicatedAttributes DedicatedAttributes `parquet:""`
}

type ResourceSpans struct {
	Resource   Resource     `parquet:""`
	ScopeSpans []ScopeSpans `parquet:"ss,list"`
}

type ServiceStats struct {
	SpanCount  uint32 `parquet:",delta"`
	ErrorCount uint32 `parquet:",delta"`
}

type Trace struct {
	// TraceID is a byte slice as it helps maintain the sort order of traces within a parquet file
	TraceID []byte `parquet:""`
	// TraceIDText is for better usability on downstream systems i.e: something other than Tempo is reading these files.
	// It will not be used as the primary traceID field within Tempo and is only helpful for debugging purposes.
	TraceIDText string `parquet:",snappy"`

	// Trace-level attributes for searching
	StartTimeUnixNano uint64                  `parquet:",delta"`
	EndTimeUnixNano   uint64                  `parquet:",delta"`
	DurationNano      uint64                  `parquet:",delta"`
	RootServiceName   string                  `parquet:",dict"`
	RootSpanName      string                  `parquet:",dict"`
	ServiceStats      map[string]ServiceStats `parquet:""`

	ResourceSpans []ResourceSpans `parquet:"rs,list"`
}

func attrToParquet(a *v1.KeyValue, p *Attribute) {
	p.Key = a.Key
	p.IsArray = false
	p.Value = p.Value[:0]
	p.ValueInt = p.ValueInt[:0]
	p.ValueDouble = p.ValueDouble[:0]
	p.ValueBool = p.ValueBool[:0]
	p.ValueUnsupported = nil

	switch v := a.GetValue().Value.(type) {
	case *v1.AnyValue_StringValue:
		p.Value = append(p.Value, v.StringValue)
	case *v1.AnyValue_IntValue:
		p.ValueInt = append(p.ValueInt, v.IntValue)
	case *v1.AnyValue_DoubleValue:
		p.ValueDouble = append(p.ValueDouble, v.DoubleValue)
	case *v1.AnyValue_BoolValue:
		p.ValueBool = append(p.ValueBool, v.BoolValue)
	case *v1.AnyValue_ArrayValue:
		p.IsArray = true
		if v.ArrayValue == nil || len(v.ArrayValue.Values) == 0 {
			return
		}
		switch v.ArrayValue.Values[0].Value.(type) {
		case *v1.AnyValue_StringValue:
			for _, e := range v.ArrayValue.Values {
				ev, ok := e.Value.(*v1.AnyValue_StringValue)
				if !ok {
					p.Value = p.Value[:0]
					attrToParquetTypeUnsupported(a, p)
					return
				}

				p.Value = append(p.Value, ev.StringValue)
			}
		case *v1.AnyValue_IntValue:
			for _, e := range v.ArrayValue.Values {
				ev, ok := e.Value.(*v1.AnyValue_IntValue)
				if !ok {
					p.ValueInt = p.ValueInt[:0]
					attrToParquetTypeUnsupported(a, p)
					return
				}

				p.ValueInt = append(p.ValueInt, ev.IntValue)
			}
		case *v1.AnyValue_DoubleValue:
			for _, e := range v.ArrayValue.Values {
				ev, ok := e.Value.(*v1.AnyValue_DoubleValue)
				if !ok {
					p.ValueDouble = p.ValueDouble[:0]
					attrToParquetTypeUnsupported(a, p)
					return
				}

				p.ValueDouble = append(p.ValueDouble, ev.DoubleValue)
			}
		case *v1.AnyValue_BoolValue:
			for _, e := range v.ArrayValue.Values {
				ev, ok := e.Value.(*v1.AnyValue_BoolValue)
				if !ok {
					p.ValueBool = p.ValueBool[:0]
					attrToParquetTypeUnsupported(a, p)
					return
				}

				p.ValueBool = append(p.ValueBool, ev.BoolValue)
			}
		default:
			attrToParquetTypeUnsupported(a, p)
		}
	default:
		attrToParquetTypeUnsupported(a, p)
	}
}

func attrToParquetTypeUnsupported(a *v1.KeyValue, p *Attribute) {
	jsonBytes := &bytes.Buffer{}
	_ = jsonMarshaler.Marshal(jsonBytes, a.Value) // deliberately marshalling a.Value because of AnyValue logic
	jsonStr := jsonBytes.String()
	p.ValueUnsupported = &jsonStr
	p.IsArray = false
}

// traceToParquet converts a tempopb.Trace to this schema's object model. Returns the new object and
// a bool indicating if it's a connected trace or not
func traceToParquet(meta *backend.BlockMeta, id common.ID, tr *tempopb.Trace, ot *Trace) (*Trace, bool) {
	// Dedicated attribute column assignments
	dedicatedResourceAttributes := dedicatedColumnsToColumnMapping(meta.DedicatedColumns, backend.DedicatedColumnScopeResource)
	dedicatedSpanAttributes := dedicatedColumnsToColumnMapping(meta.DedicatedColumns, backend.DedicatedColumnScopeSpan)

	return traceToParquetWithMapping(id, tr, ot, dedicatedResourceAttributes, dedicatedSpanAttributes)
}

func traceToParquetWithMapping(id common.ID, tr *tempopb.Trace, ot *Trace, dedicatedResourceAttributes, dedicatedSpanAttributes dedicatedColumnMapping) (*Trace, bool) {
	if ot == nil {
		ot = &Trace{}
	}

	ot.TraceIDText = util.TraceIDToHexString(id)
	ot.TraceID = util.PadTraceIDTo16Bytes(id)

	// Trace-level items
	traceStart := uint64(0)
	traceEnd := uint64(0)
	var rootSpan *v1_trace.Span
	var rootBatch *v1_trace.ResourceSpans

	ot.ResourceSpans = extendReuseSlice(len(tr.ResourceSpans), ot.ResourceSpans)
	for ib, b := range tr.ResourceSpans {
		ob := &ot.ResourceSpans[ib]
		// Clear out any existing fields in case they were set on the original
		ob.Resource.DroppedAttributesCount = 0
		ob.Resource.ServiceName = ""
		ob.Resource.DedicatedAttributes.Reset()

		if b.Resource != nil {
			ob.Resource.Attrs = extendReuseSlice(len(b.Resource.Attributes), ob.Resource.Attrs)
			ob.Resource.DroppedAttributesCount = int32(b.Resource.DroppedAttributesCount)

			attrCount := 0
			for _, a := range b.Resource.Attributes {
				var written bool
				if strVal, ok := a.Value.Value.(*v1.AnyValue_StringValue); ok && a.Key == LabelServiceName {
					ob.Resource.ServiceName = strVal.StringValue
					written = true
				}

				if !written {
					// Dynamically assigned dedicated resource attribute columns
					if spareColumn, exists := dedicatedResourceAttributes.get(a.Key); exists {
						written = spareColumn.writeValue(&ob.Resource.DedicatedAttributes, a.Value)
					}
				}

				if !written {
					// Other attributes put in generic columns
					attrToParquet(a, &ob.Resource.Attrs[attrCount])
					attrCount++
				}
			}
			ob.Resource.Attrs = ob.Resource.Attrs[:attrCount]
		}

		ob.ScopeSpans = extendReuseSlice(len(b.ScopeSpans), ob.ScopeSpans)
		for iils, ils := range b.ScopeSpans {
			oils := &ob.ScopeSpans[iils]
			instrumentationScopeToParquet(ils.Scope, &oils.Scope)

			oils.Spans = extendReuseSlice(len(ils.Spans), oils.Spans)
			oils.SpanCount = int32(len(ils.Spans))
			for is, s := range ils.Spans {
				ss := &oils.Spans[is]

				if traceStart == 0 || s.StartTimeUnixNano < traceStart {
					traceStart = s.StartTimeUnixNano
				}
				if s.EndTimeUnixNano > traceEnd {
					traceEnd = s.EndTimeUnixNano
				}
				var hasChildOfLink bool
				for _, spanLink := range s.Links {
					if bytes.Equal(s.TraceId, spanLink.TraceId) {
						for _, attr := range spanLink.GetAttributes() {
							if attr.Key == "opentracing.ref_type" && attr.GetValue().GetStringValue() == "child_of" {
								hasChildOfLink = true
								break
							}
						}
						if hasChildOfLink {
							break
						}
					}
				}

				if len(s.ParentSpanId) == 0 && !hasChildOfLink {
					rootSpan = s
					rootBatch = b
				}

				ss.Events = extendReuseSlice(len(s.Events), ss.Events)
				for ie, e := range s.Events {
					eventToParquet(e, &ss.Events[ie], s.StartTimeUnixNano)
				}

				// nested set values do not come from the proto, they are calculated
				// later. set all to 0
				ss.NestedSetLeft = 0
				ss.NestedSetRight = 0
				ss.ParentID = 0

				ss.SpanID = s.SpanId
				ss.ParentSpanID = s.ParentSpanId
				ss.Name = s.Name
				ss.Kind = int(s.Kind)
				ss.TraceState = s.TraceState
				if s.Status != nil {
					ss.StatusCode = int(s.Status.Code)
					ss.StatusMessage = s.Status.Message
				} else {
					ss.StatusCode = 0
					ss.StatusMessage = ""
				}
				ss.StartTimeUnixNano = s.StartTimeUnixNano
				if s.StartTimeUnixNano == 0 {
					ss.StartTimeRounded15 = 0
					ss.StartTimeRounded60 = 0
					ss.StartTimeRounded300 = 0
					ss.StartTimeRounded3600 = 0
				} else {
					ss.StartTimeRounded15 = uint32(intervalMapper15Seconds.Interval(s.StartTimeUnixNano))
					ss.StartTimeRounded60 = uint32(intervalMapper60Seconds.Interval(s.StartTimeUnixNano))
					ss.StartTimeRounded300 = uint32(intervalMapper300Seconds.Interval(s.StartTimeUnixNano))
					ss.StartTimeRounded3600 = uint32(intervalMapper3600Seconds.Interval(s.StartTimeUnixNano))
				}
				ss.DurationNano = s.EndTimeUnixNano - s.StartTimeUnixNano
				ss.DroppedAttributesCount = int32(s.DroppedAttributesCount)
				ss.DroppedEventsCount = int32(s.DroppedEventsCount)
				ss.DedicatedAttributes.Reset()

				ss.Links = extendReuseSlice(len(s.Links), ss.Links)
				for ie, e := range s.Links {
					linkToParquet(e, &ss.Links[ie])
				}

				ss.DroppedLinksCount = int32(s.DroppedLinksCount)

				ss.Attrs = extendReuseSlice(len(s.Attributes), ss.Attrs)
				attrCount := 0
				for _, a := range s.Attributes {
					written := false

					// Dynamically assigned dedicated span attribute columns
					if spareColumn, exists := dedicatedSpanAttributes.get(a.Key); exists {
						written = spareColumn.writeValue(&ss.DedicatedAttributes, a.Value)
					}

					if !written {
						// Other attributes put in generic columns
						attrToParquet(a, &ss.Attrs[attrCount])
						attrCount++
					}
				}
				ss.Attrs = ss.Attrs[:attrCount]
			}
		}
	}

	ot.StartTimeUnixNano = traceStart
	ot.EndTimeUnixNano = traceEnd
	ot.DurationNano = traceEnd - traceStart
	ot.RootSpanName = ""
	ot.RootServiceName = ""

	if rootSpan != nil && rootBatch != nil && rootBatch.Resource != nil {
		ot.RootSpanName = rootSpan.Name

		for _, a := range rootBatch.Resource.Attributes {
			if a.Key == LabelServiceName {
				ot.RootServiceName = a.Value.GetStringValue()
				break
			}
		}
	}

	return finalizeTrace(ot)
}

// finalizeTrace augments and optimized the trace by calculating service stats, nested set model bounds
// and removing redundant scope spans and resource spans. The function returns the modified trace as well
// as a boolean indicating whether the trace is a connected graph.
func finalizeTrace(trace *Trace) (*Trace, bool) {
	rebatchTrace(trace)
	return trace, assignNestedSetModelBoundsAndServiceStats(trace)
}

func instrumentationScopeToParquet(s *v1.InstrumentationScope, ss *InstrumentationScope) {
	if s == nil {
		ss.Name = ""
		ss.Version = ""
		ss.DroppedAttributesCount = 0
		ss.Attrs = ss.Attrs[:0]
		return
	}

	ss.Name = s.Name
	ss.Version = s.Version
	ss.DroppedAttributesCount = int32(s.DroppedAttributesCount)

	ss.Attrs = extendReuseSlice(len(s.Attributes), ss.Attrs)
	for i, a := range s.Attributes {
		attrToParquet(a, &ss.Attrs[i])
	}
}

func eventToParquet(e *v1_trace.Span_Event, ee *Event, spanStartTime uint64) {
	ee.Name = e.Name
	ee.TimeSinceStartNano = e.TimeUnixNano - spanStartTime
	ee.DroppedAttributesCount = int32(e.DroppedAttributesCount)

	ee.Attrs = extendReuseSlice(len(e.Attributes), ee.Attrs)
	for i, a := range e.Attributes {
		attrToParquet(a, &ee.Attrs[i])
	}
}

func linkToParquet(l *v1_trace.Span_Link, ll *Link) {
	ll.TraceID = l.TraceId
	ll.SpanID = l.SpanId
	ll.TraceState = l.TraceState
	ll.DroppedAttributesCount = int32(l.DroppedAttributesCount)

	ll.Attrs = extendReuseSlice(len(l.Attributes), ll.Attrs)
	for i, a := range l.Attributes {
		attrToParquet(a, &ll.Attrs[i])
	}
}

func parquetToProtoAttrs(parquetAttrs []Attribute) []*v1.KeyValue {
	var protoAttrs []*v1.KeyValue

	for _, attr := range parquetAttrs {
		var protoVal v1.AnyValue

		if !attr.IsArray {
			switch {
			case len(attr.Value) > 0:
				protoVal.Value = &v1.AnyValue_StringValue{StringValue: attr.Value[0]}
			case len(attr.ValueInt) > 0:
				protoVal.Value = &v1.AnyValue_IntValue{IntValue: attr.ValueInt[0]}
			case len(attr.ValueDouble) > 0:
				protoVal.Value = &v1.AnyValue_DoubleValue{DoubleValue: attr.ValueDouble[0]}
			case len(attr.ValueBool) > 0:
				protoVal.Value = &v1.AnyValue_BoolValue{BoolValue: attr.ValueBool[0]}
			case attr.ValueUnsupported != nil:
				_ = jsonpb.Unmarshal(bytes.NewBufferString(*attr.ValueUnsupported), &protoVal)
			default:
				continue
			}
		} else {
			switch {
			case len(attr.Value) > 0:
				values := make([]*v1.AnyValue, len(attr.Value))

				anyValues := make([]v1.AnyValue, len(values))
				strValues := make([]v1.AnyValue_StringValue, len(values))
				for i, v := range attr.Value {
					s := &strValues[i]
					s.StringValue = v
					values[i] = &anyValues[i]
					values[i].Value = s
				}

				protoVal.Value = &v1.AnyValue_ArrayValue{ArrayValue: &v1.ArrayValue{Values: values}}
			case len(attr.ValueInt) > 0:
				values := make([]*v1.AnyValue, len(attr.ValueInt))

				anyValues := make([]v1.AnyValue, len(values))
				intValues := make([]v1.AnyValue_IntValue, len(values))
				for i, v := range attr.ValueInt {
					n := &intValues[i]
					n.IntValue = v
					values[i] = &anyValues[i]
					values[i].Value = n
				}

				protoVal.Value = &v1.AnyValue_ArrayValue{ArrayValue: &v1.ArrayValue{Values: values}}
			case len(attr.ValueDouble) > 0:
				values := make([]*v1.AnyValue, len(attr.ValueDouble))

				anyValues := make([]v1.AnyValue, len(values))
				doubleValues := make([]v1.AnyValue_DoubleValue, len(values))
				for i, v := range attr.ValueDouble {
					n := &doubleValues[i]
					n.DoubleValue = v
					values[i] = &anyValues[i]
					values[i].Value = n
				}

				protoVal.Value = &v1.AnyValue_ArrayValue{ArrayValue: &v1.ArrayValue{Values: values}}
			case len(attr.ValueBool) > 0:
				values := make([]*v1.AnyValue, len(attr.ValueBool))

				anyValues := make([]v1.AnyValue, len(values))
				boolValues := make([]v1.AnyValue_BoolValue, len(values))
				for i, v := range attr.ValueBool {
					n := &boolValues[i]
					n.BoolValue = v
					values[i] = &anyValues[i]
					values[i].Value = n
				}

				protoVal.Value = &v1.AnyValue_ArrayValue{ArrayValue: &v1.ArrayValue{Values: values}}
			default:
				protoVal.Value = &v1.AnyValue_ArrayValue{ArrayValue: &v1.ArrayValue{Values: []*v1.AnyValue{}}}
			}
		}

		protoAttrs = append(protoAttrs, &v1.KeyValue{
			Key:   attr.Key,
			Value: &protoVal,
		})
	}

	return protoAttrs
}

func parquetToProtoInstrumentationScope(parquetScope *InstrumentationScope) *v1.InstrumentationScope {
	scope := v1.InstrumentationScope{
		Name:                   parquetScope.Name,
		Version:                parquetScope.Version,
		DroppedAttributesCount: uint32(parquetScope.DroppedAttributesCount),
	}

	if len(parquetScope.Attrs) > 0 {
		scope.Attributes = parquetToProtoAttrs(parquetScope.Attrs)
	}

	return &scope
}

func parquetToProtoLinks(parquetLinks []Link) []*v1_trace.Span_Link {
	var protoLinks []*v1_trace.Span_Link

	if len(parquetLinks) > 0 {
		protoLinks = make([]*v1_trace.Span_Link, 0, len(parquetLinks))
		for _, l := range parquetLinks {
			protoLink := &v1_trace.Span_Link{
				TraceId:                l.TraceID,
				SpanId:                 l.SpanID,
				TraceState:             l.TraceState,
				DroppedAttributesCount: uint32(l.DroppedAttributesCount),
				Attributes:             nil,
			}

			if len(l.Attrs) > 0 {
				protoLink.Attributes = parquetToProtoAttrs(l.Attrs)
			}

			protoLinks = append(protoLinks, protoLink)
		}
	}

	return protoLinks
}

func parquetToProtoEvents(parquetEvents []Event, spanStartTimeNano uint64) []*v1_trace.Span_Event {
	var protoEvents []*v1_trace.Span_Event

	if len(parquetEvents) > 0 {
		protoEvents = make([]*v1_trace.Span_Event, 0, len(parquetEvents))

		for _, e := range parquetEvents {

			protoEvent := &v1_trace.Span_Event{
				TimeUnixNano:           e.TimeSinceStartNano + spanStartTimeNano,
				Name:                   e.Name,
				Attributes:             nil,
				DroppedAttributesCount: uint32(e.DroppedAttributesCount),
			}

			if len(e.Attrs) > 0 {
				protoEvent.Attributes = parquetToProtoAttrs(e.Attrs)
			}

			protoEvents = append(protoEvents, protoEvent)
		}
	}

	return protoEvents
}

func ParquetTraceToTempopbTrace(meta *backend.BlockMeta, parquetTrace *Trace) *tempopb.Trace {
	protoTrace := &tempopb.Trace{}
	protoTrace.ResourceSpans = make([]*v1_trace.ResourceSpans, 0, len(parquetTrace.ResourceSpans))

	// dedicated attribute column assignments
	dedicatedResourceAttributes := dedicatedColumnsToColumnMapping(meta.DedicatedColumns, backend.DedicatedColumnScopeResource)
	dedicatedSpanAttributes := dedicatedColumnsToColumnMapping(meta.DedicatedColumns, backend.DedicatedColumnScopeSpan)

	for _, rs := range parquetTrace.ResourceSpans {
		protoBatch := &v1_trace.ResourceSpans{}
		resAttrs := parquetToProtoAttrs(rs.Resource.Attrs)
		protoBatch.Resource = &v1_resource.Resource{
			Attributes:             resAttrs,
			DroppedAttributesCount: uint32(rs.Resource.DroppedAttributesCount),
		}

		// dynamically assigned dedicated resource attribute columns
		for attr, col := range dedicatedResourceAttributes.items() {
			val := col.readValue(&rs.Resource.DedicatedAttributes)
			if val != nil {
				protoBatch.Resource.Attributes = append(protoBatch.Resource.Attributes, &v1.KeyValue{
					Key:   attr,
					Value: val,
				})
			}
		}

		// known resource attributes
		if rs.Resource.ServiceName != "" {
			protoBatch.Resource.Attributes = append(protoBatch.Resource.Attributes, &v1.KeyValue{
				Key: LabelServiceName,
				Value: &v1.AnyValue{
					Value: &v1.AnyValue_StringValue{
						StringValue: rs.Resource.ServiceName,
					},
				},
			})
		}

		protoBatch.ScopeSpans = make([]*v1_trace.ScopeSpans, 0, len(rs.ScopeSpans))

		for _, scopeSpan := range rs.ScopeSpans {
			protoSS := &v1_trace.ScopeSpans{
				Scope: parquetToProtoInstrumentationScope(&scopeSpan.Scope),
			}

			protoSS.Spans = make([]*v1_trace.Span, 0, len(scopeSpan.Spans))
			for _, span := range scopeSpan.Spans {

				spanAttr := parquetToProtoAttrs(span.Attrs)
				protoSpan := &v1_trace.Span{
					TraceId:           parquetTrace.TraceID,
					SpanId:            span.SpanID,
					TraceState:        span.TraceState,
					Name:              span.Name,
					Kind:              v1_trace.Span_SpanKind(span.Kind),
					ParentSpanId:      span.ParentSpanID,
					StartTimeUnixNano: span.StartTimeUnixNano,
					EndTimeUnixNano:   span.StartTimeUnixNano + span.DurationNano,
					Status: &v1_trace.Status{
						Message: span.StatusMessage,
						Code:    v1_trace.Status_StatusCode(span.StatusCode),
					},
					Attributes:             spanAttr,
					DroppedAttributesCount: uint32(span.DroppedAttributesCount),
					Events:                 parquetToProtoEvents(span.Events, span.StartTimeUnixNano),
					DroppedEventsCount:     uint32(span.DroppedEventsCount),
					DroppedLinksCount:      uint32(span.DroppedLinksCount),
				}

				protoSpan.Links = parquetToProtoLinks(span.Links)

				// dynamically assigned dedicated resource attribute columns
				for attr, col := range dedicatedSpanAttributes.items() {
					val := col.readValue(&span.DedicatedAttributes)
					if val != nil {
						protoSpan.Attributes = append(protoSpan.Attributes, &v1.KeyValue{
							Key:   attr,
							Value: val,
						})
					}
				}

				protoSS.Spans = append(protoSS.Spans, protoSpan)
			}

			protoBatch.ScopeSpans = append(protoBatch.ScopeSpans, protoSS)
		}
		protoTrace.ResourceSpans = append(protoTrace.ResourceSpans, protoBatch)
	}

	return protoTrace
}

func extendReuseSlice[T any](sz int, in []T) []T {
	if cap(in) >= sz {
		// slice is large enough
		return in[:sz]
	}

	// append until we're large enough
	in = in[:cap(in)]
	return append(in, make([]T, sz-len(in))...)
}
