// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package zipkinv2 // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/translator/zipkin/zipkinv2"

import (
	"encoding/json"

	zipkinmodel "github.com/openzipkin/zipkin-go/model"
	zipkinreporter "github.com/openzipkin/zipkin-go/reporter"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

type jsonUnmarshaler struct {
	toTranslator ToTranslator
}

// UnmarshalTraces from JSON bytes.
func (j jsonUnmarshaler) UnmarshalTraces(buf []byte) (ptrace.Traces, error) {
	var spans []*zipkinmodel.SpanModel
	if err := json.Unmarshal(buf, &spans); err != nil {
		return ptrace.Traces{}, err
	}
	return j.toTranslator.ToTraces(spans)
}

// NewJSONTracesUnmarshaler returns an unmarshaler for JSON bytes.
func NewJSONTracesUnmarshaler(parseStringTags bool) ptrace.Unmarshaler {
	return jsonUnmarshaler{toTranslator: ToTranslator{ParseStringTags: parseStringTags}}
}

// NewJSONTracesMarshaler returns a marshaler to JSON bytes.
func NewJSONTracesMarshaler() ptrace.Marshaler {
	return marshaler{
		serializer: zipkinreporter.JSONSerializer{},
	}
}
