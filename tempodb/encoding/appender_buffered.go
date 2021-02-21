package encoding

import (
	"io"

	"github.com/grafana/tempo/tempodb/encoding/common"
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

// bufferedAppender buffers objects into pages and builds a downsampled
// index
type bufferedAppender struct {
	// output writer
	writer common.PageWriter

	// record keeping
	records             []*common.Record
	totalObjects        int
	currentOffset       uint64
	currentRecord       *common.Record
	currentBytesWritten int

	// config
	indexDownsampleBytes int
}

// NewBufferedAppender returns an bufferedAppender.  This appender builds a writes to
//  the provided writer and also builds a downsampled records slice.
func NewBufferedAppender(writer common.PageWriter, indexDownsample int, totalObjectsEstimate int) (common.Appender, error) {
	return &bufferedAppender{
		writer:               writer,
		indexDownsampleBytes: indexDownsample,
		records:              make([]*common.Record, 0, totalObjectsEstimate/indexDownsample+1),
	}, nil
}

// Append appends the id/object to the writer.  Note that the caller is giving up ownership of the two byte arrays backing the slices.
//   Copies should be made and passed in if this is a problem
func (a *bufferedAppender) Append(id common.ID, b []byte) error {
	bytesWritten, err := a.writer.Write(id, b)
	if err != nil {
		return err
	}

	if a.currentRecord == nil {
		a.currentRecord = &common.Record{
			Start: a.currentOffset,
		}
	}
	a.totalObjects++
	a.currentBytesWritten += bytesWritten
	a.currentRecord.ID = id

	if a.currentBytesWritten > a.indexDownsampleBytes {
		err := a.flush()
		if err != nil {
			return err
		}
	}

	return nil
}

// Records returns a slice of the current records
func (a *bufferedAppender) Records() []*common.Record {
	return a.records
}

// Length returns the number of written objects
func (a *bufferedAppender) Length() int {
	return a.totalObjects
}

// DataLength returns the number of written bytes
func (a *bufferedAppender) DataLength() uint64 {
	return a.currentOffset
}

// Complete flushes all buffers and releases resources
func (a *bufferedAppender) Complete() error {
	err := a.flush()
	if err != nil {
		return err
	}

	return a.writer.Complete()
}

func (a *bufferedAppender) flush() error {
	if a.currentRecord == nil {
		return nil
	}

	bytesWritten, err := a.writer.CutPage()
	if err != nil {
		return err
	}

	a.currentBytesWritten = 0
	a.currentOffset += uint64(bytesWritten)
	a.currentRecord.Length += uint32(bytesWritten)

	// update index
	a.records = append(a.records, a.currentRecord)
	a.currentRecord = nil

	return nil
}
