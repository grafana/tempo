package backend

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIndexMarshalUnmarshal(t *testing.T) {
	tests := []struct {
		idx *TenantIndex
	}{
		{
			idx: &TenantIndex{},
		},
		{
			idx: &TenantIndex{
				CreatedAt: time.Now(),
				Meta: []*BlockMeta{
					NewBlockMeta("test", uuid.New(), "v1", EncGZIP, "adsf"),
					NewBlockMeta("test", uuid.New(), "v2", EncNone, "adsf"),
					NewBlockMeta("test", uuid.New(), "v3", EncLZ4_4M, "adsf"),
				},
			},
		},
		{
			idx: &TenantIndex{
				CreatedAt: time.Now(),
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
			},
		},
		{
			idx: &TenantIndex{
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
			},
		},
	}

	for _, tc := range tests {
		// json
		buff, err := tc.idx.marshal()
		require.NoError(t, err)

		actual := &TenantIndex{}
		err = actual.unmarshal(buff)
		require.NoError(t, err)

		// cmp.Equal used due to time marshalling: https://github.com/stretchr/testify/issues/502
		assert.True(t, cmp.Equal(tc.idx, actual))
	}
}

func TestIndexUnmarshalErrors(t *testing.T) {
	test := &TenantIndex{}
	err := test.unmarshal([]byte("bad data"))
	assert.Error(t, err)
}
