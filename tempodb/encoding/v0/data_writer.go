package v0

import (
	"io"

	"github.com/grafana/tempo/tempodb/encoding/base"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

type dataWriter struct {
	w            io.Writer
	bytesWritten int
}

// NewDataWriter creates a v0 page writer.  This page writer
// writes raw bytes only
func NewDataWriter(writer io.Writer) common.DataWriter {
	return &dataWriter{
		w: writer,
	}
}

// Write implements DataWriter
func (p *dataWriter) Write(id common.ID, obj []byte) (int, error) {
	written, err := base.MarshalObjectToWriter(id, obj, p.w)
	if err != nil {
		return 0, err
	}

	p.bytesWritten += written
	return written, nil
}

// CutPage implements DataWriter
func (p *dataWriter) CutPage() (int, error) {
	ret := p.bytesWritten
	p.bytesWritten = 0

	return ret, nil
}

// Complete implements DataWriter
func (p *dataWriter) Complete() error {
	return nil
}
