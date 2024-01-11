package frontend

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/grafana/dskit/user"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/boundedwaitgroup"
	"github.com/grafana/tempo/tempodb"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/opentracing/opentracing-go"
)

type tagResultsHandler interface {
	shouldQuit() bool
	addResponse(io.ReadCloser) error
	marshalResult() (string, error)
}

type (
	tagResultHandlerFactory func(limit int) tagResultsHandler
	parseRequestFunction    func(r *http.Request) (tagSearchReq, error)
	tagSearchReq            interface {
		start() uint32
		end() uint32
		newWithRange(start, end uint32) tagSearchReq
		buildSearchTagRequest(subR *http.Request) (*http.Request, error)
		buildTagSearchBlockRequest(*http.Request, string, int, int, *backend.BlockMeta) (*http.Request, error)
	}
)

type tagResultCollector struct {
	delegate   tagResultsHandler
	mtx        sync.Mutex
	err        error
	statusCode int
	statusMsg  string
	ctx        context.Context
}

type searchTagSharder struct {
	next                   http.RoundTripper
	reader                 tempodb.Reader
	overrides              overrides.Interface
	tagShardHandlerFactory tagResultHandlerFactory
	cfg                    SearchSharderConfig
	logger                 log.Logger
	parseRequest           parseRequestFunction
}

type tagResults struct {
	response    string
	statusCode  int
	statusMsg   string
	err         error
	marshallErr error
}

func (r *tagResultCollector) shouldQuit() bool {
	r.mtx.Lock()
	defer r.mtx.Unlock()
	if r.err != nil {
		return true
	}
	if r.ctx.Err() != nil {
		return true
	}
	if r.statusCode/100 != 2 {
		return true
	}
	if r.delegate.shouldQuit() {
		return true
	}
	return false
}

func (r *tagResultCollector) setStatus(statusCode int, statusMsg string) {
	r.mtx.Lock()
	defer r.mtx.Unlock()

	r.statusCode = statusCode
	r.statusMsg = statusMsg
}

func (r *tagResultCollector) setError(err error) {
	r.mtx.Lock()
	defer r.mtx.Unlock()

	r.err = err
}

func (r *tagResultCollector) addResponseToResult(ic io.ReadCloser) error {
	r.mtx.Lock()
	defer r.mtx.Unlock()
	return r.delegate.addResponse(ic)
}

func (r *tagResultCollector) Result() *tagResults {
	r.mtx.Lock()
	defer r.mtx.Unlock()

	response, err := r.delegate.marshalResult()
	return &tagResults{
		statusCode:  r.statusCode,
		statusMsg:   r.statusMsg,
		err:         r.err,
		response:    response,
		marshallErr: err,
	}
}

func newTagResultCollector(ctx context.Context, factory tagResultHandlerFactory, limit int) *tagResultCollector {
	return &tagResultCollector{
		statusCode: http.StatusOK,
		ctx:        ctx,
		delegate:   factory(limit),
	}
}

func (s searchTagSharder) httpErrorResponse(err error) *http.Response {
	return &http.Response{
		StatusCode: http.StatusBadRequest,
		Body:       io.NopCloser(strings.NewReader(err.Error())),
	}
}

// RoundTrip implements http.RoundTripper
// execute up to concurrentRequests simultaneously where each request scans ~targetMBsPerRequest
// until limit results are found
func (s searchTagSharder) RoundTrip(r *http.Request) (*http.Response, error) {
	requestCtx := r.Context()

	tenantID, err := user.ExtractOrgID(requestCtx)
	if err != nil {
		return s.httpErrorResponse(err), nil
	}

	// TODO: Need to review this and only applies to tag values and no tag names, also consolidate the logic in one single palce
	handler := newTagResultCollector(requestCtx, s.tagShardHandlerFactory, s.overrides.MaxBytesPerTagValuesQuery(tenantID))

	searchReq, err := s.parseRequest(r)
	if err != nil {
		return s.httpErrorResponse(err), nil
	}
	span, ctx := opentracing.StartSpanFromContext(requestCtx, "frontend.ShardSearch")
	defer span.Finish()

	// sub context to cancel in-progress sub requests
	subCtx, subCancel := context.WithCancel(ctx)
	defer subCancel()

	// calculate and enforce max search duration
	maxDuration := s.maxDuration(tenantID)
	if maxDuration != 0 && time.Duration(searchReq.end()-searchReq.start())*time.Second > maxDuration {
		return s.httpErrorResponse(fmt.Errorf("range specified by start and end exceeds %s."+
			" received start=%d end=%d", maxDuration, searchReq.start(), searchReq.end())), nil
	}

	// build request to search ingester based on query_ingesters_until config and time range
	// pass subCtx in requests, so we can cancel and exit early
	ingesterReq, err := s.ingesterRequest(subCtx, tenantID, r, searchReq)
	if err != nil {
		return nil, err
	}

	reqCh := make(chan *backendReqMsg, 1) // buffer of 1 allows us to insert ingestReq if it exists

	stopCh := make(chan struct{})
	defer close(stopCh)

	if ingesterReq != nil {
		reqCh <- &backendReqMsg{req: ingesterReq}
	}
	// TODO: Needs to be reviewed how to shard this property, as this is not very accurate and needs more testing.
	// 		 even for regular search this is not precise.
	// pass subCtx in requests, so we can cancel and exit early
	s.backendRequests(subCtx, tenantID, r, searchReq, reqCh, stopCh)

	// execute requests
	wg := boundedwaitgroup.New(uint(s.cfg.ConcurrentRequests))

	for req := range reqCh {
		if req.err != nil {
			return nil, fmt.Errorf("unexpected err building reqs: %w", req.err)
		}
		// When we hit capacity of boundedwaitgroup, wg.Add will block
		wg.Add(1)

		go func(innerR *http.Request) {
			defer func() {
				if handler.shouldQuit() {
					subCancel()
				}
				wg.Done()
			}()

			resp, err := s.next.RoundTrip(innerR)
			if err != nil {
				// context cancelled error happens when we exit early. bail, and don't log and don't set this error.
				if errors.Is(err, context.Canceled) {
					_ = level.Debug(s.logger).Log("msg", "exiting early from sharded query", "url",
						innerR.RequestURI, "err", err)
					return
				}
				_ = level.Error(s.logger).Log("msg", "error executing sharded query", "url",
					innerR.RequestURI, "err", err)
				handler.setError(err)
				return
			}

			// if the status code is anything but happy, save the error and pass it down the line
			if resp.StatusCode != http.StatusOK {
				statusCode := resp.StatusCode
				bytesMsg, err := io.ReadAll(resp.Body)
				if err != nil {
					_ = level.Error(s.logger).Log("msg", "error reading response body status != ok", "url",
						innerR.RequestURI, "err", err)
				}
				statusMsg := fmt.Sprintf("upstream: (%d) %s", statusCode, string(bytesMsg))
				handler.setStatus(statusCode, statusMsg)
				return
			}

			err = handler.addResponseToResult(resp.Body)

			if err != nil {
				_ = level.Error(s.logger).Log("msg", "error reading response body status == ok", "url",
					innerR.RequestURI, "err", err)
				handler.setError(err)
				return
			}
		}(req.req)
	}

	// wait for all goroutines running in wg to finish or cancelled
	wg.Wait()

	overallResponse := handler.Result()

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

	if overallResponse.marshallErr != nil {
		return nil, err
	}

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header: http.Header{
			api.HeaderContentType: {api.HeaderAcceptJSON},
		},
		Body:          io.NopCloser(strings.NewReader(overallResponse.response)),
		ContentLength: int64(len([]byte(overallResponse.response))),
	}

	return resp, nil
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
func (s searchTagSharder) backendRequests(ctx context.Context, tenantID string, parent *http.Request, searchReq tagSearchReq, reqCh chan<- *backendReqMsg, stopCh <-chan struct{}) {
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
		s.buildBackendRequests(ctx, tenantID, parent, blocks, targetBytesPerRequest, reqCh, stopCh, searchReq)
	}()
}

// buildBackendRequests returns a slice of requests that cover all blocks in the store
// that are covered by start/end.
func (s searchTagSharder) buildBackendRequests(ctx context.Context, tenantID string, parent *http.Request, metas []*backend.BlockMeta, bytesPerRequest int, reqCh chan<- *backendReqMsg, stopCh <-chan struct{}, searchReq tagSearchReq) {
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

// newTagsSharding creates a sharding middleware for search
func newTagsSharding(
	reader tempodb.Reader, o overrides.Interface,
	cfg SearchSharderConfig, tagShardHandler tagResultHandlerFactory, logger log.Logger,
	parseRequest parseRequestFunction,
) Middleware {
	return MiddlewareFunc(func(next http.RoundTripper) http.RoundTripper {
		return searchTagSharder{
			next:                   next,
			reader:                 reader,
			overrides:              o,
			cfg:                    cfg,
			logger:                 logger,
			tagShardHandlerFactory: tagShardHandler,
			parseRequest:           parseRequest,
		}
	})
}
