// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package opencensus // import "go.opentelemetry.io/otel/bridge/opencensus"

import (
	octrace "go.opencensus.io/trace"

	"go.opentelemetry.io/otel/bridge/opencensus/internal"
	"go.opentelemetry.io/otel/bridge/opencensus/internal/oc2otel"
	"go.opentelemetry.io/otel/bridge/opencensus/internal/otel2oc"
	"go.opentelemetry.io/otel/trace"
)

// NewTracer returns an implementation of the OpenCensus Tracer interface which
// uses OpenTelemetry APIs.  Using this implementation of Tracer "upgrades"
// libraries that use OpenCensus to OpenTelemetry to facilitate a migration.
func NewTracer(tracer trace.Tracer) octrace.Tracer {
	return internal.NewTracer(tracer)
}

// OTelSpanContextToOC converts from an OpenTelemetry SpanContext to an
// OpenCensus SpanContext, and handles any incompatibilities with the global
// error handler.
func OTelSpanContextToOC(sc trace.SpanContext) octrace.SpanContext {
	return otel2oc.SpanContext(sc)
}

// OCSpanContextToOTel converts from an OpenCensus SpanContext to an
// OpenTelemetry SpanContext.
func OCSpanContextToOTel(sc octrace.SpanContext) trace.SpanContext {
	return oc2otel.SpanContext(sc)
}
