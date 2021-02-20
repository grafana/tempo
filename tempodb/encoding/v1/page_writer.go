package v1

import (
	"bytes"
	"io"

	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
	v0 "github.com/grafana/tempo/tempodb/encoding/v0"
)

// meteredWriter is a struct that is used to count the number of bytes
// written to a block after compression.  Unfortunately the compression io.Reader
// returns bytes before compression so this is necessary to know the actual number of
// byte written.
type meteredWriter struct {
	wrappedWriter io.Writer
	bytesWritten  int
}

func (m *meteredWriter) Write(p []byte) (n int, err error) {
	m.bytesWritten += len(p)
	return m.wrappedWriter.Write(p)
}

type pageWriter struct {
	v0Buffer     *bytes.Buffer
	outputWriter *meteredWriter

	pool              WriterPool
	compressionWriter io.WriteCloser
}

// NewPageWriter creates a v0 page writer.  This page writer
// writes raw bytes only
func NewPageWriter(writer io.Writer, encoding backend.Encoding) (common.PageWriter, error) {
	pool, err := GetWriterPool(encoding)
	if err != nil {
		return nil, err
	}

	outputWriter := &meteredWriter{
		wrappedWriter: writer,
	}

	compressionWriter, err := pool.GetWriter(outputWriter)
	if err != nil {
		return nil, err
	}

	return &pageWriter{
		v0Buffer:          &bytes.Buffer{},
		outputWriter:      outputWriter,
		pool:              pool,
		compressionWriter: compressionWriter,
	}, nil
}

// Write implements common.PageWriter
func (p *pageWriter) Write(id common.ID, obj []byte) (int, error) {
	return v0.MarshalObjectToWriter(id, obj, p.v0Buffer)
}

// CutPage implements common.PageWriter
func (p *pageWriter) CutPage() (int, error) {
	var err error
	p.compressionWriter, err = p.pool.ResetWriter(p.outputWriter, p.compressionWriter)

	buffer := p.v0Buffer.Bytes()
	_, err = p.compressionWriter.Write(buffer)
	if err != nil {
		return 0, err
	}

	// now clear our v0 buffer so we can start the new block page
	p.compressionWriter.Close()
	p.v0Buffer.Reset()

	bytesWritten := p.outputWriter.bytesWritten
	p.outputWriter.bytesWritten = 0

	return bytesWritten, nil
}

// Complete implements common.PageWriter
func (p *pageWriter) Complete() error {
	if p.compressionWriter != nil {
		p.pool.PutWriter(p.compressionWriter)
		p.compressionWriter = nil
	}

	return nil
}
