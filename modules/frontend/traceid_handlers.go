package frontend

import (
	"net/http"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level" //nolint:all //deprecated
	"github.com/grafana/tempo/modules/frontend/combiner"
	"github.com/grafana/tempo/modules/frontend/pipeline"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/tempopb"
)

// newTraceIDHandler creates a http.handler for trace by id requests
func newTraceIDHandler(cfg Config, next pipeline.AsyncRoundTripper[combiner.PipelineResponse], o overrides.Interface, combinerFn func(int, string, combiner.TraceRedactor) *combiner.TraceByIDCombiner, logger log.Logger, dataAccessController DataAccessController) http.RoundTripper {
	postSLOHook := traceByIDSLOPostHook(cfg.TraceByID.SLO)

	return RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		tenant, errResp := extractTenant(req, logger)
		if errResp != nil {
			return errResp, nil
		}

		// validate traceID
		_, err := api.ParseTraceID(req)
		if err != nil {
			return httpInvalidRequest(err), nil
		}

		// validate start and end parameter
		_, _, _, _, _, _, reqErr := api.ValidateAndSanitizeRequest(req)
		if reqErr != nil {
			return httpInvalidRequest(reqErr), nil
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

		var traceRedactor combiner.TraceRedactor
		if dataAccessController != nil {
			traceRedactor, err = dataAccessController.HandleHTTPTraceByIDReq(req)
			if err != nil {
				level.Error(logger).Log("msg", "trace id: failed to get trace redactor", "err", err)
				return httpInvalidRequest(err), nil
			}
		}

		comb := combinerFn(o.MaxBytesPerTrace(tenant), marshallingFormat, traceRedactor)
		rt := pipeline.NewHTTPCollector(next, cfg.ResponseConsumers, comb)

		start := time.Now()
		resp, err := rt.RoundTrip(req)
		elapsed := time.Since(start)

		var inspectBytes uint64
		if comb.MetricsCombiner != nil && comb.MetricsCombiner.Metrics != nil {
			inspectBytes = comb.MetricsCombiner.Metrics.InspectedBytes
		}
		postSLOHook(resp, tenant, inspectBytes, elapsed, err)

		level.Info(logger).Log(
			"msg", "trace id response",
			"tenant", tenant,
			"path", req.URL.Path,
			"duration_seconds", elapsed.Seconds(),
			"inspected_bytes", inspectBytes,
			"request_throughput", float64(inspectBytes)/elapsed.Seconds(),
			"err", err)

		return resp, err
	})
}

// newTraceIDV2Handler creates a http.handler for trace by id requests
func newTraceIDV2Handler(cfg Config, next pipeline.AsyncRoundTripper[combiner.PipelineResponse], o overrides.Interface, combinerFn func(int, string, combiner.TraceRedactor) combiner.GRPCCombiner[*tempopb.TraceByIDResponse], logger log.Logger, dataAccessController DataAccessController) http.RoundTripper {
	postSLOHook := traceByIDSLOPostHook(cfg.TraceByID.SLO)

	return RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		tenant, errResp := extractTenant(req, logger)
		if errResp != nil {
			return errResp, nil
		}

		// validate traceID
		_, err := api.ParseTraceID(req)
		if err != nil {
			return httpInvalidRequest(err), nil
		}

		// validate start and end parameter
		_, _, _, _, _, _, reqErr := api.ValidateAndSanitizeRequest(req)
		if reqErr != nil {
			return httpInvalidRequest(reqErr), nil
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

		var traceRedactor combiner.TraceRedactor
		if dataAccessController != nil {
			traceRedactor, err = dataAccessController.HandleHTTPTraceByIDReq(req)
			if err != nil {
				level.Error(logger).Log("msg", "trace id v2: failed to get trace redactor", "err", err)
				return httpInvalidRequest(err), nil
			}
		}

		comb := combinerFn(o.MaxBytesPerTrace(tenant), marshallingFormat, traceRedactor)
		rt := pipeline.NewHTTPCollector(next, cfg.ResponseConsumers, comb)

		start := time.Now()
		resp, err := rt.RoundTrip(req)
		elapsed := time.Since(start)

		var bytesProcessed uint64
		findResp, _ := comb.GRPCFinal()
		if findResp != nil && findResp.Metrics != nil {
			bytesProcessed = findResp.Metrics.InspectedBytes
		}

		postSLOHook(resp, tenant, bytesProcessed, elapsed, err)

		level.Info(logger).Log(
			"msg", "trace id response",
			"tenant", tenant,
			"path", req.URL.Path,
			"inspected_bytes", bytesProcessed,
			"request_throughput", float64(bytesProcessed)/elapsed.Seconds(),
			"duration_seconds", elapsed.Seconds(),
			"err", err)

		return resp, err
	})
}
