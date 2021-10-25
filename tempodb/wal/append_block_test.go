package wal

import (
	"testing"

	"github.com/google/uuid"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/stretchr/testify/assert"
)

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
