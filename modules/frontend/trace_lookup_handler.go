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

// newTraceLookupHandler creates a http.handler for trace lookup requests
func newTraceLookupHandler(cfg Config, next pipeline.AsyncRoundTripper[combiner.PipelineResponse], o overrides.Interface, logger log.Logger) http.RoundTripper {

	return RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		tenant, err := user.ExtractOrgID(req.Context())
		if err != nil {
			level.Error(logger).Log("msg", "trace lookup: failed to extract tenant id", "err", err)
			return &http.Response{
				StatusCode: http.StatusBadRequest,
				Status:     http.StatusText(http.StatusBadRequest),
				Body:       io.NopCloser(strings.NewReader(err.Error())),
			}, nil
		}

		level.Info(logger).Log(
			"msg", "trace lookup request",
			"tenant", tenant)

		comb := combiner.NewTraceLookup()
		rt := pipeline.NewHTTPCollector(next, cfg.ResponseConsumers, comb)

		start := time.Now()
		resp, err := rt.RoundTrip(req)
		elapsed := time.Since(start)

		level.Info(logger).Log(
			"msg", "trace lookup response",
			"tenant", tenant,
			"duration_seconds", elapsed.Seconds(),
			"err", err)

		return resp, err
	})
}