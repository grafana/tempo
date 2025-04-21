// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package unmarshaler // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/kafkareceiver/internal/unmarshaler"

import (
	"bytes"

	"github.com/gogo/protobuf/jsonpb"
	jaegerproto "github.com/jaegertracing/jaeger-idl/model/v1"
	"go.opentelemetry.io/collector/pdata/ptrace"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/translator/jaeger"
)

var (
	_ ptrace.Unmarshaler = JaegerProtoSpanUnmarshaler{}
	_ ptrace.Unmarshaler = JaegerJSONSpanUnmarshaler{}
)

type JaegerProtoSpanUnmarshaler struct{}

func (j JaegerProtoSpanUnmarshaler) UnmarshalTraces(bytes []byte) (ptrace.Traces, error) {
	span := &jaegerproto.Span{}
	err := span.Unmarshal(bytes)
	if err != nil {
		return ptrace.NewTraces(), err
	}
	return jaegerSpanToTraces(span)
}

type JaegerJSONSpanUnmarshaler struct{}

func (j JaegerJSONSpanUnmarshaler) UnmarshalTraces(data []byte) (ptrace.Traces, error) {
	span := &jaegerproto.Span{}
	err := jsonpb.Unmarshal(bytes.NewReader(data), span)
	if err != nil {
		return ptrace.NewTraces(), err
	}
	return jaegerSpanToTraces(span)
}

func jaegerSpanToTraces(span *jaegerproto.Span) (ptrace.Traces, error) {
	batch := jaegerproto.Batch{
		Spans:   []*jaegerproto.Span{span},
		Process: span.Process,
	}
	return jaeger.ProtoToTraces([]*jaegerproto.Batch{&batch})
}
