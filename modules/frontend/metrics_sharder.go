package frontend

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level" //nolint:all deprecated
	"github.com/gogo/protobuf/jsonpb"
	"github.com/grafana/dskit/user"
	"github.com/opentracing/opentracing-go"

	"github.com/grafana/tempo/modules/frontend/pipeline"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/modules/querier"
	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/tempopb"
	common_v1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
	"github.com/grafana/tempo/pkg/traceql"
	"github.com/grafana/tempo/tempodb"
	"github.com/grafana/tempo/tempodb/backend"
)

type queryRangeSharder struct {
	next      pipeline.AsyncRoundTripper[*http.Response]
	reader    tempodb.Reader
	overrides overrides.Interface
	cfg       QueryRangeSharderConfig
	logger    log.Logger
}

// jpe - rename to metrics sharder? or rename everything back to query range?
type QueryRangeSharderConfig struct {
	ConcurrentRequests    int           `yaml:"concurrent_jobs,omitempty"`
	TargetBytesPerRequest int           `yaml:"target_bytes_per_job,omitempty"`
	MaxDuration           time.Duration `yaml:"max_duration"`
	QueryBackendAfter     time.Duration `yaml:"query_backend_after,omitempty"`
	Interval              time.Duration `yaml:"interval,omitempty"`
}

// newAsyncSearchSharder creates a sharding middleware for search
func newQueryRangeSharder(reader tempodb.Reader, o overrides.Interface, cfg SearchSharderConfig, logger log.Logger) pipeline.AsyncMiddleware[*http.Response] {
	return pipeline.AsyncMiddlewareFunc[*http.Response](func(next pipeline.AsyncRoundTripper[*http.Response]) pipeline.AsyncRoundTripper[*http.Response] {
		return asyncSearchSharder{
			next:      next,
			reader:    reader,
			overrides: o,

			cfg:    cfg,
			logger: logger,
		}
	})
}

func (s queryRangeSharder) RoundTrip(r *http.Request) (pipeline.Responses[*http.Response], error) {
	span, ctx := opentracing.StartSpanFromContext(r.Context(), "frontend.QueryRangeSharder")
	defer span.Finish()

	var (
		isProm bool
		err    error
		now    = time.Now()
	)

	// This route supports two flavors. (1) Prometheus-compatible (2) Tempo native
	// Remember which flavor this is and swap it so all
	// upstream calls are always Tempo native.
	if strings.Contains(r.RequestURI, api.PathPromQueryRange) {
		isProm = true
		// Swap upstream calls to the Tempo-native paths
		r.URL.Path = strings.ReplaceAll(r.URL.Path, api.PathPromQueryRange, api.PathMetricsQueryRange) // jpe - move to handler?
		r.RequestURI = strings.ReplaceAll(r.RequestURI, api.PathPromQueryRange, api.PathMetricsQueryRange)
		// Prom endpoint is called with 1-second precision timestamps
		// Round "now" to 1-second also.
		now = time.Unix(now.Unix(), 0)
	}

	req, err := api.ParseQueryRangeRequest(r)
	if err != nil {
		return s.respErrHandler(isProm, err)
	}

	expr, err := traceql.Parse(req.Query)
	if err != nil {
		return s.respErrHandler(isProm, err)
	}

	tenantID, err := user.ExtractOrgID(ctx)
	if err != nil {
		return s.respErrHandler(isProm, err)
	}

	alignTimeRange(req)

	// calculate and enforce max search duration
	maxDuration := s.maxDuration(tenantID)
	if maxDuration != 0 && time.Duration(req.End-req.Start)*time.Nanosecond > maxDuration {
		err = fmt.Errorf(fmt.Sprintf("range specified by start and end (%s) exceeds %s. received start=%d end=%d", time.Duration(req.End-req.Start), maxDuration, req.Start, req.End))
		return s.respErrHandler(isProm, err)
	}

	var (
		allowUnsafe           = s.overrides.UnsafeQueryHints(tenantID)
		samplingRate          = s.samplingRate(expr, allowUnsafe)
		targetBytesPerRequest = s.jobSize(expr, samplingRate, allowUnsafe)
		interval              = s.jobInterval(expr, allowUnsafe)
	)

	generatorReq := s.generatorRequest(*req, r, tenantID, now, samplingRate)
	reqCh := make(chan *http.Request, 2) // buffer of 2 allows us to insert generatorReq and metrics
	stopCh := make(chan struct{})
	defer close(stopCh)

	if generatorReq != nil {
		reqCh <- generatorReq
	}

	totalBlocks, totalBlockBytes := s.backendRequests(ctx, tenantID, r, *req, now, samplingRate, targetBytesPerRequest, interval, reqCh, func(err error) {
		// todo: actually find a way to return this error to the user
		s.logger.Log("msg", "query range: failed to build backend requests", "err", err)
	})

	// send a job to communicate the search metrics. this is consumed by the combiner to calculate totalblocks/bytes/jobs (jpe: make sure this propagates to the combiner)
	var jobMetricsResponse pipeline.Responses[*http.Response]
	if totalBlocks > 0 {
		resp := &tempopb.QueryRangeResponse{
			Metrics: &tempopb.SearchMetrics{
				TotalBlocks:     totalBlocks,
				TotalBlockBytes: totalBlockBytes, // jpe - total jobs?
			},
		}

		m := jsonpb.Marshaler{}
		body, err := m.MarshalToString(resp)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal search metrics: %w", err)
		}

		jobMetricsResponse = pipeline.NewSuccessfulResponse(body)
	}

	return pipeline.NewAsyncSharderChan(ctx, s.cfg.ConcurrentRequests, reqCh, jobMetricsResponse, s.next), nil
}

// blockMetas returns all relevant blockMetas given a start/end
func (s *queryRangeSharder) blockMetas(start, end int64, tenantID string) []*backend.BlockMeta {
	// reduce metas to those in the requested range
	allMetas := s.reader.BlockMetas(tenantID)
	metas := make([]*backend.BlockMeta, 0, len(allMetas)/50) // divide by 50 for luck
	for _, m := range allMetas {
		if m.StartTime.UnixNano() <= end &&
			m.EndTime.UnixNano() >= start {
			metas = append(metas, m)
		}
	}

	return metas
}

func (s *queryRangeSharder) backendRequests(ctx context.Context, tenantID string, parent *http.Request, searchReq tempopb.QueryRangeRequest, now time.Time, samplingRate float64, targetBytesPerRequest int, interval time.Duration, reqCh chan *http.Request, errFn func(error)) (totalBlocks uint32, totalBlockBytes uint64) {
	// request without start or end, search only in generator
	if searchReq.Start == 0 || searchReq.End == 0 {
		close(reqCh)
		return
	}

	// Make a copy and limit to backend time range.
	backendReq := searchReq
	backendReq.Start, backendReq.End = s.backendRange(now, backendReq.Start, backendReq.End, s.cfg.QueryBackendAfter)
	alignTimeRange(&backendReq)

	// If empty window then no need to search backend
	if backendReq.Start == backendReq.End {
		close(reqCh)
		return
	}

	// Blocks within overall time range. This is just for instrumentation, more precise time
	// range is checked for each window.
	blocks := s.blockMetas(int64(backendReq.Start), int64(backendReq.End), tenantID)
	if len(blocks) == 0 {
		// no need to search backend
		close(reqCh)
		return
	}

	totalBlocks = uint32(len(blocks))
	for _, b := range blocks {
		totalBlockBytes += b.Size
	}

	go func() {
		s.buildBackendRequests(ctx, tenantID, parent, backendReq, samplingRate, targetBytesPerRequest, interval, reqCh, errFn)
	}()

	return
}

// jpe - caching!
func (s *queryRangeSharder) buildBackendRequests(ctx context.Context, tenantID string, parent *http.Request, searchReq tempopb.QueryRangeRequest, samplingRate float64, targetBytesPerRequest int, interval time.Duration, reqCh chan *http.Request, errFn func(error)) {
	defer close(reqCh)

	var (
		start          = searchReq.Start
		end            = searchReq.End
		timeWindowSize = uint64(interval.Nanoseconds())
	)

	for start < end {

		thisStart := start
		thisEnd := start + timeWindowSize
		if thisEnd > end {
			thisEnd = end
		}

		blocks := s.blockMetas(int64(thisStart), int64(thisEnd), tenantID)
		if len(blocks) == 0 {
			start = thisEnd
			continue
		}

		totalBlockSize := uint64(0)
		for _, b := range blocks {
			totalBlockSize += b.Size
		}

		shards := uint32(math.Ceil(float64(totalBlockSize) / float64(targetBytesPerRequest)))

		for i := uint32(1); i <= shards; i++ {
			shardR := searchReq
			shardR.Start = thisStart
			shardR.End = thisEnd
			shardR.ShardID = i
			shardR.ShardCount = shards

			if samplingRate != 1.0 {
				shardR.ShardID *= uint32(1.0 / samplingRate)
				shardR.ShardCount *= uint32(1.0 / samplingRate)

				// Set final sampling rate after integer rounding
				// samplingRate = float64(shards) / float64(shardR.ShardCount) - jpe restore
			}

			select {
			case reqCh <- s.toUpstreamRequest(ctx, shardR, parent, tenantID):
			case <-ctx.Done():
				return
			}
		}

		start = thisEnd
	}
}

func (s *queryRangeSharder) backendRange(now time.Time, start, end uint64, queryBackendAfter time.Duration) (uint64, uint64) {
	backendAfter := uint64(now.Add(-queryBackendAfter).UnixNano())

	// adjust start/end if necessary. if the entire query range was inside backendAfter then
	// start will == end. This signals we don't need to query the backend.
	if end > backendAfter {
		end = backendAfter
	}
	if start > backendAfter {
		start = backendAfter
	}

	return start, end
}

func (s *queryRangeSharder) generatorRequest(searchReq tempopb.QueryRangeRequest, parent *http.Request, tenantID string, now time.Time, samplingRate float64) *http.Request {
	cutoff := uint64(now.Add(-s.cfg.QueryBackendAfter).UnixNano())

	// if there's no overlap between the query and ingester range just return nil
	if searchReq.End < cutoff {
		return nil
	}

	if searchReq.Start < cutoff {
		searchReq.Start = cutoff
	}

	alignTimeRange(&searchReq)

	// if start == end then we don't need to query it
	if searchReq.Start == searchReq.End {
		return nil
	}

	searchReq.QueryMode = querier.QueryModeRecent

	// No sharding on the generators (unnecessary), but we do apply sampling
	// rates.  In this case we execute a single arbitrary shard. Choosing
	// the last shard works. The first shard should be avoided because it is
	// weighted slightly off due to int63/128 sharding boundaries.
	searchReq.ShardID = uint32(1.0 / samplingRate)
	searchReq.ShardCount = uint32(1.0 / samplingRate)

	// Set final sampling rate after integer rounding
	// samplingRate = 1.0 / float64(searchReq.ShardCount) - restore?

	return s.toUpstreamRequest(parent.Context(), searchReq, parent, tenantID)
}

func (s *queryRangeSharder) toUpstreamRequest(ctx context.Context, req tempopb.QueryRangeRequest, parent *http.Request, tenantID string) *http.Request {
	subR := parent.Clone(ctx)
	subR = api.BuildQueryRangeRequest(subR, &req)

	prepareRequestForQueriers(subR, tenantID, parent.URL.Path, subR.URL.Query())
	return subR
}

// alignTimeRange shifts the start and end times of the request to align with the step
// interval.  This gives more consistent results across refreshes of queries like "last 1 hour".
// Without alignment each refresh is shifted by seconds or even milliseconds and the time series
// calculations are sublty different each time. It's not wrong, but less preferred behavior.
func alignTimeRange(req *tempopb.QueryRangeRequest) {
	// It doesn't really matter but the request fields are expected to be in nanoseconds.
	req.Start = req.Start / req.Step * req.Step
	req.End = req.End / req.Step * req.Step
}

// maxDuration returns the max search duration allowed for this tenant.
func (s *queryRangeSharder) maxDuration(tenantID string) time.Duration {
	// check overrides first, if no overrides then grab from our config
	maxDuration := s.overrides.MaxMetricsDuration(tenantID)
	if maxDuration != 0 {
		return maxDuration
	}

	return s.cfg.MaxDuration
}

func (s *queryRangeSharder) samplingRate(expr *traceql.RootExpr, allowUnsafe bool) float64 {
	samplingRate := 1.0
	if v, ok := expr.Hints.GetFloat(traceql.HintSample, allowUnsafe); ok {
		if v > 0 && v < 1.0 {
			samplingRate = v
		}
	}
	return samplingRate
}

func (s *queryRangeSharder) jobSize(expr *traceql.RootExpr, samplingRate float64, allowUnsafe bool) int {
	// If we have a query hint then use it
	if v, ok := expr.Hints.GetInt(traceql.HintJobSize, allowUnsafe); ok && v > 0 {
		return v
	}

	// Else use configured value.
	size := s.cfg.TargetBytesPerRequest

	// Automatically scale job size when sampling less than 100%
	// This improves performance.
	if samplingRate < 1.0 {
		factor := 1.0 / samplingRate

		// Keep it within reason
		if factor > 10.0 {
			factor = 10.0
		}

		size = int(float64(size) * factor)
	}

	return size
}

func (s *queryRangeSharder) jobInterval(expr *traceql.RootExpr, allowUnsafe bool) time.Duration {
	// If we have a query hint then use it
	if v, ok := expr.Hints.GetDuration(traceql.HintJobInterval, allowUnsafe); ok && v > 0 {
		return v
	}

	// Else use configured value
	return s.cfg.Interval
}

func (s *queryRangeSharder) convertToPromFormat(resp *tempopb.QueryRangeResponse) PromResponse {
	promResp := PromResponse{
		Status: "success",
		Data:   &PromData{ResultType: "matrix"},
	}

	for _, series := range resp.Series {
		promResult := PromResult{
			Metric: map[string]string{},
		}

		for _, label := range series.Labels {
			var s string
			switch v := label.Value.Value.(type) {
			case *common_v1.AnyValue_StringValue:
				s = v.StringValue
			case *common_v1.AnyValue_IntValue:
				s = strconv.Itoa(int(v.IntValue))
			case *common_v1.AnyValue_DoubleValue:
				s = strconv.FormatFloat(v.DoubleValue, 'g', -1, 64)
			case *common_v1.AnyValue_BoolValue:
				s = strconv.FormatBool(v.BoolValue)
			}
			promResult.Metric[label.Key] = s
		}

		promResult.Values = make([]interface{}, 0, len(series.Samples))
		for _, ts := range series.Samples {
			promResult.Values = append(promResult.Values, []interface{}{
				float64(ts.TimestampMs) / 1000.0,           // float for timestamp. assume it's seconds
				strconv.FormatFloat(ts.Value, 'f', -1, 64), // making assumptions about the float format returned from prom
			})
		}

		promResp.Data.Result = append(promResp.Data.Result, promResult)
	}

	return promResp
}

// returns an HTTP response or an error
func (s *queryRangeSharder) respErrHandler(isProm bool, err error) (pipeline.Responses[*http.Response], error) { // jpe - move to handler?
	if isProm {
		resp := s.convertToPromError(err)
		_ = level.Debug(s.logger).Log("resp", fmt.Sprintf("%+v", resp))

		bytes, marshalErr := json.Marshal(resp)
		if marshalErr != nil {
			return nil, fmt.Errorf("marshal failed with: %w: %w", marshalErr, err)
		}
		bodyString := string(bytes)

		return pipeline.NewSyncToAsyncResponse(&http.Response{
			StatusCode: http.StatusOK,
			Header: http.Header{
				api.HeaderContentType: {api.HeaderAcceptJSON},
			},
			Body:          io.NopCloser(strings.NewReader(bodyString)),
			ContentLength: int64(len([]byte(bodyString))),
		}), nil
	}

	return pipeline.NewSyncToAsyncResponse(&http.Response{
		StatusCode: http.StatusBadRequest,
		Body:       io.NopCloser(strings.NewReader(err.Error())),
	}), nil
}

func (s *queryRangeSharder) convertToPromError(err error) PromResponse {
	return PromResponse{
		Status:    "error",
		ErrorType: "bad_data",
		Error:     err.Error(),
	}
}

type PromResponse struct {
	Status    string    `json:"status"`
	Data      *PromData `json:"data,omitempty"`
	ErrorType string    `json:"errorType,omitempty"`
	Error     string    `json:"error,omitempty"`
}

type PromData struct {
	ResultType string       `json:"resultType"`
	Result     []PromResult `json:"result"`
}

type PromResult struct {
	Metric    map[string]string `json:"metric"`
	Values    []interface{}     `json:"values"`    // first entry is timestamp (float), second is value (string)
	Exemplars []interface{}     `json:"exemplars"` // first entry is timestamp (float), second is duration (float seconds), third is traceID (string)
}

type queryRangeJob struct {
	req          tempopb.QueryRangeRequest
	err          error
	samplingRate float64
}
