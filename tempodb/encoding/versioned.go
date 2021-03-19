package encoding

import (
	"io"

	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
	v0 "github.com/grafana/tempo/tempodb/encoding/v0"
	v1 "github.com/grafana/tempo/tempodb/encoding/v1"
	v2 "github.com/grafana/tempo/tempodb/encoding/v2"
)

const currentVersion = "v2"

// latestEncoding is used by Compactor and Complete block
func latestEncoding() versionedEncoding {
	return v2Encoding{}
}

// allEncodings returns all encodings
func allEncodings() []versionedEncoding {
	return []versionedEncoding{
		v0Encoding{},
		v1Encoding{},
		v2Encoding{},
	}
}

// versionedEncoding has a whole bunch of versioned functionality.  This is
//  currently quite sloppy and could easily be tightened up to just a few methods
//  but it is what it is for now!
type versionedEncoding interface {
	newDataWriter(writer io.Writer, encoding backend.Encoding) (common.DataWriter, error)
	newIndexWriter(pageSizeBytes int) common.IndexWriter

	newDataReader(ra backend.ContextReader, encoding backend.Encoding) (common.DataReader, error)
	newIndexReader(ra backend.ContextReader, pageSizeBytes int, totalPages int) (common.IndexReader, error)

	newObjectReaderWriter() common.ObjectReaderWriter
	newRecordReaderWriter() common.RecordReaderWriter
}

// v0Encoding
type v0Encoding struct{}

func (v v0Encoding) newIndexWriter(pageSizeBytes int) common.IndexWriter {
	return v0.NewIndexWriter()
}
func (v v0Encoding) newDataWriter(writer io.Writer, encoding backend.Encoding) (common.DataWriter, error) {
	return v0.NewDataWriter(writer), nil // ignore encoding.  v0 DataWriter writes raw bytes
}
func (v v0Encoding) newIndexReader(ra backend.ContextReader, pageSizeBytes int, totalPages int) (common.IndexReader, error) {
	return v0.NewIndexReader(ra)
}
func (v v0Encoding) newDataReader(ra backend.ContextReader, encoding backend.Encoding) (common.DataReader, error) {
	return v0.NewDataReader(ra), nil
}
func (v v0Encoding) newObjectReaderWriter() common.ObjectReaderWriter {
	return v0.NewObjectReaderWriter()
}
func (v v0Encoding) newRecordReaderWriter() common.RecordReaderWriter {
	return v0.NewRecordReaderWriter()
}

// v1Encoding
type v1Encoding struct{}

func (v v1Encoding) newIndexWriter(pageSizeBytes int) common.IndexWriter {
	return v1.NewIndexWriter()
}
func (v v1Encoding) newDataWriter(writer io.Writer, encoding backend.Encoding) (common.DataWriter, error) {
	return v1.NewDataWriter(writer, encoding)
}
func (v v1Encoding) newIndexReader(ra backend.ContextReader, pageSizeBytes int, totalPages int) (common.IndexReader, error) {
	return v1.NewIndexReader(ra)
}
func (v v1Encoding) newDataReader(ra backend.ContextReader, encoding backend.Encoding) (common.DataReader, error) {
	return v1.NewDataReader(v0.NewDataReader(ra), encoding)
}
func (v v1Encoding) newObjectReaderWriter() common.ObjectReaderWriter {
	return v1.NewObjectReaderWriter()
}
func (v v1Encoding) newRecordReaderWriter() common.RecordReaderWriter {
	return v1.NewRecordReaderWriter()
}

// v2Encoding
type v2Encoding struct{}

func (v v2Encoding) newIndexWriter(pageSizeBytes int) common.IndexWriter {
	return v2.NewIndexWriter(pageSizeBytes)
}
func (v v2Encoding) newDataWriter(writer io.Writer, encoding backend.Encoding) (common.DataWriter, error) {
	return v2.NewDataWriter(writer, encoding)
}
func (v v2Encoding) newIndexReader(ra backend.ContextReader, pageSizeBytes int, totalPages int) (common.IndexReader, error) {
	return v2.NewIndexReader(ra, pageSizeBytes, totalPages)
}
func (v v2Encoding) newDataReader(ra backend.ContextReader, encoding backend.Encoding) (common.DataReader, error) {
	return v2.NewDataReader(v0.NewDataReader(ra), encoding)
}
func (v v2Encoding) newObjectReaderWriter() common.ObjectReaderWriter {
	return v2.NewObjectReaderWriter()
}
func (v v2Encoding) newRecordReaderWriter() common.RecordReaderWriter {
	return v2.NewRecordReaderWriter()
}
