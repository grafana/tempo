package memcached

import (
	"context"
	"testing"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/cortexproject/cortex/pkg/chunk/cache"
	"github.com/go-kit/kit/log"
	"github.com/google/uuid"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

type mockReader struct {
	tenants []string
	blocks  []uuid.UUID
	meta    *encoding.BlockMeta
	bloom   []byte
	index   []byte
	object  []byte
}

func (m *mockReader) Tenants() ([]string, error) {
	return m.tenants, nil
}
func (m *mockReader) Blocks(tenantID string) ([]uuid.UUID, error) {
	return m.blocks, nil
}
func (m *mockReader) BlockMeta(blockID uuid.UUID, tenantID string) (*encoding.BlockMeta, error) {
	return m.meta, nil
}
func (m *mockReader) Bloom(blockID uuid.UUID, tenantID string) ([]byte, error) {
	return m.bloom, nil
}
func (m *mockReader) Index(blockID uuid.UUID, tenantID string) ([]byte, error) {
	return m.index, nil
}
func (m *mockReader) Object(blockID uuid.UUID, tenantID string, offset uint64, buffer []byte) error {
	copy(buffer, m.object)
	return nil
}
func (m *mockReader) Shutdown() {}

type mockWriter struct {
}

func (m *mockWriter) Write(ctx context.Context, meta *encoding.BlockMeta, bBloom []byte, bIndex []byte, objectFilePath string) error {
	return nil
}
func (m *mockWriter) WriteBlockMeta(ctx context.Context, tracker backend.AppendTracker, meta *encoding.BlockMeta, bBloom []byte, bIndex []byte) error {
	return nil
}
func (m *mockWriter) AppendObject(ctx context.Context, tracker backend.AppendTracker, meta *encoding.BlockMeta, bObject []byte) (backend.AppendTracker, error) {
	return nil, nil
}

type mockCache struct {
	stuff map[string]*memcache.Item
}

func (m *mockCache) GetMulti(keys []string) (map[string]*memcache.Item, error) {
	retStuff := make(map[string]*memcache.Item)
	for k, v := range m.stuff {
		if k == keys[0] { // we only ever request one key at a time :(
			retStuff[k] = v
			break
		}
	}

	return m.stuff, nil
}
func (m *mockCache) Set(item *memcache.Item) error {
	m.stuff[item.Key] = item
	return nil
}

func TestCache(t *testing.T) {
	tenantID := "test"
	blockID := uuid.New()

	tests := []struct {
		name            string
		readerTenants   []string
		readerBlocks    []uuid.UUID
		readerMeta      *encoding.BlockMeta
		readerBloom     []byte
		readerIndex     []byte
		readerObject    []byte
		expectedTenants []string
		expectedBlocks  []uuid.UUID
		expectedMeta    *encoding.BlockMeta
		expectedBloom   []byte
		expectedIndex   []byte
		expectedObject  []byte
	}{
		{
			name:            "tenants passthrough",
			expectedTenants: []string{"1"},
			readerTenants:   []string{"1"},
		},
		{
			name:           "blocks passthrough",
			expectedBlocks: []uuid.UUID{blockID},
			readerBlocks:   []uuid.UUID{blockID},
		},
		{
			name:          "index",
			expectedIndex: []byte{0x01},
			readerIndex:   []byte{0x01},
		},
		{
			name:          "bloom",
			expectedBloom: []byte{0x02},
			readerBloom:   []byte{0x02},
		},
		{
			name:           "object",
			expectedObject: []byte{0x03},
			readerObject:   []byte{0x03},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockR := &mockReader{
				tenants: tt.readerTenants,
				blocks:  tt.readerBlocks,
				meta:    tt.readerMeta,
				bloom:   tt.readerBloom,
				index:   tt.readerIndex,
				object:  tt.readerObject,
			}
			mockW := &mockWriter{}
			mockC := &mockCache{
				stuff: make(map[string]*memcache.Item),
			}

			logger := log.NewNopLogger()
			rw := &readerWriter{
				client:     cache.NewMemcached(cache.MemcachedConfig{}, mockC, "tempo", prometheus.NewRegistry(), logger),
				nextReader: mockR,
				nextWriter: mockW,
				logger:     logger,
			}

			tenants, _ := rw.Tenants()
			assert.Equal(t, tt.expectedTenants, tenants)
			blocks, _ := rw.Blocks(tenantID)
			assert.Equal(t, tt.expectedBlocks, blocks)
			meta, _ := rw.BlockMeta(blockID, tenantID)
			assert.Equal(t, tt.expectedMeta, meta)
			bloom, _ := rw.Bloom(blockID, tenantID)
			assert.Equal(t, tt.expectedBloom, bloom)
			index, _ := rw.Index(blockID, tenantID)
			assert.Equal(t, tt.expectedIndex, index)

			if tt.expectedObject != nil {
				object := make([]byte, 1)
				rw.Object(blockID, tenantID, 0, object)
				assert.Equal(t, tt.expectedObject, object)
			}

			// clear reader and re-request.  things should be cached!
			mockR.bloom = nil
			mockR.index = nil
			mockR.tenants = nil
			mockR.blocks = nil
			mockR.meta = nil

			bloom, _ = rw.Bloom(blockID, tenantID)
			assert.Equal(t, tt.expectedBloom, bloom)
			index, _ = rw.Index(blockID, tenantID)
			assert.Equal(t, tt.expectedIndex, index)

			// others should be nil
			tenants, _ = rw.Tenants()
			assert.Nil(t, tenants)
			blocks, _ = rw.Blocks(tenantID)
			assert.Nil(t, blocks)
			meta, _ = rw.BlockMeta(blockID, tenantID)
			assert.Nil(t, tt.expectedMeta, meta)
		})
	}
}
