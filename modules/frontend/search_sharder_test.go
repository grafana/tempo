package frontend

import (
	"context"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/google/uuid"
	"github.com/grafana/dskit/user"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/modules/frontend/combiner"
	"github.com/grafana/tempo/modules/frontend/pipeline"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/traceql"
	"github.com/grafana/tempo/tempodb"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/blocklist"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

var _ tempodb.Reader = (*mockReader)(nil)

// implements tempodb.Reader interface
type mockReader struct {
	metas   []*backend.BlockMeta
	tenants []string
}

func (m *mockReader) SearchTags(context.Context, *backend.BlockMeta, *tempopb.SearchTagsBlockRequest, common.SearchOptions) (*tempopb.SearchTagsV2Response, error) {
	return nil, nil
}

func (m *mockReader) SearchTagValues(context.Context, *backend.BlockMeta, *tempopb.SearchTagValuesBlockRequest, common.SearchOptions) (*tempopb.SearchTagValuesResponse, error) {
	return nil, nil
}

func (m *mockReader) SearchTagValuesV2(context.Context, *backend.BlockMeta, *tempopb.SearchTagValuesRequest, common.SearchOptions) (*tempopb.SearchTagValuesV2Response, error) {
	return nil, nil
}

func (m *mockReader) FetchTagValues(context.Context, *backend.BlockMeta, traceql.FetchTagValuesRequest, traceql.FetchTagValuesCallback, common.MetricsCallback, common.SearchOptions) error {
	return nil
}

func (m *mockReader) Find(context.Context, string, common.ID, string, string, int64, int64, common.SearchOptions) ([]*tempopb.TraceByIDResponse, []error, error) {
	return nil, nil, nil
}

func (m *mockReader) BlockMetas(string) []*backend.BlockMeta {
	return m.metas
}

func (m *mockReader) Tenants() []string {
	return m.tenants
}

func (m *mockReader) Search(context.Context, *backend.BlockMeta, *tempopb.SearchRequest, common.SearchOptions) (*tempopb.SearchResponse, error) {
	return nil, nil
}

func (m *mockReader) Fetch(context.Context, *backend.BlockMeta, traceql.FetchSpansRequest, common.SearchOptions) (traceql.FetchSpansResponse, error) {
	return traceql.FetchSpansResponse{}, nil
}

func (m *mockReader) FetchTagNames(context.Context, *backend.BlockMeta, traceql.FetchTagsRequest, traceql.FetchTagsCallback, common.MetricsCallback, common.SearchOptions) error {
	return nil
}

func (m *mockReader) EnablePolling(context.Context, blocklist.JobSharder) {}
func (m *mockReader) Shutdown()                                           {}

//nolint:all deprecated

func TestBuildBackendRequests(t *testing.T) {
	tests := []struct {
		targetBytesPerRequest int
		metas                 []*backend.BlockMeta
		expectedURIs          []string
	}{
		{
			expectedURIs: []string{},
		},
		// block with no size
		{
			metas: []*backend.BlockMeta{
				{
					BlockID: backend.MustParse("00000000-0000-0000-0000-000000000000"),
				},
			},
			expectedURIs: []string{},
		},
		// block with no records
		{
			metas: []*backend.BlockMeta{
				{
					Size_:   1000,
					BlockID: backend.MustParse("00000000-0000-0000-0000-000000000000"),
				},
			},
			expectedURIs: []string{},
		},
		// meta.json fields
		{
			targetBytesPerRequest: 1000,
			metas: []*backend.BlockMeta{
				{
					Size_:         1000,
					TotalRecords:  100,
					BlockID:       backend.MustParse("00000000-0000-0000-0000-000000000000"),
					DataEncoding:  "json",
					Encoding:      backend.EncGZIP,
					IndexPageSize: 13,
					Version:       "glarg",
				},
			},
			expectedURIs: []string{
				"/querier?blockID=00000000-0000-0000-0000-000000000000&dataEncoding=json&encoding=gzip&end=20&footerSize=0&indexPageSize=13&k=test&pagesToSearch=100&size=1000&start=10&startPage=0&totalRecords=100&v=test&version=glarg",
			},
		},
		// meta.json with dedicated columns
		{
			targetBytesPerRequest: 1000,
			metas: []*backend.BlockMeta{
				{
					Size_:         1000,
					TotalRecords:  10,
					BlockID:       backend.MustParse("00000000-0000-0000-0000-000000000000"),
					Encoding:      backend.EncNone,
					IndexPageSize: 13,
					Version:       "vParquet3",
					DedicatedColumns: backend.DedicatedColumns{
						{Scope: "span", Name: "net.sock.host.addr", Type: "string"},
					},
				},
			},
			expectedURIs: []string{
				"/querier?blockID=00000000-0000-0000-0000-000000000000&dataEncoding=&dc=%5B%7B%22name%22%3A%22net.sock.host.addr%22%7D%5D&encoding=none&end=20&footerSize=0&indexPageSize=13&k=test&pagesToSearch=10&size=1000&start=10&startPage=0&totalRecords=10&v=test&version=vParquet3",
			},
		},
		// bytes/per request is too small for the page size
		{
			targetBytesPerRequest: 1,
			metas: []*backend.BlockMeta{
				{
					Size_:        1000,
					TotalRecords: 3,
					BlockID:      backend.MustParse("00000000-0000-0000-0000-000000000000"),
				},
			},
			expectedURIs: []string{
				"/querier?blockID=00000000-0000-0000-0000-000000000000&dataEncoding=&encoding=none&end=20&footerSize=0&indexPageSize=0&k=test&pagesToSearch=1&size=1000&start=10&startPage=0&totalRecords=3&v=test&version=",
				"/querier?blockID=00000000-0000-0000-0000-000000000000&dataEncoding=&encoding=none&end=20&footerSize=0&indexPageSize=0&k=test&pagesToSearch=1&size=1000&start=10&startPage=1&totalRecords=3&v=test&version=",
				"/querier?blockID=00000000-0000-0000-0000-000000000000&dataEncoding=&encoding=none&end=20&footerSize=0&indexPageSize=0&k=test&pagesToSearch=1&size=1000&start=10&startPage=2&totalRecords=3&v=test&version=",
			},
		},
		// 100 pages, 10 bytes per page, 1k allowed per request
		{
			targetBytesPerRequest: 1000,
			metas: []*backend.BlockMeta{
				{
					Size_:        1000,
					TotalRecords: 100,
					BlockID:      backend.MustParse("00000000-0000-0000-0000-000000000000"),
				},
			},
			expectedURIs: []string{
				"/querier?blockID=00000000-0000-0000-0000-000000000000&dataEncoding=&encoding=none&end=20&footerSize=0&indexPageSize=0&k=test&pagesToSearch=100&size=1000&start=10&startPage=0&totalRecords=100&v=test&version=",
			},
		},
		// 100 pages, 10 bytes per page, 900 allowed per request
		{
			targetBytesPerRequest: 900,
			metas: []*backend.BlockMeta{
				{
					Size_:        1000,
					TotalRecords: 100,
					BlockID:      backend.MustParse("00000000-0000-0000-0000-000000000000"),
				},
			},
			expectedURIs: []string{
				"/querier?blockID=00000000-0000-0000-0000-000000000000&dataEncoding=&encoding=none&end=20&footerSize=0&indexPageSize=0&k=test&pagesToSearch=90&size=1000&start=10&startPage=0&totalRecords=100&v=test&version=",
				"/querier?blockID=00000000-0000-0000-0000-000000000000&dataEncoding=&encoding=none&end=20&footerSize=0&indexPageSize=0&k=test&pagesToSearch=90&size=1000&start=10&startPage=90&totalRecords=100&v=test&version=",
			},
		},
		// two blocks
		{
			targetBytesPerRequest: 900,
			metas: []*backend.BlockMeta{
				{
					Size_:        1000,
					TotalRecords: 100,
					BlockID:      backend.MustParse("00000000-0000-0000-0000-000000000000"),
				},
				{
					Size_:        1000,
					TotalRecords: 200,
					BlockID:      backend.MustParse("00000000-0000-0000-0000-000000000001"),
				},
			},
			expectedURIs: []string{
				"/querier?blockID=00000000-0000-0000-0000-000000000000&dataEncoding=&encoding=none&end=20&footerSize=0&indexPageSize=0&k=test&pagesToSearch=90&size=1000&start=10&startPage=0&totalRecords=100&v=test&version=",
				"/querier?blockID=00000000-0000-0000-0000-000000000000&dataEncoding=&encoding=none&end=20&footerSize=0&indexPageSize=0&k=test&pagesToSearch=90&size=1000&start=10&startPage=90&totalRecords=100&v=test&version=",
				"/querier?blockID=00000000-0000-0000-0000-000000000001&dataEncoding=&encoding=none&end=20&footerSize=0&indexPageSize=0&k=test&pagesToSearch=180&size=1000&start=10&startPage=0&totalRecords=200&v=test&version=",
				"/querier?blockID=00000000-0000-0000-0000-000000000001&dataEncoding=&encoding=none&end=20&footerSize=0&indexPageSize=0&k=test&pagesToSearch=180&size=1000&start=10&startPage=180&totalRecords=200&v=test&version=",
			},
		},
	}

	for _, tc := range tests {
		req := httptest.NewRequest("GET", "/?k=test&v=test&start=10&end=20", nil)
		searchReq, err := api.ParseSearchRequest(req)
		require.NoError(t, err)

		ctx, cancelCause := context.WithCancelCause(context.Background())
		reqCh := make(chan pipeline.Request)
		iterFn := backendJobsFunc(tc.metas, tc.targetBytesPerRequest, defaultMostRecentShards, math.MaxUint32)

		go func() {
			buildBackendRequests(ctx, "test", pipeline.NewHTTPRequest(req), searchReq, iterFn, reqCh, cancelCause)
		}()

		actualURIs := []string{}
		for r := range reqCh {
			if r != nil {
				actualURIs = append(actualURIs, r.HTTPRequest().RequestURI)
			}
		}

		assert.NoError(t, ctx.Err())

		urisEqual(t, tc.expectedURIs, actualURIs)
	}
}

func TestBackendRequests(t *testing.T) {
	bm := backend.NewBlockMeta("test", uuid.New(), "wdwad", backend.EncGZIP, "asdf")
	bm.StartTime = time.Unix(100, 0)
	bm.EndTime = time.Unix(200, 0)
	bm.Size_ = defaultTargetBytesPerRequest * 2
	bm.TotalRecords = 2

	s := &asyncSearchSharder{
		cfg: SearchSharderConfig{
			MostRecentShards: defaultMostRecentShards,
		},
		reader: &mockReader{metas: []*backend.BlockMeta{bm}},
	}

	tests := []struct {
		name               string
		request            string
		expectedReqsURIs   []string
		expectedJobs       int
		expectedBlocks     int
		expectedBlockBytes uint64
	}{
		{
			name:    "start and end same as block",
			request: "/?tags=foo%3Dbar&minDuration=10ms&maxDuration=30ms&limit=50&start=100&end=200",
			expectedReqsURIs: []string{
				"/querier?blockID=" + bm.BlockID.String() + "&dataEncoding=asdf&encoding=gzip&end=200&footerSize=0&indexPageSize=0&limit=50&maxDuration=30ms&minDuration=10ms&pagesToSearch=1&size=209715200&start=100&startPage=0&tags=foo%3Dbar&totalRecords=2&version=wdwad",
				"/querier?blockID=" + bm.BlockID.String() + "&dataEncoding=asdf&encoding=gzip&end=200&footerSize=0&indexPageSize=0&limit=50&maxDuration=30ms&minDuration=10ms&pagesToSearch=1&size=209715200&start=100&startPage=1&tags=foo%3Dbar&totalRecords=2&version=wdwad",
			},
			expectedJobs:       2,
			expectedBlocks:     1,
			expectedBlockBytes: defaultTargetBytesPerRequest * 2,
		},
		{
			name:    "start and end in block",
			request: "/?tags=foo%3Dbar&minDuration=10ms&maxDuration=30ms&limit=50&start=110&end=150",
			expectedReqsURIs: []string{
				"/querier?blockID=" + bm.BlockID.String() + "&dataEncoding=asdf&encoding=gzip&end=150&footerSize=0&indexPageSize=0&limit=50&maxDuration=30ms&minDuration=10ms&pagesToSearch=1&size=209715200&start=110&startPage=0&tags=foo%3Dbar&totalRecords=2&version=wdwad",
				"/querier?blockID=" + bm.BlockID.String() + "&dataEncoding=asdf&encoding=gzip&end=150&footerSize=0&indexPageSize=0&limit=50&maxDuration=30ms&minDuration=10ms&pagesToSearch=1&size=209715200&start=110&startPage=1&tags=foo%3Dbar&totalRecords=2&version=wdwad",
			},
			expectedJobs:       2,
			expectedBlocks:     1,
			expectedBlockBytes: defaultTargetBytesPerRequest * 2,
		},
		{
			name:             "start and end out of block",
			request:          "/?tags=foo%3Dbar&minDuration=10ms&maxDuration=30ms&limit=50&start=10&end=20",
			expectedReqsURIs: make([]string, 0),
		},
		{
			name:             "no start and end",
			request:          "/?tags=foo%3Dbar&minDuration=10ms&maxDuration=30ms&limit=50",
			expectedReqsURIs: make([]string, 0),
		},
		{
			name:             "only tags",
			request:          "/?tags=foo%3Dbar",
			expectedReqsURIs: make([]string, 0),
		},
		{
			name:             "no params",
			request:          "/",
			expectedReqsURIs: make([]string, 0),
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", tc.request, nil)
			searchReq, err := api.ParseSearchRequest(r)
			require.NoError(t, err)

			stopCh := make(chan struct{})
			defer close(stopCh)
			reqCh := make(chan pipeline.Request)

			ctx, cancelCause := context.WithCancelCause(context.Background())
			pipelineRequest := pipeline.NewHTTPRequest(r)

			searchJobResponse := &combiner.SearchJobResponse{}
			s.backendRequests(ctx, "test", pipelineRequest, searchReq, searchJobResponse, reqCh, cancelCause)
			require.Equal(t, tc.expectedJobs, searchJobResponse.TotalJobs)
			require.Equal(t, tc.expectedBlocks, searchJobResponse.TotalBlocks)
			require.Equal(t, tc.expectedBlockBytes, searchJobResponse.TotalBytes)

			actualReqURIs := []string{}
			for r := range reqCh {
				if r != nil {
					actualReqURIs = append(actualReqURIs, r.HTTPRequest().RequestURI)
				}
			}
			require.NoError(t, ctx.Err())
			urisEqual(t, tc.expectedReqsURIs, actualReqURIs)
		})
	}
}

func TestIngesterRequests(t *testing.T) {
	nownow := time.Now()

	now := int(time.Now().Unix())

	ago := func(d string) int {
		duration, err := time.ParseDuration(d)
		require.NoError(t, err)
		return int(nownow.Add(-duration).Unix())
	}
	tenMinutesAgo := int(time.Now().Add(-10 * time.Minute).Unix())
	fifteenMinutesAgo := int(time.Now().Add(-15 * time.Minute).Unix())
	twentyMinutesAgo := int(time.Now().Add(-20 * time.Minute).Unix())

	tests := []struct {
		request             string
		queryIngestersUntil time.Duration
		ingesterShards      int
		expectedURI         []string
		expectedError       error
	}{
		// start/end is outside queryIngestersUntil
		{
			request:             "/?tags=foo%3Dbar&minDuration=10ms&maxDuration=30ms&limit=50&start=10&end=20",
			queryIngestersUntil: 10 * time.Minute,
			expectedURI:         []string{},
			ingesterShards:      1,
		},
		// start/end is inside queryBackendAfter
		{
			request:             "/?tags=foo%3Dbar&minDuration=10ms&maxDuration=30ms&limit=50&start=" + strconv.Itoa(tenMinutesAgo) + "&end=" + strconv.Itoa(now),
			queryIngestersUntil: 30 * time.Minute,
			expectedURI:         []string{"/querier?end=" + strconv.Itoa(now) + "&limit=50&maxDuration=30ms&minDuration=10ms&spss=3&start=" + strconv.Itoa(tenMinutesAgo) + "&tags=foo%3Dbar"},
			ingesterShards:      1,
		},
		// backendAfter/ingsetersUntil = 0 results in no ingester query
		{
			request:             "/?tags=foo%3Dbar&minDuration=10ms&maxDuration=30ms&limit=50&start=" + strconv.Itoa(tenMinutesAgo) + "&end=" + strconv.Itoa(now),
			queryIngestersUntil: 0,
			expectedURI:         []string{},
			ingesterShards:      1,
		},
		// start/end = 20 - 10 mins ago - break across query ingesters until
		//  ingester start/End = 15 - 10 mins ago
		{
			request:             "/?tags=foo%3Dbar&minDuration=10ms&maxDuration=30ms&limit=50&start=" + strconv.Itoa(twentyMinutesAgo) + "&end=" + strconv.Itoa(tenMinutesAgo),
			queryIngestersUntil: 15 * time.Minute,
			expectedURI:         []string{"/querier?end=" + strconv.Itoa(tenMinutesAgo) + "&limit=50&maxDuration=30ms&minDuration=10ms&spss=3&start=" + strconv.Itoa(fifteenMinutesAgo) + "&tags=foo%3Dbar"},
			ingesterShards:      1,
		},
		// start/end = 10 - now mins ago - break across query backend after
		//  ingester start/End = 10 - now mins ago
		//  backend start/End = 15 - 10 mins ago
		{
			request:             "/?tags=foo%3Dbar&minDuration=10ms&maxDuration=30ms&limit=50&start=" + strconv.Itoa(tenMinutesAgo) + "&end=" + strconv.Itoa(now),
			queryIngestersUntil: 15 * time.Minute,
			expectedURI:         []string{"/querier?end=" + strconv.Itoa(now) + "&limit=50&maxDuration=30ms&minDuration=10ms&spss=3&start=" + strconv.Itoa(tenMinutesAgo) + "&tags=foo%3Dbar"},
			ingesterShards:      1,
		},
		// start/end = 20 - now mins ago - break across both query ingesters until and backend after
		//  ingester start/End = 15 - now mins ago
		//  backend start/End = 20 - 5 mins ago
		{
			request:             "/?tags=foo%3Dbar&minDuration=10ms&maxDuration=30ms&limit=50&start=" + strconv.Itoa(twentyMinutesAgo) + "&end=" + strconv.Itoa(now),
			queryIngestersUntil: 15 * time.Minute,
			expectedURI:         []string{"/querier?end=" + strconv.Itoa(now) + "&limit=50&maxDuration=30ms&minDuration=10ms&spss=3&start=" + strconv.Itoa(fifteenMinutesAgo) + "&tags=foo%3Dbar"},
			ingesterShards:      1,
		},
		{
			request:             "/?tags=foo%3Dbar&minDuration=10ms&maxDuration=30ms&limit=50",
			queryIngestersUntil: 15 * time.Minute,
			expectedURI:         []string{"/querier?end=0&limit=50&maxDuration=30ms&minDuration=10ms&spss=3&start=0&tags=foo%3Dbar"},
			ingesterShards:      1,
		},
		{
			request:             "/?limit=50",
			queryIngestersUntil: 15 * time.Minute,
			expectedURI:         []string{"/querier?end=0&limit=50&spss=3&start=0"},
			ingesterShards:      1,
		},
		// start/end = 20 - 10 mins ago - break across query ingesters until
		//  ingester start/End = 15 - 10 mins ago -- 5 minutes split in 2 shards.
		{
			request:             "/?tags=foo%3Dbar&minDuration=12ms&maxDuration=30ms&limit=50&start=" + strconv.Itoa(ago("20m")) + "&end=" + strconv.Itoa(ago("10m")),
			queryIngestersUntil: 15 * time.Minute,
			expectedURI: []string{
				"/querier?end=" + strconv.Itoa(ago("12.5m")) + "&limit=50&maxDuration=30ms&minDuration=12ms&spss=3&start=" + strconv.Itoa(ago("15m")) + "&tags=foo%3Dbar",
				"/querier?end=" + strconv.Itoa(ago("10m")) + "&limit=50&maxDuration=30ms&minDuration=12ms&spss=3&start=" + strconv.Itoa(ago("12.5m")) + "&tags=foo%3Dbar",
			},
			ingesterShards: 2,
		},
		// start/end when entirely within the ingester search window when split across 3 shards.
		{
			request:             "/?tags=foo%3Dbar&minDuration=11ms&maxDuration=30ms&limit=50&start=" + strconv.Itoa(ago("15m")) + "&end=" + strconv.Itoa(ago("0s")),
			queryIngestersUntil: 15 * time.Minute,
			expectedURI: []string{
				"/querier?end=" + strconv.Itoa(ago("10m")) + "&limit=50&maxDuration=30ms&minDuration=11ms&spss=3&start=" + strconv.Itoa(ago("15m")) + "&tags=foo%3Dbar",
				"/querier?end=" + strconv.Itoa(ago("5m")) + "&limit=50&maxDuration=30ms&minDuration=11ms&spss=3&start=" + strconv.Itoa(ago("10m")) + "&tags=foo%3Dbar",
				"/querier?end=" + strconv.Itoa(now) + "&limit=50&maxDuration=30ms&minDuration=11ms&spss=3&start=" + strconv.Itoa(ago("5m")) + "&tags=foo%3Dbar",
			},
			ingesterShards: 3,
		},
		// start/end when entirely within ingeste search window, but check that we don't shard too much.
		{
			request:             "/?tags=foo%3Dbar&minDuration=11ms&maxDuration=30ms&limit=50&start=" + strconv.Itoa(ago("15m")) + "&end=" + strconv.Itoa(ago("0s")),
			queryIngestersUntil: 5*time.Minute + 10*time.Second,
			expectedURI: []string{
				"/querier?end=" + strconv.Itoa(ago("4m10s")) + "&limit=50&maxDuration=30ms&minDuration=11ms&spss=3&start=" + strconv.Itoa(ago("5m10s")) + "&tags=foo%3Dbar",
				"/querier?end=" + strconv.Itoa(ago("3m10s")) + "&limit=50&maxDuration=30ms&minDuration=11ms&spss=3&start=" + strconv.Itoa(ago("4m10s")) + "&tags=foo%3Dbar",
				"/querier?end=" + strconv.Itoa(ago("2m10s")) + "&limit=50&maxDuration=30ms&minDuration=11ms&spss=3&start=" + strconv.Itoa(ago("3m10s")) + "&tags=foo%3Dbar",
				"/querier?end=" + strconv.Itoa(ago("1m10s")) + "&limit=50&maxDuration=30ms&minDuration=11ms&spss=3&start=" + strconv.Itoa(ago("2m10s")) + "&tags=foo%3Dbar",
				"/querier?end=" + strconv.Itoa(ago("10s")) + "&limit=50&maxDuration=30ms&minDuration=11ms&spss=3&start=" + strconv.Itoa(ago("1m10s")) + "&tags=foo%3Dbar",
				"/querier?end=" + strconv.Itoa(now) + "&limit=50&maxDuration=30ms&minDuration=11ms&spss=3&start=" + strconv.Itoa(ago("10s")) + "&tags=foo%3Dbar",
			},
			ingesterShards: 6,
		},
		// start/end when entirely within ingeste search window, but check that we don't shard too much with a large number of shards for a small window.
		{
			request:             "/?tags=foo%3Dbar&minDuration=11ms&maxDuration=30ms&limit=50&start=" + strconv.Itoa(ago("15m")) + "&end=" + strconv.Itoa(ago("0s")),
			queryIngestersUntil: 5 * time.Minute,
			expectedURI: []string{
				"/querier?end=" + strconv.Itoa(ago("4m")) + "&limit=50&maxDuration=30ms&minDuration=11ms&spss=3&start=" + strconv.Itoa(ago("5m")) + "&tags=foo%3Dbar",
				"/querier?end=" + strconv.Itoa(ago("3m")) + "&limit=50&maxDuration=30ms&minDuration=11ms&spss=3&start=" + strconv.Itoa(ago("4m")) + "&tags=foo%3Dbar",
				"/querier?end=" + strconv.Itoa(ago("2m")) + "&limit=50&maxDuration=30ms&minDuration=11ms&spss=3&start=" + strconv.Itoa(ago("3m")) + "&tags=foo%3Dbar",
				"/querier?end=" + strconv.Itoa(ago("1m")) + "&limit=50&maxDuration=30ms&minDuration=11ms&spss=3&start=" + strconv.Itoa(ago("2m")) + "&tags=foo%3Dbar",
				"/querier?end=" + strconv.Itoa(now) + "&limit=50&maxDuration=30ms&minDuration=11ms&spss=3&start=" + strconv.Itoa(ago("1m")) + "&tags=foo%3Dbar",
			},
			ingesterShards: 30,
		},
		{
			request:             "/?tags=foo%3Dbar&minDuration=11ms&maxDuration=30ms&limit=50&start=" + strconv.Itoa(ago("15m")) + "&end=" + strconv.Itoa(ago("0s")),
			queryIngestersUntil: 350 * time.Second,
			expectedURI: []string{
				"/querier?end=" + strconv.Itoa(ago("3m54s")) + "&limit=50&maxDuration=30ms&minDuration=11ms&spss=3&start=" + strconv.Itoa(ago("5m50s")) + "&tags=foo%3Dbar",
				"/querier?end=" + strconv.Itoa(ago("1m58s")) + "&limit=50&maxDuration=30ms&minDuration=11ms&spss=3&start=" + strconv.Itoa(ago("3m54s")) + "&tags=foo%3Dbar",
				"/querier?end=" + strconv.Itoa(now) + "&limit=50&maxDuration=30ms&minDuration=11ms&spss=3&start=" + strconv.Itoa(ago("1m58s")) + "&tags=foo%3Dbar",
			},
			ingesterShards: 3,
		},
	}

	for i, tc := range tests {
		t.Logf("test case %d", i)
		require.Greater(t, tc.ingesterShards, 0)
		s := &asyncSearchSharder{
			cfg: SearchSharderConfig{
				QueryIngestersUntil: tc.queryIngestersUntil,
				IngesterShards:      tc.ingesterShards,
			},
		}
		req := httptest.NewRequest("GET", tc.request, nil)

		searchReq, err := api.ParseSearchRequest(req)
		require.NoError(t, err)

		reqChan := make(chan pipeline.Request, tc.ingesterShards)
		defer close(reqChan)

		pr := pipeline.NewHTTPRequest(req)
		pr.SetWeight(2)
		actualSearchResponse, err := s.ingesterRequests("test", pr, *searchReq, reqChan)
		if tc.expectedError != nil {
			require.Equal(t, tc.expectedError, err)
			continue
		}
		require.NoError(t, err)
		require.Equal(t, len(tc.expectedURI), len(reqChan))
		require.Equal(t, len(tc.expectedURI), actualSearchResponse.TotalJobs)
		if len(tc.expectedURI) > 0 {
			require.Equal(t, len(tc.expectedURI), int(actualSearchResponse.Shards[0].TotalJobs))
			expectedCompletedThrough := math.MaxUint32      // normal ingester shard completes no time on purpose
			if searchReq.Start == 0 && searchReq.End == 0 { // ingester only search completes all time on purpose
				expectedCompletedThrough = 1
			}
			require.Equal(t, expectedCompletedThrough, int(actualSearchResponse.Shards[0].CompletedThroughSeconds))
		} else {
			require.Equal(t, 0, len(actualSearchResponse.Shards))
		}

		// drain the channel and check the URIs
		for _, expectedURI := range tc.expectedURI {
			req := <-reqChan
			require.NotNil(t, req)

			values := req.HTTPRequest().URL.Query()
			expectedQueryStringValues, err := url.ParseQuery(expectedURI)
			require.NoError(t, err)

			for k, v := range expectedQueryStringValues {
				key := k

				// Due the way the query string is parse, we need to ensure that
				// the first query param is captured.  Split the key on the first ? and
				// use the second part as the key.
				if strings.Contains(k, "?") {
					parts := strings.Split(k, "?")
					require.Equal(t, 2, len(parts))
					key = parts[1]
				}

				if key == "start" || key == "end" {
					// check the time difference between the expected and actual
					// start/end times is within a tollerance for the use of time.Now()
					// in the code compared to when the tests check the values.

					actual := timeFrom(t, values[key][0])
					expected := timeFrom(t, v[0])

					diff := expected.Sub(actual)
					assert.LessOrEqual(t, diff, time.Millisecond)

					diff = actual.Sub(expected)
					assert.LessOrEqual(t, diff, time.Millisecond)

					continue
				}

				require.Equal(t, v, values[k])
				require.Equal(t, 2, req.Weight())
			}
		}
	}
}

func timeFrom(t *testing.T, n string) time.Time {
	i, err := strconv.ParseInt(n, 10, 32)
	require.NoError(t, err)
	return time.Unix(i, 0)
}

func TestBackendRange(t *testing.T) {
	now := int(time.Now().Unix())
	fiveMinutesAgo := int(time.Now().Add(-5 * time.Minute).Unix())
	tenMinutesAgo := int(time.Now().Add(-10 * time.Minute).Unix())
	fifteenMinutesAgo := int(time.Now().Add(-15 * time.Minute).Unix())
	twentyMinutesAgo := int(time.Now().Add(-20 * time.Minute).Unix())

	tests := []struct {
		request           string
		queryBackendAfter time.Duration
		expectedStart     uint32
		expectedEnd       uint32
	}{
		// start/end is outside queryIngestersUntil
		{
			request:           "/?tags=foo%3Dbar&minDuration=10ms&maxDuration=30ms&limit=50&start=10&end=20",
			queryBackendAfter: time.Minute,
			expectedStart:     10,
			expectedEnd:       20,
		},
		// start/end is inside queryBackendAfter
		{
			request:           "/?tags=foo%3Dbar&minDuration=10ms&maxDuration=30ms&limit=50&start=" + strconv.Itoa(tenMinutesAgo) + "&end=" + strconv.Itoa(now),
			queryBackendAfter: 15 * time.Minute,
			expectedStart:     uint32(fifteenMinutesAgo),
			expectedEnd:       uint32(fifteenMinutesAgo),
		},
		// backendAfter/ingsetersUntil = 0 results in no ingester query
		{
			request:           "/?tags=foo%3Dbar&minDuration=10ms&maxDuration=30ms&limit=50&start=" + strconv.Itoa(tenMinutesAgo) + "&end=" + strconv.Itoa(now),
			queryBackendAfter: 0,
			expectedStart:     uint32(tenMinutesAgo),
			expectedEnd:       uint32(now),
		},
		// start/end = 20 - 10 mins ago - break across query ingesters until
		//  ingester start/End = 15 - 10 mins ago
		//  backend start/End = 20 - 10 mins ago
		{
			request:           "/?tags=foo%3Dbar&minDuration=10ms&maxDuration=30ms&limit=50&start=" + strconv.Itoa(twentyMinutesAgo) + "&end=" + strconv.Itoa(tenMinutesAgo),
			queryBackendAfter: 5 * time.Minute,
			expectedStart:     uint32(twentyMinutesAgo),
			expectedEnd:       uint32(tenMinutesAgo),
		},
		// start/end = 10 - now mins ago - break across query backend after
		//  ingester start/End = 10 - now mins ago
		//  backend start/End = 15 - 10 mins ago
		{
			request:           "/?tags=foo%3Dbar&minDuration=10ms&maxDuration=30ms&limit=50&start=" + strconv.Itoa(tenMinutesAgo) + "&end=" + strconv.Itoa(now),
			queryBackendAfter: 5 * time.Minute,
			expectedStart:     uint32(tenMinutesAgo),
			expectedEnd:       uint32(fiveMinutesAgo),
		},
		// start/end = 20 - now mins ago - break across both query ingesters until and backend after
		//  ingester start/End = 15 - now mins ago
		//  backend start/End = 20 - 5 mins ago
		{
			request:           "/?tags=foo%3Dbar&minDuration=10ms&maxDuration=30ms&limit=50&start=" + strconv.Itoa(twentyMinutesAgo) + "&end=" + strconv.Itoa(now),
			queryBackendAfter: 5 * time.Minute,
			expectedStart:     uint32(twentyMinutesAgo),
			expectedEnd:       uint32(fiveMinutesAgo),
		},
		// request without start and end should return start and end as 0
		{
			request:           "/?tags=foo%3Dbar&minDuration=10ms&maxDuration=30ms&limit=50",
			queryBackendAfter: 5 * time.Minute,
			expectedStart:     0,
			expectedEnd:       0,
		},
	}

	for _, tc := range tests {
		req := httptest.NewRequest("GET", tc.request, nil)

		searchReq, err := api.ParseSearchRequest(req)
		require.NoError(t, err)

		actualStart, actualEnd := backendRange(searchReq.Start, searchReq.End, tc.queryBackendAfter)
		assert.Equal(t, int(tc.expectedStart), int(actualStart))
		assert.Equal(t, int(tc.expectedEnd), int(actualEnd))
	}
}

func TestTotalJobsIncludesIngester(t *testing.T) {
	next := pipeline.AsyncRoundTripperFunc[combiner.PipelineResponse](func(_ pipeline.Request) (pipeline.Responses[combiner.PipelineResponse], error) {
		resString, err := (&jsonpb.Marshaler{}).MarshalToString(&tempopb.SearchResponse{
			Metrics: &tempopb.SearchMetrics{},
		})
		require.NoError(t, err)

		return pipeline.NewHTTPToAsyncResponse(&http.Response{
			Body:       io.NopCloser(strings.NewReader(resString)),
			StatusCode: 200,
		}), nil
	})

	o, err := overrides.NewOverrides(overrides.Config{}, nil, prometheus.DefaultRegisterer)
	require.NoError(t, err)

	now := time.Now().Add(-10 * time.Minute).Unix()

	sharder := newAsyncSearchSharder(&mockReader{
		metas: []*backend.BlockMeta{ // one block with 2 records that are each the target bytes per request will force 2 sub queries
			{
				StartTime:    time.Unix(now, 0),
				EndTime:      time.Unix(now, 0),
				Size_:        defaultTargetBytesPerRequest * 2,
				TotalRecords: 2,
				BlockID:      backend.MustParse("00000000-0000-0000-0000-000000000000"),
			},
		},
	}, o, SearchSharderConfig{
		QueryIngestersUntil:   15 * time.Minute,
		ConcurrentRequests:    1, // 1 concurrent request to force order
		TargetBytesPerRequest: defaultTargetBytesPerRequest,
		MostRecentShards:      defaultMostRecentShards,
		IngesterShards:        1,
	}, log.NewNopLogger())
	testRT := sharder.Wrap(next)

	path := fmt.Sprintf("/?start=%d&end=%d", now-1, now+1)
	req := httptest.NewRequest("GET", path, nil)
	ctx := req.Context()
	ctx = user.InjectOrgID(ctx, "blerg")
	req = req.WithContext(ctx)

	resps, err := testRT.RoundTrip(pipeline.NewHTTPRequest(req))
	require.NoError(t, err)
	// find a response with total jobs > . this is the metadata response

	totalJobs := 0
	for {
		res, done, err := resps.Next(context.Background())

		if res.IsMetadata() {
			searchJobResponse := res.(*combiner.SearchJobResponse)
			totalJobs += searchJobResponse.TotalJobs

			break
		}

		require.NoError(t, err)
		require.False(t, done)
	}

	// 2 jobs for the meta + 1 for the ingester
	assert.Equal(t, 3, totalJobs)
}

func TestSearchSharderRoundTripBadRequest(t *testing.T) {
	next := pipeline.AsyncRoundTripperFunc[combiner.PipelineResponse](func(_ pipeline.Request) (pipeline.Responses[combiner.PipelineResponse], error) {
		return nil, nil
	})

	o, err := overrides.NewOverrides(overrides.Config{}, nil, prometheus.DefaultRegisterer)
	require.NoError(t, err)

	sharder := newAsyncSearchSharder(&mockReader{}, o, SearchSharderConfig{
		ConcurrentRequests:    defaultConcurrentRequests,
		TargetBytesPerRequest: defaultTargetBytesPerRequest,
		MostRecentShards:      defaultMostRecentShards,
		MaxDuration:           5 * time.Minute,
		MaxSpansPerSpanSet:    100,
	}, log.NewNopLogger())
	testRT := sharder.Wrap(next)

	// no org id
	req := httptest.NewRequest("GET", "/?start=1000&end=1100", nil)
	resp, err := testRT.RoundTrip(pipeline.NewHTTPRequest(req))
	testBadRequestFromResponses(t, resp, err, "no org id")

	// start/end outside of max duration
	req = httptest.NewRequest("GET", "/?start=1000&end=1500", nil)
	req = req.WithContext(user.InjectOrgID(req.Context(), "blerg"))
	resp, err = testRT.RoundTrip(pipeline.NewHTTPRequest(req))
	testBadRequestFromResponses(t, resp, err, "range specified by start and end exceeds 5m0s. received start=1000 end=1500")

	// spans per span set greater than maximum
	req = httptest.NewRequest("GET", "/?spss=200", nil)
	req = req.WithContext(user.InjectOrgID(req.Context(), "blerg"))
	resp, err = testRT.RoundTrip(pipeline.NewHTTPRequest(req))
	testBadRequestFromResponses(t, resp, err, "spans per span set exceeds 100. received 200")

	// bad request
	req = httptest.NewRequest("GET", "/?start=asdf&end=1500", nil)
	resp, err = testRT.RoundTrip(pipeline.NewHTTPRequest(req))
	testBadRequestFromResponses(t, resp, err, "invalid start: strconv.ParseInt: parsing \"asdf\": invalid syntax")

	// test max duration error with overrides
	o, err = overrides.NewOverrides(overrides.Config{
		Defaults: overrides.Overrides{
			Read: overrides.ReadOverrides{
				MaxSearchDuration: model.Duration(time.Minute),
			},
		},
	}, nil, prometheus.DefaultRegisterer)
	require.NoError(t, err)

	sharder = newAsyncSearchSharder(&mockReader{}, o, SearchSharderConfig{
		ConcurrentRequests:    defaultConcurrentRequests,
		TargetBytesPerRequest: defaultTargetBytesPerRequest,
		MostRecentShards:      defaultMostRecentShards,
		MaxDuration:           5 * time.Minute,
	}, log.NewNopLogger())
	testRT = sharder.Wrap(next)

	req = httptest.NewRequest("GET", "/?start=1000&end=1500", nil)
	req = req.WithContext(user.InjectOrgID(req.Context(), "blerg"))
	resp, err = testRT.RoundTrip(pipeline.NewHTTPRequest(req))
	testBadRequestFromResponses(t, resp, err, "range specified by start and end exceeds 1m0s. received start=1000 end=1500")
}

func testBadRequestFromResponses(t *testing.T, resp pipeline.Responses[combiner.PipelineResponse], err error, expectedBody string) {
	require.NoError(t, err)

	r, done, err := resp.Next(context.Background())
	require.NoError(t, err)
	require.True(t, done) // there should only be one response

	testBadRequest(t, r.HTTPResponse(), err, expectedBody)
}

func testBadRequest(t *testing.T, resp *http.Response, err error, expectedBody string) {
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Nil(t, err)
	buff, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.Equal(t, expectedBody, string(buff))
}

func TestAdjustLimit(t *testing.T) {
	l, err := adjustLimit(0, 10, 0)
	require.Equal(t, uint32(10), l)
	require.NoError(t, err)

	l, err = adjustLimit(3, 10, 0)
	require.Equal(t, uint32(3), l)
	require.NoError(t, err)

	l, err = adjustLimit(3, 10, 20)
	require.Equal(t, uint32(3), l)
	require.NoError(t, err)

	l, err = adjustLimit(25, 10, 20)
	require.Equal(t, uint32(0), l)
	require.EqualError(t, err, "limit 25 exceeds max limit 20")
}

func TestMaxDuration(t *testing.T) {
	//
	o, err := overrides.NewOverrides(overrides.Config{}, nil, prometheus.DefaultRegisterer)
	require.NoError(t, err)
	sharder := asyncSearchSharder{
		cfg: SearchSharderConfig{
			MaxDuration: 5 * time.Minute,
		},
		overrides: o,
	}
	actual := sharder.maxDuration("test")
	assert.Equal(t, 5*time.Minute, actual)

	o, err = overrides.NewOverrides(overrides.Config{
		Defaults: overrides.Overrides{
			Read: overrides.ReadOverrides{
				MaxSearchDuration: model.Duration(10 * time.Minute),
			},
		},
	}, nil, prometheus.DefaultRegisterer)
	require.NoError(t, err)
	sharder = asyncSearchSharder{
		cfg: SearchSharderConfig{
			MaxDuration: 5 * time.Minute,
		},
		overrides: o,
	}
	actual = sharder.maxDuration("test")
	assert.Equal(t, 10*time.Minute, actual)
}

func TestHashTraceQLQuery(t *testing.T) {
	// exact same queries should have the same hash
	h1 := hashForSearchRequest(&tempopb.SearchRequest{Query: "{ span.foo = `bar` }"})
	h2 := hashForSearchRequest(&tempopb.SearchRequest{Query: "{ span.foo = `bar` }"})
	require.Equal(t, h1, h2)

	// equivalent queries should have the same hash
	h1 = hashForSearchRequest(&tempopb.SearchRequest{Query: "{ span.foo = `bar`     }"})
	h2 = hashForSearchRequest(&tempopb.SearchRequest{Query: "{ span.foo = `bar` }"})
	require.Equal(t, h1, h2)

	h1 = hashForSearchRequest(&tempopb.SearchRequest{Query: "{ (span.foo = `bar`) || (span.bar = `foo`) }"})
	h2 = hashForSearchRequest(&tempopb.SearchRequest{Query: "{ span.foo = `bar` || span.bar = `foo` }"})
	require.Equal(t, h1, h2)

	// different queries should have different hashes
	h1 = hashForSearchRequest(&tempopb.SearchRequest{Query: "{ span.foo = `bar` }"})
	h2 = hashForSearchRequest(&tempopb.SearchRequest{Query: "{ span.foo = `baz` }"})
	require.NotEqual(t, h1, h2)

	// invalid queries should return 0
	h1 = hashForSearchRequest(&tempopb.SearchRequest{Query: "{ span.foo = `bar` "})
	require.Equal(t, uint64(0), h1)

	h1 = hashForSearchRequest(&tempopb.SearchRequest{Query: ""})
	require.Equal(t, uint64(0), h1)

	// same queries with different spss and limit should have the different hash
	h1 = hashForSearchRequest(&tempopb.SearchRequest{Query: "{ span.foo = `bar` }", Limit: 1})
	h2 = hashForSearchRequest(&tempopb.SearchRequest{Query: "{ span.foo = `bar` }", Limit: 2})
	require.NotEqual(t, h1, h2)

	h1 = hashForSearchRequest(&tempopb.SearchRequest{Query: "{ span.foo = `bar` }", SpansPerSpanSet: 1})
	h2 = hashForSearchRequest(&tempopb.SearchRequest{Query: "{ span.foo = `bar` }", SpansPerSpanSet: 2})
	require.NotEqual(t, h1, h2)
}

func TestBackendShards(t *testing.T) {
	tcs := []struct {
		name      string
		maxShards int
		searchEnd uint32
		expected  []combiner.SearchShards
	}{
		{
			name:      "1 shard, puts all jobs in one shard",
			maxShards: 1,
			searchEnd: 50,
			expected: []combiner.SearchShards{
				{TotalJobs: 8, CompletedThroughSeconds: 1},
			},
		},
		{
			name:      "2 shards, split evenly between",
			maxShards: 2,
			searchEnd: 50,
			expected: []combiner.SearchShards{
				{TotalJobs: 4, CompletedThroughSeconds: 30},
				{TotalJobs: 4, CompletedThroughSeconds: 1},
			},
		},
		{
			name:      "3 shards, one for each block",
			maxShards: 3,
			searchEnd: 50,
			expected: []combiner.SearchShards{
				{TotalJobs: 2, CompletedThroughSeconds: 40},
				{TotalJobs: 2, CompletedThroughSeconds: 30},
				{TotalJobs: 4, CompletedThroughSeconds: 1},
			},
		},
		{
			name:      "4 shards, one for each block",
			maxShards: 4,
			searchEnd: 50,
			expected: []combiner.SearchShards{
				{TotalJobs: 2, CompletedThroughSeconds: 40},
				{TotalJobs: 2, CompletedThroughSeconds: 30},
				{TotalJobs: 2, CompletedThroughSeconds: 20},
				{TotalJobs: 2, CompletedThroughSeconds: 1},
			},
		},
		{
			name:      "5 shards, one for each block",
			maxShards: 5,
			searchEnd: 50,
			expected: []combiner.SearchShards{
				{TotalJobs: 2, CompletedThroughSeconds: 40},
				{TotalJobs: 2, CompletedThroughSeconds: 30},
				{TotalJobs: 2, CompletedThroughSeconds: 20},
				{TotalJobs: 2, CompletedThroughSeconds: 10},
			},
		},
		{
			name:      "4 shards, search end forces 2 blocks in the first shard",
			maxShards: 4,
			searchEnd: 35,
			expected: []combiner.SearchShards{
				{TotalJobs: 4, CompletedThroughSeconds: 30},
				{TotalJobs: 2, CompletedThroughSeconds: 20},
				{TotalJobs: 2, CompletedThroughSeconds: 10},
			},
		},
		{
			name:      "4 shards, search end forces 3 blocks in the first shard",
			maxShards: 4,
			searchEnd: 25,
			expected: []combiner.SearchShards{
				{TotalJobs: 6, CompletedThroughSeconds: 20},
				{TotalJobs: 2, CompletedThroughSeconds: 10},
			},
		},
		{
			name:      "2 shards, search end forces 2 blocks in the first shard",
			maxShards: 2,
			searchEnd: 35,
			expected: []combiner.SearchShards{
				{TotalJobs: 4, CompletedThroughSeconds: 30},
				{TotalJobs: 4, CompletedThroughSeconds: 1},
			},
		},
	}

	// create 4 metas with 2 records each for all the above test cases to use. 8 jobs total
	metas := make([]*backend.BlockMeta, 0, 4)
	for i := 0; i < 4; i++ {
		metas = append(metas, &backend.BlockMeta{
			StartTime:    time.Unix(int64(i*10), 0),        // block 0 starts at 0
			EndTime:      time.Unix(int64(i*10)+10, 0),     // block 0 ends a 10
			Size_:        defaultTargetBytesPerRequest * 2, // 2 jobs per block
			TotalRecords: 2,
			BlockID:      backend.MustParse("00000000-0000-0000-0000-000000000000"),
		})
	}

	// sort
	sort.Slice(metas, func(i, j int) bool {
		return metas[i].EndTime.After(metas[j].EndTime)
	})

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			fn := backendJobsFunc(metas, defaultTargetBytesPerRequest, tc.maxShards, tc.searchEnd)
			actualShards := []combiner.SearchShards{}

			fn(func(jobs int, _ uint64, completedThroughTime uint32) {
				actualShards = append(actualShards, combiner.SearchShards{
					TotalJobs:               uint32(jobs),
					CompletedThroughSeconds: completedThroughTime,
				})
			}, nil)

			assert.Equal(t, tc.expected, actualShards)
		})
	}
}

func urisEqual(t *testing.T, expectedURIs, actualURIs []string) {
	require.Equal(t, len(expectedURIs), len(actualURIs))

	for i, expected := range expectedURIs {
		actual := actualURIs[i]

		e, err := url.Parse(expected)
		require.NoError(t, err)
		a, err := url.Parse(actual)
		require.NoError(t, err)

		e.RawQuery = e.Query().Encode()
		a.RawQuery = a.Query().Encode()

		assert.Equal(t, e, a)
	}
}
