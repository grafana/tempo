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

	"github.com/grafana/tempo/pkg/util/test"
	"github.com/prometheus/common/model"
	"go.uber.org/atomic"

	"github.com/go-kit/log"
	"github.com/gogo/protobuf/jsonpb" //nolint:all deprecated
	"github.com/google/uuid"
	"github.com/grafana/dskit/user"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type faceReq struct {
	startValue uint32
	endValue   uint32
}

func (r *faceReq) start() uint32 {
	return r.startValue
}

func (r *faceReq) end() uint32 {
	return r.endValue
}

func (r *faceReq) newWithRange(start, end uint32) tagSearchReq {
	return &faceReq{
		startValue: start,
		endValue:   end,
	}
}

func (r *faceReq) buildSearchTagRequest(subR *http.Request) (*http.Request, error) {
	newReq := subR.Clone(subR.Context())
	q := subR.URL.Query()
	q.Set("start", strconv.FormatUint(uint64(r.startValue), 10))
	q.Set("end", strconv.FormatUint(uint64(r.endValue), 10))
	newReq.URL.RawQuery = q.Encode()

	return newReq, nil
}

func (r *faceReq) buildTagSearchBlockRequest(subR *http.Request, blockID string,
	startPage int, pages int, _ *backend.BlockMeta,
) (*http.Request, error) {
	newReq := subR.Clone(subR.Context())
	q := subR.URL.Query()
	q.Set("size", "209715200")
	q.Set("blockID", blockID)
	q.Set("startPage", strconv.FormatUint(uint64(startPage), 10))
	q.Set("pagesToSearch", strconv.FormatUint(uint64(pages), 10))
	q.Set("encoding", "gzip")
	q.Set("indexPageSize", strconv.FormatUint(0, 10))
	q.Set("totalRecords", strconv.FormatUint(2, 10))
	q.Set("dataEncoding", "asdf")
	q.Set("version", "wdwad")
	q.Set("footerSize", strconv.FormatUint(0, 10))

	newReq.URL.RawQuery = q.Encode()

	return newReq, nil
}

func TestTagsBackendRequests(t *testing.T) {
	bm := backend.NewBlockMeta("test", uuid.New(), "wdwad", backend.EncGZIP, "asdf")
	bm.StartTime = time.Unix(100, 0)
	bm.EndTime = time.Unix(200, 0)
	bm.Size = defaultTargetBytesPerRequest * 2
	bm.TotalRecords = 2

	s := &searchTagSharder{
		cfg:    SearchSharderConfig{},
		reader: &mockReader{metas: []*backend.BlockMeta{bm}},
	}

	type params struct {
		start int
		end   int
	}

	tests := []struct {
		name             string
		params           *params
		expectedReqsURIs []string
		expectedError    error
	}{
		{
			name: "start and end same as block",
			params: &params{
				100, 200,
			},
			expectedReqsURIs: []string{
				"/querier?blockID=" + bm.BlockID.String() + "&dataEncoding=asdf&encoding=gzip&end=200&footerSize=0&indexPageSize=0&pagesToSearch=1&size=209715200&start=100&startPage=0&totalRecords=2&version=wdwad",
				"/querier?blockID=" + bm.BlockID.String() + "&dataEncoding=asdf&encoding=gzip&end=200&footerSize=0&indexPageSize=0&pagesToSearch=1&size=209715200&start=100&startPage=1&totalRecords=2&version=wdwad",
			},
			expectedError: nil,
		},
		{
			name: "start and end in block",
			params: &params{
				110, 150,
			},
			expectedReqsURIs: []string{
				"/querier?blockID=" + bm.BlockID.String() + "&dataEncoding=asdf&encoding=gzip&end=150&footerSize=0&indexPageSize=0&pagesToSearch=1&size=209715200&start=110&startPage=0&totalRecords=2&version=wdwad",
				"/querier?blockID=" + bm.BlockID.String() + "&dataEncoding=asdf&encoding=gzip&end=150&footerSize=0&indexPageSize=0&pagesToSearch=1&size=209715200&start=110&startPage=1&totalRecords=2&version=wdwad",
			},
			expectedError: nil,
		},
		{
			name: "start and end out of block",
			params: &params{
				10, 20,
			},
			expectedReqsURIs: make([]string, 0),
			expectedError:    nil,
		},
		{
			name:             "no params",
			expectedReqsURIs: make([]string, 0),
			expectedError:    nil,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			request := "/?"
			if tc.params != nil {
				request = fmt.Sprintf("/?start=%d&end=%d", tc.params.start, tc.params.end)
			}
			r := httptest.NewRequest("GET", request, nil)

			stopCh := make(chan struct{})
			defer close(stopCh)
			reqCh := make(chan *backendReqMsg)
			req := faceReq{}
			if tc.params != nil {
				req.startValue = uint32(tc.params.start)
				req.endValue = uint32(tc.params.end)

			}
			s.backendRequests(context.TODO(), "test", r, &req, reqCh, stopCh)
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

func TestTagsIngesterRequest(t *testing.T) {
	now := int(time.Now().Unix())
	tenMinutesAgo := int(time.Now().Add(-10 * time.Minute).Unix())
	fifteenMinutesAgo := int(time.Now().Add(-15 * time.Minute).Unix())
	twentyMinutesAgo := int(time.Now().Add(-20 * time.Minute).Unix())

	urlStartReq := "/?start="
	startPart := "&start="

	tests := []struct {
		request             string
		queryIngestersUntil time.Duration
		expectedURI         string
		expectedError       error
		start               int
		end                 int
	}{
		// start/end is outside queryIngestersUntil
		{
			request:             "/?start=10&end=20",
			queryIngestersUntil: 10 * time.Minute,
			expectedURI:         "",
			start:               10,
			end:                 20,
		},
		// start/end is inside queryBackendAfter
		{
			request:             urlStartReq + strconv.Itoa(tenMinutesAgo) + "&end=" + strconv.Itoa(now),
			queryIngestersUntil: 30 * time.Minute,
			expectedURI:         "/querier?end=" + strconv.Itoa(now) + startPart + strconv.Itoa(tenMinutesAgo),
			start:               tenMinutesAgo,
			end:                 now,
		},
		// backendAfter/ingsetersUntil = 0 results in no ingester query
		{
			request: urlStartReq + strconv.Itoa(tenMinutesAgo) + "&end=" + strconv.Itoa(now),
			start:   tenMinutesAgo,
			end:     now,
		},
		// start/end = 20 - 10 mins ago - break across query ingesters until
		//  ingester start/End = 15 - 10 mins ago
		{
			request:             urlStartReq + strconv.Itoa(twentyMinutesAgo) + "&end=" + strconv.Itoa(tenMinutesAgo),
			queryIngestersUntil: 15 * time.Minute,
			expectedURI:         "/querier?end=" + strconv.Itoa(tenMinutesAgo) + startPart + strconv.Itoa(fifteenMinutesAgo),
			start:               twentyMinutesAgo,
			end:                 tenMinutesAgo,
		},
		// start/end = 10 - now mins ago - break across query backend after
		//  ingester start/End = 10 - now mins ago
		//  backend start/End = 15 - 10 mins ago
		{
			request:             urlStartReq + strconv.Itoa(tenMinutesAgo) + "&end=" + strconv.Itoa(now),
			queryIngestersUntil: 15 * time.Minute,
			expectedURI:         "/querier?end=" + strconv.Itoa(now) + startPart + strconv.Itoa(tenMinutesAgo),
			start:               tenMinutesAgo,
			end:                 now,
		},
		// start/end = 20 - now mins ago - break across both query ingesters until and backend after
		//  ingester start/End = 15 - now mins ago
		//  backend start/End = 20 - 5 mins ago
		{
			request:             urlStartReq + strconv.Itoa(twentyMinutesAgo) + "&end=" + strconv.Itoa(now),
			queryIngestersUntil: 15 * time.Minute,
			expectedURI:         "/querier?end=" + strconv.Itoa(now) + startPart + strconv.Itoa(fifteenMinutesAgo),
			start:               twentyMinutesAgo,
			end:                 now,
		},
		{
			request:             "/?",
			queryIngestersUntil: 15 * time.Minute,
			expectedURI:         "/querier?end=0&start=0",
		},
	}

	for _, tc := range tests {
		s := &searchTagSharder{
			cfg: SearchSharderConfig{
				QueryIngestersUntil: tc.queryIngestersUntil,
			},
		}

		req := httptest.NewRequest("GET", tc.request, nil)

		searchReq := faceReq{
			startValue: uint32(tc.start),
			endValue:   uint32(tc.end),
		}

		copyReq := searchReq
		actualReq, err := s.ingesterRequest(context.Background(), "test", req, &searchReq)
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

func TestTagsSearchSharderRoundTrip(t *testing.T) {
	tests := []struct {
		name             string
		status1          int
		status2          int
		response1        *tempopb.SearchTagsResponse
		response2        *tempopb.SearchTagsResponse
		err1             error
		err2             error
		expectedStatus   int
		expectedResponse *tempopb.SearchTagsResponse
		expectedError    error
	}{
		{
			name:             "empty returns",
			status1:          200,
			status2:          200,
			expectedStatus:   200,
			response1:        &tempopb.SearchTagsResponse{},
			response2:        &tempopb.SearchTagsResponse{},
			expectedResponse: &tempopb.SearchTagsResponse{},
		},
		{
			name:           "404+200",
			status1:        404,
			status2:        200,
			response2:      &tempopb.SearchTagsResponse{},
			expectedStatus: 500,
		},
		{
			name:           "200+400",
			status1:        200,
			response1:      &tempopb.SearchTagsResponse{},
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
			response2:      &tempopb.SearchTagsResponse{},
			expectedStatus: 500,
		},
		{
			name:           "200+500",
			status1:        200,
			response1:      &tempopb.SearchTagsResponse{},
			status2:        500,
			expectedStatus: 500,
		},
		{
			name:    "200+200",
			status1: 200,
			response1: &tempopb.SearchTagsResponse{
				TagNames: []string{
					"tag1",
					"tag2",
					"tag3",
					"tag4",
				},
			},
			status2: 200,
			response2: &tempopb.SearchTagsResponse{
				TagNames: []string{
					"tag1",
					"tag3",
					"tag5",
				},
			},
			expectedStatus: 200,
			expectedResponse: &tempopb.SearchTagsResponse{
				TagNames: []string{
					"tag1",
					"tag2",
					"tag3",
					"tag4",
					"tag5",
				},
			},
		},
		{
			name:          "200+err",
			status1:       200,
			response1:     &tempopb.SearchTagsResponse{},
			err2:          errors.New("booo"),
			expectedError: errors.New("booo"),
		},
		{
			name:          "err+200",
			err1:          errors.New("booo"),
			status2:       200,
			response2:     &tempopb.SearchTagsResponse{},
			expectedError: errors.New("booo"),
		},
		{
			name:           "500+err",
			status1:        500,
			response1:      &tempopb.SearchTagsResponse{},
			err2:           errors.New("booo"),
			expectedStatus: 500,
			expectedError:  errors.New("booo"),
		},
		{
			name:          "err+500",
			err1:          errors.New("booo"),
			status2:       500,
			response2:     &tempopb.SearchTagsResponse{},
			expectedError: errors.New("booo"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			next := RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
				var response *tempopb.SearchTagsResponse
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

			sharder := newTagsSharding(&mockReader{
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
			}, tagsResultHandlerFactory, log.NewNopLogger(), parseTagsRequest)
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
				actualResp := &tempopb.SearchTagsResponse{}
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

func TestTagsSearchSubRequestsCancelled(t *testing.T) {
	totalJobs := 5

	wg := sync.WaitGroup{}
	nextSuccess := RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
		wg.Done()
		wg.Wait()

		resString, err := (&jsonpb.Marshaler{}).MarshalToString(&tempopb.SearchTagsResponse{
			TagNames: []string{
				test.RandomString(),
			},
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

	sharder := newTagsSharding(&mockReader{
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
	}, tagsResultHandlerFactory, log.NewNopLogger(), parseTagsRequest)

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

func TestTagsSearchSharderRoundTripBadRequest(t *testing.T) {
	next := RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
		return nil, nil
	})

	o, err := overrides.NewOverrides(overrides.Config{})
	require.NoError(t, err)

	sharder := newTagsSharding(&mockReader{}, o, SearchSharderConfig{
		ConcurrentRequests:    defaultConcurrentRequests,
		TargetBytesPerRequest: defaultTargetBytesPerRequest,
		MaxDuration:           5 * time.Minute,
	}, tagsResultHandlerFactory, log.NewNopLogger(), parseTagsRequest)
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
	req = req.WithContext(user.InjectOrgID(req.Context(), "blerg"))
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

	sharder = newTagsSharding(&mockReader{}, o, SearchSharderConfig{
		ConcurrentRequests:    defaultConcurrentRequests,
		TargetBytesPerRequest: defaultTargetBytesPerRequest,
		MaxDuration:           5 * time.Minute,
	}, tagsResultHandlerFactory, log.NewNopLogger(), parseTagsRequest)
	testRT = NewRoundTripper(next, sharder)

	req = httptest.NewRequest("GET", "/?start=1000&end=1500", nil)
	req = req.WithContext(user.InjectOrgID(req.Context(), "blerg"))
	resp, err = testRT.RoundTrip(req)
	testBadRequest(t, resp, err, "range specified by start and end exceeds 1m0s. received start=1000 end=1500")
}
