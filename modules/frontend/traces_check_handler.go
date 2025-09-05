package frontend

import (
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/grafana/dskit/user"
	"github.com/grafana/tempo/modules/frontend/combiner"
	"github.com/grafana/tempo/modules/frontend/pipeline"
	"github.com/grafana/tempo/modules/overrides"
)

// newTracesCheckHandler creates a http.handler for traces check requests
func newTracesCheckHandler(cfg Config, next pipeline.AsyncRoundTripper[combiner.PipelineResponse], o overrides.Interface, logger log.Logger) http.RoundTripper {

	return RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		tenant, err := user.ExtractOrgID(req.Context())
		if err != nil {
			level.Error(logger).Log("msg", "traces check: failed to extract tenant id", "err", err)
			return &http.Response{
				StatusCode: http.StatusBadRequest,
				Status:     http.StatusText(http.StatusBadRequest),
				Body:       io.NopCloser(strings.NewReader(err.Error())),
			}, nil
		}

		level.Info(logger).Log(
			"msg", "traces check request",
			"tenant", tenant)

		comb := combiner.NewTracesCheck()
		rt := pipeline.NewHTTPCollector(next, cfg.ResponseConsumers, comb)

		start := time.Now()
		resp, err := rt.RoundTrip(req)
		elapsed := time.Since(start)

		level.Info(logger).Log(
			"msg", "traces check response",
			"tenant", tenant,
			"duration_seconds", elapsed.Seconds(),
			"err", err)

		return resp, err
	})
}
