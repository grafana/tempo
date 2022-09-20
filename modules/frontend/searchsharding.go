package frontend

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/gogo/protobuf/jsonpb" //nolint:all deprecated
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/boundedwaitgroup"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/tempodb"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/opentracing/opentracing-go"
	"github.com/weaveworks/common/user"
)

const (
	defaultTargetBytesPerRequest = 10 * 1024 * 1024
	defaultConcurrentRequests    = 50
)

type searchSharder struct {
	next      http.RoundTripper
	reader    tempodb.Reader
	overrides *overrides.Overrides

	cfg    SearchSharderConfig
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

// newSearchSharder creates a sharding middleware for search
func newSearchSharder(reader tempodb.Reader, o *overrides.Overrides, cfg SearchSharderConfig, logger log.Logger) Middleware {
	return MiddlewareFunc(func(next http.RoundTripper) http.RoundTripper {
		return searchSharder{
			next:      next,
			reader:    reader,
			overrides: o,
			logger:    logger,
			cfg:       cfg,
		}
	})
}

// Roundtrip implements http.RoundTripper
// execute up to concurrentRequests simultaneously where each request scans ~targetMBsPerRequest
// until limit results are found
// keeping things simple. current query params are only:
// limit=<number>
// start=<unix epoch seconds>
// end=<unix epoch seconds>
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

	// calculate duration (start and end) to search the backend blocks
	start, end := s.backendRange(searchReq)

	// get block metadata of blocks in start, end duration
	blocks := s.blockMetas(int64(start), int64(end), tenantID)
	span.SetTag("block-count", len(blocks))

	var reqs []*http.Request
	// add backend requests if we need them
	if start != end {
		// pass subCtx in requests so we can cancel and exit early
		reqs, err = s.backendRequests(subCtx, tenantID, r, blocks)
		if err != nil {
			return nil, err
		}
	}
	// add ingester request if we have one. it's important to add the ingester request to
	// the beginning of the slice so that it is prioritized over the possibly enormous
	// number of backend requests
	if ingesterReq != nil {
		reqs = append([]*http.Request{ingesterReq}, reqs...)
	}
	span.SetTag("request-count", len(reqs))

	// execute requests
	wg := boundedwaitgroup.New(uint(s.cfg.ConcurrentRequests))
	overallResponse := newSearchResponse(ctx, int(searchReq.Limit), subCancel)
	overallResponse.resultsMetrics.InspectedBlocks = uint32(len(blocks))

	totalBlockBytes := uint64(0)
	for _, b := range blocks {
		totalBlockBytes += b.Size
	}
	overallResponse.resultsMetrics.TotalBlockBytes = totalBlockBytes

	executedReqs := 0
	for _, req := range reqs {
		// if shouldQuit is true, terminate and abandon requests
		if overallResponse.shouldQuit() {
			break
		}

		// When we hit capacity of boundedwaitgroup, wg.Add will block
		wg.Add(1)
		executedReqs++

		go func(innerR *http.Request) {
			defer wg.Done()

			resp, err := s.next.RoundTrip(innerR)
			if err != nil {
				// context cancelled error happens when we exit early.
				// bail, and don't log and don't set this error.
				if errors.Is(err, context.Canceled) {
					_ = level.Debug(s.logger).Log("msg", "exiting early from sharded query", "url", innerR.RequestURI, "err", err)
					return
				}

				_ = level.Error(s.logger).Log("msg", "error executing sharded query", "url", innerR.RequestURI, "err", err)
				overallResponse.setError(err)
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
				overallResponse.setStatus(statusCode, statusMsg)
				return
			}

			// successful query, read the body
			results := &tempopb.SearchResponse{}
			err = jsonpb.Unmarshal(resp.Body, results)
			if err != nil {
				_ = level.Error(s.logger).Log("msg", "error reading response body status == ok", "url", innerR.RequestURI, "err", err)
				overallResponse.setError(err)
				return
			}

			// happy path
			overallResponse.addResponse(results)
		}(req)
	}

	// wait for all goroutines running in wg to finish or cancelled
	wg.Wait()
	_ = level.Info(s.logger).Log(fmt.Sprintf("search requests total: %d, executed: %d", len(reqs), executedReqs))

	// all goroutines have finished, we can safely access searchResults fields directly now
	span.SetTag("inspectedBlocks", overallResponse.resultsMetrics.InspectedBlocks)
	span.SetTag("skippedBlocks", overallResponse.resultsMetrics.SkippedBlocks)
	span.SetTag("inspectedBytes", overallResponse.resultsMetrics.InspectedBytes)
	span.SetTag("inspectedTraces", overallResponse.resultsMetrics.InspectedTraces)
	span.SetTag("skippedTraces", overallResponse.resultsMetrics.SkippedTraces)
	span.SetTag("totalBlockBytes", overallResponse.resultsMetrics.TotalBlockBytes)

	// set total executed request count
	span.SetTag("executed-request-count", executedReqs)

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
	bodyString, err := m.MarshalToString(overallResponse.result())
	if err != nil {
		return nil, err
	}

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
	metas := []*backend.BlockMeta{}
	allMetas := s.reader.BlockMetas(tenantID)
	for _, m := range allMetas {
		if m.StartTime.Unix() <= end &&
			m.EndTime.Unix() >= start {
			metas = append(metas, m)
		}
	}

	return metas
}

// backendRequests returns a slice of requests that cover all blocks in the store
// that are covered by start/end.
func (s *searchSharder) backendRequests(ctx context.Context, tenantID string, parent *http.Request, metas []*backend.BlockMeta) ([]*http.Request, error) {
	reqs := []*http.Request{}
	for _, m := range metas {
		if m.Size == 0 || m.TotalRecords == 0 {
			continue
		}

		bytesPerPage := m.Size / uint64(m.TotalRecords)
		if bytesPerPage == 0 {
			return nil, fmt.Errorf("block %s has an invalid 0 bytes per page", m.BlockID)
		}
		pagesPerQuery := s.cfg.TargetBytesPerRequest / int(bytesPerPage)
		if pagesPerQuery == 0 {
			pagesPerQuery = 1 // have to have at least 1 page per query
		}

		blockID := m.BlockID.String()
		for startPage := 0; startPage < int(m.TotalRecords); startPage += pagesPerQuery {
			subR := parent.Clone(ctx)
			subR.Header.Set(user.OrgIDHeaderName, tenantID)

			subR, err := api.BuildSearchBlockRequest(subR, &tempopb.SearchBlockRequest{
				BlockID:       blockID,
				StartPage:     uint32(startPage),
				PagesToSearch: uint32(pagesPerQuery),
				Encoding:      m.Encoding.String(),
				IndexPageSize: m.IndexPageSize,
				TotalRecords:  m.TotalRecords,
				DataEncoding:  m.DataEncoding,
				Version:       m.Version,
				Size_:         m.Size,
				FooterSize:    m.FooterSize,
			})

			if err != nil {
				return nil, err
			}

			subR.RequestURI = buildUpstreamRequestURI(parent.URL.Path, subR.URL.Query())
			reqs = append(reqs, subR)
		}
	}

	return reqs, nil
}

// queryIngesterWithin returns a new start and end time range for the backend as well as an http request
// that covers the ingesters. If nil is returned for the http.Request then there is no ingesters query.
// since this function modifies searchReq.Start and End we are taking a value instead of a pointer to prevent it from
// unexpectedly changing the passed searchReq.
func (s *searchSharder) ingesterRequest(ctx context.Context, tenantID string, parent *http.Request, searchReq tempopb.SearchRequest) (*http.Request, error) {
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

	subR := parent.Clone(ctx)
	subR.Header.Set(user.OrgIDHeaderName, tenantID)

	searchReq.Start = ingesterStart
	searchReq.End = ingesterEnd
	subR, err := api.BuildSearchRequest(subR, &searchReq)
	if err != nil {
		return nil, err
	}
	subR.RequestURI = buildUpstreamRequestURI(parent.URL.Path, subR.URL.Query())

	return subR, nil
}

// backendRange returns a new start/end range for the backend based on the config parameter
// query_backend_after. If the returned start == the returned end then backend querying is not necessary.
func (s *searchSharder) backendRange(searchReq *tempopb.SearchRequest) (uint32, uint32) {
	now := time.Now()
	backendAfter := uint32(now.Add(-s.cfg.QueryBackendAfter).Unix())

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

// maxDuration returns the max search duration allowed for this tenant.
func (s *searchSharder) maxDuration(tenantID string) time.Duration {
	// check overrides first, if no overrides then grab from our config
	maxDuration := s.overrides.MaxSearchDuration(tenantID)
	if maxDuration != 0 {
		return maxDuration
	}

	return s.cfg.MaxDuration
}
