package combiner

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
	"github.com/gogo/status"
	spanpruningprocessor "github.com/open-telemetry/opentelemetry-collector-contrib/processor/spanpruningprocessor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/tempopb"
	commonv1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
	tracev1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/util/test"
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
	resBytes, err := proto.Marshal(traceResponse)
	require.NoError(t, err)
	response := http.Response{
		StatusCode: 200,
		Header: map[string][]string{
			"Content-Type": {"application/protobuf"},
		},
		Body: io.NopCloser(bytes.NewReader(resBytes)),
	}
	combiner := NewTraceByIDV2(10, api.HeaderAcceptJSON, nil, TraceByIDV2Options{})
	err = combiner.AddResponse(MockResponse{&response})
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
	resBytes, err := proto.Marshal(traceResponse)
	require.NoError(t, err)
	response := http.Response{
		StatusCode: 200,
		Header: map[string][]string{
			"Content-Type": {"application/protobuf"},
		},
		Body: io.NopCloser(bytes.NewReader(resBytes)),
	}
	combiner := NewTraceByIDV2(10, api.HeaderAcceptJSON, nil, TraceByIDV2Options{})
	err = combiner.AddResponse(MockResponse{&response})
	require.NoError(t, err)

	res, err := combiner.HTTPFinal()
	require.NoError(t, err)

	actualResp := &tempopb.TraceByIDResponse{}
	err = new(jsonpb.Unmarshaler).Unmarshal(res.Body, actualResp)
	require.NoError(t, err)
	assert.Equal(t, actualResp.Status, tempopb.PartialStatus_PARTIAL)
}

func TestTraceByIDV2RedactorHidesTrace(t *testing.T) {
	traceResponse := &tempopb.TraceByIDResponse{
		Trace:   test.MakeTrace(2, []byte{0x01, 0x02}),
		Metrics: &tempopb.TraceByIDMetrics{},
	}

	newMockResponse := func(t *testing.T) MockResponse {
		resBytes, err := proto.Marshal(traceResponse)
		require.NoError(t, err)
		return MockResponse{&http.Response{
			StatusCode: 200,
			Header:     map[string][]string{"Content-Type": {"application/protobuf"}},
			Body:       io.NopCloser(bytes.NewReader(resBytes)),
		}}
	}

	t.Run("HTTPFinal returns 404 with empty body", func(t *testing.T) {
		c := NewTraceByIDV2(100_000, api.HeaderAcceptJSON, hidingRedactor{}, TraceByIDV2Options{})
		err := c.AddResponse(newMockResponse(t))
		require.NoError(t, err)

		resp, err := c.HTTPFinal()
		require.NoError(t, err)
		require.Equal(t, http.StatusNotFound, resp.StatusCode)
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		require.Empty(t, body)
	})

	t.Run("GRPCFinal returns codes.NotFound", func(t *testing.T) {
		c := NewTypedTraceByIDV2(100_000, api.HeaderAcceptJSON, hidingRedactor{}, TraceByIDV2Options{})
		err := c.AddResponse(newMockResponse(t))
		require.NoError(t, err)

		resp, err := c.GRPCFinal()
		require.Error(t, err)
		require.Nil(t, resp)
		st, ok := status.FromError(err)
		require.True(t, ok)
		require.Equal(t, codes.NotFound, st.Code())
		require.Equal(t, "trace hidden by access policy", st.Message())
	})
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
		combiner := NewTraceByIDV2(100_000, api.HeaderAcceptJSON, nil, TraceByIDV2Options{})
		err = combiner.AddResponse(MockResponse{&response})
		require.NoError(t, err)

		res, err := combiner.HTTPFinal()
		require.NoError(t, err)
		assert.Equal(t, api.HeaderAcceptJSON, res.Header.Get(api.HeaderContentType))

		actualResp := &tempopb.TraceByIDResponse{}
		err = new(jsonpb.Unmarshaler).Unmarshal(res.Body, actualResp)
		require.NoError(t, err)
	})
	t.Run("returns a combined trace response as protobuff", func(t *testing.T) {
		combiner := NewTraceByIDV2(100_000, api.HeaderAcceptProtobuf, nil, TraceByIDV2Options{})
		err = combiner.AddResponse(MockResponse{&response})
		require.NoError(t, err)

		res, err := combiner.HTTPFinal()
		require.NoError(t, err)
		require.NotNil(t, res)
	})
}

// TestNewTraceByIDV2WithSpanPruning mimics TestPruneTrace_BasicAggregation from
// pkg/spanpruning to verify that the combiner actually invokes span pruning: 3 identical
// leaf spans below a parent should collapse into a single summary span.
func TestNewTraceByIDV2WithSpanPruning(t *testing.T) {
	traceID := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	parent := &tracev1.Span{
		TraceId: traceID,
		SpanId:  []byte{1, 0, 0, 0, 0, 0, 0, 0},
		Name:    "parent",
	}
	spans := []*tracev1.Span{parent}
	for i := byte(0); i < 3; i++ {
		spans = append(spans, &tracev1.Span{
			TraceId:      traceID,
			SpanId:       []byte{2, i, 0, 0, 0, 0, 0, 0},
			ParentSpanId: parent.SpanId,
			Name:         "SELECT",
			Attributes: []*commonv1.KeyValue{
				{Key: "db.operation", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "select"}}},
			},
		})
	}

	traceResponse := &tempopb.TraceByIDResponse{
		Trace: &tempopb.Trace{
			ResourceSpans: []*tracev1.ResourceSpans{
				{ScopeSpans: []*tracev1.ScopeSpans{{Spans: spans}}},
			},
		},
		Metrics: &tempopb.TraceByIDMetrics{},
	}
	resBytes, err := proto.Marshal(traceResponse)
	require.NoError(t, err)
	response := http.Response{
		StatusCode: 200,
		Header:     map[string][]string{"Content-Type": {"application/protobuf"}},
		Body:       io.NopCloser(bytes.NewReader(resBytes)),
	}

	cfg := spanpruningprocessor.NewFactory().CreateDefaultConfig().(*spanpruningprocessor.Config)
	cfg.MinSpansToAggregate = 2
	cfg.MaxParentDepth = 0

	c := NewTraceByIDV2(100_000, api.HeaderAcceptProtobuf, nil, TraceByIDV2Options{SpanPruningConfig: cfg})
	err = c.AddResponse(MockResponse{&response})
	require.NoError(t, err)

	res, err := c.HTTPFinal()
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, res.StatusCode)

	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	actualResp := &tempopb.TraceByIDResponse{}
	require.NoError(t, proto.Unmarshal(body, actualResp))

	var (
		spanCount int
		summary   *tracev1.Span
	)
	for _, rs := range actualResp.Trace.ResourceSpans {
		for _, ss := range rs.ScopeSpans {
			spanCount += len(ss.Spans)
			for _, s := range ss.Spans {
				for _, kv := range s.Attributes {
					if kv.Key == "aggregation.is_summary" && kv.Value.GetBoolValue() {
						summary = s
					}
				}
			}
		}
	}

	// parent + 1 summary replacing the 3 aggregated SELECT spans
	assert.Equal(t, 2, spanCount)
	require.NotNil(t, summary, "expected a pruned summary span")
	for _, kv := range summary.Attributes {
		if kv.Key == "aggregation.span_count" {
			assert.Equal(t, int64(3), kv.Value.GetIntValue())
		}
	}
}
