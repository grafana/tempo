package v2

import (
	"bytes"
	crand "crypto/rand"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEncodeDecodeRecord(t *testing.T) {
	expected, err := makeRecord(t)
	assert.NoError(t, err, "unexpected error making trace record")

	buff := make([]byte, recordLength)

	r := record{}
	marshalRecord(expected, buff)
	actual := r.UnmarshalRecord(buff)

	assert.Equal(t, expected, actual)
}

func TestSortRecord(t *testing.T) {
	numRecords := 10
	expected := make([]Record, 0, numRecords)

	for i := 0; i < numRecords; i++ {
		r, err := makeRecord(t)
		if err != nil {
			assert.NoError(t, err, "unexpected error making trace record")
		}
		expected = append(expected, r)
	}

	SortRecords(expected)

	for i := range expected {
		if i == 0 {
			continue
		}

		idSmaller := expected[i-1].ID
		idLarger := expected[i].ID

		assert.NotEqual(t, 1, bytes.Compare(idSmaller, idLarger))
	}
}

func makeRecord(t *testing.T) (Record, error) {
	t.Helper()

	r := Record{
		ID:     make([]byte, 16), // 128 bits
		Start:  0,
		Length: 0,
	}

	_, err := crand.Read(r.ID)
	if err != nil {
		return Record{}, err
	}

	return r, nil
}
