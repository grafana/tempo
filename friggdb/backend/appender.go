package backend

import (
	"bytes"
	"io"
	"sort"
)

type Appender interface {
	Append(ID, []byte) error
	Complete()
	Records() []*Record
	Length() int
}

type appender struct {
	writer        io.Writer
	records       []*Record
	currentOffset int
}

func NewAppender(writer io.Writer) Appender {
	return &appender{
		writer: writer,
	}
}

func (a *appender) Append(id ID, b []byte) error {
	length, err := marshalObjectToWriter(id, b, a.writer)
	if err != nil {
		return err
	}

	i := sort.Search(len(a.records), func(idx int) bool {
		return bytes.Compare(a.records[idx].ID, id) == 1
	})
	a.records = append(a.records, nil)
	copy(a.records[i+1:], a.records[i:])
	a.records[i] = &Record{
		ID:     id, // jpe memory leak?
		Start:  uint64(a.currentOffset),
		Length: uint32(length),
	}

	a.currentOffset += length
	return nil
}

func (a *appender) Records() []*Record {
	return a.records
}

func (a *appender) Length() int {
	return len(a.records)
}

func (a *appender) Complete() {

}
