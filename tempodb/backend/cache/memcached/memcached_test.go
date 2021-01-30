package memcached

import (
	"context"
	"testing"

	"github.com/bradfitz/gomemcache/memcache"
	cortex_cache "github.com/cortexproject/cortex/pkg/chunk/cache"
	"github.com/go-kit/kit/log"
	"github.com/google/uuid"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/cache"
	"github.com/grafana/tempo/tempodb/backend/util"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

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
		readerMeta      *backend.BlockMeta
		readerRead      []byte
		expectedTenants []string
		expectedBlocks  []uuid.UUID
		expectedMeta    *backend.BlockMeta
		expectedRead    []byte
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
			name:         "read",
			expectedRead: []byte{0x02},
			readerRead:   []byte{0x02},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockR := &util.MockReader{
				T: tt.readerTenants,
				B: tt.readerBlocks,
				M: tt.readerMeta,
				R: tt.readerRead,
			}
			mockW := &util.MockWriter{}
			mockC := &mockCache{
				stuff: make(map[string]*memcache.Item),
			}

			logger := log.NewNopLogger()

			rw, _, _ := cache.NewCache(mockR, mockW, &Client{
				client: cortex_cache.NewMemcached(cortex_cache.MemcachedConfig{}, mockC, "tempo", prometheus.NewRegistry(), logger),
			})

			ctx := context.Background()
			tenants, _ := rw.Tenants(ctx)
			assert.Equal(t, tt.expectedTenants, tenants)
			blocks, _ := rw.Blocks(ctx, tenantID)
			assert.Equal(t, tt.expectedBlocks, blocks)
			meta, _ := rw.BlockMeta(ctx, blockID, tenantID)
			assert.Equal(t, tt.expectedMeta, meta)
			read, _ := rw.Read(ctx, "test", blockID, tenantID)
			assert.Equal(t, tt.expectedRead, read)

			// clear reader and re-request.  things should be cached!
			mockR.T = nil
			mockR.B = nil
			mockR.M = nil

			read, _ = rw.Read(ctx, "test", blockID, tenantID)
			assert.Equal(t, tt.expectedRead, read)

			// others should be nil
			tenants, _ = rw.Tenants(ctx)
			assert.Nil(t, tenants)
			blocks, _ = rw.Blocks(ctx, tenantID)
			assert.Nil(t, blocks)
			meta, _ = rw.BlockMeta(ctx, blockID, tenantID)
			assert.Nil(t, tt.expectedMeta, meta)
		})
	}
}
