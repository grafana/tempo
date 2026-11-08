package tempopb

import (
	"strings"

	"github.com/gogo/protobuf/jsonpb"
)

// MarshalToJSONV1 encodes a Trace to the V1 JSON format. Historically, the
// Trace proto message named its array of resource spans `batches`; it was
// renamed to `resourceSpans` to be OTEL compatible. For backward
// compatibility, the V1 format keeps the legacy `batches` field name.
func MarshalToJSONV1(t *Trace) ([]byte, error) {
	if t == nil {
		return nil, errMarshalNil
	}
	e := newTraceJSONEncoder(t.Size())
	e.trace(t, "batches")
	return e.result()
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
