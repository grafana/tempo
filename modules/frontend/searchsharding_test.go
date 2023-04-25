package frontend

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/golang/protobuf/jsonpb" //nolint:all deprecated
	"github.com/google/uuid"
	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/weaveworks/common/user"
	"go.uber.org/atomic"

	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/traceql"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/blocklist"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

var testSLOcfg = SLOConfig{
	ThroughputBytesSLO: 0,
	DurationSLO:        0,
}

// implements tempodb.Reader interface
type mockReader struct {
	metas []*backend.BlockMeta
}

func (m *mockReader) Find(ctx context.Context, tenantID string, id common.ID, blockStart string, blockEnd string, timeStart int64, timeEnd int64) ([]*tempopb.Trace, []error, error) {
	return nil, nil, nil
}

func (m *mockReader) BlockMetas(tenantID string) []*backend.BlockMeta {
	return m.metas
}

func (m *mockReader) Search(ctx context.Context, meta *backend.BlockMeta, req *tempopb.SearchRequest, opts common.SearchOptions) (*tempopb.SearchResponse, error) {
	return nil, nil
}

func (m *mockReader) Fetch(ctx context.Context, meta *backend.BlockMeta, req traceql.FetchSpansRequest, opts common.SearchOptions) (traceql.FetchSpansResponse, error) {
	return traceql.FetchSpansResponse{}, nil
}

func (m *mockReader) EnablePolling(sharder blocklist.JobSharder) {}
func (m *mockReader) Shutdown()                                  {}

func TestBackendRequests(t *testing.T) {
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
		s := &searchSharder{
			cfg: SearchSharderConfig{
				TargetBytesPerRequest: tc.targetBytesPerRequest,
			},
		}
		req := httptest.NewRequest("GET", "/?k=test&v=test&start=10&end=20", nil)

		reqs, err := s.backendRequests(context.Background(), "test", req, tc.metas)
		if tc.expectedError != nil {
			assert.Equal(t, tc.expectedError, err)
			continue
		}
		assert.NoError(t, err)

		actualURIs := []string{}
		for _, r := range reqs {
			actualURIs = append(actualURIs, r.RequestURI)
		}

		assert.Equal(t, tc.expectedURIs, actualURIs)
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
			expectedURI:         "/querier?end=" + strconv.Itoa(now) + "&limit=50&maxDuration=30ms&minDuration=10ms&start=" + strconv.Itoa(tenMinutesAgo) + "&tags=foo%3Dbar",
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
			expectedURI:         "/querier?end=" + strconv.Itoa(tenMinutesAgo) + "&limit=50&maxDuration=30ms&minDuration=10ms&start=" + strconv.Itoa(fifteenMinutesAgo) + "&tags=foo%3Dbar",
		},
		// start/end = 10 - now mins ago - break across query backend after
		//  ingester start/End = 10 - now mins ago
		//  backend start/End = 15 - 10 mins ago
		{
			request:             "/?tags=foo%3Dbar&minDuration=10ms&maxDuration=30ms&limit=50&start=" + strconv.Itoa(tenMinutesAgo) + "&end=" + strconv.Itoa(now),
			queryIngestersUntil: 15 * time.Minute,
			expectedURI:         "/querier?end=" + strconv.Itoa(now) + "&limit=50&maxDuration=30ms&minDuration=10ms&start=" + strconv.Itoa(tenMinutesAgo) + "&tags=foo%3Dbar",
		},
		// start/end = 20 - now mins ago - break across both query ingesters until and backend after
		//  ingester start/End = 15 - now mins ago
		//  backend start/End = 20 - 5 mins ago
		{
			request:             "/?tags=foo%3Dbar&minDuration=10ms&maxDuration=30ms&limit=50&start=" + strconv.Itoa(twentyMinutesAgo) + "&end=" + strconv.Itoa(now),
			queryIngestersUntil: 15 * time.Minute,
			expectedURI:         "/querier?end=" + strconv.Itoa(now) + "&limit=50&maxDuration=30ms&minDuration=10ms&start=" + strconv.Itoa(fifteenMinutesAgo) + "&tags=foo%3Dbar",
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
	}

	for _, tc := range tests {
		s := &searchSharder{
			cfg: SearchSharderConfig{
				QueryBackendAfter: tc.queryBackendAfter,
			},
		}
		req := httptest.NewRequest("GET", tc.request, nil)

		searchReq, err := api.ParseSearchRequest(req)
		require.NoError(t, err)

		actualStart, actualEnd := s.backendRange(searchReq)
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
				}},
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
				}},
			expectedStatus: 200,
			expectedResponse: &tempopb.SearchResponse{
				Traces: []*tempopb.TraceSearchMetadata{
					{
						TraceID:           "1234",
						StartTimeUnixNano: 1,
					},
					{
						TraceID:           "5678",
						StartTimeUnixNano: 0,
					},
				},
				Metrics: &tempopb.SearchMetrics{
					InspectedTraces: 6,
					TotalBlocks:     1,
					InspectedBytes:  10,
					CompletedJobs:   2,
					TotalJobs:       2,
					TotalBlockBytes: defaultTargetBytesPerRequest * 2,
				}},
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

			o, err := overrides.NewOverrides(overrides.Limits{})
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
			}, testSLOcfg, newSearchProgress, log.NewNopLogger())
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

func TestSearchSharderRoundTripBadRequest(t *testing.T) {
	next := RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
		return nil, nil
	})

	o, err := overrides.NewOverrides(overrides.Limits{})
	require.NoError(t, err)

	sharder := newSearchSharder(&mockReader{}, o, SearchSharderConfig{
		ConcurrentRequests:    defaultConcurrentRequests,
		TargetBytesPerRequest: defaultTargetBytesPerRequest,
		MaxDuration:           5 * time.Minute,
	}, testSLOcfg, newSearchProgress, log.NewNopLogger())
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
	o, err = overrides.NewOverrides(overrides.Limits{
		MaxSearchDuration: model.Duration(time.Minute),
	})
	require.NoError(t, err)

	sharder = newSearchSharder(&mockReader{}, o, SearchSharderConfig{
		ConcurrentRequests:    defaultConcurrentRequests,
		TargetBytesPerRequest: defaultTargetBytesPerRequest,
		MaxDuration:           5 * time.Minute,
	}, testSLOcfg, newSearchProgress, log.NewNopLogger())
	testRT = NewRoundTripper(next, sharder)

	req = httptest.NewRequest("GET", "/?start=1000&end=1500", nil)
	req = req.WithContext(user.InjectOrgID(req.Context(), "blerg"))
	resp, err = testRT.RoundTrip(req)
	testBadRequest(t, resp, err, "range specified by start and end exceeds 1m0s. received start=1000 end=1500")
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
	o, err := overrides.NewOverrides(overrides.Limits{})
	require.NoError(t, err)
	sharder := searchSharder{
		cfg: SearchSharderConfig{
			MaxDuration: 5 * time.Minute,
		},
		overrides: o,
	}
	actual := sharder.maxDuration("test")
	assert.Equal(t, 5*time.Minute, actual)

	o, err = overrides.NewOverrides(overrides.Limits{
		MaxSearchDuration: model.Duration(10 * time.Minute),
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

	o, err := overrides.NewOverrides(overrides.Limits{})
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
	}, testSLOcfg, newSearchProgress, log.NewNopLogger())

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
