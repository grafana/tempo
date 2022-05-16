package vparquet

import (
	"encoding/json"
	"math"

	"github.com/grafana/tempo/pkg/model/trace"
	"github.com/grafana/tempo/pkg/tempopb"
	v1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
	v1_trace "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/util"
)

// These definition levels match the schema below

const DefinitionLevelTrace = 0
const DefinitionLevelResourceSpans = 1
const DefinitionLevelResourceAttrs = 2
const DefinitionLevelResourceSpansILSSpan = 3

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
	ParentSpanID           string      `parquet:",snappy"`
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

func attrsToParquet(attrs []*v1.KeyValue) []Attribute {
	aa := []Attribute{}
	for _, a := range attrs {
		aa = append(aa, attrToParquet(a))
	}
	return aa
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
				case "service.name":
					ob.Resource.ServiceName = a.Value.GetStringValue()
				case "cluster":
					c := a.Value.GetStringValue()
					ob.Resource.Cluster = &c
				case "namespace":
					n := a.Value.GetStringValue()
					ob.Resource.Namespace = &n
				case "pod":
					p := a.Value.GetStringValue()
					ob.Resource.Pod = &p
				case "container":
					c := a.Value.GetStringValue()
					ob.Resource.Container = &c

				case "k8s.cluster.name":
					c := a.Value.GetStringValue()
					ob.Resource.K8sClusterName = &c
				case "k8s.namespace.name":
					n := a.Value.GetStringValue()
					ob.Resource.K8sNamespaceName = &n
				case "k8s.pod.name":
					p := a.Value.GetStringValue()
					ob.Resource.K8sPodName = &p
				case "k8s.container.name":
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
		j, _ := json.Marshal(a.Value)

		ee.Attrs = append(ee.Attrs, EventAttribute{
			Key:   a.Key,
			Value: string(j),
		})
	}

	return ee
}
