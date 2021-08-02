package frontend

import (
	"net/http"

	"github.com/opentracing/opentracing-go"
	ot_log "github.com/opentracing/opentracing-go/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/weaveworks/common/httpgrpc"
)

func RetryWare(maxRetries int, registerer prometheus.Registerer) Middleware {
	return MiddlewareFunc(func(next Handler) Handler {
		return retryWare{
			next:       next,
			maxRetries: maxRetries,
			retriesCount: promauto.With(registerer).NewHistogram(prometheus.HistogramOpts{
				Namespace: "tempo",
				Name:      "query_frontend_retries",
				Help:      "Number of times a request is retried.",
				Buckets:   []float64{0, 1, 2, 3, 4, 5},
			}),
		}
	})
}

type retryWare struct {
	next         Handler
	maxRetries   int
	retriesCount prometheus.Histogram
}

// Do implements Handler
func (r retryWare) Do(req *http.Request) (*http.Response, error) {
	ctx := req.Context()
	span, ctx := opentracing.StartSpanFromContext(ctx, "frontend.Retry")
	defer span.Finish()

	// context propagation
	req = req.WithContext(ctx)

	tries := 0
	defer func() { r.retriesCount.Observe(float64(tries)) }()

	for {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		resp, err := r.next.Do(req)

		// do not retry if no error and reponse is not HTTP 5xx
		if err == nil && resp.StatusCode/100 != 5 {
			return resp, nil
		}

		// do not retry if GRPC error contains response that is not HTTP 5xx
		httpResp, ok := httpgrpc.HTTPResponseFromError(err)
		if ok && httpResp.Code/100 != 5 {
			return resp, err
		}

		// reached max retries
		tries++
		if tries >= r.maxRetries {
			return resp, err
		}

		span.LogFields(ot_log.String("msg", "error processing request. retrying"))
	}
}
