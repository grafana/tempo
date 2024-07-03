package main

import (
	"bytes"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/go-test/deep"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/pkg/tempopb"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/util"
)

func TestHasMissingSpans(t *testing.T) {
	cases := []struct {
		trace    *tempopb.Trace
		expected bool
	}{
		{
			&tempopb.Trace{
				Batches: []*v1.ResourceSpans{
					{
						ScopeSpans: []*v1.ScopeSpans{
							{
								Spans: []*v1.Span{
									{
										ParentSpanId: []byte("01234"),
									},
								},
							},
						},
					},
				},
			},
			true,
		},
		{
			&tempopb.Trace{
				Batches: []*v1.ResourceSpans{
					{
						ScopeSpans: []*v1.ScopeSpans{
							{
								Spans: []*v1.Span{
									{
										SpanId: []byte("01234"),
									},
									{
										ParentSpanId: []byte("01234"),
									},
								},
							},
						},
					},
				},
			},
			false,
		},
	}

	for _, tc := range cases {
		require.Equal(t, tc.expected, hasMissingSpans(tc.trace))
	}
}

func TestResponseFixture(t *testing.T) {
	f, err := os.Open("testdata/trace.json")
	require.NoError(t, err)
	defer f.Close()

	expected := &tempopb.Trace{}
	err = jsonpb.Unmarshal(f, expected)
	require.NoError(t, err)

	seed := time.Unix(1636729665, 0)
	info := util.NewTraceInfo(seed, "")

	generatedTrace, err := info.ConstructTraceFromEpoch()
	require.NoError(t, err)

	// print the generated trace
	var jsonTrace bytes.Buffer
	marshaller := &jsonpb.Marshaler{}
	err = marshaller.Marshal(&jsonTrace, generatedTrace)
	require.NoError(t, err)
	fmt.Println(jsonTrace.String())

	assert.True(t, equalTraces(expected, generatedTrace))

	if diff := deep.Equal(expected, generatedTrace); diff != nil {
		for _, d := range diff {
			t.Error(d)
		}
	}
}

func TestEqualTraces(t *testing.T) {
	seed := time.Now()
	info1 := util.NewTraceInfo(seed, "")
	info2 := util.NewTraceInfo(seed, "")

	a, err := info1.ConstructTraceFromEpoch()
	require.NoError(t, err)
	b, err := info2.ConstructTraceFromEpoch()
	require.NoError(t, err)

	require.True(t, equalTraces(a, b))
}

func TestGetGrpcEndpoint(t *testing.T) {
	_, err := getGRPCEndpoint("http://%gh&%ij")
	require.Error(t, err)

	got, err := getGRPCEndpoint("http://localhost:4000")
	require.NoError(t, err)
	assert.Equal(t, "localhost:4000", got, "Address endpoint should keep the given port")

	got, err = getGRPCEndpoint("http://localhost")
	require.NoError(t, err)
	assert.Equal(t, "localhost:14250", got, "Address without a port should be defaulted to 14250")
}
