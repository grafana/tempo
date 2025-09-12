package combiner

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type MockResponse struct {
	resp *http.Response
}

func (m MockResponse) HTTPResponse() *http.Response {
	return m.resp
}

func (m MockResponse) RequestData() any {
	return nil
}

func (m MockResponse) IsMetadata() bool {
	return false
}

func TestNewTraceByIdV2ReturnsAPartialTrace(t *testing.T) {
	traceResponse := &tempopb.TraceByIDResponse{
		Trace:   test.MakeTrace(2, []byte{0x01, 0x02}),
		Metrics: &tempopb.TraceByIDMetrics{},
	}

	combiner := NewTraceByIDV2(10, api.HeaderAcceptJSON)
	err := combiner.AddResponse(MockResponse{getResponse(t, traceResponse)})
	require.NoError(t, err)

	res, err := combiner.HTTPFinal()
	require.NoError(t, err)

	actualResp := &tempopb.TraceByIDResponse{}
	err = new(jsonpb.Unmarshaler).Unmarshal(res.Body, actualResp)
	require.NoError(t, err)
	assert.Equal(t, actualResp.Status, tempopb.PartialStatus_PARTIAL)
}

func TestNewTraceByIdV2ReturnsAPartialTraceOnPartialTraceReturnedByQuerier(t *testing.T) {
	traceResponse := &tempopb.TraceByIDResponse{
		Trace:   test.MakeTrace(2, []byte{0x01, 0x02}),
		Status:  tempopb.PartialStatus_PARTIAL,
		Metrics: &tempopb.TraceByIDMetrics{},
	}

	combiner := NewTraceByIDV2(10, api.HeaderAcceptJSON)
	err := combiner.AddResponse(MockResponse{getResponse(t, traceResponse)})
	require.NoError(t, err)

	res, err := combiner.HTTPFinal()
	require.NoError(t, err)

	actualResp := &tempopb.TraceByIDResponse{}
	err = new(jsonpb.Unmarshaler).Unmarshal(res.Body, actualResp)
	require.NoError(t, err)
	assert.Equal(t, actualResp.Status, tempopb.PartialStatus_PARTIAL)
}

func TestNewTraceByIdV2ShouldQuitOnPartialTraceReached(t *testing.T) {
	completeResponse := &tempopb.TraceByIDResponse{
		Trace:   test.MakeTrace(2, []byte{0x01, 0x02}),
		Status:  tempopb.PartialStatus_COMPLETE,
		Metrics: &tempopb.TraceByIDMetrics{},
	}

	partialResponse := &tempopb.TraceByIDResponse{
		Trace:   test.MakeTrace(2, []byte{0x01, 0x02}),
		Status:  tempopb.PartialStatus_PARTIAL,
		Metrics: &tempopb.TraceByIDMetrics{},
	}

	combiner := NewTraceByIDV2(completeResponse.Size()*2, api.HeaderAcceptJSON)

	// This checks that the combiner never quits on startup
	assert.False(t, combiner.ShouldQuit())

	// First complete trace should not make it exit
	err := combiner.AddResponse(MockResponse{getResponse(t, completeResponse)})
	require.NoError(t, err)
	assert.False(t, combiner.ShouldQuit())

	// Second trace is flagged as partial, it should bail
	err = combiner.AddResponse(MockResponse{getResponse(t, partialResponse)})
	require.NoError(t, err)
	assert.True(t, combiner.ShouldQuit())

	res, err := combiner.HTTPFinal()
	require.NoError(t, err)

	actualResp := &tempopb.TraceByIDResponse{}
	err = new(jsonpb.Unmarshaler).Unmarshal(res.Body, actualResp)
	require.NoError(t, err)
	assert.Equal(t, actualResp.Status, tempopb.PartialStatus_PARTIAL)
}

func getResponse(t *testing.T, res *tempopb.TraceByIDResponse) *http.Response {
	resBytes, err := proto.Marshal(res)
	require.NoError(t, err)
	return &http.Response{
		StatusCode: 200,
		Header: map[string][]string{
			"Content-Type": {"application/protobuf"},
		},
		Body: io.NopCloser(bytes.NewReader(resBytes)),
	}
}

func TestNewTraceByIDV2(t *testing.T) {
	traceResponse := &tempopb.TraceByIDResponse{
		Trace:   test.MakeTrace(2, []byte{0x01, 0x02}),
		Metrics: &tempopb.TraceByIDMetrics{},
	}
	resBytes, err := proto.Marshal(traceResponse)
	require.NoError(t, err)
	response := http.Response{
		StatusCode: 200,
		Header: map[string][]string{
			"Content-Type": {"application/protobuf"},
		},
		Body: io.NopCloser(bytes.NewReader(resBytes)),
	}

	t.Run("returns a combined trace response as JSON", func(t *testing.T) {
		combiner := NewTraceByIDV2(100_000, api.HeaderAcceptJSON)
		err = combiner.AddResponse(MockResponse{&response})
		require.NoError(t, err)

		res, err := combiner.HTTPFinal()
		require.NoError(t, err)
		assert.Equal(t, "application/json", res.Header.Get("Content-Type"))

		actualResp := &tempopb.TraceByIDResponse{}
		err = new(jsonpb.Unmarshaler).Unmarshal(res.Body, actualResp)
		require.NoError(t, err)
	})
	t.Run("returns a combined trace response as protobuff", func(t *testing.T) {
		combiner := NewTraceByIDV2(100_000, api.HeaderAcceptProtobuf)
		err = combiner.AddResponse(MockResponse{&response})
		require.NoError(t, err)

		res, err := combiner.HTTPFinal()
		require.NoError(t, err)
		require.NotNil(t, res)
	})
}
