package frontend

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/grafana/tempo/modules/querier"
	"github.com/grafana/tempo/modules/storage"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/opentracing/opentracing-go"
	"github.com/weaveworks/common/user"
)

const (
	maxRange     = 1800 // 30 minutes
	defaultLimit = 20
)

type searchSharder struct {
	next  http.RoundTripper
	store storage.Store

	// todo(search): make configurable
	concurrentRequests    int
	targetBytesPerRequest int
	defaultLimit          int
}

func NewSearchSharder(store storage.Store) Middleware {
	return MiddlewareFunc(func(next http.RoundTripper) http.RoundTripper {
		return searchSharder{
			next:                  next,
			store:                 store,
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
		return nil, err
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
	for _, m := range metas {
		// assume each page is roughly the same size
		bytesPerPage := m.Size / uint64(m.TotalRecords)
		pagesPerQuery := s.targetBytesPerRequest / int(bytesPerPage)

		for startPage := 0; startPage < int(m.TotalRecords); startPage += pagesPerQuery {
			subR := r.Clone(ctx)
			subR.Header.Set(user.OrgIDHeaderName, tenantID)

			q := subR.URL.Query()
			q.Add("blockID", m.BlockID.String()) // jpe lot of repeated work building these strings
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

	// now execute concurrentRequests at a time
	// jpe - do this
	return nil, nil
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

	if end-start > maxRange { // jpe these should 400
		err = fmt.Errorf("range specified by start and end exceeds %d seconds", maxRange)
		return
	}
	if end <= start {
		err = fmt.Errorf("start %d must be after end %d", start, end)
		return
	}

	return
}
