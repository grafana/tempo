package model

import (
	"testing"
	"time"

	"github.com/grafana/tempo/pkg/tempopb"
	v1common "github.com/grafana/tempo/pkg/tempopb/common/v1"
	v1resource "github.com/grafana/tempo/pkg/tempopb/resource/v1"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMatches(t *testing.T) {
	testTrace := &tempopb.Trace{
		Batches: []*v1.ResourceSpans{
			{
				Resource: &v1resource.Resource{
					Attributes: []*v1common.KeyValue{
						{
							Key:   "service.name",
							Value: &v1common.AnyValue{Value: &v1common.AnyValue_StringValue{StringValue: "svc"}},
						},
					},
				},
				InstrumentationLibrarySpans: []*v1.InstrumentationLibrarySpans{
					{
						Spans: []*v1.Span{
							{
								Name:              "test",
								StartTimeUnixNano: uint64(10 * time.Second),
								EndTimeUnixNano:   uint64(20 * time.Second),
								Attributes: []*v1common.KeyValue{
									{
										Key:   "foo",
										Value: &v1common.AnyValue{Value: &v1common.AnyValue_StringValue{StringValue: "barricus"}},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	testMetadata := &tempopb.TraceSearchMetadata{
		TraceID:           "1",
		RootServiceName:   "svc",
		RootTraceName:     "test",
		StartTimeUnixNano: uint64(10 * time.Second),
		DurationMs:        10000,
	}

	tests := []struct {
		name     string
		trace    *tempopb.Trace
		req      *tempopb.SearchRequest
		expected *tempopb.TraceSearchMetadata
	}{
		{
			name:  "range before doesn't match",
			trace: testTrace,
			req: &tempopb.SearchRequest{
				Start: 0,
				End:   5,
			},
			expected: nil,
		},
		{
			name:  "range after doesn't match",
			trace: testTrace,
			req: &tempopb.SearchRequest{
				Start: 25,
				End:   30,
			},
			expected: nil,
		},
		{
			name:  "encompassing range matches",
			trace: testTrace,
			req: &tempopb.SearchRequest{
				Start: 5,
				End:   30,
			},
			expected: testMetadata,
		},
		{
			name:  "encompassed range matches",
			trace: testTrace,
			req: &tempopb.SearchRequest{
				Start: 12,
				End:   15,
			},
			expected: testMetadata,
		},
		{
			name:  "overlap start matches",
			trace: testTrace,
			req: &tempopb.SearchRequest{
				Start: 8,
				End:   15,
			},
			expected: testMetadata,
		},
		{
			name:  "overlap end matches",
			trace: testTrace,
			req: &tempopb.SearchRequest{
				Start: 12,
				End:   25,
			},
			expected: testMetadata,
		},
		{
			name:  "max duration excludes",
			trace: testTrace,
			req: &tempopb.SearchRequest{
				Start:         12,
				End:           15,
				MaxDurationMs: 1,
			},
			expected: nil,
		},
		{
			name:  "max duration includes",
			trace: testTrace,
			req: &tempopb.SearchRequest{
				Start:         12,
				End:           15,
				MaxDurationMs: 10000,
			},
			expected: testMetadata,
		},
		{
			name:  "min duration excludes",
			trace: testTrace,
			req: &tempopb.SearchRequest{
				Start:         12,
				End:           15,
				MaxDurationMs: 1,
				MinDurationMs: 10000,
			},
			expected: nil,
		},
		{
			name:  "min duration includes",
			trace: testTrace,
			req: &tempopb.SearchRequest{
				Start:         12,
				End:           15,
				MaxDurationMs: 10000,
				MinDurationMs: 5000,
			},
			expected: testMetadata,
		},
		{
			name:  "tag excludes",
			trace: testTrace,
			req: &tempopb.SearchRequest{
				Start:         12,
				End:           15,
				MaxDurationMs: 1,
				MinDurationMs: 10000,
				Tags:          map[string]string{"foo": "baz"},
			},
			expected: nil,
		},
		{
			name:  "tag includes",
			trace: testTrace,
			req: &tempopb.SearchRequest{
				Start:         12,
				End:           15,
				MaxDurationMs: 10000,
				MinDurationMs: 5000,
				Tags:          map[string]string{"foo": "bar"},
			},
			expected: testMetadata,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			obj, err := marshal(tc.trace, CurrentEncoding)
			require.NoError(t, err)

			actual, err := Matches([]byte{0x01}, obj, CurrentEncoding, tc.req)
			require.NoError(t, err)

			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestMatchesFails(t *testing.T) {
	_, err := Matches([]byte{0x01}, []byte{0x02, 0x03}, "blerg", nil)
	assert.Error(t, err)

	_, err = Matches([]byte{0x01}, []byte{0x02, 0x03}, CurrentEncoding, nil)
	assert.Error(t, err)
}
