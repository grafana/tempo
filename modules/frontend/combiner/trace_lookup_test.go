package combiner

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/gogo/protobuf/jsonpb"

	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/stretchr/testify/require"
)

func TestTraceLookupShouldQuit(t *testing.T) {
	// new combiner should not quit
	c := NewTraceLookup()
	should := c.ShouldQuit()
	require.False(t, should)

	// 500 response should quit
	c = NewTraceLookup()
	err := c.AddResponse(toHTTPResponse(t, &tempopb.TraceLookupResponse{}, 500))
	require.NoError(t, err)
	should = c.ShouldQuit()
	require.True(t, should)

	// 429 response should quit
	c = NewTraceLookup()
	err = c.AddResponse(toHTTPResponse(t, &tempopb.TraceLookupResponse{}, 429))
	require.NoError(t, err)
	should = c.ShouldQuit()
	require.True(t, should)

	// 404 response should quit
	c = NewTraceLookup()
	err = c.AddResponse(toHTTPResponse(t, &tempopb.TraceLookupResponse{}, 404))
	require.NoError(t, err)
	should = c.ShouldQuit()
	require.True(t, should)

	// unparseable body should not quit, but should return an error
	c = NewTraceLookup()
	err = c.AddResponse(&testPipelineResponse{r: &http.Response{Body: io.NopCloser(strings.NewReader("foo")), StatusCode: 200}})
	require.Error(t, err)
	should = c.ShouldQuit()
	require.False(t, should)
}

func TestTraceLookupCombinesResults(t *testing.T) {
	tests := []struct {
		name      string
		responses []*tempopb.TraceLookupResponse
		expected  *tempopb.TraceLookupResponse
	}{
		{
			name: "single response",
			responses: []*tempopb.TraceLookupResponse{
				{
					TraceIDs: []string{
						"trace1",
						"trace2",
					},
					Metrics: &tempopb.SearchMetrics{InspectedBytes: 100},
				},
			},
			expected: &tempopb.TraceLookupResponse{
				TraceIDs: []string{
					"trace1",
					"trace2",
				},
				Metrics: &tempopb.SearchMetrics{
					InspectedBytes: 100,
					CompletedJobs:  1,
				},
			},
		},
		{
			name: "multiple responses - union of results",
			responses: []*tempopb.TraceLookupResponse{
				{
					TraceIDs: []string{
						"trace1",
						"trace2",
						"trace3",
					},
					Metrics: &tempopb.SearchMetrics{InspectedBytes: 100},
				},
				{
					TraceIDs: []string{
						"trace1",
						"trace2",
						"trace4",
					},
					Metrics: &tempopb.SearchMetrics{InspectedBytes: 200},
				},
			},
			expected: &tempopb.TraceLookupResponse{
				TraceIDs: []string{
					"trace1",
					"trace2",
					"trace3",
					"trace4",
				},
				Metrics: &tempopb.SearchMetrics{
					InspectedBytes: 300,
					CompletedJobs:  2,
				},
			},
		},
		{
			name: "empty responses",
			responses: []*tempopb.TraceLookupResponse{
				{
					TraceIDs: []string{},
					Metrics: &tempopb.SearchMetrics{InspectedBytes: 0},
				},
			},
			expected: &tempopb.TraceLookupResponse{
				TraceIDs: []string{},
				Metrics: &tempopb.SearchMetrics{
					InspectedBytes: 0,
					CompletedJobs:  1,
				},
			},
		},
		{
			name: "nil metrics handled gracefully",
			responses: []*tempopb.TraceLookupResponse{
				{
					TraceIDs: []string{
						"trace1",
					},
					Metrics: nil,
				},
			},
			expected: &tempopb.TraceLookupResponse{
				TraceIDs: []string{
					"trace1",
				},
				Metrics: &tempopb.SearchMetrics{
					InspectedBytes: 0,
					CompletedJobs:  1,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewTraceLookup()

			for _, resp := range tt.responses {
				err := c.AddResponse(toHTTPResponse(t, resp, 200))
				require.NoError(t, err)
			}

			httpResp, err := c.HTTPFinal()
			require.NoError(t, err)
			require.Equal(t, 200, httpResp.StatusCode)

			// Read and unmarshal the response
			bodyBytes, err := io.ReadAll(httpResp.Body)
			require.NoError(t, err)

			actual := &tempopb.TraceLookupResponse{}
			err = jsonpb.UnmarshalString(string(bodyBytes), actual)
			require.NoError(t, err)

			require.Equal(t, tt.expected.TraceIDs, actual.TraceIDs)
			require.Equal(t, tt.expected.Metrics.InspectedBytes, actual.Metrics.InspectedBytes)
			require.Equal(t, tt.expected.Metrics.CompletedJobs, actual.Metrics.CompletedJobs)
		})
	}
}

func TestTraceLookupHonorsContentType(t *testing.T) {
	expectedResults := []string{
		"trace1",
		"trace2",
	}
	expectedMetrics := &tempopb.SearchMetrics{InspectedBytes: 100}

	testResp := &tempopb.TraceLookupResponse{
		TraceIDs: expectedResults,
		Metrics: expectedMetrics,
	}

	// JSON response
	c := NewTraceLookup()
	err := c.AddResponse(toHTTPResponse(t, testResp, 200))
	require.NoError(t, err)

	resp, err := c.HTTPFinal()
	require.NoError(t, err)
	require.Equal(t, api.HeaderAcceptJSON, resp.Header.Get(api.HeaderContentType))

	actual := &tempopb.TraceLookupResponse{}
	bodyBytes, _ := io.ReadAll(resp.Body)
	err = jsonpb.UnmarshalString(string(bodyBytes), actual)
	require.NoError(t, err)
	require.Equal(t, expectedResults, actual.TraceIDs)
}

func TestTraceLookupStatusCode(t *testing.T) {
	c := NewTraceLookup()
	require.Equal(t, 200, c.StatusCode())

	// Add a successful response
	err := c.AddResponse(toHTTPResponse(t, &tempopb.TraceLookupResponse{
		TraceIDs: []string{"trace1"},
		Metrics: &tempopb.SearchMetrics{InspectedBytes: 100},
	}, 200))
	require.NoError(t, err)
	require.Equal(t, 200, c.StatusCode())

	// Add an error response
	c = NewTraceLookup()
	err = c.AddResponse(toHTTPResponse(t, &tempopb.TraceLookupResponse{}, 500))
	require.NoError(t, err)
	require.Equal(t, 500, c.StatusCode())
}

func TestTraceLookupMetricsCombining(t *testing.T) {
	c := NewTraceLookup()

	// Add first response
	err := c.AddResponse(toHTTPResponse(t, &tempopb.TraceLookupResponse{
		TraceIDs: []string{"trace1"},
		Metrics: &tempopb.SearchMetrics{InspectedBytes: 100},
	}, 200))
	require.NoError(t, err)

	// Add second response
	err = c.AddResponse(toHTTPResponse(t, &tempopb.TraceLookupResponse{
		TraceIDs: []string{"trace2"},
		Metrics: &tempopb.SearchMetrics{InspectedBytes: 200},
	}, 200))
	require.NoError(t, err)

	httpResp, err := c.HTTPFinal()
	require.NoError(t, err)

	bodyBytes, err := io.ReadAll(httpResp.Body)
	require.NoError(t, err)

	actual := &tempopb.TraceLookupResponse{}
	err = jsonpb.UnmarshalString(string(bodyBytes), actual)
	require.NoError(t, err)

	// Check that metrics were combined correctly
	require.Equal(t, uint64(300), actual.Metrics.InspectedBytes)
	require.Equal(t, uint32(2), actual.Metrics.CompletedJobs)
}