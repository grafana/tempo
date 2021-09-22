package v2

import (
	"bytes"
	"io"

	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

type dataWriter struct {
	outputWriter io.Writer

	pool              WriterPool
	compressedBuffer  *bytes.Buffer
	compressionWriter io.WriteCloser

	objectRW     common.ObjectReaderWriter
	objectBuffer *bytes.Buffer
}

// NewDataWriter creates a paged page writer
func NewDataWriter(writer io.Writer, encoding backend.Encoding) (common.DataWriter, error) {
	pool, err := GetWriterPool(encoding)
	if err != nil {
		return nil, err
	}

	compressedBuffer := &bytes.Buffer{}
	compressionWriter, err := pool.GetWriter(compressedBuffer)
	if err != nil {
		return nil, err
	}

	return &dataWriter{
		outputWriter:      writer,
		pool:              pool,
		compressionWriter: compressionWriter,
		compressedBuffer:  compressedBuffer,
		objectRW:          NewObjectReaderWriter(),
		objectBuffer:      &bytes.Buffer{},
	}, nil
}

// Write implements DataWriter
func (p *dataWriter) Write(id common.ID, obj []byte) (int, error) {
	return p.objectRW.MarshalObjectToWriter(id, obj, p.objectBuffer)
}

// CutPage implements DataWriter
func (p *dataWriter) CutPage() (int, error) {
	var err error
	// reset buffers for the write
	p.compressedBuffer.Reset()
	p.compressionWriter, err = p.pool.ResetWriter(p.compressedBuffer, p.compressionWriter)
	if err != nil {
		return 0, err
	}

	// compress the raw object buffer
	buffer := p.objectBuffer.Bytes()
	_, err = p.compressionWriter.Write(buffer)
	if err != nil {
		return 0, err
	}

	// force flush everything
	p.compressionWriter.Close()

	// now marshal the buffer as a page to the output
	bytesWritten, err := marshalPageToWriter(p.compressedBuffer.Bytes(), p.outputWriter, constDataHeader)
	if err != nil {
		return 0, err
	}

	// reset buffers for the next write
	p.objectBuffer.Reset()

	return bytesWritten, err
}

// Complete implements DataWriter
func (p *dataWriter) Complete() error {
	if p.compressionWriter != nil {
		p.pool.PutWriter(p.compressionWriter)
		p.compressionWriter = nil
	}

	return nil
}
