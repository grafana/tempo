package v0

import (
	"bytes"
	"io"
	"sort"

	"github.com/grafana/tempo/tempodb/encoding/common"
)

type appender struct {
	writer        io.Writer
	records       []*common.Record
	currentOffset uint64
}

// NewAppender returns an appender.  This appender simply appends new objects
//  to the provided io.Writer.
func NewAppender(writer io.Writer) common.Appender {
	return &appender{
		writer: writer,
	}
}

// NewPrefilledAppender builds its internal index based on the contents of reader
//  and writes future appends to writer.  This method is meant to be used when an
//  already existing block of records exist.  Perhaps when replaying a WAL.
func NewPrefilledAppender(reader io.Reader, writer io.Writer) (common.Appender, error) {
	a := &appender{
		writer: writer,
	}

	var offset uint64
	for {
		r, err := unmarshalObjectRecordFromReader(reader, &offset)
		if err != nil {
			return nil, err
		}
		if r == nil {
			break
		}

		a.addRecord(r)
	}

	a.currentOffset = offset

	return a, nil
}

// Append appends the id/object to the writer.  Note that the caller is giving up ownership of the two byte arrays backing the slices.
//   Copies should be made and passed in if this is a problem
func (a *appender) Append(id common.ID, b []byte) error {
	length, err := marshalObjectToWriter(id, b, a.writer)
	if err != nil {
		return err
	}

	a.addRecord(&common.Record{
		ID:     id,
		Start:  a.currentOffset,
		Length: uint32(length),
	})

	return nil
}

func (a *appender) Records() []*common.Record {
	return a.records
}

func (a *appender) Length() int {
	return len(a.records)
}

func (a *appender) Complete() {

}

func (a *appender) addRecord(r *common.Record) {
	id := r.ID

	i := sort.Search(len(a.records), func(idx int) bool {
		return bytes.Compare(a.records[idx].ID, id) == 1
	})
	a.records = append(a.records, nil)
	copy(a.records[i+1:], a.records[i:])
	a.records[i] = r

	a.currentOffset += uint64(r.Length)
}
