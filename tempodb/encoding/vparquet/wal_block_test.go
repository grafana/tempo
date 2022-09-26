package vparquet

import (
	"testing"

	"github.com/google/uuid"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Note: Standard wal block functionality (appending, searching, finding, etc.) is tested with all other wal blocks
//  in /tempodb/wal/wal_test.go

func TestFullFilename(t *testing.T) {
	tests := []struct {
		name     string
		b        *walBlock
		expected string
	}{
		{
			name: "basic",
			b: &walBlock{
				meta: backend.NewBlockMeta("foo", uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"), VersionString, backend.EncNone, ""),
				path: "/blerg",
			},
			expected: "/blerg/123e4567-e89b-12d3-a456-426614174000+foo+vParquet",
		},
		{
			name: "no path",
			b: &walBlock{
				meta: backend.NewBlockMeta("foo", uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"), VersionString, backend.EncNone, ""),
				path: "",
			},
			expected: "123e4567-e89b-12d3-a456-426614174000+foo+vParquet",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual := tc.b.walPath()
			assert.Equal(t, tc.expected, actual)
		})
	}
}

// jpe - restore this when what partial failure conditions exist
// func TestPartialBlock(t *testing.T) {
// 	blockID := uuid.New()
// 	block, err := createWALBlock(blockID, testTenantID, t.TempDir(), backend.EncSnappy, "v2", 0)
// 	require.NoError(t, err, "unexpected error creating block")

// 	enc := model.MustNewSegmentDecoder(model.CurrentEncoding)
// 	dec := model.MustNewObjectDecoder(model.CurrentEncoding)

// 	numMsgs := 100
// 	reqs := make([]*tempopb.Trace, 0, numMsgs)
// 	for i := 0; i < numMsgs; i++ {
// 		id := make([]byte, 4)
// 		binary.LittleEndian.PutUint32(id, uint32(i)) // using i for the id b/c the iterator below requires a sorted ascending list of ids

// 		id = test.ValidTraceID(id)
// 		req := test.MakeTrace(rand.Intn(10), id)
// 		reqs = append(reqs, req)

// 		b1, err := enc.PrepareForWrite(req, 0, 0)
// 		require.NoError(t, err)

// 		b2, err := enc.ToObject([][]byte{b1})
// 		require.NoError(t, err)

// 		err = block.Append(id, b2, 0, 0)
// 		require.NoError(t, err)
// 	}

// 	// append garbage data
// 	v2Block := block.(*walBlock)
// 	garbo := make([]byte, 100)
// 	_, err = rand.Read(garbo)
// 	require.NoError(t, err)

// 	appendFile, err := os.OpenFile(v2Block.fullFilename(), os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
// 	require.NoError(t, err)
// 	_, err = appendFile.Write(garbo)
// 	require.NoError(t, err)
// 	err = appendFile.Close()
// 	require.NoError(t, err)

// 	// confirm all objects are still read
// 	i := 0
// 	iter, err := block.Iterator()
// 	bytesIter := iter.(BytesIterator)
// 	require.NoError(t, err)
// 	for {
// 		_, bytesObject, err := bytesIter.NextBytes(context.Background())
// 		if err == io.EOF {
// 			break
// 		}
// 		require.NoError(t, err)

// 		req, err := dec.PrepareForRead(bytesObject)
// 		require.NoError(t, err)

// 		require.True(t, proto.Equal(req, reqs[i]))
// 		i++
// 	}
// 	require.Equal(t, numMsgs, i)
// }

func TestParseFilename(t *testing.T) {
	tests := []struct {
		name            string
		filename        string
		expectUUID      uuid.UUID
		expectTenant    string
		expectedVersion string
		expectError     bool
	}{
		{
			name:            "happy path",
			filename:        "123e4567-e89b-12d3-a456-426614174000+tenant+vParquet",
			expectUUID:      uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"),
			expectTenant:    "tenant",
			expectedVersion: "vParquet",
		},
		{
			name:        "path fails",
			filename:    "/blerg/123e4567-e89b-12d3-a456-426614174000+tenant+vParquet",
			expectError: true,
		},
		{
			name:        "no +",
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
			filename:    "123e4+tenant+vParquet",
			expectError: true,
		},
		{
			name:        "no tenant",
			filename:    "123e4567-e89b-12d3-a456-426614174000++vParquet",
			expectError: true,
		},
		{
			name:        "no version",
			filename:    "123e4567-e89b-12d3-a456-426614174000+tenant+",
			expectError: true,
		},
		{
			name:        "wrong version",
			filename:    "123e4567-e89b-12d3-a456-426614174000+tenant+v2",
			expectError: true,
		},
		{
			name:        "wrong splits - 4",
			filename:    "123e4567-e89b-12d3-a456-426614174000+test+test+test",
			expectError: true,
		},
		{
			name:        "wrong splits - 2",
			filename:    "123e4567-e89b-12d3-a456-426614174000+test",
			expectError: true,
		},
		{
			name:        "wrong splits - 1",
			filename:    "123e4567-e89b-12d3-a456-426614174000",
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actualUUID, actualTenant, actualVersion, err := parseName(tc.filename)

			if tc.expectError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.expectUUID, actualUUID)
			require.Equal(t, tc.expectTenant, actualTenant)
			require.Equal(t, tc.expectedVersion, actualVersion)
		})
	}
}
