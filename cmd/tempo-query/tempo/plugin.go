package tempo

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/grafana/tempo/pkg/tempopb"

	jaeger "github.com/jaegertracing/jaeger/model"
	jaeger_spanstore "github.com/jaegertracing/jaeger/storage/spanstore"
	opentelemetry_proto_common_v1 "github.com/open-telemetry/opentelemetry-proto/gen/go/common/v1"
	opentelemetry_resource_trace_v1 "github.com/open-telemetry/opentelemetry-proto/gen/go/resource/v1"
	opentelemetry_proto_trace_v1 "github.com/open-telemetry/opentelemetry-proto/gen/go/trace/v1"
)

type Backend struct {
	tempoEndpoint string
}

func New(cfg *Config) *Backend {
	return &Backend{
		tempoEndpoint: "http://" + cfg.Backend + "/api/traces/",
	}
}

func (b *Backend) GetDependencies(endTs time.Time, lookback time.Duration) ([]jaeger.DependencyLink, error) {
	return nil, nil
}
func (b *Backend) GetTrace(ctx context.Context, traceID jaeger.TraceID) (*jaeger.Trace, error) {
	hexID := fmt.Sprintf("%016x%016x", traceID.High, traceID.Low)
	resp, err := http.Get(b.tempoEndpoint + hexID)
	if err != nil {
		return nil, err
	}

	out := &tempopb.Trace{}
	err = json.NewDecoder(resp.Body).Decode(out)
	resp.Body.Close()
	if err != nil {
		return nil, err
	}

	if len(out.Batches) == 0 {
		return nil, fmt.Errorf("TraceID Not Found: " + hexID)
	}

	jaegerTrace := &jaeger.Trace{
		Spans:      []*jaeger.Span{},
		ProcessMap: []jaeger.Trace_ProcessMapping{},
	}

	// now convert trace to jaeger
	// todo: remove custom code in favor of otelcol once it's complete
	for _, batch := range out.Batches {
		for _, span := range batch.Spans {
			jaegerTrace.Spans = append(jaegerTrace.Spans, protoSpanToJaegerSpan(span, batch.Resource))
		}
	}

	return jaegerTrace, nil
}

func (b *Backend) GetServices(ctx context.Context) ([]string, error) {
	return nil, nil
}
func (b *Backend) GetOperations(ctx context.Context, query jaeger_spanstore.OperationQueryParameters) ([]jaeger_spanstore.Operation, error) {
	return nil, nil
}
func (b *Backend) FindTraces(ctx context.Context, query *jaeger_spanstore.TraceQueryParameters) ([]*jaeger.Trace, error) {
	return nil, nil
}
func (b *Backend) FindTraceIDs(ctx context.Context, query *jaeger_spanstore.TraceQueryParameters) ([]jaeger.TraceID, error) {
	return nil, nil
}
func (b *Backend) WriteSpan(span *jaeger.Span) error {
	return nil
}

func protoSpanToJaegerSpan(in *opentelemetry_proto_trace_v1.Span, resource *opentelemetry_resource_trace_v1.Resource) *jaeger.Span {

	traceID := jaeger.TraceID{
		High: binary.BigEndian.Uint64(in.TraceId[:8]),
		Low:  binary.BigEndian.Uint64(in.TraceId[8:]),
	}

	s := &jaeger.Span{
		TraceID:       traceID,
		SpanID:        jaeger.SpanID(binary.BigEndian.Uint64(in.SpanId)),
		OperationName: in.Name,
		StartTime:     time.Unix(0, int64(in.StartTimeUnixnano)),
		Duration:      time.Unix(0, int64(in.EndTimeUnixnano)).Sub(time.Unix(0, int64(in.StartTimeUnixnano))),
		Tags:          protoAttsToJaegerTags(in.Attributes),
		Events:        protoEventsToJaegerLogs(in.Events),
	}

	for _, link := range in.Links {
		s.References = append(s.References, jaeger.SpanRef{
			TraceID: traceID,
			SpanID:  jaeger.SpanID(binary.BigEndian.Uint64(link.SpanId)),
			RefType: jaeger.SpanRefType_CHILD_OF,
		})
	}

	s.Process = &jaeger.Process{}
	for _, att := range resource.Attributes {
		if att.Key == "process_name" {
			s.Process.ServiceName = att.StringValue
		}
	}

	return s
}

func protoAttsToJaegerTags(ocAttribs []*opentelemetry_proto_common_v1.AttributeKeyValue) []jaeger.KeyValue {
	if ocAttribs == nil {
		return nil
	}

	// Pre-allocate assuming that few attributes, if any at all, are nil.
	jTags := make([]jaeger.KeyValue, 0, len(ocAttribs))
	for _, attrib := range ocAttribs {
		if attrib == nil {
			continue
		}

		jTag := jaeger.KeyValue{Key: attrib.Key}
		switch attrib.Type {
		case opentelemetry_proto_common_v1.AttributeKeyValue_STRING:
			// Jaeger-to-OC maps binary tags to string attributes and encodes them as
			// base64 strings. Blindingly attempting to decode base64 seems too much.
			str := attrib.StringValue
			jTag.VStr = str
			jTag.VType = jaeger.ValueType_STRING
		case opentelemetry_proto_common_v1.AttributeKeyValue_INT:
			i := attrib.IntValue
			jTag.VInt64 = i
			jTag.VType = jaeger.ValueType_INT64
		case opentelemetry_proto_common_v1.AttributeKeyValue_BOOL:
			b := attrib.BoolValue
			jTag.VBool = b
			jTag.VType = jaeger.ValueType_BOOL
		case opentelemetry_proto_common_v1.AttributeKeyValue_DOUBLE:
			d := attrib.DoubleValue
			jTag.VFloat64 = d
			jTag.VType = jaeger.ValueType_FLOAT64
		default:
			str := "<Unknown OpenTelemetry Attribute for key \"" + attrib.Key + "\">"
			jTag.VStr = str
			jTag.VType = jaeger.ValueType_STRING
		}
		jTags = append(jTags, jTag)
	}

	return jTags
}

func protoEventsToJaegerLogs(ocSpanTimeEvents []*opentelemetry_proto_trace_v1.Span_Event) []jaeger.Log {
	if ocSpanTimeEvents == nil {
		return nil
	}

	// Assume that in general no time events are going to produce nil Jaeger logs.
	jLogs := make([]jaeger.Log, 0, len(ocSpanTimeEvents))
	for _, ocTimeEvent := range ocSpanTimeEvents {
		jLog := jaeger.Log{
			Timestamp: time.Unix(0, int64(ocTimeEvent.TimeUnixnano)),
			Fields:    protoAttsToJaegerTags(ocTimeEvent.Attributes),
		}

		jLogs = append(jLogs, jLog)
	}

	return jLogs
}
