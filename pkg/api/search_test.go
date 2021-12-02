package api

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsBackendSearch(t *testing.T) {
	assert.False(t, IsBackendSearch(httptest.NewRequest("GET", "/api/search", nil)))
	assert.False(t, IsBackendSearch(httptest.NewRequest("GET", "/api/search/?start=1", nil)))
	assert.False(t, IsBackendSearch(httptest.NewRequest("GET", "/api/search/?start=1&end=2", nil)))
	assert.False(t, IsBackendSearch(httptest.NewRequest("GET", "/api/search/stuff/?start=1&end=2&tags=test", nil)))
	assert.False(t, IsBackendSearch(httptest.NewRequest("GET", "/stuff/?start=1&end=2&tags=test", nil)))

	assert.True(t, IsBackendSearch(httptest.NewRequest("GET", "/api/search?start=1&end=2&tags=test", nil)))
	assert.True(t, IsBackendSearch(httptest.NewRequest("GET", "/api/search/?start=1&end=2&tags=test", nil)))
}
