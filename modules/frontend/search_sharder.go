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
	"github.com/segmentio/fasthash/fnv1a"

	"github.com/grafana/tempo/modules/frontend/combiner"
	"github.com/grafana/tempo/modules/frontend/pipeline"
	"github.com/grafana/tempo/modules/frontend/shardtracker"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/traceql"
	"github.com/grafana/tempo/pkg/validation"
	"github.com/grafana/tempo/tempodb"
	"github.com/grafana/tempo/tempodb/backend"
)

const (
	defaultTargetBytesPerRequest = 100 * 1024 * 1024
	defaultConcurrentRequests    = 1000
	defaultMostRecentShards      = 200
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
		return asyncSearchSharder{
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
func (s asyncSearchSharder) RoundTrip(pipelineRequest pipeline.Request) (pipeline.Responses[combiner.PipelineResponse], error) {
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
	level.Info(s.logger).Log("msg", "parsed query", "query", searchReq.Query, "parseErr", err, "hasLinkTraversal", err == nil && rootExpr != nil && rootExpr.HasLinkTraversal())
	if err == nil && rootExpr.HasLinkTraversal() {
		maxSpanIDsPerQuery := s.overrides.MaxSpanIDsPerLinkQuery(tenantID)
		if maxSpanIDsPerQuery <= 0 {
			// If not configured, use the query limit as the max span IDs per phase
			maxSpanIDsPerQuery = int(searchReq.Limit)
		}
		level.Info(s.logger).Log("msg", "using cross-trace link traversal", "maxSpanIDsPerQuery", maxSpanIDsPerQuery, "query", searchReq.Query)
		// Use cross-trace execution path (BACKEND BLOCKS ONLY)
		return s.crossTraceRoundTrip(ctx, pipelineRequest, searchReq, rootExpr, maxSpanIDsPerQuery, tenantID)
	}

	// buffer of shards+1 allows us to insert ingestReq and metrics
	reqCh := make(chan pipeline.Request, s.cfg.IngesterShards+1)

	// build request to search ingesters based on query_ingesters_until config and time range
	// pass subCtx in requests so we can cancel and exit early
	jobMetrics, err := s.ingesterRequests(tenantID, pipelineRequest, *searchReq, reqCh)
	if err != nil {
		return nil, err
	}

	// pass subCtx in requests so we can cancel and exit early
	s.backendRequests(ctx, tenantID, pipelineRequest, searchReq, jobMetrics, reqCh, func(err error) {
		// todo: actually find a way to return this error to the user
		s.logger.Log("msg", "search: failed to build backend requests", "err", err)
	})

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
				IndexPageSize: m.IndexPageSize,
				TotalRecords:  m.TotalRecords,
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

	// Create a copy of searchReq with no limit for phase execution
	// This ensures we collect all traces in the chain, not just the first N
	// The combiner will handle limiting the final result
	unlimitedReq := &tempopb.SearchRequest{
		Query:           searchReq.Query,
		Limit:           0, // No limit during phases to get complete chains
		Start:           searchReq.Start,
		End:             searchReq.End,
		SpansPerSpanSet: searchReq.SpansPerSpanSet,
		RF1After:        searchReq.RF1After,
	}

	// Execute all phases sequentially and return combined results
	// We need to execute phases sequentially since each phase depends on the previous one's results

	// Start with the terminal phase (first in execution order)
	terminal := linkChain[0]
	terminalQuery := buildQueryFromExpression(terminal.Conditions)

	level.Info(s.logger).Log(
		"msg", "executing terminal phase",
		"phase", 0,
		"query", terminalQuery,
	)

	// Execute terminal phase and collect span IDs
	terminalResp, terminalTraces, spanIDs, err := s.executePhaseAndExtractSpanIDs(ctx, pipelineRequest, terminalQuery, unlimitedReq, tenantID)
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
		phase := linkChain[i]

		// Build query with link:spanID filter
		phaseQuery := buildLinkFilterQuery(phase.Conditions, spanIDs, maxSpanIDs)

		// Expand time range for non-terminal phases
		// Linked spans may start after the terminal span, so we need a wider window
		phaseReq := s.expandTimeRangeForPhase(unlimitedReq, i, len(linkChain))

		level.Info(s.logger).Log(
			"msg", "executing link phase",
			"phase", i,
			"query", phaseQuery,
			"inputSpanIDs", len(spanIDs),
			"timeRange", fmt.Sprintf("%d-%d", phaseReq.Start, phaseReq.End),
		)

		// Execute this phase and collect both results and span IDs
		_, phaseTraces, nextSpanIDs, err := s.executePhaseAndExtractSpanIDs(ctx, pipelineRequest, phaseQuery, phaseReq, tenantID)
		if err != nil {
			return pipeline.NewBadRequest(fmt.Errorf("phase %d failed: %w", i, err)), nil
		}

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
	return s.filterCompleteChains(ctx, allTracesByPhase, allSpanIDsByPhase, linkChain), nil
}

// expandTimeRangeForPhase expands the time range for non-terminal phases to capture
// linked spans that may start after the initiating span
func (s *asyncSearchSharder) expandTimeRangeForPhase(
	baseReq *tempopb.SearchRequest,
	phaseIndex int,
	totalPhases int,
) *tempopb.SearchRequest {
	if phaseIndex == 0 {
		// Terminal phase uses original time range
		return baseReq
	}

	// For subsequent phases, expand the time window to account for:
	// 1. Linked spans may start/end outside the original query window
	// 2. Async operations may have significant delays
	// 3. Each subsequent phase may be further delayed

	originalDuration := uint64(baseReq.End - baseReq.Start)

	// Expand the end time based on phase depth
	// Each phase gets progressively more buffer since spans further in the chain
	// may be increasingly delayed from the terminal span
	expansionFactor := uint64(phaseIndex) // Phases 1, 2, 3... get 1x, 2x, 3x expansion
	expandedEnd := baseReq.End + uint32(originalDuration*expansionFactor)

	// Also consider expanding start time backwards for ->> (forward link) traversal
	// where earlier spans may have started before the terminal
	expandedStart := baseReq.Start
	if originalDuration > 60 { // Only expand start for longer queries (>60s)
		expandedStart = baseReq.Start - uint32(originalDuration*expansionFactor/2)
	}

	return &tempopb.SearchRequest{
		Query:           baseReq.Query,
		Limit:           baseReq.Limit,
		Start:           expandedStart,
		End:             expandedEnd,
		SpansPerSpanSet: baseReq.SpansPerSpanSet,
		RF1After:        baseReq.RF1After,
	}
}

// filterCompleteChains ensures only traces with complete link chains are returned
// It validates that for each trace, all linked traces exist in the result set
func (s *asyncSearchSharder) filterCompleteChains(
	ctx context.Context,
	tracesByPhase [][]*tempopb.TraceSearchMetadata,
	spanIDsByPhase [][]string,
	linkChain []*traceql.LinkOperationInfo,
) pipeline.Responses[combiner.PipelineResponse] {
	// Build span ID mappings and index traces by span ID
	spanIDToTrace := make(map[string]*tempopb.TraceSearchMetadata)
	allTraces := make([]*tempopb.TraceSearchMetadata, 0)

	// Index traces by their span IDs
	for _, phaseTraces := range tracesByPhase {
		for _, trace := range phaseTraces {
			allTraces = append(allTraces, trace)

			// Index each span in the trace
			for _, spanSet := range trace.SpanSets {
				for _, span := range spanSet.Spans {
					if span.SpanID != "" {
						spanIDToTrace[span.SpanID] = trace
					}
				}
			}
			if trace.SpanSet != nil {
				for _, span := range trace.SpanSet.Spans {
					if span.SpanID != "" {
						spanIDToTrace[span.SpanID] = trace
					}
				}
			}
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

	// Now filter traces to only include complete chains
	// Identify all (traceID, phaseIdx) pairs that are part of a complete chain
	type phaseKey struct {
		traceID string
		phase   int
	}
	completePhaseTraces := make(map[phaseKey]*tempopb.TraceSearchMetadata)
	completeTraceIDs := make(map[string]struct{})

	for phaseIdx, phaseTraces := range tracesByPhase {
		for _, trace := range phaseTraces {
			if s.isCompleteChain(trace, phaseIdx, validSpanIDsByPhase, spanIDToTrace) {
				key := phaseKey{trace.TraceID, phaseIdx}
				completePhaseTraces[key] = trace
				completeTraceIDs[trace.TraceID] = struct{}{}
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

	level.Info(s.logger).Log(
		"msg", "filtered for complete chains",
		"totalTraces", len(allTraces),
		"completeChainTraces", len(completeTraces),
	)

	// Convert filtered traces back to responses
	return s.tracesToResponses(ctx, completeTraces)
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

// isCompleteChain checks if a trace is part of a complete link chain
func (s *asyncSearchSharder) isCompleteChain(
	trace *tempopb.TraceSearchMetadata,
	tracePhase int,
	validSpanIDsByPhase []map[string]struct{},
	spanIDToTrace map[string]*tempopb.TraceSearchMetadata,
) bool {
	// Get the span IDs from this trace
	var traceSpanIDs []string
	extractIDs := func(spans []*tempopb.Span) {
		for _, span := range spans {
			if span.SpanID != "" {
				traceSpanIDs = append(traceSpanIDs, span.SpanID)
			}
		}
	}

	for _, spanSet := range trace.SpanSets {
		extractIDs(spanSet.Spans)
	}
	if trace.SpanSet != nil {
		extractIDs(trace.SpanSet.Spans)
	}

	// Check link to previous phase (if not terminal)
	if tracePhase > 0 {
		foundLinkToPrev := false
		checkLinks := func(spans []*tempopb.Span) bool {
			for _, span := range spans {
				for _, attr := range span.Attributes {
					if attr.Key == "link:spanID" && attr.Value != nil {
						targetID := attr.Value.GetStringValue()
						if _, ok := validSpanIDsByPhase[tracePhase-1][targetID]; ok {
							return true
						}
					}
				}
			}
			return false
		}

		for _, spanSet := range trace.SpanSets {
			if checkLinks(spanSet.Spans) {
				foundLinkToPrev = true
				break
			}
		}
		if !foundLinkToPrev && trace.SpanSet != nil {
			foundLinkToPrev = checkLinks(trace.SpanSet.Spans)
		}

		if !foundLinkToPrev {
			return false
		}
	}

	// Check if something from the next phase links to us (if not last)
	if tracePhase < len(validSpanIDsByPhase)-1 {
		foundLinkFromNext := false
		for _, mySpanID := range traceSpanIDs {
			// Check if any trace in the next phase has a link attribute pointing to this span
			// We only need to check traces that own span IDs from the next phase
			for nextPhaseSpanID := range validSpanIDsByPhase[tracePhase+1] {
				if nextTrace, ok := spanIDToTrace[nextPhaseSpanID]; ok {
					if s.traceLinksToSpan(nextTrace, mySpanID) {
						foundLinkFromNext = true
						break
					}
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

// traceLinksToSpan checks if a trace has a link:spanID attribute pointing to the given span
func (s *asyncSearchSharder) traceLinksToSpan(trace *tempopb.TraceSearchMetadata, targetSpanID string) bool {
	checkSpans := func(spans []*tempopb.Span) bool {
		for _, span := range spans {
			for _, attr := range span.Attributes {
				if attr.Key == "link:spanID" && attr.Value != nil && attr.Value.GetStringValue() == targetSpanID {
					return true
				}
			}
		}
		return false
	}

	for _, spanSet := range trace.SpanSets {
		if checkSpans(spanSet.Spans) {
			return true
		}
	}
	if trace.SpanSet != nil && checkSpans(trace.SpanSet.Spans) {
		return true
	}

	return false
}

// tracesToResponses converts a list of traces back into a streaming response
func (s *asyncSearchSharder) tracesToResponses(
	_ context.Context,
	traces []*tempopb.TraceSearchMetadata,
) pipeline.Responses[combiner.PipelineResponse] {
	// Create a search response with the filtered traces
	searchResp := &tempopb.SearchResponse{
		Traces:  traces,
		Metrics: &tempopb.SearchMetrics{},
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

// combinePhaseResponses combines results from multiple phases into a single response stream
// This returns all spans in the link chain (gateway + backend + database)
func (s *asyncSearchSharder) combinePhaseResponses(
	ctx context.Context,
	responses []pipeline.Responses[combiner.PipelineResponse],
) pipeline.Responses[combiner.PipelineResponse] {
	if len(responses) == 0 {
		return pipeline.NewAsyncResponse(nil)
	}
	if len(responses) == 1 {
		return responses[0]
	}

	// Create an async response to collect all results
	asyncResp := newCombinedResponse(ctx, responses, s.logger, nil)
	return asyncResp
}

// combinePhaseResponsesWithQueries combines results from multiple phases with query tracking
func (s *asyncSearchSharder) combinePhaseResponsesWithQueries(
	ctx context.Context,
	responses []pipeline.Responses[combiner.PipelineResponse],
	queries []string,
) pipeline.Responses[combiner.PipelineResponse] {
	if len(responses) == 0 {
		return pipeline.NewAsyncResponse(nil)
	}
	if len(responses) == 1 {
		return responses[0]
	}

	// Create an async response to collect all results with query tracking
	asyncResp := newCombinedResponse(ctx, responses, s.logger, queries)
	return asyncResp
}

// combinedResponse implements Responses by iterating through multiple response streams
type combinedResponse struct {
	ctx          context.Context
	responses    []pipeline.Responses[combiner.PipelineResponse]
	current      int
	logger       log.Logger
	resultCounts []int    // Track results per phase
	queries      []string // Track query for each phase (optional)
}

func newCombinedResponse(ctx context.Context, responses []pipeline.Responses[combiner.PipelineResponse], logger log.Logger, queries []string) *combinedResponse {
	return &combinedResponse{
		ctx:          ctx,
		responses:    responses,
		current:      0,
		logger:       logger,
		resultCounts: make([]int, len(responses)),
		queries:      queries,
	}
}

func (c *combinedResponse) Next(ctx context.Context) (combiner.PipelineResponse, bool, error) {
	// Iterate through all response streams
	for c.current < len(c.responses) {
		resp, done, err := c.responses[c.current].Next(ctx)

		if err != nil {
			return nil, false, err
		}

		if !done {
			// Count this response if it's not metadata
			if resp != nil && !resp.IsMetadata() {
				c.resultCounts[c.current]++
			}
			// Return this response
			return resp, false, nil
		}

		// This stream is done, log the count for this phase and move to next
		logFields := []interface{}{
			"msg", "phase results collected",
			"phase", c.current,
			"resultCount", c.resultCounts[c.current],
		}
		// Add query if available
		if c.queries != nil && c.current < len(c.queries) {
			logFields = append(logFields, "query", c.queries[c.current])
		}
		level.Info(c.logger).Log(logFields...)
		c.current++
	}

	// All streams exhausted, log total summary
	totalResults := 0
	for _, count := range c.resultCounts {
		totalResults += count
	}
	level.Info(c.logger).Log(
		"msg", "all phases complete",
		"totalPhases", len(c.responses),
		"totalResults", totalResults,
	)
	return nil, true, nil
}

// limitedCombinedResponse collects all traces from all phases and limits the final result
type limitedCombinedResponse struct {
	ctx            context.Context
	responses      []pipeline.Responses[combiner.PipelineResponse]
	limit          uint32
	allTraces      []*tempopb.TraceSearchMetadata
	collectionDone bool
	index          int
	logger         log.Logger
	resultCounts   []int // Track traces collected per phase
}

func newLimitedCombinedResponse(ctx context.Context, responses []pipeline.Responses[combiner.PipelineResponse], limit uint32, logger log.Logger) *limitedCombinedResponse {
	return &limitedCombinedResponse{
		ctx:          ctx,
		responses:    responses,
		limit:        limit,
		allTraces:    make([]*tempopb.TraceSearchMetadata, 0),
		logger:       logger,
		resultCounts: make([]int, len(responses)),
	}
}

func (l *limitedCombinedResponse) Next(ctx context.Context) (combiner.PipelineResponse, bool, error) {
	// First, collect all traces from all phases if not done yet
	if !l.collectionDone {
		traceIDs := make(map[string]struct{})

		for phaseIdx, phaseResp := range l.responses {
			phaseTraceCount := 0
			for {
				resp, done, err := phaseResp.Next(ctx)
				if done || err != nil {
					break
				}
				if resp == nil || resp.IsMetadata() {
					continue
				}

				// Parse the response to extract traces
				httpResp := resp.HTTPResponse()
				if httpResp == nil || httpResp.Body == nil {
					continue
				}

				bodyBytes, err := io.ReadAll(httpResp.Body)
				if err != nil {
					continue
				}
				httpResp.Body = io.NopCloser(bytes.NewReader(bodyBytes))

				var searchResp tempopb.SearchResponse
				contentType := httpResp.Header.Get("Content-Type")
				if strings.Contains(contentType, "application/json") {
					unmarshaler := &jsonpb.Unmarshaler{AllowUnknownFields: true}
					if err := unmarshaler.Unmarshal(bytes.NewReader(bodyBytes), &searchResp); err != nil {
						continue
					}
				} else if strings.Contains(contentType, "application/protobuf") {
					if err := searchResp.Unmarshal(bodyBytes); err != nil {
						continue
					}
				} else {
					unmarshaler := &jsonpb.Unmarshaler{AllowUnknownFields: true}
					_ = unmarshaler.Unmarshal(bytes.NewReader(bodyBytes), &searchResp)
				}

				// Collect unique traces
				for _, trace := range searchResp.Traces {
					phaseTraceCount++
					if _, seen := traceIDs[trace.TraceID]; !seen {
						l.allTraces = append(l.allTraces, trace)
						traceIDs[trace.TraceID] = struct{}{}

						// Stop if we've reached the limit
						if uint32(len(l.allTraces)) >= l.limit {
							break
						}
					}
				}

				if uint32(len(l.allTraces)) >= l.limit {
					break
				}
			}

			// Log this phase's results
			l.resultCounts[phaseIdx] = phaseTraceCount
			level.Info(l.logger).Log(
				"msg", "phase traces collected",
				"phase", phaseIdx,
				"traceCount", phaseTraceCount,
				"uniqueTracesCollected", len(l.allTraces),
			)

			if uint32(len(l.allTraces)) >= l.limit {
				break
			}
		}

		l.collectionDone = true
		l.index = 0
	}

	// Now return traces one at a time (or in batches)
	if l.index >= len(l.allTraces) {
		return nil, true, nil
	}

	// Return all collected traces in a single response
	searchResp := &tempopb.SearchResponse{
		Traces:  l.allTraces,
		Metrics: &tempopb.SearchMetrics{},
	}

	// Marshal to JSON using jsonpb for correct protobuf handling
	var buf bytes.Buffer
	marshaler := &jsonpb.Marshaler{EmitDefaults: true}
	if err := marshaler.Marshal(&buf, searchResp); err != nil {
		return nil, false, fmt.Errorf("failed to marshal limited traces: %w", err)
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
	result := &filteredResponse{
		httpResp: httpResp,
	}

	l.index = len(l.allTraces) // Mark as done
	return result, false, nil
}

// executePhaseAndExtractSpanIDs executes a single phase of link traversal
// Returns the responses, the extracted traces, and span IDs from the results for use in the next phase
func (s *asyncSearchSharder) executePhaseAndExtractSpanIDs(
	ctx context.Context,
	parent pipeline.Request,
	query string,
	searchReq *tempopb.SearchRequest,
	tenantID string,
) (pipeline.Responses[combiner.PipelineResponse], []*tempopb.TraceSearchMetadata, []string, error) {
	// Create a new search request with the phase query
	phaseReq := &tempopb.SearchRequest{
		Query:           query,
		Limit:           searchReq.Limit,
		Start:           searchReq.Start,
		End:             searchReq.End,
		SpansPerSpanSet: searchReq.SpansPerSpanSet,
		RF1After:        searchReq.RF1After,
	}

	// Create a new parent request with the phase query
	// We need to do this because backendRequests will clone the parent request
	// and we don't want the original query with link operators
	origReq := parent.HTTPRequest()
	// Clone the URL but clear the query parameters
	freshURL := *origReq.URL
	freshURL.RawQuery = ""

	freshReq, err := http.NewRequestWithContext(ctx, origReq.Method, freshURL.String(), http.NoBody)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create fresh request: %w", err)
	}
	// Copy headers from original request
	freshReq.Header = origReq.Header.Clone()

	phaseHTTPReq, err := api.BuildSearchRequest(freshReq, phaseReq)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to build phase request: %w", err)
	}
	phaseParent := pipeline.NewHTTPRequest(phaseHTTPReq)

	// Build requests for this phase - query both ingesters and backend blocks
	// This ensures we find spans regardless of whether they've been flushed yet
	reqCh := make(chan pipeline.Request, s.cfg.IngesterShards+100)

	// Build ingester requests
	jobMetrics, err := s.ingesterRequests(tenantID, phaseParent, *phaseReq, reqCh)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to build ingester requests: %w", err)
	}

	// Build backend requests
	s.backendRequests(ctx, tenantID, phaseParent, phaseReq, jobMetrics, reqCh, func(err error) {
		level.Error(s.logger).Log("msg", "failed to build backend requests for phase", "err", err)
	})

	// Execute requests and collect responses
	phaseResp := pipeline.NewAsyncSharderChan(ctx, s.cfg.ConcurrentRequests, reqCh, pipeline.NewAsyncResponse(jobMetrics), s.next)

	// Collect all responses, traces and extract span IDs
	// We need to buffer responses to return them and also extract span IDs
	responses := make([]combiner.PipelineResponse, 0)
	allTraces := make([]*tempopb.TraceSearchMetadata, 0)
	spanIDs := make([]string, 0, 1000)
	seenSpanIDs := make(map[string]struct{})

	for {
		resp, done, err := phaseResp.Next(ctx)
		if done {
			break
		}
		if err != nil {
			return nil, nil, nil, fmt.Errorf("error reading phase response: %w", err)
		}

		// Store response for later
		if resp != nil {
			responses = append(responses, resp)

			// Extract traces and span IDs from non-metadata responses
			if !resp.IsMetadata() {
				httpResp := resp.HTTPResponse()
				if httpResp != nil && httpResp.Body != nil {
					traces, ids, err := extractTracesAndSpanIDsFromResponse(httpResp)
					if err != nil {
						level.Warn(s.logger).Log("msg", "failed to extract traces and span IDs", "err", err)
						continue
					}

					allTraces = append(allTraces, traces...)

					// Deduplicate span IDs
					for _, id := range ids {
						if _, seen := seenSpanIDs[id]; !seen {
							seenSpanIDs[id] = struct{}{}
							spanIDs = append(spanIDs, id)
						}
					}
				}
			}
		}
	}

	// Convert buffered responses back to a Responses interface
	// We create a simple multi-response wrapper
	multiResp := &multiResponse{responses: responses, index: 0}

	level.Info(s.logger).Log("msg", "phase execution complete", "responsesCollected", len(responses), "spanIDsExtracted", len(spanIDs), "tracesExtracted", len(allTraces))

	return multiResp, allTraces, spanIDs, nil
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
func buildLinkFilterQuery(conditions traceql.SpansetExpression, spanIDs []string, maxSpanIDs int) string {
	// Example output: { link:spanID =~ "(span1|span2|span3)" && name="backend" }

	// Limit span IDs
	limitedIDs := spanIDs
	if len(spanIDs) > maxSpanIDs {
		limitedIDs = spanIDs[:maxSpanIDs]
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
		return fmt.Sprintf(`{ link:spanID =~ "%s" }`, regexPattern)
	}

	// Combine: { link:spanID =~ "pattern" && conditions }
	return fmt.Sprintf(`{ link:spanID =~ "%s" && %s }`, regexPattern, conditionsStr)
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
func extractTracesAndSpanIDsFromResponse(httpResp *http.Response) ([]*tempopb.TraceSearchMetadata, []string, error) {
	if httpResp == nil || httpResp.Body == nil {
		return nil, nil, nil
	}

	// Read body
	bodyBytes, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read response body: %w", err)
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
			return nil, nil, fmt.Errorf("failed to unmarshal JSON response: %w", err)
		}
	} else if strings.Contains(contentType, "application/protobuf") {
		// Parse protobuf
		if err := searchResp.Unmarshal(bodyBytes); err != nil {
			return nil, nil, fmt.Errorf("failed to unmarshal protobuf response: %w", err)
		}
	} else {
		// Try JSON by default using jsonpb
		unmarshaler := &jsonpb.Unmarshaler{AllowUnknownFields: true}
		if err := unmarshaler.Unmarshal(bytes.NewReader(bodyBytes), &searchResp); err != nil {
			// Ignore and return empty
			return nil, nil, nil
		}
	}

	// Extract span IDs from traces
	spanIDs := make([]string, 0)
	for _, trace := range searchResp.Traces {
		// Extract from SpanSets (new format)
		for _, spanSet := range trace.SpanSets {
			for _, span := range spanSet.Spans {
				if span.SpanID != "" {
					spanIDs = append(spanIDs, span.SpanID)
				}
			}
		}

		// Also check deprecated SpanSet field for backwards compatibility
		if trace.SpanSet != nil {
			for _, span := range trace.SpanSet.Spans {
				if span.SpanID != "" {
					spanIDs = append(spanIDs, span.SpanID)
				}
			}
		}
	}

	return searchResp.Traces, spanIDs, nil
}
