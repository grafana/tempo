package backend

import (
	"bytes"
	"context"
	tempo_io "github.com/grafana/tempo/pkg/io"
	"io"
	"io/ioutil"

	"github.com/google/uuid"
)

var _ RawReader = (*MockRawReader)(nil)
var _ RawWriter = (*MockRawWriter)(nil)
var _ Compactor = (*MockCompactor)(nil)

// MockRawReader
type MockRawReader struct {
	L      []string
	ListFn func(ctx context.Context, keypath KeyPath) ([]string, error)
	R      []byte // read
	Range  []byte // ReadRange
	ReadFn func(ctx context.Context, name string, keypath KeyPath, shouldCache bool) (io.ReadCloser, int64, error)
}

func (m *MockRawReader) List(ctx context.Context, keypath KeyPath) ([]string, error) {
	if m.ListFn != nil {
		return m.ListFn(ctx, keypath)
	}

	return m.L, nil
}
func (m *MockRawReader) Read(ctx context.Context, name string, keypath KeyPath, shouldCache bool) (io.ReadCloser, int64, error) {
	if m.ReadFn != nil {
		return m.ReadFn(ctx, name, keypath, shouldCache)
	}

	return ioutil.NopCloser(bytes.NewReader(m.R)), int64(len(m.R)), nil
}
func (m *MockRawReader) ReadRange(ctx context.Context, name string, keypath KeyPath, offset uint64, buffer []byte) error {
	copy(buffer, m.Range)

	return nil
}
func (m *MockRawReader) Shutdown() {}

// MockRawWriter
type MockRawWriter struct {
	writeBuffer       []byte
	writeStreamBuffer []byte
	appendBuffer      []byte
	closeAppendCalled bool
}

func (m *MockRawWriter) Write(ctx context.Context, name string, keypath KeyPath, data io.Reader, size int64, shouldCache bool) error {
	var err error
	m.writeBuffer, err = tempo_io.ReadAllWithEstimate(data, size)
	return err
}
func (m *MockRawWriter) Append(ctx context.Context, name string, keypath KeyPath, tracker AppendTracker, buffer []byte) (AppendTracker, error) {
	m.appendBuffer = buffer
	return nil, nil
}
func (m *MockRawWriter) CloseAppend(ctx context.Context, tracker AppendTracker) error {
	m.closeAppendCalled = true
	return nil
}

// MockCompactor
type MockCompactor struct {
	BlockMetaFn func(blockID uuid.UUID, tenantID string) (*CompactedBlockMeta, error)
}

func (c *MockCompactor) MarkBlockCompacted(blockID uuid.UUID, tenantID string) error {
	return nil
}

func (c *MockCompactor) ClearBlock(blockID uuid.UUID, tenantID string) error {
	return nil
}

func (c *MockCompactor) CompactedBlockMeta(blockID uuid.UUID, tenantID string) (*CompactedBlockMeta, error) {
	return c.BlockMetaFn(blockID, tenantID)
}

// MockReader
type MockReader struct {
	T           []string
	B           []uuid.UUID // blocks
	BlockFn     func(ctx context.Context, tenantID string) ([]uuid.UUID, error)
	M           *BlockMeta // meta
	BlockMetaFn func(ctx context.Context, blockID uuid.UUID, tenantID string) (*BlockMeta, error)
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
func (m *MockReader) BlockMeta(ctx context.Context, blockID uuid.UUID, tenantID string) (*BlockMeta, error) {
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

func (m *MockReader) StreamReader(ctx context.Context, name string, blockID uuid.UUID, tenantID string) (io.ReadCloser, int64, error) {
	panic("StreamReader is not yet supported for mock reader")
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
func (m *MockWriter) StreamWriter(ctx context.Context, name string, blockID uuid.UUID, tenantID string, data io.Reader, size int64) error {
	return nil
}
func (m *MockWriter) WriteBlockMeta(ctx context.Context, meta *BlockMeta) error {
	return nil
}
func (m *MockWriter) Append(ctx context.Context, name string, blockID uuid.UUID, tenantID string, tracker AppendTracker, buffer []byte) (AppendTracker, error) {
	return nil, nil
}
func (m *MockWriter) CloseAppend(ctx context.Context, tracker AppendTracker) error {
	return nil
}
