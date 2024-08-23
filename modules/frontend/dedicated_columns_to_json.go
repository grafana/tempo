package frontend

import (
	"encoding/json"
	"unsafe"

	"github.com/grafana/tempo/tempodb/backend"
)

type DedicatedColumnsToJSON struct {
	columnsToJSON map[uint64]string
}

func NewDedicatedColumnsToJSON() *DedicatedColumnsToJSON {
	return &DedicatedColumnsToJSON{
		columnsToJSON: make(map[uint64]string),
	}
}

func (d *DedicatedColumnsToJSON) JSONForDedicatedColumns(cols backend.DedicatedColumns) (string, error) {
	if len(cols) == 0 {
		return "", nil // jpe - api.Build needs to handle empty string
	}

	hash := cols.Hash()
	if jsonString, ok := d.columnsToJSON[hash]; ok {
		return jsonString, nil
	}

	proto, err := cols.ToTempopb()
	if err != nil {
		return "", err
	}

	jsonBytes, err := json.Marshal(proto)
	if err != nil {
		return "", err
	}

	jsonString := unsafe.String(unsafe.SliceData(jsonBytes), len(jsonBytes))
	d.columnsToJSON[hash] = jsonString

	return jsonString, nil
}
