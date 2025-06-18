package frontend

import (
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/modules/frontend/pipeline"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/tempodb/backend"
)

func TestBuildBackendRequestsExemplarsOneBlock(t *testing.T) {
	// Create the test sharder with exemplars enabled
	sharder := &queryRangeSharder{
		logger: log.NewNopLogger(),
		cfg: QueryRangeSharderConfig{
			MaxExemplars: 100,
		},
	}
	tenantID := "test-tenant"
	targetBytesPerRequest := 1000

	testCases := []struct {
		name              string
		totalRecords      uint32
		blockSize         uint64
		exemplars         uint32
		expectedBatches   int
		expectedExemplars int
	}{
		{
			name:              "basic",
			totalRecords:      100,
			blockSize:         uint64(targetBytesPerRequest),
			exemplars:         5,
			expectedExemplars: 6, // 5 * 1.2
			expectedBatches:   1,
		},
		{
			name:              "two batches",
			totalRecords:      100,
			blockSize:         uint64(2 * targetBytesPerRequest),
			exemplars:         5,
			expectedExemplars: 6, // 5 * 1.2
			expectedBatches:   2,
		},
		{
			name:              "high record count",
			totalRecords:      10000,
			blockSize:         50000,
			exemplars:         10,
			expectedExemplars: 50, // 1 per each batch
			expectedBatches:   50,
		},
		{
			name:              "totalRecords == blockSize == targetBytesPerRequest",
			totalRecords:      uint32(targetBytesPerRequest),
			blockSize:         uint64(targetBytesPerRequest),
			exemplars:         10,
			expectedExemplars: 12, // 10 * 1.2
			expectedBatches:   1,
		},
		{
			name:              "large block size",
			totalRecords:      500,
			blockSize:         50000,
			exemplars:         20,
			expectedExemplars: 50, // 1 per each batch
			expectedBatches:   50,
		},
		{
			name:              "small block",
			totalRecords:      10,
			blockSize:         100,
			exemplars:         1,
			expectedExemplars: 2, // 1 * 1.2 -> rounded up to 2
			expectedBatches:   1,
		},
		{
			name:              "block with single record",
			totalRecords:      1,
			blockSize:         uint64(2 * targetBytesPerRequest),
			exemplars:         1,
			expectedExemplars: 2, // 1 * 1.2 -> rounded up to 2
			expectedBatches:   1,
		},
		{
			name:              "block with single record",
			totalRecords:      1,
			blockSize:         uint64(1.5 * float64(targetBytesPerRequest)),
			exemplars:         1,
			expectedExemplars: 2, // 1 * 1.2 -> rounded up to 2
			expectedBatches:   1,
		},
		{
			name:              "block with 2 records",
			totalRecords:      2,
			blockSize:         uint64(2 * targetBytesPerRequest),
			exemplars:         1,
			expectedExemplars: 2, // 1 * 1.2 -> rounded up to 2
			expectedBatches:   2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a test requests with exemplars enabled
			req := httptest.NewRequest("GET", "/test", nil)
			parentReq := pipeline.NewHTTPRequest(req)
			searchReq := tempopb.QueryRangeRequest{
				Query:     "test_query",
				Start:     uint64(time.Now().Add(-1 * time.Hour).UnixNano()),
				End:       uint64(time.Now().UnixNano()),
				Step:      uint64(60 * time.Second.Nanoseconds()),
				Exemplars: tc.exemplars,
			}

			// Create mock block metadata
			blockMeta := &backend.BlockMeta{
				BlockID:      backend.MustParse(uuid.NewString()),
				TotalRecords: tc.totalRecords,
				Size_:        tc.blockSize,
				StartTime:    time.Now().Add(-1 * time.Hour),
				EndTime:      time.Now(),
			}

			reqCh := make(chan pipeline.Request, 10)

			go func() {
				sharder.buildBackendRequests(t.Context(), tenantID, parentReq, searchReq, []*backend.BlockMeta{blockMeta}, targetBytesPerRequest, reqCh)
			}()

			// Collect requests
			var generatedRequests []pipeline.Request
			for req := range reqCh {
				generatedRequests = append(generatedRequests, req)
			}
			assert.Equal(t, tc.expectedBatches, len(generatedRequests), "Number of generated requests should match expected value")

			var totalExemplars int
			for _, req := range generatedRequests {
				uri := req.HTTPRequest().URL.String()
				exemplarsValue := extractExemplarsValue(t, uri)
				assert.Greater(t, exemplarsValue, 0, "Exemplars per batch should be at least 1")
				totalExemplars += exemplarsValue
			}
			assert.Equal(t, tc.expectedExemplars, totalExemplars, "Total exemplars should match expected value")
		})
	}
}

// extractExemplarsValue extracts the exemplars value from the URL
func extractExemplarsValue(t *testing.T, uri string) int {
	require.True(t, strings.Contains(uri, "exemplars="), "Request should contain exemplars parameter")
	exemplarsParam := ""
	for param := range strings.SplitSeq(uri, "&") {
		if strings.HasPrefix(param, "exemplars=") {
			exemplarsParam = strings.TrimPrefix(param, "exemplars=")
			break
		}
	}
	require.NotEmpty(t, exemplarsParam, "Exemplars parameter should not be empty")

	exemplarsValue, err := strconv.Atoi(exemplarsParam)
	require.NoError(t, err, "Should be able to parse exemplars value")

	return exemplarsValue
}

func TestExemplarsPerShard(t *testing.T) {
	s := &queryRangeSharder{}

	createBlockMeta := func(durationSeconds int) *backend.BlockMeta {
		now := time.Now()
		return &backend.BlockMeta{
			BlockID:   backend.MustParse(uuid.NewString()),
			StartTime: now.Add(-time.Duration(durationSeconds) * time.Second),
			EndTime:   now,
		}
	}

	testCases := []struct {
		name           string
		metas          []*backend.BlockMeta
		limit          uint32
		expectedResult []uint32
	}{
		{
			name:           "limit is zero",
			metas:          []*backend.BlockMeta{createBlockMeta(60)},
			limit:          0,
			expectedResult: []uint32{0},
		},
		{
			name:           "metas is empty",
			metas:          []*backend.BlockMeta{},
			limit:          100,
			expectedResult: []uint32{},
		},
		{
			name: "proportional distribution based on duration",
			metas: []*backend.BlockMeta{
				createBlockMeta(90),
				createBlockMeta(10),
			},
			limit:          100,
			expectedResult: []uint32{108, 12}, // 90*1.2 = 108, 10*1.2 = 12
		},
		{
			name: "at least one exemplar per valid block",
			metas: []*backend.BlockMeta{
				createBlockMeta(1000), // large block
				createBlockMeta(1),    // very small block
			},
			limit:          10,
			expectedResult: []uint32{12, 1}, // First gets 9*1.2, second gets at least 1
		},
		{
			name: "mixed valid and invalid blocks",
			metas: []*backend.BlockMeta{
				createBlockMeta(60),
				createBlockMeta(-60), // invalid block
				createBlockMeta(60),
			},
			limit:          100,
			expectedResult: []uint32{60, 0, 60},
		},
		{
			name: "only invalid blocks",
			metas: []*backend.BlockMeta{
				createBlockMeta(-60),
				createBlockMeta(-60),
			},
			limit:          100,
			expectedResult: []uint32{0, 0},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := s.exemplarsPerShard(tc.metas, tc.limit)
			assert.Equal(t, tc.expectedResult, result)
		})
	}
}

func FuzzExemplarsPerShard(f *testing.F) {
	f.Add(uint32(100), uint32(60)) // limit = 100, duration = 60s
	f.Add(uint32(0), uint32(30))   // limit = 0, duration = 30s
	f.Add(uint32(1000), uint32(0)) // limit = 1000, duration = 0s

	s := &queryRangeSharder{}

	f.Fuzz(func(t *testing.T, limit uint32, value uint32) {
		now := time.Now()
		metas := []*backend.BlockMeta{
			{
				BlockID:   backend.MustParse(uuid.NewString()),
				StartTime: now.Add(-time.Duration(value) * time.Second),
				EndTime:   now,
			},
		}

		result := s.exemplarsPerShard(metas, limit)
		require.Len(t, result, 1, "result should have one element")

		if limit == 0 || value == 0 {
			assert.Equal(t, uint32(0), result[0], "result should be 0")
		} else {
			assert.Greater(t, result[0], uint32(0), "result should be greater than 0")
		}
	})
}

// nolint: gosec // G115
func TestExemplarsCutoff(t *testing.T) {
	s := &queryRangeSharder{}
	now := time.Now()
	cutoff := now.Add(-1 * time.Hour)

	testCases := []struct {
		name              string
		req               tempopb.QueryRangeRequest
		expectedBeforeCut uint32
		expectedAfterCut  uint32
	}{
		{
			// When all data is after the cutoff, all exemplars should go to the 'after' portion
			name: "all data after cutoff",
			req: tempopb.QueryRangeRequest{
				Start:     uint64(cutoff.Add(50 * time.Minute).UnixNano()),
				End:       uint64(now.UnixNano()),
				Exemplars: 100,
			},
			expectedBeforeCut: 0,
			expectedAfterCut:  100,
		},
		{
			// When all data is before the cutoff, all exemplars should go to the 'before' portion
			name: "all data before cutoff",
			req: tempopb.QueryRangeRequest{
				Start:     uint64(cutoff.Add(-2 * time.Hour).UnixNano()),
				End:       uint64(cutoff.Add(-10 * time.Minute).UnixNano()),
				Exemplars: 100,
			},
			expectedBeforeCut: 100,
			expectedAfterCut:  0,
		},
		{
			name: "data spans the cutoff - 75% after",
			req: tempopb.QueryRangeRequest{
				Start:     uint64(cutoff.Add(-20 * time.Minute).UnixNano()),
				End:       uint64(now.UnixNano()),
				Exemplars: 100,
			},
			expectedBeforeCut: 25,
			expectedAfterCut:  75,
		},
		{
			name: "data spans the cutoff - 25% after",
			req: tempopb.QueryRangeRequest{
				Start:     uint64(cutoff.Add(-3 * time.Hour).UnixNano()),
				End:       uint64(cutoff.Add(1 * time.Hour).UnixNano()),
				Exemplars: 100,
			},
			expectedBeforeCut: 75,
			expectedAfterCut:  25,
		},
		// in case of small limits, it gives favor to after (request to generator)
		{
			name: "small limit: 25% after",
			req: tempopb.QueryRangeRequest{
				Start:     uint64(cutoff.Add(-3 * time.Hour).UnixNano()),
				End:       uint64(cutoff.Add(1 * time.Hour).UnixNano()),
				Exemplars: 2,
			},
			expectedBeforeCut: 1,
			expectedAfterCut:  1,
		},
		{
			name: "small limit: 25% after",
			req: tempopb.QueryRangeRequest{
				Start:     uint64(cutoff.Add(-3 * time.Hour).UnixNano()),
				End:       uint64(cutoff.Add(1 * time.Hour).UnixNano()),
				Exemplars: 1,
			},
			expectedBeforeCut: 0,
			expectedAfterCut:  1,
		},
		{
			name: "small limit: 75% after",
			req: tempopb.QueryRangeRequest{
				Start:     uint64(cutoff.Add(-20 * time.Minute).UnixNano()),
				End:       uint64(now.UnixNano()),
				Exemplars: 2,
			},
			expectedBeforeCut: 0,
			expectedAfterCut:  2,
		},
		{
			name: "small limit: 75% after",
			req: tempopb.QueryRangeRequest{
				Start:     uint64(cutoff.Add(-20 * time.Minute).UnixNano()),
				End:       uint64(now.UnixNano()),
				Exemplars: 1,
			},
			expectedBeforeCut: 0,
			expectedAfterCut:  1,
		},
		{
			name: "exactly at cutoff",
			req: tempopb.QueryRangeRequest{
				Start:     uint64(cutoff.UnixNano()),
				End:       uint64(now.UnixNano()),
				Exemplars: 100,
			},
			expectedBeforeCut: 0,
			expectedAfterCut:  100,
		},
		{
			name: "exactly at cutoff",
			req: tempopb.QueryRangeRequest{
				Start:     uint64(cutoff.Add(-1 * time.Hour).UnixNano()),
				End:       uint64(cutoff.UnixNano()),
				Exemplars: 100,
			},
			expectedBeforeCut: 100,
			expectedAfterCut:  0,
		},
		{
			name: "start equals end",
			req: tempopb.QueryRangeRequest{
				Start:     uint64(now.UnixNano()),
				End:       uint64(now.UnixNano()),
				Exemplars: 100,
			},
			expectedBeforeCut: 100,
			expectedAfterCut:  0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			beforeCut, afterCut := s.exemplarsCutoff(tc.req, cutoff)
			assert.Equal(t, tc.expectedBeforeCut, beforeCut, "Exemplars before cutoff should match expected value")
			assert.Equal(t, tc.expectedAfterCut, afterCut, "Exemplars after cutoff should match expected value")
		})
	}
}
