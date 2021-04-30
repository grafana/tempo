package encoding

import (
	"bytes"
	"context"

	"github.com/grafana/tempo/tempodb/encoding/common"
)

type recordAppender struct {
	records []*common.Record
}

// NewRecordAppender returns an appender that stores records only.
func NewRecordAppender(records []*common.Record) Appender {
	return &recordAppender{
		records: records,
	}
}

// Append appends the id/object to the writer.  Note that the caller is giving up ownership of the two byte arrays backing the slices.
//   Copies should be made and passed in if this is a problem
func (a *recordAppender) Append(id common.ID, b []byte) error {
	return common.ErrUnsupported
}

func (a *recordAppender) Records() []*common.Record {
	return a.records
}

func (a *recordAppender) RecordsForID(id common.ID) []*common.Record {
	_, i, _ := common.Records(a.records).Find(context.Background(), id)
	if i >= len(a.records) || i < 0 {
		return nil
	}

	sliceRecords := make([]*common.Record, 0, 1)
	for bytes.Equal(a.records[i].ID, id) {
		sliceRecords = append(sliceRecords, a.records[i])

		i++
		if i >= len(a.records) {
			break
		}
	}

	return sliceRecords
}

func (a *recordAppender) Length() int {
	return len(a.records)
}

func (a *recordAppender) DataLength() uint64 {
	return 0
}

func (a *recordAppender) Complete() error {
	return nil
}
