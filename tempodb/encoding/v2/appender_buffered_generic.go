package v2

import (
	"context"

	"github.com/grafana/tempo/v2/tempodb/encoding/common"
)

// bufferedAppender buffers objects into pages and builds a downsampled
// index
type BufferedAppenderGeneric struct {
	// output writer
	writer DataWriterGeneric

	// record keeping
	records             []Record
	totalObjects        int
	currentOffset       uint64
	currentRecord       *Record
	currentBytesWritten int

	// config
	maxPageSize int
}

// NewBufferedAppender returns an bufferedAppender.  This appender builds a writes to
// the provided writer and also builds a downsampled records slice.
func NewBufferedAppenderGeneric(writer DataWriterGeneric, maxPageSize int) *BufferedAppenderGeneric {
	return &BufferedAppenderGeneric{
		writer:      writer,
		maxPageSize: maxPageSize,
		records:     make([]Record, 0),
	}
}

// Append appends the id/object to the writer.  Note that the caller is giving up ownership of the two byte arrays backing the slices.
// Copies should be made and passed in if this is a problem
func (a *BufferedAppenderGeneric) Append(ctx context.Context, id common.ID, i interface{}) error {
	bytesWritten, err := a.writer.Write(ctx, id, i)
	if err != nil {
		return err
	}

	if a.currentRecord == nil {
		a.currentRecord = &Record{
			Start: a.currentOffset,
		}
	}
	a.totalObjects++
	a.currentBytesWritten += bytesWritten
	a.currentRecord.ID = id

	if a.currentBytesWritten > a.maxPageSize {
		err := a.flush(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

// Records returns a slice of the current records
func (a *BufferedAppenderGeneric) Records() []Record {
	return a.records
}

// Complete flushes all buffers and releases resources
func (a *BufferedAppenderGeneric) Complete(ctx context.Context) error {
	err := a.flush(ctx)
	if err != nil {
		return err
	}

	return a.writer.Complete(ctx)
}

func (a *BufferedAppenderGeneric) flush(ctx context.Context) error {
	if a.currentRecord == nil {
		return nil
	}

	bytesWritten, err := a.writer.CutPage(ctx)
	if err != nil {
		return err
	}

	a.currentOffset += uint64(bytesWritten)
	a.currentRecord.Length += uint32(bytesWritten)

	// update index
	a.records = append(a.records, *a.currentRecord)
	a.currentRecord = nil
	a.currentBytesWritten = 0

	return nil
}
