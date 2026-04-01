package combiner

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/pkg/tempopb"
	commonv1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
	resourcev1 "github.com/grafana/tempo/pkg/tempopb/resource/v1"
	tracev1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
)

func TestRenderPunchCard(t *testing.T) {
	card := renderPunchCard("HELLO WORLD")
	lines := strings.Split(card, "\n")

	// A punch card has: 1 top border + 1 text line + 12 rows + 1 bottom border = 15 lines
	assert.Equal(t, 15, len(lines))

	// Top border
	assert.Contains(t, lines[0], "____")

	// Text line should contain our text
	assert.Contains(t, lines[1], "HELLO WORLD")

	// Bottom border
	assert.Contains(t, lines[14], "____")

	// Each punch row should start with "| " and end with " |"
	for i := 2; i <= 13; i++ {
		assert.True(t, strings.HasPrefix(lines[i], "|"), "row %d should start with |", i)
		assert.True(t, strings.HasSuffix(lines[i], " |"), "row %d should end with |", i)
	}
}

func TestRenderPunchCardEncoding(t *testing.T) {
	// "A" should punch rows 12 and 1
	card := renderPunchCard("A")
	lines := strings.Split(card, "\n")

	// Row 12 is display row 0 (lines[2]), row 1 is display row 3 (lines[5])
	// Position 0 in the 80-char text
	assert.Equal(t, byte('O'), lines[2][5], "row 12 should be punched for 'A'") // "| 12 O..."
	assert.Equal(t, byte('O'), lines[5][5], "row 1 should be punched for 'A'")  // "| 1 O..."
	assert.Equal(t, byte(' '), lines[3][5], "row 11 should not be punched for 'A'")
}

func TestTraceToPunchCards(t *testing.T) {
	trace := &tempopb.Trace{
		ResourceSpans: []*tracev1.ResourceSpans{
			{
				Resource: &resourcev1.Resource{
					Attributes: []*commonv1.KeyValue{
						{Key: "service.name", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "my-svc"}}},
					},
				},
				ScopeSpans: []*tracev1.ScopeSpans{
					{
						Spans: []*tracev1.Span{
							{
								TraceId:           []byte{0xab, 0xcd, 0xef, 0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef, 0x01, 0x23, 0x45, 0x67, 0x89},
								SpanId:            []byte{0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0},
								Name:              "GET /api/traces",
								StartTimeUnixNano: 1000000000,
								EndTimeUnixNano:   1005000000, // 5ms
								Status:            &tracev1.Status{Code: tracev1.Status_STATUS_CODE_OK},
							},
						},
					},
				},
			},
		},
	}

	result, err := traceToPunchCards(trace)
	require.NoError(t, err)

	// Should contain header card with trace ID
	assert.Contains(t, result, "TRACE ABCDEF")

	// Should contain span info
	assert.Contains(t, result, "MY-SVC")
	assert.Contains(t, result, "GET /API/TRACES")

	// Should contain footer
	assert.Contains(t, result, "END OF TRACE")
	assert.Contains(t, result, "GRAFANA TEMPO")
}

func TestEmptyTrace(t *testing.T) {
	result, err := traceToPunchCards(&tempopb.Trace{})
	require.NoError(t, err)
	assert.Contains(t, result, "EMPTY TRACE")
}

func TestNilTrace(t *testing.T) {
	result, err := traceToPunchCards(nil)
	require.NoError(t, err)
	assert.Contains(t, result, "EMPTY TRACE")
}

func TestPunchCardMarshalerUnsupportedType(t *testing.T) {
	m := &punchCardMarshaler{}
	_, err := m.marshalToString(&tempopb.SearchResponse{})
	assert.Error(t, err)
}

func TestPunchCardMarshalerTraceByIDResponse(t *testing.T) {
	m := &punchCardMarshaler{}
	resp := &tempopb.TraceByIDResponse{
		Trace: &tempopb.Trace{
			ResourceSpans: []*tracev1.ResourceSpans{
				{
					Resource: &resourcev1.Resource{
						Attributes: []*commonv1.KeyValue{
							{Key: "service.name", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "test"}}},
						},
					},
					ScopeSpans: []*tracev1.ScopeSpans{
						{
							Spans: []*tracev1.Span{
								{
									TraceId:           make([]byte, 16),
									SpanId:            make([]byte, 8),
									Name:              "test-span",
									StartTimeUnixNano: 0,
									EndTimeUnixNano:   1000000, // 1ms
								},
							},
						},
					},
				},
			},
		},
	}

	result, err := m.marshalToString(resp)
	require.NoError(t, err)
	assert.Contains(t, result, "TRACE")
	assert.Contains(t, result, "TEST")
}

func TestRowIndexToCardRow(t *testing.T) {
	assert.Equal(t, 12, rowIndexToCardRow(0))
	assert.Equal(t, 11, rowIndexToCardRow(1))
	assert.Equal(t, 0, rowIndexToCardRow(2))
	assert.Equal(t, 1, rowIndexToCardRow(3))
	assert.Equal(t, 9, rowIndexToCardRow(11))
}
