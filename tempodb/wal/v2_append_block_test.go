package wal

import (
	"context"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/gogo/protobuf/proto"
	"github.com/google/uuid"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/grafana/tempo/tempodb/backend"
	v2 "github.com/grafana/tempo/tempodb/encoding/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFullFilename(t *testing.T) {
	tests := []struct {
		name     string
		b        *v2AppendBlock
		expected string
	}{
		{
			name: "legacy",
			b: &v2AppendBlock{
				meta:     backend.NewBlockMeta("foo", uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"), "v0", backend.EncNone, ""),
				filepath: "/blerg",
			},
			expected: "/blerg/123e4567-e89b-12d3-a456-426614174000:foo",
		},
		{
			name: "ez-mode",
			b: &v2AppendBlock{
				meta:     backend.NewBlockMeta("foo", uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"), "v1", backend.EncNone, ""),
				filepath: "/blerg",
			},
			expected: "/blerg/123e4567-e89b-12d3-a456-426614174000:foo:v1:none",
		},
		{
			name: "nopath",
			b: &v2AppendBlock{
				meta:     backend.NewBlockMeta("foo", uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"), "v1", backend.EncNone, ""),
				filepath: "",
			},
			expected: "123e4567-e89b-12d3-a456-426614174000:foo:v1:none",
		},
		{
			name: "gzip",
			b: &v2AppendBlock{
				meta:     backend.NewBlockMeta("foo", uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"), "v2", backend.EncGZIP, ""),
				filepath: "",
			},
			expected: "123e4567-e89b-12d3-a456-426614174000:foo:v2:gzip",
		},
		{
			name: "lz41M",
			b: &v2AppendBlock{
				meta:     backend.NewBlockMeta("foo", uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"), "v2", backend.EncLZ4_1M, ""),
				filepath: "",
			},
			expected: "123e4567-e89b-12d3-a456-426614174000:foo:v2:lz4-1M",
		},
		{
			name: "lz4256k",
			b: &v2AppendBlock{
				meta:     backend.NewBlockMeta("foo", uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"), "v2", backend.EncLZ4_256k, ""),
				filepath: "",
			},
			expected: "123e4567-e89b-12d3-a456-426614174000:foo:v2:lz4-256k",
		},
		{
			name: "lz4M",
			b: &v2AppendBlock{
				meta:     backend.NewBlockMeta("foo", uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"), "v2", backend.EncLZ4_4M, ""),
				filepath: "",
			},
			expected: "123e4567-e89b-12d3-a456-426614174000:foo:v2:lz4",
		},
		{
			name: "lz64k",
			b: &v2AppendBlock{
				meta:     backend.NewBlockMeta("foo", uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"), "v2", backend.EncLZ4_64k, ""),
				filepath: "",
			},
			expected: "123e4567-e89b-12d3-a456-426614174000:foo:v2:lz4-64k",
		},
		{
			name: "snappy",
			b: &v2AppendBlock{
				meta:     backend.NewBlockMeta("foo", uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"), "v2", backend.EncSnappy, ""),
				filepath: "",
			},
			expected: "123e4567-e89b-12d3-a456-426614174000:foo:v2:snappy",
		},
		{
			name: "zstd",
			b: &v2AppendBlock{
				meta:     backend.NewBlockMeta("foo", uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"), "v2", backend.EncZstd, ""),
				filepath: "",
			},
			expected: "123e4567-e89b-12d3-a456-426614174000:foo:v2:zstd",
		},
		{
			name: "data encoding",
			b: &v2AppendBlock{
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

func TestAdjustTimeRangeForSlack(t *testing.T) {
	a := &v2AppendBlock{
		meta: &backend.BlockMeta{
			TenantID: "test",
		},
		ingestionSlack: 2 * time.Minute,
	}

	// test happy path
	start := uint32(time.Now().Unix())
	end := uint32(time.Now().Unix())
	actualStart, actualEnd := a.adjustTimeRangeForSlack(start, end, 0)
	assert.Equal(t, start, actualStart)
	assert.Equal(t, end, actualEnd)

	// test start out of range
	now := uint32(time.Now().Unix())
	start = uint32(time.Now().Add(-time.Hour).Unix())
	end = uint32(time.Now().Unix())
	actualStart, actualEnd = a.adjustTimeRangeForSlack(start, end, 0)
	assert.Equal(t, now, actualStart)
	assert.Equal(t, end, actualEnd)

	// test end out of range
	now = uint32(time.Now().Unix())
	start = uint32(time.Now().Unix())
	end = uint32(time.Now().Add(time.Hour).Unix())
	actualStart, actualEnd = a.adjustTimeRangeForSlack(start, end, 0)
	assert.Equal(t, start, actualStart)
	assert.Equal(t, now, actualEnd)

	// test additional start slack honored
	start = uint32(time.Now().Add(-time.Hour).Unix())
	end = uint32(time.Now().Unix())
	actualStart, actualEnd = a.adjustTimeRangeForSlack(start, end, time.Hour)
	assert.Equal(t, start, actualStart)
	assert.Equal(t, end, actualEnd)
}

func TestErrorConditions(t *testing.T) {
	tempDir := t.TempDir()

	wal, err := New(&Config{
		Filepath: tempDir,
		Encoding: backend.EncGZIP,
	})
	require.NoError(t, err, "unexpected error creating temp wal")

	blockID := uuid.New()

	// create partially corrupt block
	block, err := wal.NewBlock(blockID, testTenantID, "")
	require.NoError(t, err, "unexpected error creating block")

	objects := 10
	for i := 0; i < objects; i++ {
		id := make([]byte, 16)
		rand.Read(id)
		obj := test.MakeTrace(rand.Int()%10, id)
		bObj, err := proto.Marshal(obj)
		require.NoError(t, err)

		err = block.Append(id, bObj, 0, 0)
		require.NoError(t, err, "unexpected error writing req")
	}
	v2Block := block.(*v2AppendBlock)

	appendFile, err := os.OpenFile(v2Block.fullFilename(), os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	require.NoError(t, err)
	_, err = appendFile.Write([]byte{0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01})
	require.NoError(t, err)
	err = appendFile.Close()
	require.NoError(t, err)

	// create unparseable filename
	err = os.WriteFile(filepath.Join(tempDir, "fe0b83eb-a86b-4b6c-9a74-dc272cd5700e:tenant:v2:notanencoding"), []byte{}, 0644)
	require.NoError(t, err)

	// create empty block
	err = os.WriteFile(filepath.Join(tempDir, "fe0b83eb-a86b-4b6c-9a74-dc272cd5700e:blerg:v2:gzip"), []byte{}, 0644)
	require.NoError(t, err)

	blocks, err := wal.RescanBlocks(func([]byte, string) (uint32, uint32, error) {
		return 0, 0, nil
	}, 0, log.NewNopLogger())
	require.NoError(t, err, "unexpected error getting blocks")
	require.Len(t, blocks, 1)

	// confirm block has been removed
	require.NoFileExists(t, filepath.Join(tempDir, "fe0b83eb-a86b-4b6c-9a74-dc272cd5700e:tenant:v2:gzip"))
	require.NoFileExists(t, filepath.Join(tempDir, "fe0b83eb-a86b-4b6c-9a74-dc272cd5700e:blerg:v2:gzip"))
}

func TestAppend(t *testing.T) {
	wal, err := New(&Config{
		Filepath: t.TempDir(),
	})
	require.NoError(t, err, "unexpected error creating temp wal")

	blockID := uuid.New()

	block, err := wal.NewBlock(blockID, testTenantID, "")
	require.NoError(t, err, "unexpected error creating block")

	v2block := block.(*v2AppendBlock)

	numMsgs := 100
	reqs := make([]*tempopb.Trace, 0, numMsgs)
	for i := 0; i < numMsgs; i++ {
		req := test.MakeTrace(rand.Int()%1000, []byte{0x01})
		reqs = append(reqs, req)
		bReq, err := proto.Marshal(req)
		require.NoError(t, err)
		err = block.Append([]byte{0x01}, bReq, 0, 0)
		require.NoError(t, err, "unexpected error writing req")
	}

	records := v2block.appender.Records()
	file, err := v2block.file()
	require.NoError(t, err)

	dataReader, err := v2.NewDataReader(backend.NewContextReaderWithAllReader(file), backend.EncNone)
	require.NoError(t, err)
	iterator := v2.NewRecordIterator(records, dataReader, v2.NewObjectReaderWriter())
	defer iterator.Close()
	i := 0

	for {
		_, bytesObject, err := iterator.Next(context.Background())
		if err == io.EOF {
			break
		}
		require.NoError(t, err)

		req := &tempopb.Trace{}
		err = proto.Unmarshal(bytesObject, req)
		require.NoError(t, err)

		require.True(t, proto.Equal(req, reqs[i]))
		i++
	}
	require.Equal(t, numMsgs, i)
}
