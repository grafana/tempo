package tempopb

import (
	"bytes"
	"strings"

	"github.com/gogo/protobuf/jsonpb"
	v1 "github.com/grafana/tempo/v3/pkg/tempopb/trace/v1"
)

// traceJSONV1 has the same layout as Trace, but emits the legacy batches field
// name. The pointer conversion in MarshalToJSONV1 enforces that the types stay
// identical apart from struct tags, without copying or mutating the trace.
type traceJSONV1 struct {
	ResourceSpans []*v1.ResourceSpans `protobuf:"bytes,1,rep,name=batches,proto3" json:"batches,omitempty"`
}

func (t *traceJSONV1) Reset()         { *t = traceJSONV1{} }
func (t *traceJSONV1) String() string { return (*Trace)(t).String() }
func (*traceJSONV1) ProtoMessage()    {}

// It marshal a Trace to an OTEL compatible JSON.
// Historically, our Trace proto message used `batches` to define the array of resourses spans.
// To be OTEL compatible we renamed it to `resourcesSpans`.
// To be backward compatible, this function use jsonpb to marshal the Trace to an OTEL compatible JSON
// using the legacy `batches` field name instead of `resourceSpans`.
func MarshalToJSONV1(t *Trace) ([]byte, error) {
	var buf bytes.Buffer
	marshaler := &jsonpb.Marshaler{}
	if err := marshaler.Marshal(&buf, (*traceJSONV1)(t)); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
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
