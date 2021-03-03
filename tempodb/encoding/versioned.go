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
	newPageWriter(writer io.Writer, encoding backend.Encoding) (common.PageWriter, error)
	newIndexWriter() common.IndexWriter

	newPageReader(ra backend.ContextReader, encoding backend.Encoding) (common.PageReader, error)
	newIndexReader(ra backend.ContextReader) (common.IndexReader, error)
}

// v0Encoding
type v0Encoding struct{}

func (v v0Encoding) newIndexWriter() common.IndexWriter {
	return v0.NewIndexWriter()
}
func (v v0Encoding) newPageWriter(writer io.Writer, encoding backend.Encoding) (common.PageWriter, error) {
	return v0.NewPageWriter(writer), nil // ignore encoding.  v0 PageWriter writes raw bytes
}
func (v v0Encoding) newIndexReader(ra backend.ContextReader) (common.IndexReader, error) {
	return v0.NewIndexReader(ra)
}
func (v v0Encoding) newPageReader(ra backend.ContextReader, encoding backend.Encoding) (common.PageReader, error) {
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
func (v v1Encoding) newPageReader(ra backend.ContextReader, encoding backend.Encoding) (common.PageReader, error) {
	return v1.NewPageReader(v0.NewPageReader(ra), encoding)
}
func (v v1Encoding) newIndexReader(ra backend.ContextReader) (common.IndexReader, error) {
	return v1.NewIndexReader(ra)
}

// v2Encoding
type v2Encoding struct{}

func (v v2Encoding) newIndexWriter() common.IndexWriter {
	return v2.NewIndexWriter()
}
func (v v2Encoding) newPageWriter(writer io.Writer, encoding backend.Encoding) (common.PageWriter, error) {
	return v2.NewPageWriter(writer, encoding)
}
func (v v2Encoding) newPageReader(ra backend.ContextReader, encoding backend.Encoding) (common.PageReader, error) {
	return v2.NewPageReader(v0.NewPageReader(ra), encoding)
}
func (v v2Encoding) newIndexReader(ra backend.ContextReader) (common.IndexReader, error) {
	return v2.NewIndexReader(ra)
}
