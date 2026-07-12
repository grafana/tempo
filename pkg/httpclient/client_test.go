package httpclient

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/stretchr/testify/assert"
)

type MockRoundTripper func(r *http.Request) *http.Response

func (f MockRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r), nil
}

func TestQueryTrace(t *testing.T) {
	trace := &tempopb.Trace{}
	t.Run("returns a trace when is found", func(t *testing.T) {
		mockTransport := MockRoundTripper(func(req *http.Request) *http.Response {
			assert.Equal(t, "www.tempo.com/api/traces/100", req.URL.Path)
			assert.Equal(t, "application/protobuf", req.Header.Get("Accept"))
			response, _ := proto.Marshal(trace)
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewReader(response)),
			}
		})

		client := New("www.tempo.com", "1000")
		client.WithTransport(mockTransport)
		response, err := client.QueryTrace("100")

		assert.NoError(t, err)
		assert.True(t, proto.Equal(trace, response))
	})

	t.Run("includes the recentDataTarget header", func(t *testing.T) {
		mockTransport := MockRoundTripper(func(req *http.Request) *http.Response {
			assert.Equal(t, liveStoreHeaderValue, req.Header.Get(recentDataTargetHeader))
			response, _ := proto.Marshal(trace)
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewReader(response)),
			}
		})

		client := New("www.tempo.com", "1000")
		client.QueryLiveStores = true
		client.WithTransport(mockTransport)
		response, err := client.QueryTrace("100")

		assert.NoError(t, err)
		assert.True(t, proto.Equal(trace, response))
	})

	t.Run("returns a trace not found error on 404", func(t *testing.T) {
		mockTransport := MockRoundTripper(func(_ *http.Request) *http.Response {
			return &http.Response{
				StatusCode: 404,
				Body:       nil,
			}
		})

		client := New("www.tempo.com", "1000")
		client.WithTransport(mockTransport)
		response, err := client.QueryTrace("notfound")

		assert.Error(t, err)
		assert.Nil(t, response)
	})
}

func TestQueryTraceWithRange(t *testing.T) {
	trace := &tempopb.Trace{}
	t.Run("returns an error if start time is greater than end time", func(t *testing.T) {
		client := New("www.tempo.com", "1000")
		response, err := client.QueryTraceWithRange(context.Background(), "notfound", 3000, 2000)

		assert.Error(t, err)
		assert.Nil(t, response)
	})

	t.Run("returns a trace with range when is found", func(t *testing.T) {
		mockTransport := MockRoundTripper(func(req *http.Request) *http.Response {
			assert.Equal(t, "www.tempo.com/api/traces/100?end=2000&start=1000", fmt.Sprint(req.URL))
			assert.Equal(t, "application/protobuf", req.Header.Get("Accept"))
			response, _ := proto.Marshal(trace)
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewReader(response)),
			}
		})

		client := New("www.tempo.com", "1000")
		client.WithTransport(mockTransport)
		response, err := client.QueryTraceWithRange(context.Background(), "100", 1000, 2000)

		assert.NoError(t, err)
		assert.True(t, proto.Equal(trace, response))
	})

	t.Run("returns a trace with range not found error on 404", func(t *testing.T) {
		mockTransport := MockRoundTripper(func(_ *http.Request) *http.Response {
			return &http.Response{
				StatusCode: 404,
				Body:       nil,
			}
		})

		client := New("www.tempo.com", "1000")
		client.WithTransport(mockTransport)
		response, err := client.QueryTraceWithRange(context.Background(), "notfound", 1000, 2000)

		assert.Error(t, err)
		assert.Nil(t, response)
	})
}

func TestQueryTraceV2WithQueryParams(t *testing.T) {
	resp := &tempopb.TraceByIDResponse{}

	tests := []struct {
		name        string
		global      map[string]string
		params      map[string]string
		expectedURL string
	}{
		{
			// no params must match QueryTraceV2 exactly, with no forced trailing ?.
			name:        "no params yields the same URL as QueryTraceV2",
			params:      nil,
			expectedURL: "www.tempo.com/api/v2/traces/100",
		},
		{
			name:        "encodes the provided query params",
			params:      map[string]string{"keep_hierarchy": "false"},
			expectedURL: "www.tempo.com/api/v2/traces/100?keep_hierarchy=false",
		},
		{
			// client-wide params (SetQueryParam) must not be dropped for this endpoint.
			name:        "merges client-wide params",
			global:      map[string]string{"mode": "recent"},
			params:      map[string]string{"keep_hierarchy": "false"},
			expectedURL: "www.tempo.com/api/v2/traces/100?keep_hierarchy=false&mode=recent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockTransport := MockRoundTripper(func(req *http.Request) *http.Response {
				assert.Equal(t, tt.expectedURL, fmt.Sprint(req.URL))
				body, _ := proto.Marshal(resp)
				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(bytes.NewReader(body)),
				}
			})

			client := New("www.tempo.com", "1000")
			client.WithTransport(mockTransport)
			for k, v := range tt.global {
				client.SetQueryParam(k, v)
			}
			response, err := client.QueryTraceV2WithQueryParams("100", tt.params)

			assert.NoError(t, err)
			assert.True(t, proto.Equal(resp, response))
		})
	}
}
