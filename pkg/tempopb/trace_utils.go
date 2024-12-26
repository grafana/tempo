package tempopb

import (
	"strings"

	"github.com/gogo/protobuf/jsonpb"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// It marshal a Trace to an OTEL compatible JSON.
// Historically, our Trace proto message used `batches` to define the array of resourses spans.
// To be OTEL compatible we renamed it to `resourcesSpans`.
// To be backward compatible, this function use jsonpb to marshal the Trace to an OTEL compatible JSON
// and then replace the first occurrence of `resourceSpan` by `batches`.
func MarshalToJSONV1(t *Trace) ([]byte, error) {
	marshaler := &jsonpb.Marshaler{}
	jsonStr, err := marshaler.MarshalToString(t)
	if err != nil {
		return nil, err
	}
	jsonStr = strings.Replace(jsonStr, `"resourceSpans":`, `"batches":`, 1)
	return []byte(jsonStr), nil
}

// It unmarshal an OTEL compatible JSON to a Trace.
// Historically, our Trace proto message used `batches` to define the array of resourses spans.
// To be OTEL compatible we renamed it to `resourcesSpan`.
// To be backward compatible, this function replaces the first occurrence of `batches` by `resourcesSpan`
// and then use jsonpb to unmarshal JSON into a Trace.
func UnmarshalFromJSONV1(data []byte, t *Trace) error {
	marshaler := &jsonpb.Unmarshaler{}
	jsonStr := strings.Replace(string(data), `"batches":`, `"resourceSpans":`, 1)
	err := marshaler.Unmarshal(strings.NewReader(jsonStr), t)
	return err
}

// ConvertFromOTLP creates a Trace from ptrace.Traces (OTLP). It does so by converting the trace to
// bytes and unmarshaling it. This is unfortunate for efficiency, but it works around the OTel
// Collector internalization of otel-proto.
func ConvertFromOTLP(traces ptrace.Traces) (*Trace, error) {
	b, err := (&ptrace.ProtoMarshaler{}).MarshalTraces(traces)
	if err != nil {
		return nil, err
	}

	var t Trace
	err = t.Unmarshal(b)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// ConvertToOTLP creates a ptrace.Traces (OTLP) from Trace. It does so by converting the trace to
// bytes and unmarshaling it. This is unfortunate for efficiency, but it works around the OTel
// Collector internalization of otel-proto.
func (t *Trace) ConvertToOTLP() (ptrace.Traces, error) {
	b, err := t.Marshal()
	if err != nil {
		return ptrace.Traces{}, err
	}

	traces, err := (&ptrace.ProtoUnmarshaler{}).UnmarshalTraces(b)
	if err != nil {
		return ptrace.Traces{}, err
	}
	return traces, nil
}

func (t *Trace) SpanCount() int {
	spanCount := 0
	rss := t.GetResourceSpans()
	for i := 0; i < len(rss); i++ {
		rs := rss[i]
		ilss := rs.GetScopeSpans()
		for j := 0; j < len(ilss); j++ {
			spanCount += len(ilss[j].GetSpans())
		}
	}
	return spanCount
}
