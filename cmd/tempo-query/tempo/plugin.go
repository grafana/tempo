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
	ot_common "github.com/open-telemetry/opentelemetry-proto/gen/go/common/v1"
	ot_resource "github.com/open-telemetry/opentelemetry-proto/gen/go/resource/v1"
	ot_trace "github.com/open-telemetry/opentelemetry-proto/gen/go/trace/v1"
	"go.opentelemetry.io/collector/translator/conventions"
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
			jSpan := protoSpanToJaegerSpan(span)
			jProcess, processID := protoResourceToJaegerProcess(batch.Resource)

			jSpan.ProcessID = processID
			jSpan.Process = &jProcess

			jaegerTrace.Spans = append(jaegerTrace.Spans, jSpan)
			jaegerTrace.ProcessMap = append(jaegerTrace.ProcessMap, jaeger.Trace_ProcessMapping{
				Process:   jProcess,
				ProcessID: processID,
			})
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

func protoResourceToJaegerProcess(in *ot_resource.Resource) (jaeger.Process, string) {
	processName := ""
	p := jaeger.Process{
		Tags: make([]jaeger.KeyValue, 0, len(in.Attributes)),
	}

	for _, att := range in.Attributes {
		if att == nil {
			continue
		}

		tag := protoAttToJaegerTag(att)
		if tag.Key == conventions.AttributeServiceName {
			p.ServiceName = tag.VStr
		}

		if tag.Key == conventions.AttributeHostHostname {
			processName = tag.VStr
		}

		p.Tags = append(p.Tags, tag)
	}

	return p, processName
}

func protoSpanToJaegerSpan(in *ot_trace.Span) *jaeger.Span {
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
		Logs:          protoEventsToJaegerLogs(in.Events),
	}

	for _, link := range in.Links {
		s.References = append(s.References, jaeger.SpanRef{
			TraceID: traceID,
			SpanID:  jaeger.SpanID(binary.BigEndian.Uint64(link.SpanId)),
			RefType: jaeger.SpanRefType_CHILD_OF,
		})
	}

	return s
}

func protoAttsToJaegerTags(ocAttribs []*ot_common.AttributeKeyValue) []jaeger.KeyValue {
	if ocAttribs == nil {
		return nil
	}

	// Pre-allocate assuming that few attributes, if any at all, are nil.
	jTags := make([]jaeger.KeyValue, 0, len(ocAttribs))
	for _, att := range ocAttribs {
		if att == nil {
			continue
		}

		jTags = append(jTags, protoAttToJaegerTag(att))
	}

	return jTags
}

func protoAttToJaegerTag(attrib *ot_common.AttributeKeyValue) jaeger.KeyValue {
	jTag := jaeger.KeyValue{Key: attrib.Key}
	switch attrib.Type {
	case ot_common.AttributeKeyValue_STRING:
		// Jaeger-to-OC maps binary tags to string attributes and encodes them as
		// base64 strings. Blindingly attempting to decode base64 seems too much.
		str := attrib.StringValue
		jTag.VStr = str
		jTag.VType = jaeger.ValueType_STRING
	case ot_common.AttributeKeyValue_INT:
		i := attrib.IntValue
		jTag.VInt64 = i
		jTag.VType = jaeger.ValueType_INT64
	case ot_common.AttributeKeyValue_BOOL:
		b := attrib.BoolValue
		jTag.VBool = b
		jTag.VType = jaeger.ValueType_BOOL
	case ot_common.AttributeKeyValue_DOUBLE:
		d := attrib.DoubleValue
		jTag.VFloat64 = d
		jTag.VType = jaeger.ValueType_FLOAT64
	default:
		str := "<Unknown OpenTelemetry Attribute for key \"" + attrib.Key + "\">"
		jTag.VStr = str
		jTag.VType = jaeger.ValueType_STRING
	}

	return jTag
}

func protoEventsToJaegerLogs(ocSpanTimeEvents []*ot_trace.Span_Event) []jaeger.Log {
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
