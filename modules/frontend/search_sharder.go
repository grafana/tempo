package frontend

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"time"

	"github.com/go-kit/log" //nolint:all deprecated
	"github.com/grafana/dskit/user"
	"github.com/segmentio/fasthash/fnv1a"

	"github.com/grafana/tempo/modules/frontend/combiner"
	"github.com/grafana/tempo/modules/frontend/pipeline"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/traceql"
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
	QueryBackendAfter     time.Duration `yaml:"query_backend_after,omitempty"`
	QueryIngestersUntil   time.Duration `yaml:"query_ingesters_until,omitempty"`
	IngesterShards        int           `yaml:"ingester_shards,omitempty"`
	MostRecentShards      int           `yaml:"most_recent_shards,omitempty"`
	MaxSpansPerSpanSet    uint32        `yaml:"max_spans_per_span_set,omitempty"`
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

	searchReq, err := api.ParseSearchRequest(r)
	if err != nil {
		return pipeline.NewBadRequest(err), nil
	}

	// adjust limit based on config
	searchReq.Limit, err = adjustLimit(searchReq.Limit, s.cfg.DefaultLimit, s.cfg.MaxLimit)
	if err != nil {
		return pipeline.NewBadRequest(err), nil
	}

	requestCtx := r.Context()
	tenantID, err := user.ExtractOrgID(requestCtx)
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

	if s.cfg.MaxSpansPerSpanSet != 0 && searchReq.SpansPerSpanSet > s.cfg.MaxSpansPerSpanSet {
		return pipeline.NewBadRequest(fmt.Errorf("spans per span set exceeds %d. received %d", s.cfg.MaxSpansPerSpanSet, searchReq.SpansPerSpanSet)), nil
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

	blocks := blockMetasForSearch(s.reader.BlockMetas(tenantID), start, end, backend.DefaultReplicationFactor)

	// calculate metrics to return to the caller
	resp.TotalBlocks = len(blocks)

	blockIter := backendJobsFunc(blocks, s.cfg.TargetBytesPerRequest, s.cfg.MostRecentShards, searchReq.End)
	blockIter(func(jobs int, sz uint64, completedThroughTime uint32) {
		resp.TotalJobs += jobs
		resp.TotalBytes += sz

		resp.Shards = append(resp.Shards, combiner.SearchShards{
			TotalJobs:               uint32(jobs),
			CompletedThroughSeconds: completedThroughTime,
		})
	}, nil)

	go func() {
		buildBackendRequests(ctx, tenantID, parent, searchReq, blockIter, reqCh, errFn)
	}()
}

// ingesterRequest returns a new start and end time range for the backend as well as an http request
// that covers the ingesters. If nil is returned for the http.Request then there is no ingesters query.
// since this function modifies searchReq.Start and End we are taking a value instead of a pointer to prevent it from
// unexpectedly changing the passed searchReq.
func (s *asyncSearchSharder) ingesterRequests(tenantID string, parent pipeline.Request, searchReq tempopb.SearchRequest, reqCh chan pipeline.Request) (*combiner.SearchJobResponse, error) {
	resp := &combiner.SearchJobResponse{
		Shards: make([]combiner.SearchShards, 0, s.cfg.MostRecentShards+1), // +1 for the ingester shard
	}

	// request without start or end, search only in ingester
	if searchReq.Start == 0 || searchReq.End == 0 {
		// one shard that covers all time
		resp.TotalJobs = 1
		resp.Shards = append(resp.Shards, combiner.SearchShards{
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
	resp.Shards = append(resp.Shards, combiner.SearchShards{
		TotalJobs:               uint32(ingesterJobs),
		CompletedThroughSeconds: math.MaxUint32,
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
func buildBackendRequests(ctx context.Context, tenantID string, parent pipeline.Request, searchReq *tempopb.SearchRequest, blockIter func(shardIterFn, jobIterFn), reqCh chan<- pipeline.Request, errFn func(error)) {
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

		key := searchJobCacheKey(tenantID, queryHash, int64(searchReq.Start), int64(searchReq.End), m, startPage, pages)
		pipelineR.SetCacheKey(key)
		pipelineR.SetResponseData(shard)

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
