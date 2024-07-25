package tempopb

import (
	"strings"

	"github.com/gogo/protobuf/jsonpb"
)

// Marshal a Trace to an OTEL compatible JSON replacing resourceSpans by batches
func MarshalToJSONV1(t *Trace) ([]byte, error) {
	marshaler := &jsonpb.Marshaler{}
	jsonStr, err := marshaler.MarshalToString(t)
	if err != nil {
		return nil, err
	}
	// It will replace only the first coincidence
	jsonStr = strings.Replace(jsonStr, `"resourceSpans":`, `"batches":`, 1)
	return []byte(jsonStr), nil
}

// Unmarshal an OTEL compatible JSON to a Trace replacing batches by resourceSpans
func UnmarshalFromJSONV1(data []byte, t *Trace) error {
	marshaler := &jsonpb.Unmarshaler{}
	// It will replace only the first coincidence
	jsonStr := strings.Replace(string(data), `"batches":`, `"resourceSpans":`, 1)
	err := marshaler.Unmarshal(strings.NewReader(jsonStr), t)
	return err
}
