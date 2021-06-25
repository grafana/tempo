package test

import (
	"context"
	"io"

	"github.com/google/uuid"
	"github.com/grafana/tempo/tempodb/backend"
)

var _ backend.Reader = (*MockReader)(nil)
var _ backend.Writer = (*MockWriter)(nil)
var _ backend.Compactor = (*MockCompactor)(nil)

// MockReader
type MockReader struct {
	L      []string
	ListFn func(ctx context.Context, keypath backend.KeyPath) ([]string, error)
	R      []byte // read
	Range  []byte // ReadRange
	ReadFn func(name string, keypath backend.KeyPath) ([]byte, error)
}

func (m *MockReader) List(ctx context.Context, keypath backend.KeyPath) ([]string, error) {
	if m.ListFn != nil {
		return m.ListFn(ctx, keypath)
	}

	return m.L, nil
}
func (m *MockReader) Read(ctx context.Context, name string, keypath backend.KeyPath) ([]byte, error) {
	if m.ReadFn != nil {
		return m.ReadFn(name, keypath)
	}

	return m.R, nil
}
func (m *MockReader) ReadReader(ctx context.Context, name string, keypath backend.KeyPath) (io.ReadCloser, int64, error) {
	panic("ReadReader is not yet supported for mock reader")
}
func (m *MockReader) ReadRange(ctx context.Context, name string, keypath backend.KeyPath, offset uint64, buffer []byte) error {
	copy(buffer, m.Range)

	return nil
}
func (m *MockReader) Shutdown() {}

// MockWriter
type MockWriter struct {
}

func (m *MockWriter) Write(ctx context.Context, name string, keypath backend.KeyPath, buffer []byte) error {
	return nil
}
func (m *MockWriter) WriteReader(ctx context.Context, name string, keypath backend.KeyPath, data io.Reader, size int64) error {
	return nil
}
func (m *MockWriter) Append(ctx context.Context, name string, keypath backend.KeyPath, tracker backend.AppendTracker, buffer []byte) (backend.AppendTracker, error) {
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
