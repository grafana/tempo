package v2

import (
	"bytes"
	"io"

	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/klauspost/compress/zstd"
)

type dataWriter struct {
	outputWriter io.Writer

	pool                  WriterPool
	compressionWriter     io.WriteCloser
	encoding              backend.Encoding
	compressedBuffer      *bytes.Buffer
	otherCompressedBuffer []byte

	objectRW     common.ObjectReaderWriter
	objectBuffer *bytes.Buffer
}

// NewDataWriter creates a paged page writer
func NewDataWriter(writer io.Writer, encoding backend.Encoding) (common.DataWriter, error) {
	pool, err := GetWriterPool(encoding)
	if err != nil {
		return nil, err
	}

	var compressedBuffer *bytes.Buffer
	if encoding != backend.EncZstd {
		compressedBuffer = &bytes.Buffer{}
	}
	compressionWriter, err := pool.GetWriter(compressedBuffer)
	if err != nil {
		return nil, err
	}

	return &dataWriter{
		outputWriter:      writer,
		pool:              pool,
		compressionWriter: compressionWriter,
		compressedBuffer:  compressedBuffer,
		encoding:          encoding,
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

	var uncompressedBytes []byte
	zstdEncoder, ok := p.compressionWriter.(*zstd.Encoder)
	if ok {
		p.otherCompressedBuffer = zstdEncoder.EncodeAll(buffer, p.otherCompressedBuffer[:0])
		uncompressedBytes = p.otherCompressedBuffer
	} else {
		_, err := p.compressionWriter.Write(buffer)
		if err != nil {
			return 0, err
		}

		// force flush everything
		p.compressionWriter.Close()

		uncompressedBytes = p.compressedBuffer.Bytes()
	}

	// now marshal the buffer as a page to the output
	bytesWritten, err := marshalPageToWriter(uncompressedBytes, p.outputWriter, constDataHeader)
	if err != nil {
		return 0, err
	}

	// reset buffers for the next write
	p.objectBuffer.Reset()
	if !ok {
		p.compressedBuffer.Reset()
		p.compressionWriter, err = p.pool.ResetWriter(p.compressedBuffer, p.compressionWriter)
		if err != nil {
			return 0, err
		}
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
