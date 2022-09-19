package wal

import (
	"bytes"
	"context"
	"io"
	"math/rand"
	"os"
	"path"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/golang/protobuf/proto" //nolint:all
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/pkg/util/test"
	"github.com/grafana/tempo/tempodb/backend"
)

const (
	testTenantID = "fake"
)

func TestCompletedDirIsRemoved(t *testing.T) {
	// Create /completed/testfile and verify it is removed.
	tempDir := t.TempDir()

	err := os.MkdirAll(path.Join(tempDir, completedDir), os.ModePerm)
	require.NoError(t, err, "unexpected error creating completedDir")

	_, err = os.Create(path.Join(tempDir, completedDir, "testfile"))
	require.NoError(t, err, "unexpected error creating testfile")

	_, err = New(&Config{
		Filepath: tempDir,
	})
	require.NoError(t, err, "unexpected error creating temp wal")

	_, err = os.Stat(path.Join(tempDir, completedDir))
	require.Error(t, err, "completedDir should not exist")
}

func TestAppendBlockStartEnd(t *testing.T) {
	wal, err := New(&Config{
		Filepath:       t.TempDir(),
		Encoding:       backend.EncNone,
		IngestionSlack: 2 * time.Minute,
	})
	require.NoError(t, err, "unexpected error creating temp wal")

	blockID := uuid.New()
	block, err := wal.NewBlock(blockID, testTenantID, "")
	require.NoError(t, err, "unexpected error creating block")

	// create a new block and confirm start/end times are correct
	blockStart := uint32(time.Now().Unix())
	blockEnd := uint32(time.Now().Add(time.Minute).Unix())

	for i := 0; i < 10; i++ {
		bytes := make([]byte, 16)
		rand.Read(bytes)

		err = block.Append(bytes, bytes, blockStart, blockEnd)
		require.NoError(t, err, "unexpected error writing req")
	}

	require.Equal(t, blockStart, uint32(block.Meta().StartTime.Unix()))
	require.Equal(t, blockEnd, uint32(block.Meta().EndTime.Unix()))

	// rescan the block and make sure that start/end times are correct
	blockStart = uint32(time.Now().Add(-time.Hour).Unix())
	blockEnd = uint32(time.Now().Unix())

	blocks, err := wal.RescanBlocks(func([]byte, string) (uint32, uint32, error) {
		return blockStart, blockEnd, nil
	}, time.Hour, log.NewNopLogger())
	require.NoError(t, err, "unexpected error getting blocks")
	require.Len(t, blocks, 1)

	require.Equal(t, blockStart, uint32(blocks[0].Meta().StartTime.Unix()))
	require.Equal(t, blockEnd, uint32(blocks[0].Meta().EndTime.Unix()))
}

func TestAppendReplayFind(t *testing.T) {
	for _, e := range backend.SupportedEncoding {
		t.Run(e.String(), func(t *testing.T) {
			testAppendReplayFind(t, backend.EncZstd)
		})
	}
}

func testAppendReplayFind(t *testing.T, e backend.Encoding) {
	wal, err := New(&Config{
		Filepath: t.TempDir(),
		Encoding: e,
	})
	require.NoError(t, err, "unexpected error creating temp wal")

	blockID := uuid.New()

	block, err := wal.NewBlock(blockID, testTenantID, "")
	require.NoError(t, err, "unexpected error creating block")

	objects := 1000
	objs := make([][]byte, 0, objects)
	ids := make([][]byte, 0, objects)
	for i := 0; i < objects; i++ {
		id := make([]byte, 16)
		rand.Read(id)
		obj := test.MakeTrace(rand.Int()%10, id)
		ids = append(ids, id)
		bObj, err := proto.Marshal(obj)
		require.NoError(t, err)
		objs = append(objs, bObj)

		err = block.Append(id, bObj, 0, 0)
		require.NoError(t, err, "unexpected error writing req")
	}

	for i, id := range ids {
		obj, err := block.Find(id)
		require.NoError(t, err)
		require.Equal(t, objs[i], obj)
	}

	blocks, err := wal.RescanBlocks(func([]byte, string) (uint32, uint32, error) {
		return 0, 0, nil
	}, 0, log.NewNopLogger())
	require.NoError(t, err, "unexpected error getting blocks")
	require.Len(t, blocks, 1)

	iterator, err := blocks[0].Iterator()
	require.NoError(t, err)
	defer iterator.Close()

	// append block find
	for i, id := range ids {
		obj, err := blocks[0].Find(id)
		require.NoError(t, err)
		require.Equal(t, objs[i], obj)
	}

	i := 0
	for {
		id, obj, err := iterator.Next(context.Background())
		if err == io.EOF {
			break
		} else {
			require.NoError(t, err)
		}

		found := false
		j := 0
		for ; j < len(ids); j++ {
			if bytes.Equal(ids[j], id) {
				found = true
				break
			}
		}

		require.True(t, found)
		require.Equal(t, objs[j], obj)
		require.Equal(t, ids[j], []byte(id))
		i++
	}

	require.Equal(t, objects, i)

	err = blocks[0].Clear()
	require.NoError(t, err)
}

func TestParseFilename(t *testing.T) {
	tests := []struct {
		name                 string
		filename             string
		expectUUID           uuid.UUID
		expectTenant         string
		expectedVersion      string
		expectedEncoding     backend.Encoding
		expectedDataEncoding string
		expectError          bool
	}{
		{
			name:                 "version, enc snappy and dataencoding",
			filename:             "123e4567-e89b-12d3-a456-426614174000:foo:v2:snappy:dataencoding",
			expectUUID:           uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"),
			expectTenant:         "foo",
			expectedVersion:      "v2",
			expectedEncoding:     backend.EncSnappy,
			expectedDataEncoding: "dataencoding",
		},
		{
			name:                 "version, enc none and dataencoding",
			filename:             "123e4567-e89b-12d3-a456-426614174000:foo:v2:none:dataencoding",
			expectUUID:           uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"),
			expectTenant:         "foo",
			expectedVersion:      "v2",
			expectedEncoding:     backend.EncNone,
			expectedDataEncoding: "dataencoding",
		},
		{
			name:                 "empty dataencoding",
			filename:             "123e4567-e89b-12d3-a456-426614174000:foo:v2:snappy",
			expectUUID:           uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"),
			expectTenant:         "foo",
			expectedVersion:      "v2",
			expectedEncoding:     backend.EncSnappy,
			expectedDataEncoding: "",
		},
		{
			name:                 "empty dataencoding with semicolon",
			filename:             "123e4567-e89b-12d3-a456-426614174000:foo:v2:snappy:",
			expectUUID:           uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"),
			expectTenant:         "foo",
			expectedVersion:      "v2",
			expectedEncoding:     backend.EncSnappy,
			expectedDataEncoding: "",
		},
		{
			name:        "path fails",
			filename:    "/blerg/123e4567-e89b-12d3-a456-426614174000:foo",
			expectError: true,
		},
		{
			name:        "no :",
			filename:    "123e4567-e89b-12d3-a456-426614174000",
			expectError: true,
		},
		{
			name:        "empty string",
			filename:    "",
			expectError: true,
		},
		{
			name:        "bad uuid",
			filename:    "123e4:foo",
			expectError: true,
		},
		{
			name:        "no tenant",
			filename:    "123e4567-e89b-12d3-a456-426614174000:",
			expectError: true,
		},
		{
			name:        "no version",
			filename:    "123e4567-e89b-12d3-a456-426614174000:test::none",
			expectError: true,
		},
		{
			name:        "wrong splits - 6",
			filename:    "123e4567-e89b-12d3-a456-426614174000:test:test:test:test:test",
			expectError: true,
		},
		{
			name:        "wrong splits - 3",
			filename:    "123e4567-e89b-12d3-a456-426614174000:test:test",
			expectError: true,
		},
		{
			name:        "wrong splits - 1",
			filename:    "123e4567-e89b-12d3-a456-426614174000",
			expectError: true,
		},
		{
			name:        "bad encoding",
			filename:    "123e4567-e89b-12d3-a456-426614174000:test:v1:asdf",
			expectError: true,
		},
		{
			name:        "ez-mode old format",
			filename:    "123e4567-e89b-12d3-a456-426614174000:foo",
			expectError: true,
		},
		{
			name:        "deprecated version",
			filename:    "123e4567-e89b-12d3-a456-426614174000:foo:v1:snappy",
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actualUUID, actualTenant, actualVersion, actualEncoding, actualDataEncoding, err := ParseFilename(tc.filename)

			if tc.expectError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.expectUUID, actualUUID)
			require.Equal(t, tc.expectTenant, actualTenant)
			require.Equal(t, tc.expectedEncoding, actualEncoding)
			require.Equal(t, tc.expectedVersion, actualVersion)
			require.Equal(t, tc.expectedDataEncoding, actualDataEncoding)
		})
	}
}

func BenchmarkWALNone(b *testing.B) {
	benchmarkWriteFindReplay(b, backend.EncNone)
}
func BenchmarkWALSnappy(b *testing.B) {
	benchmarkWriteFindReplay(b, backend.EncSnappy)
}
func BenchmarkWALLZ4(b *testing.B) {
	benchmarkWriteFindReplay(b, backend.EncLZ4_1M)
}
func BenchmarkWALGZIP(b *testing.B) {
	benchmarkWriteFindReplay(b, backend.EncGZIP)
}
func BenchmarkWALZSTD(b *testing.B) {
	benchmarkWriteFindReplay(b, backend.EncZstd)
}

func BenchmarkWALS2(b *testing.B) {
	benchmarkWriteFindReplay(b, backend.EncS2)
}

func benchmarkWriteFindReplay(b *testing.B, encoding backend.Encoding) {
	objects := 1000
	objs := make([][]byte, 0, objects)
	ids := make([][]byte, 0, objects)
	for i := 0; i < objects; i++ {
		id := make([]byte, 16)
		rand.Read(id)
		obj := make([]byte, rand.Intn(100)+1)
		rand.Read(obj)
		ids = append(ids, id)
		objs = append(objs, obj)
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		wal, _ := New(&Config{
			Filepath: b.TempDir(),
			Encoding: encoding,
		})

		blockID := uuid.New()
		block, err := wal.NewBlock(blockID, testTenantID, "")
		require.NoError(b, err)

		// write
		for j, obj := range objs {
			err := block.Append(ids[j], obj, 0, 0)
			require.NoError(b, err)
		}

		// find
		for _, id := range ids {
			_, err := block.Find(id)
			require.NoError(b, err)
		}

		// replay
		_, err = wal.RescanBlocks(func([]byte, string) (uint32, uint32, error) {
			return 0, 0, nil
		}, 0, log.NewNopLogger())
		require.NoError(b, err)
	}
}
