package frontend

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-kit/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockNextTripperware struct{}

func (s *mockNextTripperware) RoundTrip(_ *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewReader([]byte("next"))),
	}, nil
}

func TestFrontendRoundTripsSearch(t *testing.T) {
	next := &mockNextTripperware{}
	f, err := New(Config{QueryShards: minQueryShards}, next, nil, log.NewNopLogger(), nil)
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "/", nil)

	// search is a blind passthrough. easy!
	res := httptest.NewRecorder()
	f.Search.ServeHTTP(res, req)
	assert.Equal(t, string(res.Body.Bytes()), "next")
}

func TestFrontendBadConfigFails(t *testing.T) {
	f, err := New(Config{QueryShards: minQueryShards - 1}, nil, nil, log.NewNopLogger(), nil)
	assert.Error(t, err)
	assert.Nil(t, f)

	f, err = New(Config{QueryShards: maxQueryShards + 1}, nil, nil, log.NewNopLogger(), nil)
	assert.Error(t, err)
	assert.Nil(t, f)
}
