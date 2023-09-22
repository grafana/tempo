package frontend

import (
	"bytes"
	"context"
	"errors"
	"io"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/gogo/protobuf/proto"
	"github.com/grafana/dskit/user"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/atomic"

	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/blockboundary"
	"github.com/grafana/tempo/pkg/model/trace"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util/test"
)

func TestBuildShardedRequests(t *testing.T) {
	queryShards := 2

	sharder := &shardQuery{
		cfg: &TraceByIDConfig{
			QueryShards: queryShards,
		},
		blockBoundaries: blockboundary.CreateBlockBoundaries(queryShards - 1),
	}

	ctx := user.InjectOrgID(context.Background(), "blerg")
	req := httptest.NewRequest("GET", "/", nil).WithContext(ctx)

	shardedReqs, err := sharder.buildShardedRequests(req)
	require.NoError(t, err)
	require.Len(t, shardedReqs, queryShards)

	require.Equal(t, "/querier?mode=ingesters", shardedReqs[0].RequestURI)
	require.Equal(t, "/querier?blockEnd=ffffffffffffffffffffffffffffffff&blockStart=00000000000000000000000000000000&mode=blocks", shardedReqs[1].RequestURI)
}

func TestShardingWareDoRequest(t *testing.T) {
	// create and split a splitTrace
	splitTrace := test.MakeTrace(10, []byte{0x01, 0x02})
	trace1 := &tempopb.Trace{}
	trace2 := &tempopb.Trace{}

	for _, b := range splitTrace.Batches {
		if rand.Int()%2 == 0 {
			trace1.Batches = append(trace1.Batches, b)
		} else {
			trace2.Batches = append(trace2.Batches, b)
		}
	}

	tests := []struct {
		name                string
		status1             int
		status2             int
		trace1              *tempopb.Trace
		trace2              *tempopb.Trace
		err1                error
		err2                error
		failedBlockQueries1 int
		failedBlockQueries2 int
		expectedStatus      int
		expectedTrace       *tempopb.Trace
		expectedError       error
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
			name:           "503+200",
			status1:        503,
			status2:        200,
			trace2:         trace2,
			expectedStatus: 500,
		},
		{
			name:           "200+503",
			status1:        200,
			trace1:         trace1,
			status2:        503,
			expectedStatus: 500,
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
			name:          "200+err",
			status1:       200,
			trace1:        trace1,
			err2:          errors.New("booo"),
			expectedError: errors.New("booo"),
		},
		{
			name:          "err+200",
			err1:          errors.New("booo"),
			status2:       200,
			trace2:        trace1,
			expectedError: errors.New("booo"),
		},
		{
			name:          "500+err",
			status1:       500,
			trace1:        trace1,
			err2:          errors.New("booo"),
			expectedError: errors.New("booo"),
		},
	}

	o, err := overrides.NewOverrides(overrides.Config{})
	require.NoError(t, err)

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sharder := newTraceByIDSharder(&TraceByIDConfig{
				QueryShards: 2,
			}, o, log.NewNopLogger())

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

			testRT := NewRoundTripper(next, sharder)

			req := httptest.NewRequest("GET", "/api/traces/1234", nil)
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

func TestConcurrentShards(t *testing.T) {
	concurrency := 2

	o, err := overrides.NewOverrides(overrides.Config{})
	require.NoError(t, err)

	sharder := newTraceByIDSharder(&TraceByIDConfig{
		QueryShards:      20,
		ConcurrentShards: concurrency,
	}, o, log.NewNopLogger())

	sawMaxConcurrncy := atomic.NewBool(false)
	currentlyExecuting := atomic.NewInt32(0)
	next := RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
		current := currentlyExecuting.Inc()
		if current > int32(concurrency) {
			t.Fatal("too many concurrent requests")
		}
		if current == int32(concurrency) {
			// future developer. i'm concerned under pressure this won't be set b/c only 1 request will be executed at a time
			// feel free to remove
			sawMaxConcurrncy.Store(true)
		}

		// force concurrency
		time.Sleep(100 * time.Millisecond)
		resBytes, err := proto.Marshal(&tempopb.TraceByIDResponse{
			Trace:   &tempopb.Trace{},
			Metrics: &tempopb.TraceByIDMetrics{},
		})
		require.NoError(t, err)

		currentlyExecuting.Dec()
		return &http.Response{
			Body:       io.NopCloser(bytes.NewReader(resBytes)),
			StatusCode: 200,
		}, nil
	})

	testRT := NewRoundTripper(next, sharder)

	req := httptest.NewRequest("GET", "/api/traces/1234", nil)
	ctx := req.Context()
	ctx = user.InjectOrgID(ctx, "blerg")
	req = req.WithContext(ctx)

	_, err = testRT.RoundTrip(req)
	require.NoError(t, err)
	require.True(t, sawMaxConcurrncy.Load())
}
