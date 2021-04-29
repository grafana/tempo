package encoding

import (
	"github.com/grafana/tempo/tempodb/encoding/common"
)

type recordAppender struct {
	records []common.Record
}

// NewRecordAppender returns an appender that stores records only.
func NewRecordAppender(records []common.Record) Appender {
	return &recordAppender{
		records: records,
	}
}

// Append appends the id/object to the writer.  Note that the caller is giving up ownership of the two byte arrays backing the slices.
//   Copies should be made and passed in if this is a problem
func (a *recordAppender) Append(id common.ID, b []byte) error {
	return common.ErrUnsupported
}

func (a *recordAppender) IndexReader() common.IndexReader {
	return common.Records(a.records)
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
