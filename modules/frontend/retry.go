package frontend

import (
	"fmt"
	"net/http"

	"github.com/grafana/dskit/httpgrpc"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	ot_log "github.com/opentracing/opentracing-go/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

func newRetryWare(maxRetries int, registerer prometheus.Registerer) Middleware {
	retriesCount := promauto.With(registerer).NewHistogram(prometheus.HistogramOpts{
		Namespace: "tempo",
		Name:      "query_frontend_retries",
		Help:      "Number of times a request is retried.",
		Buckets:   []float64{0, 1, 2, 3, 4, 5},
	})

	return MiddlewareFunc(func(next http.RoundTripper) http.RoundTripper {
		return retryWare{
			next:         next,
			maxRetries:   maxRetries,
			retriesCount: retriesCount,
		}
	})
}

type retryWare struct {
	next         http.RoundTripper
	maxRetries   int
	retriesCount prometheus.Histogram
}

// RoundTrip implements http.RoundTripper
func (r retryWare) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx := req.Context()
	span, ctx := opentracing.StartSpanFromContext(ctx, "frontend.Retry")
	defer span.Finish()
	ext.SpanKindRPCClient.Set(span)

	// context propagation
	req = req.WithContext(ctx)

	tries := 0
	defer func() { r.retriesCount.Observe(float64(tries)) }()

	for {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		resp, err := r.next.RoundTrip(req)

		// do not retry if no error and response is not HTTP 5xx
		if err == nil && !shouldRetry(resp.StatusCode) {
			return resp, nil
		}

		// do not retry if GRPC error contains response that is not HTTP 5xx
		httpResp, ok := httpgrpc.HTTPResponseFromError(err)
		if ok && !shouldRetry(int(httpResp.Code)) {
			return resp, err
		}

		// reached max retries
		tries++
		if tries >= r.maxRetries {
			return resp, err
		}

		statusCode := 0
		if resp != nil {
			statusCode = resp.StatusCode
		}
		if httpResp != nil {
			statusCode = int(httpResp.Code)
		}

		// avoid calling err.Error() on an error returned by frontend middleware
		// https://github.com/grafana/tempo/issues/857
		errMsg := fmt.Sprint(err)

		span.LogFields(
			ot_log.String("msg", "error processing request. retrying"),
			ot_log.Int("try", tries),
			ot_log.Int("status_code", statusCode),
			ot_log.String("errMsg", errMsg),
		)
	}
}

func shouldRetry(statusCode int) bool {
	return statusCode/100 == 5
}
