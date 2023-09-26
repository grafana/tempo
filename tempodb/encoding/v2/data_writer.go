package v2

import (
	"bytes"
	"fmt"
	"io"

	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

type dataWriter struct {
	outputWriter io.Writer

	pool              WriterPool
	compressedBuffer  *bytes.Buffer
	compressionWriter io.WriteCloser

	objectRW     ObjectReaderWriter
	objectBuffer *bytes.Buffer
}

// NewDataWriter creates a paged page writer
func NewDataWriter(writer io.Writer, encoding backend.Encoding) (DataWriter, error) {
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
	// compress the raw object buffer
	buffer := p.objectBuffer.Bytes()
	_, err := p.compressionWriter.Write(buffer)
	if err != nil {
		return 0, err
	}

	// force flush everything
	p.compressionWriter.Close()

	// now marshal the buffer as a page to the output
	bytesWritten, marshalErr := marshalPageToWriter(p.compressedBuffer.Bytes(), p.outputWriter, constDataHeader)

	// reset buffers for the next write
	p.objectBuffer.Reset()
	p.compressedBuffer.Reset()
	p.compressionWriter, err = p.pool.ResetWriter(p.compressedBuffer, p.compressionWriter)
	if err != nil {
		return 0, err
	}

	// deliberately checking marshalErr after resetting the compression writer to avoid "writer is closed" errors in
	// case of issues while writing to disk
	// for more details hop on to https://github.com/grafana/tempo/issues/1374
	if marshalErr != nil {
		return 0, fmt.Errorf("error marshalling page to writer: %w", marshalErr)
	}

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
