package search

import (
	"bytes"
	"context"

	"github.com/google/uuid"
	"github.com/grafana/tempo/pkg/tempofb"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

// backendSearchBlockWriter is a DataWriter for search data. Instead of receiving bytes slices, it
// receives search data objects and combintes them into a single FlatBuffer Builder and
// flushes periodically, one page per flush.
type backendSearchBlockWriter struct {
	// input
	blockID  uuid.UUID
	tenantID string
	w        backend.RawWriter

	// vars
	builder  *tempofb.SearchPageBuilder
	pageBuf  []byte
	tracker  backend.AppendTracker
	finalBuf *bytes.Buffer
	dw       common.DataWriter
}

var _ common.DataWriterGeneric = (*backendSearchBlockWriter)(nil)

func newBackendSearchBlockWriter(blockID uuid.UUID, tenantID string, w backend.RawWriter, v encoding.VersionedEncoding, enc backend.Encoding) (*backendSearchBlockWriter, error) {
	finalBuf := &bytes.Buffer{}

	dw, err := v.NewDataWriter(finalBuf, enc)
	if err != nil {
		return nil, err
	}

	return &backendSearchBlockWriter{
		blockID:  blockID,
		tenantID: tenantID,
		w:        w,

		pageBuf:  make([]byte, 0, 1024*1024),
		finalBuf: finalBuf,
		builder:  tempofb.NewSearchPageBuilder(),
		dw:       dw,
	}, nil
}

// Write the data to the flatbuffer builder. Input must be a SearchEntryMutable. Returns
// the number of bytes written, which is determined from the current object in the builder.
func (w *backendSearchBlockWriter) Write(ctx context.Context, ID common.ID, i interface{}) (int, error) {
	data := i.(*tempofb.SearchEntryMutable)
	bytesWritten := w.builder.AddData(data)
	return bytesWritten, nil
}

func (w *backendSearchBlockWriter) CutPage(ctx context.Context) (int, error) {

	// Finish fb page
	buf := w.builder.Finish()

	// Write to data writer and cut which will encode/compress
	w.finalBuf.Reset()
	_, err := w.dw.Write(uuid.Nil[:], buf)
	if err != nil {
		return 0, err
	}

	_, err = w.dw.CutPage()
	if err != nil {
		return 0, err
	}

	w.pageBuf = w.finalBuf.Bytes()

	// Append to backend
	w.tracker, err = w.w.Append(ctx, "search", backend.KeyPathForBlock(w.blockID, w.tenantID), w.tracker, w.pageBuf)
	if err != nil {
		return 0, err
	}

	bytesFlushed := len(w.pageBuf)

	// Reset for next page
	w.builder.Reset()

	return bytesFlushed, nil
}

func (w *backendSearchBlockWriter) Complete(ctx context.Context) error {
	err := w.dw.Complete()
	if err != nil {
		return err
	}

	return w.w.CloseAppend(ctx, w.tracker)
}
