package combiner

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/stretchr/testify/require"
)

func TestTraceByIDShouldQuit(t *testing.T) {
	// new combiner should not quit
	c := NewTraceByID(0)
	should := c.ShouldQuit()
	require.False(t, should)

	// 500 response should quit
	c = NewTraceByID(0)
	err := c.AddResponse(toHTTPResponse(t, &tempopb.SearchResponse{}, 500))
	require.NoError(t, err)
	should = c.ShouldQuit()
	require.True(t, should)

	// 429 response should quit
	c = NewTraceByID(0)
	err = c.AddResponse(toHTTPProtoResponse(t, &tempopb.SearchResponse{}, 429))
	require.NoError(t, err)
	should = c.ShouldQuit()
	require.True(t, should)

	// 404 response should not quit
	c = NewTraceByID(0)
	err = c.AddResponse(toHTTPProtoResponse(t, &tempopb.SearchResponse{}, 404))
	require.NoError(t, err)
	should = c.ShouldQuit()
	require.False(t, should)

	// unparseable body should not quit, but should return an error
	c = NewTraceByID(0)
	err = c.AddResponse(&http.Response{Body: io.NopCloser(strings.NewReader("foo")), StatusCode: 200})
	require.Error(t, err)
	should = c.ShouldQuit()
	require.False(t, should)

	// trace too large, should not quit but should return an error
	c = NewTraceByID(1)
	err = c.AddResponse(toHTTPProtoResponse(t, &tempopb.TraceByIDResponse{
		Trace:   test.MakeTrace(1, nil),
		Metrics: &tempopb.TraceByIDMetrics{},
	}, 200))
	require.Error(t, err)
	should = c.ShouldQuit()
	require.False(t, should)
}

func toHTTPProtoResponse(t *testing.T, pb proto.Message, statusCode int) *http.Response {
	var body []byte

	if pb != nil {
		var err error
		body, err = proto.Marshal(pb)
		require.NoError(t, err)
	}

	return &http.Response{
		Body:       io.NopCloser(bytes.NewReader(body)),
		StatusCode: statusCode,
	}
}
