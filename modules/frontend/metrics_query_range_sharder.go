package frontend

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"time"

	"github.com/go-kit/log" //nolint:all deprecated
	"github.com/go-kit/log/level"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/grafana/dskit/user"
	"github.com/opentracing/opentracing-go"
	"github.com/segmentio/fasthash/fnv1a"

	"github.com/grafana/tempo/v2/modules/frontend/combiner"
	"github.com/grafana/tempo/v2/modules/frontend/pipeline"
	"github.com/grafana/tempo/v2/modules/overrides"
	"github.com/grafana/tempo/v2/modules/querier"
	"github.com/grafana/tempo/v2/pkg/api"
	"github.com/grafana/tempo/v2/pkg/tempopb"
	"github.com/grafana/tempo/v2/pkg/traceql"
	"github.com/grafana/tempo/v2/tempodb"
	"github.com/grafana/tempo/v2/tempodb/backend"
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
	Exemplars             bool          `yaml:"exemplars,omitempty"`
	MaxExemplars          int           `yaml:"max_exemplars,omitempty"`
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

func (s queryRangeSharder) RoundTrip(pipelineRequest pipeline.Request) (pipeline.Responses[combiner.PipelineResponse], error) {
	r := pipelineRequest.HTTPRequest()

	span, ctx := opentracing.StartSpanFromContext(r.Context(), "frontend.QueryRangeSharder")
	defer span.Finish()

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

	traceql.AlignRequest(req)

	// calculate and enforce max search duration
	// Note: this is checked after alignment for consistency.
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
		cutoff                = time.Now().Add(-s.cfg.QueryBackendAfter)
	)

	// if interval is 0 then the backend requests code will loop forever. technically if we are here with a 0 interval it should mean a bad request
	// b/c it was specified using a query hint. we're going to assume that and return 400
	if interval == 0 {
		return pipeline.NewBadRequest(errors.New("invalid interval specified: 0")), nil
	}

	generatorReq := s.generatorRequest(*req, r, tenantID, cutoff)
	reqCh := make(chan pipeline.Request, 2) // buffer of 2 allows us to insert generatorReq and metrics

	if generatorReq != nil {
		reqCh <- pipeline.NewHTTPRequest(generatorReq)
	}

	var (
		totalJobs, totalBlocks uint32
		totalBlockBytes        uint64
	)
	if s.cfg.RF1ReadPath {
		totalJobs, totalBlocks, totalBlockBytes = s.backendRequests(ctx, tenantID, r, *req, cutoff, samplingRate, targetBytesPerRequest, interval, reqCh)
	} else {
		totalJobs, totalBlocks, totalBlockBytes = s.shardedBackendRequests(ctx, tenantID, r, *req, cutoff, samplingRate, targetBytesPerRequest, interval, reqCh, nil)
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

func (s *queryRangeSharder) shardedBackendRequests(ctx context.Context, tenantID string, parent *http.Request, searchReq tempopb.QueryRangeRequest, cutoff time.Time, samplingRate float64, targetBytesPerRequest int, interval time.Duration, reqCh chan pipeline.Request, _ func(error)) (totalJobs, totalBlocks uint32, totalBlockBytes uint64) {
	// request without start or end, search only in generator
	if searchReq.Start == 0 || searchReq.End == 0 {
		close(reqCh)
		return
	}

	// Make a copy and limit to backend time range.
	backendReq := searchReq
	traceql.TrimToBefore(&backendReq, cutoff)

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

func (s *queryRangeSharder) buildShardedBackendRequests(ctx context.Context, tenantID string, parent *http.Request, searchReq tempopb.QueryRangeRequest, samplingRate float64, targetBytesPerRequest int, interval time.Duration, reqCh chan pipeline.Request) {
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
		exemplars := max(s.exemplarsPerShard(shards), 1)

		for i := uint32(1); i <= shards; i++ {

			shardR := searchReq
			shardR.Start = thisStart
			shardR.End = thisEnd
			shardR.ShardID = i
			shardR.ShardCount = shards
			shardR.Exemplars = exemplars
			httpReq := s.toUpstreamRequest(ctx, shardR, parent, tenantID)

			pipelineR := pipeline.NewHTTPRequest(httpReq)
			if samplingRate != 1.0 {
				shardR.ShardID *= uint32(1.0 / samplingRate)
				shardR.ShardCount *= uint32(1.0 / samplingRate)

				// Set final sampling rate after integer rounding
				samplingRate = float64(shards) / float64(shardR.ShardCount)

				pipelineR.SetResponseData(samplingRate)
			}

			select {
			case reqCh <- pipelineR:
			case <-ctx.Done():
				return
			}
		}

		start = thisEnd
	}
}

func (s *queryRangeSharder) exemplarsPerShard(total uint32) uint32 {
	if !s.cfg.Exemplars {
		return 0
	}
	return uint32(math.Ceil(float64(s.cfg.MaxExemplars)*1.2)) / total
}

func (s *queryRangeSharder) backendRequests(ctx context.Context, tenantID string, parent *http.Request, searchReq tempopb.QueryRangeRequest, cutoff time.Time, _ float64, targetBytesPerRequest int, _ time.Duration, reqCh chan pipeline.Request) (totalJobs, totalBlocks uint32, totalBlockBytes uint64) {
	// request without start or end, search only in generator
	if searchReq.Start == 0 || searchReq.End == 0 {
		close(reqCh)
		return
	}

	// Make a copy and limit to backend time range.
	// Preserve instant nature of request if needed
	backendReq := searchReq
	traceql.TrimToBefore(&backendReq, cutoff)

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

func (s *queryRangeSharder) buildBackendRequests(ctx context.Context, tenantID string, parent *http.Request, searchReq tempopb.QueryRangeRequest, metas []*backend.BlockMeta, targetBytesPerRequest int, reqCh chan<- pipeline.Request) {
	defer close(reqCh)

	queryHash := hashForQueryRangeRequest(&searchReq)

	exemplars := s.exemplarsPerShard(uint32(len(metas)))
	for _, m := range metas {
		if m.EndTime.Before(m.StartTime) {
			// Ignore blocks with bad timings from debugging
			continue
		}

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

			// Trim and align the request for this block. I.e. if the request is "Last Hour" we don't want to
			// cache the response for that, we want only the few minutes time range for this block. This has
			// size savings but the main thing is that the response is reuseable for any overlapping query.
			start, end, step := traceql.TrimToOverlap(searchReq.Start, searchReq.End, searchReq.Step, uint64(m.StartTime.UnixNano()), uint64(m.EndTime.UnixNano()))
			if start == end || step == 0 {
				level.Warn(s.logger).Log("msg", "invalid start/step end. skipping", "start", start, "end", end, "step", step, "blockStart", m.StartTime.UnixNano(), "blockEnd", m.EndTime.UnixNano())
				continue
			}

			queryRangeReq := &tempopb.QueryRangeRequest{
				Query: searchReq.Query,
				Start: start,
				End:   end,
				Step:  step,
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
				Exemplars:        max(exemplars/(m.TotalRecords/uint32(pages)), 1),
			}

			subR = api.BuildQueryRangeRequest(subR, queryRangeReq)
			subR.Header.Set(api.HeaderAccept, api.HeaderAcceptProtobuf)

			prepareRequestForQueriers(subR, tenantID)
			pipelineR := pipeline.NewHTTPRequest(subR)

			// TODO: Handle sampling rate
			key := queryRangeCacheKey(tenantID, queryHash, int64(queryRangeReq.Start), int64(queryRangeReq.End), m, int(queryRangeReq.StartPage), int(queryRangeReq.PagesToSearch))
			if len(key) > 0 {
				pipelineR.SetCacheKey(key)
			}

			select {
			case reqCh <- pipelineR:
			case <-ctx.Done():
				return
			}
		}
	}
}

func max(a, b uint32) uint32 {
	if a > b {
		return a
	}
	return b
}

func (s *queryRangeSharder) generatorRequest(searchReq tempopb.QueryRangeRequest, parent *http.Request, tenantID string, cutoff time.Time) *http.Request {
	traceql.TrimToAfter(&searchReq, cutoff)

	// if start == end then we don't need to query it
	if searchReq.Start == searchReq.End {
		return nil
	}

	searchReq.QueryMode = querier.QueryModeRecent
	searchReq.Exemplars = uint32(s.cfg.MaxExemplars) // TODO: Review this

	req := s.toUpstreamRequest(parent.Context(), searchReq, parent, tenantID)
	req.Header.Set(api.HeaderAccept, api.HeaderAcceptProtobuf)

	return req
}

func (s *queryRangeSharder) toUpstreamRequest(ctx context.Context, req tempopb.QueryRangeRequest, parent *http.Request, tenantID string) *http.Request {
	subR := parent.Clone(ctx)
	subR = api.BuildQueryRangeRequest(subR, &req)

	prepareRequestForQueriers(subR, tenantID)
	return subR
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

func hashForQueryRangeRequest(req *tempopb.QueryRangeRequest) uint64 {
	if req.Query == "" {
		return 0
	}

	ast, err := traceql.Parse(req.Query)
	if err != nil { // this should never occur. if we've made this far we've already validated the query can parse. however, for sanity, just fail to cache if we can't parse
		return 0
	}

	// forces the query into a canonical form
	query := ast.String()

	// add the query, limit and spss to the hash
	hash := fnv1a.HashString64(query)
	hash = fnv1a.AddUint64(hash, req.Step)

	return hash
}
