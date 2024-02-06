package vparquet3

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
)

// These definition levels match the schema below
const (
	DefinitionLevelTrace                     = 0
	DefinitionLevelResourceSpans             = 1
	DefinitionLevelResourceAttrs             = 2
	DefinitionLevelResourceSpansILSSpan      = 3
	DefinitionLevelResourceSpansILSSpanAttrs = 4

	FieldResourceAttrKey       = "rs.list.element.Resource.Attrs.list.element.Key"
	FieldResourceAttrVal       = "rs.list.element.Resource.Attrs.list.element.Value"
	FieldResourceAttrValInt    = "rs.list.element.Resource.Attrs.list.element.ValueInt"
	FieldResourceAttrValDouble = "rs.list.element.Resource.Attrs.list.element.ValueDouble"
	FieldResourceAttrValBool   = "rs.list.element.Resource.Attrs.list.element.ValueBool"

	FieldSpanAttrKey       = "rs.list.element.ss.list.element.Spans.list.element.Attrs.list.element.Key"
	FieldSpanAttrVal       = "rs.list.element.ss.list.element.Spans.list.element.Attrs.list.element.Value"
	FieldSpanAttrValInt    = "rs.list.element.ss.list.element.Spans.list.element.Attrs.list.element.ValueInt"
	FieldSpanAttrValDouble = "rs.list.element.ss.list.element.Spans.list.element.Attrs.list.element.ValueDouble"
	FieldSpanAttrValBool   = "rs.list.element.ss.list.element.Spans.list.element.Attrs.list.element.ValueBool"
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

	parquetSchema = parquet.SchemaOf(&Trace{})
)

type Attribute struct {
	Key string `parquet:",snappy,dict"`

	// This is a bad design that leads to millions of null values. How can we fix this?
	Value       *string  `parquet:",dict,snappy,optional"`
	ValueInt    *int64   `parquet:",snappy,optional"`
	ValueDouble *float64 `parquet:",snappy,optional"`
	ValueBool   *bool    `parquet:",snappy,optional"`
	ValueKVList string   `parquet:",snappy,optional"`
	ValueArray  string   `parquet:",snappy,optional"`
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

type EventAttribute struct {
	Key   string `parquet:",snappy,dict"`
	Value []byte `parquet:",snappy"` // Was json-encoded data, is now proto encoded data
}

type Event struct {
	TimeUnixNano           uint64           `parquet:",delta"`
	Name                   string           `parquet:",snappy"`
	Attrs                  []EventAttribute `parquet:",list"`
	DroppedAttributesCount int32            `parquet:",snappy,delta"`
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
	DroppedAttributesCount int32       `parquet:",snappy"`
	Events                 []Event     `parquet:",list"`
	DroppedEventsCount     int32       `parquet:",snappy"`
	Links                  []byte      `parquet:",snappy"` // proto encoded []*v1_trace.Span_Link
	DroppedLinksCount      int32       `parquet:",snappy"`

	// Static dedicated attribute columns
	HttpMethod     *string `parquet:",snappy,optional,dict"`
	HttpUrl        *string `parquet:",snappy,optional,dict"`
	HttpStatusCode *int64  `parquet:",snappy,optional"`

	// Dynamically assignable dedicated attribute columns
	DedicatedAttributes DedicatedAttributes `parquet:""`
}

func (s *Span) IsRoot() bool {
	return len(s.ParentSpanID) == 0
}

type InstrumentationScope struct {
	Name    string `parquet:",snappy,dict"`
	Version string `parquet:",snappy,dict"`
}

type ScopeSpans struct {
	Scope InstrumentationScope `parquet:""`
	Spans []Span               `parquet:",list"`
}

type Resource struct {
	Attrs []Attribute `parquet:",list"`

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

type ResourceSpans struct {
	Resource   Resource     `parquet:""`
	ScopeSpans []ScopeSpans `parquet:"ss,list"`
}

type Trace struct {
	// TraceID is a byte slice as it helps maintain the sort order of traces within a parquet file
	TraceID []byte `parquet:""`
	// TraceIDText is for better usability on downstream systems i.e: something other than Tempo is reading these files.
	// It will not be used as the primary traceID field within Tempo and is only helpful for debugging purposes.
	TraceIDText string `parquet:",snappy"`

	// Trace-level attributes for searching
	StartTimeUnixNano uint64 `parquet:",delta"`
	EndTimeUnixNano   uint64 `parquet:",delta"`
	DurationNano      uint64 `parquet:",delta"`
	RootServiceName   string `parquet:",dict"`
	RootSpanName      string `parquet:",dict"`

	ResourceSpans []ResourceSpans `parquet:"rs,list"`
}

func attrToParquet(a *v1.KeyValue, p *Attribute) {
	p.Key = a.Key
	p.Value = nil
	p.ValueArray = ""
	p.ValueBool = nil
	p.ValueDouble = nil
	p.ValueInt = nil
	p.ValueKVList = ""

	switch v := a.GetValue().Value.(type) {
	case *v1.AnyValue_StringValue:
		p.Value = &v.StringValue
	case *v1.AnyValue_IntValue:
		p.ValueInt = &v.IntValue
	case *v1.AnyValue_DoubleValue:
		p.ValueDouble = &v.DoubleValue
	case *v1.AnyValue_BoolValue:
		p.ValueBool = &v.BoolValue
	case *v1.AnyValue_ArrayValue:
		jsonBytes := &bytes.Buffer{}
		_ = jsonMarshaler.Marshal(jsonBytes, a.Value) // deliberately marshalling a.Value because of AnyValue logic
		p.ValueArray = jsonBytes.String()
	case *v1.AnyValue_KvlistValue:
		jsonBytes := &bytes.Buffer{}
		_ = jsonMarshaler.Marshal(jsonBytes, a.Value) // deliberately marshalling a.Value because of AnyValue logic
		p.ValueKVList = jsonBytes.String()
	}
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
					attrToParquet(a, &ob.Resource.Attrs[attrCount])
					attrCount++
				}
			}
			ob.Resource.Attrs = ob.Resource.Attrs[:attrCount]
		}

		ob.ScopeSpans = extendReuseSlice(len(b.ScopeSpans), ob.ScopeSpans)
		for iils, ils := range b.ScopeSpans {
			oils := &ob.ScopeSpans[iils]
			if ils.Scope != nil {
				oils.Scope = InstrumentationScope{
					Name:    ils.Scope.Name,
					Version: ils.Scope.Version,
				}
			} else {
				oils.Scope.Name = ""
				oils.Scope.Version = ""
			}

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
					eventToParquet(e, &ss.Events[ie])
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
				if len(s.Links) > 0 {
					links := tempopb.LinkSlice{
						Links: s.Links,
					}
					ss.Links = extendReuseSlice(links.Size(), ss.Links)
					_, _ = links.MarshalToSizedBuffer(ss.Links)
				} else {
					ss.Links = ss.Links[:0] // you can 0 length slice a nil slice
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

	return ot, assignNestedSetModelBounds(ot)
}

func eventToParquet(e *v1_trace.Span_Event, ee *Event) {
	ee.Name = e.Name
	ee.TimeUnixNano = e.TimeUnixNano
	ee.DroppedAttributesCount = int32(e.DroppedAttributesCount)

	ee.Attrs = extendReuseSlice(len(e.Attributes), ee.Attrs)
	for i, a := range e.Attributes {
		ee.Attrs[i].Key = a.Key
		ee.Attrs[i].Value = extendReuseSlice(a.Value.Size(), ee.Attrs[i].Value)
		_, _ = a.Value.MarshalToSizedBuffer(ee.Attrs[i].Value)
	}
}

func parquetToProtoAttrs(parquetAttrs []Attribute) []*v1.KeyValue {
	var protoAttrs []*v1.KeyValue

	for _, attr := range parquetAttrs {
		protoVal := &v1.AnyValue{}

		if attr.Value != nil {
			protoVal.Value = &v1.AnyValue_StringValue{
				StringValue: *attr.Value,
			}
		} else if attr.ValueInt != nil {
			protoVal.Value = &v1.AnyValue_IntValue{
				IntValue: *attr.ValueInt,
			}
		} else if attr.ValueDouble != nil {
			protoVal.Value = &v1.AnyValue_DoubleValue{
				DoubleValue: *attr.ValueDouble,
			}
		} else if attr.ValueBool != nil {
			protoVal.Value = &v1.AnyValue_BoolValue{
				BoolValue: *attr.ValueBool,
			}
		} else if attr.ValueArray != "" {
			_ = jsonpb.Unmarshal(bytes.NewBufferString(attr.ValueArray), protoVal)
		} else if attr.ValueKVList != "" {
			_ = jsonpb.Unmarshal(bytes.NewBufferString(attr.ValueKVList), protoVal)
		}

		protoAttrs = append(protoAttrs, &v1.KeyValue{
			Key:   attr.Key,
			Value: protoVal,
		})
	}

	return protoAttrs
}

func parquetToProtoEvents(parquetEvents []Event) []*v1_trace.Span_Event {
	var protoEvents []*v1_trace.Span_Event

	if len(parquetEvents) > 0 {
		protoEvents = make([]*v1_trace.Span_Event, 0, len(parquetEvents))

		for _, e := range parquetEvents {

			protoEvent := &v1_trace.Span_Event{
				TimeUnixNano:           e.TimeUnixNano,
				Name:                   e.Name,
				Attributes:             nil,
				DroppedAttributesCount: uint32(e.DroppedAttributesCount),
			}

			if len(e.Attrs) > 0 {
				protoEvent.Attributes = make([]*v1.KeyValue, 0, len(e.Attrs))

				for _, a := range e.Attrs {
					protoAttr := &v1.KeyValue{
						Key:   a.Key,
						Value: &v1.AnyValue{},
					}

					// event attributes are currently encoded as proto, but were previously json.
					// this code attempts proto first and, if there was an error, falls back to json
					err := protoAttr.Value.Unmarshal(a.Value)
					if err != nil {
						_ = jsonpb.Unmarshal(bytes.NewBuffer(a.Value), protoAttr.Value)
					}

					protoEvent.Attributes = append(protoEvent.Attributes, protoAttr)
				}
			}

			protoEvents = append(protoEvents, protoEvent)
		}
	}

	return protoEvents
}

func parquetTraceToTempopbTrace(meta *backend.BlockMeta, parquetTrace *Trace) *tempopb.Trace {
	protoTrace := &tempopb.Trace{}
	protoTrace.Batches = make([]*v1_trace.ResourceSpans, 0, len(parquetTrace.ResourceSpans))

	// dedicated attribute column assignments
	dedicatedResourceAttributes := dedicatedColumnsToColumnMapping(meta.DedicatedColumns, backend.DedicatedColumnScopeResource)
	dedicatedSpanAttributes := dedicatedColumnsToColumnMapping(meta.DedicatedColumns, backend.DedicatedColumnScopeSpan)

	for _, rs := range parquetTrace.ResourceSpans {
		protoBatch := &v1_trace.ResourceSpans{}
		protoBatch.Resource = &v1_resource.Resource{
			Attributes: parquetToProtoAttrs(rs.Resource.Attrs),
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

		for _, span := range rs.ScopeSpans {
			protoSS := &v1_trace.ScopeSpans{
				Scope: &v1.InstrumentationScope{
					Name:    span.Scope.Name,
					Version: span.Scope.Version,
				},
			}

			protoSS.Spans = make([]*v1_trace.Span, 0, len(span.Spans))
			for _, span := range span.Spans {

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
					DroppedAttributesCount: uint32(span.DroppedAttributesCount),
					DroppedEventsCount:     uint32(span.DroppedEventsCount),
					DroppedLinksCount:      uint32(span.DroppedLinksCount),
					Attributes:             parquetToProtoAttrs(span.Attrs),
					Events:                 parquetToProtoEvents(span.Events),
				}

				// unmarshal links
				if len(span.Links) > 0 {
					links := tempopb.LinkSlice{}
					_ = links.Unmarshal(span.Links) // todo: bubble these errors up
					protoSpan.Links = links.Links
				}

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
