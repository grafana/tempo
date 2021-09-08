package encoding

import (
	"fmt"
	"io"

	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
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
		v2Encoding{},
	}
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
