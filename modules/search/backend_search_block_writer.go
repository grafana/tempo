package search

import (
	"context"

	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/google/uuid"
	"github.com/grafana/tempo/pkg/tempofb"
	"github.com/grafana/tempo/tempodb/backend"
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
	builder     *flatbuffers.Builder
	pageEntries []flatbuffers.UOffsetT
	pageBuf     []byte
	tracker     backend.AppendTracker
}

var _ common.DataWriterGeneric = (*backendSearchBlockWriter)(nil)

/*
// snappy
func defaultEncode(dst []byte, src []byte) []byte {
	dst = dst[:0]
	return snappy.Encode(dst, src)
}
func defaultDecode(dst []byte, src []byte) ([]byte, error) {
	dst = dst[:0]
	return snappy.Decode(dst, src)
}
*/

// none
func defaultEncode(dst, src []byte) []byte {
	return src
}
func defaultDecode(dst, src []byte) ([]byte, error) {
	return src, nil
}

func NewBackendSearchBbackendSearchBlockWriter(blockID uuid.UUID, tenantID string, w backend.RawWriter) *backendSearchBlockWriter {
	return &backendSearchBlockWriter{
		blockID:  blockID,
		tenantID: tenantID,
		w:        w,

		pageBuf: make([]byte, 0, 1024*1024),
		builder: flatbuffers.NewBuilder(1024),
	}
}

// Write the data to the flatbuffer builder. Input must be a SearchDataMutable. Returns
// the number of bytes written, which is determined from the current object in the builder.
func (w *backendSearchBlockWriter) Write(ctx context.Context, ID common.ID, i interface{}) (int, error) {
	oldOffset := w.builder.Offset()

	data := i.(*tempofb.SearchDataMutable)
	offset := data.WriteToBuilder(w.builder)
	w.pageEntries = append(w.pageEntries, offset)

	return int(offset - oldOffset), nil
}

func (w *backendSearchBlockWriter) CutPage(ctx context.Context) (int, error) {

	// At this point all individual search entries have been written
	// to the fb builder. Now we need to wrap them up in the final
	// batch object.

	// Create vector
	tempofb.BatchSearchDataStartEntriesVector(w.builder, len(w.pageEntries))
	for _, entry := range w.pageEntries {
		w.builder.PrependUOffsetT(entry)
	}
	entryVector := w.builder.EndVector(len(w.pageEntries))

	// Write final batch object
	tempofb.BatchSearchDataStart(w.builder)
	tempofb.BatchSearchDataAddEntries(w.builder, entryVector)
	batch := tempofb.BatchSearchDataEnd(w.builder)
	w.builder.Finish(batch)
	buf := w.builder.FinishedBytes()

	w.pageBuf = defaultEncode(w.pageBuf, buf)

	var err error
	w.tracker, err = w.w.Append(ctx, "search", backend.KeyPathForBlock(w.blockID, w.tenantID), w.tracker, w.pageBuf)
	if err != nil {
		return 0, err
	}

	bytesFlushed := len(w.pageBuf)

	// Reset for next page
	w.builder.Reset()
	w.pageEntries = w.pageEntries[:0]

	return bytesFlushed, nil
}

func (w *backendSearchBlockWriter) Complete(ctx context.Context) error {
	return w.w.CloseAppend(ctx, w.tracker)
}
