package redis

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis"
	"github.com/cortexproject/cortex/pkg/chunk/cache"
	"github.com/go-kit/kit/log"
	"github.com/google/uuid"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding"
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

func (m *mockReader) Tenants(ctx context.Context) ([]string, error) {
	return m.tenants, nil
}
func (m *mockReader) Blocks(ctx context.Context, tenantID string) ([]uuid.UUID, error) {
	return m.blocks, nil
}
func (m *mockReader) BlockMeta(ctx context.Context, blockID uuid.UUID, tenantID string) (*encoding.BlockMeta, error) {
	return m.meta, nil
}
func (m *mockReader) Bloom(ctx context.Context, blockID uuid.UUID, tenantID string, shardNum int) ([]byte, error) {
	return m.bloom, nil
}
func (m *mockReader) Index(ctx context.Context, blockID uuid.UUID, tenantID string) ([]byte, error) {
	return m.index, nil
}
func (m *mockReader) Object(ctx context.Context, blockID uuid.UUID, tenantID string, offset uint64, buffer []byte) error {
	copy(buffer, m.object)
	return nil
}
func (m *mockReader) Shutdown() {}

type mockWriter struct {
}

func (m *mockWriter) Write(ctx context.Context, meta *encoding.BlockMeta, bBloom [][]byte, bIndex []byte, objectFilePath string) error {
	return nil
}
func (m *mockWriter) WriteBlockMeta(ctx context.Context, tracker backend.AppendTracker, meta *encoding.BlockMeta, bBloom [][]byte, bIndex []byte) error {
	return nil
}
func (m *mockWriter) AppendObject(ctx context.Context, tracker backend.AppendTracker, meta *encoding.BlockMeta, bObject []byte) (backend.AppendTracker, error) {
	return nil, nil
}

func TestCache(t *testing.T) {
	tenantID := "test"
	blockID := uuid.New()
	shardNum := 0

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
			mr, _ := miniredis.Run()

			mockC := cache.NewRedisClient(&cache.RedisConfig{
				Endpoint: mr.Addr(),
			})
			logger := log.NewNopLogger()
			rw := &readerWriter{
				client:     cache.NewRedisCache("tempo", mockC, logger),
				nextReader: mockR,
				nextWriter: mockW,
				logger:     logger,
			}

			ctx := context.Background()
			tenants, _ := rw.Tenants(ctx)
			assert.Equal(t, tt.expectedTenants, tenants)
			blocks, _ := rw.Blocks(ctx, tenantID)
			assert.Equal(t, tt.expectedBlocks, blocks)
			meta, _ := rw.BlockMeta(ctx, blockID, tenantID)
			assert.Equal(t, tt.expectedMeta, meta)
			bloom, _ := rw.Bloom(ctx, blockID, tenantID, shardNum)
			assert.Equal(t, tt.expectedBloom, bloom)
			index, _ := rw.Index(ctx, blockID, tenantID)
			assert.Equal(t, tt.expectedIndex, index)

			if tt.expectedObject != nil {
				object := make([]byte, 1)
				_ = rw.Object(ctx, blockID, tenantID, 0, object)
				assert.Equal(t, tt.expectedObject, object)
			}

			// clear reader and re-request.  things should be cached!
			mockR.bloom = nil
			mockR.index = nil
			mockR.tenants = nil
			mockR.blocks = nil
			mockR.meta = nil

			bloom, _ = rw.Bloom(ctx, blockID, tenantID, shardNum)
			assert.Equal(t, tt.expectedBloom, bloom)
			index, _ = rw.Index(ctx, blockID, tenantID)
			assert.Equal(t, tt.expectedIndex, index)

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
