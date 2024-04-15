package frontend

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-kit/log" //nolint:all deprecated
	"github.com/gogo/protobuf/jsonpb"
	"github.com/grafana/dskit/user"
	"github.com/opentracing/opentracing-go"
	"github.com/segmentio/fasthash/fnv1a"

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
)

type SearchSharderConfig struct {
	ConcurrentRequests    int           `yaml:"concurrent_jobs,omitempty"`
	TargetBytesPerRequest int           `yaml:"target_bytes_per_job,omitempty"`
	DefaultLimit          uint32        `yaml:"default_result_limit"`
	MaxLimit              uint32        `yaml:"max_result_limit"`
	MaxDuration           time.Duration `yaml:"max_duration"`
	QueryBackendAfter     time.Duration `yaml:"query_backend_after,omitempty"`
	QueryIngestersUntil   time.Duration `yaml:"query_ingesters_until,omitempty"`
}

type backendReqMsg struct {
	req *http.Request
	err error
}

type asyncSearchSharder struct {
	next      pipeline.AsyncRoundTripper[*http.Response]
	reader    tempodb.Reader
	overrides overrides.Interface

	cfg    SearchSharderConfig
	logger log.Logger
}

// newAsyncSearchSharder creates a sharding middleware for search
func newAsyncSearchSharder(reader tempodb.Reader, o overrides.Interface, cfg SearchSharderConfig, logger log.Logger) pipeline.AsyncMiddleware[*http.Response] {
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

// RoundTrip implements http.RoundTripper
// execute up to concurrentRequests simultaneously where each request scans ~targetMBsPerRequest
// until limit results are found
func (s asyncSearchSharder) RoundTrip(r *http.Request) (pipeline.Responses[*http.Response], error) {
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
	span, ctx := opentracing.StartSpanFromContext(requestCtx, "frontend.ShardSearch")
	defer span.Finish()

	// calculate and enforce max search duration
	maxDuration := s.maxDuration(tenantID)
	if maxDuration != 0 && time.Duration(searchReq.End-searchReq.Start)*time.Second > maxDuration {
		return pipeline.NewBadRequest(fmt.Errorf("range specified by start and end exceeds %s. received start=%d end=%d", maxDuration, searchReq.Start, searchReq.End)), nil
	}

	// build request to search ingester based on query_ingesters_until config and time range
	// pass subCtx in requests so we can cancel and exit early
	ingesterReq, err := s.ingesterRequest(ctx, tenantID, r, *searchReq)
	if err != nil {
		return nil, err
	}

	reqCh := make(chan *http.Request, 2) // buffer of 2 allows us to insert ingestReq and metrics
	if ingesterReq != nil {
		reqCh <- ingesterReq
	}

	// pass subCtx in requests so we can cancel and exit early
	totalJobs, totalBlocks, totalBlockBytes := s.backendRequests(ctx, tenantID, r, searchReq, reqCh, func(err error) {
		// todo: actually find a way to return this error to the user
		s.logger.Log("msg", "search: failed to build backend requests", "err", err)
	})
	if ingesterReq != nil {
		totalJobs++
	}

	// send a job to communicate the search metrics. this is consumed by the combiner to calculate totalblocks/bytes/jobs
	var jobMetricsResponse pipeline.Responses[*http.Response]
	if totalJobs > 0 {
		resp := &tempopb.SearchResponse{
			Metrics: &tempopb.SearchMetrics{
				TotalBlocks:     uint32(totalBlocks),
				TotalBlockBytes: totalBlockBytes,
				TotalJobs:       uint32(totalJobs),
			},
		}

		m := jsonpb.Marshaler{}
		body, err := m.MarshalToString(resp)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal search metrics: %w", err)
		}

		jobMetricsResponse = pipeline.NewSuccessfulResponse(body)
	}

	// execute requests
	return pipeline.NewAsyncSharderChan(ctx, s.cfg.ConcurrentRequests, reqCh, jobMetricsResponse, s.next), nil
}

// blockMetas returns all relevant blockMetas given a start/end
func (s *asyncSearchSharder) blockMetas(start, end int64, tenantID string) []*backend.BlockMeta {
	// reduce metas to those in the requested range
	allMetas := s.reader.BlockMetas(tenantID)
	metas := make([]*backend.BlockMeta, 0, len(allMetas)/50) // divide by 50 for luck
	for _, m := range allMetas {
		if m.StartTime.Unix() <= end &&
			m.EndTime.Unix() >= start {
			metas = append(metas, m)
		}
	}

	return metas
}

// backendRequest builds backend requests to search backend blocks. backendRequest takes ownership of reqCh and closes it.
// it returns 3 int values: totalBlocks, totalBlockBytes, and estimated jobs
func (s *asyncSearchSharder) backendRequests(ctx context.Context, tenantID string, parent *http.Request, searchReq *tempopb.SearchRequest, reqCh chan<- *http.Request, errFn func(error)) (totalJobs, totalBlocks int, totalBlockBytes uint64) {
	var blocks []*backend.BlockMeta

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

	// get block metadata of blocks in start, end duration
	blocks = s.blockMetas(int64(start), int64(end), tenantID)

	targetBytesPerRequest := s.cfg.TargetBytesPerRequest

	// calculate metrics to return to the caller
	totalBlocks = len(blocks)
	for _, b := range blocks {
		p := pagesPerRequest(b, targetBytesPerRequest)

		totalJobs += int(b.TotalRecords) / p
		if int(b.TotalRecords)%p != 0 {
			totalJobs++
		}
		totalBlockBytes += b.Size
	}

	go func() {
		buildBackendRequests(ctx, tenantID, parent, searchReq, blocks, targetBytesPerRequest, reqCh, errFn)
	}()

	return
}

// ingesterRequest returns a new start and end time range for the backend as well as an http request
// that covers the ingesters. If nil is returned for the http.Request then there is no ingesters query.
// since this function modifies searchReq.Start and End we are taking a value instead of a pointer to prevent it from
// unexpectedly changing the passed searchReq.
func (s *asyncSearchSharder) ingesterRequest(ctx context.Context, tenantID string, parent *http.Request, searchReq tempopb.SearchRequest) (*http.Request, error) {
	// request without start or end, search only in ingester
	if searchReq.Start == 0 || searchReq.End == 0 {
		return buildIngesterRequest(ctx, tenantID, parent, &searchReq)
	}

	now := time.Now()
	ingesterUntil := uint32(now.Add(-s.cfg.QueryIngestersUntil).Unix())

	// if there's no overlap between the query and ingester range just return nil
	if searchReq.End < ingesterUntil {
		return nil, nil
	}

	ingesterStart := searchReq.Start
	ingesterEnd := searchReq.End

	// adjust ingesterStart if necessary
	if ingesterStart < ingesterUntil {
		ingesterStart = ingesterUntil
	}

	// if ingester start == ingester end then we don't need to query it
	if ingesterStart == ingesterEnd {
		return nil, nil
	}

	searchReq.Start = ingesterStart
	searchReq.End = ingesterEnd

	return buildIngesterRequest(ctx, tenantID, parent, &searchReq)
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
func buildBackendRequests(ctx context.Context, tenantID string, parent *http.Request, searchReq *tempopb.SearchRequest, metas []*backend.BlockMeta, bytesPerRequest int, reqCh chan<- *http.Request, errFn func(error)) {
	defer close(reqCh)

	queryHash := hashForSearchRequest(searchReq)

	for _, m := range metas {
		pages := pagesPerRequest(m, bytesPerRequest)
		if pages == 0 {
			continue
		}

		blockID := m.BlockID.String()
		for startPage := 0; startPage < int(m.TotalRecords); startPage += pages {
			subR := parent.Clone(ctx)

			dc, err := m.DedicatedColumns.ToTempopb()
			if err != nil {
				errFn(fmt.Errorf("failed to convert dedicated columns. block: %s tempopb: %w", blockID, err))
				continue
			}

			subR, err = api.BuildSearchBlockRequest(subR, &tempopb.SearchBlockRequest{
				BlockID:          blockID,
				StartPage:        uint32(startPage),
				PagesToSearch:    uint32(pages),
				Encoding:         m.Encoding.String(),
				IndexPageSize:    m.IndexPageSize,
				TotalRecords:     m.TotalRecords,
				DataEncoding:     m.DataEncoding,
				Version:          m.Version,
				Size_:            m.Size,
				FooterSize:       m.FooterSize,
				DedicatedColumns: dc,
			})
			if err != nil {
				errFn(fmt.Errorf("failed to build search block request. block: %s tempopb: %w", blockID, err))
				continue
			}

			prepareRequestForQueriers(subR, tenantID, subR.URL.Path, subR.URL.Query())
			key := searchJobCacheKey(tenantID, queryHash, int64(searchReq.Start), int64(searchReq.End), m, startPage, pages)
			if len(key) > 0 {
				subR = pipeline.AddCacheKey(key, subR)
			}

			select {
			case reqCh <- subR:
			case <-ctx.Done():
				// ignore the error if there is one. it will be handled elsewhere
				return
			}
		}
	}
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
	if m.Size == 0 || m.TotalRecords == 0 {
		return 0
	}

	bytesPerPage := m.Size / uint64(m.TotalRecords)
	if bytesPerPage == 0 {
		return 0
	}

	pagesPerQuery := bytesPerRequest / int(bytesPerPage)
	if pagesPerQuery == 0 {
		pagesPerQuery = 1 // have to have at least 1 page per query
	}

	return pagesPerQuery
}

func buildIngesterRequest(ctx context.Context, tenantID string, parent *http.Request, searchReq *tempopb.SearchRequest) (*http.Request, error) {
	subR := parent.Clone(ctx)
	subR, err := api.BuildSearchRequest(subR, searchReq)
	if err != nil {
		return nil, err
	}

	prepareRequestForQueriers(subR, tenantID, subR.URL.Path, subR.URL.Query())
	return subR, nil
}
