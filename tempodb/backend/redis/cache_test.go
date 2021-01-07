package redis

import (
	"context"
	"io"
	"testing"

	"github.com/alicebob/miniredis"
	"github.com/cortexproject/cortex/pkg/chunk/cache"
	"github.com/go-kit/kit/log"
	"github.com/google/uuid"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/stretchr/testify/assert"
)

type mockReader struct {
	tenants []string
	blocks  []uuid.UUID
	meta    *backend.BlockMeta
	read    []byte
}

func (m *mockReader) Tenants(ctx context.Context) ([]string, error) {
	return m.tenants, nil
}
func (m *mockReader) Blocks(ctx context.Context, tenantID string) ([]uuid.UUID, error) {
	return m.blocks, nil
}
func (m *mockReader) BlockMeta(ctx context.Context, blockID uuid.UUID, tenantID string) (*backend.BlockMeta, error) {
	return m.meta, nil
}
func (m *mockReader) Read(ctx context.Context, name string, blockID uuid.UUID, tenantID string) ([]byte, error) {
	return m.read, nil
}
func (m *mockReader) ReadRange(ctx context.Context, name string, blockID uuid.UUID, tenantID string, offset uint64, buffer []byte) error {
	return nil
}
func (m *mockReader) Shutdown() {}

type mockWriter struct {
}

func (m *mockWriter) Write(ctx context.Context, name string, blockID uuid.UUID, tenantID string, buffer []byte) error {
	return nil
}
func (m *mockWriter) WriteReader(ctx context.Context, name string, blockID uuid.UUID, tenantID string, data io.Reader, size int64) error {
	return nil
}
func (m *mockWriter) WriteBlockMeta(ctx context.Context, meta *backend.BlockMeta) error {
	return nil
}
func (m *mockWriter) Append(ctx context.Context, name string, blockID uuid.UUID, tenantID string, tracker backend.AppendTracker, buffer []byte) (backend.AppendTracker, error) {
	return nil, nil
}
func (m *mockWriter) CloseAppend(ctx context.Context, tracker backend.AppendTracker) error {
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
			mr, _ := miniredis.Run()
			mockR := &mockReader{
				tenants: tt.readerTenants,
				blocks:  tt.readerBlocks,
				meta:    tt.readerMeta,
				read:    tt.readerRead,
			}
			mockW := &mockWriter{}
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
			read, _ := rw.Read(ctx, "test", blockID, tenantID)
			assert.Equal(t, tt.expectedRead, read)

			// clear reader and re-request.  things should be cached!
			mockR.tenants = nil
			mockR.blocks = nil
			mockR.meta = nil

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
