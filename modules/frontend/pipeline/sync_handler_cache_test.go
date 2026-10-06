package pipeline

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-kit/log"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/grafana/tempo/v3/pkg/api"
	"github.com/grafana/tempo/v3/pkg/cache"
	"github.com/grafana/tempo/v3/pkg/tempopb"
	"github.com/grafana/tempo/v3/pkg/util/test"
	"github.com/stretchr/testify/require"
)

func TestNilProvider(t *testing.T) {
	c := newFrontendCache(nil, cache.RoleFrontendSearch, log.NewNopLogger())
	require.Nil(t, c)
}

func TestCacheCaches(t *testing.T) {
	expected := &tempopb.SearchTagsResponse{
		TagNames: []string{"foo", "bar"},
	}

	// marshal mesage to bytes
	buf := bytes.NewBuffer([]byte{})
	err := (&jsonpb.Marshaler{}).Marshal(buf, expected)
	require.NoError(t, err)

	testKey := "key"
	testData := buf.Bytes()

	p := test.NewMockProvider()
	c := newFrontendCache(p, cache.RoleBloom, log.NewNopLogger())
	require.NotNil(t, c)

	// create response
	c.store(context.Background(), testKey, testData)

	actual := &tempopb.SearchTagsResponse{}
	buffer := c.fetchBytes(context.Background(), testKey)
	err = (&jsonpb.Unmarshaler{AllowUnknownFields: true}).Unmarshal(bytes.NewReader(buffer), actual)

	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

func TestDetermineContentType(t *testing.T) {
	// Create and marshal a real protobuf message
	protoMsg := &tempopb.SearchTagsResponse{
		TagNames: []string{"foo", "bar"},
	}
	protobufContent, err := protoMsg.Marshal()
	require.NoError(t, err)

	// Also create JSON content for comparison
	jsonBuf := bytes.NewBuffer([]byte{})
	err = (&jsonpb.Marshaler{}).Marshal(jsonBuf, protoMsg)
	require.NoError(t, err)
	jsonContent := jsonBuf.Bytes()

	testCases := []struct {
		name         string
		body         []byte
		expectedType string
	}{
		{
			name:         "JSON content",
			body:         jsonContent,
			expectedType: api.HeaderAcceptJSON,
		},
		{
			name:         "Protobuf content",
			body:         protobufContent,
			expectedType: api.HeaderAcceptProtobuf,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			contentType := determineContentType(tc.body)
			require.Equal(t, tc.expectedType, contentType)
		})
	}
}

func TestCachingWareNotFound(t *testing.T) {
	tcs := []struct {
		name          string
		cacheNotFound bool
		status        int
		expectedCalls int
	}{
		{name: "200 is cached", status: http.StatusOK, expectedCalls: 1},
		{name: "404 is not cached by default", status: http.StatusNotFound, expectedCalls: 2},
		{name: "404 is cached when enabled", cacheNotFound: true, status: http.StatusNotFound, expectedCalls: 1},
		{name: "200 is cached when 404 caching is enabled", cacheNotFound: true, status: http.StatusOK, expectedCalls: 1},
		{name: "500 is never cached", cacheNotFound: true, status: http.StatusInternalServerError, expectedCalls: 2},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			next := RoundTripperFunc(func(_ Request) (*http.Response, error) {
				calls++
				return &http.Response{
					StatusCode: tc.status,
					Body:       io.NopCloser(bytes.NewReader([]byte("{}"))),
					Header:     http.Header{},
				}, nil
			})

			ware := NewCachingWare(test.NewMockProvider(), cache.RoleFrontendTraceByID, log.NewNopLogger())
			if tc.cacheNotFound {
				ware = NewCachingWareWithNotFound(test.NewMockProvider(), cache.RoleFrontendTraceByID, log.NewNopLogger())
			}
			rt := ware.Wrap(next)

			for range 2 {
				req := NewHTTPRequest(httptest.NewRequest(http.MethodGet, "/", nil))
				req.SetCacheKey("key")
				resp, err := rt.RoundTrip(req)
				require.NoError(t, err)
				require.Equal(t, tc.status, resp.StatusCode)
			}
			require.Equal(t, tc.expectedCalls, calls)
		})
	}
}
