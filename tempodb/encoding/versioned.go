package encoding

import (
	"fmt"
	"io"

	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
	v0 "github.com/grafana/tempo/tempodb/encoding/v0"
	v1 "github.com/grafana/tempo/tempodb/encoding/v1"
	v2 "github.com/grafana/tempo/tempodb/encoding/v2"
)

const currentVersion = "v2"

// VersionedEncoding has a whole bunch of versioned functionality.  This is
//  currently quite sloppy and could easily be tightened up to just a few methods
//  but it is what it is for now!
type VersionedEncoding interface {
	Version() string

	NewDataWriter(writer io.Writer, encoding backend.Encoding) (common.DataWriter, error)
	NewIndexWriter(pageSizeBytes int) common.IndexWriter

	NewDataReader(ra backend.ContextReader, encoding backend.Encoding) (common.DataReader, error)
	NewIndexReader(ra backend.ContextReader, pageSizeBytes int, totalPages int) (common.IndexReader, error)

	NewObjectReaderWriter() common.ObjectReaderWriter
	NewRecordReaderWriter() common.RecordReaderWriter
}

// FromVersion returns a versioned encoding for the provided string
func FromVersion(v string) (VersionedEncoding, error) {
	switch v {
	case "v0":
		return v0Encoding{}, nil
	case "v1":
		return v1Encoding{}, nil
	case "v2":
		return v2Encoding{}, nil
	}

	return nil, fmt.Errorf("%s is not a valid block version", v)
}

// LatestEncoding is used by Compactor and Complete block
func LatestEncoding() VersionedEncoding {
	return v2Encoding{}
}

// allEncodings returns all encodings
func allEncodings() []VersionedEncoding {
	return []VersionedEncoding{
		v0Encoding{},
		v1Encoding{},
		v2Encoding{},
	}
}

// v0Encoding
type v0Encoding struct{}

func (v v0Encoding) Version() string {
	return "v0"
}
func (v v0Encoding) NewIndexWriter(pageSizeBytes int) common.IndexWriter {
	return v0.NewIndexWriter()
}
func (v v0Encoding) NewDataWriter(writer io.Writer, encoding backend.Encoding) (common.DataWriter, error) {
	return v0.NewDataWriter(writer), nil // ignore encoding.  v0 DataWriter writes raw bytes
}
func (v v0Encoding) NewIndexReader(ra backend.ContextReader, pageSizeBytes int, totalPages int) (common.IndexReader, error) {
	return v0.NewIndexReader(ra)
}
func (v v0Encoding) NewDataReader(ra backend.ContextReader, encoding backend.Encoding) (common.DataReader, error) {
	return v0.NewDataReader(ra), nil
}
func (v v0Encoding) NewObjectReaderWriter() common.ObjectReaderWriter {
	return v0.NewObjectReaderWriter()
}
func (v v0Encoding) NewRecordReaderWriter() common.RecordReaderWriter {
	return v0.NewRecordReaderWriter()
}

// v1Encoding
type v1Encoding struct{}

func (v v1Encoding) Version() string {
	return "v1"
}
func (v v1Encoding) NewIndexWriter(pageSizeBytes int) common.IndexWriter {
	return v1.NewIndexWriter()
}
func (v v1Encoding) NewDataWriter(writer io.Writer, encoding backend.Encoding) (common.DataWriter, error) {
	return v1.NewDataWriter(writer, encoding)
}
func (v v1Encoding) NewIndexReader(ra backend.ContextReader, pageSizeBytes int, totalPages int) (common.IndexReader, error) {
	return v1.NewIndexReader(ra)
}
func (v v1Encoding) NewDataReader(ra backend.ContextReader, encoding backend.Encoding) (common.DataReader, error) {
	return v1.NewDataReader(ra, encoding)
}
func (v v1Encoding) NewObjectReaderWriter() common.ObjectReaderWriter {
	return v1.NewObjectReaderWriter()
}
func (v v1Encoding) NewRecordReaderWriter() common.RecordReaderWriter {
	return v1.NewRecordReaderWriter()
}

// v2Encoding
type v2Encoding struct{}

func (v v2Encoding) Version() string {
	return "v2"
}
func (v v2Encoding) NewIndexWriter(pageSizeBytes int) common.IndexWriter {
	return v2.NewIndexWriter(pageSizeBytes)
}
func (v v2Encoding) NewDataWriter(writer io.Writer, encoding backend.Encoding) (common.DataWriter, error) {
	return v2.NewDataWriter(writer, encoding)
}
func (v v2Encoding) NewIndexReader(ra backend.ContextReader, pageSizeBytes int, totalPages int) (common.IndexReader, error) {
	return v2.NewIndexReader(ra, pageSizeBytes, totalPages)
}
func (v v2Encoding) NewDataReader(ra backend.ContextReader, encoding backend.Encoding) (common.DataReader, error) {
	return v2.NewDataReader(ra, encoding)
}
func (v v2Encoding) NewObjectReaderWriter() common.ObjectReaderWriter {
	return v2.NewObjectReaderWriter()
}
func (v v2Encoding) NewRecordReaderWriter() common.RecordReaderWriter {
	return v2.NewRecordReaderWriter()
}
