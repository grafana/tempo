package cache

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/google/uuid"
	"github.com/grafana/tempo/pkg/cache"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCacheFor(t *testing.T) {
	reader, _, err := NewCache(&BloomConfig{
		CacheMaxBlockAge:        time.Hour,
		CacheMinCompactionLevel: 1,
	}, nil, nil, test.NewMockProvider(), log.NewNopLogger())
	require.NoError(t, err)

	rw := reader.(*readerWriter)

	testCases := []struct {
		name          string
		cacheInfo     *backend.CacheInfo
		expectedCache cache.Cache
	}{
		// first three caches are unconditionally returned
		{
			name:          "footer is always returned",
			cacheInfo:     &backend.CacheInfo{Role: cache.RoleParquetFooter},
			expectedCache: rw.footerCache,
		},
		{
			name:          "col idx is always returned",
			cacheInfo:     &backend.CacheInfo{Role: cache.RoleParquetColumnIdx},
			expectedCache: rw.columnIdxCache,
		},
		{
			name:          "offset idx is always returned",
			cacheInfo:     &backend.CacheInfo{Role: cache.RoleParquetOffsetIdx},
			expectedCache: rw.offsetIdxCache,
		},
		{
			name:          "trace id idx is always returned",
			cacheInfo:     &backend.CacheInfo{Role: cache.RoleTraceIDIdx},
			expectedCache: rw.traceIDIdxCache,
		},
		// bloom cache is returned if the meta is valid given the bloom config
		{
			name: "bloom - no meta means no cache",
			cacheInfo: &backend.CacheInfo{
				Role: cache.RoleBloom,
			},
		},
		{
			name: "bloom - compaction lvl and start time valid",
			cacheInfo: &backend.CacheInfo{
				Role: cache.RoleBloom,
				Meta: &backend.BlockMeta{CompactionLevel: 1, StartTime: time.Now()},
			},
			expectedCache: rw.bloomCache,
		},
		{
			name: "bloom - start time invalid",
			cacheInfo: &backend.CacheInfo{
				Role: cache.RoleBloom,
				Meta: &backend.BlockMeta{CompactionLevel: 1, StartTime: time.Now().Add(-2 * time.Hour)},
			},
		},
		{
			name: "bloom - compaction lvl invalid",
			cacheInfo: &backend.CacheInfo{
				Role: cache.RoleBloom,
				Meta: &backend.BlockMeta{CompactionLevel: 2, StartTime: time.Now()},
			},
		},
		{
			name: "bloom - both invalid",
			cacheInfo: &backend.CacheInfo{
				Role: cache.RoleBloom,
				Meta: &backend.BlockMeta{CompactionLevel: 2, StartTime: time.Now().Add(-2 * time.Hour)},
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expectedCache, rw.cacheFor(tt.cacheInfo))
		})
	}
}

func TestCacheForReturnsBloomWithNoConfig(t *testing.T) {
	reader, _, err := NewCache(nil, nil, nil, test.NewMockProvider(), log.NewNopLogger())
	require.NoError(t, err)

	rw := reader.(*readerWriter)

	testCases := []struct {
		name          string
		cacheInfo     *backend.CacheInfo
		expectedCache cache.Cache
	}{
		// bloom cache is returned if the meta is valid given the bloom config
		{
			name: "bloom - no meta means no cache",
			cacheInfo: &backend.CacheInfo{
				Role: cache.RoleBloom,
			},
			expectedCache: rw.bloomCache,
		},
		{
			name: "bloom - compaction lvl and start time valid",
			cacheInfo: &backend.CacheInfo{
				Role: cache.RoleBloom,
				Meta: &backend.BlockMeta{CompactionLevel: 1, StartTime: time.Now()},
			},
			expectedCache: rw.bloomCache,
		},
		{
			name: "bloom - start time invalid",
			cacheInfo: &backend.CacheInfo{
				Role: cache.RoleBloom,
				Meta: &backend.BlockMeta{CompactionLevel: 1, StartTime: time.Now().Add(-2 * time.Hour)},
			},
			expectedCache: rw.bloomCache,
		},
		{
			name: "bloom - compaction lvl invalid",
			cacheInfo: &backend.CacheInfo{
				Role: cache.RoleBloom,
				Meta: &backend.BlockMeta{CompactionLevel: 2, StartTime: time.Now()},
			},
			expectedCache: rw.bloomCache,
		},
		{
			name: "bloom - both invalid",
			cacheInfo: &backend.CacheInfo{
				Role: cache.RoleBloom,
				Meta: &backend.BlockMeta{CompactionLevel: 2, StartTime: time.Now().Add(-2 * time.Hour)},
			},
			expectedCache: rw.bloomCache,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expectedCache, rw.cacheFor(tt.cacheInfo))
		})
	}
}

func TestReadWrite(t *testing.T) {
	tenantID := "test"
	blockID := uuid.New()

	tests := []struct {
		name          string
		readerRead    []byte
		readerName    string
		cacheInfo     *backend.CacheInfo
		expectedRead  []byte
		expectedCache []byte
	}{
		{
			name:          "should cache",
			readerName:    "foo",
			readerRead:    []byte{0x02},
			cacheInfo:     &backend.CacheInfo{Role: cache.RoleParquetFooter},
			expectedRead:  []byte{0x02},
			expectedCache: []byte{0x02},
		},
		{
			name:         "should not cache",
			readerName:   "bar",
			cacheInfo:    &backend.CacheInfo{Role: cache.Role("foo")}, // fake role name will not find a matching cache
			readerRead:   []byte{0x02},
			expectedRead: []byte{0x02},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockR := &backend.MockRawReader{
				R: tt.readerRead,
			}
			mockW := &backend.MockRawWriter{}

			// READ
			r, _, err := NewCache(nil, mockR, mockW, test.NewMockProvider(), log.NewNopLogger())
			require.NoError(t, err)

			ctx := context.Background()
			reader, _, _ := r.Read(ctx, tt.readerName, backend.KeyPathForBlock(blockID, tenantID), tt.cacheInfo)
			read, _ := io.ReadAll(reader)
			assert.Equal(t, tt.expectedRead, read)

			// clear reader and re-request
			mockR.R = nil

			reader, _, _ = r.Read(ctx, tt.readerName, backend.KeyPathForBlock(blockID, tenantID), tt.cacheInfo)
			read, _ = io.ReadAll(reader)
			assert.Equal(t, len(tt.expectedCache), len(read))

			// WRITE
			_, w, err := NewCache(nil, mockR, mockW, test.NewMockProvider(), log.NewNopLogger())
			require.NoError(t, err)

			_ = w.Write(ctx, tt.readerName, backend.KeyPathForBlock(blockID, tenantID), bytes.NewReader(tt.readerRead), int64(len(tt.readerRead)), tt.cacheInfo)
			reader, _, _ = r.Read(ctx, tt.readerName, backend.KeyPathForBlock(blockID, tenantID), tt.cacheInfo)
			read, _ = io.ReadAll(reader)
			assert.Equal(t, len(tt.expectedCache), len(read))
		})
	}
}

func TestList(t *testing.T) {
	tenantID := "test"
	blockID := uuid.New()

	tests := []struct {
		name          string
		readerList    []string
		expectedList  []string
		expectedCache []string
	}{
		{
			name:          "list passthrough",
			readerList:    []string{"1"},
			expectedList:  []string{"1"},
			expectedCache: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockR := &backend.MockRawReader{
				L: tt.readerList,
			}
			mockW := &backend.MockRawWriter{}

			rw, _, _ := NewCache(nil, mockR, mockW, test.NewMockProvider(), log.NewNopLogger())

			ctx := context.Background()
			list, _ := rw.List(ctx, backend.KeyPathForBlock(blockID, tenantID))
			assert.Equal(t, tt.expectedList, list)

			// clear reader and re-request.  things should be cached!
			mockR.L = nil

			// list is not cached
			list, _ = rw.List(ctx, backend.KeyPathForBlock(blockID, tenantID))
			assert.Equal(t, tt.expectedCache, list)
		})
	}
}
