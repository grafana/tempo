package v0

import (
	"io"

	"github.com/grafana/tempo/tempodb/encoding/base"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

type pageWriter struct {
	w            io.Writer
	bytesWritten int
}

// NewPageWriter creates a v0 page writer.  This page writer
// writes raw bytes only
func NewPageWriter(writer io.Writer) common.PageWriter {
	return &pageWriter{
		w: writer,
	}
}

// Write implements common.PageWriter
func (p *pageWriter) Write(id common.ID, obj []byte) (int, error) {
	written, err := base.MarshalObjectToWriter(id, obj, p.w)
	if err != nil {
		return 0, err
	}

	p.bytesWritten += written
	return written, nil
}

// CutPage implements common.PageWriter
func (p *pageWriter) CutPage() (int, error) {
	ret := p.bytesWritten
	p.bytesWritten = 0

	return ret, nil
}

// Complete implements common.PageWriter
func (p *pageWriter) Complete() error {
	return nil
}
