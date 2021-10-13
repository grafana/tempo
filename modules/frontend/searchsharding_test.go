package frontend

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/golang/protobuf/jsonpb"
	"github.com/google/uuid"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/blocklist"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/weaveworks/common/user"
)

// implements tempodb.Reader interface
type mockReader struct {
	metas []*backend.BlockMeta
}

func (m *mockReader) Find(ctx context.Context, tenantID string, id common.ID, blockStart string, blockEnd string) ([][]byte, []string, []error, error) {
	return nil, nil, nil, nil
}

func (m *mockReader) BlockMetas(tenantID string) []*backend.BlockMeta {
	return m.metas
}

func (m *mockReader) EnablePolling(sharder blocklist.JobSharder) {}
func (m *mockReader) Shutdown()                                  {}

func TestSearchResponseShouldQuit(t *testing.T) {
	ctx := context.Background()

	// brand new response should not quit
	sr := newSearchResponse(ctx, 10)
	assert.False(t, sr.shouldQuit())

	// errored response should quit
	sr = newSearchResponse(ctx, 10)
	sr.setError(errors.New("blerg"))
	assert.True(t, sr.shouldQuit())

	// happy status code should not quit
	sr = newSearchResponse(ctx, 10)
	sr.setStatus(200, "")
	assert.False(t, sr.shouldQuit())

	// sad status code should quit
	sr = newSearchResponse(ctx, 10)
	sr.setStatus(400, "")
	assert.True(t, sr.shouldQuit())

	sr = newSearchResponse(ctx, 10)
	sr.setStatus(500, "")
	assert.True(t, sr.shouldQuit())

	// cancelled context should quit
	cancellableContext, cancel := context.WithCancel(ctx)
	sr = newSearchResponse(cancellableContext, 10)
	cancel()
	assert.True(t, sr.shouldQuit())

	// limit reached should quit
	sr = newSearchResponse(ctx, 2)
	sr.addResponse(&tempopb.SearchResponse{
		Traces: []*tempopb.TraceSearchMetadata{
			{},
		},
		Metrics: &tempopb.SearchMetrics{},
	})
	assert.False(t, sr.shouldQuit())
	sr.addResponse(&tempopb.SearchResponse{
		Traces: []*tempopb.TraceSearchMetadata{
			{},
			{},
		},
		Metrics: &tempopb.SearchMetrics{},
	})
	assert.True(t, sr.shouldQuit())
}

func TestSearchSharderParams(t *testing.T) {
	tests := []struct {
		start         int64
		end           int64
		limit         int
		expectedLimit int
		expectedError error
	}{
		{
			expectedError: errors.New("please provide non-zero values for http parameters start and end"),
		},
		{
			start:         10,
			expectedError: errors.New("please provide non-zero values for http parameters start and end"),
		},
		{
			end:           10,
			expectedError: errors.New("please provide non-zero values for http parameters start and end"),
		},
		{
			start:         15,
			end:           10,
			expectedError: errors.New("http parameter start must be before end. received start=15 end=10"),
		},
		{
			start:         10,
			end:           100000,
			expectedError: errors.New("range specified by start and end exceeds 1800 seconds. received start=10 end=100000"),
		},
		{
			start:         10,
			end:           20,
			expectedLimit: 20,
		},
		{
			start:         10,
			end:           20,
			limit:         30,
			expectedLimit: 30,
		},
	}

	for _, tc := range tests {
		url := "/blerg?"
		if tc.start != 0 {
			url += fmt.Sprintf("&start=%d", tc.start)
		}
		if tc.end != 0 {
			url += fmt.Sprintf("&end=%d", tc.end)
		}
		if tc.limit != 0 {
			url += fmt.Sprintf("&limit=%d", tc.limit)
		}
		r := httptest.NewRequest("GET", url, nil)

		actualStart, actualEnd, actualLimit, actualError := searchSharderParams(r)

		if tc.expectedError != nil {
			assert.Equal(t, tc.expectedError, actualError)
			continue
		}
		assert.NoError(t, actualError)
		assert.Equal(t, tc.start, actualStart)
		assert.Equal(t, tc.end, actualEnd)
		assert.Equal(t, tc.expectedLimit, actualLimit)
	}
}

func TestShardedRequests(t *testing.T) {
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
			expectedError: errors.New("block 00000000-0000-0000-0000-000000000000 has an invalid 0 size"),
		},
		// block with no records
		{
			metas: []*backend.BlockMeta{
				{
					Size:    1000,
					BlockID: uuid.MustParse("00000000-0000-0000-0000-000000000000"),
				},
			},
			expectedError: errors.New("block 00000000-0000-0000-0000-000000000000 has an invalid 0 records"),
		},
		// bytes/per request is too small for the page size
		{
			targetBytesPerRequest: 1,
			metas: []*backend.BlockMeta{
				{
					Size:         1000,
					TotalRecords: 100,
					BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000000"),
				},
			},
			expectedError: errors.New("block 00000000-0000-0000-0000-000000000000 has an invalid 0 pages per query"),
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
				"/querier/?blockID=00000000-0000-0000-0000-000000000000&end=20&k=test&start=10&startPage=0&totalPages=100&v=test",
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
				"/querier/?blockID=00000000-0000-0000-0000-000000000000&end=20&k=test&start=10&startPage=0&totalPages=90&v=test",
				"/querier/?blockID=00000000-0000-0000-0000-000000000000&end=20&k=test&start=10&startPage=90&totalPages=90&v=test",
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
				"/querier/?blockID=00000000-0000-0000-0000-000000000000&end=20&k=test&start=10&startPage=0&totalPages=90&v=test",
				"/querier/?blockID=00000000-0000-0000-0000-000000000000&end=20&k=test&start=10&startPage=90&totalPages=90&v=test",
				"/querier/?blockID=00000000-0000-0000-0000-000000000001&end=20&k=test&start=10&startPage=0&totalPages=180&v=test",
				"/querier/?blockID=00000000-0000-0000-0000-000000000001&end=20&k=test&start=10&startPage=180&totalPages=180&v=test",
			},
		},
	}

	for _, tc := range tests {
		s := &searchSharder{
			targetBytesPerRequest: tc.targetBytesPerRequest,
		}
		req := httptest.NewRequest("GET", "/?k=test&v=test&start=10&end=20", nil)

		reqs, err := s.shardedRequests(context.Background(), tc.metas, "test", req)
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
			name:             "empty returns",
			status1:          200,
			status2:          200,
			expectedStatus:   200,
			response1:        &tempopb.SearchResponse{Metrics: &tempopb.SearchMetrics{}},
			response2:        &tempopb.SearchResponse{Metrics: &tempopb.SearchMetrics{}},
			expectedResponse: &tempopb.SearchResponse{Metrics: &tempopb.SearchMetrics{}},
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
						TraceID: "1234",
					},
				},
				Metrics: &tempopb.SearchMetrics{
					InspectedTraces: 1,
					InspectedBlocks: 2,
					InspectedBytes:  3,
					SkippedBlocks:   4,
				}},
			status2: 200,
			response2: &tempopb.SearchResponse{
				Traces: []*tempopb.TraceSearchMetadata{
					{
						TraceID: "5678",
					},
				},
				Metrics: &tempopb.SearchMetrics{
					InspectedTraces: 5,
					InspectedBlocks: 6,
					InspectedBytes:  7,
					SkippedBlocks:   8,
				}},
			expectedStatus: 200,
			expectedResponse: &tempopb.SearchResponse{
				Traces: []*tempopb.TraceSearchMetadata{
					{
						TraceID: "1234",
					},
					{
						TraceID: "5678",
					},
				},
				Metrics: &tempopb.SearchMetrics{
					InspectedTraces: 6,
					InspectedBlocks: 8,
					InspectedBytes:  10,
					SkippedBlocks:   12,
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
			}, 1, log.NewNopLogger()) // 1 concurrent request to force order
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
			if tc.expectedResponse != nil {
				actualResp := &tempopb.SearchResponse{}
				bytesResp, err := io.ReadAll(resp.Body)
				require.NoError(t, err)
				err = jsonpb.Unmarshal(bytes.NewReader(bytesResp), actualResp)
				require.NoError(t, err)

				assert.Equal(t, tc.expectedResponse, actualResp)
			}
		})
	}
}
