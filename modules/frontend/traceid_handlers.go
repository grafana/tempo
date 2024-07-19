package frontend

import (
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level" //nolint:all //deprecated
	"github.com/grafana/dskit/user"
	"github.com/grafana/tempo/modules/frontend/combiner"
	"github.com/grafana/tempo/modules/frontend/pipeline"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/api"
)

// newTraceIDHandler creates a http.handler for trace by id requests
func newTraceIDHandler(cfg Config, o overrides.Interface, next pipeline.AsyncRoundTripper[combiner.PipelineResponse], logger log.Logger) http.RoundTripper {
	postSLOHook := traceByIDSLOPostHook(cfg.TraceByID.SLO)

	return RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		tenant, err := user.ExtractOrgID(req.Context())
		if err != nil {
			level.Error(logger).Log("msg", "trace id: failed to extract tenant id", "err", err)
			return &http.Response{
				StatusCode: http.StatusBadRequest,
				Status:     http.StatusText(http.StatusBadRequest),
				Body:       io.NopCloser(strings.NewReader(err.Error())),
			}, nil
		}

		// validate traceID
		_, err = api.ParseTraceID(req)
		if err != nil {
			return &http.Response{
				StatusCode: http.StatusBadRequest,
				Body:       io.NopCloser(strings.NewReader(err.Error())),
				Header:     http.Header{},
			}, nil
		}

		// validate start and end parameter
		_, _, _, _, _, reqErr := api.ValidateAndSanitizeRequest(req)
		if reqErr != nil {
			return &http.Response{
				StatusCode: http.StatusBadRequest,
				Body:       io.NopCloser(strings.NewReader(reqErr.Error())),
				Header:     http.Header{},
			}, nil
		}

		// check marshalling format
		marshallingFormat := api.HeaderAcceptJSON
		if req.Header.Get(api.HeaderAccept) == api.HeaderAcceptProtobuf {
			marshallingFormat = api.HeaderAcceptProtobuf
		}

		// enforce all communication internal to Tempo to be in protobuf bytes
		req.Header.Set(api.HeaderAccept, api.HeaderAcceptProtobuf)

		level.Info(logger).Log(
			"msg", "trace id request",
			"tenant", tenant,
			"path", req.URL.Path)

		combiner := combiner.NewTraceByID(o.MaxBytesPerTrace(tenant), marshallingFormat)
		rt := pipeline.NewHTTPCollector(next, cfg.ResponseConsumers, combiner)

		start := time.Now()
		resp, err := rt.RoundTrip(req)

		elapsed := time.Since(start)
		postSLOHook(resp, tenant, 0, elapsed, err)

		level.Info(logger).Log(
			"msg", "trace id response",
			"tenant", tenant,
			"path", req.URL.Path,
			"duration_seconds", elapsed.Seconds(),
			"err", err)

		return resp, err
	})
}
