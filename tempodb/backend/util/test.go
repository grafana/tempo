package util // jpe : rename test

import (
	"context"
	"io"

	"github.com/google/uuid"
	"github.com/grafana/tempo/tempodb/backend"
)

// MockReader
type MockReader struct {
	T           []string
	B           []uuid.UUID // blocks
	BlockFn     func(ctx context.Context, tenantID string) ([]uuid.UUID, error)
	M           *backend.BlockMeta // meta
	BlockMetaFn func(ctx context.Context, blockID uuid.UUID, tenantID string) (*backend.BlockMeta, error)
	R           []byte // read
	Range       []byte // ReadRange
	ReadFn      func(name string, blockID uuid.UUID, tenantID string) ([]byte, error)
}

func (m *MockReader) Tenants(ctx context.Context) ([]string, error) {
	return m.T, nil
}
func (m *MockReader) Blocks(ctx context.Context, tenantID string) ([]uuid.UUID, error) {
	if m.BlockFn != nil {
		return m.BlockFn(ctx, tenantID)
	}

	return m.B, nil
}
func (m *MockReader) BlockMeta(ctx context.Context, blockID uuid.UUID, tenantID string) (*backend.BlockMeta, error) {
	if m.BlockMetaFn != nil {
		return m.BlockMetaFn(ctx, blockID, tenantID)
	}

	return m.M, nil
}
func (m *MockReader) Read(ctx context.Context, name string, blockID uuid.UUID, tenantID string) ([]byte, error) {
	if m.ReadFn != nil {
		return m.ReadFn(name, blockID, tenantID)
	}

	return m.R, nil
}

func (m *MockReader) ReadReader(ctx context.Context, name string, blockID uuid.UUID, tenantID string) (io.ReadCloser, int64, error) {
	panic("ReadReader is not yet supported for mock reader")
}

func (m *MockReader) ReadRange(ctx context.Context, name string, blockID uuid.UUID, tenantID string, offset uint64, buffer []byte) error {
	copy(buffer, m.Range)

	return nil
}
func (m *MockReader) Shutdown() {}

// MockWriter
type MockWriter struct {
}

func (m *MockWriter) Write(ctx context.Context, name string, blockID uuid.UUID, tenantID string, buffer []byte) error {
	return nil
}
func (m *MockWriter) WriteReader(ctx context.Context, name string, blockID uuid.UUID, tenantID string, data io.Reader, size int64) error {
	return nil
}
func (m *MockWriter) WriteBlockMeta(ctx context.Context, meta *backend.BlockMeta) error {
	return nil
}
func (m *MockWriter) Append(ctx context.Context, name string, blockID uuid.UUID, tenantID string, tracker backend.AppendTracker, buffer []byte) (backend.AppendTracker, error) {
	return nil, nil
}
func (m *MockWriter) CloseAppend(ctx context.Context, tracker backend.AppendTracker) error {
	return nil
}

// MockCompactor
type MockCompactor struct {
	BlockMetaFn func(blockID uuid.UUID, tenantID string) (*backend.CompactedBlockMeta, error)
}

func (c *MockCompactor) MarkBlockCompacted(blockID uuid.UUID, tenantID string) error {
	return nil
}

func (c *MockCompactor) ClearBlock(blockID uuid.UUID, tenantID string) error {
	return nil
}

func (c *MockCompactor) CompactedBlockMeta(blockID uuid.UUID, tenantID string) (*backend.CompactedBlockMeta, error) {
	return c.BlockMetaFn(blockID, tenantID)
}
