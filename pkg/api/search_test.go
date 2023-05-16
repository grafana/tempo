package api

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsBackendSearch(t *testing.T) {
	assert.False(t, IsBackendSearch(httptest.NewRequest("GET", "/api/search", nil)))
	assert.False(t, IsBackendSearch(httptest.NewRequest("GET", "/api/search/?start=1", nil)))
	assert.False(t, IsBackendSearch(httptest.NewRequest("GET", "/api/search/?end=1", nil)))
	assert.False(t, IsBackendSearch(httptest.NewRequest("GET", "/api/search?tags=vulture-1%3DuxyWcCSQHOuRvM", nil)))
	assert.False(t, IsBackendSearch(httptest.NewRequest("GET", "/api/search/tag/vulture-2/values", nil)))

	assert.True(t, IsBackendSearch(httptest.NewRequest("GET", "/api/search/?start=1&end=2", nil)))
	assert.True(t, IsBackendSearch(httptest.NewRequest("GET", "/api/search?start=1&end=2&tags=test", nil)))
	assert.True(t, IsBackendSearch(httptest.NewRequest("GET", "/api/search/?start=1&end=2&tags=test", nil)))
	assert.True(t, IsBackendSearch(httptest.NewRequest("GET", "/querier/api/search?start=1&end=2&tags=test", nil)))
	assert.True(t, IsBackendSearch(httptest.NewRequest("GET", "/querier/api/search/?start=1&end=2&tags=test", nil)))
}

func TestIsSearchBlock(t *testing.T) {
	assert.False(t, IsSearchBlock(httptest.NewRequest("GET", "/api/search", nil)))
	assert.False(t, IsSearchBlock(httptest.NewRequest("GET", "/api/search/?start=1", nil)))

	assert.True(t, IsSearchBlock(httptest.NewRequest("GET", "/api/search?blockID=blerg", nil)))
	assert.True(t, IsSearchBlock(httptest.NewRequest("GET", "/api/search/?blockID=blerg", nil)))
	assert.True(t, IsSearchBlock(httptest.NewRequest("GET", "/querier/api/search?blockID=blerg", nil)))
	assert.True(t, IsSearchBlock(httptest.NewRequest("GET", "/querier/api/search/?blockID=blerg", nil)))
}

func TestSetStartAndEnd(t *testing.T) {
	reqA := httptest.NewRequest("GET", "/search?start=10&end=20", nil)
	reqABefore := reqA.URL.Query()
	assert.Equal(t, "10", reqA.URL.Query().Get(urlParamStart))
	assert.Equal(t, "20", reqA.URL.Query().Get(urlParamEnd))
	// this should overwrite the start and end
	SetStartAndEnd(reqA, 100, 200)
	assert.NotEqual(t, reqABefore, reqA.URL.Query())
	assert.Equal(t, "100", reqA.URL.Query().Get(urlParamStart))
	assert.Equal(t, "200", reqA.URL.Query().Get(urlParamEnd))

	reqB := httptest.NewRequest("GET", "/search?tags=vulture-1%3DuxyWcCSQHOuRvM", nil)
	reqBBefore := reqB.URL.Query()
	// start and end is missing in this query
	assert.Equal(t, "", reqB.URL.Query().Get(urlParamStart))
	assert.Equal(t, "", reqB.URL.Query().Get(urlParamEnd))
	// this should add missing start and end params
	SetStartAndEnd(reqB, 100, 200)
	assert.NotEqual(t, reqBBefore, reqB.URL.Query())
	assert.Equal(t, "100", reqB.URL.Query().Get(urlParamStart))
	assert.Equal(t, "200", reqB.URL.Query().Get(urlParamEnd))

	reqC := httptest.NewRequest("GET", "/search", nil)
	reqCBefore := reqC.URL.Query()
	// start and end is missing in this query
	assert.Equal(t, "", reqC.URL.Query().Get(urlParamStart))
	assert.Equal(t, "", reqC.URL.Query().Get(urlParamEnd))
	// this should add missing start and end params
	SetStartAndEnd(reqC, 100, 200)
	assert.NotEqual(t, reqCBefore, reqC.URL.Query())
	assert.Equal(t, "100", reqC.URL.Query().Get(urlParamStart))
	assert.Equal(t, "200", reqC.URL.Query().Get(urlParamEnd))
}
