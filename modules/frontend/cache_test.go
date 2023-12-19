package frontend

import (
	"bytes"
	"context"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/go-kit/log"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/grafana/tempo/pkg/cache"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/stretchr/testify/require"
)

func TestNilProvider(t *testing.T) {
	testKey := "key"

	c := newFrontendCache(nil, cache.RoleBloom, log.NewNopLogger())
	require.NotNil(t, c)

	rec := httptest.NewRecorder()

	bodyBuffer, err := io.ReadAll(rec.Result().Body)
	require.NoError(t, err)

	c.store(context.Background(), testKey, bodyBuffer)
	found := c.fetch(testKey, &tempopb.SearchResponse{})

	require.False(t, found)
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
	found := c.fetch(testKey, actual)

	require.True(t, found)
	require.Equal(t, expected, actual)
}
