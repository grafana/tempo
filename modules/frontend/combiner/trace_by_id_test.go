package combiner

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
	"github.com/grafana/tempo/v3/pkg/api"
	"github.com/grafana/tempo/v3/pkg/tempopb"
	"github.com/grafana/tempo/v3/pkg/util/test"
	"github.com/stretchr/testify/require"
)

func TestTraceByIDShouldQuit(t *testing.T) {
	// new combiner should not quit
	c := NewTraceByID(0, api.HeaderAcceptJSON, nil)
	should := c.ShouldQuit()
	require.False(t, should)

	// 500 response should quit
	c = NewTraceByID(0, api.HeaderAcceptJSON, nil)
	err := c.AddResponse(toHTTPResponse(t, &tempopb.SearchResponse{}, 500))
	require.NoError(t, err)
	should = c.ShouldQuit()
	require.True(t, should)

	// 429 response should quit
	c = NewTraceByID(0, api.HeaderAcceptJSON, nil)
	err = c.AddResponse(toHTTPProtoResponse(t, &tempopb.SearchResponse{}, 429))
	require.NoError(t, err)
	should = c.ShouldQuit()
	require.True(t, should)

	// 404 response should not quit
	c = NewTraceByID(0, api.HeaderAcceptJSON, nil)
	err = c.AddResponse(toHTTPProtoResponse(t, &tempopb.SearchResponse{}, 404))
	require.NoError(t, err)
	should = c.ShouldQuit()
	require.False(t, should)

	// unparseable body should not quit, but should return an error
	c = NewTraceByID(0, api.HeaderAcceptJSON, nil)
	err = c.AddResponse(&testPipelineResponse{r: &http.Response{Body: io.NopCloser(strings.NewReader("foo")), StatusCode: 200}})
	require.Error(t, err)
	should = c.ShouldQuit()
	require.False(t, should)

	// trace too large, should quit and should not return an error
	c = NewTraceByID(1, api.HeaderAcceptJSON, nil)
	err = c.AddResponse(toHTTPProtoResponse(t, &tempopb.TraceByIDResponse{
		Trace:   test.MakeTrace(1, nil),
		Metrics: &tempopb.TraceByIDMetrics{},
	}, 200))
	require.NoError(t, err)
	should = c.ShouldQuit()
	require.True(t, should)
}

func TestTraceByIDHonorsContentType(t *testing.T) {
	expected := test.MakeTrace(2, nil)

	// json
	c := NewTraceByID(0, api.HeaderAcceptJSON, nil)
	err := c.AddResponse(toHTTPProtoResponse(t, &tempopb.TraceByIDResponse{Trace: expected, Metrics: &tempopb.TraceByIDMetrics{InspectedBytes: 100}}, 200))
	require.NoError(t, err)

	resp, err := c.HTTPFinal(context.Background())
	require.NoError(t, err)

	actual := &tempopb.Trace{}
	bodyBytes, _ := io.ReadAll(resp.Body)
	err = tempopb.UnmarshalFromJSONV1(bodyBytes, actual)
	require.NoError(t, err)
	require.Equal(t, expected, actual)

	// proto
	c = NewTraceByID(0, api.HeaderAcceptProtobuf, nil)
	err = c.AddResponse(toHTTPProtoResponse(t, &tempopb.TraceByIDResponse{Trace: expected, Metrics: &tempopb.TraceByIDMetrics{InspectedBytes: 100}}, 200))
	require.NoError(t, err)

	resp, err = c.HTTPFinal(context.Background())
	require.NoError(t, err)

	actual = &tempopb.Trace{}
	buff, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	err = proto.Unmarshal(buff, actual)
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

// hidingRedactor is a TraceRedactor stub that always returns ErrTraceHidden,
// simulating an access-policy that hides the trace entirely.
type hidingRedactor struct{}

func (h hidingRedactor) RedactTraceAttributes(_ *tempopb.Trace) error { return ErrTraceHidden }

func TestTraceByIDRedactorHidesTrace(t *testing.T) {
	c := NewTraceByID(0, api.HeaderAcceptJSON, hidingRedactor{})

	err := c.AddResponse(toHTTPProtoResponse(t, &tempopb.TraceByIDResponse{
		Trace:   test.MakeTrace(2, nil),
		Metrics: &tempopb.TraceByIDMetrics{},
	}, 200))
	require.NoError(t, err)

	resp, err := c.HTTPFinal(context.Background())
	require.NoError(t, err)
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Empty(t, body)
}

func toHTTPProtoResponse(t *testing.T, pb proto.Message, statusCode int) PipelineResponse {
	var body []byte

	if pb != nil {
		var err error
		body, err = proto.Marshal(pb)
		require.NoError(t, err)
	}

	return &testPipelineResponse{r: &http.Response{
		Body:       io.NopCloser(bytes.NewReader(body)),
		StatusCode: statusCode,
	}}
}

// The JSON response must match jsonpb's encoding of the protobuf response.
func TestTraceByIDHTTPFinalJSONMatchesJsonpb(t *testing.T) {
	response := &tempopb.TraceByIDResponse{
		Trace:   test.MakeTrace(3, []byte{0x01, 0x02}),
		Metrics: &tempopb.TraceByIDMetrics{InspectedBytes: 100},
	}
	final := func(format api.MarshallingFormat) []byte {
		c := NewTraceByID(0, format, nil)
		require.NoError(t, c.AddResponse(toHTTPProtoResponse(t, response, 200)))
		resp, err := c.HTTPFinal(context.Background())
		require.NoError(t, err)
		require.Equal(t, string(format), resp.Header.Get(api.HeaderContentType))
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		return body
	}

	trace := &tempopb.Trace{}
	require.NoError(t, proto.Unmarshal(final(api.MarshallingFormatProtobuf), trace))
	want, err := new(jsonpb.Marshaler).MarshalToString(trace)
	require.NoError(t, err)
	require.Equal(t, strings.Replace(want, `"resourceSpans":`, `"batches":`, 1), string(final(api.MarshallingFormatJSON)))
}

// Jobs that find nothing still report the bytes they read.
func TestTraceByIDCombinesMetricsOfNotFoundResponses(t *testing.T) {
	notFoundWithMetrics := func(t *testing.T, bytes uint64) PipelineResponse {
		return toHTTPProtoResponse(t, &tempopb.TraceByIDResponse{Metrics: &tempopb.TraceByIDMetrics{InspectedBytes: bytes}}, http.StatusNotFound)
	}
	cacheHit := func(r PipelineResponse) PipelineResponse {
		r.HTTPResponse().Header = http.Header{TempoCacheHeader: {TempoCacheHit}}
		return r
	}

	tests := []struct {
		name          string
		responses     []PipelineResponse
		expectedBytes uint64
		expectedCode  int
	}{
		{
			name:          "not found with metrics",
			responses:     []PipelineResponse{notFoundWithMetrics(t, 100)},
			expectedBytes: 100,
			expectedCode:  http.StatusNotFound,
		},
		{
			name:          "multiple not found are summed",
			responses:     []PipelineResponse{notFoundWithMetrics(t, 100), notFoundWithMetrics(t, 23)},
			expectedBytes: 123,
			expectedCode:  http.StatusNotFound,
		},
		{
			name: "not found next to a found trace",
			responses: []PipelineResponse{
				notFoundWithMetrics(t, 100),
				toHTTPProtoResponse(t, &tempopb.TraceByIDResponse{Trace: test.MakeTrace(1, nil), Metrics: &tempopb.TraceByIDMetrics{InspectedBytes: 20}}, http.StatusOK),
			},
			expectedBytes: 120,
			expectedCode:  http.StatusOK,
		},
		{
			name:          "not found cache hit is not counted",
			responses:     []PipelineResponse{cacheHit(notFoundWithMetrics(t, 100))},
			expectedBytes: 0,
			expectedCode:  http.StatusNotFound,
		},
		{
			name:          "not found without body",
			responses:     []PipelineResponse{toHTTPProtoResponse(t, nil, http.StatusNotFound)},
			expectedBytes: 0,
			expectedCode:  http.StatusNotFound,
		},
		{
			name:          "not found with a body that is not a trace by id response",
			responses:     []PipelineResponse{&testPipelineResponse{r: &http.Response{Body: io.NopCloser(strings.NewReader("404 page not found\n")), StatusCode: http.StatusNotFound}}},
			expectedBytes: 0,
			expectedCode:  http.StatusNotFound,
		},
		{
			name:          "not found with nil body",
			responses:     []PipelineResponse{&testPipelineResponse{r: &http.Response{StatusCode: http.StatusNotFound}}},
			expectedBytes: 0,
			expectedCode:  http.StatusNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var reported uint64
			c := NewTypedTraceByID(0, api.HeaderAcceptProtobuf, nil, WithInspectedBytesReporter(func(b uint64) { reported += b }))

			for _, r := range tc.responses {
				require.NoError(t, c.AddResponse(r))
				require.False(t, c.ShouldQuit())
			}

			require.Equal(t, tc.expectedBytes, c.MetricsCombiner.Metrics.InspectedBytes)
			require.Equal(t, tc.expectedBytes, reported)
			require.Equal(t, tc.expectedCode, c.StatusCode())
		})
	}
}

// The bytes of a response whose trace is too large were still read.
func TestTraceByIDCombinesMetricsOfTooLargeResponses(t *testing.T) {
	var reported uint64
	c := NewTypedTraceByID(1, api.HeaderAcceptProtobuf, nil, WithInspectedBytesReporter(func(b uint64) { reported += b }))

	err := c.AddResponse(toHTTPProtoResponse(t, &tempopb.TraceByIDResponse{
		Trace:   test.MakeTrace(1, nil),
		Metrics: &tempopb.TraceByIDMetrics{InspectedBytes: 100},
	}, http.StatusOK))
	require.NoError(t, err)

	require.True(t, c.ShouldQuit())
	require.Equal(t, http.StatusUnprocessableEntity, c.StatusCode())
	require.Equal(t, uint64(100), c.MetricsCombiner.Metrics.InspectedBytes)
	require.Equal(t, uint64(100), reported)
}
