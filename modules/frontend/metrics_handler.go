package frontend

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level" //nolint:all //deprecated
	"github.com/grafana/dskit/user"
	"github.com/grafana/tempo/modules/frontend/combiner"
	"github.com/grafana/tempo/modules/frontend/pipeline"

	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/tempopb"
)

// newQueryRangeStreamingGRPCHandler returns a handler that streams results from the HTTP handler
func newQueryRangeStreamingGRPCHandler(cfg Config, next pipeline.AsyncRoundTripper[*http.Response], apiPrefix string, logger log.Logger) streamingQueryRangeHandler {
	postSLOHook := searchSLOPostHook(cfg.Search.SLO)
	downstreamPath := path.Join(apiPrefix, api.PathMetricsQueryRange)

	return func(req *tempopb.QueryRangeRequest, srv tempopb.StreamingQuerier_QueryRangeServer) error {
		httpReq := api.BuildQueryRangeRequest(&http.Request{
			URL:    &url.URL{Path: downstreamPath}, // jpe - foo
			Header: http.Header{},
			Body:   io.NopCloser(bytes.NewReader([]byte{})),
		}, req)

		ctx := srv.Context()
		httpReq = httpReq.WithContext(ctx)
		tenant, _ := user.ExtractOrgID(ctx)
		start := time.Now()

		var finalResponse *tempopb.QueryRangeResponse
		c := combiner.NewTypedQueryRange(false) // jpe - isProm?
		collector := pipeline.NewGRPCCollector[*tempopb.QueryRangeResponse](next, c, func(qrr *tempopb.QueryRangeResponse) error {
			finalResponse = qrr // sadly we can't srv.Send directly into the collector. we need bytesProcessed for the SLO calculations
			return srv.Send(qrr)
		})

		logQueryRangeRequest(logger, tenant, req)
		err := collector.RoundTrip(httpReq)

		duration := time.Since(start)
		bytesProcessed := uint64(0)
		if finalResponse != nil && finalResponse.Metrics != nil {
			bytesProcessed = finalResponse.Metrics.InspectedBytes
		}
		postSLOHook(nil, tenant, bytesProcessed, duration, err)
		logQueryRangeResult(logger, tenant, duration.Seconds(), req, finalResponse, err)
		return err
	}
}

// newQueryRangeHTTPHandler returns a handler that returns a single response from the HTTP handler
func newQueryRangeHTTPHandler(cfg Config, next pipeline.AsyncRoundTripper[*http.Response], logger log.Logger) http.RoundTripper {
	postSLOHook := searchSLOPostHook(cfg.Search.SLO)

	return pipeline.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		tenant, _ := user.ExtractOrgID(req.Context())
		start := time.Now()

		// parse request
		queryRangeReq, err := api.ParseQueryRangeRequest(req)
		if err != nil {
			level.Error(logger).Log("msg", "query range: parse search request failed", "err", err)
			return &http.Response{
				StatusCode: http.StatusBadRequest,
				Status:     http.StatusText(http.StatusBadRequest),
				Body:       io.NopCloser(strings.NewReader(err.Error())),
			}, nil
		}

		logQueryRangeRequest(logger, tenant, queryRangeReq)

		// build and use roundtripper
		combiner := combiner.NewTypedQueryRange(false) // jpe - isProm?
		rt := pipeline.NewHTTPCollector(next, combiner)

		resp, err := rt.RoundTrip(req)

		// ask for the typed diff and use that for the SLO hook. it will have up to date metrics
		var bytesProcessed uint64
		queryRangeResp, _ := combiner.GRPCDiff()
		if queryRangeResp != nil && queryRangeResp.Metrics != nil {
			bytesProcessed = queryRangeResp.Metrics.InspectedBytes
		}

		duration := time.Since(start)
		postSLOHook(resp, tenant, bytesProcessed, duration, err)
		logQueryRangeResult(logger, tenant, duration.Seconds(), queryRangeReq, queryRangeResp, err)
		return resp, err
	})
}

func logQueryRangeResult(logger log.Logger, tenantID string, durationSeconds float64, req *tempopb.QueryRangeRequest, resp *tempopb.QueryRangeResponse, err error) { // jpe standardize these?
	if resp == nil {
		level.Info(logger).Log(
			"msg", "search results - no resp",
			"tenant", tenantID,
			"duration_seconds", durationSeconds,
			"error", err)

		return
	}

	if resp.Metrics == nil {
		level.Info(logger).Log(
			"msg", "search results - no metrics",
			"tenant", tenantID,
			"query", req.Query,
			"range_seconds", req.End-req.Start,
			"duration_seconds", durationSeconds,
			"error", err)
		return
	}

	level.Info(logger).Log(
		"msg", "search results",
		"tenant", tenantID,
		"query", req.Query,
		"range_seconds", req.End-req.Start,
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

func logQueryRangeRequest(logger log.Logger, tenantID string, req *tempopb.QueryRangeRequest) { // jpe standardize these?
	level.Info(logger).Log(
		"msg", "search request",
		"tenant", tenantID,
		"query", req.Query,
		"range_seconds", req.End-req.Start,
		"mode", req.QueryMode,
		"step", req.Step)
}

/*
	for job := range reqCh {
		if job.err != nil {
			jobErr.Store(fmt.Errorf("unexpected err building reqs: %w", job.err))
			break
		}

		if jErr := jobErr.Load(); jErr != nil {
			break
		}

		// When we hit capacity of boundedwaitgroup, wg.Add will block
		wg.Add(1)
		startedReqs++

		go func(job *queryRangeJob) {
			defer wg.Done()

			innerR := s.toUpstreamRequest(subCtx, job.req, r, tenantID)
			resp, err := s.next.RoundTrip(innerR)
			if err != nil {
				// context cancelled error happens when we exit early.
				// bail, and don't log and don't set this error.
				if errors.Is(err, context.Canceled) {
					_ = level.Debug(s.logger).Log("msg", "exiting early from sharded query", "url", innerR.RequestURI, "err", err)
					return
				}

				_ = level.Error(s.logger).Log("msg", "error executing sharded query", "url", innerR.RequestURI, "err", err)
				return
			}

			// if the status code is anything but happy, save the error and pass it down the line
			if resp.StatusCode != http.StatusOK {
				bytesMsg, err := io.ReadAll(resp.Body)
				if err != nil {
					_ = level.Error(s.logger).Log("msg", "error reading response body status != ok", "url", innerR.RequestURI, "err", err)
				}
				statusMsg := fmt.Sprintf("upstream: (%d) %s", resp.StatusCode, string(bytesMsg))
				jobErr.Store(fmt.Errorf(statusMsg))
				return
			}

			// successful query, read the body
			results := &tempopb.QueryRangeResponse{}
			err = (&jsonpb.Unmarshaler{AllowUnknownFields: true}).Unmarshal(resp.Body, results)
			if err != nil {
				_ = level.Error(s.logger).Log("msg", "error reading response body status == ok", "url", innerR.RequestURI, "err", err)
				return
			}

			// Multiply up the sampling rate
			if job.samplingRate != 1.0 {
				for _, series := range results.Series {
					for i, sample := range series.Samples {
						sample.Value *= 1.0 / job.samplingRate
						series.Samples[i] = sample
					}
				}
			}

			mtx.Lock()
			defer mtx.Unlock()
			c.Combine(results)
		}(job)
	}

	// wait for all goroutines running in wg to finish or cancelled
	wg.Wait()

	res := c.Response()
	res.Metrics.CompletedJobs = uint32(startedReqs)
	res.Metrics.TotalBlocks = uint32(totalBlocks)
	res.Metrics.TotalBlockBytes = uint64(totalBlockBytes)

	// Sort all output, series alphabetically, samples by time
	sort.SliceStable(res.Series, func(i, j int) bool {
		return strings.Compare(res.Series[i].PromLabels, res.Series[j].PromLabels) == -1
	})
	for _, series := range res.Series {
		sort.Slice(series.Samples, func(i, j int) bool {
			return series.Samples[i].TimestampMs < series.Samples[j].TimestampMs
		})
	}

	var (
		reqTime        = time.Since(now)
		throughput     = math.Round(float64(res.Metrics.InspectedBytes) / reqTime.Seconds())
		spanThroughput = math.Round(float64(res.Metrics.InspectedSpans) / reqTime.Seconds())
	)

	span.SetTag("totalBlocks", res.Metrics.TotalBlocks)
	span.SetTag("inspectedBytes", res.Metrics.InspectedBytes)
	span.SetTag("inspectedTraces", res.Metrics.InspectedTraces)
	span.SetTag("inspectedSpans", res.Metrics.InspectedSpans)
	span.SetTag("totalBlockBytes", res.Metrics.TotalBlockBytes)
	span.SetTag("totalJobs", res.Metrics.TotalJobs)
	span.SetTag("finishedJobs", res.Metrics.CompletedJobs)
	span.SetTag("requestThroughput", throughput)
	span.SetTag("spanThroughput", spanThroughput)

	if jErr := jobErr.Load(); jErr != nil {
		return s.respErrHandler(isProm, jErr)
	}

	var bodyString string
	if isProm {
		promResp := s.convertToPromFormat(res)
		bytes, err := json.Marshal(promResp)
		if err != nil {
			return nil, err
		}
		bodyString = string(bytes)
	} else {
		m := &jsonpb.Marshaler{EmitDefaults: true}
		bodyString, err = m.MarshalToString(res)
		if err != nil {
			return nil, err
		}
	}

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header: http.Header{
			api.HeaderContentType: {api.HeaderAcceptJSON},
		},
		Body:          io.NopCloser(strings.NewReader(bodyString)),
		ContentLength: int64(len([]byte(bodyString))),
	}

	return resp, nil
*/
