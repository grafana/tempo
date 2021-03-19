package v2

import (
	"bytes"
	"sort"

	"github.com/grafana/tempo/tempodb/encoding/common"
	v0 "github.com/grafana/tempo/tempodb/encoding/v0"
)

func NewRecordReaderWriter() common.RecordReaderWriter {
	return v0.NewRecordReaderWriter()
}

type recordSorter struct {
	records []*common.Record
}

// sortRecords sorts a slice of record pointers
func sortRecords(records []*common.Record) {
	sort.Sort(&recordSorter{
		records: records,
	})
}

func (t *recordSorter) Len() int {
	return len(t.records)
}

func (t *recordSorter) Less(i, j int) bool {
	a := t.records[i]
	b := t.records[j]

	return bytes.Compare(a.ID, b.ID) == -1
}

func (t *recordSorter) Swap(i, j int) {
	t.records[i], t.records[j] = t.records[j], t.records[i]
}
