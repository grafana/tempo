package v1

import (
	"bytes"
	"io"

	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
	v0 "github.com/grafana/tempo/tempodb/encoding/v0"
)

// buffer up in memory and then write a big ol' compressed block o shit at once
//  used by CompleteBlock/CompactorBlock
// may need additional code?  i.e. a signal that it's "about to flush" triggering a compression

type bufferedAppender struct {
	v0Appender      common.Appender
	v0Buffer        *bytes.Buffer
	indexDownsample int

	compressedWriter  io.WriteCloser
	compressedDataLen uint64
	outputWriter      io.Writer
	pool              WriterPool
}

// NewBufferedAppender returns an bufferedAppender.  This appender builds a writes to
//  the provided writer and also builds a downsampled records slice.
func NewBufferedAppender(writer io.Writer, encoding backend.Encoding, indexDownsample int, totalObjectsEstimate int) (common.Appender, error) {
	v0Buffer := &bytes.Buffer{}
	pool, err := getWriterPool(encoding)
	if err != nil {
		return nil, err
	}

	return &bufferedAppender{
		v0Appender:      v0.NewBufferedAppender(v0Buffer, indexDownsample, totalObjectsEstimate),
		v0Buffer:        v0Buffer,
		indexDownsample: indexDownsample,

		compressedWriter: pool.GetWriter(writer),
		outputWriter:     writer,
		pool:             pool,
	}, nil
}

// Append appends the id/object to the writer.  Note that the caller is giving up ownership of the two byte arrays backing the slices.
//   Copies should be made and passed in if this is a problem
func (a *bufferedAppender) Append(id common.ID, b []byte) error {
	err := a.v0Appender.Append(id, b)
	if err != nil {
		return err
	}

	// each "index downsample" number of records is a "block page" and is independently compressed
	if a.v0Appender.Length()%a.indexDownsample == 0 {
		// write compressed data
		len, err := a.compressedWriter.Write(a.v0Buffer.Bytes())
		if err != nil {
			return err
		}
		a.compressedDataLen += uint64(len)

		// now clear our v0 buffer so we can start the new block page
		a.v0Buffer.Reset()
	}

	return nil
}

func (a *bufferedAppender) Records() []*common.Record {
	return a.v0Appender.Records()
}

func (a *bufferedAppender) Length() int {
	return a.v0Appender.Length()
}

func (a *bufferedAppender) DataLength() uint64 {
	return a.compressedDataLen
}

// compress everything left and flush?
func (a *bufferedAppender) Complete() {
	a.v0Appender.Complete()

	a.pool.PutWriter(a.compressedWriter)
}
