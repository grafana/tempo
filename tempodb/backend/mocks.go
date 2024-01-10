package backend

import (
	"bytes"
	"context"
	"io"
	"strings"
	"sync"

	tempo_io "github.com/grafana/tempo/pkg/io"

	"github.com/google/uuid"
)

var (
	_ RawReader = (*MockRawReader)(nil)
	_ RawWriter = (*MockRawWriter)(nil)
	_ Reader    = (*MockReader)(nil)
	_ Writer    = (*MockWriter)(nil)
	_ Compactor = (*MockCompactor)(nil)
)

// MockRawReader
type MockRawReader struct {
	L            []string
	ListFn       func(ctx context.Context, keypath KeyPath) ([]string, error)
	ListBlocksFn func(ctx context.Context, tenant string) ([]uuid.UUID, []uuid.UUID, error)
	R            []byte // read
	Range        []byte // ReadRange
	ReadFn       func(ctx context.Context, name string, keypath KeyPath, cacheInfo *CacheInfo) (io.ReadCloser, int64, error)

	BlockIDs          []uuid.UUID
	CompactedBlockIDs []uuid.UUID
	FindResult        []string
}

func (m *MockRawReader) List(ctx context.Context, keypath KeyPath) ([]string, error) {
	if m.ListFn != nil {
		return m.ListFn(ctx, keypath)
	}

	return m.L, nil
}

func (m *MockRawReader) ListBlocks(ctx context.Context, tenant string) ([]uuid.UUID, []uuid.UUID, error) {
	if m.ListBlocksFn != nil {
		return m.ListBlocksFn(ctx, tenant)
	}

	return m.BlockIDs, m.CompactedBlockIDs, nil
}

func (m *MockRawReader) Read(ctx context.Context, name string, keypath KeyPath, cacheInfo *CacheInfo) (io.ReadCloser, int64, error) {
	if m.ReadFn != nil {
		return m.ReadFn(ctx, name, keypath, cacheInfo)
	}

	return io.NopCloser(bytes.NewReader(m.R)), int64(len(m.R)), nil
}

func (m *MockRawReader) ReadRange(_ context.Context, _ string, _ KeyPath, _ uint64, buffer []byte, _ *CacheInfo) error {
	copy(buffer, m.Range)

	return nil
}

func (m *MockRawReader) Shutdown() {}

// MockRawWriter
type MockRawWriter struct {
	writeBuffer       []byte
	appendBuffer      []byte
	closeAppendCalled bool
	deleteCalls       map[string]map[string]int
	err               error
}

func (m *MockRawWriter) Write(_ context.Context, _ string, _ KeyPath, data io.Reader, size int64, _ *CacheInfo) error {
	var err error
	m.writeBuffer, err = tempo_io.ReadAllWithEstimate(data, size)
	return err
}

func (m *MockRawWriter) Append(_ context.Context, _ string, _ KeyPath, _ AppendTracker, buffer []byte) (AppendTracker, error) {
	m.appendBuffer = buffer
	return nil, nil
}

func (m *MockRawWriter) CloseAppend(context.Context, AppendTracker) error {
	m.closeAppendCalled = true
	return nil
}

func (m *MockRawWriter) Delete(_ context.Context, name string, keypath KeyPath, _ *CacheInfo) error {
	if m.deleteCalls == nil {
		m.deleteCalls = make(map[string]map[string]int)
	}

	if _, ok := m.deleteCalls[name]; !ok {
		m.deleteCalls[name] = make(map[string]int)
	}

	if _, ok := m.deleteCalls[name][strings.Join(keypath, "/")]; !ok {
		m.deleteCalls[name][strings.Join(keypath, "/")] = 0
	}

	m.deleteCalls[name][strings.Join(keypath, "/")]++
	return m.err
}

// MockCompactor
type MockCompactor struct {
	sync.Mutex

	BlockMetaFn             func(blockID uuid.UUID, tenantID string) (*CompactedBlockMeta, error)
	CompactedBlockMetaCalls map[string]map[uuid.UUID]int
}

func (c *MockCompactor) MarkBlockCompacted(uuid.UUID, string) error {
	return nil
}

func (c *MockCompactor) ClearBlock(uuid.UUID, string) error {
	return nil
}

func (c *MockCompactor) CompactedBlockMeta(blockID uuid.UUID, tenantID string) (*CompactedBlockMeta, error) {
	c.Lock()
	defer c.Unlock()
	if c.CompactedBlockMetaCalls == nil {
		c.CompactedBlockMetaCalls = make(map[string]map[uuid.UUID]int)
	}
	if _, ok := c.CompactedBlockMetaCalls[tenantID]; !ok {
		c.CompactedBlockMetaCalls[tenantID] = make(map[uuid.UUID]int)
	}
	c.CompactedBlockMetaCalls[tenantID][blockID]++

	return c.BlockMetaFn(blockID, tenantID)
}

// MockReader
type MockReader struct {
	sync.Mutex

	T                 []string
	BlocksFn          func(ctx context.Context, tenantID string) ([]uuid.UUID, []uuid.UUID, error)
	M                 *BlockMeta // meta
	BlockMetaFn       func(ctx context.Context, blockID uuid.UUID, tenantID string) (*BlockMeta, error)
	TenantIndexFn     func(ctx context.Context, tenantID string) (*TenantIndex, error)
	R                 []byte // read
	Range             []byte // ReadRange
	ReadFn            func(name string, blockID uuid.UUID, tenantID string) ([]byte, error)
	BlockMetaCalls    map[string]map[uuid.UUID]int
	BlockIDs          []uuid.UUID // blocks
	CompactedBlockIDs []uuid.UUID // blocks
}

func (m *MockReader) Tenants(context.Context) ([]string, error) {
	return m.T, nil
}

func (m *MockReader) Blocks(ctx context.Context, tenantID string) ([]uuid.UUID, []uuid.UUID, error) {
	if m.BlocksFn != nil {
		return m.BlocksFn(ctx, tenantID)
	}

	return m.BlockIDs, m.CompactedBlockIDs, nil
}

func (m *MockReader) BlockMeta(ctx context.Context, blockID uuid.UUID, tenantID string) (*BlockMeta, error) {
	m.Lock()
	defer m.Unlock()

	// Update the BlockMetaCalls map based on the tenantID and blockID
	if m.BlockMetaCalls == nil {
		m.BlockMetaCalls = make(map[string]map[uuid.UUID]int)
	}
	if _, ok := m.BlockMetaCalls[tenantID]; !ok {
		m.BlockMetaCalls[tenantID] = make(map[uuid.UUID]int)
	}
	m.BlockMetaCalls[tenantID][blockID]++

	if m.BlockMetaFn != nil {
		return m.BlockMetaFn(ctx, blockID, tenantID)
	}

	return m.M, nil
}

func (m *MockReader) Read(_ context.Context, name string, blockID uuid.UUID, tenantID string, _ *CacheInfo) ([]byte, error) {
	if m.ReadFn != nil {
		return m.ReadFn(name, blockID, tenantID)
	}

	return m.R, nil
}

func (m *MockReader) StreamReader(context.Context, string, uuid.UUID, string) (io.ReadCloser, int64, error) {
	panic("StreamReader is not yet supported for mock reader")
}

func (m *MockReader) ReadRange(_ context.Context, _ string, _ uuid.UUID, _ string, _ uint64, buffer []byte, _ *CacheInfo) error {
	copy(buffer, m.Range)

	return nil
}

func (m *MockReader) TenantIndex(ctx context.Context, tenantID string) (*TenantIndex, error) {
	if m.TenantIndexFn != nil {
		return m.TenantIndexFn(ctx, tenantID)
	}

	return &TenantIndex{}, nil
}

func (m *MockReader) Shutdown() {}

// MockWriter
type MockWriter struct {
	sync.Mutex
	IndexMeta          map[string][]*BlockMeta
	IndexCompactedMeta map[string][]*CompactedBlockMeta
}

func (m *MockWriter) Write(context.Context, string, uuid.UUID, string, []byte, *CacheInfo) error {
	return nil
}

func (m *MockWriter) StreamWriter(context.Context, string, uuid.UUID, string, io.Reader, int64) error {
	return nil
}

func (m *MockWriter) WriteBlockMeta(context.Context, *BlockMeta) error {
	return nil
}

func (m *MockWriter) Append(context.Context, string, uuid.UUID, string, AppendTracker, []byte) (AppendTracker, error) {
	return nil, nil
}

func (m *MockWriter) CloseAppend(context.Context, AppendTracker) error {
	return nil
}

func (m *MockWriter) WriteTenantIndex(_ context.Context, tenantID string, meta []*BlockMeta, compactedMeta []*CompactedBlockMeta) error {
	m.Lock()
	defer m.Unlock()

	if m.IndexMeta == nil {
		m.IndexMeta = make(map[string][]*BlockMeta)
	}
	if m.IndexCompactedMeta == nil {
		m.IndexCompactedMeta = make(map[string][]*CompactedBlockMeta)
	}
	m.IndexMeta[tenantID] = meta
	m.IndexCompactedMeta[tenantID] = compactedMeta
	return nil
}

type MockBlocklist struct {
	MetasFn          func(tenantID string) []*BlockMeta
	CompactedMetasFn func(tenantID string) []*CompactedBlockMeta
}

func (m *MockBlocklist) Metas(tenantID string) []*BlockMeta {
	return m.MetasFn(tenantID)
}

func (m *MockBlocklist) CompactedMetas(tenantID string) []*CompactedBlockMeta {
	return m.CompactedMetasFn(tenantID)
}
