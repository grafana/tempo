package v0

import (
	"io"

	"github.com/grafana/tempo/tempodb/encoding/common"
)

type bufferedAppender struct {
	writer  io.Writer
	records []*common.Record

	totalObjects    int
	currentOffset   uint64
	currentRecord   *common.Record
	indexDownsample int
}

// NewBufferedAppender returns an bufferedAppender.  This appender builds a writes to
//  the provided writer and also builds a downsampled records slice.
func NewBufferedAppender(writer io.Writer, indexDownsample int, totalObjectsEstimate int) common.Appender {
	return &bufferedAppender{
		writer:          writer,
		records:         make([]*common.Record, 0, totalObjectsEstimate/indexDownsample+1),
		indexDownsample: indexDownsample,
	}
}

// Append appends the id/object to the writer.  Note that the caller is giving up ownership of the two byte arrays backing the slices.
//   Copies should be made and passed in if this is a problem
func (a *bufferedAppender) Append(id common.ID, b []byte) error {
	length, err := MarshalObjectToWriter(id, b, a.writer)
	if err != nil {
		return err
	}

	if a.currentRecord == nil {
		a.currentRecord = &common.Record{
			Start: a.currentOffset,
		}
	}
	a.totalObjects++
	a.currentOffset += uint64(length)

	a.currentRecord.ID = id
	a.currentRecord.Length += uint32(length)

	if a.totalObjects%a.indexDownsample == 0 {
		a.records = append(a.records, a.currentRecord)
		a.currentRecord = nil
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

func (a *bufferedAppender) Complete() error {
	if a.currentRecord == nil {
		return nil
	}
	a.records = append(a.records, a.currentRecord)
	a.currentRecord = nil

	return nil
}
