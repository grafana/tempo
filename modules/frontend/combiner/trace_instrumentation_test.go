package combiner

import (
	"context"
	"net/http"
	"testing"

	"github.com/grafana/tempo/v3/pkg/api"
	"github.com/grafana/tempo/v3/pkg/tempopb"
	"github.com/grafana/tempo/v3/pkg/util/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

type instrumentationRedactor struct{ err error }

func (r instrumentationRedactor) RedactTraceAttributes(_ *tempopb.Trace) error { return r.err }

func TestTraceFinalizationEvents(t *testing.T) {
	for _, tc := range []struct {
		name       string
		combiner   Combiner
		grpc       bool
		phases     []string
		format     string
		wantStatus int
		wantErr    error
	}{
		{name: "v1 JSON", combiner: NewTraceByID(0, api.MarshallingFormatJSON, nil), phases: []string{"deduplicate", "marshal"}, format: "json", wantStatus: http.StatusOK},
		{name: "v2 protobuf", combiner: NewTraceByIDV2(0, api.MarshallingFormatProtobuf, nil, TraceByIDV2Options{}), phases: []string{"deduplicate", "marshal"}, format: "protobuf", wantStatus: http.StatusOK},
		{name: "redaction", combiner: NewTraceByID(0, api.MarshallingFormatJSON, instrumentationRedactor{}), phases: []string{"deduplicate", "redact", "marshal"}, format: "json", wantStatus: http.StatusOK},
		{name: "hidden trace", combiner: NewTraceByID(0, api.MarshallingFormatJSON, hidingRedactor{}), phases: []string{"deduplicate", "redact"}, wantStatus: http.StatusNotFound},
		{name: "redaction failure", combiner: NewTraceByIDV2(0, api.MarshallingFormatJSON, instrumentationRedactor{assert.AnError}, TraceByIDV2Options{}), phases: []string{"deduplicate", "redact"}, wantErr: assert.AnError},
		{name: "gRPC", combiner: NewTypedTraceByIDV2(0, api.MarshallingFormatProtobuf, instrumentationRedactor{}, TraceByIDV2Options{}), grpc: true, phases: []string{"deduplicate", "redact"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			recorder := tracetest.NewSpanRecorder()
			provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
			t.Cleanup(func() { require.NoError(t, provider.Shutdown(context.Background())) })
			ctx, span := provider.Tracer("test").Start(context.Background(), "caller")
			input := toHTTPProtoResponse(t, &tempopb.TraceByIDResponse{
				Trace: test.MakeTraceWithSpanCount(1, 2, nil), Metrics: &tempopb.TraceByIDMetrics{},
			}, http.StatusOK)
			input.HTTPResponse().Header = http.Header{api.HeaderContentType: {api.HeaderAcceptProtobuf}}
			require.NoError(t, tc.combiner.AddResponse(input))

			var resp *http.Response
			var err error
			if tc.grpc {
				_, err = tc.combiner.(GRPCCombiner[*tempopb.TraceByIDResponse]).GRPCFinal(ctx)
			} else {
				resp, err = tc.combiner.HTTPFinal(ctx)
			}
			span.End()
			wantCode := codes.Unset
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
				wantCode = codes.Error
			} else {
				require.NoError(t, err)
				if resp != nil {
					require.Equal(t, tc.wantStatus, resp.StatusCode)
					require.NoError(t, resp.Body.Close())
				}
			}

			ended := recorder.Ended()
			require.Len(t, ended, 1, "phases must use the caller span, not child spans")
			require.Equal(t, "caller", ended[0].Name())
			require.Equal(t, wantCode, ended[0].Status().Code)
			events := ended[0].Events()
			require.Len(t, events, 2*len(tc.phases))
			for i, phase := range tc.phases {
				require.Equal(t, "trace."+phase+".start", events[2*i].Name)
				require.Equal(t, "trace."+phase+".done", events[2*i+1].Name)
			}
			require.Contains(t, ended[0].Attributes(), attribute.Int("span_count", 2))
			if tc.format != "" {
				require.Contains(t, ended[0].Attributes(), attribute.String("format", tc.format))
				require.Contains(t, ended[0].Attributes(), attribute.Int("response_bytes", int(resp.ContentLength)))
			}
		})
	}
}
