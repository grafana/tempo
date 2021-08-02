package frontend

import (
	"net/http"

	"github.com/go-kit/kit/log"
	"github.com/opentracing/opentracing-go"
	ot_log "github.com/opentracing/opentracing-go/log"
)

func RetryWare(maxRetries int, logger log.Logger) Middleware {
	return MiddlewareFunc(func(next Handler) Handler {
		return retryWare{
			next:       next,
			logger:     logger,
			maxRetries: maxRetries,
		}
	})
}

type retryWare struct {
	next       Handler
	logger     log.Logger
	maxRetries int
}

// Do implements Handler
func (r retryWare) Do(req *http.Request) (*http.Response, error) {
	ctx := req.Context()
	span, ctx := opentracing.StartSpanFromContext(ctx, "frontend.Retry")
	defer span.Finish()

	// context propagation
	req = req.WithContext(ctx)

	try := 0

	for {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		resp, err := r.next.Do(req)

		try++
		if try >= r.maxRetries {
			return resp, err
		}

		if err == nil && resp.StatusCode/100 != 5 {
			return resp, nil
		}

		statusCode := 0
		if resp != nil {
			statusCode = resp.StatusCode
		}

		span.LogFields(
			ot_log.String("msg", "error processing request"),
			ot_log.Int("status_code", statusCode),
			ot_log.Error(err),
			ot_log.Int("try", try),
		)
	}
}
