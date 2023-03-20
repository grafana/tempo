package trace

import v1trace "github.com/grafana/tempo/pkg/tempopb/trace/v1"

func StatusToString(s v1trace.Status_StatusCode) string {
	var status string
	switch s {
	case v1trace.Status_STATUS_CODE_UNSET:
		status = "unset"
	case v1trace.Status_STATUS_CODE_OK:
		status = "ok"
	case v1trace.Status_STATUS_CODE_ERROR:
		status = "error"
	}
	return status
}

func KindToString(k v1trace.Span_SpanKind) string {
	var kind string
	switch k {
	case v1trace.Span_SPAN_KIND_UNSPECIFIED:
		kind = "unspecified"
	case v1trace.Span_SPAN_KIND_INTERNAL:
		kind = "internal"
	case v1trace.Span_SPAN_KIND_SERVER:
		kind = "server"
	case v1trace.Span_SPAN_KIND_CLIENT:
		kind = "client"
	case v1trace.Span_SPAN_KIND_PRODUCER:
		kind = "producer"
	case v1trace.Span_SPAN_KIND_CONSUMER:
		kind = "consumer"
	}
	return kind
}
