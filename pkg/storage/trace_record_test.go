package storage

import (
	"bytes"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/willf/bloom"
)

func TestEncodeDecodeRecord(t *testing.T) {
	expected, err := makeTraceRecord(t)
	assert.NoError(t, err, "unexpected error making trace record")

	actual := newTraceRecord()
	buff := make([]byte, 28)

	encodeRecord(expected, buff)
	decodeRecord(buff, actual)

	assert.Equal(t, expected, actual)
}

func TestEncodeDecodeRecords(t *testing.T) {
	numRecords := 10
	expected := make([]*TraceRecord, 0, numRecords)

	for i := 0; i < numRecords; i++ {
		r, err := makeTraceRecord(t)
		if err != nil {
			assert.NoError(t, err, "unexpected error making trace record")
		}
		expected = append(expected, r)
	}

	recordBytes, bloomBytes, err := encodeRecords(expected)
	assert.NoError(t, err, "unexpected error encoding records")

	actual, err := decodeRecords(recordBytes)
	assert.NoError(t, err, "unexpected error decoding records")

	filter := bloom.New(1, 1)
	filter.GobDecode(bloomBytes)

	for _, r := range expected {
		assert.True(t, filter.Test(r.TraceID))
	}

	assert.Equal(t, expected, actual)
}

func TestFindRecord(t *testing.T) {
	numRecords := 10
	expected := make([]*TraceRecord, 0, numRecords)

	for i := 0; i < numRecords; i++ {
		r, err := makeTraceRecord(t)
		if err != nil {
			assert.NoError(t, err, "unexpected error making trace record")
		}
		expected = append(expected, r)
	}

	SortRecords(expected)

	recordBytes, _, err := encodeRecords(expected)
	assert.NoError(t, err, "unexpected error encoding records")

	for _, r := range expected {
		found, err := findRecord(r.TraceID, recordBytes)

		assert.NoError(t, err, "unexpected error finding records")
		assert.Equal(t, r, found)
	}
}

func TestSortRecord(t *testing.T) {
	numRecords := 10
	expected := make([]*TraceRecord, 0, numRecords)

	for i := 0; i < numRecords; i++ {
		r, err := makeTraceRecord(t)
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

		idSmaller := expected[i-1].TraceID
		idLarger := expected[i].TraceID

		assert.NotEqual(t, 1, bytes.Compare(idSmaller, idLarger))
	}
}

// todo: belongs in util/test?
func makeTraceRecord(t *testing.T) (*TraceRecord, error) {
	t.Helper()

	r := newTraceRecord()
	r.Start = rand.Uint64()
	r.Length = rand.Uint32()

	_, err := rand.Read(r.TraceID)
	if err != nil {
		return nil, err
	}

	return r, nil
}
