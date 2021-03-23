package frontend

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"

	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/pkg/util/test"
)

func TestCreateBlockShards(t *testing.T) {
	tests := []struct {
		name        string
		queryShards int
		expected    [][]byte
	}{
		{
			name:        "single shard",
			queryShards: 1,
			expected: [][]byte{
				{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0},
				{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
			},
		},
		{
			name:        "multiple shards",
			queryShards: 4,
			expected: [][]byte{
				{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0},  // 0
				{0x3f, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0}, // 0x3f = 255/4 * 1
				{0x7e, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0}, // 0x7e = 255/4 * 2
				{0xbd, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0}, // 0xbd = 255/4 * 3
				{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bb := createBlockBoundaries(tt.queryShards)
			assert.Len(t, bb, len(tt.expected))

			for i := 0; i < len(bb); i++ {
				assert.Equal(t, tt.expected[i], bb[i])
			}
		})
	}
}

func TestMergeResponses(t *testing.T) {
	t1 := test.MakeTrace(10, []byte{0x01, 0x02})
	t2 := test.MakeTrace(10, []byte{0x01, 0x03})

	b1, err := proto.Marshal(t1)
	assert.NoError(t, err)
	b2, err := proto.Marshal(t2)
	assert.NoError(t, err)

	combinedTrace, _, err := util.CombineTraces(b1, b2)
	assert.NoError(t, err)

	traceObject := &tempopb.Trace{}
	err = proto.Unmarshal(combinedTrace, traceObject)
	assert.NoError(t, err)

	var combinedTraceJSON bytes.Buffer
	marshaller := &jsonpb.Marshaler{}
	err = marshaller.Marshal(&combinedTraceJSON, traceObject)
	assert.NoError(t, err)

	tests := []struct {
		name              string
		requestResponse   []RequestResponse
		marshallingFormat string
		expected          *http.Response
	}{
		{
			name: "combine status ok responses",
			requestResponse: []RequestResponse{
				{
					Response: &http.Response{
						StatusCode: http.StatusOK,
						Body:       ioutil.NopCloser(bytes.NewReader(b1)),
					},
				},
				{
					Response: &http.Response{
						StatusCode: http.StatusOK,
						Body:       ioutil.NopCloser(bytes.NewReader(b2)),
					},
				},
				{
					Response: &http.Response{
						StatusCode: http.StatusNotFound,
						Body:       ioutil.NopCloser(bytes.NewReader([]byte("foo"))),
					},
				},
			},
			expected: &http.Response{
				StatusCode: http.StatusOK,
				Body:       ioutil.NopCloser(bytes.NewReader(combinedTrace)),
				Header:     http.Header{},
			},
		},
		{
			name: "report 5xx with hit",
			requestResponse: []RequestResponse{
				{
					Response: &http.Response{
						StatusCode: http.StatusOK,
						Body:       ioutil.NopCloser(bytes.NewReader(b1)),
					},
				},
				{
					Response: &http.Response{
						StatusCode: http.StatusInternalServerError,
						Body:       ioutil.NopCloser(bytes.NewReader([]byte("bar"))),
					},
				},
			},
			expected: &http.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       ioutil.NopCloser(bytes.NewReader([]byte("bar"))),
				Header:     http.Header{},
			},
		},
		{
			name: "report 5xx with no hit",
			requestResponse: []RequestResponse{
				{
					Response: &http.Response{
						StatusCode: http.StatusNotFound,
						Body:       ioutil.NopCloser(bytes.NewReader([]byte("foo"))),
					},
				},
				{
					Response: &http.Response{
						StatusCode: http.StatusInternalServerError,
						Body:       ioutil.NopCloser(bytes.NewReader([]byte("bar"))),
					},
				},
			},
			expected: &http.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       ioutil.NopCloser(bytes.NewReader([]byte("bar"))),
				Header:     http.Header{},
			},
		},
		{
			name: "translate 4xx other than 404 to 500",
			requestResponse: []RequestResponse{
				{
					Response: &http.Response{
						StatusCode: http.StatusOK,
						Body:       ioutil.NopCloser(bytes.NewReader(b1)),
					},
				},
				{
					Response: &http.Response{
						StatusCode: http.StatusForbidden,
						Body:       ioutil.NopCloser(bytes.NewReader([]byte("foo"))),
					},
				},
			},
			expected: &http.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       ioutil.NopCloser(bytes.NewReader([]byte("foo"))),
				Header:     http.Header{},
			},
		},
		{
			name: "accept application/json",
			requestResponse: []RequestResponse{
				{
					Response: &http.Response{
						StatusCode: http.StatusOK,
						Body:       ioutil.NopCloser(bytes.NewReader(b1)),
					},
				},
				{
					Response: &http.Response{
						StatusCode: http.StatusOK,
						Body:       ioutil.NopCloser(bytes.NewReader(b2)),
					},
				},
			},
			marshallingFormat: util.JSONTypeHeaderValue,
			expected: &http.Response{
				StatusCode: http.StatusOK,
				Body:       ioutil.NopCloser(bytes.NewReader(combinedTraceJSON.Bytes())),
				Header:     http.Header{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			marshallingFormat := util.ProtobufTypeHeaderValue
			if len(tt.marshallingFormat) > 0 {
				marshallingFormat = tt.marshallingFormat
			}
			merged, err := mergeResponses(context.Background(), marshallingFormat, tt.requestResponse)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, merged)
		})
	}

}
