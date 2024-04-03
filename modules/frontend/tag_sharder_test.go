package frontend

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"testing"
	"time"

	//nolint:all deprecated

	"github.com/go-kit/log"
	"github.com/google/uuid"
	"github.com/grafana/dskit/user"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/modules/frontend/pipeline"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/tempodb/backend"
)

type fakeReq struct {
	startValue uint32
	endValue   uint32
}

func (r *fakeReq) start() uint32 {
	return r.startValue
}

func (r *fakeReq) end() uint32 {
	return r.endValue
}

func (r *fakeReq) newWithRange(start, end uint32) tagSearchReq {
	return &fakeReq{
		startValue: start,
		endValue:   end,
	}
}

func (r *fakeReq) hash() uint64 {
	return 0
}

func (r *fakeReq) keyPrefix() string {
	return ""
}

func (r *fakeReq) buildSearchTagRequest(subR *http.Request) (*http.Request, error) {
	newReq := subR.Clone(subR.Context())
	q := subR.URL.Query()
	q.Set("start", strconv.FormatUint(uint64(r.startValue), 10))
	q.Set("end", strconv.FormatUint(uint64(r.endValue), 10))
	newReq.URL.RawQuery = q.Encode()

	return newReq, nil
}

func (r *fakeReq) buildTagSearchBlockRequest(subR *http.Request, blockID string,
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
			reqCh := make(chan *http.Request)
			req := fakeReq{}
			if tc.params != nil {
				req.startValue = uint32(tc.params.start)
				req.endValue = uint32(tc.params.end)

			}
			s.backendRequests(context.TODO(), "test", r, &req, reqCh, func(err error) {
				require.Equal(t, tc.expectedError, err)
			})

			actualReqURIs := []string{}
			for r := range reqCh {
				actualReqURIs = append(actualReqURIs, r.RequestURI)
			}
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

		searchReq := fakeReq{
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

func TestTagsSearchSharderRoundTripBadRequest(t *testing.T) {
	next := pipeline.AsyncRoundTripperFunc[*http.Response](func(r *http.Request) (pipeline.Responses[*http.Response], error) {
		return nil, nil
	})

	o, err := overrides.NewOverrides(overrides.Config{}, nil, prometheus.NewRegistry())
	require.NoError(t, err)

	sharder := newAsyncTagSharder(&mockReader{}, o, SearchSharderConfig{
		ConcurrentRequests:    defaultConcurrentRequests,
		TargetBytesPerRequest: defaultTargetBytesPerRequest,
		MaxDuration:           5 * time.Minute,
	}, parseTagsRequest, log.NewNopLogger())
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
	req = req.WithContext(user.InjectOrgID(req.Context(), "blerg"))
	resp, err = testRT.RoundTrip(req)
	testBadRequestFromResponses(t, resp, err, "invalid start: strconv.ParseInt: parsing \"asdf\": invalid syntax")

	// test max duration error with overrides
	o, err = overrides.NewOverrides(overrides.Config{
		Defaults: overrides.Overrides{
			Read: overrides.ReadOverrides{
				MaxSearchDuration: model.Duration(time.Minute),
			},
		},
	}, nil, prometheus.NewRegistry())
	require.NoError(t, err)

	sharder = newAsyncTagSharder(&mockReader{}, o, SearchSharderConfig{
		ConcurrentRequests:    defaultConcurrentRequests,
		TargetBytesPerRequest: defaultTargetBytesPerRequest,
		MaxDuration:           5 * time.Minute,
	}, parseTagsRequest, log.NewNopLogger())
	testRT = sharder.Wrap(next)

	req = httptest.NewRequest("GET", "/?start=1000&end=1500", nil)
	req = req.WithContext(user.InjectOrgID(req.Context(), "blerg"))
	resp, err = testRT.RoundTrip(req)
	testBadRequestFromResponses(t, resp, err, "range specified by start and end exceeds 1m0s. received start=1000 end=1500")
}
