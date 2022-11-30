package vparquet

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/segmentio/parquet-go"

	"github.com/grafana/tempo/pkg/model"
	"github.com/grafana/tempo/pkg/model/trace"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/traceql"
	"github.com/grafana/tempo/pkg/warnings"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

var _ common.WALBlock = (*walBlock)(nil)

// path + filename = folder to create
//   path/folder/00001
//   	        /00002
//              /00003
//              /00004

// folder = <blockID>+<tenantID>+vParquet

// openWALBlock opens an existing appendable block.  It is read-only by
// not assigning a decoder.
func openWALBlock(filename string, path string, ingestionSlack time.Duration, _ time.Duration) (common.WALBlock, error, error) { // jpe what returns a warning?
	dir := filepath.Join(path, filename)
	_, _, version, err := parseName(filename)
	if err != nil {
		return nil, nil, err
	}

	if version != VersionString {
		return nil, nil, fmt.Errorf("mismatched version in vparquet wal: %s, %s, %s", version, path, filename)
	}

	metaPath := filepath.Join(dir, backend.MetaName)
	metaBytes, err := os.ReadFile(metaPath)
	if err != nil {
		return nil, nil, fmt.Errorf("error reading wal meta json: %s %w", metaPath, err)
	}

	meta := &backend.BlockMeta{}
	err = json.Unmarshal(metaBytes, meta)
	if err != nil {
		return nil, nil, fmt.Errorf("error unmarshaling wal meta json: %s %w", metaPath, err)
	}

	b := &walBlock{
		meta:           meta,
		path:           path,
		ids:            common.NewIDMap[int64](),
		ingestionSlack: ingestionSlack,
	}

	// read all files in dir
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil, fmt.Errorf("error reading dir: %w", err)
	}

	var warning error
	for _, f := range files {
		if f.Name() == backend.MetaName {
			continue
		}

		// Ignore 0-byte files which are pages that were
		// opened but not flushed.
		i, err := f.Info()
		if err != nil {
			return nil, nil, fmt.Errorf("error getting file info: %s %w", f.Name(), err)
		}
		if i.Size() == 0 {
			continue
		}

		// attempt to load in a parquet.file
		pf, sz, err := openLocalParquetFile(filepath.Join(dir, f.Name()))
		if err != nil {
			warning = fmt.Errorf("error opening file: %w", err)
			continue
		}

		b.flushed = append(b.flushed, &walBlockFlush{
			file: pf,
			ids:  common.NewIDMap[int64](),
		})
		b.flushedSize += sz
	}

	// iterate through all files and build meta
	for i, page := range b.flushed {
		iter := makeIterFunc(context.Background(), page.file.RowGroups(), page.file)(columnPathTraceID, nil, columnPathTraceID)
		defer iter.Close()

		for {
			match, err := iter.Next()
			if err != nil {
				return nil, nil, fmt.Errorf("error opening wal folder [%s %d]: %w", b.meta.BlockID.String(), i, err)
			}
			if match == nil {
				break
			}

			for _, e := range match.Entries {
				switch e.Key {
				case columnPathTraceID:
					traceID := e.Value.ByteArray()
					b.meta.ObjectAdded(traceID, 0, 0)
					page.ids.Set(traceID, match.RowNumber[0]) // Save rownumber for the trace ID
				}
			}
		}
	}

	return b, warning, nil
}

// createWALBlock creates a new appendable block
func createWALBlock(id uuid.UUID, tenantID string, filepath string, _ backend.Encoding, dataEncoding string, ingestionSlack time.Duration) (*walBlock, error) {
	b := &walBlock{
		meta: &backend.BlockMeta{
			Version:  VersionString,
			BlockID:  id,
			TenantID: tenantID,
		},
		path:           filepath,
		ids:            common.NewIDMap[int64](),
		ingestionSlack: ingestionSlack,
	}

	// build folder
	err := os.MkdirAll(b.walPath(), os.ModePerm)
	if err != nil {
		return nil, err
	}

	dec, err := model.NewObjectDecoder(dataEncoding)
	if err != nil {
		return nil, err
	}
	b.decoder = dec

	err = b.openWriter()

	return b, err
}

func ownsWALBlock(entry fs.DirEntry) bool {
	// all vParquet wal blocks are folders
	if !entry.IsDir() {
		return false
	}

	_, _, version, err := parseName(entry.Name())
	if err != nil {
		return false
	}

	return version == VersionString
}

type walBlockFlush struct {
	file *parquet.File
	ids  *common.IDMap[int64]
}

type walBlock struct {
	meta           *backend.BlockMeta
	path           string
	ingestionSlack time.Duration

	// Unflushed data
	buffer        *Trace
	ids           *common.IDMap[int64]
	file          *os.File
	writer        *parquet.GenericWriter[*Trace]
	decoder       model.ObjectDecoder
	unflushedSize int64

	// Flushed data
	flushed     []*walBlockFlush
	flushedSize int64
}

func (b *walBlock) BlockMeta() *backend.BlockMeta {
	return b.meta // jpe make ingestion slack a handled by BlockMeta
}

func (b *walBlock) Append(id common.ID, buff []byte, start, end uint32) error {
	// if decoder = nil we were created with OpenWALBlock and will not accept writes
	if b.decoder == nil {
		return nil
	}

	trace, err := b.decoder.PrepareForRead(buff)
	if err != nil {
		return fmt.Errorf("error preparing trace for read: %w", err)
	}

	b.buffer = traceToParquet(id, trace, b.buffer)

	start, end = b.adjustTimeRangeForSlack(start, end, 0)

	// add to current
	_, err = b.writer.Write([]*Trace{b.buffer})
	if err != nil {
		return fmt.Errorf("error writing row: %w", err)
	}

	b.meta.ObjectAdded(id, start, end)
	b.ids.Set(id, int64(b.ids.Len())) // Next row number

	// This is actually the protobuf size but close enough
	// for this purpose and only temporary until next flush.
	b.unflushedSize += int64(len(buff))

	return nil
}

func (b *walBlock) adjustTimeRangeForSlack(start uint32, end uint32, additionalStartSlack time.Duration) (uint32, uint32) {
	now := time.Now()
	startOfRange := uint32(now.Add(-b.ingestionSlack).Add(-additionalStartSlack).Unix())
	endOfRange := uint32(now.Add(b.ingestionSlack).Unix())

	warn := false
	if start < startOfRange {
		warn = true
		start = uint32(now.Unix())
	}
	if end > endOfRange {
		warn = true
		end = uint32(now.Unix())
	}

	if warn {
		warnings.Metric.WithLabelValues(b.meta.TenantID, warnings.ReasonOutsideIngestionSlack).Inc()
	}

	return start, end
}

func (b *walBlock) filepathOf(page int) string {
	filename := fmt.Sprintf("%010d", page)
	filename = filepath.Join(b.walPath(), filename)
	return filename
}

func (b *walBlock) openWriter() (err error) {

	nextFile := len(b.flushed) + 1
	filename := b.filepathOf(nextFile)

	b.file, err = os.OpenFile(filename, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("error opening file: %w", err)
	}

	if b.writer == nil {
		b.writer = parquet.NewGenericWriter[*Trace](b.file)
	} else {
		b.writer.Reset(b.file)
	}

	return nil
}

func (b *walBlock) Flush() (err error) {

	if b.ids.Len() == 0 {
		return nil
	}

	// Flush latest meta first
	// This mainly contains the slack-adjusted start/end times
	metaBytes, err := json.Marshal(b.BlockMeta())
	if err != nil {
		return fmt.Errorf("error marshaling meta json: %w", err)
	}

	metaPath := filepath.Join(b.walPath(), backend.MetaName)
	err = os.WriteFile(metaPath, metaBytes, 0600)
	if err != nil {
		return fmt.Errorf("error writing meta json: %w", err)
	}

	// Now flush/close current writer
	err = b.writer.Close()
	if err != nil {
		return fmt.Errorf("error closing writer: %w", err)
	}

	err = b.file.Close()
	if err != nil {
		return fmt.Errorf("error closing file: %w", err)
	}

	pf, sz, err := openLocalParquetFile(b.file.Name())
	if err != nil {
		return fmt.Errorf("error opening local file [%s]: %w", b.file.Name(), err)
	}

	b.flushed = append(b.flushed, &walBlockFlush{
		file: pf,
		ids:  b.ids,
	})
	b.flushedSize += sz
	b.unflushedSize = 0
	b.ids = common.NewIDMap[int64]()

	// Open next one
	return b.openWriter()
}

// DataLength returns estimated size of WAL files on disk. Used for
// cutting WAL files by max size.
func (b *walBlock) DataLength() uint64 {
	return uint64(b.flushedSize + b.unflushedSize)
}

func (b *walBlock) Iterator() (common.Iterator, error) {
	bookmarks := make([]*bookmark[*Trace], 0, len(b.flushed))
	for _, page := range b.flushed {

		r := parquet.NewGenericReader[*Trace](page.file)
		iter := &traceIterator{reader: r, rowNumbers: page.ids.ValuesSortedByID()}

		bookmarks = append(bookmarks, newBookmark[*Trace](iter))
	}

	iter := newMultiblockIterator(bookmarks, func(ts []*Trace) (*Trace, error) {
		t := CombineTraces(ts...)
		return t, nil
	})

	return &commonIterator{
		iter: iter,
	}, nil
}

func (b *walBlock) Clear() error {
	return os.RemoveAll(b.walPath())
}

// jpe what to do with common.SearchOptions?
func (b *walBlock) FindTraceByID(ctx context.Context, id common.ID, _ common.SearchOptions) (*tempopb.Trace, error) {
	trs := make([]*tempopb.Trace, 0)

	for _, page := range b.flushed {
		if rowNumber, ok := page.ids.Get(id); ok {
			r := parquet.NewReader(page.file)
			err := r.SeekToRow(rowNumber)
			if err != nil {
				return nil, errors.Wrap(err, "seek to row")
			}

			tr := new(Trace)
			err = r.Read(tr)
			if err != nil {
				return nil, errors.Wrap(err, "error reading row from backend")
			}

			trp := parquetTraceToTempopbTrace(tr)

			trs = append(trs, trp)
		}
	}

	combiner := trace.NewCombiner()
	for i, tr := range trs {
		combiner.ConsumeWithFinal(tr, i == len(trs)-1)
	}

	tr, _ := combiner.Result()
	return tr, nil
}

func (b *walBlock) Search(ctx context.Context, req *tempopb.SearchRequest, opts common.SearchOptions) (*tempopb.SearchResponse, error) {
	results := &tempopb.SearchResponse{
		Metrics: &tempopb.SearchMetrics{},
	}

	// jpe parrallelize?
	for i, page := range b.flushed {
		r, err := searchParquetFile(ctx, page.file, req, page.file.RowGroups())
		if err != nil {
			return nil, fmt.Errorf("error searching block [%s %d]: %w", b.meta.BlockID.String(), i, err)
		}

		results.Traces = append(results.Traces, r.Traces...)
		if len(results.Traces) >= int(req.Limit) {
			break
		}
	}

	results.Metrics.InspectedBlocks++
	results.Metrics.InspectedBytes += b.DataLength()
	results.Metrics.InspectedTraces += uint32(b.meta.TotalObjects)

	return results, nil
}

func (b *walBlock) SearchTags(ctx context.Context, cb common.TagCallback, opts common.SearchOptions) error {
	// jpe parallelize?
	for i, page := range b.flushed {
		err := searchTags(ctx, cb, page.file)
		if err != nil {
			return fmt.Errorf("error searching block [%s %d]: %w", b.meta.BlockID.String(), i, err)
		}
	}

	return nil
}

func (b *walBlock) SearchTagValues(ctx context.Context, tag string, cb common.TagCallback, opts common.SearchOptions) error {
	// jpe parallelize?
	for i, f := range b.flushed {
		err := searchTagValues(ctx, tag, cb, f.file)
		if err != nil {
			return fmt.Errorf("error searching block [%s %d]: %w", b.meta.BlockID.String(), i, err)
		}
	}

	return nil
}

func (b *walBlock) Fetch(ctx context.Context, req traceql.FetchSpansRequest, opts common.SearchOptions) (traceql.FetchSpansResponse, error) {
	// todo: this same method is called in backendBlock.Fetch. is there anyway to share this?
	err := checkConditions(req.Conditions)
	if err != nil {
		return traceql.FetchSpansResponse{}, errors.Wrap(err, "conditions invalid")
	}

	iters := make([]*spansetIterator, 0, len(b.flushed))
	for _, f := range b.flushed {
		iter, err := fetch(ctx, req, f.file)
		if err != nil {
			return traceql.FetchSpansResponse{}, errors.Wrap(err, "creating fetch iter")
		}
		iters = append(iters, iter)
	}

	// combine iters?
	return traceql.FetchSpansResponse{
		Results: &mergeSpansetIterator{
			iters: iters,
		},
	}, nil
}

func (b *walBlock) walPath() string {
	filename := fmt.Sprintf("%s+%s+%s", b.meta.BlockID, b.meta.TenantID, VersionString)
	return filepath.Join(b.path, filename)
}

// <blockID>+<tenantID>+vParquet
func parseName(filename string) (uuid.UUID, string, string, error) {
	splits := strings.Split(filename, "+")

	if len(splits) != 3 {
		return uuid.UUID{}, "", "", fmt.Errorf("unable to parse %s. unexpected number of segments", filename)
	}

	// first segment is blockID
	id, err := uuid.Parse(splits[0])
	if err != nil {
		return uuid.UUID{}, "", "", fmt.Errorf("unable to parse %s. error parsing uuid: %w", filename, err)
	}

	// second segment is tenant
	tenant := splits[1]
	if len(tenant) == 0 {
		return uuid.UUID{}, "", "", fmt.Errorf("unable to parse %s. 0 length tenant", filename)
	}

	// third segment is version
	version := splits[2]
	if version != VersionString {
		return uuid.UUID{}, "", "", fmt.Errorf("unable to parse %s. unexpected version %s", filename, version)
	}

	return id, tenant, version, nil
}

// jpe iterators feel like a mess, clean up ?

var tracePool sync.Pool

func tracePoolGet() *Trace {
	o := tracePool.Get()
	if o == nil {
		return &Trace{}
	}

	return o.(*Trace)
}

func tracePoolPut(t *Trace) {
	tracePool.Put(t)
}

// traceIterator is used to iterate a parquet file and implement iterIterator
// traces are iterated according to the given row numbers, because there is
// not a guarantee that the underlying parquet file is sorted
type traceIterator struct {
	reader     *parquet.GenericReader[*Trace]
	rowNumbers []int64
}

func (i *traceIterator) Next(ctx context.Context) (common.ID, *Trace, error) {
	if len(i.rowNumbers) == 0 {
		return nil, nil, nil
	}

	nextRowNumber := i.rowNumbers[0]
	i.rowNumbers = i.rowNumbers[1:]

	err := i.reader.SeekToRow(nextRowNumber)
	if err != nil {
		return nil, nil, err
	}

	tr := tracePoolGet()
	_, err = i.reader.Read([]*Trace{tr})
	if err != nil {
		return nil, nil, err
	}

	return tr.TraceID, tr, nil
}

func (i *traceIterator) Close() {
	i.reader.Close()
}

var _ TraceIterator = (*commonIterator)(nil)
var _ common.Iterator = (*commonIterator)(nil)

// commonIterator implements both TraceIterator and common.Iterator. it is returned from the AppendFile and is meant
// to be passed to a CreateBlock
type commonIterator struct {
	iter *MultiBlockIterator[*Trace]
}

func (i *commonIterator) Next(ctx context.Context) (common.ID, *tempopb.Trace, error) {
	id, obj, err := i.iter.Next(ctx)
	if err != nil && err != io.EOF {
		return nil, nil, err
	}

	if obj == nil || err == io.EOF {
		return nil, nil, nil
	}

	tr := parquetTraceToTempopbTrace(obj)
	return id, tr, nil
}

func (i *commonIterator) NextTrace(ctx context.Context) (common.ID, *Trace, error) {
	return i.iter.Next(ctx)
}

func (i *commonIterator) Close() {
	i.iter.Close()
}

func openLocalParquetFile(filename string) (*parquet.File, int64, error) {
	file, err := os.OpenFile(filename, os.O_RDONLY, 0644)
	if err != nil {
		return nil, 0, fmt.Errorf("error opening file: %w", err)
	}
	info, err := file.Stat()
	if err != nil {
		return nil, 0, fmt.Errorf("error getting file info: %w", err)
	}
	sz := info.Size()
	pf, err := parquet.OpenFile(file, sz)
	if err != nil {
		return nil, 0, fmt.Errorf("error opening parquet file: %w", err)
	}

	return pf, sz, nil
}
