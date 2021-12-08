package frontend

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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
	f, err := New(Config{QueryShards: minQueryShards,
		Search: SearchSharderConfig{
			ConcurrentRequests:    defaultConcurrentRequests,
			TargetBytesPerRequest: defaultTargetBytesPerRequest,
		},
	}, next, nil, log.NewNopLogger(), nil)
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "/", nil)

	// search is a blind passthrough. easy!
	res := httptest.NewRecorder()
	f.Search.ServeHTTP(res, req)
	assert.Equal(t, res.Body.String(), "next")
}

func TestFrontendBadConfigFails(t *testing.T) {
	f, err := New(Config{QueryShards: minQueryShards - 1,
		Search: SearchSharderConfig{
			ConcurrentRequests:    defaultConcurrentRequests,
			TargetBytesPerRequest: defaultTargetBytesPerRequest,
		},
	}, nil, nil, log.NewNopLogger(), nil)
	assert.EqualError(t, err, "frontend query shards should be between 2 and 256 (both inclusive)")
	assert.Nil(t, f)

	f, err = New(Config{QueryShards: maxQueryShards + 1,
		Search: SearchSharderConfig{
			ConcurrentRequests:    defaultConcurrentRequests,
			TargetBytesPerRequest: defaultTargetBytesPerRequest,
		},
	}, nil, nil, log.NewNopLogger(), nil)
	assert.EqualError(t, err, "frontend query shards should be between 2 and 256 (both inclusive)")
	assert.Nil(t, f)

	f, err = New(Config{QueryShards: maxQueryShards,
		Search: SearchSharderConfig{
			ConcurrentRequests:    0,
			TargetBytesPerRequest: defaultTargetBytesPerRequest,
		},
	}, nil, nil, log.NewNopLogger(), nil)
	assert.EqualError(t, err, "frontend search concurrent requests should be greater than 0")
	assert.Nil(t, f)

	f, err = New(Config{QueryShards: maxQueryShards,
		Search: SearchSharderConfig{
			ConcurrentRequests:    defaultConcurrentRequests,
			TargetBytesPerRequest: 0,
		},
	}, nil, nil, log.NewNopLogger(), nil)
	assert.EqualError(t, err, "frontend search target bytes per request should be greater than 0")
	assert.Nil(t, f)

	f, err = New(Config{QueryShards: maxQueryShards,
		Search: SearchSharderConfig{
			ConcurrentRequests:      defaultConcurrentRequests,
			TargetBytesPerRequest:   defaultTargetBytesPerRequest,
			QueryIngestersWithinMin: time.Hour,
			QueryIngestersWithinMax: time.Minute,
		},
	}, nil, nil, log.NewNopLogger(), nil)
	assert.EqualError(t, err, "query ingesters within min should be less than query ingesters within max")
	assert.Nil(t, f)
}
