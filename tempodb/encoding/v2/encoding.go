package v2

import (
	"io"

	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

const versionString = "v2"

// v2Encoding
type Encoding struct{}

func (v Encoding) Version() string {
	return versionString
}
func (v Encoding) NewIndexWriter(pageSizeBytes int) common.IndexWriter {
	return NewIndexWriter(pageSizeBytes)
}
func (v Encoding) NewDataWriter(writer io.Writer, encoding backend.Encoding) (common.DataWriter, error) {
	return NewDataWriter(writer, encoding)
}
func (v Encoding) NewIndexReader(ra backend.ContextReader, pageSizeBytes int, totalPages int) (common.IndexReader, error) {
	return NewIndexReader(ra, pageSizeBytes, totalPages)
}
func (v Encoding) NewDataReader(ra backend.ContextReader, encoding backend.Encoding) (common.DataReader, error) {
	return NewDataReader(ra, encoding)
}
func (v Encoding) NewObjectReaderWriter() common.ObjectReaderWriter {
	return NewObjectReaderWriter()
}
func (v Encoding) NewCompactor() common.Compactor {
	return NewCompactor()
}
