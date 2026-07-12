package traceql

import (
	"testing"

	"github.com/stretchr/testify/assert"

	v1trace "github.com/grafana/tempo/pkg/tempopb/trace/v1"
)

func TestKindFromOTLP(t *testing.T) {
	cases := map[v1trace.Span_SpanKind]Kind{
		v1trace.Span_SPAN_KIND_UNSPECIFIED: KindUnspecified,
		v1trace.Span_SPAN_KIND_INTERNAL:    KindInternal,
		v1trace.Span_SPAN_KIND_CLIENT:      KindClient,
		v1trace.Span_SPAN_KIND_SERVER:      KindServer,
		v1trace.Span_SPAN_KIND_PRODUCER:    KindProducer,
		v1trace.Span_SPAN_KIND_CONSUMER:    KindConsumer,
		v1trace.Span_SpanKind(99):          Kind(99), // unknown kind falls through to the raw integer
	}
	for in, want := range cases {
		assert.Equal(t, want, KindFromOTLP(in), "kind %v", in)
	}
}

func TestStatusFromOTLP(t *testing.T) {
	cases := map[v1trace.Status_StatusCode]Status{
		v1trace.Status_STATUS_CODE_UNSET: StatusUnset,
		v1trace.Status_STATUS_CODE_OK:    StatusOk,
		v1trace.Status_STATUS_CODE_ERROR: StatusError,
		v1trace.Status_StatusCode(99):    Status(99), // unknown status falls through to the raw integer
	}
	for in, want := range cases {
		assert.Equal(t, want, StatusFromOTLP(in), "status %v", in)
	}
}
