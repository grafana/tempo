package backend

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	tenantID      = "test"
	storagePrefix = "test/prefix"
)

// todo: add tests that check the appropriate keypath is passed
func TestWriter(t *testing.T) {
	m := &MockRawWriter{}
	w := NewWriter(m)
	ctx := context.Background()

	expected := []byte{0x01, 0x02, 0x03, 0x04}

	u := uuid.New()
	err := w.Write(ctx, "test", u, "test", expected, nil)
	assert.NoError(t, err)
	assert.Equal(t, expected, m.writeBuffer["test/"+u.String()+"/test"])

	_, err = w.Append(ctx, "test", uuid.New(), "test", nil, expected)
	assert.NoError(t, err)
	assert.Equal(t, expected, m.appendBuffer)

	err = w.CloseAppend(ctx, nil)
	assert.NoError(t, err)
	assert.True(t, m.closeAppendCalled)

	u = uuid.New()
	expectedPath := filepath.Join("test", u.String(), MetaName)
	meta := NewBlockMeta("test", u, "blerg", EncGZIP, "glarg")
	jsonBytes, err := json.Marshal(meta)
	assert.NoError(t, err)
	assert.NoError(t, err)

	// Write the block meta to the backend and validate the payloads.
	err = w.WriteBlockMeta(ctx, meta)
	assert.NoError(t, err)
	assert.Equal(t, jsonBytes, m.writeBuffer[expectedPath])

	tenantIndexPath := filepath.Join("test", TenantIndexName)
	tenantIndexPathPb := filepath.Join("test", TenantIndexNamePb)
	// Write the tenant index to the backend and validate the payloads.
	err = w.WriteTenantIndex(ctx, "test", []*BlockMeta{meta}, nil)
	assert.NoError(t, err)

	// proto
	idxP := &TenantIndex{}
	err = idxP.unmarshalPb(m.writeBuffer[tenantIndexPathPb])
	assert.NoError(t, err)

	assert.Equal(t, []*BlockMeta{meta}, idxP.Meta)
	assert.True(t, cmp.Equal([]*BlockMeta{meta}, idxP.Meta))                  // using cmp.Equal to compare json datetimes
	assert.True(t, cmp.Equal([]*CompactedBlockMeta(nil), idxP.CompactedMeta)) // using cmp.Equal to compare json datetimes

	// json
	idxJ := &TenantIndex{}
	err = idxJ.unmarshal(m.writeBuffer[tenantIndexPath])
	assert.NoError(t, err)

	assert.Equal(t, []*BlockMeta{meta}, idxJ.Meta)
	assert.True(t, cmp.Equal([]*BlockMeta{meta}, idxJ.Meta))                  // using cmp.Equal to compare json datetimes
	assert.True(t, cmp.Equal([]*CompactedBlockMeta(nil), idxJ.CompactedMeta)) // using cmp.Equal to compare json datetimes

	// When there are no blocks, the tenant index should be deleted
	assert.Equal(t, map[string]map[string]int(nil), w.(*writer).w.(*MockRawWriter).deleteCalls)

	err = w.WriteTenantIndex(ctx, "test", nil, nil)
	assert.NoError(t, err)

	expectedDeleteMap := map[string]map[string]int{TenantIndexName: {"test": 1}, TenantIndexNamePb: {"test": 1}}
	assert.Equal(t, expectedDeleteMap, w.(*writer).w.(*MockRawWriter).deleteCalls)

	// When a backend returns ErrDoesNotExist, the tenant index should be deleted, but no error should be returned if the tenant index does not exist
	m = &MockRawWriter{err: ErrDoesNotExist}
	w = NewWriter(m)
	err = w.WriteTenantIndex(ctx, "test", nil, nil)
	assert.NoError(t, err)
}

func TestReader(t *testing.T) {
	m := &MockRawReader{}
	r := NewReader(m)
	ctx := context.Background()

	expected := []byte{0x01, 0x02, 0x03, 0x04}
	m.R = expected
	actual, err := r.Read(ctx, "test", uuid.New(), "test", nil)
	assert.NoError(t, err)
	assert.Equal(t, expected, actual)

	m.Range = expected
	err = r.ReadRange(ctx, "test", uuid.New(), "test", 10, actual, nil)
	assert.NoError(t, err)
	assert.Equal(t, expected, actual)

	expectedTenants := []string{"a", "b", "c"}
	m.L = expectedTenants
	actualTenants, err := r.Tenants(ctx)
	assert.NoError(t, err)
	assert.Equal(t, expectedTenants, actualTenants)

	uuid1, uuid2, uuid3 := uuid.New(), uuid.New(), uuid.New()
	expectedBlocks := []uuid.UUID{uuid1, uuid2}
	expectedCompactedBlocks := []uuid.UUID{uuid3}

	m.BlockIDs = append(m.BlockIDs, uuid1)
	m.BlockIDs = append(m.BlockIDs, uuid2)
	m.CompactedBlockIDs = append(m.CompactedBlockIDs, uuid3)

	actualBlocks, actualCompactedBlocks, err := r.Blocks(ctx, "test")
	assert.NoError(t, err)
	assert.Equal(t, expectedBlocks, actualBlocks)
	assert.Equal(t, expectedCompactedBlocks, actualCompactedBlocks)

	// should fail b/c meta is not valid
	meta, err := r.BlockMeta(ctx, uuid.New(), "test")
	assert.Error(t, err)
	assert.Nil(t, meta)

	expectedMeta := NewBlockMeta("test", uuid.New(), "blerg", EncGZIP, "glarg")
	m.R, _ = json.Marshal(expectedMeta)
	meta, err = r.BlockMeta(ctx, uuid.New(), "test")
	assert.NoError(t, err)
	assert.Equal(t, expectedMeta, meta)

	// should fail b/c tenant index is not valid
	idx, err := r.TenantIndex(ctx, "test")
	assert.Error(t, err)
	assert.Nil(t, idx)

	expectedIdx := newTenantIndex([]*BlockMeta{expectedMeta}, nil)
	m.R, _ = expectedIdx.marshalPb()
	idx, err = r.TenantIndex(ctx, "test")
	assert.NoError(t, err)
	assert.True(t, cmp.Equal(expectedIdx, idx))
}

func TestKeyPathForBlock(t *testing.T) {
	b := uuid.New()
	tid := tenantID
	keypath := KeyPathForBlock(b, tid)

	assert.Equal(t, KeyPath([]string{tid, b.String()}), keypath)
}

func TestMetaFileName(t *testing.T) {
	// WithoutPrefix
	b := uuid.New()
	tid := tenantID
	prefix := ""
	metaFilename := MetaFileName(b, tid, prefix)

	assert.Equal(t, tid+"/"+b.String()+"/"+MetaName, metaFilename)

	// WithPrefix
	prefix = storagePrefix
	metaFilename = MetaFileName(b, tid, prefix)

	assert.Equal(t, prefix+"/"+tid+"/"+b.String()+"/"+MetaName, metaFilename)
}

func TestCompactedMetaFileName(t *testing.T) {
	// WithoutPrefix
	b := uuid.New()
	tid := tenantID
	prefix := ""
	compactedMetaFilename := CompactedMetaFileName(b, tid, prefix)

	assert.Equal(t, tid+"/"+b.String()+"/"+CompactedMetaName, compactedMetaFilename)

	// WithPrefix
	prefix = storagePrefix
	compactedMetaFilename = CompactedMetaFileName(b, tid, prefix)

	assert.Equal(t, prefix+"/"+tid+"/"+b.String()+"/"+CompactedMetaName, compactedMetaFilename)
}

func TestRootPath(t *testing.T) {
	// WithoutPrefix
	b := uuid.New()
	tid := tenantID
	prefix := ""
	rootPath := RootPath(b, tid, prefix)

	assert.Equal(t, tid+"/"+b.String(), rootPath)

	// WithPrefix
	prefix = storagePrefix
	rootPath = RootPath(b, tid, prefix)

	assert.Equal(t, prefix+"/"+tid+"/"+b.String(), rootPath)
}

func TestRoundTripMeta(t *testing.T) {
	// RoundTrip with empty DedicatedColumns
	meta := NewBlockMeta("test", uuid.New(), "blerg", EncGZIP, "glarg")
	// RoundTrip with empty DedicatedColumns
	expectedPb, err := meta.Marshal()
	assert.NoError(t, err)
	expectedPb2 := &BlockMeta{}
	err = expectedPb2.Unmarshal(expectedPb)
	assert.NoError(t, err)
	assert.Equal(t, meta, expectedPb2)

	// RoundTrip with non-empty DedicatedColumns
	meta.DedicatedColumns = DedicatedColumns{
		{Scope: "resource", Name: "namespace", Type: "string"},
		{Scope: "span", Name: "http.method", Type: "string"},
		{Scope: "span", Name: "namespace", Type: "string"},
	}

	expectedPb, err = meta.Marshal()
	assert.NoError(t, err)
	expectedPb3 := &BlockMeta{}
	err = expectedPb3.Unmarshal(expectedPb)
	assert.NoError(t, err)
	assert.Equal(t, meta, expectedPb3)

	// Round trip the json
	jsonBytes, err := json.Marshal(meta)
	assert.NoError(t, err)
	expected2 := &BlockMeta{}
	err = json.Unmarshal(jsonBytes, expected2)
	assert.NoError(t, err)
	assert.Equal(t, meta, expected2)
}

func TestTenantIndexFallback(t *testing.T) {
	var (
		mr       = &MockRawReader{}
		r        = NewReader(mr)
		mw       = &MockRawWriter{}
		w        = NewWriter(mw)
		ctx      = context.Background()
		tenantID = "test"

		u           = uuid.New()
		meta        = NewBlockMeta(tenantID, u, "blerg", EncGZIP, "glarg")
		expectedIdx = newTenantIndex([]*BlockMeta{meta}, nil)
	)

	err := w.WriteTenantIndex(ctx, tenantID, []*BlockMeta{meta}, nil)
	assert.NoError(t, err)

	mr.R, err = expectedIdx.marshal()
	assert.NoError(t, err)
	mr.ReadFn = func(_ context.Context, name string, _ KeyPath, _ *CacheInfo) (io.ReadCloser, int64, error) {
		if name == TenantIndexNamePb {
			return nil, 0, fmt.Errorf("meow: %w", ErrDoesNotExist)
		}

		return io.NopCloser(bytes.NewReader(mr.R)), int64(len(mr.R)), nil
	}

	idx, err := r.TenantIndex(ctx, tenantID)
	assert.NoError(t, err)
	assert.True(t, cmp.Equal(expectedIdx, idx))

	// Corrupt the proto to ensure we don't fall back
	mr.ReadFn = func(_ context.Context, name string, _ KeyPath, _ *CacheInfo) (io.ReadCloser, int64, error) {
		if name == TenantIndexNamePb {
			return io.NopCloser(bytes.NewReader([]byte{0x00})), int64(1), nil
		}

		return io.NopCloser(bytes.NewReader(mr.R)), int64(len(mr.R)), nil
	}

	idx, err = r.TenantIndex(ctx, tenantID)
	assert.ErrorIs(t, err, io.ErrUnexpectedEOF)
	assert.Nil(t, idx)

	// Remove all indexes and error check
	mr.ReadFn = func(_ context.Context, _ string, _ KeyPath, _ *CacheInfo) (io.ReadCloser, int64, error) {
		return nil, 0, fmt.Errorf("meow: %w", ErrDoesNotExist)
	}

	idx, err = r.TenantIndex(ctx, tenantID)
	assert.ErrorIs(t, err, ErrDoesNotExist)
	assert.Nil(t, idx)
}

func TestNoCompactFlag(t *testing.T) {
	ctx := context.Background()
	tenantID := "test-tenant"
	blockID := uuid.New()

	rawReader := &MockRawReader{}
	rawWriter := &MockRawWriter{}
	rawReader.ReadFn = func(_ context.Context, name string, keyPath KeyPath, _ *CacheInfo) (io.ReadCloser, int64, error) {
		key := strings.Join(keyPath, "/") + "/" + name
		val, ok := rawWriter.writeBuffer[key]
		if !ok {
			return nil, 0, ErrDoesNotExist
		}
		return io.NopCloser(bytes.NewReader(val)), int64(len(val)), nil
	}

	reader := NewReader(rawReader)
	writer := NewWriter(rawWriter)

	hasFlag, err := reader.HasNoCompactFlag(ctx, blockID, tenantID)
	require.NoError(t, err)
	assert.False(t, hasFlag)

	err = writer.WriteNoCompactFlag(ctx, blockID, tenantID)
	require.NoError(t, err)

	hasFlag, err = reader.HasNoCompactFlag(ctx, blockID, tenantID)
	require.NoError(t, err)
	assert.True(t, hasFlag)

	err = writer.DeleteNoCompactFlag(ctx, blockID, tenantID)
	require.NoError(t, err)
	assert.Equal(t, map[string]map[string]int{
		NoCompactFileName: {
			fmt.Sprintf("%s/%s", tenantID, blockID.String()): 1,
		},
	}, rawWriter.deleteCalls)
}
