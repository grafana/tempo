package frontend

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/tempodb/backend"
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
				MostRecentShards:      defaultMostRecentShards,
			},
			SLO: testSLOcfg,
		},
		Metrics: MetricsConfig{
			Sharder: QueryRangeSharderConfig{
				ConcurrentRequests:    defaultConcurrentRequests,
				TargetBytesPerRequest: defaultTargetBytesPerRequest,
				Interval:              time.Second,
			},
			SLO: testSLOcfg,
		},
	}, next, nil, nil, nil, "", fakeHTTPAuthMiddleware, log.NewNopLogger(), nil)
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
				MostRecentShards:      defaultMostRecentShards,
			},
			SLO: testSLOcfg,
		},
	}, nil, nil, nil, nil, "", fakeHTTPAuthMiddleware, log.NewNopLogger(), nil)
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
				MostRecentShards:      defaultMostRecentShards,
			},
			SLO: testSLOcfg,
		},
	}, nil, nil, nil, nil, "", fakeHTTPAuthMiddleware, log.NewNopLogger(), nil)
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
				MostRecentShards:      defaultMostRecentShards,
			},
			SLO: testSLOcfg,
		},
	}, nil, nil, nil, nil, "", fakeHTTPAuthMiddleware, log.NewNopLogger(), nil)
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
				MostRecentShards:      defaultMostRecentShards,
			},
			SLO: testSLOcfg,
		},
	}, nil, nil, nil, nil, "", fakeHTTPAuthMiddleware, log.NewNopLogger(), nil)
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
				MostRecentShards:      defaultMostRecentShards,
				QueryIngestersUntil:   time.Minute,
				QueryBackendAfter:     time.Hour,
			},
			SLO: testSLOcfg,
		},
	}, nil, nil, nil, nil, "", fakeHTTPAuthMiddleware, log.NewNopLogger(), nil)
	assert.EqualError(t, err, "query backend after should be less than or equal to query ingester until")
	assert.Nil(t, f)

	f, err = New(Config{
		TraceByID: TraceByIDConfig{
			QueryShards: maxQueryShards,
		},
		Search: SearchConfig{
			Sharder: SearchSharderConfig{
				ConcurrentRequests:    defaultConcurrentRequests,
				TargetBytesPerRequest: defaultTargetBytesPerRequest,
				MostRecentShards:      defaultMostRecentShards,
			},
			SLO: testSLOcfg,
		},
	}, nil, nil, nil, nil, "", fakeHTTPAuthMiddleware, log.NewNopLogger(), nil)
	assert.EqualError(t, err, "frontend metrics concurrent requests should be greater than 0")
	assert.Nil(t, f)

	f, err = New(Config{
		TraceByID: TraceByIDConfig{
			QueryShards: maxQueryShards,
		},
		Search: SearchConfig{
			Sharder: SearchSharderConfig{
				ConcurrentRequests:    defaultConcurrentRequests,
				TargetBytesPerRequest: defaultTargetBytesPerRequest,
				MostRecentShards:      defaultMostRecentShards,
			},
			SLO: testSLOcfg,
		},
		Metrics: MetricsConfig{
			Sharder: QueryRangeSharderConfig{
				ConcurrentRequests:    defaultConcurrentRequests,
				TargetBytesPerRequest: 0,
			},
			SLO: testSLOcfg,
		},
	}, nil, nil, nil, nil, "", fakeHTTPAuthMiddleware, log.NewNopLogger(), nil)
	assert.EqualError(t, err, "frontend metrics target bytes per request should be greater than 0")
	assert.Nil(t, f)

	f, err = New(Config{
		TraceByID: TraceByIDConfig{
			QueryShards: maxQueryShards,
		},
		Search: SearchConfig{
			Sharder: SearchSharderConfig{
				ConcurrentRequests:    defaultConcurrentRequests,
				TargetBytesPerRequest: defaultTargetBytesPerRequest,
				MostRecentShards:      defaultMostRecentShards,
			},
			SLO: testSLOcfg,
		},
		Metrics: MetricsConfig{
			Sharder: QueryRangeSharderConfig{
				ConcurrentRequests:    defaultConcurrentRequests,
				TargetBytesPerRequest: defaultTargetBytesPerRequest,
				Interval:              0,
			},
			SLO: testSLOcfg,
		},
	}, nil, nil, nil, nil, "", fakeHTTPAuthMiddleware, log.NewNopLogger(), nil)
	assert.EqualError(t, err, "frontend metrics interval should be greater than 0")
	assert.Nil(t, f)

	f, err = New(Config{
		TraceByID: TraceByIDConfig{
			QueryShards: maxQueryShards,
		},
		Search: SearchConfig{
			Sharder: SearchSharderConfig{
				ConcurrentRequests:    defaultConcurrentRequests,
				TargetBytesPerRequest: defaultTargetBytesPerRequest,
				MostRecentShards:      0,
			},
			SLO: testSLOcfg,
		},
		Metrics: MetricsConfig{
			Sharder: QueryRangeSharderConfig{
				ConcurrentRequests:    defaultConcurrentRequests,
				TargetBytesPerRequest: defaultTargetBytesPerRequest,
				Interval:              5 * time.Minute,
			},
			SLO: testSLOcfg,
		},
	}, nil, nil, nil, nil, "", fakeHTTPAuthMiddleware, log.NewNopLogger(), nil)
	assert.EqualError(t, err, "most recent shards must be greater than 0")
	assert.Nil(t, f)
}

func TestBlockMetasForSearch(t *testing.T) {
	now := time.Now()

	createBlockMeta := func(id string, startTime, endTime time.Time, rf uint32) *backend.BlockMeta {
		return &backend.BlockMeta{
			BlockID:           backend.UUID(uuid.MustParse(id)),
			StartTime:         startTime,
			EndTime:           endTime,
			ReplicationFactor: rf,
		}
	}

	rfFilter := func(m *backend.BlockMeta) bool {
		return m.ReplicationFactor > 1
	}

	acceptAllFilter := func(_ *backend.BlockMeta) bool {
		return true
	}

	t.Run("time range overlap logic", func(t *testing.T) {
		searchStart := now.Add(-2 * time.Hour)
		searchEnd := now.Add(-1 * time.Hour)

		blocks := []*backend.BlockMeta{
			createBlockMeta("00000000-0000-0000-0000-000000000001", now.Add(-4*time.Hour), now.Add(-3*time.Hour), 3),
			createBlockMeta("00000000-0000-0000-0000-000000000002", now.Add(-3*time.Hour), now.Add(-90*time.Minute), 3),
			createBlockMeta("00000000-0000-0000-0000-000000000003", now.Add(-110*time.Minute), now.Add(-70*time.Minute), 3),
			createBlockMeta("00000000-0000-0000-0000-000000000004", now.Add(-90*time.Minute), now.Add(-30*time.Minute), 3),
			createBlockMeta("00000000-0000-0000-0000-000000000005", now.Add(-30*time.Minute), now, 3),
			createBlockMeta("00000000-0000-0000-0000-000000000006", now.Add(-3*time.Hour), now.Add(-30*time.Minute), 3),
		}

		result := blockMetasForSearch(blocks, searchStart, searchEnd, acceptAllFilter)

		// Should return blocks 2, 3, 4, 6 (those that overlap with search range)
		expectedIDs := []string{
			"00000000-0000-0000-0000-000000000002",
			"00000000-0000-0000-0000-000000000003",
			"00000000-0000-0000-0000-000000000004",
			"00000000-0000-0000-0000-000000000006",
		}

		require.Len(t, result, 4)

		resultIDs := make([]string, len(result))
		for i, block := range result {
			resultIDs[i] = block.BlockID.String()
		}

		for _, expectedID := range expectedIDs {
			assert.Contains(t, resultIDs, expectedID)
		}
	})

	t.Run("sorting behavior - backwards in time with deterministic ordering", func(t *testing.T) {
		searchStart := now.Add(-2 * time.Hour)
		searchEnd := now.Add(-1 * time.Hour)

		blocks := []*backend.BlockMeta{
			createBlockMeta("00000000-0000-0000-0000-000000000003", now.Add(-2*time.Hour), now.Add(-90*time.Minute), 3),
			createBlockMeta("00000000-0000-0000-0000-000000000001", now.Add(-2*time.Hour), now.Add(-90*time.Minute), 3),
			createBlockMeta("00000000-0000-0000-0000-000000000002", now.Add(-2*time.Hour), now.Add(-90*time.Minute), 3),
			createBlockMeta("00000000-0000-0000-0000-000000000004", now.Add(-2*time.Hour), now.Add(-70*time.Minute), 3),
			createBlockMeta("00000000-0000-0000-0000-000000000005", now.Add(-2*time.Hour), now.Add(-110*time.Minute), 3),
		}

		result := blockMetasForSearch(blocks, searchStart, searchEnd, acceptAllFilter)

		require.Len(t, result, 5)

		assert.Equal(t, "00000000-0000-0000-0000-000000000004", result[0].BlockID.String())
		assert.Equal(t, now.Add(-70*time.Minute), result[0].EndTime)

		assert.Equal(t, "00000000-0000-0000-0000-000000000001", result[1].BlockID.String())
		assert.Equal(t, "00000000-0000-0000-0000-000000000002", result[2].BlockID.String())
		assert.Equal(t, "00000000-0000-0000-0000-000000000003", result[3].BlockID.String())

		assert.Equal(t, "00000000-0000-0000-0000-000000000005", result[4].BlockID.String())
		assert.Equal(t, now.Add(-110*time.Minute), result[4].EndTime)
	})

	t.Run("filter function functionality", func(t *testing.T) {
		searchStart := now.Add(-2 * time.Hour)
		searchEnd := now.Add(-1 * time.Hour)

		blocks := []*backend.BlockMeta{
			createBlockMeta("00000000-0000-0000-0000-000000000001", now.Add(-2*time.Hour), now.Add(-90*time.Minute), 1),
			createBlockMeta("00000000-0000-0000-0000-000000000002", now.Add(-2*time.Hour), now.Add(-90*time.Minute), 3),
			createBlockMeta("00000000-0000-0000-0000-000000000003", now.Add(-2*time.Hour), now.Add(-90*time.Minute), 2),
		}

		result := blockMetasForSearch(blocks, searchStart, searchEnd, rfFilter)
		require.Len(t, result, 2)

		resultIDs := make([]string, len(result))
		for i, block := range result {
			resultIDs[i] = block.BlockID.String()
		}

		assert.Contains(t, resultIDs, "00000000-0000-0000-0000-000000000002")
		assert.Contains(t, resultIDs, "00000000-0000-0000-0000-000000000003")
		assert.NotContains(t, resultIDs, "00000000-0000-0000-0000-000000000001")

		result = blockMetasForSearch(blocks, searchStart, searchEnd, acceptAllFilter)
		require.Len(t, result, 3)
	})

	t.Run("edge cases", func(t *testing.T) {
		searchStart := now.Add(-2 * time.Hour)
		searchEnd := now.Add(-1 * time.Hour)

		t.Run("empty block list", func(t *testing.T) {
			result := blockMetasForSearch([]*backend.BlockMeta{}, searchStart, searchEnd, acceptAllFilter)
			assert.Empty(t, result)
		})

		t.Run("no overlapping blocks", func(t *testing.T) {
			blocks := []*backend.BlockMeta{
				createBlockMeta("00000000-0000-0000-0000-000000000001", now.Add(-4*time.Hour), now.Add(-3*time.Hour), 3),
				createBlockMeta("00000000-0000-0000-0000-000000000002", now.Add(-30*time.Minute), now, 3),
			}

			result := blockMetasForSearch(blocks, searchStart, searchEnd, acceptAllFilter)
			assert.Empty(t, result)
		})

		t.Run("all blocks filtered out", func(t *testing.T) {
			blocks := []*backend.BlockMeta{
				createBlockMeta("00000000-0000-0000-0000-000000000001", now.Add(-2*time.Hour), now.Add(-90*time.Minute), 1),
				createBlockMeta("00000000-0000-0000-0000-000000000002", now.Add(-2*time.Hour), now.Add(-70*time.Minute), 1),
			}

			result := blockMetasForSearch(blocks, searchStart, searchEnd, rfFilter)
			assert.Empty(t, result)
		})

		t.Run("exact time boundary matches", func(t *testing.T) {
			blocks := []*backend.BlockMeta{
				createBlockMeta("00000000-0000-0000-0000-000000000001", now.Add(-3*time.Hour), searchStart, 3),
				createBlockMeta("00000000-0000-0000-0000-000000000002", searchEnd, now, 3),
				createBlockMeta("00000000-0000-0000-0000-000000000003", searchStart, searchEnd, 3),
			}

			result := blockMetasForSearch(blocks, searchStart, searchEnd, acceptAllFilter)
			require.Len(t, result, 3)

			resultIDs := make([]string, len(result))
			for i, block := range result {
				resultIDs[i] = block.BlockID.String()
			}

			assert.Contains(t, resultIDs, "00000000-0000-0000-0000-000000000001")
			assert.Contains(t, resultIDs, "00000000-0000-0000-0000-000000000002")
			assert.Contains(t, resultIDs, "00000000-0000-0000-0000-000000000003")
		})
	})
}
