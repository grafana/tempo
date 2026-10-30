package frontend

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-kit/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestFormatRequestHeaders(t *testing.T) {
	h := http.Header{}
	h.Add("X-Header-To-Log", "i should be logged!")
	h.Add("X-Header-To-Not-Log", "i shouldn't be logged!")

	fields := formatRequestHeaders(&h, []string{"X-Header-To-Log", "X-Header-Not-Present"})

	expected := []interface{}{
		"header_x_header_to_log",
		"i should be logged!",
	}

	require.Equal(t, expected, fields)
}

type handlerRoundTripperFunc func(*http.Request) (*http.Response, error)

func (f handlerRoundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

type failingResponseWriter struct{ http.ResponseWriter }

func (w failingResponseWriter) Write(_ []byte) (int, error) {
	return 2, assert.AnError
}

func TestHandlerResponseWriteEvents(t *testing.T) {
	for _, tc := range []struct {
		name        string
		noBody      bool
		writeError  bool
		wantWritten int64
		wantCode    codes.Code
	}{
		{name: "success", wantWritten: 4},
		{name: "nil body", noBody: true},
		{name: "partial write failure", writeError: true, wantWritten: 2, wantCode: codes.Error},
	} {
		t.Run(tc.name, func(t *testing.T) {
			recorder := tracetest.NewSpanRecorder()
			provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
			t.Cleanup(func() { require.NoError(t, provider.Shutdown(context.Background())) })
			ctx, span := provider.Tracer("test").Start(context.Background(), "http.request")
			h := newHandler(nil, handlerRoundTripperFunc(func(_ *http.Request) (*http.Response, error) {
				var body io.ReadCloser
				if !tc.noBody {
					body = io.NopCloser(strings.NewReader("test"))
				}
				return &http.Response{StatusCode: http.StatusOK, Body: body}, nil
			}), log.NewNopLogger())
			var w http.ResponseWriter = httptest.NewRecorder()
			if tc.writeError {
				w = failingResponseWriter{w}
			}
			h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/traces/1234", nil).WithContext(ctx))
			span.End()

			ended := recorder.Ended()
			require.Len(t, ended, 1, "response writing must not create a child span")
			require.Equal(t, tc.wantCode, ended[0].Status().Code)
			events := ended[0].Events()
			require.Len(t, events, 2)
			require.Equal(t, "http.response.write.start", events[0].Name)
			require.Equal(t, "http.response.write.done", events[1].Name)
			require.Contains(t, ended[0].Attributes(), attribute.Int64("response_bytes", tc.wantWritten))
		})
	}
}
