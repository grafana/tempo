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

func TestNewTraceByIdV2ReturnsAPartialTrace(t *testing.T) {
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
	combiner := NewTraceByIDV2(10, api.HeaderAcceptJSON)
	err = combiner.AddResponse(MockResponse{&response})
	require.NoError(t, err)

	res, err := combiner.HTTPFinal()
	require.NoError(t, err)

	actualResp := &tempopb.TraceByIDResponse{}
	err = new(jsonpb.Unmarshaler).Unmarshal(res.Body, actualResp)
	require.NoError(t, err)
	assert.Equal(t, actualResp.Status, tempopb.TraceByIDResponse_PARTIAL)
}

func TestNewTraceByIdV2ReturnsAPartialTraceOnPartialTraceReturnedByQuerier(t *testing.T) {
	traceResponse := &tempopb.TraceByIDResponse{
		Trace:   test.MakeTrace(2, []byte{0x01, 0x02}),
		Status:  tempopb.TraceByIDResponse_PARTIAL,
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
	combiner := NewTraceByIDV2(10, api.HeaderAcceptJSON)
	err = combiner.AddResponse(MockResponse{&response})
	require.NoError(t, err)

	res, err := combiner.HTTPFinal()
	require.NoError(t, err)

	actualResp := &tempopb.TraceByIDResponse{}
	err = new(jsonpb.Unmarshaler).Unmarshal(res.Body, actualResp)
	require.NoError(t, err)
	assert.Equal(t, actualResp.Status, tempopb.TraceByIDResponse_PARTIAL)
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
