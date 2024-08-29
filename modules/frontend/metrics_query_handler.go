package frontend

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/grafana/dskit/user"
	"github.com/grafana/tempo/modules/frontend/combiner"
	"github.com/grafana/tempo/modules/frontend/pipeline"
	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/tempopb"
)

func newQueryInstantStreamingGRPCHandler(cfg Config, next pipeline.AsyncRoundTripper[combiner.PipelineResponse], apiPrefix string, logger log.Logger) streamingQueryInstantHandler {
	postSLOHook := metricsSLOPostHook(cfg.Metrics.SLO)
	downstreamPath := path.Join(apiPrefix, api.PathMetricsQueryRange)

	return func(req *tempopb.QueryInstantRequest, srv tempopb.StreamingQuerier_MetricsQueryInstantServer) error {
		start := time.Now()
		ctx := srv.Context()
		tenant, err := user.ExtractOrgID(ctx)
		if err != nil {
			return err
		}

		// --------------------------------------------------
		// Rewrite into a query_range request.
		// --------------------------------------------------
		qr := &tempopb.QueryRangeRequest{
			Query: req.Query,
			Start: req.Start,
			End:   req.End,
			Step:  req.End - req.Start,
		}
		httpReq := api.BuildQueryRangeRequest(&http.Request{
			URL:    &url.URL{Path: downstreamPath},
			Header: http.Header{},
			Body:   io.NopCloser(bytes.NewReader([]byte{})),
		}, qr)
		httpReq = httpReq.Clone(ctx)

		var finalResponse *tempopb.QueryInstantResponse
		c, err := combiner.NewTypedQueryRange(qr.Start, qr.End, qr.Step, qr.Query, true)
		if err != nil {
			return err
		}

		collector := pipeline.NewGRPCCollector(next, cfg.ResponseConsumers, c, func(qrr *tempopb.QueryRangeResponse) error {
			// Translate each diff into the instant version and send it
			resp := translateQueryRangeToInstant(*qrr)
			finalResponse = &resp // Save last response for bytesProcessed for the SLO calculations
			return srv.Send(&resp)
		})

		logQueryInstantRequest(logger, tenant, req)
		err = collector.RoundTrip(httpReq)

		duration := time.Since(start)
		bytesProcessed := uint64(0)
		if finalResponse != nil && finalResponse.Metrics != nil {
			bytesProcessed = finalResponse.Metrics.InspectedBytes
		}
		postSLOHook(nil, tenant, bytesProcessed, duration, err)
		logQueryInstantResult(logger, tenant, duration.Seconds(), req, finalResponse, err)
		return err
	}
}

// newMetricsQueryInstantHTTPHandler handles instant queries.  Internally these are rewritten as query_range with single step
// to make use of the existing pipeline.
func newMetricsQueryInstantHTTPHandler(cfg Config, next pipeline.AsyncRoundTripper[combiner.PipelineResponse], logger log.Logger) http.RoundTripper {
	postSLOHook := metricsSLOPostHook(cfg.Metrics.SLO)

	return RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		tenant, _ := user.ExtractOrgID(req.Context())
		start := time.Now()

		// Parse request
		i, err := api.ParseQueryInstantRequest(req)
		if err != nil {
			level.Error(logger).Log("msg", "query instant: parse search request failed", "err", err)
			return &http.Response{
				StatusCode: http.StatusBadRequest,
				Status:     http.StatusText(http.StatusBadRequest),
				Body:       io.NopCloser(strings.NewReader(err.Error())),
			}, nil
		}

		logQueryInstantRequest(logger, tenant, i)

		// --------------------------------------------------
		// Rewrite into a query_range request.
		// --------------------------------------------------
		qr := &tempopb.QueryRangeRequest{
			Query: i.Query,
			Start: i.Start,
			End:   i.End,
			Step:  i.End - i.Start,
		}

		// Clone existing to keep it unaltered.
		req = req.Clone(req.Context())
		req.URL.Path = strings.ReplaceAll(req.URL.Path, api.PathMetricsQueryInstant, api.PathMetricsQueryRange)
		req = api.BuildQueryRangeRequest(req, qr)

		combiner, err := combiner.NewTypedQueryRange(qr.Start, qr.End, qr.Step, qr.Query, false)
		if err != nil {
			level.Error(logger).Log("msg", "query instant: query range combiner failed", "err", err)
			return &http.Response{
				StatusCode: http.StatusBadRequest,
				Status:     http.StatusText(http.StatusBadRequest),
				Body:       io.NopCloser(strings.NewReader(err.Error())),
			}, nil
		}
		rt := pipeline.NewHTTPCollector(next, cfg.ResponseConsumers, combiner)

		// Roundtrip the request and look for intermediate failures
		innerResp, err := rt.RoundTrip(req)
		if err != nil {
			return nil, err
		}
		if innerResp != nil && innerResp.StatusCode != http.StatusOK {
			return innerResp, nil
		}

		// --------------------------------------------------
		// Get the final data and translate to instant.
		// --------------------------------------------------
		qrResp, err := combiner.GRPCFinal()
		if err != nil {
			return nil, err
		}

		qiResp := translateQueryRangeToInstant(*qrResp)

		bodyString, err := new(jsonpb.Marshaler).MarshalToString(&qiResp)
		if err != nil {
			return nil, fmt.Errorf("error marshalling response body: %w", err)
		}

		resp := &http.Response{
			StatusCode: combiner.StatusCode(),
			Header: http.Header{
				api.HeaderContentType: {api.HeaderAcceptJSON},
			},
			Body:          io.NopCloser(strings.NewReader(bodyString)),
			ContentLength: int64(len([]byte(bodyString))),
		}

		duration := time.Since(start)
		var bytesProcessed uint64
		if qiResp.Metrics != nil {
			bytesProcessed = qiResp.Metrics.InspectedBytes
		}
		postSLOHook(resp, tenant, bytesProcessed, duration, err)
		logQueryInstantResult(logger, tenant, duration.Seconds(), i, &qiResp, err)

		return resp, nil
	})
}

func translateQueryRangeToInstant(input tempopb.QueryRangeResponse) tempopb.QueryInstantResponse {
	output := tempopb.QueryInstantResponse{
		Metrics: input.Metrics,
	}
	for _, series := range input.Series {
		if len(series.Samples) == 0 {
			continue
		}
		// Use first value
		output.Series = append(output.Series, &tempopb.InstantSeries{
			Labels:     series.Labels,
			PromLabels: series.PromLabels,
			Value:      series.Samples[0].Value,
		})
	}
	return output
}

func logQueryInstantResult(logger log.Logger, tenantID string, durationSeconds float64, req *tempopb.QueryInstantRequest, resp *tempopb.QueryInstantResponse, err error) {
	if resp == nil {
		level.Info(logger).Log(
			"msg", "query instant results - no resp",
			"tenant", tenantID,
			"duration_seconds", durationSeconds,
			"error", err)

		return
	}

	if resp.Metrics == nil {
		level.Info(logger).Log(
			"msg", "query instant results - no metrics",
			"tenant", tenantID,
			"query", req.Query,
			"range_nanos", req.End-req.Start,
			"duration_seconds", durationSeconds,
			"error", err)
		return
	}

	level.Info(logger).Log(
		"msg", "query instant results",
		"tenant", tenantID,
		"query", req.Query,
		"range_nanos", req.End-req.Start,
		"duration_seconds", durationSeconds,
		"request_throughput", float64(resp.Metrics.InspectedBytes)/durationSeconds,
		"total_requests", resp.Metrics.TotalJobs,
		"total_blockBytes", resp.Metrics.TotalBlockBytes,
		"total_blocks", resp.Metrics.TotalBlocks,
		"completed_requests", resp.Metrics.CompletedJobs,
		"inspected_bytes", resp.Metrics.InspectedBytes,
		"inspected_traces", resp.Metrics.InspectedTraces,
		"inspected_spans", resp.Metrics.InspectedSpans,
		"error", err)
}

func logQueryInstantRequest(logger log.Logger, tenantID string, req *tempopb.QueryInstantRequest) {
	level.Info(logger).Log(
		"msg", "query instant request",
		"tenant", tenantID,
		"query", req.Query,
		"range_seconds", req.End-req.Start)
}
