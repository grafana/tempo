package backend

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func BenchmarkIndexMarshal(b *testing.B) {
	idx := &TenantIndex{
		Meta: []*BlockMeta{
			NewBlockMeta("test", uuid.New(), "v1", EncGZIP, "adsf"),
			NewBlockMeta("test", uuid.New(), "v2", EncNone, "adsf"),
			NewBlockMeta("test", uuid.New(), "v3", EncLZ4_4M, "adsf"),
		},
		CompactedMeta: []*CompactedBlockMeta{
			{
				BlockMeta:     *NewBlockMeta("test", uuid.New(), "v1", EncGZIP, "adsf"),
				CompactedTime: time.Now(),
			},
			{
				BlockMeta:     *NewBlockMeta("test", uuid.New(), "v1", EncZstd, "adsf"),
				CompactedTime: time.Now(),
			},
			{
				BlockMeta:     *NewBlockMeta("test", uuid.New(), "v1", EncSnappy, "adsf"),
				CompactedTime: time.Now(),
			},
		},
	}

	for i := range idx.Meta {
		idx.Meta[i].DedicatedColumns = DedicatedColumns{
			{Scope: "resource", Name: "namespace", Type: "string"},
			{Scope: "span", Name: "http.method", Type: "string"},
			{Scope: "span", Name: "namespace", Type: "string"},
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = idx.marshal()
	}
}

func BenchmarkIndexUnmarshal(b *testing.B) {
	idx := &TenantIndex{
		Meta: []*BlockMeta{
			NewBlockMeta("test", uuid.New(), "v1", EncGZIP, "adsf"),
			NewBlockMeta("test", uuid.New(), "v2", EncNone, "adsf"),
			NewBlockMeta("test", uuid.New(), "v3", EncLZ4_4M, "adsf"),
		},
		CompactedMeta: []*CompactedBlockMeta{
			{
				BlockMeta:     *NewBlockMeta("test", uuid.New(), "v1", EncGZIP, "adsf"),
				CompactedTime: time.Now(),
			},
			{
				BlockMeta:     *NewBlockMeta("test", uuid.New(), "v1", EncZstd, "adsf"),
				CompactedTime: time.Now(),
			},
			{
				BlockMeta:     *NewBlockMeta("test", uuid.New(), "v1", EncSnappy, "adsf"),
				CompactedTime: time.Now(),
			},
		},
	}

	for i := range idx.Meta {
		idx.Meta[i].DedicatedColumns = DedicatedColumns{
			{Scope: "resource", Name: "namespace", Type: "string"},
			{Scope: "span", Name: "http.method", Type: "string"},
			{Scope: "span", Name: "namespace", Type: "string"},
		}
	}

	buf, err := idx.marshal()
	require.NoError(b, err)

	unIdx := &TenantIndex{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = unIdx.unmarshal(buf)
	}
}

func BenchmarkIndexUnmarshalPb(b *testing.B) {
	idx := &TenantIndex{
		Meta: []*BlockMeta{
			NewBlockMeta("test", uuid.New(), "v1", EncGZIP, "adsf"),
			NewBlockMeta("test", uuid.New(), "v2", EncNone, "adsf"),
			NewBlockMeta("test", uuid.New(), "v3", EncLZ4_4M, "adsf"),
		},
		CompactedMeta: []*CompactedBlockMeta{
			{
				BlockMeta:     *NewBlockMeta("test", uuid.New(), "v1", EncGZIP, "adsf"),
				CompactedTime: time.Now(),
			},
			{
				BlockMeta:     *NewBlockMeta("test", uuid.New(), "v1", EncZstd, "adsf"),
				CompactedTime: time.Now(),
			},
			{
				BlockMeta:     *NewBlockMeta("test", uuid.New(), "v1", EncSnappy, "adsf"),
				CompactedTime: time.Now(),
			},
		},
	}

	for i := range idx.Meta {
		idx.Meta[i].DedicatedColumns = DedicatedColumns{
			{Scope: "resource", Name: "namespace", Type: "string"},
			{Scope: "span", Name: "http.method", Type: "string"},
			{Scope: "span", Name: "namespace", Type: "string"},
		}
	}

	buf, err := idx.marshalPb()
	require.NoError(b, err)

	unIdx := &TenantIndex{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = unIdx.unmarshalPb(buf)
	}
}
