package v2

import (
	"github.com/grafana/tempo/tempodb/encoding/common"
	v0 "github.com/grafana/tempo/tempodb/encoding/v0"
)

func NewObjectReaderWriter() common.ObjectReaderWriter {
	return v0.NewObjectReaderWriter()
}
