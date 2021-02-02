package cache

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/util"
	"github.com/stretchr/testify/assert"
)

type mockClient struct {
	client map[string][]byte
}

func (m *mockClient) Store(_ context.Context, key string, val []byte) {
	m.client[key] = val
}

func (m *mockClient) Fetch(_ context.Context, key string) (val []byte) {
	val, ok := m.client[key]
	if ok {
		return val
	}
	return nil
}

func (m *mockClient) Shutdown() {
}

// NewMockClient makes a new mockClient.
func NewMockClient() Client {
	return &mockClient{
		client: map[string][]byte{},
	}
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

			rw, _, _ := NewCache(mockR, mockW, NewMockClient())

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
