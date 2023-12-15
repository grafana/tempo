package frontend

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/golang/protobuf/jsonpb" //nolint:all deprecated
	"github.com/google/uuid"
	"github.com/grafana/dskit/user"
	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/atomic"

	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/cache"
	"github.com/grafana/tempo/pkg/search"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/traceql"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/blocklist"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

// implements tempodb.Reader interface
type mockReader struct {
	metas []*backend.BlockMeta
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

func TestBuildBackendRequests(t *testing.T) {
	tests := []struct {
		targetBytesPerRequest int
		metas                 []*backend.BlockMeta
		expectedURIs          []string
		expectedError         error
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

		stopCh := make(chan struct{})
		defer close(stopCh)
		reqCh := make(chan *backendReqMsg)

		go func() {
			buildBackendRequests(context.Background(), "test", req, searchReq, tc.metas, tc.targetBytesPerRequest, reqCh, stopCh)
		}()

		actualURIs := []string{}
		var actualErr error
		for r := range reqCh {
			if r.err != nil {
				actualErr = r.err
				break
			}

			if r.req != nil {
				actualURIs = append(actualURIs, r.req.RequestURI)
			}
		}

		if tc.expectedError != nil {
			assert.Equal(t, tc.expectedError, actualErr)
			continue
		}
		assert.NoError(t, actualErr)
		assert.Equal(t, tc.expectedURIs, actualURIs)
	}
}

func TestBackendRequests(t *testing.T) {
	bm := backend.NewBlockMeta("test", uuid.New(), "wdwad", backend.EncGZIP, "asdf")
	bm.StartTime = time.Unix(100, 0)
	bm.EndTime = time.Unix(200, 0)
	bm.Size = defaultTargetBytesPerRequest * 2
	bm.TotalRecords = 2

	s := &searchSharder{
		cfg:    SearchSharderConfig{},
		reader: &mockReader{metas: []*backend.BlockMeta{bm}},
	}

	tests := []struct {
		name               string
		request            string
		expectedReqsURIs   []string
		expectedError      error
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
			expectedError:      nil,
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
			expectedError:      nil,
			expectedJobs:       2,
			expectedBlocks:     1,
			expectedBlockBytes: defaultTargetBytesPerRequest * 2,
		},
		{
			name:             "start and end out of block",
			request:          "/?tags=foo%3Dbar&minDuration=10ms&maxDuration=30ms&limit=50&start=10&end=20",
			expectedReqsURIs: make([]string, 0),
			expectedError:    nil,
		},
		{
			name:             "no start and end",
			request:          "/?tags=foo%3Dbar&minDuration=10ms&maxDuration=30ms&limit=50",
			expectedReqsURIs: make([]string, 0),
			expectedError:    nil,
		},
		{
			name:             "only tags",
			request:          "/?tags=foo%3Dbar",
			expectedReqsURIs: make([]string, 0),
			expectedError:    nil,
		},
		{
			name:             "no params",
			request:          "/",
			expectedReqsURIs: make([]string, 0),
			expectedError:    nil,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", tc.request, nil)
			searchReq, err := api.ParseSearchRequest(r)
			require.NoError(t, err)

			stopCh := make(chan struct{})
			defer close(stopCh)
			reqCh := make(chan *backendReqMsg)

			jobs, blocks, blockBytes := s.backendRequests(context.TODO(), "test", r, searchReq, reqCh, stopCh)
			require.Equal(t, tc.expectedJobs, jobs)
			require.Equal(t, tc.expectedBlocks, blocks)
			require.Equal(t, tc.expectedBlockBytes, blockBytes)

			var actualErr error
			actualReqURIs := []string{}
			for r := range reqCh {
				if r.err != nil {
					actualErr = r.err
				}
				if r.req != nil {
					actualReqURIs = append(actualReqURIs, r.req.RequestURI)
				}
			}
			require.Equal(t, tc.expectedError, actualErr)
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
		s := &searchSharder{
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

		actualStart, actualEnd := backendRange(searchReq, tc.queryBackendAfter)
		assert.Equal(t, int(tc.expectedStart), int(actualStart))
		assert.Equal(t, int(tc.expectedEnd), int(actualEnd))
	}
}

func TestSearchSharderRoundTrip(t *testing.T) {
	tests := []struct {
		name             string
		status1          int
		status2          int
		response1        *tempopb.SearchResponse
		response2        *tempopb.SearchResponse
		err1             error
		err2             error
		expectedStatus   int
		expectedResponse *tempopb.SearchResponse
		expectedError    error
	}{
		{
			name:           "empty returns",
			status1:        200,
			status2:        200,
			expectedStatus: 200,
			response1:      &tempopb.SearchResponse{Metrics: &tempopb.SearchMetrics{}},
			response2:      &tempopb.SearchResponse{Metrics: &tempopb.SearchMetrics{}},
			expectedResponse: &tempopb.SearchResponse{Metrics: &tempopb.SearchMetrics{
				TotalBlocks:     1,
				CompletedJobs:   2,
				TotalJobs:       2,
				TotalBlockBytes: defaultTargetBytesPerRequest * 2,
			}},
		},
		{
			name:           "404+200",
			status1:        404,
			status2:        200,
			response2:      &tempopb.SearchResponse{Metrics: &tempopb.SearchMetrics{}},
			expectedStatus: 500,
		},
		{
			name:           "200+400",
			status1:        200,
			response1:      &tempopb.SearchResponse{Metrics: &tempopb.SearchMetrics{}},
			status2:        400,
			expectedStatus: 500,
		},
		{
			name:           "500+404",
			status1:        500,
			status2:        404,
			expectedStatus: 500,
		},
		{
			name:           "404+500",
			status1:        404,
			status2:        500,
			expectedStatus: 500,
		},
		{
			name:           "500+200",
			status1:        500,
			status2:        200,
			response2:      &tempopb.SearchResponse{Metrics: &tempopb.SearchMetrics{}},
			expectedStatus: 500,
		},
		{
			name:           "200+500",
			status1:        200,
			response1:      &tempopb.SearchResponse{Metrics: &tempopb.SearchMetrics{}},
			status2:        500,
			expectedStatus: 500,
		},
		{
			name:    "200+200",
			status1: 200,
			response1: &tempopb.SearchResponse{
				Traces: []*tempopb.TraceSearchMetadata{
					{
						TraceID:           "1234",
						StartTimeUnixNano: 1,
					},
				},
				Metrics: &tempopb.SearchMetrics{
					InspectedTraces: 1,
					TotalBlocks:     2,
					InspectedBytes:  3,
				},
			},
			status2: 200,
			response2: &tempopb.SearchResponse{
				Traces: []*tempopb.TraceSearchMetadata{
					{
						TraceID:           "5678",
						StartTimeUnixNano: 0,
					},
				},
				Metrics: &tempopb.SearchMetrics{
					InspectedTraces: 5,
					TotalBlocks:     6,
					InspectedBytes:  7,
				},
			},
			expectedStatus: 200,
			expectedResponse: &tempopb.SearchResponse{
				Traces: []*tempopb.TraceSearchMetadata{
					{
						TraceID:           "1234",
						StartTimeUnixNano: 1,
						RootServiceName:   search.RootSpanNotYetReceivedText,
					},
					{
						TraceID:           "5678",
						StartTimeUnixNano: 0,
						RootServiceName:   search.RootSpanNotYetReceivedText,
					},
				},
				Metrics: &tempopb.SearchMetrics{
					InspectedTraces: 6,
					TotalBlocks:     1,
					InspectedBytes:  10,
					CompletedJobs:   2,
					TotalJobs:       2,
					TotalBlockBytes: defaultTargetBytesPerRequest * 2,
				},
			},
		},
		{
			name:          "200+err",
			status1:       200,
			response1:     &tempopb.SearchResponse{Metrics: &tempopb.SearchMetrics{}},
			err2:          errors.New("booo"),
			expectedError: errors.New("booo"),
		},
		{
			name:          "err+200",
			err1:          errors.New("booo"),
			status2:       200,
			response2:     &tempopb.SearchResponse{Metrics: &tempopb.SearchMetrics{}},
			expectedError: errors.New("booo"),
		},
		{
			name:           "500+err",
			status1:        500,
			response1:      &tempopb.SearchResponse{Metrics: &tempopb.SearchMetrics{}},
			err2:           errors.New("booo"),
			expectedStatus: 500,
			expectedError:  errors.New("booo"),
		},
		{
			name:          "err+500",
			err1:          errors.New("booo"),
			status2:       500,
			response2:     &tempopb.SearchResponse{Metrics: &tempopb.SearchMetrics{}},
			expectedError: errors.New("booo"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			next := RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
				var response *tempopb.SearchResponse
				var statusCode int
				var err error

				if strings.Contains(r.RequestURI, "startPage=0") {
					response = tc.response1
					statusCode = tc.status1
					err = tc.err1
				} else {
					response = tc.response2
					err = tc.err2
					statusCode = tc.status2
				}

				if err != nil {
					return nil, err
				}

				var resString string
				if response != nil {
					resString, err = (&jsonpb.Marshaler{}).MarshalToString(response)
					require.NoError(t, err)
				}

				return &http.Response{
					Body:       io.NopCloser(strings.NewReader(resString)),
					StatusCode: statusCode,
				}, nil
			})

			o, err := overrides.NewOverrides(overrides.Config{})
			require.NoError(t, err)

			sharder := newSearchSharder(&mockReader{
				metas: []*backend.BlockMeta{ // one block with 2 records that are each the target bytes per request will force 2 sub queries
					{
						StartTime:    time.Unix(1100, 0),
						EndTime:      time.Unix(1200, 0),
						Size:         defaultTargetBytesPerRequest * 2,
						TotalRecords: 2,
						BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000000"),
					},
				},
			}, o, SearchSharderConfig{
				ConcurrentRequests:    1, // 1 concurrent request to force order
				TargetBytesPerRequest: defaultTargetBytesPerRequest,
			}, newSearchProgress, &frontendCache{}, log.NewNopLogger())
			testRT := NewRoundTripper(next, sharder)

			req := httptest.NewRequest("GET", "/?start=1000&end=1500", nil)
			ctx := req.Context()
			ctx = user.InjectOrgID(ctx, "blerg")
			req = req.WithContext(ctx)

			resp, err := testRT.RoundTrip(req)
			if tc.expectedError != nil {
				assert.Equal(t, tc.expectedError, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.expectedStatus, resp.StatusCode)
			if tc.expectedStatus == http.StatusOK {
				assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
			}
			if tc.expectedResponse != nil {
				actualResp := &tempopb.SearchResponse{}
				bytesResp, err := io.ReadAll(resp.Body)
				require.NoError(t, err)
				err = jsonpb.Unmarshal(bytes.NewReader(bytesResp), actualResp)
				require.NoError(t, err)

				// We don't need to check on this metric
				// actualResp.Metrics.TotalBlockBytes = 0

				assert.Equal(t, tc.expectedResponse, actualResp)
			}
		})
	}
}

func TestTotalJobsIncludesIngester(t *testing.T) {
	next := RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
		resString, err := (&jsonpb.Marshaler{}).MarshalToString(&tempopb.SearchResponse{
			Metrics: &tempopb.SearchMetrics{},
		})
		require.NoError(t, err)

		return &http.Response{
			Body:       io.NopCloser(strings.NewReader(resString)),
			StatusCode: 200,
		}, nil
	})

	o, err := overrides.NewOverrides(overrides.Config{})
	require.NoError(t, err)

	now := time.Now().Add(-10 * time.Minute).Unix()

	sharder := newSearchSharder(&mockReader{
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
	}, newSearchProgress, &frontendCache{}, log.NewNopLogger())
	testRT := NewRoundTripper(next, sharder)

	path := fmt.Sprintf("/?start=%d&end=%d", now-1, now+1)
	req := httptest.NewRequest("GET", path, nil)
	ctx := req.Context()
	ctx = user.InjectOrgID(ctx, "blerg")
	req = req.WithContext(ctx)

	resp, err := testRT.RoundTrip(req)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode)

	actualResp := &tempopb.SearchResponse{}
	bytesResp, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	err = jsonpb.Unmarshal(bytes.NewReader(bytesResp), actualResp)
	require.NoError(t, err)

	// 2 jobs for the meta + 1 for th ingester
	assert.Equal(t, uint32(3), actualResp.Metrics.TotalJobs)
}

func TestSearchSharderRoundTripBadRequest(t *testing.T) {
	next := RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
		return nil, nil
	})

	o, err := overrides.NewOverrides(overrides.Config{})
	require.NoError(t, err)

	sharder := newSearchSharder(&mockReader{}, o, SearchSharderConfig{
		ConcurrentRequests:    defaultConcurrentRequests,
		TargetBytesPerRequest: defaultTargetBytesPerRequest,
		MaxDuration:           5 * time.Minute,
	}, newSearchProgress, &frontendCache{}, log.NewNopLogger())
	testRT := NewRoundTripper(next, sharder)

	// no org id
	req := httptest.NewRequest("GET", "/?start=1000&end=1100", nil)
	resp, err := testRT.RoundTrip(req)
	testBadRequest(t, resp, err, "no org id")

	// start/end outside of max duration
	req = httptest.NewRequest("GET", "/?start=1000&end=1500", nil)
	req = req.WithContext(user.InjectOrgID(req.Context(), "blerg"))
	resp, err = testRT.RoundTrip(req)
	testBadRequest(t, resp, err, "range specified by start and end exceeds 5m0s. received start=1000 end=1500")

	// bad request
	req = httptest.NewRequest("GET", "/?start=asdf&end=1500", nil)
	resp, err = testRT.RoundTrip(req)
	testBadRequest(t, resp, err, "invalid start: strconv.ParseInt: parsing \"asdf\": invalid syntax")

	// test max duration error with overrides
	o, err = overrides.NewOverrides(overrides.Config{
		Defaults: overrides.Overrides{
			Read: overrides.ReadOverrides{
				MaxSearchDuration: model.Duration(time.Minute),
			},
		},
	})
	require.NoError(t, err)

	sharder = newSearchSharder(&mockReader{}, o, SearchSharderConfig{
		ConcurrentRequests:    defaultConcurrentRequests,
		TargetBytesPerRequest: defaultTargetBytesPerRequest,
		MaxDuration:           5 * time.Minute,
	}, newSearchProgress, &frontendCache{}, log.NewNopLogger())
	testRT = NewRoundTripper(next, sharder)

	req = httptest.NewRequest("GET", "/?start=1000&end=1500", nil)
	req = req.WithContext(user.InjectOrgID(req.Context(), "blerg"))
	resp, err = testRT.RoundTrip(req)
	testBadRequest(t, resp, err, "range specified by start and end exceeds 1m0s. received start=1000 end=1500")
}

func TestSharderAccessesCache(t *testing.T) {
	next := RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
		resString, err := (&jsonpb.Marshaler{}).MarshalToString(&tempopb.SearchResponse{
			Traces: []*tempopb.TraceSearchMetadata{{
				TraceID:         util.TraceIDToHexString(test.ValidTraceID(nil)),
				RootServiceName: "test",
				RootTraceName:   "bar",
			}},
			Metrics: &tempopb.SearchMetrics{
				InspectedBytes: 4,
			},
		})
		require.NoError(t, err)

		return &http.Response{
			Body:       io.NopCloser(strings.NewReader(resString)),
			StatusCode: 200,
		}, nil
	})

	// setup mock cache
	c := cache.NewMockCache()

	o, err := overrides.NewOverrides(overrides.Config{})
	require.NoError(t, err)

	meta := &backend.BlockMeta{
		StartTime:    time.Unix(15, 0),
		EndTime:      time.Unix(16, 0),
		Size:         defaultTargetBytesPerRequest,
		TotalRecords: 1,
		BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000123"),
	}

	// setup sharder
	sharder := newSearchSharder(&mockReader{
		metas: []*backend.BlockMeta{meta},
	}, o, SearchSharderConfig{
		QueryIngestersUntil:   15 * time.Minute,
		ConcurrentRequests:    1, // 1 concurrent request to force order
		TargetBytesPerRequest: defaultTargetBytesPerRequest,
	}, newSearchProgress, &frontendCache{
		c: c,
	}, log.NewNopLogger())
	testRT := NewRoundTripper(next, sharder)

	// setup query
	query := "{}"
	hash := hashForTraceQLQuery(query)
	start := uint32(10)
	end := uint32(20)
	cacheKey := cacheKeyForJob(hash, &tempopb.SearchRequest{Start: start, End: end}, meta, 0, 1)

	// confirm cache key coesn't exist
	_, bufs, _ := c.Fetch(context.Background(), []string{cacheKey})
	require.Equal(t, 0, len(bufs))

	// execute query
	path := fmt.Sprintf("/?start=%d&end=%d&q=%s", start, end, query) // encapsulates block above
	req := httptest.NewRequest("GET", path, nil)
	ctx := req.Context()
	ctx = user.InjectOrgID(ctx, "blerg")
	req = req.WithContext(ctx)

	resp, err := testRT.RoundTrip(req)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode)

	actualResp := &tempopb.SearchResponse{}
	bytesResp, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	err = jsonpb.Unmarshal(bytes.NewReader(bytesResp), actualResp)
	require.NoError(t, err)

	// confirm cache key exists and matches the response above
	_, bufs, _ = c.Fetch(context.Background(), []string{cacheKey})
	require.Equal(t, 1, len(bufs))

	actualCache := &tempopb.SearchResponse{}
	err = jsonpb.Unmarshal(bytes.NewReader(bufs[0]), actualCache)
	require.NoError(t, err)

	// zeroing these out b/c they are set by the sharder and won't be in cache
	cacheResponsesEqual(t, actualCache, actualResp)

	// now let's "poison" cache by writing different values directly and confirm
	// the sharder returns them
	overwriteResp := &tempopb.SearchResponse{
		Traces: []*tempopb.TraceSearchMetadata{{
			TraceID:         util.TraceIDToHexString(test.ValidTraceID(nil)),
			RootServiceName: "test2",
			RootTraceName:   "bar2",
		}},
		Metrics: &tempopb.SearchMetrics{
			InspectedBytes: 11,
		},
	}
	overwriteString, err := (&jsonpb.Marshaler{}).MarshalToString(overwriteResp)
	require.NoError(t, err)

	c.Store(context.Background(), []string{cacheKey}, [][]byte{[]byte(overwriteString)})

	resp, err = testRT.RoundTrip(req)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode)

	actualResp = &tempopb.SearchResponse{}
	bytesResp, err = io.ReadAll(resp.Body)
	require.NoError(t, err)
	err = jsonpb.Unmarshal(bytes.NewReader(bytesResp), actualResp)
	require.NoError(t, err)

	cacheResponsesEqual(t, overwriteResp, actualResp)
}

func cacheResponsesEqual(t *testing.T, cacheResponse *tempopb.SearchResponse, pipelineResp *tempopb.SearchResponse) {
	// zeroing these out b/c they are set by the sharder and won't be in cache
	pipelineResp.Metrics.TotalJobs = 0
	pipelineResp.Metrics.CompletedJobs = 0
	pipelineResp.Metrics.TotalBlockBytes = 0
	pipelineResp.Metrics.TotalBlocks = 0

	require.Equal(t, pipelineResp, cacheResponse)
}

func testBadRequest(t *testing.T, resp *http.Response, err error, expectedBody string) {
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Nil(t, err)
	buff, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.Equal(t, expectedBody, string(buff))
}

func TestAdjustLimit(t *testing.T) {
	assert.Equal(t, uint32(10), adjustLimit(0, 10, 0))
	assert.Equal(t, uint32(3), adjustLimit(3, 10, 0))
	assert.Equal(t, uint32(3), adjustLimit(3, 10, 20))
	assert.Equal(t, uint32(20), adjustLimit(25, 10, 20))
}

func TestMaxDuration(t *testing.T) {
	//
	o, err := overrides.NewOverrides(overrides.Config{})
	require.NoError(t, err)
	sharder := searchSharder{
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
	})
	require.NoError(t, err)
	sharder = searchSharder{
		cfg: SearchSharderConfig{
			MaxDuration: 5 * time.Minute,
		},
		overrides: o,
	}
	actual = sharder.maxDuration("test")
	assert.Equal(t, 10*time.Minute, actual)
}

func TestSubRequestsCancelled(t *testing.T) {
	totalJobs := 5

	wg := sync.WaitGroup{}
	nextSuccess := RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
		wg.Done()
		wg.Wait()

		resString, err := (&jsonpb.Marshaler{}).MarshalToString(&tempopb.SearchResponse{
			Traces: []*tempopb.TraceSearchMetadata{{
				TraceID: test.RandomString(),
			}},
			Metrics: &tempopb.SearchMetrics{},
		})
		require.NoError(t, err)

		return &http.Response{
			Body:       io.NopCloser(strings.NewReader(resString)),
			StatusCode: 200,
		}, nil
	})

	nextErr := RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
		wg.Done()
		wg.Wait()

		return nil, fmt.Errorf("error")
	})

	next500 := RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
		wg.Done()
		wg.Wait()

		return &http.Response{
			Body:       io.NopCloser(strings.NewReader("")),
			StatusCode: 500,
		}, nil
	})

	nextRequireCancelled := RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
		wg.Done()

		ctx := r.Context()

		select {
		case <-ctx.Done():
		case <-time.After(1 * time.Second):
		}

		if ctx.Err() == nil {
			return nil, fmt.Errorf("context should have been cancelled")
		}

		// check and see if there's an error
		return httptest.NewRecorder().Result(), nil
	})

	o, err := overrides.NewOverrides(overrides.Config{})
	require.NoError(t, err)

	sharder := newSearchSharder(&mockReader{
		metas: []*backend.BlockMeta{
			{
				StartTime:    time.Unix(1100, 0),
				EndTime:      time.Unix(1200, 0),
				Size:         uint64(defaultTargetBytesPerRequest * totalJobs),
				TotalRecords: uint32(totalJobs),
				BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000000"),
			},
		},
	}, o, SearchSharderConfig{
		ConcurrentRequests:    10,
		TargetBytesPerRequest: defaultTargetBytesPerRequest,
		DefaultLimit:          2,
	}, newSearchProgress, &frontendCache{}, log.NewNopLogger())

	// return some things and assert the right subrequests are cancelled
	// 500, err, limit
	tcs := []struct {
		name  string
		nexts []RoundTripperFunc
	}{
		{
			name:  "success",
			nexts: []RoundTripperFunc{nextSuccess},
		},
		{
			name:  "two successes -> reach limit",
			nexts: []RoundTripperFunc{nextSuccess, nextSuccess, nextRequireCancelled, nextRequireCancelled, nextRequireCancelled},
		},
		{
			name:  "one errors",
			nexts: []RoundTripperFunc{nextErr, nextRequireCancelled, nextRequireCancelled, nextRequireCancelled, nextRequireCancelled},
		},
		{
			name:  "one 500s",
			nexts: []RoundTripperFunc{next500, nextRequireCancelled, nextRequireCancelled, nextRequireCancelled, nextRequireCancelled},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			prev := atomic.NewInt32(0)

			// create a next function that round robins through the nexts.
			nextRR := RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
				next := tc.nexts[prev.Load()%int32(len(tc.nexts))]
				prev.Inc()
				return next.RoundTrip(r)
			})

			testRT := NewRoundTripper(nextRR, sharder)

			// all requests will create totalJobs subrequests. let's reset the wg here
			wg.Add(totalJobs)

			req := httptest.NewRequest("GET", "/?start=1000&end=1500", nil)
			ctx := req.Context()
			ctx = user.InjectOrgID(ctx, "blerg")
			req = req.WithContext(ctx)

			_, _ = testRT.RoundTrip(req)
		})
	}
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

func TestCacheKeyForJob(t *testing.T) {
	tcs := []struct {
		name          string
		queryHash     uint64
		req           *tempopb.SearchRequest
		meta          *backend.BlockMeta
		searchPage    int
		pagesToSearch int

		expected string
	}{
		{
			name:      "valid!",
			queryHash: 42,
			req: &tempopb.SearchRequest{
				Start: 10,
				End:   20,
			},
			meta: &backend.BlockMeta{
				BlockID:   uuid.MustParse("00000000-0000-0000-0000-000000000123"),
				StartTime: time.Unix(15, 0),
				EndTime:   time.Unix(16, 0),
			},
			searchPage:    1,
			pagesToSearch: 2,
			expected:      "sj:42:00000000-0000-0000-0000-000000000123:1:2",
		},
		{
			name:      "no query hash means no query cache",
			queryHash: 0,
			req: &tempopb.SearchRequest{
				Start: 10,
				End:   20,
			},
			meta: &backend.BlockMeta{
				BlockID:   uuid.MustParse("00000000-0000-0000-0000-000000000123"),
				StartTime: time.Unix(15, 0),
				EndTime:   time.Unix(16, 0),
			},
			searchPage:    1,
			pagesToSearch: 2,
			expected:      "",
		},
		{
			name:      "meta before start time",
			queryHash: 42,
			req: &tempopb.SearchRequest{
				Start: 10,
				End:   20,
			},
			meta: &backend.BlockMeta{
				BlockID:   uuid.MustParse("00000000-0000-0000-0000-000000000123"),
				StartTime: time.Unix(5, 0),
				EndTime:   time.Unix(6, 0),
			},
			searchPage:    1,
			pagesToSearch: 2,
			expected:      "",
		},
		{
			name:      "meta overlaps search start",
			queryHash: 42,
			req: &tempopb.SearchRequest{
				Start: 10,
				End:   20,
			},
			meta: &backend.BlockMeta{
				BlockID:   uuid.MustParse("00000000-0000-0000-0000-000000000123"),
				StartTime: time.Unix(5, 0),
				EndTime:   time.Unix(15, 0),
			},
			searchPage:    1,
			pagesToSearch: 2,
			expected:      "",
		},
		{
			name:      "meta overlaps search end",
			queryHash: 42,
			req: &tempopb.SearchRequest{
				Start: 10,
				End:   20,
			},
			meta: &backend.BlockMeta{
				BlockID:   uuid.MustParse("00000000-0000-0000-0000-000000000123"),
				StartTime: time.Unix(15, 0),
				EndTime:   time.Unix(25, 0),
			},
			searchPage:    1,
			pagesToSearch: 2,
			expected:      "",
		},
		{
			name:      "meta after search range",
			queryHash: 42,
			req: &tempopb.SearchRequest{
				Start: 10,
				End:   20,
			},
			meta: &backend.BlockMeta{
				BlockID:   uuid.MustParse("00000000-0000-0000-0000-000000000123"),
				StartTime: time.Unix(25, 0),
				EndTime:   time.Unix(30, 0),
			},
			searchPage:    1,
			pagesToSearch: 2,
			expected:      "",
		},
		{
			name:      "meta encapsulates search range",
			queryHash: 42,
			req: &tempopb.SearchRequest{
				Start: 10,
				End:   20,
			},
			meta: &backend.BlockMeta{
				BlockID:   uuid.MustParse("00000000-0000-0000-0000-000000000123"),
				StartTime: time.Unix(5, 0),
				EndTime:   time.Unix(30, 0),
			},
			searchPage:    1,
			pagesToSearch: 2,
			expected:      "",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			actual := cacheKeyForJob(tc.queryHash, tc.req, tc.meta, tc.searchPage, tc.pagesToSearch)
			require.Equal(t, tc.expected, actual)
		})
	}
}

func BenchmarkSearchSharderRoundTrip5(b *testing.B)     { benchmarkSearchSharderRoundTrip(b, 5) }
func BenchmarkSearchSharderRoundTrip500(b *testing.B)   { benchmarkSearchSharderRoundTrip(b, 500) }
func BenchmarkSearchSharderRoundTrip50000(b *testing.B) { benchmarkSearchSharderRoundTrip(b, 50000) } // max, forces all queries to run

func benchmarkSearchSharderRoundTrip(b *testing.B, s int32) {
	resString, err := (&jsonpb.Marshaler{}).MarshalToString(&tempopb.SearchResponse{Metrics: &tempopb.SearchMetrics{}})
	require.NoError(b, err)

	successResString, err := (&jsonpb.Marshaler{}).MarshalToString(&tempopb.SearchResponse{
		Traces: []*tempopb.TraceSearchMetadata{
			{
				TraceID: "1234",
			},
		},
		Metrics: &tempopb.SearchMetrics{},
	})
	require.NoError(b, err)

	succeedAfter := atomic.NewInt32(s)
	next := RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
		val := succeedAfter.Dec()

		s := resString
		if val == 0 {
			s = successResString
		}

		return &http.Response{
			Body:       io.NopCloser(strings.NewReader(s)),
			StatusCode: 200,
		}, nil
	})

	o, err := overrides.NewOverrides(overrides.Config{})
	require.NoError(b, err)

	totalMetas := 10000
	jobsPerMeta := 2
	metas := make([]*backend.BlockMeta, 0, totalMetas)
	for i := 0; i < totalMetas; i++ {
		metas = append(metas, &backend.BlockMeta{
			StartTime:    time.Unix(1100, 0),
			EndTime:      time.Unix(1200, 0),
			Size:         defaultTargetBytesPerRequest * uint64(jobsPerMeta),
			TotalRecords: uint32(jobsPerMeta),
			BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000000"),
		})
	}

	sharder := newSearchSharder(&mockReader{
		metas: metas,
	}, o, SearchSharderConfig{
		ConcurrentRequests:    100,
		TargetBytesPerRequest: defaultTargetBytesPerRequest,
	}, newSearchProgress, &frontendCache{}, log.NewNopLogger())
	testRT := NewRoundTripper(next, sharder)

	req := httptest.NewRequest("GET", "/?start=1000&end=1500&limit=1", nil) // limiting to 1 to let succeedAfter work
	ctx := req.Context()
	ctx = user.InjectOrgID(ctx, "blerg")
	req = req.WithContext(ctx)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		succeedAfter = atomic.NewInt32(s)
		_, err = testRT.RoundTrip(req)
		require.NoError(b, err)
	}
}

func BenchmarkCacheKeyForJob(b *testing.B) {
	req := &tempopb.SearchRequest{
		Start: 10,
		End:   20,
	}
	meta := &backend.BlockMeta{
		BlockID:   uuid.MustParse("00000000-0000-0000-0000-000000000123"),
		StartTime: time.Unix(15, 0),
		EndTime:   time.Unix(16, 0),
	}

	for i := 0; i < b.N; i++ {
		s := cacheKeyForJob(10, req, meta, 1, 2)
		if len(s) == 0 {
			b.Fatalf("expected non-empty string")
		}
	}
}
