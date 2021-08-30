package v2

import (
	"bytes"
	"io"
	"math/rand"
	"testing"

	"github.com/grafana/tempo/tempodb/backend"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNextPage(t *testing.T) {
	totalObjects := 10000
	objsPerPage := 100
	enc := backend.EncZstd

	ids, objs, buffer := createTestData(t, totalObjects, objsPerPage, enc)
	testNextPage(t, totalObjects, enc, ids, objs, buffer)
}

func BenchmarkReaderNextPage(b *testing.B) {
	totalObjects := 10000
	objsPerPage := 100
	enc := backend.EncZstd

	ids, objs, buffer := createTestData(b, totalObjects, objsPerPage, enc)
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
		_, _, _ = createTestData(b, totalObjects, objsPerPage, enc)
	}
}

func testNextPage(t require.TestingT, totalObjects int, enc backend.Encoding, ids [][]byte, objs [][]byte, buffer []byte) {
	reader := bytes.NewReader(buffer)
	r, err := NewDataReader(backend.NewContextReaderWithAllReader(reader), enc)
	require.NoError(t, err)
	defer r.Close()

	tempBuffer := []byte{}
	o := NewObjectReaderWriter()
	i := 0
	for {
		tempBuffer, _, err = r.NextPage(tempBuffer)
		if err == io.EOF {
			break
		}
		require.NoError(t, err)

		var id, obj []byte
		for {
			tempBuffer, id, obj, err = o.UnmarshalAndAdvanceBuffer(tempBuffer)
			if err == io.EOF {
				break
			}

			assert.Equal(t, ids[i], id)
			assert.Equal(t, objs[i], obj)
			i++
		}
	}

	assert.Equal(t, totalObjects, i)
}

func createTestData(t require.TestingT, totalObjects int, objsPerPage int, enc backend.Encoding) ([][]byte, [][]byte, []byte) {
	buffer := &bytes.Buffer{}

	w, err := NewDataWriter(buffer, enc)
	require.NoError(t, err)
	defer w.Complete()

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
			_, err := w.CutPage()
			require.NoError(t, err)
		}
	}

	return ids, objs, buffer.Bytes()
}
