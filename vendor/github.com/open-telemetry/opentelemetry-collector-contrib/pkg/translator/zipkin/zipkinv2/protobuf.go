// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package zipkinv2 // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/translator/zipkin/zipkinv2"

import (
	"github.com/openzipkin/zipkin-go/proto/zipkin_proto3"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

type protobufUnmarshaler struct {
	// debugWasSet toggles the Debug field of each Span. It is usually set to true if
	// the "X-B3-Flags" header is set to 1 on the request.
	debugWasSet bool

	toTranslator ToTranslator
}

// UnmarshalTraces from protobuf bytes.
func (p protobufUnmarshaler) UnmarshalTraces(buf []byte) (ptrace.Traces, error) {
	spans, err := zipkin_proto3.ParseSpans(buf, p.debugWasSet)
	if err != nil {
		return ptrace.Traces{}, err
	}
	return p.toTranslator.ToTraces(spans)
}

// NewProtobufTracesUnmarshaler returns an ptrace.Unmarshaler of protobuf bytes.
func NewProtobufTracesUnmarshaler(debugWasSet, parseStringTags bool) ptrace.Unmarshaler {
	return protobufUnmarshaler{
		debugWasSet:  debugWasSet,
		toTranslator: ToTranslator{ParseStringTags: parseStringTags},
	}
}

// NewProtobufTracesMarshaler returns a new ptrace.Marshaler to protobuf bytes.
func NewProtobufTracesMarshaler() ptrace.Marshaler {
	return marshaler{
		serializer: zipkin_proto3.SpanSerializer{},
	}
}
