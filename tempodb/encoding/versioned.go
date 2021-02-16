package encoding

import (
	"io"

	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
	v0 "github.com/grafana/tempo/tempodb/encoding/v0"
	v1 "github.com/grafana/tempo/tempodb/encoding/v1"
)

const currentVersion = "v1"

// versionedEncoding has a whole bunch of versioned functionality.  This is
//  currently quite sloppy and could easily be tightened up to just a few methods
//  but it is what it is for now!
type versionedEncoding interface {
	newBufferedAppender(writer io.Writer, encoding backend.Encoding, indexDownsample int, totalObjectsEstimate int) (common.Appender, error)
	newPagedFinder(indexReader common.IndexReader, pageReader common.PageReader, combiner common.ObjectCombiner) common.Finder
	newPagedIterator(chunkSizeBytes uint32, indexBytes []byte, pageReader common.PageReader) (common.Iterator, error)

	newPageReader(ra io.ReaderAt, encoding backend.Encoding) (common.PageReader, error)
	newIndexReader(indexBytes []byte) (common.IndexReader, error)
}

// latestEncoding is used by Compactor and Complete block
func latestEncoding() versionedEncoding {
	return v1Encoding{}
}

// v0Encoding
type v0Encoding struct{}

func (v v0Encoding) newBufferedAppender(writer io.Writer, _ backend.Encoding, indexDownsample int, totalObjectsEstimate int) (common.Appender, error) {
	return v0.NewBufferedAppender(writer, indexDownsample, totalObjectsEstimate), nil
}
func (v v0Encoding) newPagedIterator(chunkSizeBytes uint32, indexBytes []byte, pageReader common.PageReader) (common.Iterator, error) {
	reader, err := v0.NewIndexReader(indexBytes)
	if err != nil {
		return nil, err
	}

	return v0.NewPagedIterator(chunkSizeBytes, reader, pageReader), nil
}
func (v v0Encoding) newPagedFinder(indexReader common.IndexReader, pageReader common.PageReader, combiner common.ObjectCombiner) common.Finder {
	return v0.NewPagedFinder(indexReader, pageReader, combiner)
}
func (v v0Encoding) newIndexReader(indexBytes []byte) (common.IndexReader, error) {
	return v0.NewIndexReader(indexBytes)
}
func (v v0Encoding) newPageReader(ra io.ReaderAt, encoding backend.Encoding) (common.PageReader, error) {
	return v0.NewPageReader(ra), nil
}

// v1Encoding
type v1Encoding struct{}

func (v v1Encoding) newBufferedAppender(writer io.Writer, encoding backend.Encoding, indexDownsample int, totalObjectsEstimate int) (common.Appender, error) {
	return v1.NewBufferedAppender(writer, encoding, indexDownsample, totalObjectsEstimate)
}
func (v v1Encoding) newPagedIterator(chunkSizeBytes uint32, indexBytes []byte, pageReader common.PageReader) (common.Iterator, error) {
	reader, err := v1.NewIndexReader(indexBytes)
	if err != nil {
		return nil, err
	}

	return v1.NewPagedIterator(chunkSizeBytes, reader, pageReader), nil
}
func (v v1Encoding) newPagedFinder(indexReader common.IndexReader, pageReader common.PageReader, combiner common.ObjectCombiner) common.Finder {
	return v1.NewPagedFinder(indexReader, pageReader, combiner)
}
func (v v1Encoding) newPageReader(ra io.ReaderAt, encoding backend.Encoding) (common.PageReader, error) {
	return v1.NewPageReader(ra, encoding)
}
func (v v1Encoding) newIndexReader(indexBytes []byte) (common.IndexReader, error) {
	return v1.NewIndexReader(indexBytes)
}
