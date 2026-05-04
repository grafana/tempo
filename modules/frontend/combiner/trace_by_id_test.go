package combiner

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util/test"
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

	resp, err := c.HTTPFinal()
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

	resp, err = c.HTTPFinal()
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

	resp, err := c.HTTPFinal()
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
