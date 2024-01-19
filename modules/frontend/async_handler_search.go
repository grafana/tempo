package frontend

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/go-kit/log" //nolint:all deprecated
	"github.com/grafana/dskit/user"
	"github.com/opentracing/opentracing-go"

	"github.com/grafana/tempo/modules/frontend/pipeline"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/tempodb"
	"github.com/grafana/tempo/tempodb/backend"
)

type asyncSearchSharder struct {
	next      pipeline.AsyncRoundTripper
	reader    tempodb.Reader
	overrides overrides.Interface
	cache     *frontendCache

	cfg    SearchSharderConfig
	logger log.Logger
}

// newSearchSharder creates a sharding middleware for search
func newAsyncSearchSharder(reader tempodb.Reader, o overrides.Interface, cfg SearchSharderConfig, cache *frontendCache, logger log.Logger) pipeline.AsyncMiddleware { // jpe standardize middleware names
	return pipeline.AsyncMiddlewareFunc(func(next pipeline.AsyncRoundTripper) pipeline.AsyncRoundTripper {
		return asyncSearchSharder{
			next:      next,
			reader:    reader,
			overrides: o,
			cache:     cache,

			cfg:    cfg,
			logger: logger,
		}
	})
}

// RoundTrip implements http.RoundTripper
// execute up to concurrentRequests simultaneously where each request scans ~targetMBsPerRequest
// until limit results are found
func (s asyncSearchSharder) RoundTrip(r *http.Request) (pipeline.Responses, error) {
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

	reqCh := make(chan *backendReqMsg, 1) // buffer of 1 allows us to insert ingestReq if it exists
	stopCh := make(chan struct{})
	defer close(stopCh)
	if ingesterReq != nil {
		reqCh <- &backendReqMsg{req: ingesterReq}
	}

	// pass subCtx in requests so we can cancel and exit early
	_, _, _ = s.backendRequests(ctx, tenantID, r, searchReq, reqCh, stopCh) // jpe need to go into the combiner somehow?

	// execute requests
	// progress := s.progress(ctx, int(searchReq.Limit), uint32(totalJobs), uint32(totalBlocks), totalBlockBytes) - jpe - losing blocks, jobs, etc. get this into the combiner?
	return pipeline.NewAsyncSharder(s.cfg.ConcurrentRequests, func(_ int) (*http.Request, *http.Response) {
		req, ok := <-reqCh
		if !ok {
			return nil, nil
		}

		// try cache
		bodyBuffer := s.cache.fetchBytes(req.cacheKey)
		if bodyBuffer != nil {
			return nil, &http.Response{
				StatusCode: http.StatusOK,
				Status:     http.StatusText(http.StatusOK),
				Body:       io.NopCloser(bytes.NewBuffer(bodyBuffer)),
			}
		}

		// jpe - cache is broken! needs to be its own pipeline item

		return req.req, nil
	}, s.next), nil
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
func (s *asyncSearchSharder) backendRequests(ctx context.Context, tenantID string, parent *http.Request, searchReq *tempopb.SearchRequest, reqCh chan<- *backendReqMsg, stopCh <-chan struct{}) (totalJobs, totalBlocks int, totalBlockBytes uint64) {
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
		buildBackendRequests(ctx, tenantID, parent, searchReq, blocks, targetBytesPerRequest, reqCh, stopCh)
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
