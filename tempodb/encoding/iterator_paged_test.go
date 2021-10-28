package encoding

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math/rand"
	"strconv"
	"testing"

	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIteratorPaged tests the iterator paging functionality
func TestIteratorPaged(t *testing.T) {
	const totalObjects = 1000
	const indexDownsampleBytes = 1000
	const chunkSizeBytes = 500

	for _, enc := range allEncodings() {
		// build a paged appender with totalObjects
		buff := &bytes.Buffer{}
		writer, err := enc.NewDataWriter(buff, backend.EncNone)
		require.NoError(t, err)

		appender, err := NewBufferedAppender(writer, indexDownsampleBytes, totalObjects)
		require.NoError(t, err)

		ids := make([]common.ID, 0, totalObjects)
		objs := make([][]byte, 0, totalObjects)
		for i := 0; i < totalObjects; i++ {
			obj := make([]byte, 100)
			_, err = rand.Read(obj)
			require.NoError(t, err)
			id := []byte(strconv.Itoa(i))

			ids = append(ids, id)
			objs = append(objs, obj)

			appender.Append(id, obj)
		}

		err = appender.Complete()
		require.NoError(t, err)

		// now iterate through the pages/data created by the appender and assert that all objects are returned
		reader, err := enc.NewDataReader(backend.NewContextReaderWithAllReader(bytes.NewReader(buff.Bytes())), backend.EncNone)
		require.NoError(t, err)

		iterator := newPagedIterator(chunkSizeBytes, common.Records(appender.Records()), reader, enc.NewObjectReaderWriter())

		for {
			id, obj, err := iterator.Next(context.Background())
			if err == io.EOF {
				break
			}
			require.NoError(t, err)

			assert.Equal(t, ids[0], id)
			assert.Equal(t, objs[0], obj)

			ids = ids[1:]
			objs = objs[1:]
		}

		assert.Len(t, ids, 0)
		assert.Len(t, objs, 0)
	}
}

// TestIteratorPartialPaged tests the iterator paging functionality
func TestIteratorPartialPaged(t *testing.T) {
	const totalObjects = 1000
	const indexDownsampleBytes = 1000

	for _, enc := range allEncodings() {
		// build a paged appender with totalObjects. it's important to use a plain appender b/c
		// it creates a record for every single object. this allows us to know exactly what to expect returned
		// when using the partial iterator.
		buff := &bytes.Buffer{}
		writer, err := enc.NewDataWriter(buff, backend.EncNone)
		require.NoError(t, err)

		appender := NewAppender(writer)
		ids := make([]common.ID, 0, totalObjects)
		objs := make([][]byte, 0, totalObjects)
		for i := 0; i < totalObjects; i++ {
			obj := make([]byte, 100)
			_, err = rand.Read(obj)
			require.NoError(t, err)
			// ids and objects must be ordered ascending for this test to work. this is b/c the appender returns its
			// records sorted by id. so by creating ascending ids here we can guarantee everything lines up
			id := []byte(fmt.Sprintf("%4d", i))

			ids = append(ids, id)
			objs = append(objs, obj)

			appender.Append(id, obj)
		}

		err = appender.Complete()
		require.NoError(t, err)

		// now test the iterator
		reader, err := enc.NewDataReader(backend.NewContextReaderWithAllReader(bytes.NewReader(buff.Bytes())), backend.EncNone)
		require.NoError(t, err)

		startPage := 35
		totalPages := 117
		// chunk size is 0, to force every index to be individually retrieved. otherwise the datareader will return errors
		// due to accessing non-contiguous pages
		iterator := newPartialPagedIterator(0, common.Records(appender.Records()), reader, enc.NewObjectReaderWriter(), startPage, totalPages)
		ids = ids[startPage : startPage+totalPages]
		objs = objs[startPage : startPage+totalPages]

		for {
			id, obj, err := iterator.Next(context.Background())
			if err == io.EOF {
				break
			}
			require.NoError(t, err)

			assert.Equal(t, ids[0], id)
			assert.Equal(t, objs[0], obj)

			ids = ids[1:]
			objs = objs[1:]
		}

		assert.Len(t, ids, 0)
		assert.Len(t, objs, 0)
	}
}
