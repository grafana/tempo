package frontend

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-kit/log" //nolint:all deprecated
	"github.com/go-kit/log/level"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
	"github.com/segmentio/fasthash/fnv1a"

	"github.com/grafana/tempo/modules/frontend/combiner"
	"github.com/grafana/tempo/modules/frontend/pipeline"
	"github.com/grafana/tempo/modules/frontend/shardtracker"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/traceql"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/pkg/validation"
	"github.com/grafana/tempo/tempodb"
	"github.com/grafana/tempo/tempodb/backend"
)

const (
	defaultTargetBytesPerRequest = 100 * 1024 * 1024
	defaultConcurrentRequests    = 1000
	defaultMostRecentShards      = 200

	defaultMaxSpanIDs = 1000
	defaultMaxTraces  = 1000
)

type SearchSharderConfig struct {
	ConcurrentRequests    int           `yaml:"concurrent_jobs,omitempty"`
	TargetBytesPerRequest int           `yaml:"target_bytes_per_job,omitempty"`
	DefaultLimit          uint32        `yaml:"default_result_limit"`
	MaxLimit              uint32        `yaml:"max_result_limit"`
	MaxDuration           time.Duration `yaml:"max_duration"`
	// QueryBackendAfter determines when to query backend storage vs ingesters only.
	QueryBackendAfter      time.Duration `yaml:"query_backend_after,omitempty"`
	QueryIngestersUntil    time.Duration `yaml:"query_ingesters_until,omitempty"`
	IngesterShards         int           `yaml:"ingester_shards,omitempty"`
	MostRecentShards       int           `yaml:"most_recent_shards,omitempty"`
	DefaultSpansPerSpanSet uint32        `yaml:"default_spans_per_span_set,omitempty"`
	MaxSpansPerSpanSet     uint32        `yaml:"max_spans_per_span_set,omitempty"`

	// RF1After specifies the time after which RF1 logic is applied, injected by the configuration
	// or determined at runtime based on search request parameters.
	RF1After time.Time `yaml:"-"`
}

type asyncSearchSharder struct {
	next      pipeline.AsyncRoundTripper[combiner.PipelineResponse]
	reader    tempodb.Reader
	overrides overrides.Interface

	cfg    SearchSharderConfig
	logger log.Logger
}

// newAsyncSearchSharder creates a sharding middleware for search
func newAsyncSearchSharder(reader tempodb.Reader, o overrides.Interface, cfg SearchSharderConfig, logger log.Logger) pipeline.AsyncMiddleware[combiner.PipelineResponse] {
	return pipeline.AsyncMiddlewareFunc[combiner.PipelineResponse](func(next pipeline.AsyncRoundTripper[combiner.PipelineResponse]) pipeline.AsyncRoundTripper[combiner.PipelineResponse] {
		return &asyncSearchSharder{
			next:      next,
			reader:    reader,
			overrides: o,

			cfg:    cfg,
			logger: logger,
		}
	})
}

// RoundTrip implements http.RoundTripper
// execute up to concurrentRequests simultaneously where each request scans ~targetMBsPerRequest
// until limit results are found
func (s *asyncSearchSharder) RoundTrip(pipelineRequest pipeline.Request) (pipeline.Responses[combiner.PipelineResponse], error) {
	r := pipelineRequest.HTTPRequest()

	// Use configured default (defaults to 3 if not set in config)
	// If default_spans_per_span_set=0 is explicitly configured, it means unlimited (return all matching spans)
	searchReq, err := api.ParseSearchRequestWithDefault(r, s.cfg.DefaultSpansPerSpanSet)
	if err != nil {
		return pipeline.NewBadRequest(err), nil
	}

	// adjust limit based on config
	searchReq.Limit, err = adjustLimit(searchReq.Limit, s.cfg.DefaultLimit, s.cfg.MaxLimit)
	if err != nil {
		return pipeline.NewBadRequest(err), nil
	}

	requestCtx := r.Context()
	tenantID, err := validation.ExtractValidTenantID(requestCtx)
	if err != nil {
		return pipeline.NewBadRequest(err), nil
	}
	ctx, span := tracer.Start(requestCtx, "frontend.ShardSearch")
	defer span.End()

	// calculate and enforce max search duration
	maxDuration := s.maxDuration(tenantID)
	if maxDuration != 0 && time.Duration(searchReq.End-searchReq.Start)*time.Second > maxDuration {
		return pipeline.NewBadRequest(fmt.Errorf("range specified by start and end exceeds %s. received start=%d end=%d", maxDuration, searchReq.Start, searchReq.End)), nil
	}

	// Validate SpansPerSpanSet against MaxSpansPerSpanSet
	// If MaxSpansPerSpanSet is 0, it means unlimited spans are allowed
	// If MaxSpansPerSpanSet is non-zero, enforce the limit
	if s.cfg.MaxSpansPerSpanSet != 0 && searchReq.SpansPerSpanSet > s.cfg.MaxSpansPerSpanSet {
		return pipeline.NewBadRequest(fmt.Errorf("spans per span set exceeds %d. received %d", s.cfg.MaxSpansPerSpanSet, searchReq.SpansPerSpanSet)), nil
	}

	// Check if query has cross-trace link traversal
	rootExpr, err := traceql.Parse(searchReq.Query)
	level.Debug(s.logger).Log("msg", "parsed query", "query", searchReq.Query, "parseErr", err, "hasLinkTraversal", err == nil && rootExpr != nil && rootExpr.HasLinkTraversal())
	if err == nil && rootExpr.HasLinkTraversal() {
		maxSpanIDsPerQuery := s.overrides.MaxSpanIDsPerLinkQuery(tenantID)
		if maxSpanIDsPerQuery <= 0 {
			// If not configured, use the query limit as the max span IDs per phase
			maxSpanIDsPerQuery = int(searchReq.Limit)
		}
		level.Debug(s.logger).Log("msg", "using cross-trace link traversal", "maxSpanIDsPerQuery", maxSpanIDsPerQuery, "query", searchReq.Query)
		// Use cross-trace execution path (BACKEND BLOCKS ONLY)
		return s.crossTraceRoundTrip(ctx, pipelineRequest, searchReq, rootExpr, maxSpanIDsPerQuery, tenantID)
	}

	// build and execute requests with a small buffer
	reqCh, jobMetrics, err := s.buildSearchRequestChannel(ctx, tenantID, pipelineRequest, searchReq, 1, func(err error) {
		// TODO: actually find a way to return this error to the user
		s.logger.Log("msg", "search: failed to build backend requests", "err", err)
	})
	if err != nil {
		return nil, err
	}

	// execute requests
	return pipeline.NewAsyncSharderChan(ctx, s.cfg.ConcurrentRequests, reqCh, pipeline.NewAsyncResponse(jobMetrics), s.next), nil
}

// backendRequest builds backend requests to search backend blocks. backendRequest takes ownership of reqCh and closes it.
// it returns 3 int values: totalBlocks, totalBlockBytes, and estimated jobs
func (s *asyncSearchSharder) backendRequests(ctx context.Context, tenantID string, parent pipeline.Request, searchReq *tempopb.SearchRequest, resp *combiner.SearchJobResponse, reqCh chan<- pipeline.Request, errFn func(error)) {
	// request without start or end, search only in ingester
	if searchReq.Start == 0 || searchReq.End == 0 {
		close(reqCh)
		return
	}

	// calculate duration (start and end) to search the backend blocks
	start, end := backendRange(searchReq.Start, searchReq.End, s.cfg.QueryBackendAfter)

	// no need to search backend
	if start == end {
		close(reqCh)
		return
	}

	startT := time.Unix(int64(start), 0)
	endT := time.Unix(int64(end), 0)

	// Use RF1After from the request if it's not zero, otherwise use the config value
	rf1After := searchReq.RF1After
	if rf1After.IsZero() {
		rf1After = s.cfg.RF1After
	}

	blocks := blockMetasForSearch(s.reader.BlockMetas(tenantID), startT, endT, rf1FilterFn(rf1After))

	// calculate metrics to return to the caller
	resp.TotalBlocks = len(blocks)

	firstShardIdx := len(resp.Shards)
	blockIter := backendJobsFunc(blocks, s.cfg.TargetBytesPerRequest, s.cfg.MostRecentShards, searchReq.End)
	blockIter(func(jobs int, sz uint64, completedThroughTime uint32) {
		resp.TotalJobs += jobs
		resp.TotalBytes += sz

		resp.Shards = append(resp.Shards, shardtracker.Shard{
			TotalJobs:               uint32(jobs),
			CompletedThroughSeconds: completedThroughTime,
		})
	}, nil)

	go func() {
		buildBackendRequests(ctx, tenantID, parent, searchReq, firstShardIdx, blockIter, reqCh, errFn)
	}()
}

// ingesterRequest returns a new start and end time range for the backend as well as an http request
// that covers the ingesters. If nil is returned for the http.Request then there is no ingesters query.
// since this function modifies searchReq.Start and End we are taking a value instead of a pointer to prevent it from
// unexpectedly changing the passed searchReq.
func (s *asyncSearchSharder) ingesterRequests(tenantID string, parent pipeline.Request, searchReq tempopb.SearchRequest, reqCh chan pipeline.Request) (*combiner.SearchJobResponse, error) {
	resp := &combiner.SearchJobResponse{}
	resp.Shards = make([]shardtracker.Shard, 0, s.cfg.MostRecentShards+1) // +1 for the ingester shard

	// request without start or end, search only in ingester
	if searchReq.Start == 0 || searchReq.End == 0 {
		// one shard that covers all time
		resp.TotalJobs = 1
		resp.Shards = append(resp.Shards, shardtracker.Shard{
			TotalJobs:               1,
			CompletedThroughSeconds: 1,
		})

		return resp, buildIngesterRequest(tenantID, parent, &searchReq, reqCh)
	}

	ingesterUntil := uint32(time.Now().Add(-s.cfg.QueryIngestersUntil).Unix())

	// if there's no overlap between the query and ingester range just return nil
	if searchReq.End < ingesterUntil {
		return resp, nil
	}

	ingesterStart := searchReq.Start
	ingesterEnd := searchReq.End

	// adjust ingesterStart if necessary
	if ingesterStart < ingesterUntil {
		ingesterStart = ingesterUntil
	}

	// if ingester start == ingester end then we don't need to query it
	if ingesterStart == ingesterEnd {
		return resp, nil
	}

	searchReq.Start = ingesterStart
	searchReq.End = ingesterEnd

	// Split the start and end range into sub requests for each range.
	duration := searchReq.End - searchReq.Start
	interval := duration / uint32(s.cfg.IngesterShards)
	intervalMinimum := uint32(60)

	if interval < intervalMinimum {
		interval = intervalMinimum
	}

	for i := 0; i < s.cfg.IngesterShards; i++ {
		var (
			subReq     = searchReq
			shardStart = ingesterStart + uint32(i)*interval
			shardEnd   = shardStart + interval
		)

		// stop if we've gone past the end of the range
		if shardStart >= ingesterEnd {
			break
		}

		// snap shardEnd to the end of the query range
		if shardEnd >= ingesterEnd || i == s.cfg.IngesterShards-1 {
			shardEnd = ingesterEnd
		}

		subReq.Start = shardStart
		subReq.End = shardEnd

		err := buildIngesterRequest(tenantID, parent, &subReq, reqCh)
		if err != nil {
			return nil, err
		}
	}

	// add one shard that covers no time at all. this will force the combiner to wait
	//  for ingester requests to complete before moving on to the backend requests
	ingesterJobs := len(reqCh)
	resp.TotalJobs = ingesterJobs
	resp.Shards = append(resp.Shards, shardtracker.Shard{
		TotalJobs:               uint32(ingesterJobs),
		CompletedThroughSeconds: shardtracker.TimestampNever,
	})

	return resp, nil
}

func (s *asyncSearchSharder) buildSearchRequestChannel(ctx context.Context, tenantID string, parent pipeline.Request, searchReq *tempopb.SearchRequest, bufferExtra int, errFn func(error)) (chan pipeline.Request, *combiner.SearchJobResponse, error) {
	if bufferExtra < 0 {
		bufferExtra = 0
	}

	reqCh := make(chan pipeline.Request, s.cfg.IngesterShards+bufferExtra)

	jobMetrics, err := s.ingesterRequests(tenantID, parent, *searchReq, reqCh)
	if err != nil {
		close(reqCh)
		return nil, nil, err
	}

	s.backendRequests(ctx, tenantID, parent, searchReq, jobMetrics, reqCh, errFn)
	return reqCh, jobMetrics, nil
}

// maxDuration returns the max search duration allowed for this tenant.
func (s *asyncSearchSharder) maxDuration(tenantID string) time.Duration {
	// check overrides first, if no overrides then grab from our config
	maxDuration := s.overrides.MaxSearchDuration(tenantID)
	if maxDuration != 0 {
		return maxDuration
	}

	return s.cfg.MaxDuration
}

// backendRange returns a new start/end range for the backend based on the config parameter
// query_backend_after. If the returned start == the returned end then backend querying is not necessary.
func backendRange(start, end uint32, queryBackendAfter time.Duration) (uint32, uint32) {
	now := time.Now()
	backendAfter := uint32(now.Add(-queryBackendAfter).Unix())

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

// buildBackendRequests returns a slice of requests that cover all blocks in the store
// that are covered by start/end.
func buildBackendRequests(ctx context.Context, tenantID string, parent pipeline.Request, searchReq *tempopb.SearchRequest, firstShardIdx int, blockIter func(shardIterFn, jobIterFn), reqCh chan<- pipeline.Request, errFn func(error)) {
	defer close(reqCh)

	queryHash := hashForSearchRequest(searchReq)
	colsToJSON := api.NewDedicatedColumnsToJSON()

	blockIter(nil, func(m *backend.BlockMeta, shard, startPage, pages int) {
		blockID := m.BlockID.String()

		dedColsJSON, err := colsToJSON.JSONForDedicatedColumns(m.DedicatedColumns)
		if err != nil {
			errFn(fmt.Errorf("failed to convert dedicated columns. block: %s tempopb: %w", blockID, err))
			return
		}

		pipelineR, err := cloneRequestforQueriers(parent, tenantID, func(r *http.Request) (*http.Request, error) {
			r, err = api.BuildSearchBlockRequest(r, &tempopb.SearchBlockRequest{
				BlockID:       blockID,
				StartPage:     uint32(startPage),
				PagesToSearch: uint32(pages),
				Encoding:      m.Encoding.String(),
				IndexPageSize: m.IndexPageSize,
				TotalRecords:  m.TotalRecords,
				DataEncoding:  m.DataEncoding,
				Version:       m.Version,
				Size_:         m.Size_,
				FooterSize:    m.FooterSize,
				// DedicatedColumns: dc, for perf reason we pass dedicated columns json in directly to not have to realloc object -> proto -> json
			}, dedColsJSON)

			return r, err
		})
		if err != nil {
			errFn(fmt.Errorf("failed to build search block request. block: %s tempopb: %w", blockID, err))
			return
		}

		startTime := time.Unix(int64(searchReq.Start), 0)
		endTime := time.Unix(int64(searchReq.End), 0)
		key := searchJobCacheKey(tenantID, queryHash, startTime, endTime, m, startPage, pages)
		pipelineR.SetCacheKey(key)
		pipelineR.SetResponseData(firstShardIdx + shard)

		select {
		case reqCh <- pipelineR:
		case <-ctx.Done():
			// ignore the error if there is one. it will be handled elsewhere
			return
		}
	})
}

// hashForSearchRequest returns a uint64 hash of the query. if the query is invalid it returns a 0 hash.
// before hashing the query is forced into a canonical form so equivalent queries will hash to the same value.
func hashForSearchRequest(searchRequest *tempopb.SearchRequest) uint64 {
	if searchRequest.Query == "" {
		return 0
	}

	ast, err := traceql.Parse(searchRequest.Query)
	if err != nil { // this should never occur. if we've made this far we've already validated the query can parse. however, for sanity, just fail to cache if we can't parse
		return 0
	}

	// forces the query into a canonical form
	query := ast.String()

	// add the query, limit and spss to the hash
	hash := fnv1a.HashString64(query)
	hash = fnv1a.AddUint64(hash, uint64(searchRequest.Limit))
	hash = fnv1a.AddUint64(hash, uint64(searchRequest.SpansPerSpanSet))

	return hash
}

// pagesPerRequest returns an integer value that indicates the number of pages
// that should be searched per query. This value is based on the target number of bytes
// 0 is returned if there is no valid answer
func pagesPerRequest(m *backend.BlockMeta, bytesPerRequest int) int {
	if m.Size_ == 0 || m.TotalRecords == 0 {
		return 0
	}
	// if the block is smaller than the bytesPerRequest, we can search the entire block
	if m.Size_ < uint64(bytesPerRequest) {
		return int(m.TotalRecords)
	}

	bytesPerPage := m.Size_ / uint64(m.TotalRecords)
	if bytesPerPage == 0 {
		return 0
	}

	pagesPerQuery := bytesPerRequest / int(bytesPerPage)
	if pagesPerQuery == 0 {
		pagesPerQuery = 1 // have to have at least 1 page per query
	}

	return pagesPerQuery
}

func buildIngesterRequest(tenantID string, parent pipeline.Request, searchReq *tempopb.SearchRequest, reqCh chan pipeline.Request) error {
	subR, err := cloneRequestforQueriers(parent, tenantID, func(r *http.Request) (*http.Request, error) {
		return api.BuildSearchRequest(r, searchReq)
	})
	if err != nil {
		return err
	}

	subR.SetResponseData(0) // ingester requests are always shard 0
	reqCh <- subR

	return nil
}

type (
	shardIterFn func(jobs int, sz uint64, completedThroughTime uint32)
	jobIterFn   func(m *backend.BlockMeta, shard, startPage, pages int)
)

// backendJobsFunc provides an iter func with 2 callbacks designed to be used once to calculate job and shard metrics and a second time
// to generate actual jobs.
func backendJobsFunc(blocks []*backend.BlockMeta, targetBytesPerRequest int, maxShards int, end uint32) func(shardIterFn, jobIterFn) {
	blocksPerShard := len(blocks) / maxShards

	// if we have fewer blocks than shards then every shard is one block
	if blocksPerShard == 0 {
		blocksPerShard = 1
	}

	return func(shardIterCallback shardIterFn, jobIterCallback jobIterFn) {
		currentShard := 0
		jobsInShard := 0
		bytesInShard := uint64(0)
		blocksInShard := 0

		for _, b := range blocks {
			pages := pagesPerRequest(b, targetBytesPerRequest)
			jobsInBlock := 0

			if pages == 0 {
				continue
			}

			// if jobIterCallBack is nil we can skip the loop and directly calc the jobsInBlock
			if jobIterCallback == nil {
				jobsInBlock = int(b.TotalRecords) / pages
				if int(b.TotalRecords)%pages != 0 {
					jobsInBlock++
				}
			} else {
				for startPage := 0; startPage < int(b.TotalRecords); startPage += pages {
					jobIterCallback(b, currentShard, startPage, pages)
					jobsInBlock++
				}
			}

			// do we need to roll to a new shard?
			jobsInShard += jobsInBlock
			bytesInShard += b.Size_
			blocksInShard++

			// -1 b/c we will likely add a final shard below
			//  end comparison b/c there's no point in ending a shard that can't release any results
			if blocksInShard >= blocksPerShard && currentShard < maxShards-1 && b.EndTime.Unix() < int64(end) {
				if shardIterCallback != nil {
					shardIterCallback(jobsInShard, bytesInShard, uint32(b.EndTime.Unix()))
				}
				currentShard++

				jobsInShard = 0
				bytesInShard = 0
				blocksInShard = 0
			}
		}

		// final shard - note that we are overpacking the final shard due to the integer math as well as the limit of 200 shards total. if the search
		//  this is the least impactful shard to place extra jobs in as it is searched last. if we make it here the chances of this being an exhaustive search
		//  are higher
		if shardIterCallback != nil && jobsInShard > 0 {
			shardIterCallback(jobsInShard, bytesInShard, 1) // final shard can cover all time. we don't need to be precise
		}
	}
}

// crossTraceRoundTrip handles queries with link traversal operators
// This implementation executes multi-phase queries by:
// 1. Starting with the terminal query (no link filter)
// 2. Extracting span IDs from results
// 3. Using those span IDs to query the next phase with link:spanID filters
// 4. Repeating until all phases are complete
// 5. Streaming all results back to the caller
func (s *asyncSearchSharder) crossTraceRoundTrip(
	ctx context.Context,
	pipelineRequest pipeline.Request,
	searchReq *tempopb.SearchRequest,
	rootExpr *traceql.RootExpr,
	maxSpanIDs int,
	tenantID string,
) (pipeline.Responses[combiner.PipelineResponse], error) {
	if err := ctx.Err(); err != nil {
		return pipeline.NewAsyncResponse(nil), err
	}

	// Extract link chain in execution order (terminal first)
	linkChain := rootExpr.ExtractLinkChain()

	if len(linkChain) == 0 {
		return pipeline.NewBadRequest(fmt.Errorf("no link operations found in query")), nil
	}

	level.Info(s.logger).Log(
		"msg", "cross-trace link traversal detected",
		"phases", len(linkChain),
		"maxSpanIDsPerPhase", maxSpanIDs,
	)

	// Create a copy of searchReq for phase execution
	// We use maxSpanIDs as the limit for intermediate phases to prevent resource exhaustion
	// while still allowing enough results to build the link chain.
	phaseReq := cloneSearchRequest(searchReq)
	phaseReq.Limit = uint32(maxSpanIDs)

	// Execute all phases sequentially and return combined results
	// We need to execute phases sequentially since each phase depends on the previous one's results

	// Start with the terminal phase (first in execution order)
	terminal := linkChain[0]
	terminalQuery := buildQueryFromExpression(terminal.Conditions)

	level.Debug(s.logger).Log(
		"msg", "executing terminal phase",
		"phase", 0,
		"query", terminalQuery,
	)

	// Execute terminal phase and collect span IDs
	terminalResp, terminalTraces, spanIDs, totalMetrics, err := s.executePhaseAndExtractSpanIDs(ctx, pipelineRequest, terminalQuery, phaseReq, tenantID, maxSpanIDs, searchReq.Limit, len(linkChain) == 1)
	if err != nil {
		return pipeline.NewBadRequest(fmt.Errorf("terminal phase failed: %w", err)), nil
	}

	level.Info(s.logger).Log(
		"msg", "terminal phase complete",
		"spanIDsFound", len(spanIDs),
		"tracesFound", len(terminalTraces),
	)

	// If there's only one phase, return terminal results
	if len(linkChain) == 1 {
		return terminalResp, nil
	}

	// If no span IDs found in terminal phase, return empty
	if len(spanIDs) == 0 {
		return pipeline.NewAsyncResponse(nil), nil
	}

	// Execute all remaining phases to validate complete chain exists
	// We need to execute ALL phases before returning results to ensure
	// we only return traces that are part of COMPLETE chains
	allTracesByPhase := [][]*tempopb.TraceSearchMetadata{terminalTraces}
	allSpanIDsByPhase := [][]string{spanIDs} // Track span IDs from each phase

	for i := 1; i < len(linkChain); i++ {
		if err := ctx.Err(); err != nil {
			return pipeline.NewAsyncResponse(nil), err
		}

		phase := linkChain[i]

		// Build query with link:spanID filter
		phaseQuery, ok := buildLinkFilterQuery(phase.Conditions, spanIDs, maxSpanIDs)
		if !ok {
			level.Warn(s.logger).Log("msg", "no valid span IDs for phase", "phase", i, "inputSpanIDs", len(spanIDs))
			return pipeline.NewAsyncResponse(nil), nil
		}

		level.Debug(s.logger).Log(
			"msg", "executing link phase",
			"phase", i,
			"query", phaseQuery,
			"inputSpanIDs", len(spanIDs),
		)

		// Execute this phase and collect both results and span IDs
		_, phaseTraces, nextSpanIDs, phaseMetrics, err := s.executePhaseAndExtractSpanIDs(ctx, pipelineRequest, phaseQuery, phaseReq, tenantID, maxSpanIDs, searchReq.Limit, false)
		if err != nil {
			return pipeline.NewBadRequest(fmt.Errorf("phase %d failed: %w", i, err)), nil
		}

		// Aggregate metrics
		mergeSearchMetrics(totalMetrics, phaseMetrics)

		level.Info(s.logger).Log(
			"msg", "link phase complete",
			"phase", i,
			"spanIDsFound", len(nextSpanIDs),
			"tracesFound", len(phaseTraces),
		)

		// If any phase returns no results, the chain is incomplete
		// Return empty results (no partial chains)
		if len(nextSpanIDs) == 0 {
			level.Info(s.logger).Log("msg", "incomplete link chain, returning empty results", "phase", i)
			return pipeline.NewAsyncResponse(nil), nil
		}

		// Collect this phase's results and span IDs
		allTracesByPhase = append(allTracesByPhase, phaseTraces)
		allSpanIDsByPhase = append(allSpanIDsByPhase, nextSpanIDs)
		spanIDs = nextSpanIDs
	}

	// All phases completed successfully with results
	// Filter for complete chains and return
	level.Info(s.logger).Log("msg", "complete link chain found, filtering for complete chains", "totalPhases", len(allTracesByPhase))
	return s.filterCompleteChains(ctx, allTracesByPhase, allSpanIDsByPhase, linkChain, searchReq.Limit, totalMetrics), nil
}

// filterCompleteChains ensures only traces with complete link chains are returned
// It validates that for each trace, all linked traces exist in the result set
func (s *asyncSearchSharder) filterCompleteChains(
	ctx context.Context,
	tracesByPhase [][]*tempopb.TraceSearchMetadata,
	spanIDsByPhase [][]string,
	linkChain []*traceql.LinkOperationInfo,
	limit uint32,
	metrics *tempopb.SearchMetrics,
) pipeline.Responses[combiner.PipelineResponse] {
	if err := ctx.Err(); err != nil {
		return pipeline.NewAsyncResponse(nil)
	}

	// Indexed by [phaseIdx][traceID]
	allTraceInfosByPhase := make([]map[string]*traceInfo, len(tracesByPhase))
	// Global mapping from span ID to traceInfo for quick lookup
	spanIDToTraceInfo := make(map[string]*traceInfo)

	for phaseIdx, phaseTraces := range tracesByPhase {
		allTraceInfosByPhase[phaseIdx] = make(map[string]*traceInfo)
		for _, trace := range phaseTraces {
			info := &traceInfo{
				trace:       trace,
				spanIDs:     make(map[string]struct{}),
				linkTargets: make(map[string]struct{}),
			}
			allTraceInfosByPhase[phaseIdx][trace.TraceID] = info

			forEachSpanInTrace(trace, func(span *tempopb.Span) {
				if span.SpanID != "" {
					info.spanIDs[span.SpanID] = struct{}{}
					spanIDToTraceInfo[span.SpanID] = info
				}
				for _, attr := range span.Attributes {
					if attr.Key == "link:spanID" && attr.Value != nil {
						targetID := attr.Value.GetStringValue()
						if targetID != "" {
							info.linkTargets[targetID] = struct{}{}
						}
					}
				}
			})
		}
	}

	// Convert spanIDsByPhase to sets for quick lookup
	validSpanIDsByPhase := make([]map[string]struct{}, len(spanIDsByPhase))
	for i, ids := range spanIDsByPhase {
		validSpanIDsByPhase[i] = make(map[string]struct{})
		for _, id := range ids {
			validSpanIDsByPhase[i][id] = struct{}{}
		}
	}

	// Build a reverse mapping: which traces (and in which phase) link to which span ID
	// This allows us to quickly find if "something from the next phase links to us"
	linksToSpanID := make(map[string][]int) // targetSpanID -> []phaseIdx
	for phaseIdx, infos := range allTraceInfosByPhase {
		for _, info := range infos {
			for targetID := range info.linkTargets {
				linksToSpanID[targetID] = append(linksToSpanID[targetID], phaseIdx)
			}
		}
	}

	// Now filter traces to only include complete chains
	// Identify all (traceID, phaseIdx) pairs that are part of a complete chain
	type phaseKey struct {
		traceID string
		phase   int
	}
	completePhaseTraces := make(map[phaseKey]*tempopb.TraceSearchMetadata)
	completeTraceIDs := make(map[string]struct{})

	for phaseIdx, infos := range allTraceInfosByPhase {
		for traceID, info := range infos {
			if s.isCompleteChain(info, phaseIdx, validSpanIDsByPhase, linksToSpanID) {
				key := phaseKey{traceID, phaseIdx}
				completePhaseTraces[key] = info.trace
				completeTraceIDs[traceID] = struct{}{}
			}
		}
	}

	// Determine which phases are included in the final result based on union operators
	includedPhases := s.getIncludedPhases(linkChain)

	// Merge spans for complete traces from included phases
	mergedTraces := make(map[string]*tempopb.TraceSearchMetadata)
	for traceID := range completeTraceIDs {
		var mergedTrace *tempopb.TraceSearchMetadata

		for phaseIdx := 0; phaseIdx < len(tracesByPhase); phaseIdx++ {
			if _, isIncluded := includedPhases[phaseIdx]; !isIncluded {
				continue
			}

			key := phaseKey{traceID, phaseIdx}
			if trace, ok := completePhaseTraces[key]; ok {
				if mergedTrace == nil {
					// Initialize merged trace with metadata from this trace
					mergedTrace = &tempopb.TraceSearchMetadata{
						TraceID:           trace.TraceID,
						RootServiceName:   trace.RootServiceName,
						RootTraceName:     trace.RootTraceName,
						StartTimeUnixNano: trace.StartTimeUnixNano,
						DurationMs:        trace.DurationMs,
					}
				}

				// Merge spans from this phase version of the trace
				mergedTrace.SpanSets = append(mergedTrace.SpanSets, trace.SpanSets...)
				if trace.SpanSet != nil {
					if mergedTrace.SpanSet == nil {
						mergedTrace.SpanSet = &tempopb.SpanSet{}
					}
					mergedTrace.SpanSet.Spans = append(mergedTrace.SpanSet.Spans, trace.SpanSet.Spans...)
				}
			}
		}

		if mergedTrace != nil {
			mergedTraces[traceID] = mergedTrace
		}
	}

	completeTraces := make([]*tempopb.TraceSearchMetadata, 0, len(mergedTraces))
	for _, mergedTrace := range mergedTraces {
		// Deduplicate spans within each trace
		s.deduplicateSpansInTrace(mergedTrace)
		completeTraces = append(completeTraces, mergedTrace)
	}

	limitedTraces, truncated := applyTraceLimit(completeTraces, limit)
	level.Info(s.logger).Log(
		"msg", "filtered for complete chains",
		"totalTracesCollected", len(spanIDToTraceInfo),
		"completeChainTraces", len(limitedTraces),
		"truncated", truncated,
	)

	// Convert filtered traces back to responses
	return s.tracesToResponses(ctx, limitedTraces, metrics)
}

// traceInfo stores pre-extracted metadata for efficient chain validation
type traceInfo struct {
	trace       *tempopb.TraceSearchMetadata
	spanIDs     map[string]struct{}
	linkTargets map[string]struct{}
}

// isCompleteChain checks if a trace is part of a complete link chain using pre-calculated info
func (s *asyncSearchSharder) isCompleteChain(
	info *traceInfo,
	tracePhase int,
	validSpanIDsByPhase []map[string]struct{},
	linksToSpanID map[string][]int,
) bool {
	// Check link to previous phase (if not terminal)
	if tracePhase > 0 {
		foundLinkToPrev := false
		for targetID := range info.linkTargets {
			if _, ok := validSpanIDsByPhase[tracePhase-1][targetID]; ok {
				foundLinkToPrev = true
				break
			}
		}

		if !foundLinkToPrev {
			return false
		}
	}

	// Check if something from the next phase links to us (if not last)
	if tracePhase < len(validSpanIDsByPhase)-1 {
		foundLinkFromNext := false
		for mySpanID := range info.spanIDs {
			// Check if any trace in the next phase links to this span ID
			for _, phaseIdx := range linksToSpanID[mySpanID] {
				if phaseIdx == tracePhase+1 {
					foundLinkFromNext = true
					break
				}
			}
			if foundLinkFromNext {
				break
			}
		}

		if !foundLinkFromNext {
			return false
		}
	}

	return true
}

// getIncludedPhases determines which phases should be included in the final output
// based on the presence of union operators (&->>, &<<-) in the chain.
func (s *asyncSearchSharder) getIncludedPhases(linkChain []*traceql.LinkOperationInfo) map[int]struct{} {
	included := make(map[int]struct{})
	included[0] = struct{}{} // Terminal phase (Phase 0) is always included

	if len(linkChain) < 2 {
		return included
	}

	// Determine if the chain was reversed (LinkTo ->>)
	isReversed := linkChain[0].IsLinkTo

	// In a chain of N phases, there are N-1 links.
	// For ->> (reversed):
	// Link 0 (Phase 0-1) is at linkChain[0]
	// Link 1 (Phase 1-2) is at linkChain[1]
	// ...
	// For <<- (not reversed):
	// Link 0 (Phase 0-1) is at linkChain[1]
	// Link 1 (Phase 1-2) is at linkChain[2]
	// ...

	for i := 1; i < len(linkChain); i++ {
		isUnion := false
		if isReversed {
			isUnion = linkChain[i-1].IsUnion
		} else {
			isUnion = linkChain[i].IsUnion
		}

		// A phase is included if its predecessor was included AND the link is a union
		if _, ok := included[i-1]; ok && isUnion {
			included[i] = struct{}{}
		} else {
			// Once a link is not a union, subsequent phases in that direction are not included
			// (unless reached by another path, but this is a linear chain)
			break
		}
	}

	return included
}

// deduplicateSpansInTrace ensures each span ID appears only once in the trace metadata
func (s *asyncSearchSharder) deduplicateSpansInTrace(trace *tempopb.TraceSearchMetadata) {
	if trace == nil {
		return
	}

	seen := make(map[string]struct{})

	if trace.SpanSet != nil {
		newSpans := make([]*tempopb.Span, 0, len(trace.SpanSet.Spans))
		for _, span := range trace.SpanSet.Spans {
			if span.SpanID == "" {
				continue
			}
			if _, ok := seen[span.SpanID]; !ok {
				seen[span.SpanID] = struct{}{}
				newSpans = append(newSpans, span)
			}
		}
		if len(newSpans) > 0 {
			trace.SpanSet.Spans = newSpans
		} else {
			trace.SpanSet = nil
		}
	}

	var newSpanSets []*tempopb.SpanSet
	for _, ss := range trace.SpanSets {
		newSpans := make([]*tempopb.Span, 0, len(ss.Spans))
		for _, span := range ss.Spans {
			if span.SpanID == "" {
				continue
			}
			if _, ok := seen[span.SpanID]; !ok {
				seen[span.SpanID] = struct{}{}
				newSpans = append(newSpans, span)
			}
		}
		if len(newSpans) > 0 {
			ss.Spans = newSpans
			newSpanSets = append(newSpanSets, ss)
		}
	}
	trace.SpanSets = newSpanSets
}

// tracesToResponses converts a list of traces back into a streaming response
func (s *asyncSearchSharder) tracesToResponses(
	_ context.Context,
	traces []*tempopb.TraceSearchMetadata,
	metrics *tempopb.SearchMetrics,
) pipeline.Responses[combiner.PipelineResponse] {
	// Create a search response with the filtered traces
	searchResp := &tempopb.SearchResponse{
		Traces:  traces,
		Metrics: metrics,
	}

	// Marshal to JSON using jsonpb for correct protobuf handling
	var buf bytes.Buffer
	marshaler := &jsonpb.Marshaler{EmitDefaults: true}
	if err := marshaler.Marshal(&buf, searchResp); err != nil {
		level.Error(s.logger).Log("msg", "failed to marshal filtered traces", "err", err)
		return pipeline.NewAsyncResponse(nil)
	}
	body := buf.Bytes()

	// Create HTTP response
	httpResp := &http.Response{
		StatusCode: 200,
		Header: http.Header{
			"Content-Type": []string{"application/json"},
		},
		Body: io.NopCloser(bytes.NewReader(body)),
	}

	// Wrap in pipeline response
	filteredResp := &filteredResponse{
		httpResp: httpResp,
	}

	// Return as single response
	return &multiResponse{
		responses: []combiner.PipelineResponse{filteredResp},
		index:     0,
	}
}

// filteredResponse implements combiner.PipelineResponse for filtered results
type filteredResponse struct {
	httpResp *http.Response
}

func (f *filteredResponse) HTTPResponse() *http.Response {
	return f.httpResp
}

func (f *filteredResponse) RequestData() any {
	return nil
}

func (f *filteredResponse) IsMetadata() bool {
	return false
}

// executePhaseAndExtractSpanIDs executes a single phase of link traversal
// Returns the responses, the extracted traces, and span IDs from the results for use in the next phase
func (s *asyncSearchSharder) executePhaseAndExtractSpanIDs(
	ctx context.Context,
	parent pipeline.Request,
	query string,
	searchReq *tempopb.SearchRequest,
	tenantID string,
	maxSpanIDs int,
	maxTraces uint32,
	collectResponses bool,
) (pipeline.Responses[combiner.PipelineResponse], []*tempopb.TraceSearchMetadata, []string, *tempopb.SearchMetrics, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, nil, nil, err
	}

	// Create a new search request with the phase query
	phaseReq := cloneSearchRequest(searchReq)
	phaseReq.Query = query

	// Create a new parent request with the phase query
	// We need to do this because backendRequests will clone the parent request
	// and we don't want the original query with link operators
	phaseParent, err := s.createPhaseParentRequest(ctx, parent, phaseReq)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	// Build requests for this phase - query both ingesters and backend blocks
	// This ensures we find spans regardless of whether they've been flushed yet
	reqCh, jobMetrics, err := s.buildSearchRequestChannel(ctx, tenantID, phaseParent, phaseReq, 100, func(err error) {
		level.Error(s.logger).Log("msg", "failed to build backend requests for phase", "err", err)
	})
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to build ingester requests: %w", err)
	}

	// Execute requests and collect responses
	phaseResp := pipeline.NewAsyncSharderChan(ctx, s.cfg.ConcurrentRequests, reqCh, pipeline.NewAsyncResponse(jobMetrics), s.next)

	// Collect all responses, traces and extract span IDs
	// We need to buffer responses to return them and also extract span IDs
	responses := make([]combiner.PipelineResponse, 0)
	allTraces := make([]*tempopb.TraceSearchMetadata, 0)
	spanLimit := maxSpanIDs
	if spanLimit <= 0 {
		spanLimit = defaultMaxSpanIDs
	}
	traceLimit := int(maxTraces)
	if traceLimit <= 0 {
		traceLimit = defaultMaxTraces
	}
	spanIDs := make([]string, 0, spanLimit)
	seenSpanIDs := make(map[string]struct{})
	invalidSpanIDs := 0
	metricsCombiner := combiner.NewSearchMetricsCombiner()

	for {
		// Short-circuit if we have reached our limits
		if len(spanIDs) >= spanLimit && len(allTraces) >= traceLimit {
			if !collectResponses || len(responses) >= traceLimit {
				break
			}
		}

		resp, done, err := phaseResp.Next(ctx)
		if done {
			break
		}
		if err != nil {
			return nil, nil, nil, nil, fmt.Errorf("error reading phase response: %w", err)
		}
		if err := ctx.Err(); err != nil {
			return nil, nil, nil, nil, err
		}

		// Store response for later
		if resp != nil {
			if resp.IsMetadata() {
				if sj, ok := resp.RequestData().(*combiner.SearchJobResponse); ok && sj != nil {
					sjMetrics := &tempopb.SearchMetrics{
						TotalBlocks:     uint32(sj.TotalBlocks), //nolint:gosec
						TotalJobs:       uint32(sj.TotalJobs),   //nolint:gosec
						TotalBlockBytes: sj.TotalBytes,
					}
					metricsCombiner.CombineMetadata(sjMetrics, resp)
				}
			}

			if collectResponses && len(responses) < traceLimit {
				responses = append(responses, resp)
			}

			// Extract traces and span IDs from non-metadata responses
			if !resp.IsMetadata() {
				httpResp := resp.HTTPResponse()
				if httpResp != nil && httpResp.Body != nil {
					traces, ids, phaseMetrics, err := extractTracesAndSpanIDsFromResponse(httpResp, s.logger)
					if err != nil {
						level.Warn(s.logger).Log("msg", "failed to extract traces and span IDs", "err", err)
						continue
					}
					metricsCombiner.Combine(phaseMetrics, resp)

					if len(allTraces) < traceLimit {
						remaining := traceLimit - len(allTraces)
						if remaining > len(traces) {
							remaining = len(traces)
						}
						allTraces = append(allTraces, traces[:remaining]...)
					}

					// Deduplicate span IDs
					for _, id := range ids {
						if !isValidSpanIDString(id) {
							invalidSpanIDs++
							continue
						}
						if _, seen := seenSpanIDs[id]; !seen {
							seenSpanIDs[id] = struct{}{}
							if len(spanIDs) < spanLimit {
								spanIDs = append(spanIDs, id)
							}
						}
					}
				}
			}
		}
	}

	if invalidSpanIDs > 0 {
		level.Debug(s.logger).Log("msg", "ignored invalid span IDs during phase extraction", "count", invalidSpanIDs)
	}

	// Convert buffered responses back to a Responses interface
	// We create a simple multi-response wrapper
	var multiResp pipeline.Responses[combiner.PipelineResponse]
	if collectResponses {
		multiResp = &multiResponse{responses: responses, index: 0}
	} else {
		multiResp = pipeline.NewAsyncResponse(nil)
	}

	level.Info(s.logger).Log("msg", "phase execution complete", "responsesCollected", len(responses), "spanIDsExtracted", len(spanIDs), "tracesExtracted", len(allTraces))

	return multiResp, allTraces, spanIDs, metricsCombiner.Metrics, nil
}

// multiResponse implements Responses interface for a slice of buffered responses
type multiResponse struct {
	responses []combiner.PipelineResponse
	index     int
}

func (m *multiResponse) Next(ctx context.Context) (combiner.PipelineResponse, bool, error) {
	if err := ctx.Err(); err != nil {
		return nil, true, err
	}

	if m.index >= len(m.responses) {
		return nil, true, nil
	}

	resp := m.responses[m.index]
	m.index++

	done := m.index >= len(m.responses)
	return resp, done, nil
}

// buildLinkFilterQuery builds a query with link:spanID regex filter
func buildLinkFilterQuery(conditions traceql.SpansetExpression, spanIDs []string, maxSpanIDs int) (string, bool) {
	// Example output: { link:spanID =~ "(span1|span2|span3)" && name="backend" }

	// Limit span IDs
	limitedIDs := make([]string, 0, len(spanIDs))
	for _, id := range spanIDs {
		if !isValidSpanIDString(id) {
			continue
		}
		limitedIDs = append(limitedIDs, id)
		if maxSpanIDs > 0 && len(limitedIDs) >= maxSpanIDs {
			break
		}
	}
	if len(limitedIDs) == 0 {
		return "", false
	}

	// Build regex pattern: "(id1|id2|id3)"
	regexPattern := "(" + strings.Join(limitedIDs, "|") + ")"

	// Get conditions as string and strip outer braces for embedding
	conditionsStr := buildQueryFromExpression(conditions)
	conditionsStr = strings.TrimPrefix(conditionsStr, "{")
	conditionsStr = strings.TrimSuffix(conditionsStr, "}")
	conditionsStr = strings.TrimSpace(conditionsStr)

	// If conditions are empty, just use link filter
	if conditionsStr == "" || conditionsStr == "true" {
		return fmt.Sprintf(`{ link:spanID =~ "%s" }`, regexPattern), true
	}

	// Combine: { link:spanID =~ "pattern" && conditions }
	return fmt.Sprintf(`{ link:spanID =~ "%s" && %s }`, regexPattern, conditionsStr), true
}

// buildQueryFromExpression converts a SpansetExpression to query string
func buildQueryFromExpression(expr traceql.SpansetExpression) string {
	if expr == nil {
		return ""
	}

	// Use expr.String() to get query representation
	// Keep the curly braces as they're required for valid TraceQL syntax
	return strings.TrimSpace(expr.String())
}

// extractTracesAndSpanIDsFromResponse extracts traces and span IDs from a search response
// Handles both JSON and protobuf responses
func extractTracesAndSpanIDsFromResponse(httpResp *http.Response, logger log.Logger) ([]*tempopb.TraceSearchMetadata, []string, *tempopb.SearchMetrics, error) {
	if httpResp == nil || httpResp.Body == nil {
		return nil, nil, nil, nil
	}

	// Read body
	bodyBytes, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Restore body for other consumers
	httpResp.Body = io.NopCloser(bytes.NewReader(bodyBytes))

	// Parse based on content type
	contentType := httpResp.Header.Get("Content-Type")

	var searchResp tempopb.SearchResponse

	if strings.Contains(contentType, "application/json") {
		// Parse JSON using jsonpb for correct protobuf handling
		unmarshaler := &jsonpb.Unmarshaler{AllowUnknownFields: true}
		if err := unmarshaler.Unmarshal(bytes.NewReader(bodyBytes), &searchResp); err != nil {
			return nil, nil, nil, fmt.Errorf("failed to unmarshal JSON response: %w", err)
		}
	} else if strings.Contains(contentType, "application/protobuf") {
		// Parse protobuf
		if err := searchResp.Unmarshal(bodyBytes); err != nil {
			return nil, nil, nil, fmt.Errorf("failed to unmarshal protobuf response: %w", err)
		}
	} else {
		// Try JSON by default using jsonpb
		unmarshaler := &jsonpb.Unmarshaler{AllowUnknownFields: true}
		if err := unmarshaler.Unmarshal(bytes.NewReader(bodyBytes), &searchResp); err != nil {
			level.Debug(logger).Log("msg", "failed to unmarshal search response with fallback", "contentType", contentType, "err", err)
			return nil, nil, nil, nil
		}
	}

	// Extract span IDs from traces
	spanIDs := make([]string, 0)
	for _, trace := range searchResp.Traces {
		forEachSpanInTrace(trace, func(span *tempopb.Span) {
			if span.SpanID != "" {
				spanIDs = append(spanIDs, span.SpanID)
			}
		})
	}

	return searchResp.Traces, spanIDs, searchResp.Metrics, nil
}

// extractSpanIDsFromResponse is a convenience wrapper used in tests
func extractSpanIDsFromResponse(httpResp *http.Response) ([]string, error) {
	_, spanIDs, _, err := extractTracesAndSpanIDsFromResponse(httpResp, log.NewNopLogger())
	return spanIDs, err
}

// isValidSpanIDString validates that a span ID string is exactly 16 hex characters
// This is stricter than util.HexStringToSpanID which accepts any length up to 8 bytes.
// OpenTelemetry span IDs are always exactly 8 bytes (16 hex chars).
func isValidSpanIDString(id string) bool {
	if len(id) != 16 {
		return false
	}
	_, err := util.HexStringToSpanID(id)
	return err == nil
}

// forEachSpanInTrace iterates over all spans in a trace (both SpanSets and deprecated SpanSet)
// and calls the provided function for each span
func forEachSpanInTrace(trace *tempopb.TraceSearchMetadata, fn func(*tempopb.Span)) {
	if trace == nil {
		return
	}

	for _, spanSet := range trace.SpanSets {
		for _, span := range spanSet.Spans {
			fn(span)
		}
	}

	if trace.SpanSet != nil {
		for _, span := range trace.SpanSet.Spans {
			fn(span)
		}
	}
}

func applyTraceLimit(traces []*tempopb.TraceSearchMetadata, limit uint32) ([]*tempopb.TraceSearchMetadata, bool) {
	if limit == 0 || uint32(len(traces)) <= limit {
		return traces, false
	}
	return traces[:limit], true
}

func (s *asyncSearchSharder) createPhaseParentRequest(ctx context.Context, parent pipeline.Request, phaseReq *tempopb.SearchRequest) (pipeline.Request, error) {
	origReq := parent.HTTPRequest()
	// Clone the URL but clear the query parameters
	freshURL := *origReq.URL
	freshURL.RawQuery = ""

	freshReq, err := http.NewRequestWithContext(ctx, origReq.Method, freshURL.String(), http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create fresh request: %w", err)
	}
	// Copy headers from original request
	freshReq.Header = origReq.Header.Clone()

	phaseHTTPReq, err := api.BuildSearchRequest(freshReq, phaseReq)
	if err != nil {
		return nil, fmt.Errorf("failed to build phase request: %w", err)
	}
	return pipeline.NewHTTPRequest(phaseHTTPReq), nil
}

func cloneSearchRequest(req *tempopb.SearchRequest) *tempopb.SearchRequest {
	if req == nil {
		return nil
	}
	return proto.Clone(req).(*tempopb.SearchRequest)
}

func mergeSearchMetrics(total, phase *tempopb.SearchMetrics) {
	if phase == nil || total == nil {
		return
	}
	total.InspectedTraces += phase.InspectedTraces
	total.InspectedBytes += phase.InspectedBytes
	total.InspectedSpans += phase.InspectedSpans
	total.TotalBlocks += phase.TotalBlocks
	total.TotalJobs += phase.TotalJobs
	total.TotalBlockBytes += phase.TotalBlockBytes
	total.CompletedJobs += phase.CompletedJobs
}
