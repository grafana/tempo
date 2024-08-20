package v2

import (
	"bytes"
	"context"

	"github.com/grafana/tempo/v2/pkg/util"
	"github.com/grafana/tempo/v2/tempodb/encoding/common"
)

type recordAppender struct {
	records []Record
}

// NewRecordAppender returns an appender that stores records only.
func NewRecordAppender(records []Record) Appender {
	return &recordAppender{
		records: records,
	}
}

// Append appends the id/object to the writer.  Note that the caller is giving up ownership of the two byte arrays backing the slices.
// Copies should be made and passed in if this is a problem
func (a *recordAppender) Append(common.ID, []byte) error {
	return util.ErrUnsupported
}

func (a *recordAppender) Records() []Record {
	return a.records
}

func (a *recordAppender) RecordsForID(id common.ID) []Record {
	_, i, _ := Records(a.records).Find(context.Background(), id)
	if i >= len(a.records) || i < 0 {
		return nil
	}

	sliceRecords := make([]Record, 0, 1)
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
	if len(a.records) == 0 {
		return 0
	}

	lastRecord := a.records[len(a.records)-1]
	return lastRecord.Start + uint64(lastRecord.Length)
}

func (a *recordAppender) Complete() error {
	return nil
}
