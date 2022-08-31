package vparquet

import (
	"bytes"

	"github.com/golang/protobuf/jsonpb"
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

	LabelHTTPMethod     = "http.method"
	LabelHTTPUrl        = "http.url"
	LabelHTTPStatusCode = "http.status_code"
)

// These definition levels match the schema below

const DefinitionLevelTrace = 0
const DefinitionLevelResourceSpans = 1
const DefinitionLevelResourceAttrs = 2
const DefinitionLevelResourceSpansILSSpan = 3

var (
	jsonMarshaler = new(jsonpb.Marshaler)
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

type EventAttribute struct {
	Key   string `parquet:",snappy,dict"`
	Value []byte `parquet:",snappy"` // Was json-encoded data, is now proto encoded data
}

type Event struct {
	TimeUnixNano           uint64           `parquet:",delta"`
	Name                   string           `parquet:",snappy"`
	Attrs                  []EventAttribute `parquet:""`
	DroppedAttributesCount int32            `parquet:",snappy,delta"`
	Test                   string           `parquet:",snappy,dict,optional"` // Always empty for testing
}

// nolint:revive
// Ignore field naming warnings
type Span struct {
	// ID is []byte to save space. It doesn't need to be user
	// friendly like trace ID, and []byte is half the size of string.
	ID                     []byte      `parquet:","`
	Name                   string      `parquet:",snappy,dict"`
	Kind                   int         `parquet:",dict"`
	ParentSpanID           []byte      `parquet:","`
	TraceState             string      `parquet:",snappy"`
	StartUnixNanos         uint64      `parquet:",delta"`
	EndUnixNanos           uint64      `parquet:",delta"`
	StatusCode             int         `parquet:",delta"`
	StatusMessage          string      `parquet:",snappy"`
	Attrs                  []Attribute `parquet:""`
	DroppedAttributesCount int32       `parquet:",snappy"`
	Events                 []Event     `parquet:""`
	DroppedEventsCount     int32       `parquet:",snappy"`

	// Known attributes
	HttpMethod     *string `parquet:",snappy,optional,dict"`
	HttpUrl        *string `parquet:",snappy,optional,dict"`
	HttpStatusCode *int64  `parquet:",snappy,optional"`
}

type IL struct {
	Name    string `parquet:",snappy,dict"`
	Version string `parquet:",snappy,dict"`
}

type ILS struct {
	InstrumentationLibrary IL     `parquet:"il"`
	Spans                  []Span `parquet:""`
}

type Resource struct {
	Attrs []Attribute

	// Known attributes
	ServiceName      string  `parquet:",snappy,dict"`
	Cluster          *string `parquet:",snappy,optional,dict"`
	Namespace        *string `parquet:",snappy,optional,dict"`
	Pod              *string `parquet:",snappy,optional,dict"`
	Container        *string `parquet:",snappy,optional,dict"`
	K8sClusterName   *string `parquet:",snappy,optional,dict"`
	K8sNamespaceName *string `parquet:",snappy,optional,dict"`
	K8sPodName       *string `parquet:",snappy,optional,dict"`
	K8sContainerName *string `parquet:",snappy,optional,dict"`

	Test string `parquet:",snappy,dict,optional"` // Always empty for testing
}

type ResourceSpans struct {
	Resource                    Resource `parquet:""`
	InstrumentationLibrarySpans []ILS    `parquet:"ils"`
}

type Trace struct {
	// TraceID is a byte slice as it helps maintain the sort order of traces within a parquet file
	TraceID       []byte          `parquet:",delta"`
	ResourceSpans []ResourceSpans `parquet:"rs"`

	// TraceIDText is for better useability on downstream systems i.e: something other than Tempo is reading these files.
	// It will not be used as the primary traceID field within Tempo and is only helpful for debugging purposes.
	TraceIDText string `parquet:",delta"`

	// Trace-level attributes for searching
	StartTimeUnixNano uint64 `parquet:",delta"`
	EndTimeUnixNano   uint64 `parquet:",delta"`
	DurationNanos     uint64 `parquet:",delta"`
	RootServiceName   string `parquet:",dict"`
	RootSpanName      string `parquet:",dict"`
}

func attrToParquet(a *v1.KeyValue) Attribute {
	p := Attribute{
		Key: a.Key,
	}
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
	return p
}

func traceToParquet(id common.ID, tr *tempopb.Trace) Trace {

	ot := Trace{
		TraceIDText: util.TraceIDToHexString(id),
		TraceID:     util.PadTraceIDTo16Bytes(id),
	}

	// Trace-level items
	traceStart := uint64(0)
	traceEnd := uint64(0)
	var rootSpan *v1_trace.Span
	var rootBatch *v1_trace.ResourceSpans

	ot.ResourceSpans = make([]ResourceSpans, 0, len(tr.Batches))
	for _, b := range tr.Batches {
		ob := ResourceSpans{
			Resource: Resource{},
		}

		if b.Resource != nil {
			ob.Resource.Attrs = make([]Attribute, 0, len(b.Resource.Attributes))
			for _, a := range b.Resource.Attributes {
				switch a.Key {
				case LabelServiceName:
					ob.Resource.ServiceName = a.Value.GetStringValue()
				case LabelCluster:
					c := a.Value.GetStringValue()
					ob.Resource.Cluster = &c
				case LabelNamespace:
					n := a.Value.GetStringValue()
					ob.Resource.Namespace = &n
				case LabelPod:
					p := a.Value.GetStringValue()
					ob.Resource.Pod = &p
				case LabelContainer:
					c := a.Value.GetStringValue()
					ob.Resource.Container = &c

				case LabelK8sClusterName:
					c := a.Value.GetStringValue()
					ob.Resource.K8sClusterName = &c
				case LabelK8sNamespaceName:
					n := a.Value.GetStringValue()
					ob.Resource.K8sNamespaceName = &n
				case LabelK8sPodName:
					p := a.Value.GetStringValue()
					ob.Resource.K8sPodName = &p
				case LabelK8sContainerName:
					c := a.Value.GetStringValue()
					ob.Resource.K8sContainerName = &c

				default:
					// Other attributes put in generic columns
					ob.Resource.Attrs = append(ob.Resource.Attrs, attrToParquet(a))
				}
			}
		}

		ob.InstrumentationLibrarySpans = make([]ILS, 0, len(b.InstrumentationLibrarySpans))
		for _, ils := range b.InstrumentationLibrarySpans {
			oils := ILS{}
			if ils.InstrumentationLibrary != nil {
				oils.InstrumentationLibrary = IL{
					Name:    ils.InstrumentationLibrary.Name,
					Version: ils.InstrumentationLibrary.Version,
				}
			}

			oils.Spans = make([]Span, 0, len(ils.Spans))
			for _, s := range ils.Spans {

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

				events := make([]Event, 0, len(s.Events))
				for _, e := range s.Events {
					events = append(events, eventToParquet(e))
				}

				ss := Span{
					ID:                     s.SpanId,
					ParentSpanID:           s.ParentSpanId,
					Name:                   s.Name,
					Kind:                   int(s.Kind),
					StatusCode:             int(s.Status.Code),
					StatusMessage:          s.Status.Message,
					StartUnixNanos:         s.StartTimeUnixNano,
					EndUnixNanos:           s.EndTimeUnixNano,
					Attrs:                  make([]Attribute, 0, len(s.Attributes)),
					DroppedAttributesCount: int32(s.DroppedAttributesCount),
					Events:                 events,
					DroppedEventsCount:     int32(s.DroppedEventsCount),
				}

				for _, a := range s.Attributes {
					switch a.Key {

					case "http.method":
						m := a.Value.GetStringValue()
						ss.HttpMethod = &m
					case "http.url":
						m := a.Value.GetStringValue()
						ss.HttpUrl = &m
					case "http.status_code":
						m := a.Value.GetIntValue()
						ss.HttpStatusCode = &m
					default:
						// Other attributes put in generic columns
						ss.Attrs = append(ss.Attrs, attrToParquet(a))
					}
				}

				oils.Spans = append(oils.Spans, ss)
			}

			ob.InstrumentationLibrarySpans = append(ob.InstrumentationLibrarySpans, oils)
		}

		ot.ResourceSpans = append(ot.ResourceSpans, ob)
	}

	ot.StartTimeUnixNano = traceStart
	ot.EndTimeUnixNano = traceEnd
	ot.DurationNanos = traceEnd - traceStart

	if rootSpan != nil && rootBatch != nil && rootBatch.Resource != nil {
		ot.RootSpanName = rootSpan.Name

		for _, a := range rootBatch.Resource.Attributes {
			if a.Key == "service.name" {
				ot.RootServiceName = a.Value.GetStringValue()
				break
			}
		}
	}

	return ot
}

func eventToParquet(e *v1_trace.Span_Event) Event {
	ee := Event{
		Name:                   e.Name,
		TimeUnixNano:           e.TimeUnixNano,
		DroppedAttributesCount: int32(e.DroppedAttributesCount),
	}

	for _, a := range e.Attributes {
		b := make([]byte, a.Value.Size())
		_, _ = a.Value.MarshalToSizedBuffer(b)

		ee.Attrs = append(ee.Attrs, EventAttribute{
			Key:   a.Key,
			Value: b,
		})
	}

	return ee
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
					//  this code attempts proto first and, if there was an error, falls back to json
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

func parquetTraceToTempopbTrace(parquetTrace *Trace) (*tempopb.Trace, error) {

	protoTrace := &tempopb.Trace{}
	protoTrace.Batches = make([]*v1_trace.ResourceSpans, 0, len(parquetTrace.ResourceSpans))

	for _, rs := range parquetTrace.ResourceSpans {
		protoBatch := &v1_trace.ResourceSpans{}
		protoBatch.Resource = &v1_resource.Resource{
			Attributes: parquetToProtoAttrs(rs.Resource.Attrs),
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

		protoBatch.InstrumentationLibrarySpans = make([]*v1_trace.InstrumentationLibrarySpans, 0, len(rs.InstrumentationLibrarySpans))

		for _, ils := range rs.InstrumentationLibrarySpans {
			protoILS := &v1_trace.InstrumentationLibrarySpans{
				InstrumentationLibrary: &v1.InstrumentationLibrary{
					Name:    ils.InstrumentationLibrary.Name,
					Version: ils.InstrumentationLibrary.Version,
				},
			}

			protoILS.Spans = make([]*v1_trace.Span, 0, len(ils.Spans))
			for _, span := range ils.Spans {

				protoSpan := &v1_trace.Span{
					TraceId:           parquetTrace.TraceID,
					SpanId:            span.ID,
					TraceState:        span.TraceState,
					Name:              span.Name,
					Kind:              v1_trace.Span_SpanKind(span.Kind),
					ParentSpanId:      span.ParentSpanID,
					StartTimeUnixNano: span.StartUnixNanos,
					EndTimeUnixNano:   span.EndUnixNanos,
					Status: &v1_trace.Status{
						Message: span.StatusMessage,
						Code:    v1_trace.Status_StatusCode(span.StatusCode),
					},
					DroppedAttributesCount: uint32(span.DroppedAttributesCount),
					DroppedEventsCount:     uint32(span.DroppedEventsCount),
					Attributes:             parquetToProtoAttrs(span.Attrs),
					Events:                 parquetToProtoEvents(span.Events),
				}

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

				protoILS.Spans = append(protoILS.Spans, protoSpan)
			}

			protoBatch.InstrumentationLibrarySpans = append(protoBatch.InstrumentationLibrarySpans, protoILS)
		}
		protoTrace.Batches = append(protoTrace.Batches, protoBatch)
	}

	return protoTrace, nil
}
