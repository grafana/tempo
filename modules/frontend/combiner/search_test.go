package combiner

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gogo/status"
	"github.com/grafana/tempo/modules/frontend/shardtracker"
	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/search"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
)

func TestSearchProgressShouldQuitAnyJSON(t *testing.T) {
	testSearchProgressShouldQuitAny(t, api.MarshallingFormatJSON)
}

func TestSearchProgressShouldQuitAnyProtobuf(t *testing.T) {
	testSearchProgressShouldQuitAny(t, api.MarshallingFormatProtobuf)
}

func testSearchProgressShouldQuitAny(t *testing.T, marshalingFormat api.MarshallingFormat) {
	// new combiner should not quit
	c := NewSearch(0, false, marshalingFormat)
	should := c.ShouldQuit()
	require.False(t, should)

	// 500 response should quit
	c = NewSearch(0, false, marshalingFormat)
	err := c.AddResponse(toHTTPResponseWithFormat(t, &tempopb.SearchResponse{}, 500, nil, marshalingFormat))
	require.NoError(t, err)
	should = c.ShouldQuit()
	require.True(t, should)

	// 429 response should quit
	c = NewSearch(0, false, marshalingFormat)
	err = c.AddResponse(toHTTPResponseWithFormat(t, &tempopb.SearchResponse{}, 429, nil, marshalingFormat))
	require.NoError(t, err)
	should = c.ShouldQuit()
	require.True(t, should)

	// unparseable body should not quit, but should return an error
	c = NewSearch(0, false, marshalingFormat)
	err = c.AddResponse(&testPipelineResponse{r: &http.Response{Body: io.NopCloser(strings.NewReader("foo")), StatusCode: 200}})
	require.Error(t, err)
	should = c.ShouldQuit()
	require.False(t, should)

	// under limit should not quit
	c = NewSearch(2, false, marshalingFormat)
	err = c.AddResponse(toHTTPResponseWithFormat(t, &tempopb.SearchResponse{
		Traces: []*tempopb.TraceSearchMetadata{
			{
				TraceID: "1",
			},
		},
	}, 200, nil, marshalingFormat))
	require.NoError(t, err)
	should = c.ShouldQuit()
	require.False(t, should)

	// over limit should quit
	c = NewSearch(1, false, marshalingFormat)
	err = c.AddResponse(toHTTPResponseWithFormat(t, &tempopb.SearchResponse{
		Traces: []*tempopb.TraceSearchMetadata{
			{
				TraceID: "1",
			},
			{
				TraceID: "2",
			},
		},
	}, 200, nil, marshalingFormat))
	require.NoError(t, err)
	should = c.ShouldQuit()
	require.True(t, should)
}

func TestSearchProgressShouldQuitMostRecentJSON(t *testing.T) {
	testSearchProgressShouldQuitMostRecent(t, api.MarshallingFormatJSON)
}

func TestSearchProgressShouldQuitMostRecentProtobuf(t *testing.T) {
	testSearchProgressShouldQuitMostRecent(t, api.MarshallingFormatProtobuf)
}

func testSearchProgressShouldQuitMostRecent(t *testing.T, marshalingFormat api.MarshallingFormat) {
	// new combiner should not quit
	c := NewSearch(0, true, marshalingFormat)
	should := c.ShouldQuit()
	require.False(t, should)

	// 500 response should quit
	c = NewSearch(0, true, marshalingFormat)
	err := c.AddResponse(toHTTPResponseWithFormat(t, &tempopb.SearchResponse{}, 500, nil, marshalingFormat))
	require.NoError(t, err)
	should = c.ShouldQuit()
	require.True(t, should)

	// 429 response should quit
	c = NewSearch(0, true, marshalingFormat)
	err = c.AddResponse(toHTTPResponseWithFormat(t, &tempopb.SearchResponse{}, 429, nil, marshalingFormat))
	require.NoError(t, err)
	should = c.ShouldQuit()
	require.True(t, should)

	// unparseable body should not quit, but should return an error
	c = NewSearch(0, true, marshalingFormat)
	err = c.AddResponse(&testPipelineResponse{r: &http.Response{Body: io.NopCloser(strings.NewReader("foo")), StatusCode: 200}})
	require.Error(t, err)
	should = c.ShouldQuit()
	require.False(t, should)

	// under limit should not quit
	c = NewSearch(2, true, marshalingFormat)
	err = c.AddResponse(toHTTPResponseWithFormat(t, &tempopb.SearchResponse{
		Traces: []*tempopb.TraceSearchMetadata{
			{
				TraceID: "1",
			},
		},
	}, 200, nil, marshalingFormat))
	require.NoError(t, err)
	should = c.ShouldQuit()
	require.False(t, should)

	// over limit but no search job response, should not quit
	c = NewSearch(1, true, marshalingFormat)
	err = c.AddResponse(toHTTPResponseWithFormat(t, &tempopb.SearchResponse{
		Traces: []*tempopb.TraceSearchMetadata{
			{
				TraceID:           "1",
				StartTimeUnixNano: uint64(100 * time.Second),
			},
			{
				TraceID:           "2",
				StartTimeUnixNano: uint64(200 * time.Second),
			},
		},
	}, 200, 0, marshalingFormat)) // 0 is the shard index
	require.NoError(t, err)
	should = c.ShouldQuit()
	require.False(t, should)

	// send shards. should not quit b/c completed through is 300
	err = c.AddResponse(&SearchJobResponse{
		JobMetadata: shardtracker.JobMetadata{
			TotalJobs: 3,
			Shards: []shardtracker.Shard{
				{
					TotalJobs:               1,
					CompletedThroughSeconds: 300,
				},
				{
					TotalJobs:               1,
					CompletedThroughSeconds: 150,
				},
				{
					TotalJobs:               1,
					CompletedThroughSeconds: 50,
				},
			},
		},
	})
	require.NoError(t, err)
	should = c.ShouldQuit()
	require.False(t, should)

	// add complete the second shard. quit should be true b/c completed through is 150, our limit is one and we have a trace at 200
	err = c.AddResponse(toHTTPResponseWithFormat(t, &tempopb.SearchResponse{
		Traces: []*tempopb.TraceSearchMetadata{
			{
				TraceID:           "3",
				StartTimeUnixNano: uint64(50 * time.Second),
			},
		},
	}, 200, 1, marshalingFormat)) // 1 is the shard index
	require.NoError(t, err)
	should = c.ShouldQuit()
	require.True(t, should)
}

func TestSearchCombinesResultsJSON(t *testing.T) {
	testSearchCombinesResults(t, api.MarshallingFormatJSON)
}

func TestSearchCombinesResultsProtobuf(t *testing.T) {
	testSearchCombinesResults(t, api.MarshallingFormatProtobuf)
}

func testSearchCombinesResults(t *testing.T, marshalingFormat api.MarshallingFormat) {
	for _, keepMostRecent := range []bool{true, false} {
		start := time.Date(1, 2, 3, 4, 5, 6, 7, time.UTC)
		traceID := "traceID"

		c := NewSearch(10, keepMostRecent, marshalingFormat)
		sr := toHTTPResponseWithFormat(t, &tempopb.SearchResponse{
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
		}, 200, nil, marshalingFormat)
		err := c.AddResponse(sr)
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
		fromHTTPResponse(t, resp, actual)

		require.Equal(t, expected, actual)
	}
}

func TestSearchResponseCombinerJSON(t *testing.T) {
	testSearchResponseCombiner(t, api.MarshallingFormatJSON)
}

func TestSearchResponseCombinerProtobuf(t *testing.T) {
	testSearchResponseCombiner(t, api.MarshallingFormatProtobuf)
}

func testSearchResponseCombiner(t *testing.T, marshalingFormat api.MarshallingFormat) {
	for _, keepMostRecent := range []bool{true, false} {
		tests := []struct {
			name      string
			response1 PipelineResponse
			response2 PipelineResponse

			expectedStatus    int
			expectedResponse  *tempopb.SearchResponse
			expectedHTTPError error
			expectedGRPCError error
		}{
			{
				name:           "empty returns",
				response1:      toHTTPResponseWithFormat(t, &tempopb.SearchResponse{Metrics: &tempopb.SearchMetrics{}}, 200, nil, marshalingFormat),
				response2:      toHTTPResponseWithFormat(t, &tempopb.SearchResponse{Metrics: &tempopb.SearchMetrics{}}, 200, nil, marshalingFormat),
				expectedStatus: 200,
				expectedResponse: &tempopb.SearchResponse{
					Traces: []*tempopb.TraceSearchMetadata{},
					Metrics: &tempopb.SearchMetrics{
						CompletedJobs: 2,
					},
				},
			},
			{
				name:              "404+200",
				response1:         toHTTPResponseWithFormat(t, nil, 404, nil, marshalingFormat),
				response2:         toHTTPResponseWithFormat(t, &tempopb.SearchResponse{Metrics: &tempopb.SearchMetrics{}}, 200, nil, marshalingFormat),
				expectedStatus:    404,
				expectedGRPCError: status.Error(codes.NotFound, ""),
			},
			{
				name:              "200+400",
				response1:         toHTTPResponseWithFormat(t, &tempopb.SearchResponse{Metrics: &tempopb.SearchMetrics{}}, 200, nil, marshalingFormat),
				response2:         toHTTPResponseWithFormat(t, nil, 400, nil, marshalingFormat),
				expectedStatus:    400,
				expectedGRPCError: status.Error(codes.InvalidArgument, ""),
			},
			{
				name:              "200+429",
				response1:         toHTTPResponseWithFormat(t, &tempopb.SearchResponse{Metrics: &tempopb.SearchMetrics{}}, 200, nil, marshalingFormat),
				response2:         toHTTPResponseWithFormat(t, nil, 429, nil, marshalingFormat),
				expectedStatus:    429,
				expectedGRPCError: status.Error(codes.ResourceExhausted, ""),
			},
			{
				name:              "500+404",
				response1:         toHTTPResponseWithFormat(t, nil, 500, nil, marshalingFormat),
				response2:         toHTTPResponseWithFormat(t, nil, 404, nil, marshalingFormat),
				expectedStatus:    500,
				expectedGRPCError: status.Error(codes.Internal, ""),
			},
			{
				name:              "404+500 - first bad response wins",
				response1:         toHTTPResponseWithFormat(t, nil, 404, nil, marshalingFormat),
				response2:         toHTTPResponseWithFormat(t, nil, 500, nil, marshalingFormat),
				expectedStatus:    404,
				expectedGRPCError: status.Error(codes.NotFound, ""),
			},
			{
				name:              "500+200",
				response1:         toHTTPResponseWithFormat(t, nil, 500, nil, marshalingFormat),
				response2:         toHTTPResponseWithFormat(t, &tempopb.SearchResponse{Metrics: &tempopb.SearchMetrics{}}, 200, nil, marshalingFormat),
				expectedStatus:    500,
				expectedGRPCError: status.Error(codes.Internal, ""),
			},
			{
				name:              "200+500",
				response1:         toHTTPResponseWithFormat(t, &tempopb.SearchResponse{Metrics: &tempopb.SearchMetrics{}}, 200, nil, marshalingFormat),
				response2:         toHTTPResponseWithFormat(t, nil, 500, nil, marshalingFormat),
				expectedStatus:    500,
				expectedGRPCError: status.Error(codes.Internal, ""),
			},
			{
				name: "respects total blocks message",
				response1: &SearchJobResponse{
					JobMetadata: shardtracker.JobMetadata{
						TotalBlocks: 5,
						TotalJobs:   10,
						TotalBytes:  15,
					},
				},
				response2: toHTTPResponseWithFormat(t, &tempopb.SearchResponse{
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
				}, 200, nil, marshalingFormat),
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
				response1: toHTTPResponseWithFormat(t, &tempopb.SearchResponse{
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
				}, 200, nil, marshalingFormat),
				response2: toHTTPResponseWithFormat(t, &tempopb.SearchResponse{
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
				}, 200, nil, marshalingFormat),
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
				combiner := NewTypedSearch(20, keepMostRecent, marshalingFormat)

				err := combiner.AddResponse(tc.response1)
				require.NoError(t, err)
				err = combiner.AddResponse(tc.response2)
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
}

func TestCombinerShardsJSON(t *testing.T) {
	testCombinerShards(t, api.MarshallingFormatJSON)
}

func TestCombinerShardsProtobuf(t *testing.T) {
	testCombinerShards(t, api.MarshallingFormatProtobuf)
}

func testCombinerShards(t *testing.T, marshalingFormat api.MarshallingFormat) {
	tests := []struct {
		name             string
		pipelineResponse PipelineResponse
		expected         *tempopb.SearchResponse
	}{
		{
			name:             "initial state",
			pipelineResponse: nil,
			expected: &tempopb.SearchResponse{
				Traces:  []*tempopb.TraceSearchMetadata{},
				Metrics: &tempopb.SearchMetrics{},
			},
		},
		{
			name: "add job metadata",
			pipelineResponse: &SearchJobResponse{
				JobMetadata: shardtracker.JobMetadata{
					TotalBlocks: 5,
					TotalJobs:   6,
					TotalBytes:  15,
					Shards: []shardtracker.Shard{ // 5 shards, 2 jobs each. starting at 500 seconds and walking back 100 seconds each
						{
							TotalJobs:               2,
							CompletedThroughSeconds: 500,
						},
						{
							TotalJobs:               1,
							CompletedThroughSeconds: 400,
						},
						{
							TotalJobs:               1,
							CompletedThroughSeconds: 300,
						},
						{
							TotalJobs:               1,
							CompletedThroughSeconds: 200,
						},
						{
							TotalJobs:               1,
							CompletedThroughSeconds: 100,
						},
					},
				},
			},
			expected: &tempopb.SearchResponse{
				Traces: []*tempopb.TraceSearchMetadata{},
				Metrics: &tempopb.SearchMetrics{
					TotalBlocks:     5,
					TotalJobs:       6,
					TotalBlockBytes: 15,
				},
			},
		},
		{
			name: "add response results",
			pipelineResponse: toHTTPResponseWithFormat(t, &tempopb.SearchResponse{
				Traces: []*tempopb.TraceSearchMetadata{
					{
						TraceID:           "450",
						RootServiceName:   "root-450",
						StartTimeUnixNano: uint64(450 * time.Second),
					},
					{
						TraceID:           "550",
						RootServiceName:   "root-550",
						StartTimeUnixNano: uint64(550 * time.Second),
					},
				},
				Metrics: &tempopb.SearchMetrics{
					InspectedTraces: 1,
					InspectedBytes:  2,
				},
			}, 200, 0, marshalingFormat), // shard 0
			expected: &tempopb.SearchResponse{
				Traces: []*tempopb.TraceSearchMetadata{}, // no traces b/c only one job has finished and the first shard has 2 jobs
				Metrics: &tempopb.SearchMetrics{ // metadata is incrementing
					CompletedJobs:   1,
					InspectedTraces: 1,
					InspectedBytes:  2,
					TotalBlocks:     5,
					TotalJobs:       6,
					TotalBlockBytes: 15,
				},
			},
		},
		{
			name: "add second job to finish the first shard and get one result",
			pipelineResponse: toHTTPResponseWithFormat(t, &tempopb.SearchResponse{
				Traces: []*tempopb.TraceSearchMetadata{
					{
						TraceID:           "350",
						RootServiceName:   "root-350",
						StartTimeUnixNano: uint64(350 * time.Second),
					},
				},
				Metrics: &tempopb.SearchMetrics{
					InspectedTraces: 1,
					InspectedBytes:  2,
				},
			}, 200, 0, marshalingFormat), // shard 0,
			expected: &tempopb.SearchResponse{
				Traces: []*tempopb.TraceSearchMetadata{
					{
						TraceID:           "550",
						RootServiceName:   "root-550",
						StartTimeUnixNano: uint64(550 * time.Second),
					},
				},
				Metrics: &tempopb.SearchMetrics{ // metadata is incrementing
					CompletedJobs:   2,
					InspectedTraces: 2,
					InspectedBytes:  4,
					TotalBlocks:     5,
					TotalJobs:       6,
					TotalBlockBytes: 15,
				},
			},
		},
		{
			name: "update response results",
			pipelineResponse: toHTTPResponseWithFormat(t, &tempopb.SearchResponse{
				Traces: []*tempopb.TraceSearchMetadata{
					{
						TraceID:           "550",
						RootServiceName:   "root-550",
						RootTraceName:     "root-550",
						StartTimeUnixNano: uint64(550 * time.Second),
					},
				},
				Metrics: &tempopb.SearchMetrics{
					InspectedTraces: 1,
					InspectedBytes:  2,
				},
			}, 200, 1, marshalingFormat), // complete shard 1
			expected: &tempopb.SearchResponse{
				Traces: []*tempopb.TraceSearchMetadata{
					{ // included b/c updated
						TraceID:           "550",
						RootServiceName:   "root-550",
						RootTraceName:     "root-550",
						StartTimeUnixNano: uint64(550 * time.Second),
					},
					{ // included b/c second shard is done
						TraceID:           "450",
						RootServiceName:   "root-450",
						StartTimeUnixNano: uint64(450 * time.Second),
					},
				},
				Metrics: &tempopb.SearchMetrics{ // metadata is incrementing
					CompletedJobs:   3,
					InspectedTraces: 3,
					InspectedBytes:  6,
					TotalBlocks:     5,
					TotalJobs:       6,
					TotalBlockBytes: 15,
				},
			},
		},
		{
			name: "skip a shard and see no change",
			pipelineResponse: toHTTPResponseWithFormat(t, &tempopb.SearchResponse{
				Traces: []*tempopb.TraceSearchMetadata{
					{
						TraceID:           "50",
						RootServiceName:   "root-50",
						StartTimeUnixNano: uint64(50 * time.Second),
					},
				},
				Metrics: &tempopb.SearchMetrics{
					InspectedTraces: 1,
					InspectedBytes:  2,
				},
			}, 200, 3, marshalingFormat), // complete shard 3,
			expected: &tempopb.SearchResponse{
				Traces: []*tempopb.TraceSearchMetadata{}, // no traces b/c we skipped shard 2 and we can't include results from 3 until 2 is done
				Metrics: &tempopb.SearchMetrics{ // metadata is incrementing
					CompletedJobs:   4,
					InspectedTraces: 4,
					InspectedBytes:  8,
					TotalBlocks:     5,
					TotalJobs:       6,
					TotalBlockBytes: 15,
				},
			},
		},
		{
			name: "fill in shard 2 and see results",
			pipelineResponse: toHTTPResponseWithFormat(t, &tempopb.SearchResponse{
				Traces: []*tempopb.TraceSearchMetadata{},
				Metrics: &tempopb.SearchMetrics{
					InspectedTraces: 1,
					InspectedBytes:  2,
				},
			}, 200, 2, marshalingFormat), // complete shard 2,
			expected: &tempopb.SearchResponse{
				Traces: []*tempopb.TraceSearchMetadata{
					{
						TraceID:           "350",
						RootServiceName:   "root-350",
						StartTimeUnixNano: uint64(350 * time.Second),
					},
				},
				Metrics: &tempopb.SearchMetrics{ // metadata is incrementing
					CompletedJobs:   5,
					InspectedTraces: 5,
					InspectedBytes:  10,
					TotalBlocks:     5,
					TotalJobs:       6,
					TotalBlockBytes: 15,
				},
			},
		},
		{
			name: "complete all shards which dumps all results",
			pipelineResponse: toHTTPResponseWithFormat(t, &tempopb.SearchResponse{
				Traces: []*tempopb.TraceSearchMetadata{},
				Metrics: &tempopb.SearchMetrics{
					InspectedTraces: 1,
					InspectedBytes:  2,
				},
			}, 200, 4, marshalingFormat), // complete shard 4,
			expected: &tempopb.SearchResponse{
				Traces: []*tempopb.TraceSearchMetadata{
					{
						TraceID:           "50",
						RootServiceName:   "root-50",
						StartTimeUnixNano: uint64(50 * time.Second),
					},
				}, // 50 is BEFORE the earliest trace shard, but it's still returned b/c at this point we have completed all jobs
				Metrics: &tempopb.SearchMetrics{ // metadata is incrementing
					CompletedJobs:   6,
					InspectedTraces: 6,
					InspectedBytes:  12,
					TotalBlocks:     5,
					TotalJobs:       6,
					TotalBlockBytes: 15,
				},
			},
		},
	}

	// apply tests one at a time to the combiner and check expected results

	combiner := NewTypedSearch(5, true, marshalingFormat)
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.pipelineResponse != nil {
				err := combiner.AddResponse(tc.pipelineResponse)
				require.NoError(t, err)
			}

			resp, err := combiner.GRPCDiff()
			require.NoError(t, err)
			require.Equal(t, tc.expected, resp)
		})
	}
}
