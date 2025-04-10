package io

import (
	"bytes"
	"errors"
	"io"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBufferedReaderAtCalculateBounds(t *testing.T) {
	testCases := []struct {
		readerAtSize                   int64
		offset, length                 int64
		bufferSize                     int
		expectedOffset, expectedLength int64
	}{
		// Happy case - read in middle of file is extended
		{
			100,   // Input is 100 bytes long
			1, 50, // Read 50 bytes
			75, // Buffer attempts 75
			1, 75,
		},

		// Request is larger than buffer size
		{
			100,   // Input is 100 bytes long
			1, 50, // Read 50 bytes
			25, // Buffer is only 25
			1, 50,
		},

		// ReaderAt size smaller than the buffer size
		{
			100,    // Input is 100 bytes long
			0, 100, // Read 100 bytes at beginning
			1000, // Buffer attempts 1000
			0, 100,
		},

		// Read at end of file is backed up to buffer size
		{
			100,   // Input is 100 bytes long
			99, 1, // Read last byte
			10,     // Buffer attempts 10
			90, 10, // Offset backed up to satisfy buffer size
		},

		// No buffering
		{
			100,    // Input is 100 bytes long
			25, 50, // Read 50 bytes in the middle
			0, // No buffering
			25, 50,
		},
	}

	for _, tc := range testCases {
		o, l := calculateBounds(tc.offset, tc.length, tc.bufferSize, tc.readerAtSize)
		require.Equal(t, tc.expectedOffset, o, "check offset")
		require.Equal(t, tc.expectedLength, l, "check length")
	}
}

func TestBufferedReaderAt(t *testing.T) {
	input := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	inputReader := bytes.NewReader(input)

	testConfigs := []struct {
		bufferSize  int
		bufferCount int
	}{
		{0, 0},
		{5, 5},
		{100, 100},
	}

	testReads := []struct {
		offset, length int64
	}{
		{0, 3},
		{0, 10},
		{5, 5},
		{9, 1},
	}

	for _, tc := range testConfigs {
		r := NewBufferedReaderAt(inputReader, int64(len(input)), tc.bufferSize, tc.bufferCount)

		for _, tr := range testReads {
			b := make([]byte, tr.length)
			_, err := r.ReadAt(b, tr.offset)
			require.NoError(t, err)
			require.Equal(t, input[tr.offset:tr.offset+tr.length], b)
		}
	}
}

func TestBufferedReaderConcurrencyAndFuzz(t *testing.T) {
	const minLen = 100

	for i := 0; i < 100; i++ {
		inputLen := rand.Intn(1024) + minLen
		input := make([]byte, inputLen)
		inputReader := bytes.NewReader(input)

		// write 0 -> 1023 to input
		for i := range input {
			input[i] = byte(i)
		}

		r := NewBufferedReaderAt(inputReader, int64(len(input)), 50, 1)

		for i := 0; i < 1000; i++ {
			go func() {
				length := rand.Intn(minLen)
				offset := rand.Intn(len(input) - length)

				b := make([]byte, length)
				_, err := r.ReadAt(b, int64(offset))
				require.NoError(t, err)
				// require actual to be expected
				require.Equal(t, input[offset:offset+length], b)
			}()
		}
	}
}

type erroringReaderAt struct {
	err error
}

func (e *erroringReaderAt) ReadAt(p []byte, off int64) (n int, err error) {
	if e.err != nil && !errors.Is(e.err, io.EOF) {
		// set all bytes to 0
		for i := range p {
			p[i] = 0
		}
		return 0, e.err
	}

	// set the bytes to the offset
	for i := range p {
		p[i] = byte(off + int64(i))
	}

	return len(p), e.err
}

func TestBufferedReaderInvalidatesBufferOnErr(t *testing.T) {
	erroringReaderAt := &erroringReaderAt{
		err: nil,
	}

	r := NewBufferedReaderAt(erroringReaderAt, 100, 50, 1)

	// force the reader to return an error
	erroringReaderAt.err = errors.New("error")
	actual := make([]byte, 10)
	read, err := r.ReadAt(actual, 0)
	require.Error(t, err)
	require.Equal(t, 0, read)
	require.Equal(t, []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, actual) // first 10 bytes should be zeroed

	// clear the error and read the first 10 bytes again
	erroringReaderAt.err = nil
	actual = make([]byte, 10)
	read, err = r.ReadAt(actual, 0)
	require.NoError(t, err)
	require.Equal(t, 10, read)
	require.Equal(t, []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}, actual) // first 10 bytes should be read

	// force the reader to return io.EOF and see it handled correctly
	erroringReaderAt.err = io.EOF
	actual = make([]byte, 10)
	read, err = r.ReadAt(actual, 90)
	require.ErrorIs(t, err, io.EOF)
	require.Equal(t, 10, read)
	require.Equal(t, []byte{90, 91, 92, 93, 94, 95, 96, 97, 98, 99}, actual) // last 10 bytes should be read
}
