package v1

import (
	"bytes"
	"io"

	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
	v0 "github.com/grafana/tempo/tempodb/encoding/v0"
)

// meteredWriter is a struct that is used to count the number of bytes
// written to a block after compression.  Unfortunately the compression io.Reader
// returns bytes before compression so this is necessary to know the actual number of
// byte written.
type meteredWriter struct {
	wrappedWriter io.Writer
	bytesWritten  int
}

func (m *meteredWriter) Write(p []byte) (n int, err error) {
	m.bytesWritten += len(p)
	return m.wrappedWriter.Write(p)
}

// buffer up in memory and then write a big ol' compressed block o shit at once
//  used by CompleteBlock/CompactorBlock
// may need additional code?  i.e. a signal that it's "about to flush" triggering a compression
type bufferedAppender struct {
	v0Buffer     *bytes.Buffer
	outputWriter *meteredWriter
	pool         WriterPool
	records      []*common.Record

	totalObjects    int
	currentOffset   uint64
	currentRecord   *common.Record
	indexDownsample int
}

// NewBufferedAppender returns an bufferedAppender.  This appender builds a writes to
//  the provided writer and also builds a downsampled records slice.
func NewBufferedAppender(writer io.Writer, encoding backend.Encoding, indexDownsample int, totalObjectsEstimate int) (common.Appender, error) {
	pool, err := getWriterPool(encoding)
	if err != nil {
		return nil, err
	}

	return &bufferedAppender{
		v0Buffer:        &bytes.Buffer{},
		indexDownsample: indexDownsample,
		records:         make([]*common.Record, 0, totalObjectsEstimate/indexDownsample+1),

		outputWriter: &meteredWriter{
			wrappedWriter: writer,
		},
		pool: pool,
	}, nil
}

// Append appends the id/object to the writer.  Note that the caller is giving up ownership of the two byte arrays backing the slices.
//   Copies should be made and passed in if this is a problem
func (a *bufferedAppender) Append(id common.ID, b []byte) error {
	_, err := v0.MarshalObjectToWriter(id, b, a.v0Buffer)
	if err != nil {
		return err
	}

	if a.currentRecord == nil {
		a.currentRecord = &common.Record{
			Start: a.currentOffset,
		}
	}
	a.totalObjects++
	a.currentRecord.ID = id

	if a.totalObjects%a.indexDownsample == 0 {
		err := a.flush()
		if err != nil {
			return err
		}
	}

	return nil
}

func (a *bufferedAppender) Records() []*common.Record {
	return a.records
}

func (a *bufferedAppender) Length() int {
	return a.totalObjects
}

func (a *bufferedAppender) DataLength() uint64 {
	return a.currentOffset
}

// compress everything left and flush?
func (a *bufferedAppender) Complete() error {
	err := a.flush()
	if err != nil {
		return err
	}

	return nil
}

func (a *bufferedAppender) flush() error {
	if a.currentRecord == nil {
		return nil
	}

	compressedWriter := a.pool.GetWriter(a.outputWriter)

	// write compressed data
	buffer := a.v0Buffer.Bytes()
	_, err := compressedWriter.Write(buffer)
	if err != nil {
		return err
	}

	// now clear our v0 buffer so we can start the new block page
	compressedWriter.Close()
	a.v0Buffer.Reset()
	a.pool.PutWriter(compressedWriter)

	a.currentOffset += uint64(a.outputWriter.bytesWritten)
	a.currentRecord.Length += uint32(a.outputWriter.bytesWritten)
	a.outputWriter.bytesWritten = 0

	// update index
	a.records = append(a.records, a.currentRecord)
	a.currentRecord = nil

	return nil
}
