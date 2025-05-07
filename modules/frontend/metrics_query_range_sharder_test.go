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

func FuzzExemplarsPerShard(f *testing.F) {
	f.Add(uint32(1), uint32(10))  // total = 1, exemplars = 10
	f.Add(uint32(100), uint32(1)) // total = 100, exemplars = 1
	f.Add(uint32(10), uint32(0))  // total = 10, exemplars = 0

	s := &queryRangeSharder{}

	f.Fuzz(func(t *testing.T, total uint32, exemplars uint32) {
		result := s.exemplarsPerShard(total, exemplars)

		if exemplars == 0 || total == 0 {
			assert.Equal(t, uint32(0), result, "if exemplars is 0 or total is 0, result should be 0")
		} else {
			assert.Greater(t, result, uint32(0), "result should be greater than 0")
		}
	})
}

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
