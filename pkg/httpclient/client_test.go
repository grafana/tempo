package httpclient

import (
	"bytes"
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

func TestQueryTraceWithRance(t *testing.T) {
	trace := &tempopb.Trace{}
	t.Run("returns an error if start time is greater than end time", func(t *testing.T) {
		client := New("www.tempo.com", "1000")
		response, err := client.QueryTraceWithRange("notfound", 3000, 2000)

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
		response, err := client.QueryTraceWithRange("100", 1000, 2000)

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
		response, err := client.QueryTraceWithRange("notfound", 1000, 2000)

		assert.Error(t, err)
		assert.Nil(t, response)
	})
}
