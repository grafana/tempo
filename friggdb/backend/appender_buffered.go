package backend

import (
	"io"
)

type bufferedAppender struct {
	writer  io.Writer
	records []*Record

	totalObjects    int
	currentOffset   uint64
	currentRecord   *Record
	indexDownsample int
}

func NewBufferedAppender(writer io.Writer, indexDownsample int, totalObjectsEstimate int) Appender {
	return &bufferedAppender{
		writer:          writer,
		records:         make([]*Record, 0, totalObjectsEstimate/indexDownsample+1),
		indexDownsample: indexDownsample,
	}
}

func (a *bufferedAppender) Append(id ID, b []byte) error {
	length, err := marshalObjectToWriter(id, b, a.writer)
	if err != nil {
		return err
	}

	if a.currentRecord == nil {
		a.currentRecord = &Record{
			Start: a.currentOffset,
		}
	}
	a.totalObjects++
	a.currentOffset += uint64(length)

	a.currentRecord.ID = id // jpe consider memory leak issues here
	a.currentRecord.Length += uint32(length)

	if a.totalObjects%a.indexDownsample == 0 {
		a.records = append(a.records, a.currentRecord)
		a.currentRecord = nil
	}

	return nil
}

func (a *bufferedAppender) Records() []*Record {
	return a.records
}

func (a *bufferedAppender) Length() int {
	return a.totalObjects
}

func (a *bufferedAppender) Complete() {
	if a.currentRecord == nil {
		return
	}
	a.records = append(a.records, a.currentRecord)
	a.currentRecord = nil
}
