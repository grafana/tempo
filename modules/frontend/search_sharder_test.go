package frontend

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/google/uuid"
	"github.com/grafana/dskit/user"
	"github.com/grafana/tempo/modules/frontend/pipeline"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/traceql"
	"github.com/grafana/tempo/tempodb"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/blocklist"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var _ tempodb.Reader = (*mockReader)(nil)

// implements tempodb.Reader interface
type mockReader struct {
	metas []*backend.BlockMeta
}

func (m *mockReader) SearchTags(context.Context, *backend.BlockMeta, string, common.SearchOptions) (*tempopb.SearchTagsResponse, error) {
	return nil, nil
}

func (m *mockReader) SearchTagValues(context.Context, *backend.BlockMeta, string, common.SearchOptions) ([]string, error) {
	return nil, nil
}

func (m *mockReader) SearchTagsV2(context.Context, *backend.BlockMeta, []string, common.SearchOptions) (*tempopb.SearchTagsV2Response, error) {
	return nil, nil
}

func (m *mockReader) SearchTagValuesV2(context.Context, *backend.BlockMeta, *tempopb.SearchTagValuesRequest, common.SearchOptions) (*tempopb.SearchTagValuesV2Response, error) {
	return nil, nil
}

func (m *mockReader) FetchTagValues(context.Context, *backend.BlockMeta, traceql.AutocompleteRequest, traceql.AutocompleteCallback, common.SearchOptions) error {
	return nil
}

func (m *mockReader) Find(context.Context, string, common.ID, string, string, int64, int64, common.SearchOptions) ([]*tempopb.Trace, []error, error) {
	return nil, nil, nil
}

func (m *mockReader) BlockMetas(string) []*backend.BlockMeta {
	return m.metas
}

func (m *mockReader) Search(context.Context, *backend.BlockMeta, *tempopb.SearchRequest, common.SearchOptions) (*tempopb.SearchResponse, error) {
	return nil, nil
}

func (m *mockReader) Fetch(context.Context, *backend.BlockMeta, traceql.FetchSpansRequest, common.SearchOptions) (traceql.FetchSpansResponse, error) {
	return traceql.FetchSpansResponse{}, nil
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
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000000"),
				},
			},
			expectedURIs: []string{},
		},
		// block with no records
		{
			metas: []*backend.BlockMeta{
				{
					Size:    1000,
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000000"),
				},
			},
			expectedURIs: []string{},
		},
		// meta.json fields
		{
			targetBytesPerRequest: 1000,
			metas: []*backend.BlockMeta{
				{
					Size:          1000,
					TotalRecords:  100,
					BlockID:       uuid.MustParse("00000000-0000-0000-0000-000000000000"),
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
					Size:          1000,
					TotalRecords:  10,
					BlockID:       uuid.MustParse("00000000-0000-0000-0000-000000000000"),
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
					Size:         1000,
					TotalRecords: 3,
					BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000000"),
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
					Size:         1000,
					TotalRecords: 100,
					BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000000"),
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
					Size:         1000,
					TotalRecords: 100,
					BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000000"),
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
					Size:         1000,
					TotalRecords: 100,
					BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000000"),
				},
				{
					Size:         1000,
					TotalRecords: 200,
					BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000001"),
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
		reqCh := make(chan *http.Request)

		go func() {
			buildBackendRequests(ctx, "test", req, searchReq, tc.metas, tc.targetBytesPerRequest, reqCh, cancelCause)
		}()

		actualURIs := []string{}
		for r := range reqCh {
			if r != nil {
				actualURIs = append(actualURIs, r.RequestURI)
			}
		}

		assert.NoError(t, ctx.Err())
		assert.Equal(t, tc.expectedURIs, actualURIs)
	}
}

func TestBackendRequests(t *testing.T) {
	bm := backend.NewBlockMeta("test", uuid.New(), "wdwad", backend.EncGZIP, "asdf")
	bm.StartTime = time.Unix(100, 0)
	bm.EndTime = time.Unix(200, 0)
	bm.Size = defaultTargetBytesPerRequest * 2
	bm.TotalRecords = 2

	s := &asyncSearchSharder{
		cfg:    SearchSharderConfig{},
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
			reqCh := make(chan *http.Request)

			ctx, cancelCause := context.WithCancelCause(context.Background())

			jobs, blocks, blockBytes := s.backendRequests(ctx, "test", r, searchReq, reqCh, cancelCause)
			require.Equal(t, tc.expectedJobs, jobs)
			require.Equal(t, tc.expectedBlocks, blocks)
			require.Equal(t, tc.expectedBlockBytes, blockBytes)

			actualReqURIs := []string{}
			for r := range reqCh {
				if r != nil {
					actualReqURIs = append(actualReqURIs, r.RequestURI)
				}
			}
			require.NoError(t, ctx.Err())
			require.Equal(t, tc.expectedReqsURIs, actualReqURIs)
		})
	}
}

func TestIngesterRequest(t *testing.T) {
	now := int(time.Now().Unix())
	tenMinutesAgo := int(time.Now().Add(-10 * time.Minute).Unix())
	fifteenMinutesAgo := int(time.Now().Add(-15 * time.Minute).Unix())
	twentyMinutesAgo := int(time.Now().Add(-20 * time.Minute).Unix())

	tests := []struct {
		request             string
		queryIngestersUntil time.Duration
		expectedURI         string
		expectedError       error
	}{
		// start/end is outside queryIngestersUntil
		{
			request:             "/?tags=foo%3Dbar&minDuration=10ms&maxDuration=30ms&limit=50&start=10&end=20",
			queryIngestersUntil: 10 * time.Minute,
			expectedURI:         "",
		},
		// start/end is inside queryBackendAfter
		{
			request:             "/?tags=foo%3Dbar&minDuration=10ms&maxDuration=30ms&limit=50&start=" + strconv.Itoa(tenMinutesAgo) + "&end=" + strconv.Itoa(now),
			queryIngestersUntil: 30 * time.Minute,
			expectedURI:         "/querier?end=" + strconv.Itoa(now) + "&limit=50&maxDuration=30ms&minDuration=10ms&spss=3&start=" + strconv.Itoa(tenMinutesAgo) + "&tags=foo%3Dbar",
		},
		// backendAfter/ingsetersUntil = 0 results in no ingester query
		{
			request:             "/?tags=foo%3Dbar&minDuration=10ms&maxDuration=30ms&limit=50&start=" + strconv.Itoa(tenMinutesAgo) + "&end=" + strconv.Itoa(now),
			queryIngestersUntil: 0,
			expectedURI:         "",
		},
		// start/end = 20 - 10 mins ago - break across query ingesters until
		//  ingester start/End = 15 - 10 mins ago
		{
			request:             "/?tags=foo%3Dbar&minDuration=10ms&maxDuration=30ms&limit=50&start=" + strconv.Itoa(twentyMinutesAgo) + "&end=" + strconv.Itoa(tenMinutesAgo),
			queryIngestersUntil: 15 * time.Minute,
			expectedURI:         "/querier?end=" + strconv.Itoa(tenMinutesAgo) + "&limit=50&maxDuration=30ms&minDuration=10ms&spss=3&start=" + strconv.Itoa(fifteenMinutesAgo) + "&tags=foo%3Dbar",
		},
		// start/end = 10 - now mins ago - break across query backend after
		//  ingester start/End = 10 - now mins ago
		//  backend start/End = 15 - 10 mins ago
		{
			request:             "/?tags=foo%3Dbar&minDuration=10ms&maxDuration=30ms&limit=50&start=" + strconv.Itoa(tenMinutesAgo) + "&end=" + strconv.Itoa(now),
			queryIngestersUntil: 15 * time.Minute,
			expectedURI:         "/querier?end=" + strconv.Itoa(now) + "&limit=50&maxDuration=30ms&minDuration=10ms&spss=3&start=" + strconv.Itoa(tenMinutesAgo) + "&tags=foo%3Dbar",
		},
		// start/end = 20 - now mins ago - break across both query ingesters until and backend after
		//  ingester start/End = 15 - now mins ago
		//  backend start/End = 20 - 5 mins ago
		{
			request:             "/?tags=foo%3Dbar&minDuration=10ms&maxDuration=30ms&limit=50&start=" + strconv.Itoa(twentyMinutesAgo) + "&end=" + strconv.Itoa(now),
			queryIngestersUntil: 15 * time.Minute,
			expectedURI:         "/querier?end=" + strconv.Itoa(now) + "&limit=50&maxDuration=30ms&minDuration=10ms&spss=3&start=" + strconv.Itoa(fifteenMinutesAgo) + "&tags=foo%3Dbar",
		},
		{
			request:             "/?tags=foo%3Dbar&minDuration=10ms&maxDuration=30ms&limit=50",
			queryIngestersUntil: 15 * time.Minute,
			expectedURI:         "/querier?end=0&limit=50&maxDuration=30ms&minDuration=10ms&spss=3&start=0&tags=foo%3Dbar",
		},
		{
			request:             "/?limit=50",
			queryIngestersUntil: 15 * time.Minute,
			expectedURI:         "/querier?end=0&limit=50&spss=3&start=0",
		},
	}

	for _, tc := range tests {
		s := &asyncSearchSharder{
			cfg: SearchSharderConfig{
				QueryIngestersUntil: tc.queryIngestersUntil,
			},
		}
		req := httptest.NewRequest("GET", tc.request, nil)

		searchReq, err := api.ParseSearchRequest(req)
		require.NoError(t, err)

		copyReq := searchReq
		actualReq, err := s.ingesterRequest(context.Background(), "test", req, *searchReq)
		if tc.expectedError != nil {
			assert.Equal(t, tc.expectedError, err)
			continue
		}
		assert.NoError(t, err)
		if tc.expectedURI == "" {
			assert.Nil(t, actualReq)
		} else {
			assert.Equal(t, tc.expectedURI, actualReq.RequestURI)
		}

		// it may seem odd to test that the searchReq is not modified, but this is to prevent an issue that
		// occurs if the ingesterRequest method is changed to take a searchReq pointer
		require.True(t, reflect.DeepEqual(copyReq, searchReq))
	}
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
	next := pipeline.AsyncRoundTripperFunc[*http.Response](func(r *http.Request) (pipeline.Responses[*http.Response], error) {
		resString, err := (&jsonpb.Marshaler{}).MarshalToString(&tempopb.SearchResponse{
			Metrics: &tempopb.SearchMetrics{},
		})
		require.NoError(t, err)

		return pipeline.NewSyncToAsyncResponse(&http.Response{
			Body:       io.NopCloser(strings.NewReader(resString)),
			StatusCode: 200,
		}), nil
	})

	o, err := overrides.NewOverrides(overrides.Config{}, prometheus.DefaultRegisterer)
	require.NoError(t, err)

	now := time.Now().Add(-10 * time.Minute).Unix()

	sharder := newAsyncSearchSharder(&mockReader{
		metas: []*backend.BlockMeta{ // one block with 2 records that are each the target bytes per request will force 2 sub queries
			{
				StartTime:    time.Unix(now, 0),
				EndTime:      time.Unix(now, 0),
				Size:         defaultTargetBytesPerRequest * 2,
				TotalRecords: 2,
				BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000000"),
			},
		},
	}, o, SearchSharderConfig{
		QueryIngestersUntil:   15 * time.Minute,
		ConcurrentRequests:    1, // 1 concurrent request to force order
		TargetBytesPerRequest: defaultTargetBytesPerRequest,
	}, log.NewNopLogger())
	testRT := sharder.Wrap(next)

	path := fmt.Sprintf("/?start=%d&end=%d", now-1, now+1)
	req := httptest.NewRequest("GET", path, nil)
	ctx := req.Context()
	ctx = user.InjectOrgID(ctx, "blerg")
	req = req.WithContext(ctx)

	resps, err := testRT.RoundTrip(req)
	require.NoError(t, err)
	// find a response with total jobs > . this is the metadata response
	var resp *tempopb.SearchResponse
	for {
		r, done, err := resps.Next(context.Background())
		require.NoError(t, err)
		require.Equal(t, 200, r.StatusCode)

		actualResp := &tempopb.SearchResponse{}
		bytesResp, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		err = jsonpb.Unmarshal(bytes.NewReader(bytesResp), actualResp)
		require.NoError(t, err)

		if actualResp.Metrics.TotalJobs > 0 {
			resp = actualResp
			break
		}

		require.False(t, done)
	}

	// 2 jobs for the meta + 1 for th ingester
	assert.Equal(t, uint32(3), resp.Metrics.TotalJobs)
}

func TestSearchSharderRoundTripBadRequest(t *testing.T) {
	next := pipeline.AsyncRoundTripperFunc[*http.Response](func(r *http.Request) (pipeline.Responses[*http.Response], error) {
		return nil, nil
	})

	o, err := overrides.NewOverrides(overrides.Config{}, prometheus.DefaultRegisterer)
	require.NoError(t, err)

	sharder := newAsyncSearchSharder(&mockReader{}, o, SearchSharderConfig{
		ConcurrentRequests:    defaultConcurrentRequests,
		TargetBytesPerRequest: defaultTargetBytesPerRequest,
		MaxDuration:           5 * time.Minute,
	}, log.NewNopLogger())
	testRT := sharder.Wrap(next)

	// no org id
	req := httptest.NewRequest("GET", "/?start=1000&end=1100", nil)
	resp, err := testRT.RoundTrip(req)
	testBadRequestFromResponses(t, resp, err, "no org id")

	// start/end outside of max duration
	req = httptest.NewRequest("GET", "/?start=1000&end=1500", nil)
	req = req.WithContext(user.InjectOrgID(req.Context(), "blerg"))
	resp, err = testRT.RoundTrip(req)
	testBadRequestFromResponses(t, resp, err, "range specified by start and end exceeds 5m0s. received start=1000 end=1500")

	// bad request
	req = httptest.NewRequest("GET", "/?start=asdf&end=1500", nil)
	resp, err = testRT.RoundTrip(req)
	testBadRequestFromResponses(t, resp, err, "invalid start: strconv.ParseInt: parsing \"asdf\": invalid syntax")

	// test max duration error with overrides
	o, err = overrides.NewOverrides(overrides.Config{
		Defaults: overrides.Overrides{
			Read: overrides.ReadOverrides{
				MaxSearchDuration: model.Duration(time.Minute),
			},
		},
	}, prometheus.DefaultRegisterer)
	require.NoError(t, err)

	sharder = newAsyncSearchSharder(&mockReader{}, o, SearchSharderConfig{
		ConcurrentRequests:    defaultConcurrentRequests,
		TargetBytesPerRequest: defaultTargetBytesPerRequest,
		MaxDuration:           5 * time.Minute,
	}, log.NewNopLogger())
	testRT = sharder.Wrap(next)

	req = httptest.NewRequest("GET", "/?start=1000&end=1500", nil)
	req = req.WithContext(user.InjectOrgID(req.Context(), "blerg"))
	resp, err = testRT.RoundTrip(req)
	testBadRequestFromResponses(t, resp, err, "range specified by start and end exceeds 1m0s. received start=1000 end=1500")
}

func testBadRequestFromResponses(t *testing.T, resp pipeline.Responses[*http.Response], err error, expectedBody string) {
	require.NoError(t, err)

	r, done, err := resp.Next(context.Background())
	require.NoError(t, err)
	require.True(t, done) // there should only be one response

	testBadRequest(t, r, err, expectedBody)
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
	o, err := overrides.NewOverrides(overrides.Config{}, prometheus.DefaultRegisterer)
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
	}, prometheus.DefaultRegisterer)
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
	h1 := hashForTraceQLQuery("{ span.foo = `bar` }")
	h2 := hashForTraceQLQuery("{ span.foo = `bar` }")
	require.Equal(t, h1, h2)

	// equivalent queries should have the same hash
	h1 = hashForTraceQLQuery("{ span.foo = `bar`     }")
	h2 = hashForTraceQLQuery("{ span.foo = `bar` }")
	require.Equal(t, h1, h2)

	h1 = hashForTraceQLQuery("{ (span.foo = `bar`) || (span.bar = `foo`) }")
	h2 = hashForTraceQLQuery("{ span.foo = `bar` || span.bar = `foo` }")
	require.Equal(t, h1, h2)

	// different queries should have different hashes
	h1 = hashForTraceQLQuery("{ span.foo = `bar` }")
	h2 = hashForTraceQLQuery("{ span.foo = `baz` }")
	require.NotEqual(t, h1, h2)

	// invalid queries should return 0
	h1 = hashForTraceQLQuery("{ span.foo = `bar` ")
	require.Equal(t, uint64(0), h1)

	h1 = hashForTraceQLQuery("")
	require.Equal(t, uint64(0), h1)
}
