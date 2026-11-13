package backend

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func v2TestIndex(t *testing.T) (*TenantIndex, *BlockMeta) {
	t.Helper()
	meta := richBlockMeta(0, DedicatedColumns{{Scope: DedicatedColumnScopeSpan, Name: "a", Type: DedicatedColumnTypeString}})
	meta.TenantID = tenantID
	return &TenantIndex{CreatedAt: time.Unix(1700000000, 0), Meta: []*BlockMeta{meta}}, meta
}

func TestWriterTenantIndexV2(t *testing.T) {
	ctx := context.Background()
	_, meta := v2TestIndex(t)
	v2Path := filepath.Join(tenantID, TenantIndexNameV2)

	t.Run("always writes it alongside proto and json", func(t *testing.T) {
		m := &MockRawWriter{}
		w := NewWriter(m)
		require.NoError(t, w.WriteTenantIndex(ctx, tenantID, []*BlockMeta{meta}, nil, nil))

		require.Contains(t, m.writeBuffer, filepath.Join(tenantID, TenantIndexName))
		require.Contains(t, m.writeBuffer, filepath.Join(tenantID, TenantIndexNamePb))

		got := &TenantIndex{}
		require.NoError(t, got.unmarshalV2Zstd(m.writeBuffer[v2Path]))
		require.Len(t, got.Meta, 1)
		requireBlockMetaEqual(t, meta, got.Meta[0])
	})

	t.Run("empty tenant deletes it", func(t *testing.T) {
		m := &MockRawWriter{}
		require.NoError(t, NewWriter(m).WriteTenantIndex(ctx, tenantID, nil, nil, nil))
		require.Equal(t, map[string]int{tenantID: 1}, m.deleteCalls[TenantIndexNameV2])
	})

	t.Run("missing on delete is not an error", func(t *testing.T) {
		m := &MockRawWriter{err: ErrDoesNotExist}
		require.NoError(t, NewWriter(m).WriteTenantIndex(ctx, tenantID, nil, nil, nil))
	})
}

func TestReaderTenantIndexV2(t *testing.T) {
	ctx := context.Background()
	idx, _ := v2TestIndex(t)

	v2Bytes, err := idx.marshalV2Zstd()
	require.NoError(t, err)
	pb, err := idx.marshalPb()
	require.NoError(t, err)

	// a distinguishable proto index so tests can tell which one was read
	pbIdx := &TenantIndex{CreatedAt: time.Unix(1700000001, 0)}
	pbOnly, err := pbIdx.marshalPb()
	require.NoError(t, err)

	type fakeReader struct {
		mu    sync.Mutex
		reads []string
		files map[string][]byte
	}
	newReader := func(files map[string][]byte, opts ...Option) (Reader, *fakeReader) {
		f := &fakeReader{files: files}
		r := &MockRawReader{ReadFn: func(_ context.Context, name string, _ KeyPath, _ *CacheInfo) (io.ReadCloser, int64, error) {
			f.mu.Lock()
			defer f.mu.Unlock()
			f.reads = append(f.reads, name)
			b, ok := f.files[name]
			if !ok {
				return nil, 0, fmt.Errorf("nope: %w", ErrDoesNotExist)
			}
			return io.NopCloser(bytes.NewReader(b)), int64(len(b)), nil
		}}
		return NewReader(r, opts...), f
	}

	t.Run("enabled prefers V2", func(t *testing.T) {
		r, f := newReader(map[string][]byte{TenantIndexNameV2: v2Bytes, TenantIndexNamePb: pbOnly}, WithTenantIndexV2(true))
		got, err := r.TenantIndex(ctx, tenantID)
		require.NoError(t, err)
		require.True(t, idx.CreatedAt.Equal(got.CreatedAt))
		require.Len(t, got.Meta, 1)
		require.Equal(t, []string{TenantIndexNameV2}, f.reads)
	})

	t.Run("enabled falls back to proto when missing", func(t *testing.T) {
		r, f := newReader(map[string][]byte{TenantIndexNamePb: pb}, WithTenantIndexV2(true))
		got, err := r.TenantIndex(ctx, tenantID)
		require.NoError(t, err)
		require.Len(t, got.Meta, 1)
		require.Equal(t, []string{TenantIndexNameV2, TenantIndexNamePb}, f.reads)
	})

	t.Run("enabled does not fall back on a corrupt V2 index", func(t *testing.T) {
		r, _ := newReader(map[string][]byte{TenantIndexNameV2: {0x00}, TenantIndexNamePb: pb}, WithTenantIndexV2(true))
		got, err := r.TenantIndex(ctx, tenantID)
		require.Error(t, err)
		require.Nil(t, got)
	})

	t.Run("disabled never reads V2", func(t *testing.T) {
		r, f := newReader(map[string][]byte{TenantIndexNameV2: v2Bytes, TenantIndexNamePb: pbOnly})
		got, err := r.TenantIndex(ctx, tenantID)
		require.NoError(t, err)
		require.Empty(t, got.Meta)
		require.Equal(t, []string{TenantIndexNamePb}, f.reads)
	})

	t.Run("round trip through writer and reader", func(t *testing.T) {
		mw := &MockRawWriter{}
		w := NewWriter(mw)
		meta := NewBlockMeta(tenantID, uuid.New(), "vParquet5")
		require.NoError(t, w.WriteTenantIndex(ctx, tenantID, []*BlockMeta{meta}, nil, []UUID{meta.BlockID}))

		files := map[string][]byte{}
		for path, b := range mw.writeBuffer {
			files[filepath.Base(path)] = b
		}
		r, f := newReader(files, WithTenantIndexV2(true))
		got, err := r.TenantIndex(ctx, tenantID)
		require.NoError(t, err)
		require.Len(t, got.Meta, 1)
		require.Equal(t, meta.BlockID, got.Meta[0].BlockID)
		require.Equal(t, []UUID{meta.BlockID}, got.NoCompact)
		require.Equal(t, []string{TenantIndexNameV2}, f.reads)
	})
}
