package v2

import (
	"bytes"
	"io"

	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
	v1 "github.com/grafana/tempo/tempodb/encoding/v1"
)

type dataWriter struct {
	v1Writer common.DataWriter
	v1Buffer *bytes.Buffer

	outputWriter io.Writer
}

// NewDataWriter creates a paged page writer
func NewDataWriter(writer io.Writer, encoding backend.Encoding) (common.DataWriter, error) {
	v1Buffer := &bytes.Buffer{}
	v1Writer, err := v1.NewDataWriter(v1Buffer, encoding)
	if err != nil {
		return nil, err
	}

	return &dataWriter{
		v1Writer:     v1Writer,
		v1Buffer:     v1Buffer,
		outputWriter: writer,
	}, nil
}

// Write implements DataWriter
func (p *dataWriter) Write(id common.ID, obj []byte) (int, error) {
	return p.v1Writer.Write(id, obj)
}

// CutPage implements DataWriter
func (p *dataWriter) CutPage() (int, error) {
	_, err := p.v1Writer.CutPage()
	if err != nil {
		return 0, err
	}

	// v1Buffer currently has all of the v1 bytes. let's wrap it in our page and write
	bytesWritten, err := marshalPageToWriter(p.v1Buffer.Bytes(), p.outputWriter, constDataHeader)
	if err != nil {
		return 0, err
	}
	p.v1Buffer.Reset()

	return bytesWritten, err
}

// Complete implements DataWriter
func (p *dataWriter) Complete() error {
	return p.v1Writer.Complete()
}
