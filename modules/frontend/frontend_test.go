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
	f, err := New(Config{
		TraceByID: TraceByIDConfig{
			QueryShards: minQueryShards,
			SLO:         testSLOcfg,
		},
		Search: SearchConfig{
			Sharder: SearchSharderConfig{
				ConcurrentRequests:    defaultConcurrentRequests,
				TargetBytesPerRequest: defaultTargetBytesPerRequest,
			},
			SLO: testSLOcfg,
		},
	}, next, nil, nil, log.NewNopLogger(), nil)
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "/", nil)

	// search is a blind passthrough. easy!
	res := httptest.NewRecorder()
	f.Search.ServeHTTP(res, req)
	assert.Equal(t, res.Body.String(), "next")
}

func TestFrontendBadConfigFails(t *testing.T) {
	f, err := New(Config{
		TraceByID: TraceByIDConfig{
			QueryShards: minQueryShards - 1,
		},
		Search: SearchConfig{
			Sharder: SearchSharderConfig{
				ConcurrentRequests:    defaultConcurrentRequests,
				TargetBytesPerRequest: defaultTargetBytesPerRequest,
			},
			SLO: testSLOcfg,
		},
	}, nil, nil, nil, log.NewNopLogger(), nil)
	assert.EqualError(t, err, "frontend query shards should be between 2 and 256 (both inclusive)")
	assert.Nil(t, f)

	f, err = New(Config{
		TraceByID: TraceByIDConfig{
			QueryShards: maxQueryShards + 1,
			SLO:         testSLOcfg,
		},
		Search: SearchConfig{
			Sharder: SearchSharderConfig{
				ConcurrentRequests:    defaultConcurrentRequests,
				TargetBytesPerRequest: defaultTargetBytesPerRequest,
			},
			SLO: testSLOcfg,
		},
	}, nil, nil, nil, log.NewNopLogger(), nil)
	assert.EqualError(t, err, "frontend query shards should be between 2 and 256 (both inclusive)")
	assert.Nil(t, f)

	f, err = New(Config{
		TraceByID: TraceByIDConfig{
			QueryShards: maxQueryShards,
			SLO:         testSLOcfg,
		},
		Search: SearchConfig{
			Sharder: SearchSharderConfig{
				ConcurrentRequests:    0,
				TargetBytesPerRequest: defaultTargetBytesPerRequest,
			},
			SLO: testSLOcfg,
		},
	}, nil, nil, nil, log.NewNopLogger(), nil)
	assert.EqualError(t, err, "frontend search concurrent requests should be greater than 0")
	assert.Nil(t, f)

	f, err = New(Config{
		TraceByID: TraceByIDConfig{
			QueryShards: maxQueryShards,
			SLO:         testSLOcfg,
		},
		Search: SearchConfig{
			Sharder: SearchSharderConfig{
				ConcurrentRequests:    defaultConcurrentRequests,
				TargetBytesPerRequest: 0,
			},
			SLO: testSLOcfg,
		},
	}, nil, nil, nil, log.NewNopLogger(), nil)
	assert.EqualError(t, err, "frontend search target bytes per request should be greater than 0")
	assert.Nil(t, f)

	f, err = New(Config{
		TraceByID: TraceByIDConfig{
			QueryShards: maxQueryShards,
			SLO:         testSLOcfg,
		},
		Search: SearchConfig{
			Sharder: SearchSharderConfig{
				ConcurrentRequests:    defaultConcurrentRequests,
				TargetBytesPerRequest: defaultTargetBytesPerRequest,
				QueryIngestersUntil:   time.Minute,
				QueryBackendAfter:     time.Hour,
			},
			SLO: testSLOcfg,
		},
	}, nil, nil, nil, log.NewNopLogger(), nil)
	assert.EqualError(t, err, "query backend after should be less than or equal to query ingester until")
	assert.Nil(t, f)
}
