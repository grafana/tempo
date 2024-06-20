package frontend

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"time"

	"github.com/go-kit/log" //nolint:all deprecated
	"github.com/gogo/protobuf/jsonpb"
	"github.com/grafana/dskit/user"
	"github.com/opentracing/opentracing-go"

	"github.com/grafana/tempo/modules/frontend/combiner"
	"github.com/grafana/tempo/modules/frontend/pipeline"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/modules/querier"
	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/traceql"
	"github.com/grafana/tempo/tempodb"
	"github.com/grafana/tempo/tempodb/backend"
)

type queryRangeSharder struct {
	next              pipeline.AsyncRoundTripper[combiner.PipelineResponse]
	reader            tempodb.Reader
	overrides         overrides.Interface
	cfg               QueryRangeSharderConfig
	logger            log.Logger
	replicationFactor uint32
}

type QueryRangeSharderConfig struct {
	ConcurrentRequests    int           `yaml:"concurrent_jobs,omitempty"`
	TargetBytesPerRequest int           `yaml:"target_bytes_per_job,omitempty"`
	MaxDuration           time.Duration `yaml:"max_duration"`
	QueryBackendAfter     time.Duration `yaml:"query_backend_after,omitempty"`
	Interval              time.Duration `yaml:"interval,omitempty"`
	RF1ReadPath           bool          `yaml:"rf1_read_path,omitempty"`
}

// newAsyncQueryRangeSharder creates a sharding middleware for search
func newAsyncQueryRangeSharder(reader tempodb.Reader, o overrides.Interface, cfg QueryRangeSharderConfig, logger log.Logger) pipeline.AsyncMiddleware[combiner.PipelineResponse] {
	var replicationFactor uint32
	if cfg.RF1ReadPath {
		replicationFactor = 1
	}
	return pipeline.AsyncMiddlewareFunc[combiner.PipelineResponse](func(next pipeline.AsyncRoundTripper[combiner.PipelineResponse]) pipeline.AsyncRoundTripper[combiner.PipelineResponse] {
		return queryRangeSharder{
			next:      next,
			reader:    reader,
			overrides: o,

			cfg:    cfg,
			logger: logger,

			replicationFactor: replicationFactor,
		}
	})
}

func (s queryRangeSharder) RoundTrip(r *http.Request) (pipeline.Responses[combiner.PipelineResponse], error) {
	span, ctx := opentracing.StartSpanFromContext(r.Context(), "frontend.QueryRangeSharder")
	defer span.Finish()

	var (
		err error
		now = time.Now()
	)

	req, err := api.ParseQueryRangeRequest(r)
	if err != nil {
		return pipeline.NewBadRequest(err), nil
	}

	expr, _, _, _, err := traceql.NewEngine().Compile(req.Query)
	if err != nil {
		return pipeline.NewBadRequest(err), nil
	}

	tenantID, err := user.ExtractOrgID(ctx)
	if err != nil {
		return pipeline.NewBadRequest(err), nil
	}

	if req.Step == 0 {
		return pipeline.NewBadRequest(errors.New("step must be greater than 0")), nil
	}
	alignTimeRange(req)

	// calculate and enforce max search duration
	maxDuration := s.maxDuration(tenantID)
	if maxDuration != 0 && time.Duration(req.End-req.Start)*time.Nanosecond > maxDuration {
		err = fmt.Errorf(fmt.Sprintf("range specified by start and end (%s) exceeds %s. received start=%d end=%d", time.Duration(req.End-req.Start), maxDuration, req.Start, req.End))
		return pipeline.NewBadRequest(err), nil
	}

	var (
		allowUnsafe           = s.overrides.UnsafeQueryHints(tenantID)
		samplingRate          = s.samplingRate(expr, allowUnsafe)
		targetBytesPerRequest = s.jobSize(expr, samplingRate, allowUnsafe)
		interval              = s.jobInterval(expr, allowUnsafe)
	)

	// if interval is 0 then the backend requests code will loop forever. technically if we are here with a 0 interval it should mean a bad request
	// b/c it was specified using a query hint. we're going to assume that and return 400
	if interval == 0 {
		return pipeline.NewBadRequest(errors.New("invalid interval specified: 0")), nil
	}

	generatorReq := s.generatorRequest(*req, r, tenantID, now, samplingRate)
	reqCh := make(chan *http.Request, 2) // buffer of 2 allows us to insert generatorReq and metrics

	if generatorReq != nil {
		reqCh <- generatorReq
	}

	var (
		totalJobs, totalBlocks uint32
		totalBlockBytes        uint64
	)
	if s.cfg.RF1ReadPath {
		totalJobs, totalBlocks, totalBlockBytes = s.backendRequests(ctx, tenantID, r, *req, now, samplingRate, targetBytesPerRequest, interval, reqCh)
	} else {
		totalJobs, totalBlocks, totalBlockBytes = s.shardedBackendRequests(ctx, tenantID, r, *req, now, samplingRate, targetBytesPerRequest, interval, reqCh, nil)
	}

	span.SetTag("totalJobs", totalJobs)
	span.SetTag("totalBlocks", totalBlocks)
	span.SetTag("totalBlockBytes", totalBlockBytes)

	// send a job to communicate the search metrics. this is consumed by the combiner to calculate totalblocks/bytes/jobs
	var jobMetricsResponse pipeline.Responses[combiner.PipelineResponse]
	if totalBlocks > 0 {
		resp := &tempopb.QueryRangeResponse{
			Metrics: &tempopb.SearchMetrics{
				TotalJobs:       totalJobs,
				TotalBlocks:     totalBlocks,
				TotalBlockBytes: totalBlockBytes,
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
			m.EndTime.UnixNano() >= start &&
			m.ReplicationFactor == s.replicationFactor {
			metas = append(metas, m)
		}
	}

	return metas
}

func (s *queryRangeSharder) shardedBackendRequests(ctx context.Context, tenantID string, parent *http.Request, searchReq tempopb.QueryRangeRequest, now time.Time, samplingRate float64, targetBytesPerRequest int, interval time.Duration, reqCh chan *http.Request, _ func(error)) (totalJobs, totalBlocks uint32, totalBlockBytes uint64) {
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

	// count blocks
	totalBlocks = uint32(len(blocks))
	for _, b := range blocks {
		totalBlockBytes += b.Size
	}

	// count jobs. same loops as below
	var (
		start          = backendReq.Start
		end            = backendReq.End
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
			totalJobs++
		}

		start = thisEnd
	}

	go func() {
		s.buildShardedBackendRequests(ctx, tenantID, parent, backendReq, samplingRate, targetBytesPerRequest, interval, reqCh)
	}()

	return
}

func (s *queryRangeSharder) buildShardedBackendRequests(ctx context.Context, tenantID string, parent *http.Request, searchReq tempopb.QueryRangeRequest, samplingRate float64, targetBytesPerRequest int, interval time.Duration, reqCh chan *http.Request) {
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
			httpReq := s.toUpstreamRequest(ctx, shardR, parent, tenantID)
			if samplingRate != 1.0 {
				shardR.ShardID *= uint32(1.0 / samplingRate)
				shardR.ShardCount *= uint32(1.0 / samplingRate)

				// Set final sampling rate after integer rounding
				samplingRate = float64(shards) / float64(shardR.ShardCount)

				httpReq = pipeline.ContextAddAdditionalData(samplingRate, httpReq)
			}

			select {
			case reqCh <- httpReq:
			case <-ctx.Done():
				return
			}
		}

		start = thisEnd
	}
}

func (s *queryRangeSharder) backendRequests(ctx context.Context, tenantID string, parent *http.Request, searchReq tempopb.QueryRangeRequest, now time.Time, _ float64, targetBytesPerRequest int, _ time.Duration, reqCh chan *http.Request) (totalJobs, totalBlocks uint32, totalBlockBytes uint64) {
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

	// calculate metrics to return to the caller
	totalBlocks = uint32(len(blocks))
	for _, b := range blocks {
		p := pagesPerRequest(b, targetBytesPerRequest)

		totalJobs += b.TotalRecords / uint32(p)
		if int(b.TotalRecords)%p != 0 {
			totalJobs++
		}
		totalBlockBytes += b.Size
	}

	go func() {
		s.buildBackendRequests(ctx, tenantID, parent, backendReq, blocks, targetBytesPerRequest, reqCh)
	}()

	return
}

func (s *queryRangeSharder) buildBackendRequests(ctx context.Context, tenantID string, parent *http.Request, searchReq tempopb.QueryRangeRequest, metas []*backend.BlockMeta, targetBytesPerRequest int, reqCh chan<- *http.Request) {
	defer close(reqCh)

	for _, m := range metas {
		pages := pagesPerRequest(m, targetBytesPerRequest)
		if pages == 0 {
			continue
		}

		for startPage := 0; startPage < int(m.TotalRecords); startPage += pages {
			subR := parent.Clone(ctx)

			dc, err := m.DedicatedColumns.ToTempopb()
			if err != nil {
				// errFn(fmt.Errorf("failed to convert dedicated columns. block: %s tempopb: %w", blockID, err))
				continue
			}

			start, end := traceql.TrimToOverlap(searchReq.Start, searchReq.End, searchReq.Step, uint64(m.StartTime.UnixNano()), uint64(m.EndTime.UnixNano()))

			queryRangeReq := &tempopb.QueryRangeRequest{
				Query: searchReq.Query,
				Start: start,
				End:   end,
				Step:  searchReq.Step,
				// ShardID:    uint32, // No sharding with RF=1
				// ShardCount: uint32, // No sharding with RF=1
				QueryMode: searchReq.QueryMode,
				// New RF1 fields
				BlockID:          m.BlockID.String(),
				StartPage:        uint32(startPage),
				PagesToSearch:    uint32(pages),
				Version:          m.Version,
				Encoding:         m.Encoding.String(),
				Size_:            m.Size,
				FooterSize:       m.FooterSize,
				DedicatedColumns: dc,
			}

			subR = api.BuildQueryRangeRequest(subR, queryRangeReq)
			subR.Header.Set(api.HeaderAccept, api.HeaderAcceptProtobuf)

			prepareRequestForQueriers(subR, tenantID, subR.URL.Path, subR.URL.Query())
			// TODO: Handle sampling rate

			select {
			case reqCh <- subR:
			case <-ctx.Done():
				return
			}
		}
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

	req := s.toUpstreamRequest(parent.Context(), searchReq, parent, tenantID)
	req.Header.Set(api.HeaderAccept, api.HeaderAcceptProtobuf)

	return req
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
