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
	"github.com/prometheus/client_golang/prometheus/testutil"
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

	// test.MockProvider will return the same cache for all requests. we need
	// to override individual caches so that the test can validate the cache returned
	rw.footerCache = test.NewMockClient()
	rw.columnIdxCache = test.NewMockClient()
	rw.offsetIdxCache = test.NewMockClient()
	rw.traceIDIdxCache = test.NewMockClient()
	rw.bloomCache = test.NewMockClient()

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

func TestCacheKeys(t *testing.T) {
	provider := test.NewMockProvider()
	reader, _, err := NewCache(&BloomConfig{
		CacheMaxBlockAge:        time.Hour,
		CacheMinCompactionLevel: 1,
	}, nil, nil, provider, log.NewNopLogger())
	require.NoError(t, err)

	ctx := context.Background()
	role := cache.RoleParquetFooter // role doesn't matter b/c the mock provider always returns the same cache

	// Read : seed data at expected key
	expectedKey := "bar:baz:foo" // keypath + object name
	expectedData := []byte("test-read")
	provider.CacheFor(role).Store(ctx, []string{expectedKey}, [][]byte{expectedData})

	// make request and confirm it returns
	actualReader, actualBytes, err := reader.Read(ctx, "foo", backend.KeyPath{"bar", "baz"}, &backend.CacheInfo{Role: role})
	require.NoError(t, err)
	require.Equal(t, len(expectedData), int(actualBytes))

	actualData, err := io.ReadAll(actualReader)
	require.NoError(t, err)
	require.Equal(t, expectedData, actualData)

	// ReadRange : seed data at expected key
	expectedKey = "bar:baz:foo:10:10" // keypath + object name + offset + length
	expectedData = []byte("test-range")
	provider.CacheFor(role).Store(ctx, []string{expectedKey}, [][]byte{expectedData})

	// make request and confirm it returns
	actualBuffer := make([]byte, 10)
	err = reader.ReadRange(ctx, "foo", backend.KeyPath{"bar", "baz"}, 10, actualBuffer, &backend.CacheInfo{Role: role})
	require.NoError(t, err)
	require.Equal(t, expectedData, actualBuffer)
}

func TestCacheRequestCounters(t *testing.T) {
	tenantID := "test"
	blockID := uuid.New()
	keypath := backend.KeyPathForBlock(blockID, tenantID)

	// The metric helpers below read the current counter value for a given role.
	// Each subtest captures before/after values so the assertions stay hermetic
	// across parallel runs that share the package-level counter vars.
	hits := func(role string) float64 {
		return testutil.ToFloat64(cacheRequests.WithLabelValues(role, cacheOutcomeHit))
	}
	misses := func(role string) float64 {
		return testutil.ToFloat64(cacheRequests.WithLabelValues(role, cacheOutcomeMiss))
	}
	hitBytes := func(role string) float64 {
		return testutil.ToFloat64(cacheRequestBytes.WithLabelValues(role, cacheOutcomeHit))
	}
	missBytes := func(role string) float64 {
		return testutil.ToFloat64(cacheRequestBytes.WithLabelValues(role, cacheOutcomeMiss))
	}

	t.Run("Read miss then hit, per role", func(t *testing.T) {
		role := cache.RoleParquetFooter
		roleStr := string(role)
		hitsBefore, missesBefore := hits(roleStr), misses(roleStr)
		hitBytesBefore, missBytesBefore := hitBytes(roleStr), missBytes(roleStr)

		payload := []byte{0x01, 0x02, 0x03, 0x04}
		mockR := &backend.MockRawReader{R: payload}
		rw, _, err := NewCache(nil, mockR, &backend.MockRawWriter{}, test.NewMockProvider(), log.NewNopLogger())
		require.NoError(t, err)
		ctx := context.Background()

		// First read: miss (cache empty) then populate.
		r1, _, err := rw.Read(ctx, "foo", keypath, &backend.CacheInfo{Role: role})
		require.NoError(t, err)
		defer r1.Close()
		_, _ = io.ReadAll(r1)

		assert.Equal(t, missesBefore+1, misses(roleStr), "miss counter incremented")
		assert.Equal(t, hitsBefore, hits(roleStr), "hit counter unchanged")
		assert.Equal(t, missBytesBefore+float64(len(payload)), missBytes(roleStr), "miss bytes incremented")

		// Second read: hit (served from cache).
		mockR.R = nil
		r2, _, err := rw.Read(ctx, "foo", keypath, &backend.CacheInfo{Role: role})
		require.NoError(t, err)
		defer r2.Close()
		_, _ = io.ReadAll(r2)

		assert.Equal(t, hitsBefore+1, hits(roleStr), "hit counter incremented")
		assert.Equal(t, hitBytesBefore+float64(len(payload)), hitBytes(roleStr), "hit bytes incremented")
	})

	t.Run("ReadRange miss then hit, per role", func(t *testing.T) {
		role := cache.RoleParquetPage
		roleStr := string(role)
		hitsBefore, missesBefore := hits(roleStr), misses(roleStr)
		hitBytesBefore, missBytesBefore := hitBytes(roleStr), missBytes(roleStr)

		payload := []byte{0xaa, 0xbb, 0xcc, 0xdd}
		mockR := &backend.MockRawReader{Range: payload}
		rw, _, err := NewCache(nil, mockR, &backend.MockRawWriter{}, test.NewMockProvider(), log.NewNopLogger())
		require.NoError(t, err)
		ctx := context.Background()

		// First ReadRange: miss then populate.
		buffer := make([]byte, len(payload))
		require.NoError(t, rw.ReadRange(ctx, "bar", keypath, 0, buffer, &backend.CacheInfo{Role: role}))

		assert.Equal(t, missesBefore+1, misses(roleStr), "miss counter incremented")
		assert.Equal(t, hitsBefore, hits(roleStr), "hit counter unchanged")
		assert.Equal(t, missBytesBefore+float64(len(buffer)), missBytes(roleStr), "miss bytes incremented")

		// Second ReadRange: hit (served from cache).
		mockR.Range = nil
		buffer = make([]byte, len(payload))
		require.NoError(t, rw.ReadRange(ctx, "bar", keypath, 0, buffer, &backend.CacheInfo{Role: role}))

		assert.Equal(t, hitsBefore+1, hits(roleStr), "hit counter incremented")
		assert.Equal(t, hitBytesBefore+float64(len(payload)), hitBytes(roleStr), "hit bytes incremented")
	})

	t.Run("per-role isolation: hit on one role does not move another role's counters", func(t *testing.T) {
		hitRole := cache.RoleBloom
		otherRole := cache.RoleTraceIDIdx
		otherHitsBefore := hits(string(otherRole))
		otherMissesBefore := misses(string(otherRole))
		otherHitBytesBefore := hitBytes(string(otherRole))
		otherMissBytesBefore := missBytes(string(otherRole))

		payload := []byte{0x10, 0x20, 0x30}
		mockR := &backend.MockRawReader{R: payload}
		rw, _, err := NewCache(nil, mockR, &backend.MockRawWriter{}, test.NewMockProvider(), log.NewNopLogger())
		require.NoError(t, err)
		ctx := context.Background()

		// Drive a miss then a hit on hitRole.
		r1, _, err := rw.Read(ctx, "foo", keypath, &backend.CacheInfo{Role: hitRole})
		require.NoError(t, err)
		defer r1.Close()
		_, _ = io.ReadAll(r1)
		mockR.R = nil
		r2, _, err := rw.Read(ctx, "foo", keypath, &backend.CacheInfo{Role: hitRole})
		require.NoError(t, err)
		defer r2.Close()
		_, _ = io.ReadAll(r2)

		// otherRole's counters are unchanged.
		assert.Equal(t, otherHitsBefore, hits(string(otherRole)))
		assert.Equal(t, otherMissesBefore, misses(string(otherRole)))
		assert.Equal(t, otherHitBytesBefore, hitBytes(string(otherRole)))
		assert.Equal(t, otherMissBytesBefore, missBytes(string(otherRole)))
	})

	t.Run("bypass when no cache configured for role", func(t *testing.T) {
		// A role with no configured cache (cacheFor returns nil) must not move the counters.
		role := cache.Role("nonexistent-role")
		roleStr := string(role)
		hitsBefore, missesBefore := hits(roleStr), misses(roleStr)
		hitBytesBefore, missBytesBefore := hitBytes(roleStr), missBytes(roleStr)

		mockR := &backend.MockRawReader{R: []byte{0xff}}
		rw, _, err := NewCache(nil, mockR, &backend.MockRawWriter{}, test.NewMockProvider(), log.NewNopLogger())
		require.NoError(t, err)
		ctx := context.Background()

		reader, _, err := rw.Read(ctx, "foo", keypath, &backend.CacheInfo{Role: role})
		require.NoError(t, err)
		defer reader.Close()
		_, _ = io.ReadAll(reader)

		assert.Equal(t, hitsBefore, hits(roleStr))
		assert.Equal(t, missesBefore, misses(roleStr))
		assert.Equal(t, hitBytesBefore, hitBytes(roleStr))
		assert.Equal(t, missBytesBefore, missBytes(roleStr))
	})
}
