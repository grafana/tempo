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

	"github.com/grafana/dskit/multierror"
	"github.com/grafana/tempo/pkg/model"
	"github.com/grafana/tempo/pkg/model/trace"
	"github.com/grafana/tempo/pkg/parquetquery"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/traceql"
	"github.com/grafana/tempo/pkg/warnings"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

var _ common.WALBlock = (*walBlock)(nil)

// todo: this default size was very roughly tuned and likely should be based on config.
// likely the best candidate is some fraction of max trace size per tenant.
const defaultRowPoolSize = 100000

// completeBlockRowPool is used by the wal iterators and complete block logic to pool rows
var completeBlockRowPool = newRowPool(defaultRowPoolSize)

// walSchema is a shared schema that all wals use. it comes with minor cpu and memory improvements
var walSchema = parquet.SchemaOf(&Trace{})

// path + filename = folder to create
//   path/folder/00001
//   	        /00002
//              /00003
//              /00004

// folder = <blockID>+<tenantID>+vParquet

// openWALBlock opens an existing appendable block.  It is read-only by
// not assigning a decoder.
//
// there's an interesting bug here that does not come into play due to the fact that we do not append to a wal created with this method.
// if there are 2 wal files and the second is loaded successfully, but the first fails then b.flushed will contain one entry. then when
// calling b.openWriter() it will attempt to create a new file as path/folder/00002 which will overwrite the first file. as long as we never
// append to this file it should be ok.
func openWALBlock(filename string, path string, ingestionSlack time.Duration, _ time.Duration) (common.WALBlock, error, error) {
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

	// below we're going to iterate all of the parquet files in the wal and build the meta, this will correctly
	// recount total objects so clear them out here
	meta.TotalObjects = 0

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

		path := filepath.Join(dir, f.Name())
		page := newWalBlockFlush(path, common.NewIDMap[int64]())

		file, err := page.file()
		if err != nil {
			warning = fmt.Errorf("error opening file info: %s %w", page.path, err)
			continue
		}

		defer file.Close()
		pf := file.parquetFile

		// iterate the parquet file and build the meta
		iter := makeIterFunc(context.Background(), pf.RowGroups(), pf)(columnPathTraceID, nil, columnPathTraceID)
		defer iter.Close()

		for {
			match, err := iter.Next()
			if err != nil {
				return nil, nil, fmt.Errorf("error iterating wal page [%s %d]: %w", b.meta.BlockID.String(), i, err)
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

		b.flushed = append(b.flushed, page)
		b.flushedSize += i.Size()
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
	path string
	ids  *common.IDMap[int64]
}

func newWalBlockFlush(path string, ids *common.IDMap[int64]) *walBlockFlush {
	return &walBlockFlush{
		path: path,
		ids:  ids,
	}
}

// file() opens the parquet file and returns it. previously this method cached the file on first open
// but the memory cost of this was quite high. so instead we open it fresh every time
func (w *walBlockFlush) file() (*pageFile, error) {
	file, err := os.OpenFile(w.path, os.O_RDONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %w", err)
	}
	info, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("error getting file info: %w", err)
	}
	sz := info.Size()
	pf, err := parquet.OpenFile(file, sz, parquet.SkipBloomFilters(true), parquet.SkipPageIndex(true), parquet.FileSchema(walSchema))
	if err != nil {
		return nil, fmt.Errorf("error opening parquet file: %w", err)
	}

	f := &pageFile{parquetFile: pf, osFile: file}

	return f, nil

}

func (w *walBlockFlush) rowIterator() (*rowIterator, error) {
	file, err := w.file()
	if err != nil {
		return nil, err
	}

	pf := file.parquetFile

	idx, _ := parquetquery.GetColumnIndexByPath(pf, TraceIDColumnName)
	r := parquet.NewReader(pf)
	return newRowIterator(r, file, w.ids.EntriesSortedByID(), idx), nil
}

type pageFile struct {
	parquetFile *parquet.File
	osFile      *os.File
}

func (b *pageFile) Close() error {
	return b.osFile.Close()
}

type pageFileClosingIterator struct {
	iter     *spansetMetadataIterator
	pageFile *pageFile
}

var _ traceql.SpansetIterator = (*pageFileClosingIterator)(nil)

func (b *pageFileClosingIterator) Next(ctx context.Context) (*traceql.Spanset, error) {
	return b.iter.Next(ctx)
}

func (b *pageFileClosingIterator) Close() {
	b.iter.Close()
	b.pageFile.Close()
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
	mtx         sync.Mutex
}

func (b *walBlock) readFlushes() []*walBlockFlush {
	b.mtx.Lock()
	defer b.mtx.Unlock()
	return b.flushed
}

func (b *walBlock) writeFlush(f *walBlockFlush) {
	b.mtx.Lock()
	defer b.mtx.Unlock()
	b.flushed = append(b.flushed, f)
}

func (b *walBlock) BlockMeta() *backend.BlockMeta {
	return b.meta
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
		b.writer = parquet.NewGenericWriter[*Trace](b.file, &parquet.WriterConfig{
			Schema: walSchema,
			// setting this value low massively reduces the amount of static memory we hold onto in highly multi-tenant environments at the cost of
			// cutting pages more aggressively when writing column chunks
			PageBufferSize: 1024,
		})
	} else {
		b.writer.Reset(b.file)
	}

	return nil
}

func (b *walBlock) Flush() (err error) {

	if b.ids.Len() == 0 {
		return nil
	}

	b.buffer = nil

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

	info, err := b.file.Stat()
	if err != nil {
		return fmt.Errorf("error getting info: %w", err)
	}
	sz := info.Size()

	err = b.file.Close()
	if err != nil {
		return fmt.Errorf("error closing file: %w", err)
	}

	b.writeFlush(newWalBlockFlush(b.file.Name(), b.ids))
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
	bookmarks := make([]*bookmark[parquet.Row], 0, len(b.flushed))

	for _, page := range b.flushed {
		iter, err := page.rowIterator()
		if err != nil {
			return nil, fmt.Errorf("error creating iterator for %s: %w", page.path, err)
		}
		bookmarks = append(bookmarks, newBookmark[parquet.Row](iter))
	}

	sch := parquet.SchemaOf(new(Trace))
	iter := newMultiblockIterator(bookmarks, func(rows []parquet.Row) (parquet.Row, error) {
		if len(rows) == 1 {
			return rows[0], nil
		}

		ts := make([]*Trace, 0, len(rows))
		for _, row := range rows {
			tr := &Trace{}
			err := sch.Reconstruct(tr, row)
			if err != nil {
				return nil, err
			}
			ts = append(ts, tr)
			completeBlockRowPool.Put(row)
		}

		t := CombineTraces(ts...)
		row := completeBlockRowPool.Get()
		row = sch.Deconstruct(row, t)

		return row, nil
	})

	return newCommonIterator(iter, sch), nil
}

func (b *walBlock) Clear() error {
	var errs multierror.MultiError
	if b.file != nil {
		errClose := b.file.Close()
		errs.Add(errClose)
	}

	errRemoveAll := os.RemoveAll(b.walPath())
	errs.Add(errRemoveAll)

	return errs.Err()
}

func (b *walBlock) FindTraceByID(ctx context.Context, id common.ID, _ common.SearchOptions) (*tempopb.Trace, error) {
	trs := make([]*tempopb.Trace, 0)

	for _, page := range b.flushed {
		if rowNumber, ok := page.ids.Get(id); ok {
			file, err := page.file()
			if err != nil {
				return nil, fmt.Errorf("error opening file %s: %w", page.path, err)
			}

			defer file.Close()
			pf := file.parquetFile

			r := parquet.NewReader(pf)
			defer r.Close()

			err = r.SeekToRow(rowNumber)
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
		Metrics: &tempopb.SearchMetrics{
			InspectedBlocks: 1,
		},
	}

	for i, page := range b.readFlushes() {
		file, err := page.file()
		if err != nil {
			return nil, fmt.Errorf("error opening file %s: %w", page.path, err)
		}

		defer file.Close()
		pf := file.parquetFile

		r, err := searchParquetFile(ctx, pf, req, pf.RowGroups())
		if err != nil {
			return nil, fmt.Errorf("error searching block [%s %d]: %w", b.meta.BlockID.String(), i, err)
		}

		results.Traces = append(results.Traces, r.Traces...)
		results.Metrics.InspectedBytes += uint64(pf.Size())
		results.Metrics.InspectedTraces += uint32(pf.NumRows())
		if len(results.Traces) >= int(req.Limit) {
			break
		}
	}

	return results, nil
}

func (b *walBlock) SearchTags(ctx context.Context, cb common.TagCallback, opts common.SearchOptions) error {
	for i, page := range b.readFlushes() {
		file, err := page.file()
		if err != nil {
			return fmt.Errorf("error opening file %s: %w", page.path, err)
		}

		defer file.Close()
		pf := file.parquetFile

		err = searchTags(ctx, cb, pf)
		if err != nil {
			return fmt.Errorf("error searching block [%s %d]: %w", b.meta.BlockID.String(), i, err)
		}
	}

	return nil
}

func (b *walBlock) SearchTagValues(ctx context.Context, tag string, cb common.TagCallback, opts common.SearchOptions) error {
	att, ok := translateTagToAttribute[tag]
	if !ok {
		att = traceql.NewAttribute(tag)
	}

	// Wrap to v2-style
	cb2 := func(v traceql.Static) bool {
		cb(v.EncodeToString(false))
		return false
	}

	return b.SearchTagValuesV2(ctx, att, cb2, opts)
}

func (b *walBlock) SearchTagValuesV2(ctx context.Context, tag traceql.Attribute, cb common.TagCallbackV2, opts common.SearchOptions) error {
	for i, page := range b.readFlushes() {
		file, err := page.file()
		if err != nil {
			return fmt.Errorf("error opening file %s: %w", page.path, err)
		}

		defer file.Close()
		pf := file.parquetFile

		err = searchTagValues(ctx, tag, cb, pf)
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

	pages := b.readFlushes()
	iters := make([]traceql.SpansetIterator, 0, len(pages))
	for _, page := range pages {
		file, err := page.file()
		if err != nil {
			return traceql.FetchSpansResponse{}, fmt.Errorf("error opening file %s: %w", page.path, err)
		}

		pf := file.parquetFile

		iter, err := fetch(ctx, req, pf, opts)
		if err != nil {
			return traceql.FetchSpansResponse{}, errors.Wrap(err, "creating fetch iter")
		}

		wrappedIterator := &pageFileClosingIterator{iter: iter, pageFile: file}
		iters = append(iters, wrappedIterator)
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

// rowIterator is used to iterate a parquet file and implement iterIterator
// traces are iterated according to the given row numbers, because there is
// not a guarantee that the underlying parquet file is sorted
type rowIterator struct {
	reader       *parquet.Reader //nolint:all //deprecated
	pageFile     *pageFile
	rowNumbers   []common.IDMapEntry[int64]
	traceIDIndex int
}

func newRowIterator(r *parquet.Reader, pageFile *pageFile, rowNumbers []common.IDMapEntry[int64], traceIDIndex int) *rowIterator { //nolint:all //deprecated
	return &rowIterator{
		reader:       r,
		pageFile:     pageFile,
		rowNumbers:   rowNumbers,
		traceIDIndex: traceIDIndex,
	}
}

func (i *rowIterator) peekNextID(ctx context.Context) (common.ID, error) { //nolint:unused //this is being marked as unused, but it's required to satisfy the bookmarkIterator interface
	if len(i.rowNumbers) == 0 {
		return nil, nil
	}

	return i.rowNumbers[0].ID, nil
}

func (i *rowIterator) Next(ctx context.Context) (common.ID, parquet.Row, error) {
	if len(i.rowNumbers) == 0 {
		return nil, nil, nil
	}

	nextRowNumber := i.rowNumbers[0]
	i.rowNumbers = i.rowNumbers[1:]

	err := i.reader.SeekToRow(nextRowNumber.Entry)
	if err != nil {
		return nil, nil, err
	}

	rows := []parquet.Row{completeBlockRowPool.Get()}
	_, err = i.reader.ReadRows(rows)
	if err != nil {
		return nil, nil, err
	}

	row := rows[0]
	var id common.ID
	for _, v := range row {
		if v.Column() == i.traceIDIndex {
			id = v.ByteArray()
			break
		}
	}

	return id, row, nil
}

func (i *rowIterator) Close() {
	i.reader.Close()
	i.pageFile.Close()
}

var _ common.Iterator = (*commonIterator)(nil)

// commonIterator implements common.Iterator. it is returned from the AppendFile and is meant
// to be passed to a CreateBlock
type commonIterator struct {
	iter   *MultiBlockIterator[parquet.Row]
	schema *parquet.Schema
}

func newCommonIterator(iter *MultiBlockIterator[parquet.Row], schema *parquet.Schema) *commonIterator {
	return &commonIterator{
		iter:   iter,
		schema: schema,
	}
}

func (i *commonIterator) Next(ctx context.Context) (common.ID, *tempopb.Trace, error) {
	id, row, err := i.iter.Next(ctx)
	if err != nil && err != io.EOF {
		return nil, nil, err
	}

	if row == nil || err == io.EOF {
		return nil, nil, nil
	}

	t := &Trace{}
	err = i.schema.Reconstruct(t, row)
	if err != nil {
		return nil, nil, err
	}

	tr := parquetTraceToTempopbTrace(t)
	return id, tr, nil
}

func (i *commonIterator) NextRow(ctx context.Context) (common.ID, parquet.Row, error) {
	return i.iter.Next(ctx)
}

func (i *commonIterator) Close() {
	i.iter.Close()
}
