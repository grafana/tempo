package wal

import (
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAppendBlockBuffersAndFlushes(t *testing.T) {
	// direct call to flush
	testBufferAndFlush(t, func(a *AppendBlock) {
		err := a.FlushBuffer()
		assert.NoError(t, err)
	})

	// find
	testBufferAndFlush(t, func(a *AppendBlock) {
		_, err := a.Find([]byte{0x01}, &mockCombiner{}) // 0x01 has to be in the block or it won't flush. this happens to sync with below
		assert.NoError(t, err)
	})

	// iterator
	testBufferAndFlush(t, func(a *AppendBlock) {
		_, err := a.Iterator(&mockCombiner{})
		assert.NoError(t, err)
	})
}

func testBufferAndFlush(t *testing.T, fn func(a *AppendBlock)) {
	tmpDir := t.TempDir()
	a, err := newAppendBlock(uuid.New(), testTenantID, tmpDir, backend.EncNone, "", 1000)
	require.NoError(t, err)

	err = a.Append([]byte{0x01}, []byte{0x01})
	require.NoError(t, err)

	filename := a.fullFilename()

	// file should be 0 length
	info, err := os.Stat(filename)
	require.NoError(t, err)
	assert.Equal(t, int64(0), info.Size())

	// do something and confirm the file has flusehd something is in the file
	fn(a)
	info, err = os.Stat(filename)
	require.NoError(t, err)
	assert.NotZero(t, info.Size())
}

func TestFullFilename(t *testing.T) {
	tests := []struct {
		name     string
		b        *AppendBlock
		expected string
	}{
		{
			name: "legacy",
			b: &AppendBlock{
				meta:     backend.NewBlockMeta("foo", uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"), "v0", backend.EncNone, ""),
				filepath: "/blerg",
			},
			expected: "/blerg/123e4567-e89b-12d3-a456-426614174000:foo",
		},
		{
			name: "ez-mode",
			b: &AppendBlock{
				meta:     backend.NewBlockMeta("foo", uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"), "v1", backend.EncNone, ""),
				filepath: "/blerg",
			},
			expected: "/blerg/123e4567-e89b-12d3-a456-426614174000:foo:v1:none",
		},
		{
			name: "nopath",
			b: &AppendBlock{
				meta:     backend.NewBlockMeta("foo", uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"), "v1", backend.EncNone, ""),
				filepath: "",
			},
			expected: "123e4567-e89b-12d3-a456-426614174000:foo:v1:none",
		},
		{
			name: "gzip",
			b: &AppendBlock{
				meta:     backend.NewBlockMeta("foo", uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"), "v2", backend.EncGZIP, ""),
				filepath: "",
			},
			expected: "123e4567-e89b-12d3-a456-426614174000:foo:v2:gzip",
		},
		{
			name: "lz41M",
			b: &AppendBlock{
				meta:     backend.NewBlockMeta("foo", uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"), "v2", backend.EncLZ4_1M, ""),
				filepath: "",
			},
			expected: "123e4567-e89b-12d3-a456-426614174000:foo:v2:lz4-1M",
		},
		{
			name: "lz4256k",
			b: &AppendBlock{
				meta:     backend.NewBlockMeta("foo", uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"), "v2", backend.EncLZ4_256k, ""),
				filepath: "",
			},
			expected: "123e4567-e89b-12d3-a456-426614174000:foo:v2:lz4-256k",
		},
		{
			name: "lz4M",
			b: &AppendBlock{
				meta:     backend.NewBlockMeta("foo", uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"), "v2", backend.EncLZ4_4M, ""),
				filepath: "",
			},
			expected: "123e4567-e89b-12d3-a456-426614174000:foo:v2:lz4",
		},
		{
			name: "lz64k",
			b: &AppendBlock{
				meta:     backend.NewBlockMeta("foo", uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"), "v2", backend.EncLZ4_64k, ""),
				filepath: "",
			},
			expected: "123e4567-e89b-12d3-a456-426614174000:foo:v2:lz4-64k",
		},
		{
			name: "snappy",
			b: &AppendBlock{
				meta:     backend.NewBlockMeta("foo", uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"), "v2", backend.EncSnappy, ""),
				filepath: "",
			},
			expected: "123e4567-e89b-12d3-a456-426614174000:foo:v2:snappy",
		},
		{
			name: "zstd",
			b: &AppendBlock{
				meta:     backend.NewBlockMeta("foo", uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"), "v2", backend.EncZstd, ""),
				filepath: "",
			},
			expected: "123e4567-e89b-12d3-a456-426614174000:foo:v2:zstd",
		},
		{
			name: "data encoding",
			b: &AppendBlock{
				meta:     backend.NewBlockMeta("foo", uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"), "v1", backend.EncNone, "dataencoding"),
				filepath: "/blerg",
			},
			expected: "/blerg/123e4567-e89b-12d3-a456-426614174000:foo:v1:none:dataencoding",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual := tc.b.fullFilename()
			assert.Equal(t, tc.expected, actual)
		})
	}
}
