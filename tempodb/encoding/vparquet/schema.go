package vparquet

import (
	"bytes"
	"encoding/json"
	"math"

	"github.com/pkg/errors"

	"github.com/golang/protobuf/jsonpb"
	"github.com/grafana/tempo/pkg/model/trace"
	"github.com/grafana/tempo/pkg/tempopb"
	v1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
	v1_resource "github.com/grafana/tempo/pkg/tempopb/resource/v1"
	v1_trace "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/util"
)

// Label names for conversion b/n Proto <> Parquet

const (
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
	Key   string `parquet:",zstd,dict"`
	Value string `parquet:",zstd"` // Json-encoded data
}

type Event struct {
	TimeUnixNano           uint64           `parquet:",delta"`
	Name                   string           `parquet:",snappy"`
	Attrs                  []EventAttribute `parquet:""`
	DroppedAttributesCount int32            `parquet:",snappy,delta"`
	Test                   string           `parquet:",snappy,dict,optional"` // Always empty for testing
}

type Span struct {
	// ID is []byte to save space. It doesn't need to be user
	// friendly like trace ID, and []byte is half the size of string.
	ID                     []byte      `parquet:","`
	Name                   string      `parquet:",snappy,dict"`
	Kind                   int         `parquet:",delta"`
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
	// TraceID is string for better useability on downstream systems
	// i.e: something other than Tempo is reading these files.
	TraceID       string          `parquet:""`
	ResourceSpans []ResourceSpans `parquet:"rs"`

	// Trace-level attributes for searching
	StartTimeUnixNano uint64 `parquet:",delta"`
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
		j, _ := json.Marshal(v.ArrayValue)
		p.ValueArray = string(j)
	case *v1.AnyValue_KvlistValue:
		j, _ := json.Marshal(v.KvlistValue)
		p.ValueKVList = string(j)
	}
	return p
}

func traceToParquet(tr *tempopb.Trace) Trace {
	ot := Trace{
		TraceID: util.TraceIDToHexString(tr.Batches[0].InstrumentationLibrarySpans[0].Spans[0].TraceId),
	}

	// Trace-level items
	traceStart := uint64(math.MaxUint64)
	traceEnd := uint64(0)
	var rootSpan *v1_trace.Span
	var rootBatch *v1_trace.ResourceSpans

	for _, b := range tr.Batches {
		ob := ResourceSpans{
			Resource: Resource{},
		}

		if b.Resource != nil {
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

		for _, ils := range b.InstrumentationLibrarySpans {
			oils := ILS{}
			if ils.InstrumentationLibrary != nil {
				oils.InstrumentationLibrary = IL{
					Name:    ils.InstrumentationLibrary.Name,
					Version: ils.InstrumentationLibrary.Version,
				}
			}

			for _, s := range ils.Spans {

				if s.StartTimeUnixNano < traceStart {
					traceStart = s.StartTimeUnixNano
				}
				if s.EndTimeUnixNano > traceEnd {
					traceEnd = s.EndTimeUnixNano
				}
				if len(s.ParentSpanId) == 0 {
					rootSpan = s
					rootBatch = b
				}

				events := []Event{}
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
					Attrs:                  []Attribute{},
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
	ot.DurationNanos = traceEnd - traceStart
	ot.RootServiceName = trace.RootSpanNotYetReceivedText
	ot.RootSpanName = trace.RootSpanNotYetReceivedText

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
		jsonBytes := &bytes.Buffer{}
		jsonMarshaler.Marshal(jsonBytes, a.Value)
		ee.Attrs = append(ee.Attrs, EventAttribute{
			Key:   a.Key,
			Value: jsonBytes.String(),
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
			val := &v1.AnyValue_ArrayValue{}
			_ = json.Unmarshal([]byte(attr.ValueArray), val.ArrayValue)
			protoVal.Value = val
		} else if attr.ValueKVList != "" {
			val := &v1.AnyValue_KvlistValue{}
			_ = json.Unmarshal([]byte(attr.ValueKVList), val.KvlistValue)
			protoVal.Value = val
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
				TimeUnixNano:           uint64(e.TimeUnixNano),
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

					jsonpb.Unmarshal(bytes.NewBufferString(a.Value), protoAttr.Value)
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

	protoTraceID, err := util.HexStringToTraceID(parquetTrace.TraceID)
	if err != nil {
		return nil, errors.Wrap(err, "error converting from hex string to traceID")
	}

	for _, rs := range parquetTrace.ResourceSpans {
		proto_batch := &v1_trace.ResourceSpans{}
		proto_batch.Resource = &v1_resource.Resource{
			Attributes: parquetToProtoAttrs(rs.Resource.Attrs),
		}

		// known resource attributes
		if rs.Resource.ServiceName != "" {
			proto_batch.Resource.Attributes = append(proto_batch.Resource.Attributes, &v1.KeyValue{
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
				proto_batch.Resource.Attributes = append(proto_batch.Resource.Attributes, &v1.KeyValue{
					Key: attr.Key,
					Value: &v1.AnyValue{
						Value: &v1.AnyValue_StringValue{
							StringValue: *attr.Value,
						},
					},
				})
			}
		}

		proto_batch.InstrumentationLibrarySpans = make([]*v1_trace.InstrumentationLibrarySpans, 0, len(rs.InstrumentationLibrarySpans))

		for _, ils := range rs.InstrumentationLibrarySpans {
			proto_ils := &v1_trace.InstrumentationLibrarySpans{
				InstrumentationLibrary: &v1.InstrumentationLibrary{
					Name:    ils.InstrumentationLibrary.Name,
					Version: ils.InstrumentationLibrary.Version,
				},
			}

			proto_ils.Spans = make([]*v1_trace.Span, 0, len(ils.Spans))
			for _, span := range ils.Spans {

				proto_span := &v1_trace.Span{
					TraceId:           protoTraceID,
					SpanId:            span.ID,
					TraceState:        span.TraceState,
					Name:              span.Name,
					Kind:              v1_trace.Span_SpanKind(span.Kind),
					ParentSpanId:      span.ParentSpanID,
					StartTimeUnixNano: uint64(span.StartUnixNanos),
					EndTimeUnixNano:   uint64(span.EndUnixNanos),
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
					proto_span.Attributes = append(proto_span.Attributes, &v1.KeyValue{
						Key: LabelHTTPMethod,
						Value: &v1.AnyValue{
							Value: &v1.AnyValue_StringValue{
								StringValue: *span.HttpMethod,
							},
						},
					})
				}
				if span.HttpUrl != nil {
					proto_span.Attributes = append(proto_span.Attributes, &v1.KeyValue{
						Key: LabelHTTPUrl,
						Value: &v1.AnyValue{
							Value: &v1.AnyValue_StringValue{
								StringValue: *span.HttpUrl,
							},
						},
					})
				}
				if span.HttpStatusCode != nil {
					proto_span.Attributes = append(proto_span.Attributes, &v1.KeyValue{
						Key: LabelHTTPStatusCode,
						Value: &v1.AnyValue{
							Value: &v1.AnyValue_IntValue{
								IntValue: *span.HttpStatusCode,
							},
						},
					})
				}

				proto_ils.Spans = append(proto_ils.Spans, proto_span)
			}

			proto_batch.InstrumentationLibrarySpans = append(proto_batch.InstrumentationLibrarySpans, proto_ils)
		}
		protoTrace.Batches = append(protoTrace.Batches, proto_batch)
	}

	return protoTrace, nil
}
