package traceql

import v1trace "github.com/grafana/tempo/pkg/tempopb/trace/v1"

// KindFromOTLP maps an OTLP span kind to the TraceQL Kind. Unknown values fall through to the raw
// integer, matching the storage layer.
func KindFromOTLP(k v1trace.Span_SpanKind) Kind {
	switch k {
	case v1trace.Span_SPAN_KIND_UNSPECIFIED:
		return KindUnspecified
	case v1trace.Span_SPAN_KIND_INTERNAL:
		return KindInternal
	case v1trace.Span_SPAN_KIND_SERVER:
		return KindServer
	case v1trace.Span_SPAN_KIND_CLIENT:
		return KindClient
	case v1trace.Span_SPAN_KIND_PRODUCER:
		return KindProducer
	case v1trace.Span_SPAN_KIND_CONSUMER:
		return KindConsumer
	default:
		return Kind(k)
	}
}

// StatusFromOTLP maps an OTLP status code to the TraceQL Status. Unknown values fall through to the
// raw integer, matching the storage layer.
func StatusFromOTLP(c v1trace.Status_StatusCode) Status {
	switch c {
	case v1trace.Status_STATUS_CODE_UNSET:
		return StatusUnset
	case v1trace.Status_STATUS_CODE_OK:
		return StatusOk
	case v1trace.Status_STATUS_CODE_ERROR:
		return StatusError
	default:
		return Status(c)
	}
}
