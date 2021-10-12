package frontend

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/log/level"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/grafana/tempo/modules/querier"
	"github.com/grafana/tempo/modules/storage"
	"github.com/grafana/tempo/pkg/boundedwaitgroup"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/opentracing/opentracing-go"
	"github.com/weaveworks/common/user"
)

const (
	// todo(search): make configurable
	maxRange     = 1800 // 30 minutes
	defaultLimit = 20
)

type searchSharder struct {
	next  http.RoundTripper
	store storage.Store

	logger log.Logger

	// todo(search): make configurable
	concurrentRequests    int
	targetBytesPerRequest int
	defaultLimit          int
}

func NewSearchSharder(store storage.Store, logger log.Logger) Middleware {
	return MiddlewareFunc(func(next http.RoundTripper) http.RoundTripper {
		return searchSharder{
			next:                  next,
			store:                 store,
			logger:                logger,
			concurrentRequests:    20,
			targetBytesPerRequest: 10 * 1024 * 1024,
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
	start, end, limit, err := params(r)
	if err != nil {
		return &http.Response{
			StatusCode: http.StatusBadRequest,
			Body:       io.NopCloser(strings.NewReader(err.Error())),
		}, nil
	}

	ctx := r.Context()
	tenantID, err := user.ExtractOrgID(ctx)
	span, ctx := opentracing.StartSpanFromContext(ctx, "frontend.ShardSearch")
	defer span.Finish()

	span.SetTag("start", start)
	span.SetTag("end", end)
	span.SetTag("limit", limit)

	// reduce metas to those in the requested range
	metas := []*backend.BlockMeta{}
	allMetas := s.store.BlockMetas(tenantID)
	for _, m := range allMetas {
		if m.StartTime.Unix() <= end &&
			m.EndTime.Unix() >= start {
			metas = append(metas, m)
		}
	}
	span.SetTag("block-count", len(metas))

	// build requests
	//  downstream requests:
	//    - all the same as this endpoint
	//    - blockID=<guid>
	//    - startPage=<number>
	//    - totalPages=<number>
	reqs := []*http.Request{}
	for _, m := range metas { // jpe order this backwards in time, make a func and test
		// assume each page is roughly the same size
		bytesPerPage := m.Size / uint64(m.TotalRecords)
		pagesPerQuery := s.targetBytesPerRequest / int(bytesPerPage)

		blockID := m.BlockID.String()
		for startPage := 0; startPage < int(m.TotalRecords); startPage += pagesPerQuery {
			subR := r.Clone(ctx)
			subR.Header.Set(user.OrgIDHeaderName, tenantID)

			q := subR.URL.Query()
			q.Add("blockID", blockID)
			q.Add("startPage", strconv.Itoa(startPage))
			q.Add("totalPages", strconv.Itoa(pagesPerQuery))

			// adding to RequestURI only because weaveworks/common uses the RequestURI field to
			// translate from http.Request to httpgrpc.Request
			// https://github.com/weaveworks/common/blob/47e357f4e1badb7da17ad74bae63e228bdd76e8f/httpgrpc/server/server.go#L48
			subR.RequestURI = querierPrefix + r.URL.RequestURI() + queryDelimiter + q.Encode()
			reqs = append(reqs, subR)
		}
	}
	span.SetTag("request-count", len(reqs))

	// execute requests
	wg := boundedwaitgroup.New(uint(s.concurrentRequests))
	mtx := sync.Mutex{}

	var overallError error
	overallResults := &tempopb.SearchResponse{}
	statusCode := http.StatusOK
	statusMsg := ""

	for _, req := range reqs {
		wg.Add(1)
		go func(innerR *http.Request) {
			defer wg.Done()

			mtx.Lock()
			if shouldQuit(r.Context(), statusCode, overallError) { // jpe needs to include limit
				mtx.Unlock()
				return
			}
			mtx.Unlock()

			resp, err := s.next.RoundTrip(innerR)

			mtx.Lock()
			defer mtx.Unlock()
			if err != nil {
				overallError = err
			}

			if shouldQuit(r.Context(), statusCode, overallError) {
				return
			}

			// check http error
			if err != nil {
				_ = level.Error(s.logger).Log("msg", "error querying proxy target", "url", innerR.RequestURI, "err", err)
				overallError = err
				return
			}

			// if the status code is anything but happy, save the error and pass it down the line
			if resp.StatusCode != http.StatusOK {
				// todo: if we cancel the parent context here will it shortcircuit the other queries and fail fast?
				statusCode = resp.StatusCode
				bytesMsg, err := io.ReadAll(resp.Body)
				if err != nil {
					_ = level.Error(s.logger).Log("msg", "error reading response body status != ok", "url", innerR.RequestURI, "err", err)
				}
				statusMsg = string(bytesMsg)
				return
			}

			// successful query, read the body
			results := &tempopb.SearchResponse{}
			err = jsonpb.Unmarshal(resp.Body, results)
			if err != nil {
				_ = level.Error(s.logger).Log("msg", "error reading response body status == ok", "url", innerR.RequestURI, "err", err)
				overallError = err
				return
			}

			// happy path
			overallResults.Traces = append(overallResults.Traces, results.Traces...)
			overallResults.Metrics.InspectedBlocks += results.Metrics.InspectedBlocks
			overallResults.Metrics.InspectedBytes += results.Metrics.InspectedBytes
			overallResults.Metrics.InspectedTraces += results.Metrics.InspectedTraces
			overallResults.Metrics.SkippedBlocks += results.Metrics.SkippedBlocks
		}(req)
	}
	wg.Wait()

	if overallError != nil {
		return nil, overallError
	}

	if statusCode != http.StatusOK {
		// translate all non-200s into 500s. if, for instance, we get a 400 back from an internal component
		// it means that we created a bad request. 400 should not be propagated back to the user b/c
		// the bad request was due to a bug on our side, so return 500 instead.
		return &http.Response{
			StatusCode: statusCode,
			Body:       io.NopCloser(strings.NewReader(statusMsg)),
			Header:     http.Header{},
		}, nil
	}

	m := &jsonpb.Marshaler{}
	bodyString, err := m.MarshalToString(overallResults)
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

func params(r *http.Request) (start, end int64, limit int, err error) {
	if s := r.URL.Query().Get(querier.UrlParamStart); s != "" {
		start, err = strconv.ParseInt(s, 10, 64)
		if err != nil {
			return
		}
	}

	if s := r.URL.Query().Get(querier.UrlParamEnd); s != "" {
		end, err = strconv.ParseInt(s, 10, 64)
		if err != nil {
			return
		}
	}

	if s := r.URL.Query().Get(querier.UrlParamLimit); s != "" {
		limit, err = strconv.Atoi(s)
		if err != nil {
			return
		}
	}

	// allow negative values for ease of querying
	if start <= 0 {
		start = time.Now().Add(-time.Duration(start) * time.Second).Unix()
	}
	if end <= 0 {
		end = time.Now().Add(-time.Duration(end) * time.Second).Unix()
	}
	if limit == 0 {
		limit = defaultLimit
	}

	if end-start > maxRange {
		err = fmt.Errorf("range specified by start and end exceeds %d seconds", maxRange)
		return
	}
	if end <= start {
		err = fmt.Errorf("start %d must be after end %d", start, end)
		return
	}

	return
}
