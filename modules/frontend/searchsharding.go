package frontend

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/log/level"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/grafana/tempo/modules/querier"
	"github.com/grafana/tempo/pkg/boundedwaitgroup"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/tempodb"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/opentracing/opentracing-go"
	"github.com/weaveworks/common/user"
)

const (
	// todo(search): make configurable
	maxRange                     = 1800 // 30 minutes
	defaultLimit                 = 20
	defaultTargetBytesPerRequest = 10 * 1024 * 1024
	defaultConcurrentRequests    = 50
)

// searchResponse is a threadsafe struct used to aggregate the responses from all downstream
// queriers
type searchResponse struct {
	err        error
	statusCode int
	statusMsg  string
	ctx        context.Context
	results    *tempopb.SearchResponse

	limit int
	mtx   sync.Mutex
}

func newSearchResponse(ctx context.Context, limit int) *searchResponse {
	return &searchResponse{
		ctx:        ctx,
		statusCode: http.StatusOK,
		limit:      limit,
		results: &tempopb.SearchResponse{
			Metrics: &tempopb.SearchMetrics{},
		},
	}
}

func (r *searchResponse) setStatus(statusCode int, statusMsg string) {
	r.mtx.Lock()
	defer r.mtx.Unlock()

	r.statusCode = statusCode
	r.statusMsg = statusMsg
}

func (r *searchResponse) setError(err error) {
	r.mtx.Lock()
	defer r.mtx.Unlock()

	r.err = err
}

func (r *searchResponse) addResponse(res *tempopb.SearchResponse) {
	r.mtx.Lock()
	defer r.mtx.Unlock()

	r.results.Traces = append(r.results.Traces, res.Traces...)
	r.results.Metrics.InspectedBlocks += res.Metrics.InspectedBlocks
	r.results.Metrics.InspectedBytes += res.Metrics.InspectedBytes
	r.results.Metrics.InspectedTraces += res.Metrics.InspectedTraces
	r.results.Metrics.SkippedBlocks += res.Metrics.SkippedBlocks
}

func (r *searchResponse) shouldQuit() bool {
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
	if len(r.results.Traces) > r.limit {
		return true
	}

	return false
}

type searchSharder struct {
	next   http.RoundTripper
	reader tempodb.Reader

	logger log.Logger

	// todo(search): make configurable
	concurrentRequests    int
	targetBytesPerRequest int
}

// newSearchSharder creates a sharding middleware for search
func newSearchSharder(reader tempodb.Reader, concurrentRequests int, logger log.Logger) Middleware {
	return MiddlewareFunc(func(next http.RoundTripper) http.RoundTripper {
		return searchSharder{
			next:                  next,
			reader:                reader,
			logger:                logger,
			concurrentRequests:    concurrentRequests,
			targetBytesPerRequest: defaultTargetBytesPerRequest,
		}
	})
}

// Roundtrip implements http.RoundTripper
//  execute up to concurrentRequests simultaneously where each request scans ~targetMBsPerRequest
//  until limit results are found
//  keeping things simple. current query params are only:
//    limit=<number>
//    start=<unix epoch seconds>
//    end=<unix epoch seconds>
//    k=<string>
//    v=<string>
func (s searchSharder) RoundTrip(r *http.Request) (*http.Response, error) {
	start, end, limit, err := searchSharderParams(r)
	if err != nil {
		return &http.Response{
			StatusCode: http.StatusBadRequest,
			Body:       io.NopCloser(strings.NewReader(err.Error())),
		}, nil
	}

	ctx := r.Context()
	tenantID, err := user.ExtractOrgID(ctx)
	if err != nil {
		return nil, err
	}
	span, ctx := opentracing.StartSpanFromContext(ctx, "frontend.ShardSearch")
	defer span.Finish()

	span.SetTag("start", start)
	span.SetTag("end", end)
	span.SetTag("limit", limit)

	blocks := s.blockMetas(start, end, tenantID)
	span.SetTag("block-count", len(blocks))

	reqs, err := s.shardedRequests(ctx, blocks, tenantID, r)
	if err != nil {
		return nil, err
	}
	span.SetTag("request-count", len(reqs))

	// execute requests
	wg := boundedwaitgroup.New(uint(s.concurrentRequests))
	overallResponse := newSearchResponse(ctx, limit)

	for _, req := range reqs {
		if overallResponse.shouldQuit() {
			break
		}

		wg.Add(1)
		go func(innerR *http.Request) {
			defer wg.Done()

			if overallResponse.shouldQuit() {
				return
			}

			resp, err := s.next.RoundTrip(innerR)
			if err != nil {
				_ = level.Error(s.logger).Log("msg", "error executing sharded query", "url", innerR.RequestURI, "err", err)
				overallResponse.setError(err)
			}

			if overallResponse.shouldQuit() {
				return
			}

			// if the status code is anything but happy, save the error and pass it down the line
			if resp.StatusCode != http.StatusOK {
				// todo: if we cancel the parent context here will it shortcircuit the other queries and fail fast?
				statusCode := resp.StatusCode
				bytesMsg, err := io.ReadAll(resp.Body)
				if err != nil {
					_ = level.Error(s.logger).Log("msg", "error reading response body status != ok", "url", innerR.RequestURI, "err", err)
				}
				statusMsg := string(bytesMsg)
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
	wg.Wait()

	// all goroutines have finished, we can safely access searchResults fields directly now
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
		}, nil
	}

	m := &jsonpb.Marshaler{}
	bodyString, err := m.MarshalToString(overallResponse.results)
	if err != nil {
		return nil, err
	}

	return &http.Response{
		StatusCode:    http.StatusOK,
		Header:        http.Header{},
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

// shardedRequests returns a slice of requests that cover all blocks in the store
// that are covered by start/end.
func (s *searchSharder) shardedRequests(ctx context.Context, metas []*backend.BlockMeta, tenantID string, parent *http.Request) ([]*http.Request, error) {
	// build requests
	//  downstream requests:
	//    - all the same as this endpoint
	//    - blockID=<guid>
	//    - startPage=<number>
	//    - totalPages=<number>
	reqs := []*http.Request{}
	for _, m := range metas {
		if m.Size == 0 || m.TotalRecords == 0 {
			continue
		}

		bytesPerPage := m.Size / uint64(m.TotalRecords)
		if bytesPerPage == 0 {
			return nil, fmt.Errorf("block %s has an invalid 0 bytes per page", m.BlockID)
		}
		pagesPerQuery := s.targetBytesPerRequest / int(bytesPerPage)
		if pagesPerQuery == 0 {
			pagesPerQuery = 1 // have to have at least 1 page per query
		}

		blockID := m.BlockID.String()
		for startPage := 0; startPage < int(m.TotalRecords); startPage += pagesPerQuery {
			subR := parent.Clone(ctx)
			subR.Header.Set(user.OrgIDHeaderName, tenantID)

			q := subR.URL.Query()
			q.Add("blockID", blockID)
			q.Add("startPage", strconv.Itoa(startPage))
			q.Add("totalPages", strconv.Itoa(pagesPerQuery))

			// adding to RequestURI only because weaveworks/common uses the RequestURI field to
			// translate from http.Request to httpgrpc.Request
			// https://github.com/weaveworks/common/blob/47e357f4e1badb7da17ad74bae63e228bdd76e8f/httpgrpc/server/server.go#L48
			subR.RequestURI = querierPrefix + parent.URL.Path + queryDelimiter + q.Encode()
			reqs = append(reqs, subR)
		}
	}

	return reqs, nil
}

func searchSharderParams(r *http.Request) (start, end int64, limit int, err error) {
	if s := r.URL.Query().Get(querier.URLParamStart); s != "" {
		start, err = strconv.ParseInt(s, 10, 64)
		if err != nil {
			return
		}
	}

	if s := r.URL.Query().Get(querier.URLParamEnd); s != "" {
		end, err = strconv.ParseInt(s, 10, 64)
		if err != nil {
			return
		}
	}

	if s := r.URL.Query().Get(querier.URLParamLimit); s != "" {
		limit, err = strconv.Atoi(s)
		if err != nil {
			return
		}
	}

	if start == 0 || end == 0 {
		err = errors.New("please provide non-zero values for http parameters start and end")
		return
	}

	if limit == 0 {
		limit = defaultLimit
	}

	if end-start > maxRange {
		err = fmt.Errorf("range specified by start and end exceeds %d seconds. received start=%d end=%d", maxRange, start, end)
		return
	}
	if end <= start {
		err = fmt.Errorf("http parameter start must be before end. received start=%d end=%d", start, end)
		return
	}

	return
}
