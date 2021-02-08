package v0

import (
	"bytes"
	"math/rand"
	"testing"

	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/stretchr/testify/assert"
)

func TestEncodeDecodeRecord(t *testing.T) {
	expected, err := makeRecord(t)
	assert.NoError(t, err, "unexpected error making trace record")

	buff := make([]byte, recordLength)

	marshalRecord(expected, buff)
	actual := unmarshalRecord(buff)

	assert.Equal(t, expected, actual)
}

func TestMarshalUnmarshalRecords(t *testing.T) {
	numRecords := 10
	expected := make([]*common.Record, 0, numRecords)

	for i := 0; i < numRecords; i++ {
		r, err := makeRecord(t)
		if err != nil {
			assert.NoError(t, err, "unexpected error making trace record")
		}
		expected = append(expected, r)
	}

	recordBytes, err := marshalRecords(expected)
	assert.NoError(t, err, "unexpected error encoding records")
	assert.Equal(t, len(expected)*28, len(recordBytes))

	actual, err := unmarshalRecords(recordBytes)
	assert.NoError(t, err, "unexpected error decoding records")

	assert.Equal(t, expected, actual)
}

/*func TestFindRecord(t *testing.T) { // jpe - move to index_reader_test.go
	numRecords := 10
	expected := make([]*common.Record, 0, numRecords)

	for i := 0; i < numRecords; i++ {
		r, err := makeRecord(t)
		if err != nil {
			assert.NoError(t, err, "unexpected error making trace record")
		}
		expected = append(expected, r)
	}

	sortRecords(expected)

	recordBytes, err := marshalRecords(expected)
	assert.NoError(t, err, "unexpected error encoding records")

	for _, r := range expected {
		found, err := findRecord(r.ID, recordBytes)

		assert.NoError(t, err, "unexpected error finding records")
		assert.Equal(t, r, found)
	}
}*/

func TestSortRecord(t *testing.T) {
	numRecords := 10
	expected := make([]*common.Record, 0, numRecords)

	for i := 0; i < numRecords; i++ {
		r, err := makeRecord(t)
		if err != nil {
			assert.NoError(t, err, "unexpected error making trace record")
		}
		expected = append(expected, r)
	}

	sortRecords(expected)

	for i := range expected {
		if i == 0 {
			continue
		}

		idSmaller := expected[i-1].ID
		idLarger := expected[i].ID

		assert.NotEqual(t, 1, bytes.Compare(idSmaller, idLarger))
	}
}

// todo: belongs in util/test?
func makeRecord(t *testing.T) (*common.Record, error) {
	t.Helper()

	r := newRecord()
	r.Start = rand.Uint64()
	r.Length = rand.Uint32()

	_, err := rand.Read(r.ID)
	if err != nil {
		return nil, err
	}

	return r, nil
}
