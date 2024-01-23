package vparquet3

import (
	"bytes"

	"github.com/grafana/tempo/tempodb/backend"

	"github.com/golang/protobuf/jsonpb" //nolint:all //deprecated
	"github.com/grafana/tempo/pkg/tempopb"
	v1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
	v1_resource "github.com/grafana/tempo/pkg/tempopb/resource/v1"
	v1_trace "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/util"
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
)

// These definition levels match the schema below
const (
	DefinitionLevelTrace                     = 0
	DefinitionLevelServiceStats              = 1
	DefinitionLevelResourceSpans             = 1
	DefinitionLevelResourceAttrs             = 2
	DefinitionLevelResourceSpansILSSpan      = 3
	DefinitionLevelResourceSpansILSSpanAttrs = 4

	FieldResourceAttrKey       = "rs.list.element.Resource.Attrs.list.element.Key"
	FieldResourceAttrVal       = "rs.list.element.Resource.Attrs.list.element.Value.list.element"
	FieldResourceAttrValInt    = "rs.list.element.Resource.Attrs.list.element.ValueInt.list.element"
	FieldResourceAttrValDouble = "rs.list.element.Resource.Attrs.list.element.ValueDouble.list.element"
	FieldResourceAttrValBool   = "rs.list.element.Resource.Attrs.list.element.ValueBool.list.element"

	FieldSpanAttrKey       = "rs.list.element.ss.list.element.Spans.list.element.Attrs.list.element.Key"
	FieldSpanAttrVal       = "rs.list.element.ss.list.element.Spans.list.element.Attrs.list.element.Value.list.element"
	FieldSpanAttrValInt    = "rs.list.element.ss.list.element.Spans.list.element.Attrs.list.element.ValueInt.list.element"
	FieldSpanAttrValDouble = "rs.list.element.ss.list.element.Spans.list.element.Attrs.list.element.ValueDouble.list.element"
	FieldSpanAttrValBool   = "rs.list.element.ss.list.element.Spans.list.element.Attrs.list.element.ValueBool.list.element"
)

type AttrType int32

const (
	attrTypeNotSupported AttrType = iota
	attrTypeString
	attrTypeInt
	attrTypeDouble
	attrTypeBool
	attrTypeStringArray
	attrTypeIntArray
	attrTypeDoubleArray
	attrTypeBoolArray
)

var (
	jsonMarshaler = new(jsonpb.Marshaler)

	// todo: remove this when support for tag based search is removed. we only
	// need the below mappings for tag name search
	labelMappings = map[string]string{
		LabelRootSpanName:     "RootSpanName",
		LabelRootServiceName:  "RootServiceName",
		LabelServiceName:      "rs.list.element.Resource.ServiceName",
		LabelCluster:          "rs.list.element.Resource.Cluster",
		LabelNamespace:        "rs.list.element.Resource.Namespace",
		LabelPod:              "rs.list.element.Resource.Pod",
		LabelContainer:        "rs.list.element.Resource.Container",
		LabelK8sClusterName:   "rs.list.element.Resource.K8sClusterName",
		LabelK8sNamespaceName: "rs.list.element.Resource.K8sNamespaceName",
		LabelK8sPodName:       "rs.list.element.Resource.K8sPodName",
		LabelK8sContainerName: "rs.list.element.Resource.K8sContainerName",
		LabelName:             "rs.list.element.ss.list.element.Spans.list.element.Name",
		LabelHTTPMethod:       "rs.list.element.ss.list.element.Spans.list.element.HttpMethod",
		LabelHTTPUrl:          "rs.list.element.ss.list.element.Spans.list.element.HttpUrl",
		LabelHTTPStatusCode:   "rs.list.element.ss.list.element.Spans.list.element.HttpStatusCode",
		LabelStatusCode:       "rs.list.element.ss.list.element.Spans.list.element.StatusCode",
	}
	// the two below are used in tag name search. they only include
	//  custom attributes that are mapped to parquet "special" columns
	traceqlResourceLabelMappings = map[string]string{
		LabelServiceName:      "rs.list.element.Resource.ServiceName",
		LabelCluster:          "rs.list.element.Resource.Cluster",
		LabelNamespace:        "rs.list.element.Resource.Namespace",
		LabelPod:              "rs.list.element.Resource.Pod",
		LabelContainer:        "rs.list.element.Resource.Container",
		LabelK8sClusterName:   "rs.list.element.Resource.K8sClusterName",
		LabelK8sNamespaceName: "rs.list.element.Resource.K8sNamespaceName",
		LabelK8sPodName:       "rs.list.element.Resource.K8sPodName",
		LabelK8sContainerName: "rs.list.element.Resource.K8sContainerName",
	}
	traceqlSpanLabelMappings = map[string]string{
		LabelHTTPMethod:     "rs.list.element.ss.list.element.Spans.list.element.HttpMethod",
		LabelHTTPUrl:        "rs.list.element.ss.list.element.Spans.list.element.HttpUrl",
		LabelHTTPStatusCode: "rs.list.element.ss.list.element.Spans.list.element.HttpStatusCode",
	}
)

// droppedAttrCounter represents an entity that can count dropped attributes
type droppedAttrCounter interface {
	addDroppedAttr(n int32)
}

type Attribute struct {
	Key string `parquet:",snappy,dict"`

	ValueType    AttrType  `parquet:",snappy,delta"`
	Value        []string  `parquet:",snappy,dict,list"`
	ValueInt     []int64   `parquet:",snappy,list"`
	ValueDouble  []float64 `parquet:",snappy,list"`
	ValueBool    []bool    `parquet:",snappy,list"`
	ValueDropped string    `parquet:",snappy,optional"`
}

// DedicatedAttributes add spare columns to the schema that can be assigned to attributes at runtime.
type DedicatedAttributes struct {
	String01 *string `parquet:",snappy,optional,dict"`
	String02 *string `parquet:",snappy,optional,dict"`
	String03 *string `parquet:",snappy,optional,dict"`
	String04 *string `parquet:",snappy,optional,dict"`
	String05 *string `parquet:",snappy,optional,dict"`
	String06 *string `parquet:",snappy,optional,dict"`
	String07 *string `parquet:",snappy,optional,dict"`
	String08 *string `parquet:",snappy,optional,dict"`
	String09 *string `parquet:",snappy,optional,dict"`
	String10 *string `parquet:",snappy,optional,dict"`
}

type Event struct {
	TimeSinceStartNano     uint64      `parquet:",delta"`
	Name                   string      `parquet:",snappy,dic"`
	Attrs                  []Attribute `parquet:",list"`
	DroppedAttributesCount int32       `parquet:",snappy,delta"`
}

func (e *Event) addDroppedAttr(n int32) {
	e.DroppedAttributesCount += n
}

type Link struct {
	TraceID                []byte      `parquet:","`
	SpanID                 []byte      `parquet:","`
	TraceState             string      `parquet:",snappy"`
	Attrs                  []Attribute `parquet:",list"`
	DroppedAttributesCount int32       `parquet:",snappy,delta"`
}

func (l *Link) addDroppedAttr(n int32) {
	l.DroppedAttributesCount += n
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

	// Static dedicated attribute columns
	HttpMethod     *string `parquet:",snappy,optional,dict"`
	HttpUrl        *string `parquet:",snappy,optional,dict"`
	HttpStatusCode *int64  `parquet:",snappy,optional"`

	// Dynamically assignable dedicated attribute columns
	DedicatedAttributes DedicatedAttributes `parquet:""`
}

func (s *Span) addDroppedAttr(n int32) {
	s.DroppedAttributesCount += n
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

func (is *InstrumentationScope) addDroppedAttr(n int32) {
	is.DroppedAttributesCount += n
}

type ScopeSpans struct {
	Scope InstrumentationScope `parquet:""`
	Spans []Span               `parquet:",list"`
}

type Resource struct {
	Attrs                  []Attribute `parquet:",list"`
	DroppedAttributesCount int32       `parquet:",snappy,delta"`

	// Static dedicated attribute columns
	ServiceName      string  `parquet:",snappy,dict"`
	Cluster          *string `parquet:",snappy,optional,dict"`
	Namespace        *string `parquet:",snappy,optional,dict"`
	Pod              *string `parquet:",snappy,optional,dict"`
	Container        *string `parquet:",snappy,optional,dict"`
	K8sClusterName   *string `parquet:",snappy,optional,dict"`
	K8sNamespaceName *string `parquet:",snappy,optional,dict"`
	K8sPodName       *string `parquet:",snappy,optional,dict"`
	K8sContainerName *string `parquet:",snappy,optional,dict"`

	// Dynamically assignable dedicated attribute columns
	DedicatedAttributes DedicatedAttributes `parquet:""`
}

func (r *Resource) addDroppedAttr(n int32) {
	r.DroppedAttributesCount += n
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

func attrToParquet(a *v1.KeyValue, p *Attribute, counter droppedAttrCounter) {
	p.Key = a.Key
	p.ValueType = 0
	p.Value = p.Value[:0]
	p.ValueInt = p.ValueInt[:0]
	p.ValueDouble = p.ValueDouble[:0]
	p.ValueBool = p.ValueBool[:0]
	p.ValueDropped = ""

	switch v := a.GetValue().Value.(type) {
	case *v1.AnyValue_StringValue:
		p.Value = append(p.Value, v.StringValue)
		p.ValueType = attrTypeString
	case *v1.AnyValue_IntValue:
		p.ValueInt = append(p.ValueInt, v.IntValue)
		p.ValueType = attrTypeInt
	case *v1.AnyValue_DoubleValue:
		p.ValueDouble = append(p.ValueDouble, v.DoubleValue)
		p.ValueType = attrTypeDouble
	case *v1.AnyValue_BoolValue:
		p.ValueBool = append(p.ValueBool, v.BoolValue)
		p.ValueType = attrTypeBool
	case *v1.AnyValue_ArrayValue:
		if v.ArrayValue == nil || len(v.ArrayValue.Values) == 0 {
			p.ValueType = attrTypeStringArray
			return
		}
		switch v.ArrayValue.Values[0].Value.(type) {
		case *v1.AnyValue_StringValue:
			for _, e := range v.ArrayValue.Values {
				ev, ok := e.Value.(*v1.AnyValue_StringValue)
				if !ok {
					p.Value = p.Value[:0]
					attrToParquetTypeUnsupported(a, p, counter)
					return
				}

				p.Value = append(p.Value, ev.StringValue)
				p.ValueType = attrTypeStringArray
			}
		case *v1.AnyValue_IntValue:
			for _, e := range v.ArrayValue.Values {
				ev, ok := e.Value.(*v1.AnyValue_IntValue)
				if !ok {
					p.ValueInt = p.ValueInt[:0]
					attrToParquetTypeUnsupported(a, p, counter)
					return
				}

				p.ValueInt = append(p.ValueInt, ev.IntValue)
				p.ValueType = attrTypeIntArray
			}
		case *v1.AnyValue_DoubleValue:
			for _, e := range v.ArrayValue.Values {
				ev, ok := e.Value.(*v1.AnyValue_DoubleValue)
				if !ok {
					p.ValueDouble = p.ValueDouble[:0]
					attrToParquetTypeUnsupported(a, p, counter)
					return
				}

				p.ValueDouble = append(p.ValueDouble, ev.DoubleValue)
				p.ValueType = attrTypeDoubleArray
			}
		case *v1.AnyValue_BoolValue:
			for _, e := range v.ArrayValue.Values {
				ev, ok := e.Value.(*v1.AnyValue_BoolValue)
				if !ok {
					p.ValueBool = p.ValueBool[:0]
					attrToParquetTypeUnsupported(a, p, counter)
					return
				}

				p.ValueBool = append(p.ValueBool, ev.BoolValue)
				p.ValueType = attrTypeBoolArray
			}
		default:
			attrToParquetTypeUnsupported(a, p, counter)
		}
	default:
		attrToParquetTypeUnsupported(a, p, counter)
	}
}

func attrToParquetTypeUnsupported(a *v1.KeyValue, p *Attribute, counter droppedAttrCounter) {
	jsonBytes := &bytes.Buffer{}
	_ = jsonMarshaler.Marshal(jsonBytes, a.Value) // deliberately marshalling a.Value because of AnyValue logic
	p.ValueDropped = jsonBytes.String()
	p.ValueType = attrTypeNotSupported
	counter.addDroppedAttr(1)
}

// traceToParquet converts a tempopb.Trace to this schema's object model. Returns the new object and
// a bool indicating if it's a connected trace or not
func traceToParquet(meta *backend.BlockMeta, id common.ID, tr *tempopb.Trace, ot *Trace) (*Trace, bool) {
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

	// Dedicated attribute column assignments
	dedicatedResourceAttributes := dedicatedColumnsToColumnMapping(meta.DedicatedColumns, backend.DedicatedColumnScopeResource)
	dedicatedSpanAttributes := dedicatedColumnsToColumnMapping(meta.DedicatedColumns, backend.DedicatedColumnScopeSpan)

	ot.ResourceSpans = extendReuseSlice(len(tr.Batches), ot.ResourceSpans)
	for ib, b := range tr.Batches {
		ob := &ot.ResourceSpans[ib]
		// Clear out any existing fields in case they were set on the original
		ob.Resource.DroppedAttributesCount = 0
		ob.Resource.ServiceName = ""
		ob.Resource.Cluster = nil
		ob.Resource.Namespace = nil
		ob.Resource.Pod = nil
		ob.Resource.Container = nil
		ob.Resource.K8sClusterName = nil
		ob.Resource.K8sNamespaceName = nil
		ob.Resource.K8sPodName = nil
		ob.Resource.K8sContainerName = nil
		ob.Resource.DedicatedAttributes = DedicatedAttributes{}

		if b.Resource != nil {
			ob.Resource.Attrs = extendReuseSlice(len(b.Resource.Attributes), ob.Resource.Attrs)
			ob.Resource.DroppedAttributesCount = int32(b.Resource.DroppedAttributesCount)

			attrCount := 0
			for _, a := range b.Resource.Attributes {
				strVal, ok := a.Value.Value.(*v1.AnyValue_StringValue)
				written := ok
				if ok {
					switch a.Key {
					case LabelServiceName:
						ob.Resource.ServiceName = strVal.StringValue
					case LabelCluster:
						ob.Resource.Cluster = &strVal.StringValue
					case LabelNamespace:
						ob.Resource.Namespace = &strVal.StringValue
					case LabelPod:
						ob.Resource.Pod = &strVal.StringValue
					case LabelContainer:
						ob.Resource.Container = &strVal.StringValue

					case LabelK8sClusterName:
						ob.Resource.K8sClusterName = &strVal.StringValue
					case LabelK8sNamespaceName:
						ob.Resource.K8sNamespaceName = &strVal.StringValue
					case LabelK8sPodName:
						ob.Resource.K8sPodName = &strVal.StringValue
					case LabelK8sContainerName:
						ob.Resource.K8sContainerName = &strVal.StringValue
					default:
						written = false
					}
				}

				if !written {
					// Dynamically assigned dedicated resource attribute columns
					if spareColumn, exists := dedicatedResourceAttributes.get(a.Key); exists {
						written = spareColumn.writeValue(&ob.Resource.DedicatedAttributes, a.Value)
					}
				}

				if !written {
					// Other attributes put in generic columns
					attrToParquet(a, &ob.Resource.Attrs[attrCount], &ob.Resource)
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
			for is, s := range ils.Spans {
				ss := &oils.Spans[is]

				if traceStart == 0 || s.StartTimeUnixNano < traceStart {
					traceStart = s.StartTimeUnixNano
				}
				if s.EndTimeUnixNano > traceEnd {
					traceEnd = s.EndTimeUnixNano
				}
				if len(s.ParentSpanId) == 0 {
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
				ss.DurationNano = s.EndTimeUnixNano - s.StartTimeUnixNano
				ss.DroppedAttributesCount = int32(s.DroppedAttributesCount)
				ss.DroppedEventsCount = int32(s.DroppedEventsCount)
				ss.HttpMethod = nil
				ss.HttpUrl = nil
				ss.HttpStatusCode = nil
				ss.DedicatedAttributes = DedicatedAttributes{}

				ss.Links = extendReuseSlice(len(s.Links), ss.Links)
				for ie, e := range s.Links {
					linkToParquet(e, &ss.Links[ie])
				}

				ss.DroppedLinksCount = int32(s.DroppedLinksCount)

				ss.Attrs = extendReuseSlice(len(s.Attributes), ss.Attrs)
				attrCount := 0
				for _, a := range s.Attributes {
					written := false

					switch a.Key {
					case LabelHTTPMethod:
						strVal, ok := a.Value.Value.(*v1.AnyValue_StringValue)
						if ok {
							ss.HttpMethod = &strVal.StringValue
							written = true
						}
					case LabelHTTPUrl:
						strVal, ok := a.Value.Value.(*v1.AnyValue_StringValue)
						if ok {
							ss.HttpUrl = &strVal.StringValue
							written = true
						}
					case LabelHTTPStatusCode:
						intVal, ok := a.Value.Value.(*v1.AnyValue_IntValue)
						if ok {
							ss.HttpStatusCode = &intVal.IntValue
							written = true
						}
					}

					if !written {
						// Dynamically assigned dedicated span attribute columns
						if spareColumn, exists := dedicatedSpanAttributes.get(a.Key); exists {
							written = spareColumn.writeValue(&ss.DedicatedAttributes, a.Value)
						}
					}

					if !written {
						// Other attributes put in generic columns
						attrToParquet(a, &ss.Attrs[attrCount], ss)
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

	// Calculate service meta information/statistics per trace
	ot.ServiceStats = map[string]ServiceStats{}
	for _, res := range ot.ResourceSpans {
		stats := ot.ServiceStats[res.Resource.ServiceName]

		for _, ss := range res.ScopeSpans {
			stats.SpanCount += uint32(len(ss.Spans))
			for _, s := range ss.Spans {
				if s.StatusCode == int(v1_trace.Status_STATUS_CODE_ERROR) {
					stats.ErrorCount++
				}
			}
		}

		ot.ServiceStats[res.Resource.ServiceName] = stats
	}

	return ot, assignNestedSetModelBounds(ot)
}

func instrumentationScopeToParquet(s *v1.InstrumentationScope, ss *InstrumentationScope) {
	if s == nil {
		ss.Name = ""
		ss.Version = ""
	} else {
		ss.Name = s.Name
		ss.Version = s.Version
	}

	// TODO: handle attributes correctly once they are added to the proto
	ss.Attrs = ss.Attrs[:0]
	ss.DroppedAttributesCount = 0
}

func eventToParquet(e *v1_trace.Span_Event, ee *Event, spanStartTime uint64) {
	ee.Name = e.Name
	ee.TimeSinceStartNano = e.TimeUnixNano - spanStartTime
	ee.DroppedAttributesCount = int32(e.DroppedAttributesCount)

	ee.Attrs = extendReuseSlice(len(e.Attributes), ee.Attrs)
	for i, a := range e.Attributes {
		attrToParquet(a, &ee.Attrs[i], ee)
	}
}

func linkToParquet(l *v1_trace.Span_Link, ll *Link) {
	ll.TraceID = l.TraceId
	ll.SpanID = l.SpanId
	ll.TraceState = l.TraceState
	ll.DroppedAttributesCount = int32(l.DroppedAttributesCount)

	ll.Attrs = extendReuseSlice(len(l.Attributes), ll.Attrs)
	for i, a := range l.Attributes {
		attrToParquet(a, &ll.Attrs[i], ll)
	}
}

func parquetToProtoAttrs(parquetAttrs []Attribute, counter droppedAttrCounter, includeDroppedAttr bool) []*v1.KeyValue {
	var protoAttrs []*v1.KeyValue

	for _, attr := range parquetAttrs {
		var protoVal v1.AnyValue

		switch attr.ValueType {
		case attrTypeString:
			var v v1.AnyValue_StringValue
			if len(attr.Value) > 0 {
				v.StringValue = attr.Value[0]
			}
			protoVal.Value = &v
		case attrTypeInt:
			var v v1.AnyValue_IntValue
			if len(attr.ValueInt) > 0 {
				v.IntValue = attr.ValueInt[0]
			}
			protoVal.Value = &v
		case attrTypeDouble:
			var v v1.AnyValue_DoubleValue
			if len(attr.ValueDouble) > 0 {
				v.DoubleValue = attr.ValueDouble[0]
			}
			protoVal.Value = &v
		case attrTypeBool:
			var v v1.AnyValue_BoolValue
			if len(attr.ValueBool) > 0 {
				v.BoolValue = attr.ValueBool[0]
			}
			protoVal.Value = &v
		case attrTypeStringArray:
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
		case attrTypeIntArray:
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
		case attrTypeDoubleArray:
			values := make([]*v1.AnyValue, len(attr.ValueDouble))

			anyValues := make([]v1.AnyValue, len(values))
			intValues := make([]v1.AnyValue_DoubleValue, len(values))
			for i, v := range attr.ValueDouble {
				n := &intValues[i]
				n.DoubleValue = v
				values[i] = &anyValues[i]
				values[i].Value = n
			}

			protoVal.Value = &v1.AnyValue_ArrayValue{ArrayValue: &v1.ArrayValue{Values: values}}
		case attrTypeBoolArray:
			values := make([]*v1.AnyValue, len(attr.ValueBool))

			anyValues := make([]v1.AnyValue, len(values))
			intValues := make([]v1.AnyValue_BoolValue, len(values))
			for i, v := range attr.ValueBool {
				n := &intValues[i]
				n.BoolValue = v
				values[i] = &anyValues[i]
				values[i].Value = n
			}

			protoVal.Value = &v1.AnyValue_ArrayValue{ArrayValue: &v1.ArrayValue{Values: values}}
		case attrTypeNotSupported:
			if attr.ValueDropped == "" || !includeDroppedAttr {
				continue
			}
			_ = jsonpb.Unmarshal(bytes.NewBufferString(attr.ValueDropped), &protoVal)
			counter.addDroppedAttr(-1)
		default:
			continue
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
		Name:    parquetScope.Name,
		Version: parquetScope.Version,
	}
	// TODO: handle attributes correctly once they are added to the proto
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
				protoLink.Attributes = parquetToProtoAttrs(l.Attrs, &l, false)
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
				protoEvent.Attributes = parquetToProtoAttrs(e.Attrs, &e, false)
			}

			protoEvents = append(protoEvents, protoEvent)
		}
	}

	return protoEvents
}

func parquetTraceToTempopbTrace(meta *backend.BlockMeta, parquetTrace *Trace, includeDroppedAttr bool) *tempopb.Trace {
	protoTrace := &tempopb.Trace{}
	protoTrace.Batches = make([]*v1_trace.ResourceSpans, 0, len(parquetTrace.ResourceSpans))

	// dedicated attribute column assignments
	dedicatedResourceAttributes := dedicatedColumnsToColumnMapping(meta.DedicatedColumns, backend.DedicatedColumnScopeResource)
	dedicatedSpanAttributes := dedicatedColumnsToColumnMapping(meta.DedicatedColumns, backend.DedicatedColumnScopeSpan)

	for _, rs := range parquetTrace.ResourceSpans {
		protoBatch := &v1_trace.ResourceSpans{}
		resAttrs := parquetToProtoAttrs(rs.Resource.Attrs, &rs.Resource, includeDroppedAttr)
		protoBatch.Resource = &v1_resource.Resource{
			Attributes:             resAttrs,
			DroppedAttributesCount: uint32(rs.Resource.DroppedAttributesCount),
		}

		// dynamically assigned dedicated resource attribute columns
		dedicatedResourceAttributes.forEach(func(attr string, col dedicatedColumn) {
			val := col.readValue(&rs.Resource.DedicatedAttributes)
			if val != nil {
				protoBatch.Resource.Attributes = append(protoBatch.Resource.Attributes, &v1.KeyValue{
					Key:   attr,
					Value: val,
				})
			}
		})

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
		for _, attr := range []struct {
			Key   string
			Value *string
		}{
			{Key: LabelCluster, Value: rs.Resource.Cluster},
			{Key: LabelNamespace, Value: rs.Resource.Namespace},
			{Key: LabelPod, Value: rs.Resource.Pod},
			{Key: LabelContainer, Value: rs.Resource.Container},
			{Key: LabelK8sClusterName, Value: rs.Resource.K8sClusterName},
			{Key: LabelK8sNamespaceName, Value: rs.Resource.K8sNamespaceName},
			{Key: LabelK8sPodName, Value: rs.Resource.K8sPodName},
			{Key: LabelK8sContainerName, Value: rs.Resource.K8sContainerName},
		} {
			if attr.Value != nil {
				protoBatch.Resource.Attributes = append(protoBatch.Resource.Attributes, &v1.KeyValue{
					Key: attr.Key,
					Value: &v1.AnyValue{
						Value: &v1.AnyValue_StringValue{
							StringValue: *attr.Value,
						},
					},
				})
			}
		}

		protoBatch.ScopeSpans = make([]*v1_trace.ScopeSpans, 0, len(rs.ScopeSpans))

		for _, scopeSpan := range rs.ScopeSpans {
			protoSS := &v1_trace.ScopeSpans{
				Scope: parquetToProtoInstrumentationScope(&scopeSpan.Scope),
			}

			protoSS.Spans = make([]*v1_trace.Span, 0, len(scopeSpan.Spans))
			for _, span := range scopeSpan.Spans {

				spanAttr := parquetToProtoAttrs(span.Attrs, &span, includeDroppedAttr)
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
				dedicatedSpanAttributes.forEach(func(attr string, col dedicatedColumn) {
					val := col.readValue(&span.DedicatedAttributes)
					if val != nil {
						protoSpan.Attributes = append(protoSpan.Attributes, &v1.KeyValue{
							Key:   attr,
							Value: val,
						})
					}
				})

				// known span attributes
				if span.HttpMethod != nil {
					protoSpan.Attributes = append(protoSpan.Attributes, &v1.KeyValue{
						Key: LabelHTTPMethod,
						Value: &v1.AnyValue{
							Value: &v1.AnyValue_StringValue{
								StringValue: *span.HttpMethod,
							},
						},
					})
				}
				if span.HttpUrl != nil {
					protoSpan.Attributes = append(protoSpan.Attributes, &v1.KeyValue{
						Key: LabelHTTPUrl,
						Value: &v1.AnyValue{
							Value: &v1.AnyValue_StringValue{
								StringValue: *span.HttpUrl,
							},
						},
					})
				}
				if span.HttpStatusCode != nil {
					protoSpan.Attributes = append(protoSpan.Attributes, &v1.KeyValue{
						Key: LabelHTTPStatusCode,
						Value: &v1.AnyValue{
							Value: &v1.AnyValue_IntValue{
								IntValue: *span.HttpStatusCode,
							},
						},
					})
				}

				protoSS.Spans = append(protoSS.Spans, protoSpan)
			}

			protoBatch.ScopeSpans = append(protoBatch.ScopeSpans, protoSS)
		}
		protoTrace.Batches = append(protoTrace.Batches, protoBatch)
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
