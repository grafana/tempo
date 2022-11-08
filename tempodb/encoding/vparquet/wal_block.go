package vparquet

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/grafana/tempo/pkg/model"
	"github.com/grafana/tempo/pkg/model/trace"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/traceql"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/pkg/errors"
	"github.com/segmentio/parquet-go"
)

var _ common.WALBlock = (*walBlock)(nil)

var sharedPool = sync.Pool{}

func getPooledTrace() *Trace {
	o := sharedPool.Get()

	if o == nil {
		return &Trace{}
	}

	return o.(*Trace)
}

func putPooledTrace(tr *Trace) {
	sharedPool.Put(tr)
}

// path + filename = folder to create
//   path/folder/00001
//   	        /00002
//              /00003
//              /00004

// folder = <blockID>+<tenantID>+vParquet

// openWALBlock opens an existing appendable block.  It is read-only by
// not assigning a decoder.
func openWALBlock(filename string, path string, ingestionSlack time.Duration, additionalStartSlack time.Duration) (common.WALBlock, error, error) { // jpe what returns a warning?
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
		meta: meta,
		path: path,
		ids:  common.NewIDMap(),
	}

	// read all files in dir
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil, fmt.Errorf("error reading dir: %w", err)
	}

	for _, f := range files {
		if f.Name() == backend.MetaName {
			continue
		}

		// attempt to load in a parquet.file
		pf, sz, err := openLocalParquetFile(filepath.Join(dir, f.Name()))
		if err != nil {
			return nil, nil, fmt.Errorf("error opening file: %w", err)
		}

		b.flushed = append(b.flushed, &walBlockFlush{
			file: pf,
			ids:  common.NewIDMap(),
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
					page.ids.Set(traceID)
				}
			}
		}
	}

	return b, nil, nil
}

// createWALBlock creates a new appendable block
func createWALBlock(id uuid.UUID, tenantID string, filepath string, _ backend.Encoding, dataEncoding string, ingestionSlack time.Duration) (common.WALBlock, error) {
	b := &walBlock{
		meta: &backend.BlockMeta{
			Version:  VersionString,
			BlockID:  id,
			TenantID: tenantID,
		},
		path: filepath,
		ids:  common.NewIDMap(),
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

	return b, nil
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
	ids  *common.IDMap
}

type walBlock struct {
	meta *backend.BlockMeta
	path string

	// Unflushed data
	traces        []*Trace
	ids           *common.IDMap
	unflushedSize int64

	// Flushed data
	flushed     []*walBlockFlush
	flushedSize int64

	writer  *parquet.GenericWriter[*Trace]
	decoder model.ObjectDecoder
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

	tr := traceToParquet(id, trace, getPooledTrace())

	b.meta.ObjectAdded(id, start, end)

	// add to current
	b.traces = append(b.traces, tr)
	b.ids.Set(id)

	// This is actually the protobuf size but close enough
	// for this purpose and only temporary until next flush.
	b.unflushedSize += int64(len(buff))

	return nil
}

func (b *walBlock) Flush() (err error) {

	if len(b.traces) == 0 {
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

	nextFile := len(b.flushed) + 1
	filename := fmt.Sprintf("%010d", nextFile)
	filename = filepath.Join(b.walPath(), filename)

	// Sort currently buffered data by trace ID
	sort.Slice(b.traces, func(i, j int) bool {
		return bytes.Compare(b.traces[i].TraceID, b.traces[j].TraceID) == -1
	})

	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("error opening file: %w", err)
	}
	defer file.Close()

	if b.writer == nil {
		b.writer = parquet.NewGenericWriter[*Trace](file)
	} else {
		b.writer.Reset(file)
	}

	_, err = b.writer.Write(b.traces)
	if err != nil {
		return fmt.Errorf("error writing row: %w", err)
	}

	err = b.writer.Close()
	if err != nil {
		return fmt.Errorf("error closing writer: %w", err)
	}
	err = file.Close()
	if err != nil {
		return fmt.Errorf("error closing file: %w", err)
	}

	// Clear/repool current buffers
	for i := range b.traces {
		putPooledTrace(b.traces[i])
		b.traces[i] = nil
	}
	b.traces = b.traces[:0]

	pf, sz, err := openLocalParquetFile(filename)
	if err != nil {
		return fmt.Errorf("error opening local file [%s]: %w", filename, err)
	}

	b.flushed = append(b.flushed, &walBlockFlush{
		file: pf,
		ids:  b.ids,
	})
	b.flushedSize += sz
	b.unflushedSize = 0
	b.ids = common.NewIDMap()

	return nil
}

// DataLength returns estimated size of WAL files on disk. Used for
// cutting WAL files by max size.
func (b *walBlock) DataLength() uint64 {
	return uint64(b.flushedSize + b.unflushedSize)
}

func (b *walBlock) Iterator() (common.Iterator, error) {
	var pool sync.Pool
	bookmarks := make([]*bookmark[*Trace], 0, len(b.flushed))
	for _, page := range b.flushed {
		r := parquet.NewGenericReader[*Trace](page.file)
		iter := &traceIterator{reader: r, pool: &pool}

		bookmarks = append(bookmarks, newBookmark[*Trace](iter))
	}

	iter := newMultiblockIterator(bookmarks, func(ts []*Trace) (*Trace, error) {
		t := CombineTraces(ts...)
		return t, nil
	})

	return &commonIterator{
		iter: iter,
		pool: &pool,
	}, nil
}

func (b *walBlock) Clear() error {
	return os.RemoveAll(b.walPath())
}

// jpe what to do with common.SearchOptions?
func (b *walBlock) FindTraceByID(ctx context.Context, id common.ID, opts common.SearchOptions) (*tempopb.Trace, error) {
	trs := make([]*tempopb.Trace, 0)

	for i, page := range b.flushed {
		if page.ids.Has(id) {
			tr, err := findTraceByID(ctx, id, b.meta, page.file)
			if err != nil {
				return nil, fmt.Errorf("error finding trace by id in block [%s %d]: %w", b.meta.BlockID.String(), i, err)
			}
			trs = append(trs, tr)
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

func (b *walBlock) Fetch(ctx context.Context, req traceql.FetchSpansRequest) (traceql.FetchSpansResponse, error) {
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

// traceIterator is used to iterate a parquet file and implement iterIterator
type traceIterator struct {
	reader *parquet.GenericReader[*Trace]
	pool   *sync.Pool
}

func (i *traceIterator) Next(ctx context.Context) (common.ID, *Trace, error) {
	var tr *Trace
	tri := i.pool.Get()
	if tri != nil {
		tr = tri.(*Trace)
	}

	trs := []*Trace{tr}
	_, err := i.reader.Read(trs)
	if err != nil {
		return nil, nil, err
	}

	tr = trs[0]
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
	pool *sync.Pool
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
