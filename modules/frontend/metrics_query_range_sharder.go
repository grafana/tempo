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
	"github.com/segmentio/fasthash/fnv1a"
	"go.opentelemetry.io/otel/attribute"

	"github.com/grafana/tempo/modules/frontend/combiner"
	"github.com/grafana/tempo/modules/frontend/pipeline"
	"github.com/grafana/tempo/modules/frontend/shardtracker"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/modules/querier"
	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/traceql"
	"github.com/grafana/tempo/pkg/validation"
	"github.com/grafana/tempo/tempodb"
	"github.com/grafana/tempo/tempodb/backend"
)

const (
	defaultStreamingShards = 200
)

type queryRangeSharder struct {
	next        pipeline.AsyncRoundTripper[combiner.PipelineResponse]
	reader      tempodb.Reader
	overrides   overrides.Interface
	cfg         QueryRangeSharderConfig
	logger      log.Logger
	instantMode bool
}

type QueryRangeSharderConfig struct {
	ConcurrentRequests    int           `yaml:"concurrent_jobs,omitempty"`
	TargetBytesPerRequest int           `yaml:"target_bytes_per_job,omitempty"`
	MaxDuration           time.Duration `yaml:"max_duration"`
	// QueryBackendAfter determines when to query backend storage vs ingesters only.
	QueryBackendAfter time.Duration `yaml:"query_backend_after,omitempty"`
	Interval          time.Duration `yaml:"interval,omitempty"`
	MaxExemplars      int           `yaml:"max_exemplars,omitempty"`
	MaxResponseSeries int           `yaml:"max_response_series,omitempty"`
	StreamingShards   int           `yaml:"streaming_shards,omitempty"`
}

// newAsyncQueryRangeSharder creates a sharding middleware for search
func newAsyncQueryRangeSharder(reader tempodb.Reader, o overrides.Interface, cfg QueryRangeSharderConfig, instantMode bool, logger log.Logger) pipeline.AsyncMiddleware[combiner.PipelineResponse] {
	return pipeline.AsyncMiddlewareFunc[combiner.PipelineResponse](func(next pipeline.AsyncRoundTripper[combiner.PipelineResponse]) pipeline.AsyncRoundTripper[combiner.PipelineResponse] {
		return queryRangeSharder{
			next:        next,
			reader:      reader,
			overrides:   o,
			instantMode: instantMode,
			cfg:         cfg,
			logger:      logger,
		}
	})
}

func (s queryRangeSharder) RoundTrip(pipelineRequest pipeline.Request) (pipeline.Responses[combiner.PipelineResponse], error) {
	r := pipelineRequest.HTTPRequest()
	spanName := "frontend.QueryRangeSharder.range"

	if s.instantMode {
		spanName = "frontend.QueryRangeSharder.instant"
	}

	ctx, span := tracer.Start(r.Context(), spanName)
	defer span.End()

	req, err := api.ParseQueryRangeRequest(r)
	if err != nil {
		return pipeline.NewBadRequest(err), nil
	}

	expr, _, _, _, _, err := traceql.Compile(req.Query)
	if err != nil {
		return pipeline.NewBadRequest(err), nil
	}

	if expr.IsNoop() {
		// Empty response
		ch := make(chan pipeline.Request, 2)
		close(ch)
		return pipeline.NewAsyncSharderChan(ctx, s.cfg.ConcurrentRequests, ch, nil, s.next), nil
	}

	tenantID, err := validation.ExtractValidTenantID(ctx)
	if err != nil {
		return pipeline.NewBadRequest(err), nil
	}

	if req.Step == 0 {
		return pipeline.NewBadRequest(errors.New("step must be greater than 0")), nil
	}

	// calculate and enforce max search duration
	// This is checked before alignment because we may need to read a larger
	// range internally to satisfy the query.
	maxDuration := s.maxDuration(tenantID)
	if maxDuration != 0 && time.Duration(req.End-req.Start)*time.Nanosecond > maxDuration {
		err = fmt.Errorf("metrics query time range exceeds the maximum allowed duration of %s", maxDuration)
		return pipeline.NewBadRequest(err), nil
	}

	traceql.AlignRequest(req)

	var maxExemplars uint32
	// Instant queries must not compute exemplars
	if !s.instantMode && s.cfg.MaxExemplars > 0 {
		maxExemplars = req.Exemplars
		if maxExemplars == 0 || maxExemplars > uint32(s.cfg.MaxExemplars) {
			maxExemplars = uint32(s.cfg.MaxExemplars) // Enforce configuration
		}
	}
	req.Exemplars = maxExemplars

	// if a limit is being enforced, honor the request if it is less than the limit
	// else set it to max limit
	if s.cfg.MaxResponseSeries > 0 && (req.MaxSeries > uint32(s.cfg.MaxResponseSeries) || req.MaxSeries == 0) {
		req.MaxSeries = uint32(s.cfg.MaxResponseSeries)
	}

	var (
		allowUnsafe           = s.overrides.UnsafeQueryHints(tenantID)
		targetBytesPerRequest = s.jobSize(expr, allowUnsafe)
		cutoff                = time.Now().Add(-s.cfg.QueryBackendAfter)
	)

	backendExemplars, generatorExemplars := s.exemplarsCutoff(*req, cutoff)
	req.Exemplars = generatorExemplars
	generatorReq, jobMetadata := s.generatorRequest(tenantID, pipelineRequest, *req, cutoff)
	req.Exemplars = backendExemplars

	reqCh := make(chan pipeline.Request, 2) // buffer of 2 allows us to insert generatorReq and metrics
	if generatorReq != nil {
		reqCh <- generatorReq
	}

	s.backendRequests(ctx, tenantID, pipelineRequest, *req, cutoff, targetBytesPerRequest, reqCh, jobMetadata)

	span.SetAttributes(attribute.Int64("totalJobs", int64(jobMetadata.TotalJobs)))
	span.SetAttributes(attribute.Int64("totalBlocks", int64(jobMetadata.TotalBlocks)))
	span.SetAttributes(attribute.Int64("totalBlockBytes", int64(jobMetadata.TotalBytes)))

	return pipeline.NewAsyncSharderChan(ctx, s.cfg.ConcurrentRequests, reqCh, pipeline.NewAsyncResponse(jobMetadata), s.next), nil
}

// exemplarsCutoff calculates how to distribute exemplars between the generator (for recent data) and
// backend blocks. It returns two values: the number of exemplars for blocks before the cutoff time,
// and the number of exemplars for data after the cutoff time. The distribution is proportional to
// the time range of each segment relative to the total query time range.
func (s *queryRangeSharder) exemplarsCutoff(req tempopb.QueryRangeRequest, cutoff time.Time) (uint32, uint32) {
	timeRange := req.End - req.Start
	limit := req.Exemplars
	traceql.TrimToAfter(&req, cutoff)

	if req.Start >= req.End { // no need to query generator
		return limit, 0 // after - no exemplars needed
	}
	if req.End-req.Start >= timeRange { // no need to query backend
		return 0, limit
	}

	shareAfterCutoff := float64(limit) * float64(req.End-req.Start) / float64(timeRange)
	shareAfterCutoffCeil := uint32(math.Ceil(shareAfterCutoff))
	if limit <= shareAfterCutoffCeil {
		return 0, limit // after - receives all exemplars
	}
	return limit - shareAfterCutoffCeil, shareAfterCutoffCeil
}

func (s *queryRangeSharder) backendRequests(ctx context.Context, tenantID string, parent pipeline.Request, searchReq tempopb.QueryRangeRequest, cutoff time.Time, targetBytesPerRequest int, reqCh chan pipeline.Request, jobMetadata *combiner.QueryRangeJobResponse) {
	// request without start or end, search only in generator
	if searchReq.Start == 0 || searchReq.End == 0 {
		close(reqCh)
		return
	}

	// Make a copy and limit to backend time range.
	// Preserve instant nature of request if needed
	// Don't realign the request, preserve the range for the blocks overlapping the cutoff.
	backendReq := searchReq
	traceql.TrimToBefore(&backendReq, cutoff)

	// If empty window then no need to search backend
	if backendReq.Start == backendReq.End {
		close(reqCh)
		return
	}

	// Blocks within overall time range. This is just for instrumentation, more precise time
	// range is checked for each window.
	start := time.Unix(0, int64(backendReq.Start))
	end := time.Unix(0, int64(backendReq.End))
	blocks := blockMetasForSearch(s.reader.BlockMetas(tenantID), start, end, func(m *backend.BlockMeta) bool {
		return m.ReplicationFactor == backend.MetricsGeneratorReplicationFactor
	})
	if len(blocks) == 0 {
		// no need to search backend
		close(reqCh)
		return
	}

	// calculate metrics to return to the caller
	jobMetadata.TotalBlocks = len(blocks)

	// Calculate total duration across all blocks for exemplar distribution
	var totalDurationNanos int64
	for _, b := range blocks {
		if !b.EndTime.Before(b.StartTime) {
			totalDurationNanos += b.EndTime.UnixNano() - b.StartTime.UnixNano()
		}
	}

	// Create function to calculate exemplars per block
	getExemplarsForBlock := func(m *backend.BlockMeta) uint32 {
		return s.exemplarsForBlock(m, searchReq.Exemplars, totalDurationNanos)
	}

	// Group blocks into shards
	maxShards := s.cfg.StreamingShards
	if maxShards <= 0 {
		maxShards = defaultStreamingShards
	}

	firstShardIdx := len(jobMetadata.Shards)
	blockIter := backendJobsFunc(blocks, targetBytesPerRequest, maxShards, uint32(time.Unix(0, int64(searchReq.End)).Unix()))
	blockIter(func(jobs int, sz uint64, completedThroughTime uint32) {
		jobMetadata.TotalJobs += jobs
		jobMetadata.TotalBytes += sz

		jobMetadata.Shards = append(jobMetadata.Shards, shardtracker.Shard{
			TotalJobs:               uint32(jobs),
			CompletedThroughSeconds: completedThroughTime,
		})
	}, nil)

	go func() {
		s.buildBackendRequests(ctx, tenantID, parent, backendReq, firstShardIdx, blockIter, reqCh, getExemplarsForBlock)
	}()
}

func (s *queryRangeSharder) buildBackendRequests(ctx context.Context, tenantID string, parent pipeline.Request, searchReq tempopb.QueryRangeRequest, firstShardIdx int, blockIter func(shardIterFn, jobIterFn), reqCh chan<- pipeline.Request, getExemplarsForBlock func(*backend.BlockMeta) uint32) {
	defer close(reqCh)

	queryHash := hashForQueryRangeRequest(&searchReq)
	colsToJSON := api.NewDedicatedColumnsToJSON()

	blockIter(nil, func(m *backend.BlockMeta, shard, startPage, pages int) {
		dedColsJSON, err := colsToJSON.JSONForDedicatedColumns(m.DedicatedColumns)
		if err != nil {
			_ = level.Error(s.logger).Log("msg", "failed to convert dedicated columns in query range sharder. skipping", "block", m.BlockID, "err", err)
			return
		}

		// Trim and align the request for this block. I.e. if the request is "Last Hour" we don't want to
		// cache the response for that, we want only the few minutes time range for this block. This has
		// size savings but the main thing is that the response is reuseable for any overlapping query.
		start, end, step := traceql.TrimToBlockOverlap(searchReq.Start, searchReq.End, searchReq.Step, m.StartTime, m.EndTime)
		if start == end || step == 0 {
			level.Warn(s.logger).Log("msg", "invalid start/step end. skipping", "start", start, "end", end, "step", step, "blockStart", m.StartTime.UnixNano(), "blockEnd", m.EndTime.UnixNano())
			return
		}

		// Calculate exemplars for this specific request
		exemplars := getExemplarsForBlock(m)
		if exemplars > 0 {
			// Scale the number of exemplars per block to match the size
			// of each sub request on this block. For very small blocks or other edge cases, return at least 1.
			exemplars = max(uint32(float64(exemplars)*float64(pages)/float64(m.TotalRecords)), 1)
		}

		pipelineR, err := cloneRequestforQueriers(parent, tenantID, func(r *http.Request) (*http.Request, error) {
			queryRangeReq := &tempopb.QueryRangeRequest{
				Query:     searchReq.Query,
				Start:     start,
				End:       end,
				Step:      step,
				QueryMode: searchReq.QueryMode,
				// New RF1 fields
				BlockID:       m.BlockID.String(),
				StartPage:     uint32(startPage),
				PagesToSearch: uint32(pages),
				Version:       m.Version,
				Encoding:      m.Encoding.String(),
				Size_:         m.Size_,
				FooterSize:    m.FooterSize,
				// DedicatedColumns: dc, for perf reason we pass dedicated columns json in directly to not have to realloc object -> proto -> json
				Exemplars: exemplars,
				MaxSeries: searchReq.MaxSeries,
			}

			return api.BuildQueryRangeRequest(r, queryRangeReq, dedColsJSON), nil
		})
		if err != nil {
			_ = level.Error(s.logger).Log("msg", "failed to cloneRequestForQuerirs in the query range sharder. skipping", "block", m.BlockID, "err", err)
			return
		}

		startTime := time.Unix(0, int64(searchReq.Start)) // start/end are in nanoseconds
		endTime := time.Unix(0, int64(searchReq.End))
		// TODO: Handle sampling rate
		key := queryRangeCacheKey(tenantID, queryHash, startTime, endTime, m, startPage, pages)
		if len(key) > 0 {
			pipelineR.SetCacheKey(key)
		}

		// Set which shard this request belongs to
		pipelineR.SetResponseData(shard + firstShardIdx)

		select {
		case reqCh <- pipelineR:
		case <-ctx.Done():
			return
		}
	})
}

func (s *queryRangeSharder) generatorRequest(tenantID string, parent pipeline.Request, searchReq tempopb.QueryRangeRequest, cutoff time.Time) (pipeline.Request, *combiner.QueryRangeJobResponse) {
	jobMetadata := &combiner.QueryRangeJobResponse{}

	// Trim the time range to only the recent which is covered by the generators.
	// Important - don't align the request after trimming. We always need to ensure
	// the start/end time range sent to the generators is accurate.
	traceql.TrimToAfter(&searchReq, cutoff)

	// if start == end then we don't need to query it
	if searchReq.Start == searchReq.End {
		return nil, jobMetadata
	}

	searchReq.QueryMode = querier.QueryModeRecent

	subR, _ := cloneRequestforQueriers(parent, tenantID, func(r *http.Request) (*http.Request, error) {
		return api.BuildQueryRangeRequest(r, &searchReq, ""), nil
	})

	// Add shard metadata for the generator request, similar to ingesterRequests
	// The generator covers the most recent data, so it completes through MaxUint32
	jobMetadata.TotalJobs = 1
	jobMetadata.Shards = append(jobMetadata.Shards, shardtracker.Shard{
		TotalJobs:               1,
		CompletedThroughSeconds: shardtracker.TimestampNever,
	})

	subR.SetResponseData(0) // generator requests are always shard 0

	return subR, jobMetadata
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

func (s *queryRangeSharder) jobSize(expr *traceql.RootExpr, allowUnsafe bool) int {
	// If we have a query hint then use it
	if v, ok := expr.Hints.GetInt(traceql.HintJobSize, allowUnsafe); ok && v > 0 {
		return v
	}

	// Else use configured value.
	size := s.cfg.TargetBytesPerRequest

	return size
}

// exemplarsForBlock calculates exemplars for a single block based on its proportional duration.
// Example: if a block is 90s out of 100s total, with limit=100, it gets 90*1.2=108 exemplars.
func (s *queryRangeSharder) exemplarsForBlock(m *backend.BlockMeta, totalExemplars uint32, totalDurationNanos int64) uint32 {
	const overhead = 1.2 // 20% overhead for shard size

	if totalExemplars == 0 || totalDurationNanos <= 0 {
		return 0
	}

	if m.EndTime.Before(m.StartTime) { // Skip blocks with invalid time ranges
		return 0
	}

	blockDuration := m.EndTime.UnixNano() - m.StartTime.UnixNano()
	share := (float64(blockDuration) / float64(totalDurationNanos)) * float64(totalExemplars) * overhead
	return max(uint32(math.Ceil(share)), 1)
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

	// add the query and other fields that change the response to the hash
	hash := fnv1a.HashString64(query)
	hash = fnv1a.AddUint64(hash, req.Step)
	hash = fnv1a.AddUint64(hash, uint64(req.MaxSeries))
	hash = fnv1a.AddUint64(hash, uint64(req.Exemplars))

	return hash
}
