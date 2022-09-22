package vparquet

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/grafana/tempo/pkg/model"
	"github.com/grafana/tempo/pkg/model/trace"
	pq "github.com/grafana/tempo/pkg/parquetquery"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/traceql"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/segmentio/parquet-go"
)

// path + filename = folder to create
//   path/filename/00001
//   	          /00002
//                /00003
//                /00004

// filename = <blockID>+<tenantID>+vParquet

// openWALBlock opens an existing appendable block
// jpe refuse append to this block? return a different type?
// jpe combine filename and path?
func openWALBlock(filename string, path string, ingestionSlack time.Duration, additionalStartSlack time.Duration) (common.WALBlock, error, error) { // jpe what returns a warning?
	dir := filepath.Join(path, filename)
	blockID, tenantID, version, err := parseName(filename)
	if err != nil {
		return nil, nil, err
	}

	if version != VersionString {
		return nil, nil, fmt.Errorf("mismatched version in vparquet wal: %s, %s, %s", version, path, filename)
	}

	b := &walBlock{
		meta: &backend.BlockMeta{
			Version:  VersionString,
			BlockID:  blockID,
			TenantID: tenantID,
		},
		path:    path,
		current: parquet.NewGenericBuffer[*Trace](), // jpe parquet.ColumnBufferCapacity, need parquet.SortingColumns()?
	}

	// read all files in dir
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil, fmt.Errorf("error reading dir: %w", err)
	}

	for _, f := range files {
		// attempt to load in a parquet.file
		pf, err := openLocalParquetFile(filepath.Join(dir, f.Name()))
		if err != nil {
			return nil, nil, fmt.Errorf("error opening file: %w", err)
		}

		b.flushed = append(b.flushed, pf)
	}

	// iterate through all files and build meta
	for i, pf := range b.flushed {
		// retrieve start, end, traceid
		makeIter := makeIterFunc(context.Background(), pf.RowGroups(), pf)

		iter := pq.NewJoinIterator(DefinitionLevelTrace, []pq.Iterator{
			makeIter("TraceID", nil, "TraceID"),
			makeIter("StartTimeUnixNano", nil, "StartTimeUnixNano"),
			makeIter("DurationNanos", nil, "DurationNanos"),
		}, nil)
		defer iter.Close()

		for {
			match, err := iter.Next()
			if err != nil {
				return nil, nil, fmt.Errorf("error opening wal folder [%s %d]: %w", b.meta.BlockID.String(), i, err)
			}
			if match == nil {
				break
			}

			// find values
			var traceID common.ID
			var start uint64
			var duration uint64

			for _, e := range match.Entries {
				switch e.Key {
				case "TraceID":
					traceID = e.Value.ByteArray()
				case "StartTimeUnixNano":
					start = e.Value.Uint64()
				case "DurationNanos":
					duration = e.Value.Uint64()
				}
			}

			// convert to ms
			startMS := uint32(start / uint64(time.Millisecond))
			endMS := uint32((start + duration) / uint64(time.Millisecond))

			// jpe, handle ingestion slack and additional start slack
			b.meta.ObjectAdded(traceID, startMS, endMS)
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
		path:    filepath,
		current: parquet.NewGenericBuffer[*Trace](), // jpe parquet.ColumnBufferCapacity, need parquet.SortingColumns()?
	}

	// build folder
	err := os.MkdirAll(filepath, os.ModePerm)
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

type walBlock struct {
	meta *backend.BlockMeta
	path string

	current *parquet.GenericBuffer[*Trace]
	decoder model.ObjectDecoder

	flushed []*parquet.File // jpe prealloc?
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

	trp := traceToParquet(id, trace) // jpe find and restore pooling code

	// jpe adjust time range
	b.meta.ObjectAdded(id, start, end)

	// add to current
	b.current.Write([]*Trace{&trp}) // jpe experiment with buffering these differently and writing in batches

	return nil
}

func (b *walBlock) Flush() (err error) {
	nextFile := len(b.flushed) + 1
	filename := fmt.Sprintf("%10d", nextFile)

	sort.Sort(b.current)

	file, err := os.OpenFile(filepath.Join(b.path, filename), os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("error opening file: %w", err)
	}
	defer file.Close()

	writer := parquet.NewGenericWriter[*Trace](file) // jpe options?
	defer writer.Close()

	_, err = writer.WriteRowGroup(b.current)
	writer.Close()

	// cleanup
	b.current = parquet.NewGenericBuffer[*Trace]() // jpe parquet.ColumnBufferCapacity, need parquet.SortingColumns()?

	pf, err := openLocalParquetFile(filepath.Join(b.path, filename))
	if err != nil {
		return fmt.Errorf("error opening parquet file: %w", err)
	}

	b.flushed = append(b.flushed, pf)

	return nil
}

func (b *walBlock) DataLength() uint64 { // jpe oof, another parquet size question
	return uint64(b.current.Size()) // jpe performance? needs to take into account size of all flushed files
}

func (b *walBlock) Iterator() (common.Iterator, error) {
	bookmarks := make([]*bookmark[*Trace], len(b.flushed))
	for _, f := range b.flushed {
		r := parquet.NewGenericReader[*Trace](f)
		iter := &traceIterator{reader: r}

		bookmarks = append(bookmarks, newBookmark[*Trace](iter))
	}

	iter := newMultiblockIterator[*Trace](bookmarks, func(ts []*Trace) (*Trace, error) {
		t := CombineTraces(ts...)
		return t, nil
	})

	return &commonIterator{
		iter: iter,
	}, nil
}

func (b *walBlock) Clear() error {
	// jpe close all open files ?
	// for _, f := range b.flushed {
	// 	f.
	// }

	return os.RemoveAll(walPath(b.meta))
}

func (b *walBlock) FindTraceByID(ctx context.Context, id common.ID, opts common.SearchOptions) (*tempopb.Trace, error) {
	trs := make([]*tempopb.Trace, 0)

	// jpe do in parrallel? store a map of trace id to flushed file?
	for i, f := range b.flushed {
		tr, err := findTraceByID(ctx, id, b.meta, f)
		if err != nil {
			return nil, fmt.Errorf("error finding trace by id in block [%s %d]: %w", b.meta.BlockID.String(), i, err)
		}
		trs = append(trs, tr)
	}

	combiner := trace.NewCombiner()
	for i, tr := range trs {
		combiner.ConsumeWithFinal(tr, i == len(trs)-1)
	}

	tr, _ := combiner.Result()
	return tr, nil
}

func (b *walBlock) Search(ctx context.Context, req *tempopb.SearchRequest, opts common.SearchOptions) (*tempopb.SearchResponse, error) {
	results := &tempopb.SearchResponse{}

	// jpe parrallelize?
	for i, f := range b.flushed {
		r, err := searchParquetFile(ctx, f, req, f.RowGroups())
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
	// jpe parrallelize?
	for i, f := range b.flushed {
		err := searchTags(ctx, cb, f)
		if err != nil {
			return fmt.Errorf("error searching block [%s %d]: %w", b.meta.BlockID.String(), i, err)
		}
	}

	return nil
}

func (b *walBlock) SearchTagValues(ctx context.Context, tag string, cb common.TagCallback, opts common.SearchOptions) error {
	// jpe parrallelize?
	for i, f := range b.flushed {
		err := searchTagValues(ctx, tag, cb, f)
		if err != nil {
			return fmt.Errorf("error searching block [%s %d]: %w", b.meta.BlockID.String(), i, err)
		}
	}

	return nil
}

func (b *walBlock) Fetch(context.Context, traceql.FetchSpansRequest) (traceql.FetchSpansResponse, error) {
	return traceql.FetchSpansResponse{}, common.ErrUnsupported
}

func walPath(meta *backend.BlockMeta) string {
	return fmt.Sprintf("%s+%s+%s", meta.BlockID, meta.TenantID, VersionString)
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
}

func (i *traceIterator) Next(ctx context.Context) (common.ID, *Trace, error) {
	trs := make([]*Trace, 1) // jpe try batching?
	_, err := i.reader.Read(trs)
	if err != nil {
		return nil, nil, err
	}

	tr := trs[0]
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
	if err != nil || obj == nil {
		return nil, nil, err
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

func openLocalParquetFile(filename string) (*parquet.File, error) {
	file, err := os.OpenFile(filename, os.O_RDONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %w", err)
	}
	info, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("error getting file info: %w", err)
	}
	pf, err := parquet.OpenFile(file, info.Size())
	if err != nil {
		return nil, fmt.Errorf("error opening parquet file: %w", err)
	}

	return pf, nil
}
