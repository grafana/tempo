package vparquet4

import (
	"bytes"
	"encoding/json"
	"sort"

	"github.com/grafana/tempo/tempodb/encoding/common"
)

type index struct {
	lastID    common.ID
	dirty     bool
	RowGroups []common.ID `json:"rowGroups"`
}

func (i *index) Add(id common.ID) {
	i.lastID = id
	i.dirty = true
}

func (i *index) Flush() {
	if i.dirty {
		i.RowGroups = append(i.RowGroups, i.lastID)
		i.dirty = false
	}
}

func (i *index) Marshal() ([]byte, error) {
	return json.Marshal(i)
}

func (i *index) Find(id common.ID) int {
	n := sort.Search(len(i.RowGroups), func(j int) bool {
		return bytes.Compare(id, i.RowGroups[j]) <= 0
	})
	if n >= len(i.RowGroups) {
		// Beyond the last row group. This is the only
		// area where presence can be ruled out.
		return -1
	}
	return n
}

func unmarshalIndex(b []byte) (*index, error) {
	i := &index{}
	return i, json.Unmarshal(b, i)
}
