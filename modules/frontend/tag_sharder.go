package frontend

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-kit/log"
	"github.com/grafana/dskit/user"
	"github.com/grafana/tempo/modules/frontend/pipeline"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/tempodb"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/opentracing/opentracing-go"
)

/* tagsSearchRequest request interface for transform tags and tags V2 requests into a querier request */
type tagsSearchRequest struct {
	request tempopb.SearchTagsRequest
}

func (r *tagsSearchRequest) start() uint32 {
	return r.request.Start
}

func (r *tagsSearchRequest) end() uint32 {
	return r.request.End
}

func (r *tagsSearchRequest) newWithRange(start, end uint32) tagSearchReq {
	newReq := r.request
	newReq.Start = start
	newReq.End = end

	return &tagsSearchRequest{
		request: newReq,
	}
}

func (r *tagsSearchRequest) buildSearchTagRequest(subR *http.Request) (*http.Request, error) {
	return api.BuildSearchTagsRequest(subR, &r.request)
}

func (r *tagsSearchRequest) buildTagSearchBlockRequest(subR *http.Request, blockID string,
	startPage int, pages int, m *backend.BlockMeta,
) (*http.Request, error) {
	return api.BuildSearchTagsBlockRequest(subR, &tempopb.SearchTagsBlockRequest{
		BlockID:       blockID,
		StartPage:     uint32(startPage),
		PagesToSearch: uint32(pages),
		Encoding:      m.Encoding.String(),
		IndexPageSize: m.IndexPageSize,
		TotalRecords:  m.TotalRecords,
		DataEncoding:  m.DataEncoding,
		Version:       m.Version,
		Size_:         m.Size,
		FooterSize:    m.FooterSize,
	})
}

/* TagValue V2 handler and request implementation */
type tagValueSearchRequest struct {
	request tempopb.SearchTagValuesRequest
}

func (r *tagValueSearchRequest) start() uint32 {
	return r.request.Start
}

func (r *tagValueSearchRequest) end() uint32 {
	return r.request.End
}

func (r *tagValueSearchRequest) newWithRange(start, end uint32) tagSearchReq {
	newReq := r.request
	newReq.Start = start
	newReq.End = end

	return &tagValueSearchRequest{
		request: newReq,
	}
}

/*
  jpe - add cache key?
	  - add gprc endpoints to frontend
	  - e2e tests for :point-up:
*/

func (r *tagValueSearchRequest) buildSearchTagRequest(subR *http.Request) (*http.Request, error) {
	return api.BuildSearchTagValuesRequest(subR, &r.request)
}

func (r *tagValueSearchRequest) buildTagSearchBlockRequest(subR *http.Request, blockID string,
	startPage int, pages int, m *backend.BlockMeta,
) (*http.Request, error) {
	return api.BuildSearchTagValuesBlockRequest(subR, &tempopb.SearchTagValuesBlockRequest{
		BlockID:       blockID,
		StartPage:     uint32(startPage),
		PagesToSearch: uint32(pages),
		Encoding:      m.Encoding.String(),
		IndexPageSize: m.IndexPageSize,
		TotalRecords:  m.TotalRecords,
		DataEncoding:  m.DataEncoding,
		Version:       m.Version,
		Size_:         m.Size,
		FooterSize:    m.FooterSize,
	})
}

func parseTagsRequest(r *http.Request) (tagSearchReq, error) {
	searchReq, err := api.ParseSearchTagsRequest(r)
	if err != nil {
		return nil, err
	}
	return &tagsSearchRequest{
		request: *searchReq,
	}, nil
}

func parseTagValuesRequest(r *http.Request) (tagSearchReq, error) {
	searchReq, err := api.ParseSearchTagValuesRequest(r)
	if err != nil {
		return nil, err
	}
	return &tagValueSearchRequest{
		request: *searchReq,
	}, nil
}

type parseRequestFunction func(r *http.Request) (tagSearchReq, error)

type tagSearchReq interface {
	start() uint32
	end() uint32
	newWithRange(start, end uint32) tagSearchReq
	buildSearchTagRequest(subR *http.Request) (*http.Request, error)
	buildTagSearchBlockRequest(*http.Request, string, int, int, *backend.BlockMeta) (*http.Request, error)
}

type searchTagSharder struct {
	next      pipeline.AsyncRoundTripper[*http.Response]
	reader    tempodb.Reader
	overrides overrides.Interface

	cfg          SearchSharderConfig
	logger       log.Logger
	parseRequest parseRequestFunction
}

// newAsyncTagSharder creates a sharding middleware for tags and tag values
func newAsyncTagSharder(reader tempodb.Reader, o overrides.Interface, cfg SearchSharderConfig, parseRequest parseRequestFunction, logger log.Logger) pipeline.AsyncMiddleware[*http.Response] {
	return pipeline.AsyncMiddlewareFunc[*http.Response](func(next pipeline.AsyncRoundTripper[*http.Response]) pipeline.AsyncRoundTripper[*http.Response] {
		return searchTagSharder{
			next:         next,
			reader:       reader,
			overrides:    o,
			cfg:          cfg,
			logger:       logger,
			parseRequest: parseRequest,
		}
	})
}

// RoundTrip implements pipeline.AsyncRoundTripper
// execute up to concurrentRequests simultaneously where each request scans ~targetMBsPerRequest
// until limit results are found
func (s searchTagSharder) RoundTrip(r *http.Request) (pipeline.Responses[*http.Response], error) {
	requestCtx := r.Context()

	tenantID, err := user.ExtractOrgID(requestCtx)
	if err != nil {
		return pipeline.NewBadRequest(err), nil
	}

	searchReq, err := s.parseRequest(r)
	if err != nil {
		return pipeline.NewBadRequest(err), nil
	}
	span, ctx := opentracing.StartSpanFromContext(requestCtx, "frontend.ShardSearchTags")
	defer span.Finish()

	// calculate and enforce max search duration
	maxDuration := s.maxDuration(tenantID)
	if maxDuration != 0 && time.Duration(searchReq.end()-searchReq.start())*time.Second > maxDuration {
		return pipeline.NewBadRequest(fmt.Errorf("range specified by start and end exceeds %s."+
			" received start=%d end=%d", maxDuration, searchReq.start(), searchReq.end())), nil
	}

	// build request to search ingester based on query_ingesters_until config and time range
	// pass subCtx in requests, so we can cancel and exit early
	ingesterReq, err := s.ingesterRequest(ctx, tenantID, r, searchReq)
	if err != nil {
		return nil, err
	}

	reqCh := make(chan *http.Request, 1) // buffer of 1 allows us to insert ingestReq if it exists
	if ingesterReq != nil {
		reqCh <- ingesterReq
	}

	s.backendRequests(ctx, tenantID, r, searchReq, reqCh, func(err error) {
		// todo: actually find a way to return this error to the user
		s.logger.Log("msg", "failed to build backend requests", "err", err)
	})

	// execute requests
	return pipeline.NewAsyncSharderChan(s.cfg.ConcurrentRequests, reqCh, nil, s.next), nil
}

// blockMetas returns all relevant blockMetas given a start/end
func (s searchTagSharder) blockMetas(start, end int64, tenantID string) []*backend.BlockMeta {
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
func (s searchTagSharder) backendRequests(ctx context.Context, tenantID string, parent *http.Request, searchReq tagSearchReq, reqCh chan<- *http.Request, errFn func(error)) {
	var blocks []*backend.BlockMeta

	// request without start or end, search only in ingester
	if searchReq.start() == 0 || searchReq.end() == 0 {
		close(reqCh)
		return
	}

	// calculate duration (start and end) to search the backend blocks
	start, end := backendRange(searchReq.start(), searchReq.end(), s.cfg.QueryBackendAfter)

	// no need to search backend
	if start == end {
		close(reqCh)
		return
	}

	// get block metadata of blocks in start, end duration
	blocks = s.blockMetas(int64(start), int64(end), tenantID)

	targetBytesPerRequest := s.cfg.TargetBytesPerRequest

	go func() {
		s.buildBackendRequests(ctx, tenantID, parent, blocks, targetBytesPerRequest, reqCh, errFn, searchReq)
	}()
}

// buildBackendRequests returns a slice of requests that cover all blocks in the store
// that are covered by start/end.
func (s searchTagSharder) buildBackendRequests(ctx context.Context, tenantID string, parent *http.Request, metas []*backend.BlockMeta, bytesPerRequest int, reqCh chan<- *http.Request, errFn func(error), searchReq tagSearchReq) {
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
			subR, err := searchReq.buildTagSearchBlockRequest(subR, blockID, startPage, pages, m)
			if err != nil {
				errFn(err)
				return
			}
			subR.RequestURI = buildUpstreamRequestURI(parent.URL.Path, subR.URL.Query())
			select {
			case reqCh <- subR:
			case <-ctx.Done():
				return
			}
		}
	}
}

// ingesterRequest returns a new start and end time range for the backend as well as a http request
// that covers the ingesters. If nil is returned for the http.Request then there is no ingesters query.
// we should do a copy of the searchReq before use this function, as it is an interface, we cannot guaranteed  be passed
// by value.
func (s searchTagSharder) ingesterRequest(ctx context.Context, tenantID string, parent *http.Request, searchReq tagSearchReq) (*http.Request, error) {
	// request without start or end, search only in ingester
	if searchReq.start() == 0 || searchReq.end() == 0 {
		return s.buildIngesterRequest(ctx, tenantID, parent, searchReq)
	}

	now := time.Now()
	ingesterUntil := uint32(now.Add(-s.cfg.QueryIngestersUntil).Unix())

	// if there's no overlap between the query and ingester range just return nil
	if searchReq.end() < ingesterUntil {
		return nil, nil
	}

	ingesterStart := searchReq.start()
	ingesterEnd := searchReq.end()

	// adjust ingesterStart if necessary
	if ingesterStart < ingesterUntil {
		ingesterStart = ingesterUntil
	}

	// if ingester start == ingester end then we don't need to query it
	if ingesterStart == ingesterEnd {
		return nil, nil
	}

	newSearchReq := searchReq.newWithRange(ingesterStart, ingesterEnd)
	return s.buildIngesterRequest(ctx, tenantID, parent, newSearchReq)
}

func (s searchTagSharder) buildIngesterRequest(ctx context.Context, tenantID string, parent *http.Request, searchReq tagSearchReq) (*http.Request, error) {
	subR := parent.Clone(ctx)

	subR.Header.Set(user.OrgIDHeaderName, tenantID)
	subR, err := searchReq.buildSearchTagRequest(subR)
	if err != nil {
		return nil, err
	}

	subR.RequestURI = buildUpstreamRequestURI(subR.URL.Path, subR.URL.Query())
	return subR, nil
}

// maxDuration returns the max search duration allowed for this tenant.
func (s searchTagSharder) maxDuration(tenantID string) time.Duration {
	// check overrides first, if no overrides then grab from our config
	maxDuration := s.overrides.MaxSearchDuration(tenantID)
	if maxDuration != 0 {
		return maxDuration
	}

	return s.cfg.MaxDuration
}
