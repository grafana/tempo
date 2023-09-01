package v2

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/opentracing/opentracing-go"

	"github.com/grafana/tempo/pkg/dataquality"
	"github.com/grafana/tempo/pkg/model"
	"github.com/grafana/tempo/pkg/model/decoder"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/traceql"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

const maxDataEncodingLength = 32

var _ common.WALBlock = (*walBlock)(nil)

// walBlock is a block that is actively used to append new objects to.  It stores all data in the appendFile
// in the order it was received and an in memory sorted index.
type walBlock struct {
	meta           *backend.BlockMeta
	ingestionSlack time.Duration

	appendFile *os.File
	appender   Appender
	encoder    model.SegmentDecoder

	filepath string
	readFile *os.File
	once     sync.Once
}

func createWALBlock(id uuid.UUID, tenantID string, filepath string, e backend.Encoding, dataEncoding string, ingestionSlack time.Duration) (common.WALBlock, error) {
	if strings.ContainsRune(dataEncoding, ':') || strings.ContainsRune(dataEncoding, '+') ||
		len([]rune(dataEncoding)) > maxDataEncodingLength {
		return nil, fmt.Errorf("dataEncoding %s is invalid", dataEncoding)
	}

	enc, err := model.NewSegmentDecoder(dataEncoding)
	if err != nil {
		return nil, err
	}

	h := &walBlock{
		meta:           backend.NewBlockMeta(tenantID, id, VersionString, e, dataEncoding),
		filepath:       filepath,
		ingestionSlack: ingestionSlack,
		encoder:        enc,
	}

	name := h.fullFilename()

	f, err := os.OpenFile(name, os.O_APPEND|os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return nil, err
	}
	h.appendFile = f

	dataWriter, err := NewDataWriter(f, e)
	if err != nil {
		return nil, err
	}

	h.appender = NewAppender(dataWriter)

	return h, nil
}

// openWALBlock returns an AppendBlock that can not be appended to, but can
// be completed. It can return a warning or a fatal error
func openWALBlock(filename string, path string, ingestionSlack time.Duration, additionalStartSlack time.Duration) (common.WALBlock, error, error) {
	var warning error
	blockID, tenantID, version, e, dataEncoding, err := ParseFilename(filename)
	if err != nil {
		return nil, nil, fmt.Errorf("parsing wal filename: %w", err)
	}

	b := &walBlock{
		meta:           backend.NewBlockMeta(tenantID, blockID, version, e, dataEncoding),
		filepath:       path,
		ingestionSlack: ingestionSlack,
	}

	// replay file to extract records
	f, err := b.file()
	if err != nil {
		return nil, nil, fmt.Errorf("accessing file: %w", err)
	}

	blockStart := uint32(math.MaxUint32)
	blockEnd := uint32(0)

	dec, err := model.NewObjectDecoder(dataEncoding)
	if err != nil {
		return nil, nil, fmt.Errorf("creating object decoder: %w", err)
	}

	records, warning, err := ReplayWALAndGetRecords(f, e, func(bytes []byte) error {
		start, end, err := dec.FastRange(bytes)
		if err == decoder.ErrUnsupported {
			now := uint32(time.Now().Unix())
			start = now
			end = now
		}
		if err != nil {
			return err
		}

		start, end = b.adjustTimeRangeForSlack(start, end, additionalStartSlack)
		if start < blockStart {
			blockStart = start
		}
		if end > blockEnd {
			blockEnd = end
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	b.appender = NewRecordAppender(records)
	b.meta.TotalObjects = b.appender.Length()
	b.meta.StartTime = time.Unix(int64(blockStart), 0)
	b.meta.EndTime = time.Unix(int64(blockEnd), 0)

	return b, warning, nil
}

func ownsWALBlock(entry fs.DirEntry) bool {
	// all v2 wal blocks are files
	if entry.IsDir() {
		return false
	}

	_, _, version, _, _, err := ParseFilename(entry.Name())
	if err != nil {
		return false
	}

	return version == VersionString
}

// Append adds an id and object to this wal block. start/end should indicate the time range
// associated with the past object. They are unix epoch seconds.
func (a *walBlock) Append(id common.ID, b []byte, start, end uint32) error {
	err := a.appender.Append(id, b)
	if err != nil {
		return err
	}
	start, end = a.adjustTimeRangeForSlack(start, end, 0)
	a.meta.ObjectAdded(id, start, end)
	return nil
}

func (a *walBlock) AppendTrace(id common.ID, trace *tempopb.Trace, start, end uint32) error {
	buff, err := a.encoder.PrepareForWrite(trace, start, end)
	if err != nil {
		return err
	}

	buff2, err := a.encoder.ToObject([][]byte{buff})
	if err != nil {
		return err
	}

	return a.Append(id, buff2, start, end)
}

func (a *walBlock) Flush() error {
	return nil
}

func (a *walBlock) DataLength() uint64 {
	return a.appender.DataLength()
}

func (a *walBlock) BlockMeta() *backend.BlockMeta {
	return a.meta
}

// Iterator returns a common.Iterator that is secretly also a BytesIterator for use internally
func (a *walBlock) Iterator() (common.Iterator, error) {
	combiner := model.StaticCombiner

	if a.appendFile != nil {
		err := a.appendFile.Close()
		if err != nil {
			return nil, err
		}
		a.appendFile = nil
	}

	records := a.appender.Records()
	readFile, err := a.file()
	if err != nil {
		return nil, err
	}

	dataReader, err := NewDataReader(backend.NewContextReaderWithAllReader(readFile), a.meta.Encoding)
	if err != nil {
		return nil, err
	}

	iterator := newRecordIterator(records, dataReader, NewObjectReaderWriter())
	iterator, err = NewDedupingIterator(iterator, combiner, a.meta.DataEncoding)
	if err != nil {
		return nil, err
	}

	return iterator.(*dedupingIterator), nil
}

func (a *walBlock) Clear() error {
	if a.readFile != nil {
		_ = a.readFile.Close()
		a.readFile = nil
	}

	if a.appendFile != nil {
		_ = a.appendFile.Close()
		a.appendFile = nil
	}

	// ignore error, it's important to remove the file above all else
	_ = a.appender.Complete()

	name := a.fullFilename()
	return os.Remove(name)
}

// Find implements common.Finder
func (a *walBlock) FindTraceByID(ctx context.Context, id common.ID, _ common.SearchOptions) (*tempopb.Trace, error) {
	span, _ := opentracing.StartSpanFromContext(ctx, "v2WalBlock.FindTraceByID")
	defer span.Finish()

	combiner := model.StaticCombiner

	records := a.appender.RecordsForID(id)
	file, err := a.file()
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, nil
	}

	dataReader, err := NewDataReader(backend.NewContextReaderWithAllReader(file), a.meta.Encoding)
	if err != nil {
		return nil, err
	}
	defer dataReader.Close()
	finder := newPagedFinder(Records(records), dataReader, combiner, NewObjectReaderWriter(), a.meta.DataEncoding)

	bytes, err := finder.Find(context.Background(), id)
	if err != nil {
		return nil, err
	}

	if bytes == nil {
		return nil, nil
	}

	dec, err := model.NewObjectDecoder(a.meta.DataEncoding)
	if err != nil {
		return nil, err
	}

	return dec.PrepareForRead(bytes)
}

// Search implements common.Searcher
func (a *walBlock) Search(context.Context, *tempopb.SearchRequest, common.SearchOptions) (*tempopb.SearchResponse, error) {
	return nil, common.ErrUnsupported
}

// Search implements common.Searcher
func (a *walBlock) SearchTags(context.Context, traceql.AttributeScope, common.TagCallback, common.SearchOptions) error {
	return common.ErrUnsupported
}

// SearchTagValues implements common.Searcher
func (a *walBlock) SearchTagValues(context.Context, string, common.TagCallback, common.SearchOptions) error {
	return common.ErrUnsupported
}

func (a *walBlock) SearchTagValuesV2(context.Context, traceql.Attribute, common.TagCallbackV2, common.SearchOptions) error {
	return common.ErrUnsupported
}

// Fetch implements traceql.SpansetFetcher
func (a *walBlock) Fetch(context.Context, traceql.FetchSpansRequest, common.SearchOptions) (traceql.FetchSpansResponse, error) {
	return traceql.FetchSpansResponse{}, common.ErrUnsupported
}

func (a *walBlock) fullFilename() string {
	filename := a.fullFilenameSeparator("+")
	_, e1 := os.Stat(filename)
	if errors.Is(e1, os.ErrNotExist) {
		filenameWithOldSeparator := a.fullFilenameSeparator(":")
		_, e2 := os.Stat(filenameWithOldSeparator)
		if !errors.Is(e2, os.ErrNotExist) {
			filename = filenameWithOldSeparator
		}
	}

	return filename
}

func (a *walBlock) fullFilenameSeparator(separator string) string {
	if a.meta.Version == "v0" {
		return filepath.Join(a.filepath, fmt.Sprintf("%v%v%v", a.meta.BlockID, separator, a.meta.TenantID))
	}

	var filename string
	if a.meta.DataEncoding == "" {
		filename = fmt.Sprintf("%v%v%v%v%v%v%v", a.meta.BlockID, separator, a.meta.TenantID, separator, a.meta.Version, separator, a.meta.Encoding)
	} else {
		filename = fmt.Sprintf("%v%v%v%v%v%v%v%v%v", a.meta.BlockID, separator, a.meta.TenantID, separator, a.meta.Version, separator, a.meta.Encoding, separator, a.meta.DataEncoding)
	}

	return filepath.Join(a.filepath, filename)
}

func (a *walBlock) file() (*os.File, error) {
	var err error
	a.once.Do(func() {
		if a.readFile == nil {
			name := a.fullFilename()

			a.readFile, err = os.OpenFile(name, os.O_RDONLY, 0o644)
		}
	})

	return a.readFile, err
}

func (a *walBlock) adjustTimeRangeForSlack(start uint32, end uint32, additionalStartSlack time.Duration) (uint32, uint32) {
	now := time.Now()
	startOfRange := uint32(now.Add(-a.ingestionSlack).Add(-additionalStartSlack).Unix())
	endOfRange := uint32(now.Add(a.ingestionSlack).Unix())

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
		dataquality.WarnOutsideIngestionSlack(a.meta.TenantID)
	}

	return start, end
}

// ParseFilename returns (blockID, tenant, version, encoding, dataEncoding, error).
// Example: "00000000-0000-0000-0000-000000000000+1+v2+snappy+v1"
// Example with old separator: "00000000-0000-0000-0000-000000000000:1:v2:snappy:v1"
func ParseFilename(filename string) (uuid.UUID, string, string, backend.Encoding, string, error) {
	var splits []string
	if strings.Contains(filename, "+") {
		splits = strings.Split(filename, "+")
	} else {
		// backward-compatibility with the old separator
		splits = strings.Split(filename, ":")
	}

	if len(splits) != 4 && len(splits) != 5 {
		return uuid.UUID{}, "", "", backend.EncNone, "", fmt.Errorf("unable to parse %s. unexpected number of segments", filename)
	}

	// first segment is blockID
	id, err := uuid.Parse(splits[0])
	if err != nil {
		return uuid.UUID{}, "", "", backend.EncNone, "", fmt.Errorf("unable to parse %s. error parsing uuid: %w", filename, err)
	}

	// second segment is tenant
	tenant := splits[1]
	if len(tenant) == 0 {
		return uuid.UUID{}, "", "", backend.EncNone, "", fmt.Errorf("unable to parse %s. missing fields", filename)
	}

	// third segment is version
	version := splits[2]
	if version != VersionString {
		return uuid.UUID{}, "", "", backend.EncNone, "", fmt.Errorf("unable to parse %s. error parsing version: %w", filename, err)
	}

	// fourth is encoding
	encodingString := splits[3]
	encoding, err := backend.ParseEncoding(encodingString)
	if err != nil {
		return uuid.UUID{}, "", "", backend.EncNone, "", fmt.Errorf("unable to parse %s. error parsing encoding: %w", filename, err)
	}

	// fifth is dataEncoding
	dataEncoding := ""
	if len(splits) == 5 {
		dataEncoding = splits[4]
	}

	return id, tenant, version, encoding, dataEncoding, nil
}
