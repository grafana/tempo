package v2

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/tempodb/backend"
)

func TestReaderNextPage(t *testing.T) {
	totalObjects := 10000
	objsPerPage := 100
	enc := backend.EncZstd

	ids, objs, buffer, _ := createTestData(t, totalObjects, objsPerPage, enc)
	testNextPage(t, totalObjects, enc, ids, objs, buffer)
}

func TestReaderRead(t *testing.T) {
	totalObjects := 10000
	objsPerPage := 100
	enc := backend.EncZstd

	ids, objs, buffer, recs := createTestData(t, totalObjects, objsPerPage, enc)
	testRead(t, totalObjects, enc, ids, objs, buffer, recs)
}

func BenchmarkReaderRead(b *testing.B) {
	totalObjects := 10000
	objsPerPage := 100
	enc := backend.EncZstd

	ids, objs, buffer, recs := createTestData(b, totalObjects, objsPerPage, enc)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		testRead(b, totalObjects, enc, ids, objs, buffer, recs)
	}
}

func BenchmarkReaderNextPage(b *testing.B) {
	totalObjects := 10000
	objsPerPage := 100
	enc := backend.EncZstd

	ids, objs, buffer, _ := createTestData(b, totalObjects, objsPerPage, enc)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		testNextPage(b, totalObjects, enc, ids, objs, buffer)
	}
}

func BenchmarkWriter(b *testing.B) {
	totalObjects := 10000
	objsPerPage := 100
	enc := backend.EncZstd

	for i := 0; i < b.N; i++ {
		_, _, _, _ = createTestData(b, totalObjects, objsPerPage, enc)
	}
}

// nolint:unparam
func testNextPage(t require.TestingT, totalObjects int, enc backend.Encoding, ids, objs [][]byte, buffer []byte) {
	reader := bytes.NewReader(buffer)
	r, err := NewDataReader(backend.NewContextReaderWithAllReader(reader), enc)
	require.NoError(t, err)
	defer r.Close()

	var tempBuffer []byte
	o := NewObjectReaderWriter()
	i := 0
	for {
		tempBuffer, _, err = r.NextPage(tempBuffer)
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)

		var id, obj []byte
		bufferReader := bytes.NewReader(tempBuffer)

		for {
			id, obj, err = o.UnmarshalObjectFromReader(bufferReader)
			if errors.Is(err, io.EOF) {
				break
			}

			assert.Equal(t, ids[i], id)
			assert.Equal(t, objs[i], obj)
			i++
		}
	}

	assert.Equal(t, totalObjects, i)
}

// nolint:unparam
func testRead(t require.TestingT, totalObjects int, enc backend.Encoding, ids, objs [][]byte, buffer []byte, recs Records) {
	reader := bytes.NewReader(buffer)
	r, err := NewDataReader(backend.NewContextReaderWithAllReader(reader), enc)
	require.NoError(t, err)
	defer r.Close()

	var pages [][]byte
	ctx := context.Background()
	tempBuffer := []byte{}
	o := NewObjectReaderWriter()
	i := 0
	for j := 0; j < len(recs); j++ {
		pages, tempBuffer, err = r.Read(ctx, []Record{recs[j]}, pages, tempBuffer)
		require.NoError(t, err)
		require.Len(t, pages, 1)

		var id, obj []byte
		page := pages[0]
		for {
			page, id, obj, err = o.UnmarshalAndAdvanceBuffer(page)
			if errors.Is(err, io.EOF) {
				break
			}

			assert.Equal(t, ids[i], id)
			assert.Equal(t, objs[i], obj)
			i++
		}
	}

	assert.Equal(t, totalObjects, i)
}

// nolint:unparam
func createTestData(t require.TestingT, totalObjects, objsPerPage int, enc backend.Encoding) ([][]byte, [][]byte, []byte, Records) {
	buffer := &bytes.Buffer{}

	w, err := NewDataWriter(buffer, enc)
	require.NoError(t, err)

	bytesWritten := 0

	recs := Records{}
	ids := [][]byte{}
	objs := [][]byte{}
	for i := 0; i < totalObjects; i++ {
		id := make([]byte, 10)
		obj := make([]byte, 100)

		_, err = rand.Read(id)
		require.NoError(t, err)
		_, err = rand.Read(obj)
		require.NoError(t, err)

		_, err = w.Write(id, obj)
		require.NoError(t, err)

		ids = append(ids, id)
		objs = append(objs, obj)

		if (i+1)%objsPerPage == 0 || i == (totalObjects-1) {
			count, err := w.CutPage()
			require.NoError(t, err)

			recs = append(recs, Record{
				Start:  uint64(bytesWritten),
				Length: uint32(count),
			})
			bytesWritten += count
		}
	}
	err = w.Complete()
	require.NoError(t, err)

	return ids, objs, buffer.Bytes(), recs
}
