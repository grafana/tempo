package cache

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
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

// eofReader returns (data, io.EOF) on ReadRange, modeling an io.ReaderAt
// implementation that reaches end-of-object during a read.
type eofReader struct {
	*backend.MockRawReader
	data []byte
}

func (e *eofReader) ReadRange(_ context.Context, _ string, _ backend.KeyPath, _ uint64, buffer []byte, _ *backend.CacheInfo) error {
	copy(buffer, e.data)
	return io.EOF
}

// errReader returns a configurable non-nil non-EOF error on every ReadRange.
type errReader struct {
	*backend.MockRawReader
	err error
}

func (e *errReader) ReadRange(_ context.Context, _ string, _ backend.KeyPath, _ uint64, _ []byte, _ *backend.CacheInfo) error {
	return e.err
}

// captureLogger records every Log call so tests can assert log content.
type captureLogger struct {
	mu      sync.Mutex
	entries []map[string]interface{}
}

func (c *captureLogger) Log(keyvals ...interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	m := make(map[string]interface{}, len(keyvals)/2)
	for i := 0; i+1 < len(keyvals); i += 2 {
		k, _ := keyvals[i].(string)
		m[k] = keyvals[i+1]
	}
	c.entries = append(c.entries, m)
	return nil
}

// TestReadRangeStoresOnEOF pins the io.EOF-as-success behavior: the data is
// cached and a subsequent read hits.
func TestReadRangeStoresOnEOF(t *testing.T) {
	tenantID := "test"
	blockID := uuid.New()
	keypath := backend.KeyPathForBlock(blockID, tenantID)
	data := []byte{0x01, 0x02, 0x03, 0x04}

	provider := test.NewMockProvider()
	mockR := &eofReader{MockRawReader: &backend.MockRawReader{}, data: data}
	reader, _, err := NewCache(nil, mockR, &backend.MockRawWriter{}, provider, log.NewNopLogger())
	require.NoError(t, err)

	buf := make([]byte, len(data))
	err = reader.ReadRange(context.Background(), "data.parquet", keypath, 0, buf, &backend.CacheInfo{
		Role: cache.RoleParquetPage,
	})
	require.ErrorIs(t, err, io.EOF, "backend EOF must propagate to the caller")
	require.Equal(t, data, buf)

	// Second read of the same range hits the cache (no backend call).
	mockR.data = nil
	buf2 := make([]byte, len(data))
	err = reader.ReadRange(context.Background(), "data.parquet", keypath, 0, buf2, &backend.CacheInfo{
		Role: cache.RoleParquetPage,
	})
	require.NoError(t, err)
	require.Equal(t, data, buf2, "second read must be served from cache")
}

// TestReadRangeErrorClass verifies that backend errors increment the
// error-bytes counter with the expected class label.
func TestReadRangeErrorClass(t *testing.T) {
	tenantID := "test"
	blockID := uuid.New()
	keypath := backend.KeyPathForBlock(blockID, tenantID)

	cases := []struct {
		name  string
		inErr error
		want  string
	}{
		{"unexpected_eof", io.ErrUnexpectedEOF, "unexpected_eof"},
		{"canceled", context.Canceled, "canceled"},
		{"deadline_exceeded", context.DeadlineExceeded, "deadline_exceeded"},
		{"other", errors.New("some random error"), "other"},
		{"wrapped_canceled", fmt.Errorf("wrap: %w", context.Canceled), "canceled"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			baseClass := testutil.ToFloat64(pageCacheErrorBytes.WithLabelValues(tc.want))

			provider := test.NewMockProvider()
			mockR := &errReader{MockRawReader: &backend.MockRawReader{}, err: tc.inErr}
			reader, _, err := NewCache(nil, mockR, &backend.MockRawWriter{}, provider, log.NewNopLogger())
			require.NoError(t, err)

			buf := make([]byte, 32)
			_ = reader.ReadRange(context.Background(), "data.parquet", keypath, 0, buf, &backend.CacheInfo{
				Role: cache.RoleParquetPage,
			})

			require.Equal(t, baseClass+32, testutil.ToFloat64(pageCacheErrorBytes.WithLabelValues(tc.want)))
		})
	}
}

// TestReadRangeLogsErrors verifies the rate-limited error log fires with the
// expected fields when a non-nil non-EOF error reaches the cache layer.
func TestReadRangeLogsErrors(t *testing.T) {
	tenantID := "test"
	blockID := uuid.New()
	keypath := backend.KeyPathForBlock(blockID, tenantID)

	cl := &captureLogger{}
	provider := test.NewMockProvider()
	mockR := &errReader{MockRawReader: &backend.MockRawReader{}, err: context.Canceled}
	reader, _, err := NewCache(nil, mockR, &backend.MockRawWriter{}, provider, cl)
	require.NoError(t, err)

	buf := make([]byte, 64)
	_ = reader.ReadRange(context.Background(), "data.parquet", keypath, 12345, buf, &backend.CacheInfo{
		Role: cache.RoleParquetPage,
	})

	found := false
	cl.mu.Lock()
	defer cl.mu.Unlock()
	for _, e := range cl.entries {
		if e["msg"] == "cache write-path read error" {
			require.Equal(t, "canceled", e["class"])
			require.Equal(t, "data.parquet", e["name"])
			require.Equal(t, uint64(12345), e["offset"])
			require.Equal(t, 64, e["len"])
			found = true
			break
		}
	}
	require.True(t, found, "expected a cache write-path error log line; got entries: %v", cl.entries)
}

// TestReadRangeErrorIgnoresNonPageRoles ensures the error-bytes counter is
// scoped to RoleParquetPage and does not fire for other roles.
func TestReadRangeErrorIgnoresNonPageRoles(t *testing.T) {
	tenantID := "test"
	blockID := uuid.New()
	keypath := backend.KeyPathForBlock(blockID, tenantID)

	base := testutil.ToFloat64(pageCacheErrorBytes.WithLabelValues("other"))

	provider := test.NewMockProvider()
	mockR := &errReader{MockRawReader: &backend.MockRawReader{}, err: errors.New("boom")}
	reader, _, err := NewCache(nil, mockR, &backend.MockRawWriter{}, provider, log.NewNopLogger())
	require.NoError(t, err)

	buf := make([]byte, 64)
	_ = reader.ReadRange(context.Background(), "data.parquet", keypath, 0, buf, &backend.CacheInfo{
		Role: cache.RoleParquetFooter,
	})
	require.Equal(t, base, testutil.ToFloat64(pageCacheErrorBytes.WithLabelValues("other")))
}
