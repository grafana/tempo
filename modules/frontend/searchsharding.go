package frontend

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/gogo/protobuf/jsonpb" //nolint:all deprecated
	"github.com/grafana/dskit/user"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/boundedwaitgroup"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/tempodb"
	"github.com/grafana/tempo/tempodb/backend"
)

const (
	defaultTargetBytesPerRequest = 100 * 1024 * 1024
	defaultConcurrentRequests    = 1000
)

var (
	queryThroughput = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "tempo",
		Name:      "query_frontend_bytes_processed_per_second",
		Help:      "Bytes processed per second in the query per tenant",
		Buckets:   prometheus.ExponentialBuckets(1024*1024, 2, 10), // from 1MB up to 1GB
	}, []string{"tenant", "op"})

	searchThroughput = queryThroughput.MustCurryWith(prometheus.Labels{"op": searchOp})

	sloQueriesPerTenant = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "query_frontend_queries_within_slo_total",
		Help:      "Total Queries within SLO per tenant",
	}, []string{"tenant", "op"})

	sloTraceByIDCounter = sloQueriesPerTenant.MustCurryWith(prometheus.Labels{"op": traceByIDOp})
	sloSearchCounter    = sloQueriesPerTenant.MustCurryWith(prometheus.Labels{"op": searchOp})
)

type searchSharder struct {
	next      http.RoundTripper
	reader    tempodb.Reader
	overrides overrides.Interface
	progress  searchProgressFactory

	cfg    SearchSharderConfig
	sloCfg SLOConfig
	logger log.Logger
}

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

// newSearchSharder creates a sharding middleware for search
func newSearchSharder(reader tempodb.Reader, o overrides.Interface, cfg SearchSharderConfig, sloCfg SLOConfig, progress searchProgressFactory, logger log.Logger) Middleware {
	return MiddlewareFunc(func(next http.RoundTripper) http.RoundTripper {
		return searchSharder{
			next:      next,
			reader:    reader,
			overrides: o,
			cfg:       cfg,
			sloCfg:    sloCfg,
			logger:    logger,

			progress: progress,
		}
	})
}

// RoundTrip implements http.RoundTripper
// execute up to concurrentRequests simultaneously where each request scans ~targetMBsPerRequest
// until limit results are found
func (s searchSharder) RoundTrip(r *http.Request) (*http.Response, error) {
	searchReq, err := api.ParseSearchRequest(r)
	if err != nil {
		return &http.Response{
			StatusCode: http.StatusBadRequest,
			Body:       io.NopCloser(strings.NewReader(err.Error())),
		}, nil
	}

	// adjust limit based on config
	searchReq.Limit = adjustLimit(searchReq.Limit, s.cfg.DefaultLimit, s.cfg.MaxLimit)

	ctx := r.Context()
	tenantID, err := user.ExtractOrgID(ctx)
	if err != nil {
		return &http.Response{
			StatusCode: http.StatusBadRequest,
			Body:       io.NopCloser(strings.NewReader(err.Error())),
		}, nil
	}
	span, ctx := opentracing.StartSpanFromContext(ctx, "frontend.ShardSearch")
	defer span.Finish()

	reqStart := time.Now()
	// sub context to cancel in-progress sub requests
	subCtx, subCancel := context.WithCancel(ctx)
	defer subCancel()

	// calculate and enforce max search duration
	maxDuration := s.maxDuration(tenantID)
	if maxDuration != 0 && time.Duration(searchReq.End-searchReq.Start)*time.Second > maxDuration {
		return &http.Response{
			StatusCode: http.StatusBadRequest,
			Body:       io.NopCloser(strings.NewReader(fmt.Sprintf("range specified by start and end exceeds %s. received start=%d end=%d", maxDuration, searchReq.Start, searchReq.End))),
		}, nil
	}

	// build request to search ingester based on query_ingesters_until config and time range
	// pass subCtx in requests so we can cancel and exit early
	ingesterReq, err := s.ingesterRequest(subCtx, tenantID, r, *searchReq)
	if err != nil {
		return nil, err
	}

	reqCh := make(chan *backendReqMsg, 1) // buffer of 1 allows us to insert ingestReq if it exists
	stopCh := make(chan struct{})
	defer close(stopCh)
	if ingesterReq != nil {
		reqCh <- &backendReqMsg{req: ingesterReq}
	}

	// pass subCtx in requests so we can cancel and exit early
	totalJobs, totalBlocks, totalBlockBytes := s.backendRequests(subCtx, tenantID, r, searchReq, reqCh, stopCh)
	if ingesterReq != nil {
		totalJobs++
	}

	// execute requests
	wg := boundedwaitgroup.New(uint(s.cfg.ConcurrentRequests))
	progress := s.progress(ctx, int(searchReq.Limit), totalJobs, totalBlocks, totalBlockBytes)

	startedReqs := 0
	for req := range reqCh {
		if req.err != nil {
			return nil, fmt.Errorf("unexpected err building reqs: %w", req.err)
		}

		// if shouldQuit is true, terminate and abandon requests
		if progress.shouldQuit() {
			break
		}

		// When we hit capacity of boundedwaitgroup, wg.Add will block
		wg.Add(1)
		startedReqs++

		go func(innerR *http.Request) {
			defer func() {
				if progress.shouldQuit() {
					subCancel()
				}
				wg.Done()
			}()

			resp, err := s.next.RoundTrip(innerR)
			if err != nil {
				// context cancelled error happens when we exit early.
				// bail, and don't log and don't set this error.
				if errors.Is(err, context.Canceled) {
					_ = level.Debug(s.logger).Log("msg", "exiting early from sharded query", "url", innerR.RequestURI, "err", err)
					return
				}

				_ = level.Error(s.logger).Log("msg", "error executing sharded query", "url", innerR.RequestURI, "err", err)
				progress.setError(err)
				return
			}

			// if the status code is anything but happy, save the error and pass it down the line
			if resp.StatusCode != http.StatusOK {
				statusCode := resp.StatusCode
				bytesMsg, err := io.ReadAll(resp.Body)
				if err != nil {
					_ = level.Error(s.logger).Log("msg", "error reading response body status != ok", "url", innerR.RequestURI, "err", err)
				}
				statusMsg := fmt.Sprintf("upstream: (%d) %s", statusCode, string(bytesMsg))
				progress.setStatus(statusCode, statusMsg)
				return
			}

			// successful query, read the body
			results := &tempopb.SearchResponse{}
			err = (&jsonpb.Unmarshaler{AllowUnknownFields: true}).Unmarshal(resp.Body, results)
			if err != nil {
				_ = level.Error(s.logger).Log("msg", "error reading response body status == ok", "url", innerR.RequestURI, "err", err)
				progress.setError(err)
				return
			}

			// happy path
			progress.addResponse(results)
		}(req.req)
	}

	// wait for all goroutines running in wg to finish or cancelled
	wg.Wait()

	// print out request metrics
	overallResponse := progress.result()

	cancelledReqs := startedReqs - overallResponse.finishedRequests
	reqTime := time.Since(reqStart)
	throughput := float64(overallResponse.response.Metrics.InspectedBytes) / reqTime.Seconds()
	searchThroughput.WithLabelValues(tenantID).Observe(throughput)

	query, _ := url.PathUnescape(r.URL.RawQuery)
	span.SetTag("query", query)
	level.Info(s.logger).Log(
		"msg", "sharded search query request stats and SearchMetrics",
		"query", query,
		"duration_seconds", reqTime,
		"request_throughput", throughput,
		"total_requests", totalJobs,
		"started_requests", startedReqs,
		"cancelled_requests", cancelledReqs,
		"finished_requests", overallResponse.finishedRequests,
		"totalBlocks", overallResponse.response.Metrics.TotalBlocks,
		"inspectedBytes", overallResponse.response.Metrics.InspectedBytes,
		"inspectedTraces", overallResponse.response.Metrics.InspectedTraces,
		"totalBlockBytes", overallResponse.response.Metrics.TotalBlockBytes)

	// all goroutines have finished, we can safely access searchResults fields directly now
	span.SetTag("totalBlocks", overallResponse.response.Metrics.TotalBlocks)
	span.SetTag("inspectedBytes", overallResponse.response.Metrics.InspectedBytes)
	span.SetTag("inspectedTraces", overallResponse.response.Metrics.InspectedTraces)
	span.SetTag("totalBlockBytes", overallResponse.response.Metrics.TotalBlockBytes)
	span.SetTag("totalJobs", totalJobs)
	span.SetTag("finishedJobs", overallResponse.finishedRequests)
	span.SetTag("requestThroughput", throughput)

	if overallResponse.err != nil {
		return nil, overallResponse.err
	}

	if overallResponse.statusCode != http.StatusOK {
		// translate all non-200s into 500s. if, for instance, we get a 400 back from an internal component
		// it means that we created a bad request. 400 should not be propagated back to the user b/c
		// the bad request was due to a bug on our side, so return 500 instead.
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
			Header:     http.Header{},
			Body:       io.NopCloser(strings.NewReader(overallResponse.statusMsg)),
		}, nil
	}

	m := &jsonpb.Marshaler{}
	bodyString, err := m.MarshalToString(overallResponse.response)
	if err != nil {
		return nil, err
	}

	recordSearchSLO(s.sloCfg, tenantID, reqTime, throughput)

	return &http.Response{
		StatusCode: http.StatusOK,
		Header: http.Header{
			api.HeaderContentType: {api.HeaderAcceptJSON},
		},
		Body:          io.NopCloser(strings.NewReader(bodyString)),
		ContentLength: int64(len([]byte(bodyString))),
	}, nil
}

// blockMetas returns all relevant blockMetas given a start/end
func (s *searchSharder) blockMetas(start, end int64, tenantID string) []*backend.BlockMeta {
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
func (s *searchSharder) backendRequests(ctx context.Context, tenantID string, parent *http.Request, searchReq *tempopb.SearchRequest, reqCh chan<- *backendReqMsg, stopCh <-chan struct{}) (totalJobs, totalBlocks int, totalBlockBytes uint64) {
	var blocks []*backend.BlockMeta

	// request without start or end, search only in ingester
	if searchReq.Start == 0 || searchReq.End == 0 {
		close(reqCh)
		return
	}

	// calculate duration (start and end) to search the backend blocks
	start, end := backendRange(searchReq, s.cfg.QueryBackendAfter)

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
		buildBackendRequests(ctx, tenantID, parent, blocks, targetBytesPerRequest, reqCh, stopCh)
	}()

	return
}

// backendRange returns a new start/end range for the backend based on the config parameter
// query_backend_after. If the returned start == the returned end then backend querying is not necessary.
func backendRange(searchReq *tempopb.SearchRequest, queryBackendAfter time.Duration) (uint32, uint32) {
	now := time.Now()
	backendAfter := uint32(now.Add(-queryBackendAfter).Unix())

	start := searchReq.Start
	end := searchReq.End

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
func buildBackendRequests(ctx context.Context, tenantID string, parent *http.Request, metas []*backend.BlockMeta, bytesPerRequest int, reqCh chan<- *backendReqMsg, stopCh <-chan struct{}) {
	defer close(reqCh)

	for _, m := range metas {
		pages := pagesPerRequest(m, bytesPerRequest)
		if pages == 0 {
			continue
		}

		blockID := m.BlockID.String()
		for startPage := 0; startPage < int(m.TotalRecords); startPage += pages {
			subR := parent.Clone(ctx)
			subR.Header.Set(user.OrgIDHeaderName, tenantID)

			dc, err := m.DedicatedColumns.ToTempopb()
			if err != nil {
				reqCh <- &backendReqMsg{err: err}
				return
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
				reqCh <- &backendReqMsg{err: err}
				return
			}

			subR.RequestURI = buildUpstreamRequestURI(parent.URL.Path, subR.URL.Query())

			select {
			case reqCh <- &backendReqMsg{req: subR}:
			case <-stopCh:
				return
			}
		}
	}
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

// ingesterRequest returns a new start and end time range for the backend as well as an http request
// that covers the ingesters. If nil is returned for the http.Request then there is no ingesters query.
// since this function modifies searchReq.Start and End we are taking a value instead of a pointer to prevent it from
// unexpectedly changing the passed searchReq.
func (s *searchSharder) ingesterRequest(ctx context.Context, tenantID string, parent *http.Request, searchReq tempopb.SearchRequest) (*http.Request, error) {
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

func buildIngesterRequest(ctx context.Context, tenantID string, parent *http.Request, searchReq *tempopb.SearchRequest) (*http.Request, error) {
	subR := parent.Clone(ctx)

	subR.Header.Set(user.OrgIDHeaderName, tenantID)
	subR, err := api.BuildSearchRequest(subR, searchReq)
	if err != nil {
		return nil, err
	}

	subR.RequestURI = buildUpstreamRequestURI(subR.URL.Path, subR.URL.Query())
	return subR, nil
}

// adjusts the limit based on provided config
func adjustLimit(limit, defaultLimit, maxLimit uint32) uint32 {
	if limit == 0 {
		return defaultLimit
	}

	if maxLimit != 0 && limit > maxLimit {
		return maxLimit
	}

	return limit
}

func recordSearchSLO(sloCfg SLOConfig, tenantID string, reqTime time.Duration, throughput float64) {
	// only capture if SLOConfig is set
	if sloCfg.DurationSLO != 0 && sloCfg.ThroughputBytesSLO != 0 {
		if reqTime < sloCfg.DurationSLO || throughput > sloCfg.ThroughputBytesSLO {
			// query is within SLO if query returned 200 within DurationSLO seconds OR
			// processed ThroughputBytesSLO bytes/s data
			sloSearchCounter.WithLabelValues(tenantID).Inc()
		}
	}
}

// maxDuration returns the max search duration allowed for this tenant.
func (s *searchSharder) maxDuration(tenantID string) time.Duration {
	// check overrides first, if no overrides then grab from our config
	maxDuration := s.overrides.MaxSearchDuration(tenantID)
	if maxDuration != 0 {
		return maxDuration
	}

	return s.cfg.MaxDuration
}
