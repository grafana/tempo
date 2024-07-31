package pipeline

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/grafana/dskit/httpgrpc"
	"github.com/grafana/tempo/v2/modules/frontend/queue"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	ot_log "github.com/opentracing/opentracing-go/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

func NewRetryWare(maxRetries int, registerer prometheus.Registerer) Middleware {
	retriesCount := promauto.With(registerer).NewHistogram(prometheus.HistogramOpts{
		Namespace:                       "tempo",
		Name:                            "query_frontend_retries",
		Help:                            "Number of times a request is retried.",
		Buckets:                         []float64{0, 1, 2, 3, 4, 5},
		NativeHistogramBucketFactor:     1.1,
		NativeHistogramMaxBucketNumber:  100,
		NativeHistogramMinResetDuration: 1 * time.Hour,
	})

	return MiddlewareFunc(func(next RoundTripper) RoundTripper {
		return retryWare{
			next:         next,
			maxRetries:   maxRetries,
			retriesCount: retriesCount,
		}
	})
}

type retryWare struct {
	next         RoundTripper
	maxRetries   int
	retriesCount prometheus.Histogram
}

// RoundTrip implements http.RoundTripper
func (r retryWare) RoundTrip(req Request) (*http.Response, error) {
	ctx := req.Context()
	span, ctx := opentracing.StartSpanFromContext(ctx, "frontend.Retry")
	defer span.Finish()
	ext.SpanKindRPCClient.Set(span)

	// context propagation
	req.WithContext(ctx)

	tries := 0
	defer func() { r.retriesCount.Observe(float64(tries)) }()

	for {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		resp, err := r.next.RoundTrip(req)

		if r.maxRetries == 0 {
			return resp, err
		}

		// do not retry if no error and response is not HTTP 5xx
		if err == nil && resp != nil && !shouldRetry(resp.StatusCode) {
			return resp, nil
		}

		/* ---- HTTP GRPC translation ---- */
		// the following 2 blocks translate httpgrpc errors into something
		// the rest of the http pipeline can understand. these really should
		// be there own pipeline item independent of the retry middleware, but
		// they've always been here and for safety we're not moving them right now.
		if errors.Is(err, queue.ErrTooManyRequests) {
			return &http.Response{
				StatusCode: http.StatusTooManyRequests,
				Status:     http.StatusText(http.StatusTooManyRequests),
				Body:       io.NopCloser(strings.NewReader("job queue full")),
			}, nil
		}

		// do not retry if GRPC error contains response that is not HTTP 5xx
		httpResp, ok := httpgrpc.HTTPResponseFromError(err)
		if ok && !shouldRetry(int(httpResp.Code)) {
			return resp, err
		}
		/* ---- HTTP GRPC translation ---- */

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
