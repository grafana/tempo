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
	parent := test.MakeSpanPruningSpan(traceID, test.MakeSpanPruningSpanID(1, 0), nil, "parent", 0, 0)
	spans := []*tracev1.Span{parent}
	for i := byte(0); i < 3; i++ {
		spans = append(spans, test.MakeSpanPruningSpan(traceID, test.MakeSpanPruningSpanID(2, i), parent.SpanId, "SELECT", 0, 0,
			test.MakeAttribute("db.operation", "select")))
	}

	traceResponse := &tempopb.TraceByIDResponse{
		Trace:   test.WrapSpansAsTrace(spans...),
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

	// parent + 1 summary replacing the 3 aggregated SELECT spans
	require.Equal(t, 2, test.CountSpans(actualResp.Trace))

	summary, found := test.FindSpanPruningSummary(actualResp.Trace)
	require.True(t, found, "expected a pruned summary span")
	require.Equal(t, int64(3), test.SpanAttrInt(summary, "aggregation.span_count"))
}

// dropAllFilter drops every span and records whether it ran.
type dropAllFilter struct {
	called bool
}

func (f *dropAllFilter) Process(_ *tempopb.Trace) (*tempopb.Trace, error) {
	f.called = true
	return &tempopb.Trace{}, nil
}

// errorFilter is a TraceFilter that always fails, used to assert filter errors surface.
type errorFilter struct{}

func (errorFilter) Process(_ *tempopb.Trace) (*tempopb.Trace, error) {
	return nil, assert.AnError
}

// nilFilter returns a nil trace; the combiner must treat it as empty, not panic.
type nilFilter struct{}

func (nilFilter) Process(_ *tempopb.Trace) (*tempopb.Trace, error) { return nil, nil }

func TestTraceByIDV2AppliesTraceFilter(t *testing.T) {
	traceResponse := &tempopb.TraceByIDResponse{
		Trace:   test.MakeTrace(2, []byte{0x01, 0x02}),
		Metrics: &tempopb.TraceByIDMetrics{},
	}
	resBytes, err := proto.Marshal(traceResponse)
	require.NoError(t, err)

	newResp := func() http.Response {
		return http.Response{
			StatusCode: 200,
			Header:     map[string][]string{"Content-Type": {"application/protobuf"}},
			Body:       io.NopCloser(bytes.NewReader(resBytes)),
		}
	}

	t.Run("HTTPFinal", func(t *testing.T) {
		filter := &dropAllFilter{}
		c := NewTraceByIDV2(100_000, api.HeaderAcceptProtobuf, nil, TraceByIDV2Options{TraceFilter: filter})
		response := newResp()
		require.NoError(t, c.AddResponse(MockResponse{&response}))

		res, err := c.HTTPFinal()
		require.NoError(t, err)
		body, err := io.ReadAll(res.Body)
		require.NoError(t, err)
		actual := &tempopb.TraceByIDResponse{}
		require.NoError(t, proto.Unmarshal(body, actual))
		require.True(t, filter.called, "filter must be invoked")
		require.Empty(t, actual.Trace.ResourceSpans, "filtered trace must be reflected in the response")
		require.Equal(t, tempopb.PartialStatus_PARTIAL, actual.Status, "a filtered subset must be flagged partial")
		require.Contains(t, actual.Message, "only a subset of spans matching the filter", "message must explain the trace was filtered")
	})

	t.Run("GRPCFinal", func(t *testing.T) {
		filter := &dropAllFilter{}
		c := NewTypedTraceByIDV2(100_000, api.HeaderAcceptProtobuf, nil, TraceByIDV2Options{TraceFilter: filter})
		response := newResp()
		require.NoError(t, c.AddResponse(MockResponse{&response}))

		res, err := c.GRPCFinal()
		require.NoError(t, err)
		require.True(t, filter.called, "filter must be invoked")
		require.Empty(t, res.Trace.ResourceSpans, "filtered trace must be reflected in the grpc response")
		require.Equal(t, tempopb.PartialStatus_PARTIAL, res.Status, "a filtered subset must be flagged partial")
		require.Contains(t, res.Message, "only a subset of spans matching the filter", "message must explain the trace was filtered")
	})

	t.Run("nil filtered trace treated as empty, not a panic", func(t *testing.T) {
		c := NewTraceByIDV2(100_000, api.HeaderAcceptProtobuf, nil, TraceByIDV2Options{TraceFilter: nilFilter{}})
		response := newResp()
		require.NoError(t, c.AddResponse(MockResponse{&response}))

		res, err := c.HTTPFinal()
		require.NoError(t, err)
		body, err := io.ReadAll(res.Body)
		require.NoError(t, err)
		actual := &tempopb.TraceByIDResponse{}
		require.NoError(t, proto.Unmarshal(body, actual))
		require.Empty(t, actual.Trace.ResourceSpans)
	})

	t.Run("filter error surfaces", func(t *testing.T) {
		c := NewTraceByIDV2(100_000, api.HeaderAcceptProtobuf, nil, TraceByIDV2Options{TraceFilter: errorFilter{}})
		response := newResp()
		require.NoError(t, c.AddResponse(MockResponse{&response}))

		_, err := c.HTTPFinal()
		require.Error(t, err)
	})
}
