// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package zipkinv2 // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/translator/zipkin/zipkinv2"

import (
	zipkinreporter "github.com/openzipkin/zipkin-go/reporter"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

type marshaler struct {
	serializer     zipkinreporter.SpanSerializer
	fromTranslator FromTranslator
}

// MarshalTraces to JSON bytes.
func (j marshaler) MarshalTraces(td ptrace.Traces) ([]byte, error) {
	spans, err := j.fromTranslator.FromTraces(td)
	if err != nil {
		return nil, err
	}
	return j.serializer.Serialize(spans)
}
