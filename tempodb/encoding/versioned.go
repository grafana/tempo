package encoding

import (
	"io"

	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
	v0 "github.com/grafana/tempo/tempodb/encoding/v0"
	v1 "github.com/grafana/tempo/tempodb/encoding/v1"
)

const currentVersion = "v1"

// latestEncoding is used by Compactor and Complete block
func latestEncoding() versionedEncoding {
	return v1Encoding{}
}

// allEncodings returns all encodings
func allEncodings() []versionedEncoding {
	return []versionedEncoding{
		v0Encoding{},
		v1Encoding{},
	}
}

// versionedEncoding has a whole bunch of versioned functionality.  This is
//  currently quite sloppy and could easily be tightened up to just a few methods
//  but it is what it is for now!
type versionedEncoding interface {
	newPageWriter(writer io.Writer, encoding backend.Encoding) (common.PageWriter, error)
	newIndexWriter() common.IndexWriter

	newPageReader(ra backend.ReaderAtContext, encoding backend.Encoding) (common.PageReader, error)
	newIndexReader(indexBytes []byte) (common.IndexReader, error) // jpe indexBytes => readerAtContextThing.  Needs plain ol' read function to
}

// v0Encoding
type v0Encoding struct{}

func (v v0Encoding) newIndexWriter() common.IndexWriter {
	return v0.NewIndexWriter()
}
func (v v0Encoding) newPageWriter(writer io.Writer, encoding backend.Encoding) (common.PageWriter, error) {
	return v0.NewPageWriter(writer), nil // ignore encoding.  v0 PageWriter writes raw bytes
}
func (v v0Encoding) newIndexReader(indexBytes []byte) (common.IndexReader, error) {
	return v0.NewIndexReader(indexBytes)
}
func (v v0Encoding) newPageReader(ra backend.ReaderAtContext, encoding backend.Encoding) (common.PageReader, error) {
	return v0.NewPageReader(ra), nil
}

// v1Encoding
type v1Encoding struct{}

func (v v1Encoding) newIndexWriter() common.IndexWriter {
	return v1.NewIndexWriter()
}
func (v v1Encoding) newPageWriter(writer io.Writer, encoding backend.Encoding) (common.PageWriter, error) {
	return v1.NewPageWriter(writer, encoding)
}
func (v v1Encoding) newPageReader(ra backend.ReaderAtContext, encoding backend.Encoding) (common.PageReader, error) {
	return v1.NewPageReader(ra, encoding)
}
func (v v1Encoding) newIndexReader(indexBytes []byte) (common.IndexReader, error) {
	return v1.NewIndexReader(indexBytes)
}
