package frontend

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testSLOcfg = SLOConfig{
	ThroughputBytesSLO: 0,
	DurationSLO:        0,
}

func TestFrontendTagSearchRequiresOrgID(t *testing.T) {
	next := &mockRoundTripper{}
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
		Metrics: MetricsConfig{
			Sharder: QueryRangeSharderConfig{
				ConcurrentRequests:    defaultConcurrentRequests,
				TargetBytesPerRequest: defaultTargetBytesPerRequest,
				Interval:              time.Second,
			},
		},
	}, next, nil, nil, nil, "", log.NewNopLogger(), nil)
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "/", nil)

	// search will fail with `no org id` error
	resSearch := httptest.NewRecorder()
	f.SearchTagsValuesHandler.ServeHTTP(resSearch, req)
	assert.Equal(t, resSearch.Body.String(), "no org id")

	resSearch1 := httptest.NewRecorder()
	f.SearchTagsValuesV2Handler.ServeHTTP(resSearch1, req)
	assert.Equal(t, resSearch1.Body.String(), "no org id")

	resSearch2 := httptest.NewRecorder()
	f.SearchTagsV2Handler.ServeHTTP(resSearch2, req)
	assert.Equal(t, resSearch2.Body.String(), "no org id")

	resSearch3 := httptest.NewRecorder()
	f.SearchTagsHandler.ServeHTTP(resSearch3, req)
	assert.Equal(t, resSearch3.Body.String(), "no org id")
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
	}, nil, nil, nil, nil, "", log.NewNopLogger(), nil)
	assert.EqualError(t, err, "frontend query shards should be between 2 and 100000 (both inclusive)")

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
	}, nil, nil, nil, nil, "", log.NewNopLogger(), nil)
	assert.EqualError(t, err, "frontend query shards should be between 2 and 100000 (both inclusive)")
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
	}, nil, nil, nil, nil, "", log.NewNopLogger(), nil)
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
	}, nil, nil, nil, nil, "", log.NewNopLogger(), nil)
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
	}, nil, nil, nil, nil, "", log.NewNopLogger(), nil)
	assert.EqualError(t, err, "query backend after should be less than or equal to query ingester until")
	assert.Nil(t, f)
}
