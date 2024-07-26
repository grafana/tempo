package frontend

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
	"github.com/gorilla/mux"
	"github.com/grafana/dskit/user"
	"github.com/grafana/tempo/pkg/model/trace"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var config = &Config{
	MultiTenantQueriesEnabled: true,
	MaxRetries:                0, // disable retries or it will try twice and get success. the querier response is designed to fail exactly once
	TraceByID: TraceByIDConfig{
		QueryShards: 2,
		SLO:         testSLOcfg,
	},
	Search: SearchConfig{
		Sharder: SearchSharderConfig{
			ConcurrentRequests:    defaultConcurrentRequests,
			TargetBytesPerRequest: defaultTargetBytesPerRequest,
		},
		SLO: testSLOcfg,
	},
	Metrics: MetricsConfig{
		Sharder: QueryRangeSharderConfig{
			ConcurrentRequests:    defaultConcurrentRequests,
			TargetBytesPerRequest: defaultTargetBytesPerRequest,
			Interval:              time.Second,
		},
		SLO: testSLOcfg,
	},
}

func TestTraceIDHandler(t *testing.T) {
	// create and split a splitTrace
	splitTrace := test.MakeTrace(2, []byte{0x01, 0x02})
	trace1 := &tempopb.Trace{}
	trace2 := &tempopb.Trace{}

	for i, b := range splitTrace.ResourceSpans {
		if i%2 == 0 {
			trace1.ResourceSpans = append(trace1.ResourceSpans, b)
		} else {
			trace2.ResourceSpans = append(trace2.ResourceSpans, b)
		}
	}

	tests := []struct {
		name           string
		status1        int
		status2        int
		trace1         *tempopb.Trace
		trace2         *tempopb.Trace
		err1           error
		err2           error
		expectedStatus int
		expectedTrace  *tempopb.Trace
	}{
		{
			name:           "empty returns",
			status1:        200,
			status2:        200,
			expectedStatus: 200,
			expectedTrace:  &tempopb.Trace{},
		},
		{
			name:           "404",
			status1:        404,
			status2:        404,
			expectedStatus: 404,
		},
		{
			name:           "400",
			status1:        400,
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
			trace2:         trace2,
			expectedStatus: 500,
		},
		{
			name:           "200+500",
			status1:        200,
			trace1:         trace1,
			status2:        500,
			expectedStatus: 500,
		},
		{
			name:           "200+429",
			status1:        200,
			trace1:         trace1,
			status2:        429,
			expectedStatus: 429,
		},
		{
			name:           "200+404",
			status1:        200,
			trace1:         trace1,
			status2:        404,
			expectedStatus: 200,
			expectedTrace:  trace1,
		},
		{
			name:           "404+200",
			status1:        404,
			status2:        200,
			trace2:         trace1,
			expectedStatus: 200,
			expectedTrace:  trace1,
		},
		{
			name:           "200+200",
			status1:        200,
			trace1:         trace1,
			status2:        200,
			trace2:         trace2,
			expectedStatus: 200,
			expectedTrace:  splitTrace,
		},
		{
			name:           "200+err",
			status1:        200,
			trace1:         trace1,
			err2:           errors.New("booo"),
			expectedStatus: 500,
		},
	}

	for _, tc := range tests {
		tc := tc // copy the test case to prevent race on the loop variable
		t.Run(tc.name, func(t *testing.T) {
			next := RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
				var testTrace *tempopb.Trace
				var statusCode int
				var err error
				if r.RequestURI == "/querier/api/traces/1234?mode=ingesters" {
					testTrace = tc.trace1
					statusCode = tc.status1
					err = tc.err1
				} else {
					testTrace = tc.trace2
					err = tc.err2
					statusCode = tc.status2
				}

				if err != nil {
					return nil, err
				}

				resBytes := []byte("error occurred")
				if statusCode != 500 {
					if testTrace != nil {
						resBytes, err = proto.Marshal(&tempopb.TraceByIDResponse{
							Trace:   testTrace,
							Metrics: &tempopb.TraceByIDMetrics{},
						})
						require.NoError(t, err)
					} else {
						resBytes, err = proto.Marshal(&tempopb.TraceByIDResponse{
							Metrics: &tempopb.TraceByIDMetrics{},
						})
						require.NoError(t, err)
					}
				}

				return &http.Response{
					Body:       io.NopCloser(bytes.NewReader(resBytes)),
					StatusCode: statusCode,
				}, nil
			})

			// queriers will return one err
			f := frontendWithSettings(t, next, nil, config, nil)

			req := httptest.NewRequest("GET", "/api/traces/1234", nil)
			ctx := req.Context()
			ctx = user.InjectOrgID(ctx, "blerg")
			req = req.WithContext(ctx)
			req = mux.SetURLVars(req, map[string]string{"traceID": "1234"})
			req.Header.Set("Accept", "application/protobuf")

			httpResp := httptest.NewRecorder()
			f.TraceByIDHandler.ServeHTTP(httpResp, req)
			resp := httpResp.Result()

			if tc.expectedStatus != resp.StatusCode {
				body, _ := io.ReadAll(resp.Body)
				require.Fail(t, "unexpected status code", "expected %d, got %d. body: %s", tc.expectedStatus, resp.StatusCode, body)
			}
			if tc.expectedStatus == http.StatusOK {
				assert.Equal(t, "application/protobuf", resp.Header.Get("Content-Type"))
			}
			if tc.expectedTrace != nil {
				actualResp := &tempopb.Trace{}
				bytesTrace, err := io.ReadAll(resp.Body)
				require.NoError(t, err)
				err = proto.Unmarshal(bytesTrace, actualResp)
				require.NoError(t, err)

				trace.SortTrace(tc.expectedTrace)
				trace.SortTrace(actualResp)
				assert.True(t, proto.Equal(tc.expectedTrace, actualResp))
			}
		})
	}
}

func TestTraceIDHandlerForJSONResponse(t *testing.T) {
	next := RoundTripperFunc(func(_ *http.Request) (*http.Response, error) {
		testTrace := test.MakeTrace(2, []byte{0x01, 0x02})
		resBytes, _ := proto.Marshal(&tempopb.TraceByIDResponse{
			Trace:   testTrace,
			Metrics: &tempopb.TraceByIDMetrics{},
		})
		return &http.Response{
			Body:       io.NopCloser(bytes.NewReader(resBytes)),
			StatusCode: 200,
		}, nil
	})

	// queriers will return one err
	f := frontendWithSettings(t, next, nil, config, nil)

	req := httptest.NewRequest("GET", "/api/traces/1234", nil)
	ctx := req.Context()
	ctx = user.InjectOrgID(ctx, "blerg")
	req = req.WithContext(ctx)
	req = mux.SetURLVars(req, map[string]string{"traceID": "1234"})
	req.Header.Set("Accept", "application/json")

	httpResp := httptest.NewRecorder()
	f.TraceByIDHandler.ServeHTTP(httpResp, req)
	resp := httpResp.Result()

	bodyBytes, _ := io.ReadAll(resp.Body)
	bodyString := string(bodyBytes)

	assert.True(t, strings.Contains(bodyString, "batches"))
}

func TestTraceIDHandlerV2(t *testing.T) {
	// create and split a splitTrace
	splitTrace := test.MakeTrace(2, []byte{0x01, 0x02})
	trace1 := &tempopb.Trace{}
	trace2 := &tempopb.Trace{}

	for i, b := range splitTrace.ResourceSpans {
		if i%2 == 0 {
			trace1.ResourceSpans = append(trace1.ResourceSpans, b)
		} else {
			trace2.ResourceSpans = append(trace2.ResourceSpans, b)
		}
	}

	tests := []struct {
		name           string
		status1        int
		status2        int
		trace1         *tempopb.Trace
		trace2         *tempopb.Trace
		err1           error
		err2           error
		expectedStatus int
		expectedTrace  *tempopb.Trace
	}{
		{
			name:           "empty returns",
			status1:        200,
			status2:        200,
			expectedStatus: 200,
			expectedTrace:  &tempopb.Trace{},
		},
		{
			name:           "404",
			status1:        404,
			status2:        404,
			expectedStatus: 404,
		},
		{
			name:           "400",
			status1:        400,
			status2:        400,
			expectedStatus: 500,
		},
		{
			name:           "500+200",
			status1:        500,
			status2:        200,
			trace2:         trace2,
			expectedStatus: 500,
		},
		{
			name:           "200+500",
			status1:        200,
			trace1:         trace1,
			status2:        500,
			expectedStatus: 500,
		},
		{
			name:           "200+429",
			status1:        200,
			trace1:         trace1,
			status2:        429,
			expectedStatus: 429,
		},
		{
			name:           "200+200",
			status1:        200,
			trace1:         trace1,
			status2:        200,
			trace2:         trace2,
			expectedStatus: 200,
			expectedTrace:  splitTrace,
		},
		{
			name:           "200+err",
			status1:        200,
			trace1:         trace1,
			err2:           errors.New("booo"),
			expectedStatus: 500,
		},
	}

	for _, tc := range tests {
		tc := tc // copy the test case to prevent race on the loop variable
		t.Run(tc.name, func(t *testing.T) {
			next := RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
				var testTrace *tempopb.Trace
				var statusCode int
				var err error
				if r.RequestURI == "/querier/api/v2/traces/1234?mode=ingesters" {
					testTrace = tc.trace1
					statusCode = tc.status1
					err = tc.err1
				} else {
					testTrace = tc.trace2
					err = tc.err2
					statusCode = tc.status2
				}

				if err != nil {
					return nil, err
				}

				resBytes := []byte("error occurred")
				if statusCode != 500 {
					if testTrace != nil {
						resBytes, err = proto.Marshal(&tempopb.TraceByIDResponse{
							Trace:   testTrace,
							Metrics: &tempopb.TraceByIDMetrics{},
						})
						require.NoError(t, err)
					} else {
						resBytes, err = proto.Marshal(&tempopb.TraceByIDResponse{
							Metrics: &tempopb.TraceByIDMetrics{},
						})
						require.NoError(t, err)
					}
				}

				return &http.Response{
					Body:       io.NopCloser(bytes.NewReader(resBytes)),
					StatusCode: statusCode,
					Header: map[string][]string{
						"Content-Type": {"application/protobuf"},
					},
				}, nil
			})

			// queriers will return one err
			f := frontendWithSettings(t, next, nil, config, nil)

			req := httptest.NewRequest("GET", "/api/v2/traces/1234", nil)
			ctx := req.Context()
			ctx = user.InjectOrgID(ctx, "blerg")
			req = req.WithContext(ctx)
			req = mux.SetURLVars(req, map[string]string{"traceID": "1234"})
			req.Header.Set("Accept", "application/protobuf")

			httpResp := httptest.NewRecorder()
			f.TraceByIDHandlerV2.ServeHTTP(httpResp, req)
			resp := httpResp.Result()

			if tc.expectedStatus != resp.StatusCode {
				body, _ := io.ReadAll(resp.Body)
				require.Fail(t, "unexpected status code", "expected %d, got %d. body: %s", tc.expectedStatus, resp.StatusCode, body)
			}
			if tc.expectedStatus == http.StatusOK {
				assert.Equal(t, "application/protobuf", resp.Header.Get("Content-Type"))
			}
			if tc.expectedTrace != nil {
				actualResp := &tempopb.TraceByIDResponse{}
				bytesTrace, err := io.ReadAll(resp.Body)
				require.NoError(t, err)
				err = proto.Unmarshal(bytesTrace, actualResp)
				require.NoError(t, err)

				trace.SortTrace(tc.expectedTrace)
				trace.SortTrace(actualResp.Trace)
				assert.True(t, proto.Equal(tc.expectedTrace, actualResp.Trace))
			}
		})
	}
}

func TestTraceIDHandlerV2WithJSONResponse(t *testing.T) {
	// create and split a splitTrace
	splitTrace := test.MakeTrace(2, []byte{0x01, 0x02})
	trace1 := &tempopb.Trace{}
	trace2 := &tempopb.Trace{}

	for i, b := range splitTrace.ResourceSpans {
		if i%2 == 0 {
			trace1.ResourceSpans = append(trace1.ResourceSpans, b)
		} else {
			trace2.ResourceSpans = append(trace2.ResourceSpans, b)
		}
	}

	next := RoundTripperFunc(func(_ *http.Request) (*http.Response, error) {
		var err error
		resBytes, err := proto.Marshal(&tempopb.TraceByIDResponse{
			Trace:   splitTrace,
			Metrics: &tempopb.TraceByIDMetrics{},
		})
		require.NoError(t, err)

		return &http.Response{
			Body:       io.NopCloser(bytes.NewReader(resBytes)),
			StatusCode: 200,
			Header: map[string][]string{
				"Content-Type": {"application/protobuf"},
			},
		}, nil
	})

	// queriers will return one err
	f := frontendWithSettings(t, next, nil, config, nil)

	req := httptest.NewRequest("GET", "/api/v2/traces/1234", nil)
	ctx := req.Context()
	ctx = user.InjectOrgID(ctx, "blerg")
	req = req.WithContext(ctx)
	req = mux.SetURLVars(req, map[string]string{"traceID": "1234"})
	req.Header.Set("Accept", "application/json")

	httpResp := httptest.NewRecorder()
	f.TraceByIDHandlerV2.ServeHTTP(httpResp, req)
	resp := httpResp.Result()

	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	actualResp := &tempopb.TraceByIDResponse{}
	err := new(jsonpb.Unmarshaler).Unmarshal(resp.Body, actualResp)
	require.NoError(t, err)
}
