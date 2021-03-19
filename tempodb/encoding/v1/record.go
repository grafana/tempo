package v1

import (
	"github.com/grafana/tempo/tempodb/encoding/common"
	v0 "github.com/grafana/tempo/tempodb/encoding/v0"
)

func NewRecordReaderWriter() common.RecordReaderWriter {
	return v0.NewRecordReaderWriter()
}
