package combiner

import (
	"io"
	"math"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
	"github.com/gogo/status"
	"github.com/grafana/tempo/pkg/search"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
)

func TestSearchProgressShouldQuit(t *testing.T) {
	// new combiner should not quit
	c := NewSearch(0)
	should := c.ShouldQuit()
	require.False(t, should)

	// 500 response should quit
	c = NewSearch(0)
	err := c.AddRequest(toHttpResponse(t, &tempopb.SearchResponse{}, 500), "")
	require.NoError(t, err)
	should = c.ShouldQuit()
	require.True(t, should)

	// 429 response should quit
	c = NewSearch(0)
	err = c.AddRequest(toHttpResponse(t, &tempopb.SearchResponse{}, 429), "")
	require.NoError(t, err)
	should = c.ShouldQuit()
	require.True(t, should)

	// unparseable body should not quit, but should return an error
	c = NewSearch(0)
	err = c.AddRequest(&http.Response{Body: io.NopCloser(strings.NewReader("foo")), StatusCode: 200}, "")
	require.Error(t, err)
	should = c.ShouldQuit()
	require.False(t, should)

	// under limit should not quit
	c = NewSearch(2)
	err = c.AddRequest(toHttpResponse(t, &tempopb.SearchResponse{
		Traces: []*tempopb.TraceSearchMetadata{
			{
				TraceID: "1",
			},
		},
	}, 200), "")
	require.NoError(t, err)
	should = c.ShouldQuit()
	require.False(t, should)

	// over limit should quit
	c = NewSearch(1)
	err = c.AddRequest(toHttpResponse(t, &tempopb.SearchResponse{
		Traces: []*tempopb.TraceSearchMetadata{
			{
				TraceID: "1",
			},
			{
				TraceID: "2",
			},
		},
	}, 200), "")
	require.NoError(t, err)
	should = c.ShouldQuit()
	require.True(t, should)
}

func TestSearchCombinesResults(t *testing.T) {
	start := time.Date(1, 2, 3, 4, 5, 6, 7, time.UTC)
	traceID := "traceID"

	c := NewSearch(10)
	sr := toHttpResponse(t, &tempopb.SearchResponse{
		Traces: []*tempopb.TraceSearchMetadata{
			{
				TraceID:           traceID,
				StartTimeUnixNano: uint64(start.Add(time.Second).UnixNano()),
				DurationMs:        uint32(time.Second.Milliseconds()),
			}, // 1 second after start and shorter duration
			{
				TraceID:           traceID,
				StartTimeUnixNano: uint64(start.UnixNano()),
				DurationMs:        uint32(time.Hour.Milliseconds()),
			}, // earliest start time and longer duration
			{
				TraceID:           traceID,
				StartTimeUnixNano: uint64(start.Add(time.Hour).UnixNano()),
				DurationMs:        uint32(time.Millisecond.Milliseconds()),
			}, // 1 hour after start and shorter duration
		},
		Metrics: &tempopb.SearchMetrics{},
	}, 200)
	err := c.AddRequest(sr, "")
	require.NoError(t, err)

	expected := &tempopb.SearchResponse{
		Traces: []*tempopb.TraceSearchMetadata{
			{
				TraceID:           traceID,
				StartTimeUnixNano: uint64(start.UnixNano()),
				DurationMs:        uint32(time.Hour.Milliseconds()),
				RootServiceName:   search.RootSpanNotYetReceivedText,
			},
		},
		Metrics: &tempopb.SearchMetrics{
			CompletedJobs: 1,
		},
	}

	resp, err := c.HTTPFinal()
	require.NoError(t, err)

	actual := &tempopb.SearchResponse{}
	fromHttpResponse(t, resp, actual)

	require.Equal(t, expected, actual)
}

func TestSearchResponseCombiner(t *testing.T) {
	tests := []struct {
		name      string
		response1 *http.Response
		response2 *http.Response

		expectedStatus    int
		expectedResponse  *tempopb.SearchResponse
		expectedHTTPError error
		expectedGRPCError error
	}{
		{
			name:           "empty returns",
			response1:      toHttpResponse(t, &tempopb.SearchResponse{Metrics: &tempopb.SearchMetrics{}}, 200),
			response2:      toHttpResponse(t, &tempopb.SearchResponse{Metrics: &tempopb.SearchMetrics{}}, 200),
			expectedStatus: 200,
			expectedResponse: &tempopb.SearchResponse{
				Traces: []*tempopb.TraceSearchMetadata{},
				Metrics: &tempopb.SearchMetrics{
					CompletedJobs: 2,
				}},
		},
		{
			name:              "404+200",
			response1:         toHttpResponse(t, nil, 404),
			response2:         toHttpResponse(t, &tempopb.SearchResponse{Metrics: &tempopb.SearchMetrics{}}, 200),
			expectedStatus:    404,
			expectedGRPCError: status.Error(codes.InvalidArgument, ""),
		},
		{
			name:              "200+400",
			response1:         toHttpResponse(t, &tempopb.SearchResponse{Metrics: &tempopb.SearchMetrics{}}, 200),
			response2:         toHttpResponse(t, nil, 400),
			expectedStatus:    400,
			expectedGRPCError: status.Error(codes.InvalidArgument, ""),
		},
		{
			name:              "200+429",
			response1:         toHttpResponse(t, &tempopb.SearchResponse{Metrics: &tempopb.SearchMetrics{}}, 200),
			response2:         toHttpResponse(t, nil, 429),
			expectedStatus:    429,
			expectedGRPCError: status.Error(codes.InvalidArgument, ""),
		},
		{
			name:              "500+404",
			response1:         toHttpResponse(t, nil, 500),
			response2:         toHttpResponse(t, nil, 404),
			expectedStatus:    500,
			expectedGRPCError: status.Error(codes.Internal, ""),
		},
		{
			name:              "404+500 - first bad response wins",
			response1:         toHttpResponse(t, nil, 404),
			response2:         toHttpResponse(t, nil, 500),
			expectedStatus:    404,
			expectedGRPCError: status.Error(codes.InvalidArgument, ""),
		},
		{
			name:              "500+200",
			response1:         toHttpResponse(t, nil, 500),
			response2:         toHttpResponse(t, &tempopb.SearchResponse{Metrics: &tempopb.SearchMetrics{}}, 200),
			expectedStatus:    500,
			expectedGRPCError: status.Error(codes.Internal, ""),
		},
		{
			name:              "200+500",
			response1:         toHttpResponse(t, &tempopb.SearchResponse{Metrics: &tempopb.SearchMetrics{}}, 200),
			response2:         toHttpResponse(t, nil, 500),
			expectedStatus:    500,
			expectedGRPCError: status.Error(codes.Internal, ""),
		},
		{
			name: "respects total blocks message",
			response1: toHttpResponse(t, &tempopb.SearchResponse{
				Traces: nil,
				Metrics: &tempopb.SearchMetrics{
					TotalBlocks:     5,
					TotalJobs:       10,
					TotalBlockBytes: 15,
				},
			}, 200),
			response2: toHttpResponse(t, &tempopb.SearchResponse{
				Traces: []*tempopb.TraceSearchMetadata{
					{
						TraceID:           "5678",
						StartTimeUnixNano: 0,
					},
				},
				Metrics: &tempopb.SearchMetrics{
					InspectedTraces: 5,
					InspectedBytes:  7,
				},
			}, 200),
			expectedStatus: 200,
			expectedResponse: &tempopb.SearchResponse{
				Traces: []*tempopb.TraceSearchMetadata{
					{
						TraceID:           "5678",
						StartTimeUnixNano: 0,
						RootServiceName:   search.RootSpanNotYetReceivedText,
					},
				},
				Metrics: &tempopb.SearchMetrics{
					TotalBlocks:     5,
					TotalJobs:       10,
					TotalBlockBytes: 15,
					InspectedTraces: 5,
					InspectedBytes:  7,
					CompletedJobs:   1,
				},
			},
		},
		{
			name: "200+200",
			response1: toHttpResponse(t, &tempopb.SearchResponse{
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
			}, 200),
			response2: toHttpResponse(t, &tempopb.SearchResponse{
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
			}, 200),
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
					InspectedBytes:  10,
					CompletedJobs:   2,
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			combiner := NewTypedSearch(20)

			err := combiner.AddRequest(tc.response1, "")
			require.NoError(t, err)
			err = combiner.AddRequest(tc.response2, "")
			require.NoError(t, err)

			httpResp, err := combiner.HTTPFinal()
			require.Equal(t, tc.expectedStatus, httpResp.StatusCode)
			require.Equal(t, tc.expectedHTTPError, err)

			grpcresp, err := combiner.GRPCFinal()
			require.Equal(t, tc.expectedGRPCError, err)
			require.Equal(t, tc.expectedResponse, grpcresp)
		})
	}
}

func TestSearchDiffsResults(t *testing.T) {
	traceID := "traceID"

	c := NewTypedSearch(10)
	sr := toHttpResponse(t, &tempopb.SearchResponse{
		Traces: []*tempopb.TraceSearchMetadata{
			{
				TraceID: traceID,
			},
		},
		Metrics: &tempopb.SearchMetrics{},
	}, 200)
	expectedDiff := &tempopb.SearchResponse{
		Traces: []*tempopb.TraceSearchMetadata{
			{
				TraceID:         traceID,
				RootServiceName: search.RootSpanNotYetReceivedText,
			},
		},
		Metrics: &tempopb.SearchMetrics{
			CompletedJobs: 1,
		},
	}
	expectedNoDiff := &tempopb.SearchResponse{
		Traces:  []*tempopb.TraceSearchMetadata{},
		Metrics: &tempopb.SearchMetrics{},
	}

	// haven't added anything yet
	actual, err := c.GRPCDiff()
	require.NoError(t, err)
	require.Equal(t, expectedNoDiff, actual)

	// add a trace and get it back in diff
	err = c.AddRequest(sr, "")
	require.NoError(t, err)

	actual, err = c.GRPCDiff()
	require.NoError(t, err)
	require.Equal(t, expectedDiff, actual)

	// now we should get no diff again (with 1 completed job)
	expectedNoDiff.Metrics.CompletedJobs = 1
	actual, err = c.GRPCDiff()
	require.NoError(t, err)
	require.Equal(t, expectedNoDiff, actual)

	// let's add a different trace and get it back in diff
	traceID2 := "traceID2"
	sr2 := toHttpResponse(t, &tempopb.SearchResponse{
		Traces: []*tempopb.TraceSearchMetadata{
			{
				TraceID: traceID2,
			},
		},
		Metrics: &tempopb.SearchMetrics{},
	}, 200)
	expectedDiff2 := &tempopb.SearchResponse{
		Traces: []*tempopb.TraceSearchMetadata{
			{
				TraceID:         traceID2,
				RootServiceName: search.RootSpanNotYetReceivedText,
			},
		},
		Metrics: &tempopb.SearchMetrics{
			CompletedJobs: 2, // we will have 2 completed jobs at this point
		},
	}

	err = c.AddRequest(sr2, "")
	require.NoError(t, err)

	actual, err = c.GRPCDiff()
	require.NoError(t, err)
	require.Equal(t, expectedDiff2, actual)
}

func toHttpResponse(t *testing.T, pb proto.Message, statusCode int) *http.Response {
	var body string

	if pb != nil {
		var err error
		m := jsonpb.Marshaler{}
		body, err = m.MarshalToString(pb)
		require.NoError(t, err)
	}

	return &http.Response{
		Body:       io.NopCloser(strings.NewReader(body)),
		StatusCode: statusCode,
	}
}

func fromHttpResponse(t *testing.T, r *http.Response, pb proto.Message) {
	err := jsonpb.Unmarshal(r.Body, pb)
	require.NoError(t, err)
}

func TestCombinerDiffs(t *testing.T) {
	combiner := NewTypedSearch(100)

	// first request should be empty
	resp, err := combiner.GRPCDiff()
	require.NoError(t, err)
	require.Equal(t, &tempopb.SearchResponse{
		Traces:  []*tempopb.TraceSearchMetadata{},
		Metrics: &tempopb.SearchMetrics{},
	}, resp)

	err = combiner.AddRequest(toHttpResponse(t, &tempopb.SearchResponse{
		Traces: []*tempopb.TraceSearchMetadata{
			{
				TraceID:         "1234",
				RootServiceName: "root",
			},
		},
		Metrics: &tempopb.SearchMetrics{
			InspectedTraces: 1,
			InspectedBytes:  2,
		},
	}, 200), "")
	require.NoError(t, err)

	// now we should get the same metadata as above
	resp, err = combiner.GRPCDiff()
	require.NoError(t, err)
	require.Equal(t, &tempopb.SearchResponse{
		Traces: []*tempopb.TraceSearchMetadata{
			{
				TraceID:         "1234",
				RootServiceName: "root",
			},
		},
		Metrics: &tempopb.SearchMetrics{
			CompletedJobs:   1,
			InspectedTraces: 1,
			InspectedBytes:  2,
		},
	}, resp)

	// metrics, but the trace hasn't change
	resp, err = combiner.GRPCDiff()
	require.NoError(t, err)
	require.Equal(t, &tempopb.SearchResponse{
		Traces: []*tempopb.TraceSearchMetadata{},
		Metrics: &tempopb.SearchMetrics{
			CompletedJobs:   1,
			InspectedTraces: 1,
			InspectedBytes:  2,
		},
	}, resp)

	// new traces
	err = combiner.AddRequest(toHttpResponse(t, &tempopb.SearchResponse{
		Traces: []*tempopb.TraceSearchMetadata{
			{
				TraceID:           "5678",
				RootServiceName:   "root",
				StartTimeUnixNano: 1, // forces order
			},
			{
				TraceID:           "9011",
				RootServiceName:   "root",
				StartTimeUnixNano: 2,
			},
		},
		Metrics: &tempopb.SearchMetrics{
			InspectedTraces: 1,
			InspectedBytes:  2,
		},
	}, 200), "")
	require.NoError(t, err)

	resp, err = combiner.GRPCDiff()
	require.NoError(t, err)
	require.Equal(t, &tempopb.SearchResponse{
		Traces: []*tempopb.TraceSearchMetadata{
			{
				TraceID:           "9011",
				RootServiceName:   "root",
				StartTimeUnixNano: 2,
			},
			{
				TraceID:           "5678",
				RootServiceName:   "root",
				StartTimeUnixNano: 1,
			},
		},
		Metrics: &tempopb.SearchMetrics{
			CompletedJobs:   2,
			InspectedTraces: 2,
			InspectedBytes:  4,
		},
	}, resp)

	// write over existing trace
	err = combiner.AddRequest(toHttpResponse(t, &tempopb.SearchResponse{
		Traces: []*tempopb.TraceSearchMetadata{
			{
				TraceID:    "1234",
				DurationMs: 100,
			},
		},
		Metrics: &tempopb.SearchMetrics{
			InspectedTraces: 1,
			InspectedBytes:  2,
		},
	}, 200), "")
	require.NoError(t, err)

	resp, err = combiner.GRPCDiff()
	require.NoError(t, err)
	require.Equal(t, &tempopb.SearchResponse{
		Traces: []*tempopb.TraceSearchMetadata{
			{
				TraceID:         "1234",
				RootServiceName: "root",
				DurationMs:      100,
			},
		},
		Metrics: &tempopb.SearchMetrics{
			CompletedJobs:   3,
			InspectedTraces: 3,
			InspectedBytes:  6,
		},
	}, resp)
}

func TestSearchCombinerDoesNotRace(t *testing.T) {
	end := make(chan struct{})
	concurrent := func(f func()) {
		for {
			select {
			case <-end:
				return
			default:
				f()
			}
		}
	}

	traceID := "1234"
	combiner := NewTypedSearch(10)
	i := 0
	go concurrent(func() {
		i++
		resp := toHttpResponse(t, &tempopb.SearchResponse{
			Traces: []*tempopb.TraceSearchMetadata{
				{
					TraceID:           traceID,
					StartTimeUnixNano: math.MaxUint64 - uint64(i),
					DurationMs:        uint32(i),
					SpanSets: []*tempopb.SpanSet{{
						Matched: uint32(i),
					}},
				},
			},
			Metrics: &tempopb.SearchMetrics{
				InspectedTraces: 1,
				InspectedBytes:  1,
				TotalBlocks:     1,
				TotalJobs:       1,
				CompletedJobs:   1,
			},
		}, 200)
		combiner.AddRequest(resp, "")
	})

	go concurrent(func() {
		_, _ = combiner.GRPCFinal()
	})

	go concurrent(func() {
		_, _ = combiner.HTTPFinal()
	})

	time.Sleep(100 * time.Millisecond)
	close(end)
	// Wait for go funcs to quit before
	// exiting and cleaning up
	time.Sleep(2 * time.Second)
}
